package producer

import (
	"context"
	"time"

	"go-backend/internal/conf"
	"go-backend/internal/domain"
	"go-backend/pkg/messaging"

	"github.com/go-kratos/kratos/v2/log"
)

// VideoEventProducer 视频事件生产者
type VideoEventProducer struct {
	kafkaManager *messaging.KafkaManager
	config       *conf.Business_KafkaTopics
	log          *log.Helper
}

// NewVideoEventProducer 创建视频事件生产者
func NewVideoEventProducer(
	kafkaManager *messaging.KafkaManager,
	businessConfig *conf.Business,
	logger log.Logger,
) domain.VideoEventPublisher {
	return &VideoEventProducer{
		kafkaManager: kafkaManager,
		config:       businessConfig.KafkaTopics,
		log:          log.NewHelper(logger),
	}
}

// PublishVideoUploadedEvent 发布视频上传事件
func (p *VideoEventProducer) PublishVideoUploadedEvent(ctx context.Context, event *domain.VideoUploadedEvent) error {
	kafkaEvent := &messaging.VideoUploadEvent{
		VideoID:    event.VideoID,
		AuthorID:   event.AuthorID,
		Title:      event.Title,
		PlayURL:    event.PlayURL,
		UploadTime: event.UploadedAt.Unix(),
	}

	if err := p.kafkaManager.SendVideoUploadEvent(ctx, p.config.VideoUpload, kafkaEvent); err != nil {
		p.log.WithContext(ctx).Errorf("send video upload event failed: %v", err)
		return err
	}

	p.log.WithContext(ctx).Infof("published video uploaded event: video_id=%d", event.VideoID)
	return nil
}

// PublishVideoProcessedEvent 发布视频处理事件
func (p *VideoEventProducer) PublishVideoProcessedEvent(ctx context.Context, event *domain.VideoProcessedEvent) error {
	kafkaEvent := &messaging.VideoProcessEvent{
		VideoID:     event.VideoID,
		ProcessType: event.ProcessType,
		Status:      event.Status,
		Result:      event.Result,
		Error:       event.ErrorMessage,
	}

	if err := p.kafkaManager.SendVideoProcessEvent(ctx, p.config.VideoProcess, kafkaEvent); err != nil {
		p.log.WithContext(ctx).Errorf("send video process event failed: %v", err)
		return err
	}

	p.log.WithContext(ctx).Infof("published video processed event: video_id=%d, type=%s, status=%s",
		event.VideoID, event.ProcessType, event.Status)
	return nil
}

// PublishVideoStatsUpdatedEvent 发布视频统计更新事件
func (p *VideoEventProducer) PublishVideoStatsUpdatedEvent(ctx context.Context, event *domain.VideoStatsUpdatedEvent) error {
	kafkaEvent := &messaging.VideoStatsEvent{
		VideoID:   event.VideoID,
		StatsType: event.StatsType,
		Count:     event.Delta,
		UserID:    event.UserID,
	}

	if err := p.kafkaManager.SendVideoStatsEvent(ctx, p.config.VideoStats, kafkaEvent); err != nil {
		p.log.WithContext(ctx).Errorf("send video stats event failed: %v", err)
		return err
	}

	p.log.WithContext(ctx).Infof("published video stats updated event: video_id=%d, type=%s, delta=%d",
		event.VideoID, event.StatsType, event.Delta)
	return nil
}

// PublishVideoDeletedEvent 发布视频删除事件
func (p *VideoEventProducer) PublishVideoDeletedEvent(ctx context.Context, event *domain.VideoDeletedEvent) error {
	// 视频删除事件可以用UserAction事件表示
	kafkaEvent := &messaging.UserActionEvent{
		UserID:     event.AuthorID,
		ActionType: "delete_video",
		TargetID:   event.VideoID,
		TargetType: "video",
		Timestamp:  event.DeletedAt.Unix(),
	}

	if err := p.kafkaManager.SendUserActionEvent(ctx, p.config.UserAction, kafkaEvent); err != nil {
		p.log.WithContext(ctx).Errorf("send video deleted event failed: %v", err)
		return err
	}

	p.log.WithContext(ctx).Infof("published video deleted event: video_id=%d, author_id=%d",
		event.VideoID, event.AuthorID)
	return nil
}

// PublishUserActionEvent 发布用户行为事件
func (p *VideoEventProducer) PublishUserActionEvent(ctx context.Context, userID int64, actionType string, targetID int64, targetType string) error {
	kafkaEvent := &messaging.UserActionEvent{
		UserID:     userID,
		ActionType: actionType,
		TargetID:   targetID,
		TargetType: targetType,
		Timestamp:  time.Now().Unix(),
	}

	if err := p.kafkaManager.SendUserActionEvent(ctx, p.config.UserAction, kafkaEvent); err != nil {
		p.log.WithContext(ctx).Errorf("send user action event failed: %v", err)
		return err
	}

	p.log.WithContext(ctx).Infof("published user action event: user_id=%d, action=%s, target_id=%d",
		userID, actionType, targetID)
	return nil
}

// publishBatchEvents 批量发布事件（优化性能）
func (p *VideoEventProducer) publishBatchEvents(ctx context.Context, events []interface{}) error {
	// 将事件按类型分组
	uploadEvents := make([]*messaging.VideoUploadEvent, 0)
	processEvents := make([]*messaging.VideoProcessEvent, 0)
	statsEvents := make([]*messaging.VideoStatsEvent, 0)
	actionEvents := make([]*messaging.UserActionEvent, 0)

	for _, event := range events {
		switch e := event.(type) {
		case *messaging.VideoUploadEvent:
			uploadEvents = append(uploadEvents, e)
		case *messaging.VideoProcessEvent:
			processEvents = append(processEvents, e)
		case *messaging.VideoStatsEvent:
			statsEvents = append(statsEvents, e)
		case *messaging.UserActionEvent:
			actionEvents = append(actionEvents, e)
		}
	}

	// 批量发送各类型事件
	if len(uploadEvents) > 0 {
		for _, event := range uploadEvents {
			if err := p.kafkaManager.SendVideoUploadEvent(ctx, p.config.VideoUpload, event); err != nil {
				p.log.WithContext(ctx).Errorf("batch send video upload event failed: %v", err)
			}
		}
	}

	if len(processEvents) > 0 {
		for _, event := range processEvents {
			if err := p.kafkaManager.SendVideoProcessEvent(ctx, p.config.VideoProcess, event); err != nil {
				p.log.WithContext(ctx).Errorf("batch send video process event failed: %v", err)
			}
		}
	}

	if len(statsEvents) > 0 {
		for _, event := range statsEvents {
			if err := p.kafkaManager.SendVideoStatsEvent(ctx, p.config.VideoStats, event); err != nil {
				p.log.WithContext(ctx).Errorf("batch send video stats event failed: %v", err)
			}
		}
	}

	if len(actionEvents) > 0 {
		for _, event := range actionEvents {
			if err := p.kafkaManager.SendUserActionEvent(ctx, p.config.UserAction, event); err != nil {
				p.log.WithContext(ctx).Errorf("batch send user action event failed: %v", err)
			}
		}
	}

	p.log.WithContext(ctx).Infof("batch published events: upload=%d, process=%d, stats=%d, action=%d",
		len(uploadEvents), len(processEvents), len(statsEvents), len(actionEvents))

	return nil
}
