package repository

import (
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	"github.com/wanglejiu/llm-proxy/internal/model"
)

type RequestLogRepository struct {
	dbManager *DBManager
}

func NewRequestLogRepository(dbManager *DBManager) *RequestLogRepository {
	return &RequestLogRepository{dbManager: dbManager}
}

// Create 创建请求日志
func (r *RequestLogRepository) Create(log *model.RequestLog) error {
	return r.dbManager.GetDB().Create(log).Error
}

// GetBySession 根据会话ID获取请求日志列表
func (r *RequestLogRepository) GetBySession(sessionID uint) ([]model.RequestLog, error) {
	var logs []model.RequestLog
	err := r.dbManager.GetDB().Select("id, provider_id, model, input_tokens, output_tokens, total_tokens, cached_tokens, status, error_message, duration, created_at").
		Where("session_id = ?", sessionID).
		Order("created_at asc").
		Find(&logs).Error
	if err != nil {
		return nil, err
	}
	r.fillProviderInfoBatch(logs)
	return logs, nil
}

// UpdateSessionID 更新请求日志的会话ID（创建后补填）
func (r *RequestLogRepository) UpdateSessionID(id uint, sessionID uint) error {
	return r.dbManager.GetDB().Model(&model.RequestLog{}).
		Where("id = ?", id).
		Update("session_id", sessionID).Error
}

// GetCostRowsBySession 获取指定会话的请求成本明细
func (r *RequestLogRepository) GetCostRowsBySession(sessionID uint) ([]CostRow, error) {
	var rows []CostRow
	err := r.dbManager.GetDB().Model(&model.RequestLog{}).
		Select("provider_id, model, input_tokens, output_tokens, cached_tokens").
		Where("session_id = ? AND status = 'success'", sessionID).
		Scan(&rows).Error
	return rows, err
}

// CostRow 成本计算行
type CostRow struct {
	ProviderID   uint
	Model        string
	InputTokens  int
	OutputTokens int
	CachedTokens int
}

// GetByID 根据ID获取日志（含完整大字段，用于查看详情）
func (r *RequestLogRepository) GetByID(id uint) (*model.RequestLog, error) {
	var requestLog model.RequestLog
	err := r.dbManager.GetDB().First(&requestLog, id).Error
	if err != nil {
		slog.Error("根据ID获取日志失败", "id", id, "error", err)
		return nil, err
	}
	// 手动填充Provider信息
	r.fillProviderInfo(&requestLog)
	return &requestLog, nil
}

// GetRecent 获取最近的日志（排除 longtext 大字段，批量预加载 Provider 信息）
// modelName 非空时按模型名过滤
func (r *RequestLogRepository) GetRecent(limit int, modelName string) ([]model.RequestLog, error) {
	var logs []model.RequestLog
	q := r.dbManager.GetDB().Select("id, provider_id, model, input_tokens, output_tokens, total_tokens, cached_tokens, status, error_message, duration, created_at").
		Order("created_at desc")
	if modelName != "" {
		q = q.Where("model = ?", modelName)
	}
	err := q.Limit(limit).Find(&logs).Error
	if err != nil {
		return logs, err
	}
	r.fillProviderInfoBatch(logs)
	return logs, nil
}

// resolveProvider 根据 providerID 和 providerMap 解析 Provider 信息
func resolveProvider(providerID uint, providerMap map[uint]model.ProviderConfig) model.ProviderConfig {
	if providerID == model.DeletedProviderID {
		return model.ProviderConfig{
			ID:   model.DeletedProviderID,
			Name: "已删除",
		}
	}
	if p, ok := providerMap[providerID]; ok {
		return p
	}
	return model.ProviderConfig{
		ID:   providerID,
		Name: "未知",
	}
}

// fillProviderInfoBatch 批量填充 Provider 信息
func (r *RequestLogRepository) fillProviderInfoBatch(logs []model.RequestLog) {
	if len(logs) == 0 {
		return
	}

	// 收集需要查询的 ProviderID
	providerIDs := make(map[uint]struct{})
	for _, log := range logs {
		if log.ProviderID != model.DeletedProviderID {
			providerIDs[log.ProviderID] = struct{}{}
		}
	}

	// 批量查询所有需要的 Provider
	providerMap := make(map[uint]model.ProviderConfig)
	if len(providerIDs) > 0 {
		ids := make([]uint, 0, len(providerIDs))
		for id := range providerIDs {
			ids = append(ids, id)
		}
		var providers []model.ProviderConfig
		r.dbManager.GetDB().Where("id IN ?", ids).Find(&providers)
		for _, p := range providers {
			providerMap[p.ID] = p
		}
	}

	// 填充
	for i := range logs {
		logs[i].Provider = resolveProvider(logs[i].ProviderID, providerMap)
	}
}

// fillProviderInfo 填充Provider信息（ProviderID=DeletedProviderID时显示"已删除"）
func (r *RequestLogRepository) fillProviderInfo(log *model.RequestLog) {
	providerMap := make(map[uint]model.ProviderConfig)
	if log.ProviderID != model.DeletedProviderID {
		var provider model.ProviderConfig
		if err := r.dbManager.GetDB().First(&provider, log.ProviderID).Error; err == nil {
			providerMap[log.ProviderID] = provider
		}
	}
	log.Provider = resolveProvider(log.ProviderID, providerMap)
}

// AggregateHour 汇总指定小时的明细数据，返回 per-(provider,model) 的 HourlyStat 列表
// 只汇总 aggregated=false 且 status=success 的记录。生成：
//   - 每条 (provider_id, model) 组合一行
//   - 每个 provider_id 的合计行 (model="")
//   - 全量行 (provider_id=0, model="")
func (r *RequestLogRepository) AggregateHour(hourStart time.Time) ([]model.HourlyStat, error) {
	hourEnd := hourStart.Add(time.Hour)

	type AggResult struct {
		ProviderID    uint
		Model         string
		InputTokens   int64
		OutputTokens  int64
		TotalTokens   int64
		CachedTokens  int64
		RequestCount  int64
		TotalDuration int64
	}

	var results []AggResult
	err := r.dbManager.GetDB().Model(&model.RequestLog{}).
		Select("provider_id, COALESCE(NULLIF(model, ''), 'unknown') as model, COALESCE(SUM(input_tokens), 0) as input_tokens, COALESCE(SUM(output_tokens), 0) as output_tokens, COALESCE(SUM(total_tokens), 0) as total_tokens, COALESCE(SUM(cached_tokens), 0) as cached_tokens, COUNT(*) as request_count, COALESCE(SUM(duration), 0) as total_duration").
		Where("created_at >= ? AND created_at < ? AND aggregated = ? AND status = ?", hourStart, hourEnd, false, "success").
		Group("provider_id, model").
		Scan(&results).Error
	if err != nil {
		return nil, err
	}

	// 按 (provider_id, model) 的行 + per-provider 合计行 + 全量行
	providerTotals := make(map[uint]*model.HourlyStat)
	stats := make([]model.HourlyStat, 0, len(results)+len(providerTotals)+1)

	for _, r := range results {
		// 具体模型行
		stats = append(stats, model.HourlyStat{
			Hour:          hourStart,
			ProviderID:    r.ProviderID,
			Model:         r.Model,
			InputTokens:   r.InputTokens,
			OutputTokens:  r.OutputTokens,
			TotalTokens:   r.TotalTokens,
			CachedTokens:  r.CachedTokens,
			RequestCount:  r.RequestCount,
			TotalDuration: r.TotalDuration,
		})

		// 累加 provider 合计
		if _, ok := providerTotals[r.ProviderID]; !ok {
			providerTotals[r.ProviderID] = &model.HourlyStat{
				Hour: hourStart, ProviderID: r.ProviderID, Model: "",
			}
		}
		pt := providerTotals[r.ProviderID]
		pt.InputTokens += r.InputTokens
		pt.OutputTokens += r.OutputTokens
		pt.TotalTokens += r.TotalTokens
		pt.CachedTokens += r.CachedTokens
		pt.RequestCount += r.RequestCount
		pt.TotalDuration += r.TotalDuration
	}

	var totalInput, totalOutput, totalToken, totalCached, totalCount, totalDuration int64
	for _, pt := range providerTotals {
		stats = append(stats, *pt)
		totalInput += pt.InputTokens
		totalOutput += pt.OutputTokens
		totalToken += pt.TotalTokens
		totalCached += pt.CachedTokens
		totalCount += pt.RequestCount
		totalDuration += pt.TotalDuration
	}

	// 全量行 (provider_id=0, model="")
	stats = append(stats, model.HourlyStat{
		Hour:          hourStart,
		ProviderID:    0,
		InputTokens:   totalInput,
		OutputTokens:  totalOutput,
		TotalTokens:   totalToken,
		CachedTokens:  totalCached,
		RequestCount:  totalCount,
		TotalDuration: totalDuration,
	})

	return stats, nil
}

// MarkAggregated 将指定小时范围内未汇总的记录标记为已汇总
func (r *RequestLogRepository) MarkAggregated(hourStart time.Time) error {
	hourEnd := hourStart.Add(time.Hour)
	return r.dbManager.GetDB().Model(&model.RequestLog{}).
		Where("created_at >= ? AND created_at < ? AND aggregated = ?", hourStart, hourEnd, false).
		Update("aggregated", true).Error
}

// GetCurrentHourStats 获取当前小时的实时统计（用于混合查询保证实时性）
// providerID=0 时查询所有 provider，providerID>0 时只查询指定 provider
// modelName 非空时额外按模型名过滤
func (r *RequestLogRepository) GetCurrentHourStats(providerID uint, modelName string) (*model.TokenStats, error) {
	hourStart := time.Now().Truncate(time.Hour)
	var stats model.TokenStats

	query := `SELECT
		? as date,
		COALESCE(SUM(input_tokens), 0) as total_input_tokens,
		COALESCE(SUM(output_tokens), 0) as total_output_tokens,
		COALESCE(SUM(total_tokens), 0) as total_tokens,
		COALESCE(SUM(cached_tokens), 0) as total_cached_tokens,
		COUNT(*) as request_count
	FROM request_logs
	WHERE created_at >= ?
		AND status = 'success'`

	args := []interface{}{hourStart.Format("2006-01-02"), hourStart}
	if providerID > 0 {
		query += " AND provider_id = ?"
		args = append(args, providerID)
	}
	if modelName != "" {
		query += " AND model = ?"
		args = append(args, modelName)
	}

	err := r.dbManager.GetDB().Raw(query, args...).Scan(&stats).Error
	return &stats, err
}

// GetCurrentHourHourlyStats 获取当前小时的分时统计（用于混合查询保证实时性）
func (r *RequestLogRepository) GetCurrentHourHourlyStats(providerID uint) (*model.HourlyStatsResult, error) {
	hourStart := time.Now().Truncate(time.Hour)

	var hourExpr string
	if r.dbManager.GetDBType() == "sqlite" {
		hourExpr = "CAST(strftime('%H', created_at) AS INTEGER)"
	} else {
		hourExpr = "EXTRACT(HOUR FROM created_at)"
	}

	var result model.HourlyStatsResult
	query := fmt.Sprintf(`SELECT
		%s as hour,
		COUNT(*) as request_count,
		COALESCE(SUM(total_tokens), 0) as total_tokens,
		COALESCE(SUM(input_tokens), 0) as input_tokens,
		COALESCE(SUM(output_tokens), 0) as output_tokens,
		COALESCE(SUM(cached_tokens), 0) as cached_tokens
	FROM request_logs
	WHERE created_at >= ?
		AND status = 'success'`, hourExpr)

	args := []interface{}{hourStart}
	if providerID > 0 {
		query += " AND provider_id = ?"
		args = append(args, providerID)
	}
	query += fmt.Sprintf(" GROUP BY %s", hourExpr)

	err := r.dbManager.GetDB().Raw(query, args...).Scan(&result).Error
	return &result, err
}

// GetHourlyStatsByDateFromLogs 从明细表获取指定日期的分时统计（用于历史日期无汇总数据时的回退查询）
// providerID=0 时查询所有 provider，providerID>0 时只查询指定 provider
func (r *RequestLogRepository) GetHourlyStatsByDateFromLogs(date time.Time, providerID uint) ([]model.HourlyStatsResult, error) {
	dayStart := date.Truncate(24 * time.Hour)
	dayEnd := dayStart.Add(24 * time.Hour)

	var hourExpr string
	if r.dbManager.GetDBType() == "sqlite" {
		hourExpr = "CAST(strftime('%H', created_at) AS INTEGER)"
	} else {
		hourExpr = "EXTRACT(HOUR FROM created_at)"
	}

	args := []interface{}{dayStart, dayEnd}
	query := fmt.Sprintf(`
		SELECT
			%s as hour,
			COUNT(*) as request_count,
			COALESCE(SUM(total_tokens), 0) as total_tokens,
			COALESCE(SUM(input_tokens), 0) as input_tokens,
			COALESCE(SUM(output_tokens), 0) as output_tokens,
			COALESCE(SUM(cached_tokens), 0) as cached_tokens
		FROM request_logs
		WHERE created_at >= ? AND created_at < ?
			AND status = 'success'`, hourExpr)
	if providerID > 0 {
		query += " AND provider_id = ?"
		args = append(args, providerID)
	}
	query += fmt.Sprintf(`
		GROUP BY %s
		ORDER BY hour
	`, hourExpr)

	var results []model.HourlyStatsResult
	err := r.dbManager.GetDB().Raw(query, args...).Scan(&results).Error
	return results, err
}

// GetMinCreatedAt 获取最早记录的创建时间（用于回填起始点）
func (r *RequestLogRepository) GetMinCreatedAt() (*time.Time, error) {
	var nullTime sql.NullTime
	err := r.dbManager.GetDB().Model(&model.RequestLog{}).
		Select("MIN(created_at)").
		Scan(&nullTime).Error
	if err != nil {
		return nil, err
	}
	if !nullTime.Valid {
		return nil, nil
	}
	t := nullTime.Time
	return &t, nil
}

// DeleteOldRequestLogs 删除超过保留天数的明细记录
// 安全边界：只删除 created_at < (当前小时起始 - 1小时 - cleanupDays天) 且已汇总的记录
// 确保只删除已确认汇总完成的记录，未汇总的记录不会被误删
func (r *RequestLogRepository) DeleteOldRequestLogs(days int) (int64, error) {
	cutoffDate := time.Now().Truncate(time.Hour).Add(-time.Hour).AddDate(0, 0, -days)
	result := r.dbManager.GetDB().Where("created_at < ? AND aggregated = ?", cutoffDate, true).Delete(&model.RequestLog{})
	return result.RowsAffected, result.Error
}

// CurrentHourBreakdown 当前小时按 (provider, model) 拆分的实时统计结果
type CurrentHourBreakdown struct {
	ProviderID   uint   `json:"provider_id"`
	Model        string `json:"model"`
	InputTokens  int64  `json:"input_tokens"`
	OutputTokens int64  `json:"output_tokens"`
	TotalTokens  int64  `json:"total_tokens"`
}

// ModelDailyStat 模型级别每日统计数据
type ModelDailyStat struct {
	Date              string `json:"date"`
	ProviderID        uint   `json:"provider_id"`
	Model             string `json:"model"`
	TotalInputTokens  int64  `json:"total_input_tokens"`
	TotalOutputTokens int64  `json:"total_output_tokens"`
	TotalTokens       int64  `json:"total_tokens"`
	TotalCachedTokens int64  `json:"total_cached_tokens"`
	RequestCount      int64  `json:"request_count"`
}

// GetHourlyModelStats 获取指定日期指定模型的分时统计（直接从 request_logs 查询）
func (r *RequestLogRepository) GetHourlyModelStats(date time.Time, providerID uint, modelName string) ([]model.HourlyStatsResult, error) {
	dayStart := date.Truncate(24 * time.Hour)
	// 只查询已完成的小时（排除当前小时），当前小时由 service 层单独获取并合并
	// 对于历史日期：currentHourStart() 远大于 dayEnd，效果等同于查询全天
	upperBound := currentHourStart()

	var hourExpr string
	if r.dbManager.GetDBType() == "sqlite" {
		hourExpr = "CAST(strftime('%H', created_at) AS INTEGER)"
	} else {
		hourExpr = "EXTRACT(HOUR FROM created_at)"
	}

	var results []model.HourlyStatsResult
	err := r.dbManager.GetDB().Raw(fmt.Sprintf(`SELECT
		%s as hour,
		COUNT(*) as request_count,
		COALESCE(SUM(total_tokens), 0) as total_tokens,
		COALESCE(SUM(input_tokens), 0) as input_tokens,
		COALESCE(SUM(output_tokens), 0) as output_tokens,
		COALESCE(SUM(cached_tokens), 0) as cached_tokens
	FROM request_logs
	WHERE created_at >= ? AND created_at < ?
		AND status = 'success'
		AND provider_id = ?
		AND model = ?
	GROUP BY %s
	ORDER BY hour
	`, hourExpr, hourExpr), dayStart, upperBound, providerID, modelName).Scan(&results).Error
	return results, err
}

// GetCurrentHourModelStats 获取当前小时按 (provider, model) 拆分的模型级统计（用于混合查询）
func (r *RequestLogRepository) GetCurrentHourModelStats(providerID uint) ([]ModelDailyStat, error) {
	hourStart := time.Now().Truncate(time.Hour)

	query := `SELECT
		? as date,
		provider_id,
		COALESCE(NULLIF(model, ''), 'unknown') as model,
		COALESCE(SUM(input_tokens), 0) as total_input_tokens,
		COALESCE(SUM(output_tokens), 0) as total_output_tokens,
		COALESCE(SUM(total_tokens), 0) as total_tokens,
		COALESCE(SUM(cached_tokens), 0) as total_cached_tokens,
		COUNT(*) as request_count
	FROM request_logs
	WHERE created_at >= ?
		AND status = 'success'`

	args := []interface{}{hourStart.Format("2006-01-02"), hourStart}
	if providerID > 0 {
		query += " AND provider_id = ?"
		args = append(args, providerID)
	}

	query += " GROUP BY provider_id, model"

	var results []ModelDailyStat
	err := r.dbManager.GetDB().Raw(query, args...).Scan(&results).Error
	if err != nil {
		return nil, err
	}
	if results == nil {
		return []ModelDailyStat{}, nil
	}
	return results, nil
}

// GetCurrentHourBreakdown 获取当前小时按 (provider, model) 拆分的实时统计数据（用于堆叠图）
// providerID=0 时返回所有 provider 的数据，providerID>0 时只返回指定 provider 的数据
func (r *RequestLogRepository) GetCurrentHourBreakdown(providerID uint) ([]CurrentHourBreakdown, error) {
	hourStart := time.Now().Truncate(time.Hour)

	query := r.dbManager.GetDB().Model(&model.RequestLog{}).
		Select("provider_id, COALESCE(NULLIF(model, ''), 'unknown') as model, COALESCE(SUM(input_tokens), 0) as input_tokens, COALESCE(SUM(output_tokens), 0) as output_tokens, COALESCE(SUM(total_tokens), 0) as total_tokens").
		Where("created_at >= ? AND status = 'success'", hourStart)
	if providerID > 0 {
		query = query.Where("provider_id = ?", providerID)
	}
	var results []CurrentHourBreakdown
	err := query.Group("provider_id, model").
		Scan(&results).Error
	return results, err
}
