package service

import (
	"fmt"
	"llm-proxy/internal/model"
	"llm-proxy/internal/repository"
	"log/slog"
	"time"
)

type StatsService struct {
	hourlyStatRepo  *repository.HourlyStatRepository
	requestLogRepo  *repository.RequestLogRepository
	providerService *ProviderService
}

func NewStatsService(hourlyStatRepo *repository.HourlyStatRepository, requestLogRepo *repository.RequestLogRepository, providerService *ProviderService) *StatsService {
	return &StatsService{
		hourlyStatRepo:  hourlyStatRepo,
		requestLogRepo:  requestLogRepo,
		providerService: providerService,
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
func (s *StatsService) GetDashboardStats(providerID uint) (map[string]*model.TokenStats, error) {
	result := make(map[string]*model.TokenStats)

	// 从汇总表获取历史已完成小时的统计
	todayStats, weekStats, totalStats, err := s.hourlyStatRepo.GetDashboardStats(providerID)
	if err != nil {
		slog.Error("获取汇总统计失败", "error", err)
		return nil, err
	}

	// 从明细表获取当前小时的实时统计
	currentHourStats, err := s.requestLogRepo.GetCurrentHourStats(providerID)
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
func (s *StatsService) GetLast30DaysStats(providerID uint) ([]model.TokenStats, error) {
	// 从汇总表获取历史每日统计
	dailyStats, err := s.hourlyStatRepo.GetDailyStats(30, providerID)
	if err != nil {
		slog.Error("获取每日统计失败", "error", err)
		return nil, err
	}

	// 从明细表获取当前小时的实时统计
	currentHourStats, err := s.requestLogRepo.GetCurrentHourStats(providerID)
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
func (s *StatsService) GetTodayHourlyStats(providerID uint) ([]model.HourlyStatsResult, error) {
	// 从汇总表获取今日已完成小时的统计
	hourlyStats, err := s.hourlyStatRepo.GetTodayHourlyStats(providerID)
	if err != nil {
		slog.Error("获取分时汇总统计失败", "error", err)
		return nil, err
	}

	// 从明细表获取当前小时的实时统计
	currentHourResult, err := s.requestLogRepo.GetCurrentHourHourlyStats(providerID)
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
func (s *StatsService) GetHourlyStatsByDate(date time.Time, providerID uint) ([]model.HourlyStatsResult, error) {
	now := time.Now()
	isToday := date.Truncate(24*time.Hour).Equal(now.Truncate(24*time.Hour))

	if isToday {
		return s.GetTodayHourlyStats(providerID)
	}

	// 历史日期：优先从汇总表读取
	hourlyStats, err := s.hourlyStatRepo.GetHourlyStatsByDate(date, providerID)
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

// HourlyBreakdownItem 分时详细拆分行（Service 层内部结构体）
type HourlyBreakdownItem struct {
	Hour         int    `json:"hour"`
	ProviderID   uint   `json:"provider_id"`
	ProviderName string `json:"provider_name"`
	InputTokens  int64  `json:"input_tokens"`
	OutputTokens int64  `json:"output_tokens"`
	TotalTokens  int64  `json:"total_tokens"`
}

// GetHourlyStatsByDateWithBreakdown 获取指定日期按 provider 拆分的每小时数据（用于前端堆叠图）
func (s *StatsService) GetHourlyStatsByDateWithBreakdown(dateStr string) ([]HourlyBreakdownItem, error) {
	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return nil, fmt.Errorf("日期格式错误: %w", err)
	}

	dayStart := date.Truncate(24 * time.Hour)
	dayEnd := dayStart.Add(24 * time.Hour)
	now := time.Now()
	isToday := dayStart.Equal(now.Truncate(24 * time.Hour))

	var items []HourlyBreakdownItem

	if isToday {
		// 今日：汇总表已完成小时 + 明细表当前小时
		stats, err := s.hourlyStatRepo.GetHourlyStatsWithBreakdown(dayStart, now.Truncate(time.Hour))
		if err != nil {
			return nil, err
		}
		for _, stat := range stats {
			items = append(items, HourlyBreakdownItem{
				Hour:        stat.Hour.Hour(),
				ProviderID:  stat.ProviderID,
				InputTokens: stat.InputTokens,
				OutputTokens: stat.OutputTokens,
				TotalTokens: stat.TotalTokens,
			})
		}

		// 当前小时实时数据
		currentHour := now.Hour()
		breakdown, err := s.requestLogRepo.GetCurrentHourBreakdown()
		if err != nil {
			slog.Warn("获取当前小时实时拆分数据失败", "error", err)
		} else {
			for _, b := range breakdown {
				items = append(items, HourlyBreakdownItem{
					Hour:        currentHour,
					ProviderID:  b.ProviderID,
					InputTokens: b.InputTokens,
					OutputTokens: b.OutputTokens,
					TotalTokens: b.TotalTokens,
				})
			}
		}
	} else {
		// 历史日期：直接从汇总表读取
		stats, err := s.hourlyStatRepo.GetHourlyStatsWithBreakdown(dayStart, dayEnd)
		if err != nil {
			return nil, err
		}
		for _, stat := range stats {
			items = append(items, HourlyBreakdownItem{
				Hour:        stat.Hour.Hour(),
				ProviderID:  stat.ProviderID,
				InputTokens: stat.InputTokens,
				OutputTokens: stat.OutputTokens,
				TotalTokens: stat.TotalTokens,
			})
		}
	}

	// 填充 Provider 名称
	for i := range items {
		if items[i].ProviderID > 0 && items[i].ProviderID != model.DeletedProviderID {
			if p, err := s.providerService.GetProvider(items[i].ProviderID); err == nil {
				items[i].ProviderName = p.Name
			}
		}
		if items[i].ProviderName == "" {
			items[i].ProviderName = "已删除"
		}
	}

	return items, nil
}

// currentHourStart 返回当前小时的起始时间
func currentHourStart() time.Time {
	return time.Now().Truncate(time.Hour)
}
