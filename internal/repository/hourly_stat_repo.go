package repository

import (
	"fmt"
	"github.com/wanglejiu/llm-proxy/internal/model"
	"sort"
	"time"

	"gorm.io/gorm"
)

// HourlyStatRepository 汇总表数据访问
type HourlyStatRepository struct {
	dbManager *DBManager
}

func NewHourlyStatRepository(dbManager *DBManager) *HourlyStatRepository {
	return &HourlyStatRepository{dbManager: dbManager}
}

// Upsert 插入或累加更新汇总记录（原子操作）
// 行级 aggregated 标记保证同一条明细不会被重复汇总，此处累加是安全的
func (r *HourlyStatRepository) Upsert(stat *model.HourlyStat) error {
	if r.dbManager.GetDBType() == "sqlite" {
		return r.upsertSQLite(stat)
	}
	return r.upsertMySQL(stat)
}

// upsertSQLite 使用 SQLite 的 INSERT OR REPLACE 原子操作
func (r *HourlyStatRepository) upsertSQLite(stat *model.HourlyStat) error {
	var existing model.HourlyStat
	result := r.dbManager.GetDB().Where("hour = ? AND provider_id = ? AND model = ?", stat.Hour, stat.ProviderID, stat.Model).First(&existing)
	if result.Error == gorm.ErrRecordNotFound {
		return r.dbManager.GetDB().Create(stat).Error
	}
	if result.Error != nil {
		return result.Error
	}
	return r.dbManager.GetDB().Model(&model.HourlyStat{}).
		Where("hour = ? AND provider_id = ? AND model = ?", stat.Hour, stat.ProviderID, stat.Model).
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
	return r.dbManager.GetDB().Exec(`
		INSERT INTO hourly_stats (hour, provider_id, model, input_tokens, output_tokens, total_tokens, cached_tokens, request_count, total_duration, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE
			input_tokens = input_tokens + VALUES(input_tokens),
			output_tokens = output_tokens + VALUES(output_tokens),
			total_tokens = total_tokens + VALUES(total_tokens),
			cached_tokens = cached_tokens + VALUES(cached_tokens),
			request_count = request_count + VALUES(request_count),
			total_duration = total_duration + VALUES(total_duration),
			updated_at = VALUES(updated_at)
	`, stat.Hour, stat.ProviderID, stat.Model, stat.InputTokens, stat.OutputTokens, stat.TotalTokens, stat.CachedTokens, stat.RequestCount, stat.TotalDuration, now, now).Error
}

// GetByHourRange 获取指定时间范围内的汇总记录
func (r *HourlyStatRepository) GetByHourRange(start, end time.Time) ([]model.HourlyStat, error) {
	var stats []model.HourlyStat
	err := r.dbManager.GetDB().Where("hour >= ? AND hour < ?", start, end).
		Order("hour asc").
		Find(&stats).Error
	return stats, err
}

// GetDashboardStats 从汇总表获取仪表盘统计（历史已完成小时，provider 级别）
func (r *HourlyStatRepository) GetDashboardStats(providerID uint) (todayStats, weekStats, totalStats *model.TokenStats, err error) {
	now := time.Now()
	today := now.Truncate(24 * time.Hour)
	weekday := int(now.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	startOfWeek := now.AddDate(0, 0, -weekday+1).Truncate(24 * time.Hour)

	// 一次查询获取所有汇总数据（model='' 表示 provider 级别合计），在内存中分桶
	var stats []model.HourlyStat
	if err = r.dbManager.GetDB().Where("hour < ? AND provider_id = ? AND model = ''", currentHourStart(), providerID).Find(&stats).Error; err != nil {
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

// GetDailyStats 从汇总表获取每日统计（provider 级别）
func (r *HourlyStatRepository) GetDailyStats(days int, providerID uint) ([]model.TokenStats, error) {
	startDate := time.Now().AddDate(0, 0, -days)
	var stats []model.HourlyStat
	if err := r.dbManager.GetDB().Where("hour >= ? AND provider_id = ? AND model = ''", startDate.Truncate(24*time.Hour), providerID).Find(&stats).Error; err != nil {
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

// GetTodayHourlyStats 从汇总表获取今日已完成小时的分时统计（provider 级别）
func (r *HourlyStatRepository) GetTodayHourlyStats(providerID uint) ([]model.HourlyStatsResult, error) {
	today := time.Now().Truncate(24 * time.Hour)
	var stats []model.HourlyStat
	if err := r.dbManager.GetDB().Where("hour >= ? AND hour < ? AND provider_id = ? AND model = ''", today, currentHourStart(), providerID).Find(&stats).Error; err != nil {
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

// GetHourlyStatsByDate 获取指定日期的分时统计（provider 级别，历史日期从汇总表读取）
func (r *HourlyStatRepository) GetHourlyStatsByDate(date time.Time, providerID uint) ([]model.HourlyStatsResult, error) {
	dayStart := date.Truncate(24 * time.Hour)
	dayEnd := dayStart.Add(24 * time.Hour)
	now := time.Now()
	isToday := dayStart.Equal(now.Truncate(24 * time.Hour))

	var stats []model.HourlyStat
	var err error

	if isToday {
		// 今日：汇总表已完成小时
		err = r.dbManager.GetDB().Where("hour >= ? AND hour < ? AND provider_id = ? AND model = ''", dayStart, currentHourStart(), providerID).Find(&stats).Error
	} else {
		// 历史日期：汇总表全天
		err = r.dbManager.GetDB().Where("hour >= ? AND hour < ? AND provider_id = ? AND model = ''", dayStart, dayEnd, providerID).Find(&stats).Error
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

// GetMissingHours 获取指定范围内缺失全局汇总行(provider_id=0, model='')的小时列表
func (r *HourlyStatRepository) GetMissingHours(start, end time.Time) ([]time.Time, error) {
	var existingHours []time.Time
	if err := r.dbManager.GetDB().Model(&model.HourlyStat{}).
		Where("hour >= ? AND hour < ? AND provider_id = 0 AND model = ''", start, end).
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

// GetMissingProviderHours 获取指定范围内缺失 per-provider 汇总行(model='')的小时列表
// 返回缺少 provider 级别合计行 (provider_id>0, model='') 的 (hour, provider_id) 组合
func (r *HourlyStatRepository) GetMissingProviderHours(start, end time.Time) ([]time.Time, error) {
	type HourProvider struct {
		Hour       time.Time
		ProviderID uint
	}
	var existing []HourProvider
	if err := r.dbManager.GetDB().Model(&model.HourlyStat{}).
		Select("hour, provider_id").
		Where("hour >= ? AND hour < ? AND provider_id > 0 AND model = ''", start, end).
		Find(&existing).Error; err != nil {
		return nil, err
	}

	existingSet := make(map[string]struct{})
	for _, e := range existing {
		key := fmt.Sprintf("%d_%d", e.Hour.Unix(), e.ProviderID)
		existingSet[key] = struct{}{}
	}

	// 获取所有活跃 provider ID
	var providerIDs []uint
	if err := r.dbManager.GetDB().Model(&model.HourlyStat{}).
		Where("hour >= ? AND hour < ? AND provider_id > 0", start, end).
		Distinct("provider_id").
		Pluck("provider_id", &providerIDs).Error; err != nil {
		return nil, err
	}

	// 检查每个 (hour, provider) 组合是否都有合计行
	missingHours := make(map[time.Time]struct{})
	for h := start; h.Before(end); h = h.Add(time.Hour) {
		truncated := h.Truncate(time.Hour)
		for _, pid := range providerIDs {
			key := fmt.Sprintf("%d_%d", truncated.Unix(), pid)
			if _, ok := existingSet[key]; !ok {
				missingHours[truncated] = struct{}{}
			}
		}
	}

	var missing []time.Time
	for h := range missingHours {
		missing = append(missing, h)
	}
	return missing, nil
}

// GetModelDailyStats 从汇总表获取模型级别每日统计（provider_id>0, model!=''）
// 用于与 request_logs 的当前小时数据合并，实现混合查询
func (r *HourlyStatRepository) GetModelDailyStats(providerID uint) ([]ModelDailyStatResult, error) {
	var dateExpr string
	if r.dbManager.GetDBType() == "sqlite" {
		// SQLite 的 time.Time 存储为 ISO 8601（如 2026-06-09T00:00:00+08:00）
		// strftime 可正确处理带时区的格式，返回 YYYY-MM-DD
		dateExpr = "strftime('%Y-%m-%d', hour)"
	} else {
		dateExpr = "DATE(hour)"
	}

	query := fmt.Sprintf(`SELECT
		%s as date,
		provider_id,
		model,
		COALESCE(SUM(input_tokens), 0) as total_input_tokens,
		COALESCE(SUM(output_tokens), 0) as total_output_tokens,
		COALESCE(SUM(total_tokens), 0) as total_tokens,
		COALESCE(SUM(cached_tokens), 0) as total_cached_tokens,
		COALESCE(SUM(request_count), 0) as request_count
	FROM hourly_stats
	WHERE model <> '' AND provider_id > 0`, dateExpr)

	var args []interface{}
	if providerID > 0 {
		query += " AND provider_id = ?"
		args = append(args, providerID)
	}

	query += " GROUP BY date, provider_id, model ORDER BY date DESC, total_tokens DESC"

	var results []ModelDailyStatResult
	err := r.dbManager.GetDB().Raw(query, args...).Scan(&results).Error
	if err != nil {
		return nil, err
	}
	if results == nil {
		return []ModelDailyStatResult{}, nil
	}
	return results, nil
}

// ModelDailyStatResult 模型级别每日统计结果（用于 GetModelDailyStats）
type ModelDailyStatResult struct {
	Date              string `json:"date"`
	ProviderID        uint   `json:"provider_id"`
	Model             string `json:"model"`
	TotalInputTokens  int64  `json:"total_input_tokens"`
	TotalOutputTokens int64  `json:"total_output_tokens"`
	TotalTokens       int64  `json:"total_tokens"`
	TotalCachedTokens int64  `json:"total_cached_tokens"`
	RequestCount      int64  `json:"request_count"`
}

// GetHourlyStatsWithBreakdown 获取指定范围内按 (hour, provider_id, model) 拆分的模型级别详细数据
// providerID=0 时返回所有 provider 的数据，providerID>0 时只返回指定 provider 的数据
func (r *HourlyStatRepository) GetHourlyStatsWithBreakdown(start, end time.Time, providerID uint) ([]model.HourlyStat, error) {
	var stats []model.HourlyStat
	query := r.dbManager.GetDB().Where("hour >= ? AND hour < ? AND provider_id > 0 AND model <> ''", start, end)
	if providerID > 0 {
		query = query.Where("provider_id = ?", providerID)
	}
	err := query.Order("hour asc, provider_id asc, model asc").
		Find(&stats).Error
	return stats, err
}

// GetHourlyModelStatsFromSummary 从汇总表获取指定日期指定模型的分时统计
func (r *HourlyStatRepository) GetHourlyModelStatsFromSummary(date time.Time, providerID uint, modelName string) ([]model.HourlyStatsResult, error) {
	dayStart := date.Truncate(24 * time.Hour)
	dayEnd := dayStart.Add(24 * time.Hour)

	var results []model.HourlyStatsResult
	err := r.dbManager.GetDB().Model(&model.HourlyStat{}).
		Select("HOUR(hour) as hour, SUM(request_count) as request_count, SUM(total_tokens) as total_tokens, SUM(input_tokens) as input_tokens, SUM(output_tokens) as output_tokens, SUM(cached_tokens) as cached_tokens").
		Where("hour >= ? AND hour < ? AND provider_id = ? AND model = ?", dayStart, dayEnd, providerID, modelName).
		Group("HOUR(hour)").
		Order("hour").
		Scan(&results).Error
	return results, err
}

// currentHourStart 返回当前小时的起始时间
func currentHourStart() time.Time {
	return time.Now().Truncate(time.Hour)
}
