package consumer

import (
	"context"
	"encoding/json"
	"time"

	"go-backend/internal/biz"
	"go-backend/internal/conf"
	"go-backend/pkg/messaging"

	"github.com/go-kratos/kratos/v2/log"
)

// StatsUpdateConsumer 统计更新消费者
type StatsUpdateConsumer struct {
	kafkaManager *messaging.KafkaManager
	videoUsecase *biz.VideoUsecase
	config       *conf.Business_KafkaTopics
	log          *log.Helper
}

// NewStatsUpdateConsumer 创建统计更新消费者
func NewStatsUpdateConsumer(
	kafkaManager *messaging.KafkaManager,
	videoUsecase *biz.VideoUsecase,
	businessConfig *conf.Business,
	logger log.Logger,
) *StatsUpdateConsumer {
	return &StatsUpdateConsumer{
		kafkaManager: kafkaManager,
		videoUsecase: videoUsecase,
		config:       businessConfig.KafkaTopics,
		log:          log.NewHelper(logger),
	}
}

// Start 启动消费者
func (c *StatsUpdateConsumer) Start(ctx context.Context) error {
	consumer := c.kafkaManager.GetConsumer()

	// 订阅视频统计事件
	if err := consumer.Subscribe(c.config.VideoStats, c.handleVideoStatsEvent); err != nil {
		return err
	}

	// 订阅用户行为事件
	if err := consumer.Subscribe(c.config.UserAction, c.handleUserActionEvent); err != nil {
		return err
	}

	return consumer.Start(ctx)
}

// Stop 停止消费者
func (c *StatsUpdateConsumer) Stop() error {
	consumer := c.kafkaManager.GetConsumer()
	return consumer.Stop()
}

// handleVideoStatsEvent 处理视频统计事件
func (c *StatsUpdateConsumer) handleVideoStatsEvent(ctx context.Context, message *messaging.BaseMessage) error {
	c.log.WithContext(ctx).Infof("received video stats event: %s", message.ID)

	var event messaging.VideoStatsEvent
	data, err := json.Marshal(message.Data)
	if err != nil {
		c.log.WithContext(ctx).Errorf("marshal video stats event failed: %v", err)
		return err
	}

	if err := json.Unmarshal(data, &event); err != nil {
		c.log.WithContext(ctx).Errorf("unmarshal video stats event failed: %v", err)
		return err
	}

	return c.updateVideoStats(ctx, &event)
}

// handleUserActionEvent 处理用户行为事件
func (c *StatsUpdateConsumer) handleUserActionEvent(ctx context.Context, message *messaging.BaseMessage) error {
	c.log.WithContext(ctx).Infof("received user action event: %s", message.ID)

	var event messaging.UserActionEvent
	data, err := json.Marshal(message.Data)
	if err != nil {
		c.log.WithContext(ctx).Errorf("marshal user action event failed: %v", err)
		return err
	}

	if err := json.Unmarshal(data, &event); err != nil {
		c.log.WithContext(ctx).Errorf("unmarshal user action event failed: %v", err)
		return err
	}

	return c.handleUserAction(ctx, &event)
}

// updateVideoStats 更新视频统计
func (c *StatsUpdateConsumer) updateVideoStats(ctx context.Context, event *messaging.VideoStatsEvent) error {
	// 根据统计类型映射到具体字段
	var statsType string
	var delta int64 = 1

	switch event.StatsType {
	case "play":
		statsType = "play"
		delta = event.Count
	case "like":
		statsType = "favorite"
	case "comment":
		statsType = "comment"
	case "share":
		statsType = "share"
	default:
		c.log.WithContext(ctx).Warnf("unknown stats type: %s", event.StatsType)
		return nil
	}

	// 批量更新统计（避免频繁数据库操作）
	if err := c.videoUsecase.UpdateVideoStats(ctx, event.VideoID, statsType, delta); err != nil {
		c.log.WithContext(ctx).Errorf("update video stats failed: %v", err)
		return err
	}

	c.log.WithContext(ctx).Infof("updated video stats: video_id=%d, type=%s, delta=%d",
		event.VideoID, statsType, delta)

	return nil
}

// handleUserAction 处理用户行为
func (c *StatsUpdateConsumer) handleUserAction(ctx context.Context, event *messaging.UserActionEvent) error {
	// 只处理视频相关的用户行为
	if event.TargetType != "video" {
		return nil
	}

	var delta int64
	var statsType string

	switch event.ActionType {
	case "like":
		statsType = "favorite"
		delta = 1
	case "unlike":
		statsType = "favorite"
		delta = -1
	case "comment":
		statsType = "comment"
		delta = 1
	case "uncomment":
		statsType = "comment"
		delta = -1
	case "play":
		statsType = "play"
		delta = 1
	default:
		// 不支持的行为类型，直接返回
		return nil
	}

	// 更新视频统计
	if err := c.videoUsecase.UpdateVideoStats(ctx, event.TargetID, statsType, delta); err != nil {
		c.log.WithContext(ctx).Errorf("update video stats from user action failed: %v", err)
		return err
	}

	c.log.WithContext(ctx).Infof("updated video stats from user action: video_id=%d, action=%s, user_id=%d",
		event.TargetID, event.ActionType, event.UserID)

	return nil
}

// batchUpdateStats 批量更新统计（优化性能）
func (c *StatsUpdateConsumer) batchUpdateStats(ctx context.Context, updates []StatsUpdate) error {
	// 将同一视频的多个统计更新合并
	mergedUpdates := make(map[int64]map[string]int64)

	for _, update := range updates {
		if mergedUpdates[update.VideoID] == nil {
			mergedUpdates[update.VideoID] = make(map[string]int64)
		}
		mergedUpdates[update.VideoID][update.StatsType] += update.Delta
	}

	// 批量执行更新
	for videoID, stats := range mergedUpdates {
		for statsType, delta := range stats {
			if err := c.videoUsecase.UpdateVideoStats(ctx, videoID, statsType, delta); err != nil {
				c.log.WithContext(ctx).Errorf("batch update video stats failed: video_id=%d, type=%s, error=%v",
					videoID, statsType, err)
				continue
			}
		}
	}

	return nil
}

// StatsUpdate 统计更新结构
type StatsUpdate struct {
	VideoID   int64
	StatsType string
	Delta     int64
}

// scheduleStatsFlush 定时刷新统计（防止数据丢失）
func (c *StatsUpdateConsumer) scheduleStatsFlush(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			c.flushPendingStats(ctx)
		}
	}
}

// flushPendingStats 刷新待处理的统计
func (c *StatsUpdateConsumer) flushPendingStats(ctx context.Context) {
	// TODO: 实现待处理统计的刷新逻辑
	c.log.WithContext(ctx).Debug("flushing pending stats")
}
