package repository

import (
	"fmt"
	"llm-proxy/internal/model"
	"log/slog"
	"time"

	"gorm.io/gorm"
)

type RequestLogRepository struct {
	db *gorm.DB
}

func NewRequestLogRepository(db *gorm.DB) *RequestLogRepository {
	return &RequestLogRepository{db: db}
}

// Create 创建请求日志
func (r *RequestLogRepository) Create(log *model.RequestLog) error {
	return r.db.Create(log).Error
}

// GetByID 根据ID获取日志
func (r *RequestLogRepository) GetByID(id uint) (*model.RequestLog, error) {
	var requestLog model.RequestLog
	err := r.db.Preload("Provider").First(&requestLog, id).Error
	if err != nil {
		slog.Error("根据ID获取日志失败", "id", id, "error", err)
		return nil, err
	}
	return &requestLog, nil
}

// GetRecent 获取最近的日志
func (r *RequestLogRepository) GetRecent(limit int) ([]model.RequestLog, error) {
	var logs []model.RequestLog
	err := r.db.Preload("Provider").Order("created_at desc").Limit(limit).Find(&logs).Error
	return logs, err
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
