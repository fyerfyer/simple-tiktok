package consumer

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"go-backend/internal/conf"
	"go-backend/internal/domain"
	"go-backend/pkg/media"
	"go-backend/pkg/messaging"
	"go-backend/pkg/storage"

	"github.com/go-kratos/kratos/v2/log"
)

// VideoProcessConsumer 视频处理消费者
type VideoProcessConsumer struct {
	kafkaManager *messaging.KafkaManager
	storage      storage.VideoStorage
	processor    media.VideoProcessorInterface
	thumbnail    *media.ThumbnailGenerator
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
	// 创建FFmpeg处理器
	processor := media.NewFFmpegProcessor("/tmp")

	// 创建缩略图生成器
	thumbnail := media.NewThumbnailGenerator(480, 270, 80, processor)

	return &VideoProcessConsumer{
		kafkaManager: kafkaManager,
		storage:      storage,
		processor:    processor,
		thumbnail:    thumbnail,
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

	// 视频转码
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
	c.log.WithContext(ctx).Infof("generating thumbnail for video: %d", event.VideoID)

	// 1. 从存储下载视频文件
	objectName := c.extractObjectName(event.PlayURL)
	videoReader, err := c.storage.Download(ctx, objectName)
	if err != nil {
		return fmt.Errorf("download video failed: %w", err)
	}
	defer videoReader.Close()

	// 2. 使用缩略图生成器生成缩略图
	thumbnailReader, err := c.thumbnail.GenerateFromVideo(ctx, videoReader, 5)
	if err != nil {
		c.log.WithContext(ctx).Warnf("generate thumbnail from video failed, using default: %v", err)
		// 如果从视频生成失败，使用默认缩略图
		thumbnailReader, err = c.thumbnail.GenerateDefault(ctx)
		if err != nil {
			return fmt.Errorf("generate default thumbnail failed: %w", err)
		}
	}

	// 3. 读取缩略图数据
	thumbnailData, err := io.ReadAll(thumbnailReader)
	if err != nil {
		return fmt.Errorf("read thumbnail data failed: %w", err)
	}

	// 4. 上传缩略图到存储
	coverFilename := fmt.Sprintf("cover_%d.jpg", event.VideoID)
	coverURL, err := c.storage.UploadCover(ctx, coverFilename, bytes.NewReader(thumbnailData), int64(len(thumbnailData)))
	if err != nil {
		return fmt.Errorf("upload thumbnail failed: %w", err)
	}

	// 5. 更新视频封面URL（这里可以发送事件或直接调用repo）
	c.log.WithContext(ctx).Infof("thumbnail generated successfully: video_id=%d, cover_url=%s", event.VideoID, coverURL)

	return nil
}

// transcodeVideo 视频转码
func (c *VideoProcessConsumer) transcodeVideo(ctx context.Context, event *domain.VideoUploadedEvent) error {
	c.log.WithContext(ctx).Infof("transcoding video: %d", event.VideoID)

	// 1. 从存储下载原始视频
	objectName := c.extractObjectName(event.PlayURL)
	videoReader, err := c.storage.Download(ctx, objectName)
	if err != nil {
		return fmt.Errorf("download video failed: %w", err)
	}
	defer videoReader.Close()

	// 2. 使用处理器进行转码
	var transcodedBuffer bytes.Buffer
	opts := &media.ProcessorOptions{
		Width:   1280,
		Height:  720,
		Format:  "mp4",
		Quality: 80,
	}

	err = c.processor.TranscodeVideo(ctx, videoReader, &transcodedBuffer, opts)
	if err != nil {
		return fmt.Errorf("transcode video failed: %w", err)
	}

	// 3. 上传转码后的视频
	transcodedFilename := fmt.Sprintf("transcoded_%d.mp4", event.VideoID)
	transcodedURL, err := c.storage.UploadVideo(ctx, transcodedFilename, &transcodedBuffer, int64(transcodedBuffer.Len()))
	if err != nil {
		return fmt.Errorf("upload transcoded video failed: %w", err)
	}

	c.log.WithContext(ctx).Infof("video transcoded successfully: video_id=%d, transcoded_url=%s", event.VideoID, transcodedURL)

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

// extractObjectName 从URL提取对象名
func (c *VideoProcessConsumer) extractObjectName(url string) string {
	parts := strings.Split(url, "/")
	if len(parts) >= 2 {
		return strings.Join(parts[len(parts)-2:], "/")
	}
	return url
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

// validateVideoMetadata 验证视频元数据（可选）
func (c *VideoProcessConsumer) validateVideoMetadata(ctx context.Context, videoReader io.Reader) (*media.VideoMetadata, error) {
	metadata, err := c.processor.GetVideoInfo(ctx, videoReader)
	if err != nil {
		return nil, fmt.Errorf("get video metadata failed: %w", err)
	}

	c.log.WithContext(ctx).Debugf("video metadata: duration=%.2f, size=%dx%d, format=%s",
		metadata.Duration, metadata.Width, metadata.Height, metadata.Format)

	return metadata, nil
}
