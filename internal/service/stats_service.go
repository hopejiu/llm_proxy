package service

import (
	"log/slog"
	"llm-proxy/internal/model"
	"llm-proxy/internal/repository"
)

type StatsService struct {
	requestLogRepo *repository.RequestLogRepository
}

func NewStatsService(requestLogRepo *repository.RequestLogRepository) *StatsService {
	return &StatsService{requestLogRepo: requestLogRepo}
}

// GetDashboardStats 获取仪表盘统计数据
func (s *StatsService) GetDashboardStats() (map[string]*model.TokenStats, error) {
	result := make(map[string]*model.TokenStats)
	
	// 今日统计
	todayStats, err := s.requestLogRepo.GetTodayStats()
	if err != nil {
		slog.Error("获取今日统计失败", "error", err)
		return nil, err
	}
	result["today"] = todayStats
	
	// 本周统计
	weekStats, err := s.requestLogRepo.GetWeekStats()
	if err != nil {
		slog.Error("获取本周统计失败", "error", err)
		return nil, err
	}
	result["week"] = weekStats
	
	// 总计统计
	totalStats, err := s.requestLogRepo.GetTotalStats()
	if err != nil {
		slog.Error("获取总计统计失败", "error", err)
		return nil, err
	}
	result["total"] = totalStats
	
	return result, nil
}

// GetLast30DaysStats 获取最近30天统计
func (s *StatsService) GetLast30DaysStats() ([]model.TokenStats, error) {
	return s.requestLogRepo.GetLast30DaysStats()
}

// GetRecentLogs 获取最近的请求日志
func (s *StatsService) GetRecentLogs(limit int) ([]model.RequestLog, error) {
	return s.requestLogRepo.GetRecent(limit)
}

// GetLogDetail 获取单条请求日志详情
func (s *StatsService) GetLogDetail(id uint) (*model.RequestLog, error) {
	return s.requestLogRepo.GetByID(id)
}
