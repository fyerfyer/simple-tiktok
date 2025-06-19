package producer

import (
	"context"

	"go-backend/internal/domain"
)

// NoOpVideoEventProducer 空实现的视频事件生产者
type NoOpVideoEventProducer struct{}

// PublishVideoUploadedEvent 发布视频上传事件（空实现）
func (p *NoOpVideoEventProducer) PublishVideoUploadedEvent(ctx context.Context, event *domain.VideoUploadedEvent) error {
	return nil
}

// PublishVideoProcessedEvent 发布视频处理事件（空实现）
func (p *NoOpVideoEventProducer) PublishVideoProcessedEvent(ctx context.Context, event *domain.VideoProcessedEvent) error {
	return nil
}

// PublishVideoStatsUpdatedEvent 发布视频统计更新事件（空实现）
func (p *NoOpVideoEventProducer) PublishVideoStatsUpdatedEvent(ctx context.Context, event *domain.VideoStatsUpdatedEvent) error {
	return nil
}

// PublishVideoDeletedEvent 发布视频删除事件（空实现）
func (p *NoOpVideoEventProducer) PublishVideoDeletedEvent(ctx context.Context, event *domain.VideoDeletedEvent) error {
	return nil
}
