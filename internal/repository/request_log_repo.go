package repository

import (
	"database/sql"
	"fmt"
	"llm-proxy/internal/model"
	"log/slog"
	"time"

	"gorm.io/gorm"
)

type RequestLogRepository struct {
	db     *gorm.DB
	dbType string
}

func NewRequestLogRepository(db *gorm.DB, dbType string) *RequestLogRepository {
	return &RequestLogRepository{db: db, dbType: dbType}
}

// Create 创建请求日志
func (r *RequestLogRepository) Create(log *model.RequestLog) error {
	return r.db.Create(log).Error
}

// GetByID 根据ID获取日志（含完整大字段，用于查看详情）
func (r *RequestLogRepository) GetByID(id uint) (*model.RequestLog, error) {
	var requestLog model.RequestLog
	err := r.db.First(&requestLog, id).Error
	if err != nil {
		slog.Error("根据ID获取日志失败", "id", id, "error", err)
		return nil, err
	}
	// 手动填充Provider信息
	r.fillProviderInfo(&requestLog)
	return &requestLog, nil
}

// GetRecent 获取最近的日志（排除 longtext 大字段，批量预加载 Provider 信息）
func (r *RequestLogRepository) GetRecent(limit int) ([]model.RequestLog, error) {
	var logs []model.RequestLog
	err := r.db.Select("id, provider_id, model, input_tokens, output_tokens, total_tokens, cached_tokens, status, error_message, duration, created_at").
		Order("created_at desc").Limit(limit).Find(&logs).Error
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
		r.db.Where("id IN ?", ids).Find(&providers)
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
		if err := r.db.First(&provider, log.ProviderID).Error; err == nil {
			providerMap[log.ProviderID] = provider
		}
	}
	log.Provider = resolveProvider(log.ProviderID, providerMap)
}

// AggregateHour 汇总指定小时的明细数据，返回 per-provider 的 HourlyStat 列表
// 只汇总 aggregated=false 且 status=success 的记录，含 provider_id=0 的全量行
func (r *RequestLogRepository) AggregateHour(hourStart time.Time) ([]model.HourlyStat, error) {
	hourEnd := hourStart.Add(time.Hour)

	type AggResult struct {
		ProviderID    uint
		InputTokens   int64
		OutputTokens  int64
		TotalTokens   int64
		CachedTokens  int64
		RequestCount  int64
		TotalDuration int64
	}

	var results []AggResult
	err := r.db.Model(&model.RequestLog{}).
		Select("provider_id, COALESCE(SUM(input_tokens), 0) as input_tokens, COALESCE(SUM(output_tokens), 0) as output_tokens, COALESCE(SUM(total_tokens), 0) as total_tokens, COALESCE(SUM(cached_tokens), 0) as cached_tokens, COUNT(*) as request_count, COALESCE(SUM(duration), 0) as total_duration").
		Where("created_at >= ? AND created_at < ? AND aggregated = ? AND status = ?", hourStart, hourEnd, false, "success").
		Group("provider_id").
		Scan(&results).Error
	if err != nil {
		return nil, err
	}

	stats := make([]model.HourlyStat, 0, len(results)+1)

	var totalInput, totalOutput, totalToken, totalCached, totalCount, totalDuration int64
	for _, r := range results {
		stats = append(stats, model.HourlyStat{
			Hour:          hourStart,
			ProviderID:    r.ProviderID,
			InputTokens:   r.InputTokens,
			OutputTokens:  r.OutputTokens,
			TotalTokens:   r.TotalTokens,
			CachedTokens:  r.CachedTokens,
			RequestCount:  r.RequestCount,
			TotalDuration: r.TotalDuration,
		})
		totalInput += r.InputTokens
		totalOutput += r.OutputTokens
		totalToken += r.TotalTokens
		totalCached += r.CachedTokens
		totalCount += r.RequestCount
		totalDuration += r.TotalDuration
	}

	// 全量行 (provider_id=0)
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
	return r.db.Model(&model.RequestLog{}).
		Where("created_at >= ? AND created_at < ? AND aggregated = ?", hourStart, hourEnd, false).
		Update("aggregated", true).Error
}

// GetCurrentHourStats 获取当前小时的实时统计（用于混合查询保证实时性）
func (r *RequestLogRepository) GetCurrentHourStats(providerID uint) (*model.TokenStats, error) {
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

	err := r.db.Raw(query, args...).Scan(&stats).Error
	return &stats, err
}

// GetCurrentHourHourlyStats 获取当前小时的分时统计（用于混合查询保证实时性）
func (r *RequestLogRepository) GetCurrentHourHourlyStats(providerID uint) (*model.HourlyStatsResult, error) {
	hourStart := time.Now().Truncate(time.Hour)

	var hourExpr string
	if r.dbType == "sqlite" {
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

	err := r.db.Raw(query, args...).Scan(&result).Error
	return &result, err
}

// GetHourlyStatsByDateFromLogs 从明细表获取指定日期的分时统计（用于历史日期无汇总数据时的回退查询）
func (r *RequestLogRepository) GetHourlyStatsByDateFromLogs(date time.Time) ([]model.HourlyStatsResult, error) {
	dayStart := date.Truncate(24 * time.Hour)
	dayEnd := dayStart.Add(24 * time.Hour)

	var hourExpr string
	if r.dbType == "sqlite" {
		hourExpr = "CAST(strftime('%H', created_at) AS INTEGER)"
	} else {
		hourExpr = "EXTRACT(HOUR FROM created_at)"
	}

	var results []model.HourlyStatsResult
	err := r.db.Raw(fmt.Sprintf(`
		SELECT
			%s as hour,
			COUNT(*) as request_count,
			COALESCE(SUM(total_tokens), 0) as total_tokens,
			COALESCE(SUM(input_tokens), 0) as input_tokens,
			COALESCE(SUM(output_tokens), 0) as output_tokens,
			COALESCE(SUM(cached_tokens), 0) as cached_tokens
		FROM request_logs
		WHERE created_at >= ? AND created_at < ?
			AND status = 'success'
		GROUP BY %s
		ORDER BY hour
	`, hourExpr, hourExpr), dayStart, dayEnd).Scan(&results).Error
	return results, err
}

// GetMinCreatedAt 获取最早记录的创建时间（用于回填起始点）
func (r *RequestLogRepository) GetMinCreatedAt() (*time.Time, error) {
	var nullTime sql.NullTime
	err := r.db.Model(&model.RequestLog{}).
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
	result := r.db.Where("created_at < ? AND aggregated = ?", cutoffDate, true).Delete(&model.RequestLog{})
	return result.RowsAffected, result.Error
}

// CurrentHourBreakdown 当前小时按 provider 拆分的实时统计结果
type CurrentHourBreakdown struct {
	ProviderID   uint  `json:"provider_id"`
	InputTokens  int64 `json:"input_tokens"`
	OutputTokens int64 `json:"output_tokens"`
	TotalTokens  int64 `json:"total_tokens"`
}

// GetCurrentHourBreakdown 获取当前小时按 provider 拆分的实时统计数据（用于堆叠图）
func (r *RequestLogRepository) GetCurrentHourBreakdown() ([]CurrentHourBreakdown, error) {
	hourStart := time.Now().Truncate(time.Hour)

	var results []CurrentHourBreakdown
	err := r.db.Model(&model.RequestLog{}).
		Select("provider_id, COALESCE(SUM(input_tokens), 0) as input_tokens, COALESCE(SUM(output_tokens), 0) as output_tokens, COALESCE(SUM(total_tokens), 0) as total_tokens").
		Where("created_at >= ? AND status = 'success'", hourStart).
		Group("provider_id").
		Scan(&results).Error
	return results, err
}





