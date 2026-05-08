package service

import (
	"llm-proxy/internal/model"
	"llm-proxy/internal/repository"
	"log/slog"
	"time"
)

type StatsService struct {
	hourlyStatRepo *repository.HourlyStatRepository
	requestLogRepo *repository.RequestLogRepository
}

func NewStatsService(hourlyStatRepo *repository.HourlyStatRepository, requestLogRepo *repository.RequestLogRepository) *StatsService {
	return &StatsService{
		hourlyStatRepo: hourlyStatRepo,
		requestLogRepo: requestLogRepo,
	}
}

// addTokenStats 将增量合并到基础统计
func addTokenStats(base, delta *model.TokenStats) {
	if base == nil || delta == nil {
		return
	}
	base.TotalInputTokens += delta.TotalInputTokens
	base.TotalOutputTokens += delta.TotalOutputTokens
	base.TotalTokens += delta.TotalTokens
	base.TotalCachedTokens += delta.TotalCachedTokens
	base.RequestCount += delta.RequestCount
}

// GetDashboardStats 获取仪表盘统计数据（汇总表历史 + 明细表当前小时，保证实时性）
func (s *StatsService) GetDashboardStats() (map[string]*model.TokenStats, error) {
	result := make(map[string]*model.TokenStats)

	// 从汇总表获取历史已完成小时的统计
	todayStats, weekStats, totalStats, err := s.hourlyStatRepo.GetDashboardStats()
	if err != nil {
		slog.Error("获取汇总统计失败", "error", err)
		return nil, err
	}

	// 从明细表获取当前小时的实时统计
	currentHourStats, err := s.requestLogRepo.GetCurrentHourStats()
	if err != nil {
		slog.Warn("获取当前小时统计失败，仅使用汇总数据", "error", err)
		currentHourStats = &model.TokenStats{}
	}

	// 合并：汇总表 + 当前小时实时
	addTokenStats(todayStats, currentHourStats)
	addTokenStats(weekStats, currentHourStats)
	addTokenStats(totalStats, currentHourStats)

	result["today"] = todayStats
	result["week"] = weekStats
	result["total"] = totalStats

	return result, nil
}

// GetLast30DaysStats 获取最近30天统计（汇总表历史 + 明细表当前小时）
func (s *StatsService) GetLast30DaysStats() ([]model.TokenStats, error) {
	// 从汇总表获取历史每日统计
	dailyStats, err := s.hourlyStatRepo.GetDailyStats(30)
	if err != nil {
		slog.Error("获取每日统计失败", "error", err)
		return nil, err
	}

	// 从明细表获取当前小时的实时统计
	currentHourStats, err := s.requestLogRepo.GetCurrentHourStats()
	if err != nil {
		slog.Warn("获取当前小时统计失败", "error", err)
		return dailyStats, nil
	}

	// 将当前小时数据合并到今日统计中
	todayStr := currentHourStats.Date
	for i := range dailyStats {
		if dailyStats[i].Date == todayStr {
			addTokenStats(&dailyStats[i], currentHourStats)
			return dailyStats, nil
		}
	}

	// 今日汇总记录不存在（当前小时是今天第一个数据），追加一条
	dailyStats = append(dailyStats, model.TokenStats{
		Date:              todayStr,
		TotalInputTokens:  currentHourStats.TotalInputTokens,
		TotalOutputTokens: currentHourStats.TotalOutputTokens,
		TotalTokens:       currentHourStats.TotalTokens,
		TotalCachedTokens: currentHourStats.TotalCachedTokens,
		RequestCount:      currentHourStats.RequestCount,
	})

	return dailyStats, nil
}

// GetRecentLogs 获取最近的请求日志
func (s *StatsService) GetRecentLogs(limit int) ([]model.RequestLog, error) {
	return s.requestLogRepo.GetRecent(limit)
}

// GetLogDetail 获取单条请求日志详情
func (s *StatsService) GetLogDetail(id uint) (*model.RequestLog, error) {
	return s.requestLogRepo.GetByID(id)
}

// GetTodayHourlyStats 获取今日分时统计（汇总表已完成小时 + 明细表当前小时，保证实时性）
func (s *StatsService) GetTodayHourlyStats() ([]repository.HourlyStatsResult, error) {
	// 从汇总表获取今日已完成小时的统计
	hourlyStats, err := s.hourlyStatRepo.GetTodayHourlyStats()
	if err != nil {
		slog.Error("获取分时汇总统计失败", "error", err)
		return nil, err
	}

	// 从明细表获取当前小时的实时统计
	currentHourResult, err := s.requestLogRepo.GetCurrentHourHourlyStats()
	if err != nil {
		slog.Warn("获取当前小时分时统计失败", "error", err)
		return hourlyStats, nil
	}

	// 追加当前小时
	hourlyStats = append(hourlyStats, *currentHourResult)

	return hourlyStats, nil
}

// GetHourlyStatsByDate 获取指定日期的分时统计
// 今日：汇总表已完成小时 + 明细表当前小时（保证实时性）
// 历史日期：优先从汇总表读取，若汇总表无数据则回退到明细表查询
func (s *StatsService) GetHourlyStatsByDate(date time.Time) ([]repository.HourlyStatsResult, error) {
	now := time.Now()
	isToday := date.Truncate(24*time.Hour).Equal(now.Truncate(24*time.Hour))

	if isToday {
		return s.GetTodayHourlyStats()
	}

	// 历史日期：优先从汇总表读取
	hourlyStats, err := s.hourlyStatRepo.GetHourlyStatsByDate(date)
	if err != nil {
		slog.Error("获取指定日期分时汇总统计失败", "date", date.Format("2006-01-02"), "error", err)
		return nil, err
	}

	// 汇总表有数据则直接返回
	if len(hourlyStats) > 0 {
		return hourlyStats, nil
	}

	// 汇总表无数据，回退到明细表查询
	slog.Info("汇总表无数据，回退到明细表查询", "date", date.Format("2006-01-02"))
	results, err := s.requestLogRepo.GetHourlyStatsByDateFromLogs(date)
	if err != nil {
		slog.Error("从明细表获取指定日期分时统计失败", "date", date.Format("2006-01-02"), "error", err)
		return nil, err
	}

	return results, nil
}
