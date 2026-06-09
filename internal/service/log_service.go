package service

import (
	"github.com/wanglejiu/llm-proxy/internal/model"
	"github.com/wanglejiu/llm-proxy/internal/repository"
)

type LogService struct {
	requestLogRepo *repository.RequestLogRepository
}

func NewLogService(requestLogRepo *repository.RequestLogRepository) *LogService {
	return &LogService{requestLogRepo: requestLogRepo}
}

// GetRecentLogs 获取最近的请求日志
func (s *LogService) GetRecentLogs(limit int, modelName string) ([]model.RequestLog, error) {
	return s.requestLogRepo.GetRecent(limit, modelName)
}

// GetLogDetail 获取单条请求日志详情
func (s *LogService) GetLogDetail(id uint) (*model.RequestLog, error) {
	return s.requestLogRepo.GetByID(id)
}

// GetLogsBySession 根据会话ID获取请求日志列表
func (s *LogService) GetLogsBySession(sessionID uint) ([]model.RequestLog, error) {
	return s.requestLogRepo.GetBySession(sessionID)
}
