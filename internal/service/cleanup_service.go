package service

import (
	"context"
	"llm-proxy/internal/repository"
	"log/slog"
	"time"
)

// CleanupService 定时汇总和清理服务
type CleanupService struct {
	hourlyStatRepo *repository.HourlyStatRepository
	requestLogRepo *repository.RequestLogRepository
	cleanupDays    int
}

func NewCleanupService(hourlyStatRepo *repository.HourlyStatRepository, requestLogRepo *repository.RequestLogRepository, cleanupDays int) *CleanupService {
	return &CleanupService{
		hourlyStatRepo: hourlyStatRepo,
		requestLogRepo: requestLogRepo,
		cleanupDays:    cleanupDays,
	}
}

// AggregateLastHour 汇总上一个小时的明细数据
func (s *CleanupService) AggregateLastHour() error {
	hourStart := time.Now().Truncate(time.Hour).Add(-time.Hour)
	return s.aggregateHour(hourStart)
}

// aggregateHour 汇总指定小时的明细数据，成功后标记已汇总
func (s *CleanupService) aggregateHour(hourStart time.Time) error {
	stat, err := s.requestLogRepo.AggregateHour(hourStart)
	if err != nil {
		slog.Error("汇总小时数据失败", "hour", hourStart, "error", err)
		return err
	}

	// 没有未汇总的数据则跳过
	if stat.RequestCount == 0 {
		return nil
	}

	if err := s.hourlyStatRepo.Upsert(stat); err != nil {
		slog.Error("写入汇总数据失败", "hour", hourStart, "error", err)
		return err
	}

	// 标记明细记录为已汇总
	if err := s.requestLogRepo.MarkAggregated(hourStart); err != nil {
		slog.Warn("标记已汇总失败", "hour", hourStart, "error", err)
	}

	slog.Info("汇总小时数据完成", "hour", hourStart, "requestCount", stat.RequestCount)
	return nil
}

// BackfillMissingHours 回填缺失的历史汇总数据
func (s *CleanupService) BackfillMissingHours() error {
	// 获取明细表最早记录时间
	minTime, err := s.requestLogRepo.GetMinCreatedAt()
	if err != nil {
		slog.Error("获取最早记录时间失败", "error", err)
		return err
	}
	if minTime == nil {
		// 空表，无需回填
		return nil
	}

	// 回填范围：最早记录所在小时 ~ 当前小时前1小时
	startHour := minTime.Truncate(time.Hour)
	endHour := time.Now().Truncate(time.Hour)

	// 获取缺失的小时
	missing, err := s.hourlyStatRepo.GetMissingHours(startHour, endHour)
	if err != nil {
		slog.Error("获取缺失小时失败", "error", err)
		return err
	}

	if len(missing) == 0 {
		return nil
	}

	slog.Info("开始回填历史汇总数据", "missingCount", len(missing), "from", missing[0], "to", missing[len(missing)-1])

	for _, hourStart := range missing {
		if err := s.aggregateHour(hourStart); err != nil {
			slog.Warn("回填小时数据失败，继续下一个", "hour", hourStart, "error", err)
		}
	}

	slog.Info("历史汇总数据回填完成", "totalHours", len(missing))
	return nil
}

// DeleteOldRequestLogs 删除已汇总且超过保留天数的明细记录
func (s *CleanupService) DeleteOldRequestLogs() error {
	rowsAffected, err := s.requestLogRepo.DeleteOldRequestLogs(s.cleanupDays)
	if err != nil {
		slog.Error("删除旧明细记录失败", "error", err)
		return err
	}
	if rowsAffected > 0 {
		slog.Info("已删除旧明细记录", "rowsAffected", rowsAffected, "days", s.cleanupDays)
	}
	return nil
}

// Start 启动定时汇总和清理（后台 goroutine）
func (s *CleanupService) Start(ctx context.Context) {
	// 每小时汇总
	hourlyTicker := time.NewTicker(time.Hour)
	defer hourlyTicker.Stop()

	// 计算到下一个凌晨3点的时长
	dailyTimer := s.scheduleNextDaily()
	defer dailyTimer.Stop()

	// 启动后立即汇总上一个小时
	if err := s.AggregateLastHour(); err != nil {
		slog.Warn("启动时汇总上一小时失败", "error", err)
	}

	for {
		select {
		case <-ctx.Done():
			slog.Info("定时汇总和清理服务已停止")
			return
		case <-hourlyTicker.C:
			if err := s.AggregateLastHour(); err != nil {
				slog.Warn("定时汇总失败", "error", err)
			}
		case <-dailyTimer.C:
			// 先汇总确保数据完整，再清理
			if err := s.AggregateLastHour(); err != nil {
				slog.Warn("清理前汇总失败", "error", err)
			}
			if err := s.DeleteOldRequestLogs(); err != nil {
				slog.Warn("定时清理失败", "error", err)
			}
			// 重置为下一个凌晨3点
			dailyTimer.Reset(s.scheduleNextDailyDuration())
		}
	}
}

// scheduleNextDaily 创建一个在下一个凌晨3点触发的定时器
func (s *CleanupService) scheduleNextDaily() *time.Timer {
	return time.NewTimer(s.scheduleNextDailyDuration())
}

// scheduleNextDailyDuration 计算到下一个凌晨3点的时长
func (s *CleanupService) scheduleNextDailyDuration() time.Duration {
	now := time.Now()
	next := time.Date(now.Year(), now.Month(), now.Day(), 3, 0, 0, 0, now.Location())
	if !next.After(now) {
		next = next.Add(24 * time.Hour)
	}
	return next.Sub(now)
}
