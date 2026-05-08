package repository

import (
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

// GetByID 根据ID获取日志
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

// GetRecent 获取最近的日志（批量预加载 Provider 信息，避免 N+1 查询）
func (r *RequestLogRepository) GetRecent(limit int) ([]model.RequestLog, error) {
	var logs []model.RequestLog
	err := r.db.Order("created_at desc").Limit(limit).Find(&logs).Error
	if err != nil {
		return logs, err
	}
	r.fillProviderInfoBatch(logs)
	return logs, nil
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
		if logs[i].ProviderID == model.DeletedProviderID {
			logs[i].Provider = model.ProviderConfig{
				ID:   model.DeletedProviderID,
				Name: "已删除",
			}
		} else if p, ok := providerMap[logs[i].ProviderID]; ok {
			logs[i].Provider = p
		} else {
			logs[i].Provider = model.ProviderConfig{
				ID:   logs[i].ProviderID,
				Name: "未知",
			}
		}
	}
}

// fillProviderInfo 填充Provider信息（ProviderID=DeletedProviderID时显示"已删除"）
func (r *RequestLogRepository) fillProviderInfo(log *model.RequestLog) {
	if log.ProviderID == model.DeletedProviderID {
		log.Provider = model.ProviderConfig{
			ID:   model.DeletedProviderID,
			Name: "已删除",
		}
		return
	}
	// 正常查询Provider
	var provider model.ProviderConfig
	if err := r.db.First(&provider, log.ProviderID).Error; err == nil {
		log.Provider = provider
	} else {
		// Provider不存在，显示未知
		log.Provider = model.ProviderConfig{
			ID:   log.ProviderID,
			Name: "未知",
		}
	}
}

// GetStatsByDate 按日期获取统计
func (r *RequestLogRepository) GetStatsByDate(startDate, endDate time.Time) ([]model.TokenStats, error) {
	var stats []model.TokenStats
	
	err := r.db.Raw(`
		SELECT 
			DATE(created_at) as date,
			COALESCE(SUM(input_tokens), 0) as total_input_tokens,
			COALESCE(SUM(output_tokens), 0) as total_output_tokens,
			COALESCE(SUM(total_tokens), 0) as total_tokens,
			COALESCE(SUM(cached_tokens), 0) as total_cached_tokens,
			COUNT(*) as request_count
		FROM request_logs
		WHERE created_at BETWEEN ? AND ?
			AND status = 'success'
		GROUP BY DATE(created_at)
		ORDER BY date desc
	`, startDate, endDate).Scan(&stats).Error
	
	return stats, err
}

// GetTodayStats 获取今日统计
func (r *RequestLogRepository) GetTodayStats() (*model.TokenStats, error) {
	today := time.Now().Format("2006-01-02")
	startOfDay, _ := time.Parse("2006-01-02", today)
	endOfDay := startOfDay.Add(24 * time.Hour)
	
	var stats model.TokenStats
	err := r.db.Raw(`
		SELECT 
			? as date,
			COALESCE(SUM(input_tokens), 0) as total_input_tokens,
			COALESCE(SUM(output_tokens), 0) as total_output_tokens,
			COALESCE(SUM(total_tokens), 0) as total_tokens,
			COALESCE(SUM(cached_tokens), 0) as total_cached_tokens,
			COUNT(*) as request_count
		FROM request_logs
		WHERE created_at BETWEEN ? AND ?
			AND status = 'success'
	`, today, startOfDay, endOfDay).Scan(&stats).Error
	
	return &stats, err
}

// GetWeekStats 获取本周统计
func (r *RequestLogRepository) GetWeekStats() (*model.TokenStats, error) {
	now := time.Now()
	weekday := int(now.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	startOfWeek := now.AddDate(0, 0, -weekday+1).Truncate(24 * time.Hour)
	
	var stats model.TokenStats
	err := r.db.Raw(`
		SELECT 
			? as date,
			COALESCE(SUM(input_tokens), 0) as total_input_tokens,
			COALESCE(SUM(output_tokens), 0) as total_output_tokens,
			COALESCE(SUM(total_tokens), 0) as total_tokens,
			COALESCE(SUM(cached_tokens), 0) as total_cached_tokens,
			COUNT(*) as request_count
		FROM request_logs
		WHERE created_at >= ?
			AND status = 'success'
	`, fmt.Sprintf("%s ~ %s", startOfWeek.Format("01/02"), now.Format("01/02")), startOfWeek).Scan(&stats).Error
	
	return &stats, err
}

// GetTotalStats 获取总计统计
func (r *RequestLogRepository) GetTotalStats() (*model.TokenStats, error) {
	var stats model.TokenStats
	err := r.db.Raw(`
		SELECT 
			'total' as date,
			COALESCE(SUM(input_tokens), 0) as total_input_tokens,
			COALESCE(SUM(output_tokens), 0) as total_output_tokens,
			COALESCE(SUM(total_tokens), 0) as total_tokens,
			COALESCE(SUM(cached_tokens), 0) as total_cached_tokens,
			COUNT(*) as request_count
		FROM request_logs
		WHERE status = 'success'
	`).Scan(&stats).Error
	
	return &stats, err
}

// GetLast30DaysStats 获取最近30天统计
func (r *RequestLogRepository) GetLast30DaysStats() ([]model.TokenStats, error) {
	endDate := time.Now()
	startDate := endDate.AddDate(0, 0, -30)
	return r.GetStatsByDate(startDate, endDate)
}

// HourlyStats 每小时统计数据
type HourlyStats struct {
	Hour          int   `json:"hour"`
	RequestCount  int64 `json:"request_count"`
	TotalTokens   int64 `json:"total_tokens"`
	InputTokens   int64 `json:"input_tokens"`
	OutputTokens  int64 `json:"output_tokens"`
	CachedTokens  int64 `json:"cached_tokens"`
}

// GetTodayHourlyStats 获取今日分时统计
func (r *RequestLogRepository) GetTodayHourlyStats() ([]HourlyStats, error) {
	today := time.Now().Format("2006-01-02")
	startOfDay, _ := time.Parse("2006-01-02", today)
	endOfDay := startOfDay.Add(24 * time.Hour)

	// 根据数据库类型选择兼容的 SQL 语法
	var hourExpr string
	if r.dbType == "sqlite" {
		hourExpr = "CAST(strftime('%H', created_at) AS INTEGER)"
	} else {
		hourExpr = "EXTRACT(HOUR FROM created_at)"
	}

	query := fmt.Sprintf(`
		SELECT 
			%s as hour,
			COUNT(*) as request_count,
			COALESCE(SUM(total_tokens), 0) as total_tokens,
			COALESCE(SUM(input_tokens), 0) as input_tokens,
			COALESCE(SUM(output_tokens), 0) as output_tokens,
			COALESCE(SUM(cached_tokens), 0) as cached_tokens
		FROM request_logs
		WHERE created_at BETWEEN ? AND ?
			AND status = 'success'
		GROUP BY %s
		ORDER BY hour
	`, hourExpr, hourExpr)

	var stats []HourlyStats
	err := r.db.Raw(query, startOfDay, endOfDay).Scan(&stats).Error

	return stats, err
}

// CleanOldRequestBodies 清理指定天数前的请求体和响应体
func (r *RequestLogRepository) CleanOldRequestBodies(days int) (int64, error) {
	cutoffDate := time.Now().AddDate(0, 0, -days)
	result := r.db.Exec(`
		UPDATE request_logs
		SET request_body = '', response_body = '', thinking_content = ''
		WHERE created_at < ? AND (request_body != '' OR response_body != '' OR thinking_content != '')
	`, cutoffDate)
	return result.RowsAffected, result.Error
}
