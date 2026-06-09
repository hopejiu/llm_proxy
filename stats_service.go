package main

import (
	"context"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"

	"github.com/wanglejiu/llm-proxy/internal/handler"
	"github.com/wanglejiu/llm-proxy/internal/model"
	"github.com/wanglejiu/llm-proxy/internal/service"
	"log/slog"
)

// StatsService Wails 绑定服务：统计和日志
type StatsService struct {
	statsSvc       *service.StatsService
	providerSvc    *service.ProviderService
	tracker        *handler.ActiveRequestTracker
	ctx            context.Context
}

func NewStatsService(statsSvc *service.StatsService, providerSvc *service.ProviderService, tracker *handler.ActiveRequestTracker) *StatsService {
	return &StatsService{
		statsSvc:    statsSvc,
		providerSvc: providerSvc,
		tracker:     tracker,
	}
}

func (s *StatsService) ServiceStartup(ctx context.Context, _ application.ServiceOptions) error {
	s.ctx = ctx
	return nil
}

// GetStats 获取仪表盘统计
func (s *StatsService) GetStats(providerID uint) (map[string]*model.TokenStats, error) {
	stats, err := s.statsSvc.GetDashboardStats(providerID)
	if err != nil {
		slog.Error("获取仪表盘统计失败", "error", err)
		return nil, NewAppError("INTERNAL", "获取统计数据失败")
	}
	return stats, nil
}

// GetDailyStats 获取 30 天每日统计
func (s *StatsService) GetDailyStats(providerID uint) ([]model.TokenStats, error) {
	stats, err := s.statsSvc.GetLast30DaysStats(providerID)
	if err != nil {
		slog.Error("获取30天统计失败", "error", err)
		return nil, NewAppError("INTERNAL", "获取每日统计失败")
	}
	return stats, nil
}

// GetHourlyStatsByDate 获取分时统计
func (s *StatsService) GetHourlyStatsByDate(date string, providerID uint) ([]model.HourlyStatsResult, error) {
	if date == "" {
		stats, err := s.statsSvc.GetTodayHourlyStats(providerID)
		if err != nil {
			return nil, NewAppError("INTERNAL", "获取分时统计失败")
		}
		return stats, nil
	}

	parsedDate, err := time.Parse("2006-01-02", date)
	if err != nil {
		return nil, NewAppError("BAD_REQUEST", "日期格式错误，应为 YYYY-MM-DD")
	}

	stats, err := s.statsSvc.GetHourlyStatsByDate(parsedDate, providerID)
	if err != nil {
		return nil, NewAppError("INTERNAL", "获取分时统计失败")
	}
	return stats, nil
}

// GetHourlyStatsByDateWithBreakdown 获取按 provider 拆分的分时统计（用于堆叠图）
func (s *StatsService) GetHourlyStatsByDateWithBreakdown(date string) ([]HourlyStatBreakdownVO, error) {
	if date == "" {
		date = time.Now().Format("2006-01-02")
	}

	items, err := s.statsSvc.GetHourlyStatsByDateWithBreakdown(date)
	if err != nil {
		slog.Error("获取拆分统计失败", "error", err)
		return nil, NewAppError("INTERNAL", "获取拆分统计失败")
	}

	result := make([]HourlyStatBreakdownVO, len(items))
	for i, item := range items {
		result[i] = HourlyStatBreakdownVO{
			Hour:         item.Hour,
			ProviderID:   item.ProviderID,
			ProviderName: item.ProviderName,
			InputTokens:  item.InputTokens,
			OutputTokens: item.OutputTokens,
			TotalTokens:  item.TotalTokens,
		}
	}
	return result, nil
}

// ========== Logs ==========

// GetRecentLogs 获取最近请求日志
func (s *StatsService) GetRecentLogs(limit int) ([]RequestLogVO, error) {
	if limit <= 0 {
		limit = 20
	}
	logs, err := s.statsSvc.GetRecentLogs(limit)
	if err != nil {
		slog.Error("获取最近日志失败", "limit", limit, "error", err)
		return nil, NewAppError("INTERNAL", "获取日志列表失败")
	}

	providerNames := s.buildProviderNameMap(logs)

	result := make([]RequestLogVO, len(logs))
	for i, log := range logs {
		vo := requestLogToVO(&log)
		vo.ProviderName = providerNames[log.ProviderID]
		result[i] = vo
	}
	return result, nil
}

// GetLogDetail 获取单条请求日志详情
func (s *StatsService) GetLogDetail(id uint) (RequestLogDetailVO, error) {
	logDetail, err := s.statsSvc.GetLogDetail(id)
	if err != nil {
		slog.Error("获取日志详情失败", "id", id, "error", err)
		return RequestLogDetailVO{}, NewAppError("NOT_FOUND", "日志不存在")
	}

	vo := requestLogToDetailVO(logDetail)
	if logDetail.ProviderID != model.DeletedProviderID {
		if p, err := s.providerSvc.GetProvider(logDetail.ProviderID); err == nil {
			vo.ProviderName = p.Name
		}
	}
	return vo, nil
}

// ========== Active Requests ==========

// GetActiveRequests 获取当前活跃请求
func (s *StatsService) GetActiveRequests() []ActiveRequestVO {
	return s.tracker.GetAll()
}

// GetActiveRequest 按 ID 获取单个活跃请求
func (s *StatsService) GetActiveRequest(requestID string) (ActiveRequestVO, error) {
	reqs := s.tracker.GetAll()
	for _, req := range reqs {
		if req.RequestID == requestID {
			return req, nil
		}
	}
	return ActiveRequestVO{}, NewAppError("NOT_FOUND", "请求已完成")
}

// buildProviderNameMap 批量查询 Provider 名称，避免 N+1
func (s *StatsService) buildProviderNameMap(logs []model.RequestLog) map[uint]string {
	providerIDs := make(map[uint]bool)
	for _, log := range logs {
		if log.ProviderID != model.DeletedProviderID {
			providerIDs[log.ProviderID] = true
		}
	}

	names := make(map[uint]string)
	for id := range providerIDs {
		if p, err := s.providerSvc.GetProvider(id); err == nil {
			names[id] = p.Name
		}
	}
	return names
}
