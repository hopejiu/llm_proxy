package repository

import (
	"fmt"
	"llm-proxy/internal/model"
	"sort"
	"time"

	"gorm.io/gorm"
)

// HourlyStatRepository 汇总表数据访问
type HourlyStatRepository struct {
	db     *gorm.DB
	dbType string
}

func NewHourlyStatRepository(db *gorm.DB, dbType string) *HourlyStatRepository {
	return &HourlyStatRepository{db: db, dbType: dbType}
}

// Upsert 插入或累加更新汇总记录（原子操作）
// 行级 aggregated 标记保证同一条明细不会被重复汇总，此处累加是安全的
func (r *HourlyStatRepository) Upsert(stat *model.HourlyStat) error {
	if r.dbType == "sqlite" {
		return r.upsertSQLite(stat)
	}
	return r.upsertMySQL(stat)
}

// upsertSQLite 使用 SQLite 的 INSERT OR REPLACE 原子操作
func (r *HourlyStatRepository) upsertSQLite(stat *model.HourlyStat) error {
	// 先尝试插入，如果冲突则累加更新
	result := r.db.Where("hour = ?", stat.Hour).First(&model.HourlyStat{})
	if result.Error == gorm.ErrRecordNotFound {
		return r.db.Create(stat).Error
	}
	if result.Error != nil {
		return result.Error
	}
	// 累加更新
	return r.db.Model(&model.HourlyStat{}).
		Where("hour = ?", stat.Hour).
		Updates(map[string]interface{}{
			"input_tokens":   gorm.Expr("input_tokens + ?", stat.InputTokens),
			"output_tokens":  gorm.Expr("output_tokens + ?", stat.OutputTokens),
			"total_tokens":   gorm.Expr("total_tokens + ?", stat.TotalTokens),
			"cached_tokens":  gorm.Expr("cached_tokens + ?", stat.CachedTokens),
			"request_count":  gorm.Expr("request_count + ?", stat.RequestCount),
			"total_duration": gorm.Expr("total_duration + ?", stat.TotalDuration),
		}).Error
}

// upsertMySQL 使用 MySQL 的 ON DUPLICATE KEY UPDATE 原子操作
func (r *HourlyStatRepository) upsertMySQL(stat *model.HourlyStat) error {
	now := time.Now()
	return r.db.Exec(`
		INSERT INTO hourly_stats (hour, input_tokens, output_tokens, total_tokens, cached_tokens, request_count, total_duration, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE
			input_tokens = input_tokens + VALUES(input_tokens),
			output_tokens = output_tokens + VALUES(output_tokens),
			total_tokens = total_tokens + VALUES(total_tokens),
			cached_tokens = cached_tokens + VALUES(cached_tokens),
			request_count = request_count + VALUES(request_count),
			total_duration = total_duration + VALUES(total_duration),
			updated_at = VALUES(updated_at)
	`, stat.Hour, stat.InputTokens, stat.OutputTokens, stat.TotalTokens, stat.CachedTokens, stat.RequestCount, stat.TotalDuration, now, now).Error
}

// GetByHourRange 获取指定时间范围内的汇总记录
func (r *HourlyStatRepository) GetByHourRange(start, end time.Time) ([]model.HourlyStat, error) {
	var stats []model.HourlyStat
	err := r.db.Where("hour >= ? AND hour < ?", start, end).
		Order("hour asc").
		Find(&stats).Error
	return stats, err
}

// GetDashboardStats 从汇总表获取仪表盘统计（历史已完成小时）
func (r *HourlyStatRepository) GetDashboardStats() (todayStats, weekStats, totalStats *model.TokenStats, err error) {
	now := time.Now()
	today := now.Truncate(24 * time.Hour)
	weekday := int(now.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	startOfWeek := now.AddDate(0, 0, -weekday+1).Truncate(24 * time.Hour)

	// 一次查询获取所有汇总数据，在内存中分桶
	var stats []model.HourlyStat
	if err = r.db.Where("hour < ?", currentHourStart()).Find(&stats).Error; err != nil {
		return
	}

	todayStats = &model.TokenStats{Date: today.Format("2006-01-02")}
	weekStats = &model.TokenStats{Date: fmt.Sprintf("%s ~ %s", startOfWeek.Format("01/02"), now.Format("01/02"))}
	totalStats = &model.TokenStats{Date: "total"}

	for _, s := range stats {
		// 总计
		totalStats.TotalInputTokens += s.InputTokens
		totalStats.TotalOutputTokens += s.OutputTokens
		totalStats.TotalTokens += s.TotalTokens
		totalStats.TotalCachedTokens += s.CachedTokens
		totalStats.RequestCount += s.RequestCount

		// 本周
		if !s.Hour.Before(startOfWeek) {
			weekStats.TotalInputTokens += s.InputTokens
			weekStats.TotalOutputTokens += s.OutputTokens
			weekStats.TotalTokens += s.TotalTokens
			weekStats.TotalCachedTokens += s.CachedTokens
			weekStats.RequestCount += s.RequestCount
		}

		// 今日
		if !s.Hour.Before(today) {
			todayStats.TotalInputTokens += s.InputTokens
			todayStats.TotalOutputTokens += s.OutputTokens
			todayStats.TotalTokens += s.TotalTokens
			todayStats.TotalCachedTokens += s.CachedTokens
			todayStats.RequestCount += s.RequestCount
		}
	}

	return
}

// GetDailyStats 从汇总表获取每日统计
func (r *HourlyStatRepository) GetDailyStats(days int) ([]model.TokenStats, error) {
	startDate := time.Now().AddDate(0, 0, -days)
	var stats []model.HourlyStat
	if err := r.db.Where("hour >= ?", startDate.Truncate(24*time.Hour)).Find(&stats).Error; err != nil {
		return nil, err
	}

	// 按日期分组
	dailyMap := make(map[string]*model.TokenStats)
	for _, s := range stats {
		dateStr := s.Hour.Format("2006-01-02")
		if _, ok := dailyMap[dateStr]; !ok {
			dailyMap[dateStr] = &model.TokenStats{Date: dateStr}
		}
		d := dailyMap[dateStr]
		d.TotalInputTokens += s.InputTokens
		d.TotalOutputTokens += s.OutputTokens
		d.TotalTokens += s.TotalTokens
		d.TotalCachedTokens += s.CachedTokens
		d.RequestCount += s.RequestCount
	}

	// 转为有序切片（按日期升序）
	result := make([]model.TokenStats, 0, len(dailyMap))
	for _, v := range dailyMap {
		result = append(result, *v)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Date < result[j].Date
	})
	return result, nil
}

// GetTodayHourlyStats 从汇总表获取今日已完成小时的分时统计
func (r *HourlyStatRepository) GetTodayHourlyStats() ([]model.HourlyStatsResult, error) {
	today := time.Now().Truncate(24 * time.Hour)
	var stats []model.HourlyStat
	if err := r.db.Where("hour >= ? AND hour < ?", today, currentHourStart()).Find(&stats).Error; err != nil {
		return nil, err
	}

	result := make([]model.HourlyStatsResult, len(stats))
	for i, s := range stats {
		result[i] = model.HourlyStatsResult{
			Hour:         s.Hour.Hour(),
			RequestCount: s.RequestCount,
			TotalTokens:  s.TotalTokens,
			InputTokens:  s.InputTokens,
			OutputTokens: s.OutputTokens,
			CachedTokens: s.CachedTokens,
		}
	}
	return result, nil
}

// GetHourlyStatsByDate 获取指定日期的分时统计（历史日期从汇总表读取，今日追加当前小时实时数据）
func (r *HourlyStatRepository) GetHourlyStatsByDate(date time.Time) ([]model.HourlyStatsResult, error) {
	dayStart := date.Truncate(24 * time.Hour)
	dayEnd := dayStart.Add(24 * time.Hour)
	now := time.Now()
	isToday := dayStart.Equal(now.Truncate(24 * time.Hour))

	var stats []model.HourlyStat
	var err error

	if isToday {
		// 今日：汇总表已完成小时
		err = r.db.Where("hour >= ? AND hour < ?", dayStart, currentHourStart()).Find(&stats).Error
	} else {
		// 历史日期：汇总表全天
		err = r.db.Where("hour >= ? AND hour < ?", dayStart, dayEnd).Find(&stats).Error
	}
	if err != nil {
		return nil, err
	}

	result := make([]model.HourlyStatsResult, len(stats))
	for i, s := range stats {
		result[i] = model.HourlyStatsResult{
			Hour:         s.Hour.Hour(),
			RequestCount: s.RequestCount,
			TotalTokens:  s.TotalTokens,
			InputTokens:  s.InputTokens,
			OutputTokens: s.OutputTokens,
			CachedTokens: s.CachedTokens,
		}
	}
	return result, nil
}

// GetMissingHours 获取指定范围内缺失汇总的小时列表
func (r *HourlyStatRepository) GetMissingHours(start, end time.Time) ([]time.Time, error) {
	var existingHours []time.Time
	if err := r.db.Model(&model.HourlyStat{}).
		Where("hour >= ? AND hour < ?", start, end).
		Pluck("hour", &existingHours).Error; err != nil {
		return nil, err
	}

	existingSet := make(map[time.Time]struct{})
	for _, h := range existingHours {
		existingSet[h.Truncate(time.Hour)] = struct{}{}
	}

	var missing []time.Time
	for h := start; h.Before(end); h = h.Add(time.Hour) {
		truncated := h.Truncate(time.Hour)
		if _, ok := existingSet[truncated]; !ok {
			missing = append(missing, truncated)
		}
	}
	return missing, nil
}

// currentHourStart 返回当前小时的起始时间
func currentHourStart() time.Time {
	return time.Now().Truncate(time.Hour)
}
