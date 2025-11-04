// Package usecase internal/app/usecase/scheduler_push.go
package usecase

import (
	"GWatch/internal/domain/scheduled_push"
	"GWatch/internal/entity"
	"fmt"
	"log"
	"time"
)

// ScheduledPushSchedulerImpl 全局定时推送调度器实现
type ScheduledPushSchedulerImpl struct {
	scheduledPushUseCase scheduled_push.ScheduledPushUseCase
	config               *entity.Config
	ticker               *time.Ticker
	stopCh               chan struct{}
	lastReported         map[string]time.Time // 记录每个时间点最后报告的时间
}

// NewScheduledPushScheduler 创建全局定时推送调度器
func NewScheduledPushScheduler(scheduledPushUseCase scheduled_push.ScheduledPushUseCase) scheduled_push.ScheduledPushScheduler {
	return &ScheduledPushSchedulerImpl{
		scheduledPushUseCase: scheduledPushUseCase,
		stopCh:               make(chan struct{}),
		lastReported:         make(map[string]time.Time),
	}
}

// Start 启动全局定时推送调度
func (sps *ScheduledPushSchedulerImpl) Start(config *entity.Config, stopCh <-chan struct{}) error {
	sps.config = config

	// 每10秒检查一次是否到了推送时间，提高响应速度
	sps.ticker = time.NewTicker(10 * time.Second)

	go func() {
		defer sps.ticker.Stop()

		// 启动时立即检查一次，避免错过推送时间
		log.Println("启动时检查全局定时推送时间...")
		sps.executeScheduledPushIfNeeded(config, "启动时匹配到推送时间，立即执行全局监控报告")

		for {
			select {
			case <-sps.ticker.C:
				// 检查是否到了推送时间
				sps.executeScheduledPushIfNeeded(config, "定时器触发：开始执行全局监控报告")
			case <-stopCh:
				log.Println("全局定时推送调度器收到停止信号")
				return
			case <-sps.stopCh:
				log.Println("全局定时推送调度器停止")
				return
			}
		}
	}()

	return nil
}

// executeScheduledPushIfNeeded 如果需要则执行全局定时推送
func (sps *ScheduledPushSchedulerImpl) executeScheduledPushIfNeeded(config *entity.Config, logPrefix string) {
	if config.ScheduledPush == nil || !config.ScheduledPush.Enabled {
		return
	}

	if sps.IsTimeToPush(config.ScheduledPush.PushTimes) {
		mode := config.ScheduledPush.Mode
		if mode == "" {
			mode = "client" // 默认是 client 模式
		}

		// Server模式：延迟执行聚合，等待所有Client上传完数据
		if mode == "server" {
			delaySeconds := config.ScheduledPush.ServerAggregationDelaySeconds
			if delaySeconds <= 0 {
				delaySeconds = 60 // 默认延迟60秒
			}
			
			log.Printf("%s (Server模式，将延迟%d秒后聚合)", logPrefix, delaySeconds)
			
			// 异步延迟执行，避免阻塞调度器
			go func() {
				time.Sleep(time.Duration(delaySeconds) * time.Second)
				log.Printf("[Server模式] 延迟等待完成，开始聚合数据")
				if err := sps.scheduledPushUseCase.RunScheduledPush(config); err != nil {
					log.Printf("执行全局定时推送失败: %v", err)
				} else {
					log.Println("全局定时推送发送成功")
				}
			}()
		} else {
			// Client模式：立即执行
		log.Printf("%s", logPrefix)
		if err := sps.scheduledPushUseCase.RunScheduledPush(config); err != nil {
			log.Printf("执行全局定时推送失败: %v", err)
		} else {
			log.Println("全局定时推送发送成功")
			}
		}
	}
}

// Stop 停止全局定时推送调度
func (sps *ScheduledPushSchedulerImpl) Stop() error {
	close(sps.stopCh)
	return nil
}

// IsTimeToPush 检查是否到了推送时间
func (sps *ScheduledPushSchedulerImpl) IsTimeToPush(pushTimes []string) bool {
	now := time.Now()
	currentTime := fmt.Sprintf("%d:%02d", now.Hour(), now.Minute())

	log.Printf("检查全局推送时间: 当前时间=%s, 配置时间=%v", currentTime, pushTimes)

	for _, pushTime := range pushTimes {
		if currentTime == pushTime {
			// 检查是否已经在这个时间点推送过
			if lastReported, exists := sps.lastReported[pushTime]; exists {
				// 如果上次推送时间与当前时间在同一分钟内，则不重复推送
				if now.Truncate(time.Minute).Equal(lastReported.Truncate(time.Minute)) {
					log.Printf("时间点 %s 已在本分钟内推送过，跳过", pushTime)
					continue
				}
			}

			log.Printf("匹配到推送时间: %s", pushTime)
			// 记录推送时间
			sps.lastReported[pushTime] = now
			return true
		}
	}

	return false
}
