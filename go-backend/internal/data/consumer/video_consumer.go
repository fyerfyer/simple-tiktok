package consumer

import (
	"context"
	"encoding/json"
	"time"

	"go-backend/internal/conf"
	"go-backend/internal/domain"
	"go-backend/pkg/messaging"
	"go-backend/pkg/storage"

	"github.com/go-kratos/kratos/v2/log"
)

// VideoProcessConsumer 视频处理消费者
type VideoProcessConsumer struct {
	kafkaManager *messaging.KafkaManager
	storage      storage.VideoStorage
	config       *conf.Business_KafkaTopics
	log          *log.Helper
}

// NewVideoProcessConsumer 创建视频处理消费者
func NewVideoProcessConsumer(
	kafkaManager *messaging.KafkaManager,
	storage storage.VideoStorage,
	businessConfig *conf.Business,
	logger log.Logger,
) *VideoProcessConsumer {
	return &VideoProcessConsumer{
		kafkaManager: kafkaManager,
		storage:      storage,
		config:       businessConfig.KafkaTopics,
		log:          log.NewHelper(logger),
	}
}

// Start 启动消费者
func (c *VideoProcessConsumer) Start(ctx context.Context) error {
	consumer := c.kafkaManager.GetConsumer()

	// 订阅视频上传事件
	if err := consumer.Subscribe(c.config.VideoUpload, c.handleVideoUploadEvent); err != nil {
		return err
	}

	// 订阅视频处理事件
	if err := consumer.Subscribe(c.config.VideoProcess, c.handleVideoProcessEvent); err != nil {
		return err
	}

	return consumer.Start(ctx)
}

// Stop 停止消费者
func (c *VideoProcessConsumer) Stop() error {
	consumer := c.kafkaManager.GetConsumer()
	return consumer.Stop()
}

// handleVideoUploadEvent 处理视频上传事件
func (c *VideoProcessConsumer) handleVideoUploadEvent(ctx context.Context, message *messaging.BaseMessage) error {
	c.log.WithContext(ctx).Infof("received video upload event: %s", message.ID)

	var event domain.VideoUploadedEvent
	data, err := json.Marshal(message.Data)
	if err != nil {
		c.log.WithContext(ctx).Errorf("marshal video upload event failed: %v", err)
		return err
	}

	if err := json.Unmarshal(data, &event); err != nil {
		c.log.WithContext(ctx).Errorf("unmarshal video upload event failed: %v", err)
		return err
	}

	// 异步处理视频
	go func() {
		c.processVideo(context.Background(), &event)
	}()

	return nil
}

// handleVideoProcessEvent 处理视频处理事件
func (c *VideoProcessConsumer) handleVideoProcessEvent(ctx context.Context, message *messaging.BaseMessage) error {
	c.log.WithContext(ctx).Infof("received video process event: %s", message.ID)

	var event domain.VideoProcessedEvent
	data, err := json.Marshal(message.Data)
	if err != nil {
		c.log.WithContext(ctx).Errorf("marshal video process event failed: %v", err)
		return err
	}

	if err := json.Unmarshal(data, &event); err != nil {
		c.log.WithContext(ctx).Errorf("unmarshal video process event failed: %v", err)
		return err
	}

	return c.handleProcessResult(ctx, &event)
}

// processVideo 处理视频
func (c *VideoProcessConsumer) processVideo(ctx context.Context, event *domain.VideoUploadedEvent) {
	c.log.WithContext(ctx).Infof("start processing video: %d", event.VideoID)

	// 生成缩略图
	if err := c.generateThumbnail(ctx, event); err != nil {
		c.log.WithContext(ctx).Errorf("generate thumbnail failed: %v", err)
		c.publishProcessFailedEvent(ctx, event.VideoID, domain.ProcessTypeThumbnail, err.Error())
		return
	}

	// 视频转码（简化实现，实际项目中可能需要调用专门的转码服务）
	if err := c.transcodeVideo(ctx, event); err != nil {
		c.log.WithContext(ctx).Errorf("transcode video failed: %v", err)
		c.publishProcessFailedEvent(ctx, event.VideoID, domain.ProcessTypeTranscode, err.Error())
		return
	}

	// 发布处理成功事件
	c.publishProcessSuccessEvent(ctx, event.VideoID)
}

// generateThumbnail 生成缩略图
func (c *VideoProcessConsumer) generateThumbnail(ctx context.Context, event *domain.VideoUploadedEvent) error {
	// TODO: 简化实现：使用默认缩略图
	// 实际项目中应该从视频中截取帧作为缩略图

	c.log.WithContext(ctx).Infof("generating thumbnail for video: %d", event.VideoID)

	// 模拟缩略图生成时间
	return nil
}

// transcodeVideo 视频转码
func (c *VideoProcessConsumer) transcodeVideo(ctx context.Context, event *domain.VideoUploadedEvent) error {
	// TODO: 简化实现：跳过转码过程
	// 实际项目中应该调用FFmpeg或专门的转码服务

	c.log.WithContext(ctx).Infof("transcoding video: %d", event.VideoID)

	// 模拟转码时间
	time.Sleep(5 * time.Second)

	return nil
}

// handleProcessResult 处理处理结果
func (c *VideoProcessConsumer) handleProcessResult(ctx context.Context, event *domain.VideoProcessedEvent) error {
	c.log.WithContext(ctx).Infof("handling process result for video: %d, type: %s, status: %s",
		event.VideoID, event.ProcessType, event.Status)

	switch event.Status {
	case domain.ProcessStatusSuccess:
		c.log.WithContext(ctx).Infof("video processing completed successfully: %d", event.VideoID)
	case domain.ProcessStatusFailed:
		c.log.WithContext(ctx).Errorf("video processing failed: %d, error: %s", event.VideoID, event.ErrorMessage)
	}

	return nil
}

// publishProcessSuccessEvent 发布处理成功事件
func (c *VideoProcessConsumer) publishProcessSuccessEvent(ctx context.Context, videoID int64) {
	kafkaEvent := &messaging.VideoProcessEvent{
		VideoID:     videoID,
		ProcessType: "complete",
		Status:      "completed",
	}

	if err := c.kafkaManager.SendVideoProcessEvent(ctx, c.config.VideoProcess, kafkaEvent); err != nil {
		c.log.WithContext(ctx).Errorf("send video process success event failed: %v", err)
	}
}

// publishProcessFailedEvent 发布处理失败事件
func (c *VideoProcessConsumer) publishProcessFailedEvent(ctx context.Context, videoID int64, processType, errorMsg string) {
	kafkaEvent := &messaging.VideoProcessEvent{
		VideoID:     videoID,
		ProcessType: processType,
		Status:      "failed",
		Error:       errorMsg,
	}

	if err := c.kafkaManager.SendVideoProcessEvent(ctx, c.config.VideoProcess, kafkaEvent); err != nil {
		c.log.WithContext(ctx).Errorf("send video process failed event failed: %v", err)
	}
}
