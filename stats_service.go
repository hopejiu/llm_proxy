package main

import (
	"context"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"

	"github.com/wanglejiu/llm-proxy/internal/handler"
	"github.com/wanglejiu/llm-proxy/internal/model"
	"github.com/wanglejiu/llm-proxy/internal/repository"
	"github.com/wanglejiu/llm-proxy/internal/service"
	"log/slog"
)

// StatsService Wails 绑定服务：统计和日志
type StatsService struct {
	statsSvc       *service.StatsService
	logSvc         *service.LogService
	providerSvc    *service.ProviderService
	tracker        *handler.ActiveRequestTracker
	sessionRepo    *repository.ChatSessionRepository
	ctx            context.Context
}

func NewStatsService(statsSvc *service.StatsService, logSvc *service.LogService, providerSvc *service.ProviderService, tracker *handler.ActiveRequestTracker) *StatsService {
	return &StatsService{
		statsSvc:    statsSvc,
		logSvc:      logSvc,
		providerSvc: providerSvc,
		tracker:     tracker,
	}
}

// WithSessionRepo 设置会话仓库（启用会话统计查询）
func (s *StatsService) WithSessionRepo(repo *repository.ChatSessionRepository) {
	s.sessionRepo = repo
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

// GetHourlyModelStats 获取指定日期指定模型的分时统计
func (s *StatsService) GetHourlyModelStats(date string, providerID uint, modelName string) ([]model.HourlyStatsResult, error) {
	if date == "" {
		date = time.Now().Format("2006-01-02")
	}
	parsedDate, err := time.Parse("2006-01-02", date)
	if err != nil {
		return nil, NewAppError("BAD_REQUEST", "日期格式错误")
	}
	if modelName == "" {
		return nil, NewAppError("BAD_REQUEST", "模型名不能为空")
	}
	stats, err := s.statsSvc.GetHourlyModelStats(parsedDate, providerID, modelName)
	if err != nil {
		slog.Error("获取模型分时统计失败", "error", err)
		return nil, NewAppError("INTERNAL", "获取分时统计失败")
	}
	return stats, nil
}

// GetModelStats 获取模型级别统计（用于前端计算成本）
func (s *StatsService) GetModelStats(providerID uint) ([]ModelStatVO, error) {
	items, err := s.statsSvc.GetModelStats(providerID)
	if err != nil {
		slog.Error("获取模型统计失败", "error", err)
		return nil, NewAppError("INTERNAL", "获取模型统计失败")
	}

	providerNames := s.buildProviderNames(items)
	result := make([]ModelStatVO, len(items))
	for i, item := range items {
		name := providerNames[item.ProviderID]
		if name == "" {
			if item.ProviderID == model.DeletedProviderID {
				name = "已删除"
			} else {
				name = "未知"
			}
		}
		result[i] = ModelStatVO{
			Date:              item.Date,
			ProviderID:        item.ProviderID,
			ProviderName:      name,
			Model:             item.Model,
			TotalInputTokens:  item.TotalInputTokens,
			TotalOutputTokens: item.TotalOutputTokens,
			TotalTokens:       item.TotalTokens,
			TotalCachedTokens: item.TotalCachedTokens,
			RequestCount:      item.RequestCount,
		}
	}
	if len(result) > 0 {
		slog.Debug("[GetModelStats] 返回数据", "providerID", providerID, "count", len(result),
			"sample_date", result[0].Date, "sample_date_len", len(result[0].Date),
			"sample_model", result[0].Model, "sample_tokens", result[0].TotalTokens)
	}
	return result, nil
}

// GetHourlyStatsByDateWithBreakdown 获取按 provider+model 拆分的分时统计（用于堆叠图）
// providerID=0 时返回所有 provider 的数据，providerID>0 时只返回指定 provider 的数据
func (s *StatsService) GetHourlyStatsByDateWithBreakdown(date string, providerID uint) ([]HourlyStatBreakdownVO, error) {
	if date == "" {
		date = time.Now().Format("2006-01-02")
	}

	items, err := s.statsSvc.GetHourlyStatsByDateWithBreakdown(date, providerID)
	if err != nil {
		slog.Error("获取拆分统计失败", "error", err)
		return nil, NewAppError("INTERNAL", "获取拆分统计失败")
	}

	return items, nil
}

// ========== Logs ==========

// GetRecentLogs 获取最近请求日志，modelName 非空时按模型名过滤
func (s *StatsService) GetRecentLogs(limit int, modelName string) ([]RequestLogVO, error) {
	if limit <= 0 {
		limit = 20
	}
	logs, err := s.logSvc.GetRecentLogs(limit, modelName)
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
	logDetail, err := s.logSvc.GetLogDetail(id)
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

// ========== Sessions ==========

// GetSessions 获取所有会话列表（按创建时间倒序）
func (s *StatsService) GetSessions() ([]SessionVO, error) {
	if s.sessionRepo == nil {
		return nil, NewAppError("INTERNAL", "会话功能未启用")
	}
	sessions, err := s.sessionRepo.GetAll()
	if err != nil {
		slog.Error("获取会话列表失败", "error", err)
		return nil, NewAppError("INTERNAL", "获取会话列表失败")
	}

	result := make([]SessionVO, len(sessions))
	for i, sess := range sessions {
		result[i] = SessionVO{
			ID:           sess.ID,
			Models:       sess.Models,
			RequestCount: sess.RequestCount,
			TotalTokens:  sess.TotalTokens,
			TotalCost:    sess.TotalCost,
			CreatedAt:    sess.CreatedAt.Format("2006-01-02 15:04:05"),
			UpdatedAt:    sess.UpdatedAt.Format("2006-01-02 15:04:05"),
		}
	}
	return result, nil
}

// GetSessionRequests 获取指定会话的请求列表
func (s *StatsService) GetSessionRequests(sessionID uint) ([]RequestLogVO, error) {
	if s.sessionRepo == nil {
		return nil, NewAppError("INTERNAL", "会话功能未启用")
	}
	logs, err := s.logSvc.GetLogsBySession(sessionID)
	if err != nil {
		slog.Error("获取会话请求列表失败", "session_id", sessionID, "error", err)
		return nil, NewAppError("INTERNAL", "获取请求列表失败")
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
	return s.resolveProviderNames(providerIDs)
}

// buildProviderNames 从 ModelDailyStat 列表批量查询 Provider 名称
func (s *StatsService) buildProviderNames(stats []repository.ModelDailyStat) map[uint]string {
	providerIDs := make(map[uint]bool)
	for _, stat := range stats {
		if stat.ProviderID != model.DeletedProviderID {
			providerIDs[stat.ProviderID] = true
		}
	}
	return s.resolveProviderNames(providerIDs)
}

// resolveProviderNames 通用 ProviderID→Name 解析
func (s *StatsService) resolveProviderNames(ids map[uint]bool) map[uint]string {
	names := make(map[uint]string)
	for id := range ids {
		if p, err := s.providerSvc.GetProvider(id); err == nil {
			names[id] = p.Name
		}
	}
	return names
}
