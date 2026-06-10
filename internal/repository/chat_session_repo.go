package repository

import (
	"encoding/json"
	"log/slog"
	"time"

	"github.com/wanglejiu/llm-proxy/internal/model"
)

type ChatSessionRepository struct {
	dbManager *DBManager
}

func NewChatSessionRepository(dbManager *DBManager) *ChatSessionRepository {
	return &ChatSessionRepository{dbManager: dbManager}
}

// Create 创建新的会话，返回自增 ID
func (r *ChatSessionRepository) Create(session *model.ChatSession) error {
	session.CreatedAt = time.Now()
	session.UpdatedAt = time.Now()
	return r.dbManager.GetDB().Create(session).Error
}

// GetByID 根据 ID 获取会话
func (r *ChatSessionRepository) GetByID(id uint) (*model.ChatSession, error) {
	var session model.ChatSession
	err := r.dbManager.GetDB().First(&session, id).Error
	if err != nil {
		return nil, err
	}
	return &session, nil
}

// GetAll 获取所有会话（按创建时间倒序）
func (r *ChatSessionRepository) GetAll() ([]model.ChatSession, error) {
	var sessions []model.ChatSession
	err := r.dbManager.GetDB().Order("created_at desc").Find(&sessions).Error
	return sessions, err
}

// GetPaginated 分页查询会话列表，sessionID > 0 时按 ID 精确搜索
// 返回会话列表和总条数
func (r *ChatSessionRepository) GetPaginated(page, pageSize int, sessionID uint) ([]model.ChatSession, int64, error) {
	var sessions []model.ChatSession
	var total int64

	q := r.dbManager.GetDB().Model(&model.ChatSession{})
	if sessionID > 0 {
		q = q.Where("id = ?", sessionID)
	}
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}

	err := q.Order("created_at desc").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&sessions).Error
	return sessions, total, err
}

// RecalcSessionStats 根据 request_logs 重新聚合会话统计并回写 chat_sessions
// 在每次请求结束时调用
func (r *ChatSessionRepository) RecalcSessionStats(sessionID uint) {
	type AggResult struct {
		RequestCount int64
		TotalTokens  int64
	}

	var agg AggResult
	err := r.dbManager.GetDB().Model(&model.RequestLog{}).
		Select("COUNT(*) as request_count, COALESCE(SUM(total_tokens), 0) as total_tokens").
		Where("session_id = ?", sessionID).
		Scan(&agg).Error
	if err != nil {
		slog.Error("聚合会话统计失败", "session_id", sessionID, "error", err)
		return
	}

	// 查询该会话使用过的模型（去重）
	var models []string
	r.dbManager.GetDB().Model(&model.RequestLog{}).
		Where("session_id = ? AND model != ''", sessionID).
		Distinct("model").
		Pluck("model", &models)

	// 查询该会话的所有请求，逐个计算成本
	type CostRow struct {
		ProviderID   uint
		Model        string
		InputTokens  int
		OutputTokens int
		CachedTokens int
	}
	var rows []CostRow
	r.dbManager.GetDB().Model(&model.RequestLog{}).
		Select("provider_id, model, input_tokens, output_tokens, cached_tokens").
		Where("session_id = ? AND status = 'success'", sessionID).
		Scan(&rows)

	totalCost := 0.0
	for _, row := range rows {
		totalCost += calculateRequestCost(r.dbManager, row.ProviderID, row.Model, row.InputTokens, row.OutputTokens, row.CachedTokens)
	}

	modelsJSON, _ := json.Marshal(models)
	err = r.dbManager.GetDB().Model(&model.ChatSession{}).
		Where("id = ?", sessionID).
		Updates(map[string]interface{}{
			"models":        string(modelsJSON),
			"request_count": agg.RequestCount,
			"total_tokens":  agg.TotalTokens,
			"total_cost":    totalCost,
			"updated_at":    time.Now(),
		}).Error
	if err != nil {
		slog.Error("更新会话统计失败", "session_id", sessionID, "error", err)
	}
}

// calculateRequestCost 计算单次请求的成本
// 公式: inputFee = (inputTokens - cachedTokens) / 1_000_000 * InputPrice   ← 扣除缓存命中
//       outputFee = outputTokens / 1_000_000 * OutputPrice
//       cacheFee  = cachedTokens / 1_000_000 * CachePrice
func calculateRequestCost(dbManager *DBManager, providerID uint, modelName string, inputTokens, outputTokens, cachedTokens int) float64 {
	if providerID == 0 || modelName == "" {
		return 0
	}

	var provider model.ProviderConfig
	if err := dbManager.GetDB().First(&provider, providerID).Error; err != nil {
		return 0
	}

	entries := provider.ParseModels()
	for _, entry := range entries {
		if entry.Name == modelName {
			paidInput := float64(inputTokens - cachedTokens)
			if paidInput < 0 {
				paidInput = 0
			}
			cost := (paidInput / 1_000_000) * entry.InputPrice
			cost += (float64(outputTokens) / 1_000_000) * entry.OutputPrice
			cost += (float64(cachedTokens) / 1_000_000) * entry.CachePrice
			return cost
		}
	}
	return 0
}
