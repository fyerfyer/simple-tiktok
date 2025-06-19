package biz

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"go-backend/internal/conf"
	"go-backend/internal/domain"
	"go-backend/pkg/media"
	"go-backend/pkg/messaging"
	"go-backend/pkg/security"
	"go-backend/pkg/storage"
	"go-backend/pkg/utils"

	"github.com/go-kratos/kratos/v2/log"
)

// VideoRepo 视频仓储接口
type VideoRepo interface {
	CreateVideo(ctx context.Context, video *domain.Video) error
	GetVideo(ctx context.Context, videoID int64) (*domain.Video, error)
	GetVideos(ctx context.Context, videoIDs []int64) ([]*domain.Video, error)
	GetUserVideos(ctx context.Context, userID int64, limit int) ([]*domain.Video, error)
	GetFeedVideos(ctx context.Context, latestTime time.Time, limit int) ([]*domain.Video, error)
	UpdateVideoStats(ctx context.Context, videoID int64, field string, delta int64) error
	UpdateVideo(ctx context.Context, video *domain.Video) error
	UpdateVideoCover(ctx context.Context, videoID int64, coverURL string) error
	UpdateVideoPlayURL(ctx context.Context, videoID int64, playURL string) error
}

// VideoCacheRepo 视频缓存接口
type VideoCacheRepo interface {
	GetVideo(ctx context.Context, videoID int64) (*domain.Video, bool)
	SetVideo(ctx context.Context, video *domain.Video)
	DeleteVideo(ctx context.Context, videoID int64)
	GetUserVideos(ctx context.Context, userID int64) ([]*domain.Video, bool)
	SetUserVideos(ctx context.Context, userID int64, videos []*domain.Video)
	DeleteUserVideos(ctx context.Context, userID int64)
	GetFeedVideos(ctx context.Context, lastTime int64) ([]*domain.Video, bool)
	SetFeedVideos(ctx context.Context, lastTime int64, videos []*domain.Video)
	DeleteFeedCache(ctx context.Context)
	GetVideoStats(ctx context.Context, videoID int64) (map[string]int64, bool)
	SetVideoStats(ctx context.Context, videoID int64, stats map[string]int64)
	IncrVideoStats(ctx context.Context, videoID int64, field string, delta int64)
}

// VideoUsecase 视频用例
type VideoUsecase struct {
	repo           VideoRepo
	cache          VideoCacheRepo
	storage        storage.VideoStorage
	processor      *media.VideoProcessor
	kafkaManager   *messaging.KafkaManager
	validator      *security.Validator
	businessConfig *conf.Business
	log            *log.Helper
}

// NewVideoUseCase 创建视频用例
func NewVideoUseCase(
	repo VideoRepo,
	cache VideoCacheRepo,
	storage storage.VideoStorage,
	kafkaManager *messaging.KafkaManager,
	businessConfig *conf.Business,
	logger log.Logger,
) *VideoUsecase {
	processor := media.NewVideoProcessor(
		businessConfig.Video.MaxFileSize,
		businessConfig.Video.SupportedFormats,
		int(businessConfig.Video.CoverWidth),
		int(businessConfig.Video.CoverHeight),
		int(businessConfig.Video.CoverQuality),
	)

	return &VideoUsecase{
		repo:           repo,
		cache:          cache,
		storage:        storage,
		processor:      processor,
		kafkaManager:   kafkaManager,
		validator:      security.NewValidator(),
		businessConfig: businessConfig,
		log:            log.NewHelper(logger),
	}
}

// PublishVideo 发布视频
func (uc *VideoUsecase) PublishVideo(ctx context.Context, authorID int64, title string, videoData []byte, filename string) (*domain.Video, error) {
	// 验证标题
	if err := uc.validator.ValidateVideoTitle(title); err != nil {
		return nil, err
	}

	// 验证视频格式和大小
	if err := uc.processor.ValidateFormat(filename, int64(len(videoData))); err != nil {
		return nil, err
	}

	// 生成视频ID
	videoID := utils.MustGenerateID()

	// 上传视频到存储
	playURL, err := uc.uploadVideoToStorage(ctx, videoData, filename)
	if err != nil {
		uc.log.WithContext(ctx).Errorf("upload video to storage failed: %v", err)
		return nil, fmt.Errorf("video upload failed")
	}

	// 生成封面
	coverURL, err := uc.generateAndUploadCover(ctx, videoData, videoID)
	if err != nil {
		uc.log.WithContext(ctx).Warnf("generate cover failed: %v", err)
		coverURL = ""
	}

	// 创建视频记录
	video := &domain.Video{
		ID:            videoID,
		AuthorID:      authorID,
		Title:         title,
		PlayURL:       playURL,
		CoverURL:      coverURL,
		FavoriteCount: 0,
		CommentCount:  0,
		PlayCount:     0,
		Status:        domain.VideoStatusPublished,
	}

	// 保存到数据库
	if err := uc.repo.CreateVideo(ctx, video); err != nil {
		uc.cleanupUploadedFiles(ctx, playURL, coverURL)
		return nil, err
	}

	// 发送视频上传事件到Kafka
	uc.publishVideoUploadedEvent(ctx, video)

	// 异步处理视频
	go uc.processVideoAsync(context.Background(), video)

	uc.log.WithContext(ctx).Infof("video published successfully: %d", videoID)
	return video, nil
}

// InitiateMultipartUpload 初始化分片上传
func (uc *VideoUsecase) InitiateMultipartUpload(ctx context.Context, filename string, totalSize int64, contentType, title string) (*storage.MultipartUploadInfo, error) {
	// 验证文件格式
	if err := uc.processor.ValidateFormat(filename, totalSize); err != nil {
		return nil, err
	}

	// 验证标题
	if err := uc.validator.ValidateVideoTitle(title); err != nil {
		return nil, err
	}

	// 如果存储支持分片上传
	if multipartStorage, ok := uc.storage.(storage.MultipartStorage); ok {
		opts := &storage.MultipartUploadOptions{
			ContentType: contentType,
			ChunkSize:   4 * 1024 * 1024, // 4MB
			Metadata: map[string]string{
				"title":    title,
				"filename": filename,
			},
		}
		return multipartStorage.InitiateMultipartUpload(ctx, filename, opts)
	}

	return nil, fmt.Errorf("storage does not support multipart upload")
}

// UploadPart 上传分片
func (uc *VideoUsecase) UploadPart(ctx context.Context, uploadID string, partNumber int, reader io.Reader, size int64) (*storage.PartInfo, error) {
	if multipartStorage, ok := uc.storage.(storage.MultipartStorage); ok {
		return multipartStorage.UploadPart(ctx, uploadID, partNumber, reader, size)
	}
	return nil, fmt.Errorf("storage does not support multipart upload")
}

// CompleteMultipartUpload 完成分片上传
func (uc *VideoUsecase) CompleteMultipartUpload(ctx context.Context, uploadID string, parts []storage.PartInfo, title string, userID int64) (*domain.Video, error) {
	multipartStorage, ok := uc.storage.(storage.MultipartStorage)
	if !ok {
		return nil, fmt.Errorf("storage does not support multipart upload")
	}

	// 完成分片上传
	fileInfo, err := multipartStorage.CompleteMultipartUpload(ctx, uploadID, parts)
	if err != nil {
		return nil, err
	}

	// 创建视频记录
	video := &domain.Video{
		ID:            utils.MustGenerateID(),
		AuthorID:      userID,
		Title:         title,
		PlayURL:       fileInfo.URL,
		FavoriteCount: 0,
		CommentCount:  0,
		PlayCount:     0,
		Status:        domain.VideoStatusPending,
	}

	if err := uc.repo.CreateVideo(ctx, video); err != nil {
		return nil, err
	}

	// 发送处理事件
	uc.publishVideoUploadedEvent(ctx, video)
	return video, nil
}

// AbortMultipartUpload 取消分片上传
func (uc *VideoUsecase) AbortMultipartUpload(ctx context.Context, uploadID string) error {
	if multipartStorage, ok := uc.storage.(storage.MultipartStorage); ok {
		return multipartStorage.AbortMultipartUpload(ctx, uploadID)
	}
	return fmt.Errorf("storage does not support multipart upload")
}

// ListUploadedParts 列出已上传的分片
func (uc *VideoUsecase) ListUploadedParts(ctx context.Context, uploadID string) ([]storage.PartInfo, error) {
	if multipartStorage, ok := uc.storage.(storage.MultipartStorage); ok {
		return multipartStorage.ListParts(ctx, uploadID)
	}
	return nil, fmt.Errorf("storage does not support multipart upload")
}

// GetFeed 获取视频流
func (uc *VideoUsecase) GetFeed(ctx context.Context, latestTime int64, limit int) ([]*domain.Video, int64, error) {
	if limit <= 0 || limit > int(uc.businessConfig.Video.DefaultFeedLimit) {
		limit = int(uc.businessConfig.Video.DefaultFeedLimit)
	}

	var feedTime time.Time
	if latestTime > 0 {
		feedTime = time.Unix(latestTime, 0)
	} else {
		feedTime = time.Now()
	}

	// 先尝试从缓存获取
	if videos, ok := uc.cache.GetFeedVideos(ctx, latestTime); ok && len(videos) >= limit {
		nextTime := uc.getNextTime(videos, limit)
		return videos[:limit], nextTime, nil
	}

	// 从数据库获取
	videos, err := uc.repo.GetFeedVideos(ctx, feedTime, limit)
	if err != nil {
		return nil, 0, err
	}

	// 缓存结果
	if len(videos) > 0 {
		uc.cache.SetFeedVideos(ctx, latestTime, videos)
	}

	nextTime := uc.getNextTime(videos, limit)
	return videos, nextTime, nil
}

// GetPublishList 获取用户发布列表
func (uc *VideoUsecase) GetPublishList(ctx context.Context, userID int64) ([]*domain.Video, error) {
	if err := uc.validator.ValidateUserID(userID); err != nil {
		return nil, err
	}

	videos, err := uc.repo.GetUserVideos(ctx, userID, 100)
	if err != nil {
		return nil, err
	}

	return videos, nil
}

// GetVideo 获取视频信息
func (uc *VideoUsecase) GetVideo(ctx context.Context, videoID int64) (*domain.Video, error) {
	if err := uc.validator.ValidateVideoID(videoID); err != nil {
		return nil, err
	}

	video, err := uc.repo.GetVideo(ctx, videoID)
	if err != nil {
		return nil, err
	}

	// 异步增加播放计数
	go func() {
		uc.IncrementPlayCount(context.Background(), videoID)
	}()

	return video, nil
}

// GetVideos 批量获取视频信息
func (uc *VideoUsecase) GetVideos(ctx context.Context, videoIDs []int64) ([]*domain.Video, error) {
	if len(videoIDs) == 0 {
		return []*domain.Video{}, nil
	}

	return uc.repo.GetVideos(ctx, videoIDs)
}

// UpdateVideoStats 更新视频统计
func (uc *VideoUsecase) UpdateVideoStats(ctx context.Context, videoID int64, statsType string, delta int64) error {
	if err := uc.validator.ValidateVideoID(videoID); err != nil {
		return err
	}

	var field string
	switch statsType {
	case "favorite":
		field = "favorite_count"
	case "comment":
		field = "comment_count"
	case "play":
		field = "play_count"
	default:
		return fmt.Errorf("invalid stats type: %s", statsType)
	}

	// 更新数据库
	if err := uc.repo.UpdateVideoStats(ctx, videoID, field, delta); err != nil {
		return err
	}

	// 更新缓存
	uc.cache.IncrVideoStats(ctx, videoID, field, delta)

	// 发送统计更新事件到Kafka
	uc.publishVideoStatsUpdatedEvent(ctx, videoID, statsType, delta)

	return nil
}

// IncrementPlayCount 增加播放数
func (uc *VideoUsecase) IncrementPlayCount(ctx context.Context, videoID int64) error {
	return uc.UpdateVideoStats(ctx, videoID, "play", 1)
}

// IncrementFavoriteCount 增加点赞数
func (uc *VideoUsecase) IncrementFavoriteCount(ctx context.Context, videoID int64) error {
	return uc.UpdateVideoStats(ctx, videoID, "favorite", 1)
}

// DecrementFavoriteCount 减少点赞数
func (uc *VideoUsecase) DecrementFavoriteCount(ctx context.Context, videoID int64) error {
	return uc.UpdateVideoStats(ctx, videoID, "favorite", -1)
}

// IncrementCommentCount 增加评论数
func (uc *VideoUsecase) IncrementCommentCount(ctx context.Context, videoID int64) error {
	return uc.UpdateVideoStats(ctx, videoID, "comment", 1)
}

// DecrementCommentCount 减少评论数
func (uc *VideoUsecase) DecrementCommentCount(ctx context.Context, videoID int64) error {
	return uc.UpdateVideoStats(ctx, videoID, "comment", -1)
}

// GetUploadConfig 获取上传配置
func (uc *VideoUsecase) GetUploadConfig(ctx context.Context) (*UploadConfig, error) {
	return &UploadConfig{
		MaxFileSize:      uc.processor.GetMaxFileSize(),
		SupportedFormats: uc.processor.GetSupportedFormats(),
		ChunkSize:        4 * 1024 * 1024, // 4MB
		EnableResume:     true,            // 支持断点续传
	}, nil
}

// GetUploadProgress 获取上传进度
func (uc *VideoUsecase) GetUploadProgress(ctx context.Context, uploadID string) (*UploadProgress, error) {
	// 如果存储支持断点续传
	if resumableStorage, ok := uc.storage.(storage.ResumableStorage); ok {
		uploadedSize, err := resumableStorage.GetUploadProgress(ctx, uploadID)
		if err != nil {
			return nil, err
		}

		// 这里可以从缓存或数据库获取总大小
		return &UploadProgress{
			UploadID:      uploadID,
			Progress:      50, // 简化计算
			Status:        "uploading",
			TotalSize:     0,
			UploadedSize:  uploadedSize,
			EstimatedTime: 0,
		}, nil
	}

	return &UploadProgress{
		UploadID: uploadID,
		Progress: 100,
		Status:   "completed",
	}, nil
}

// UpdateVideoCover 更新视频封面
func (uc *VideoUsecase) UpdateVideoCover(ctx context.Context, videoID int64, coverURL string) error {
	if err := uc.repo.UpdateVideoCover(ctx, videoID, coverURL); err != nil {
		return err
	}

	// 清除缓存
	uc.cache.DeleteVideo(ctx, videoID)
	return nil
}

// UpdateVideoPlayURL 更新视频播放URL
func (uc *VideoUsecase) UpdateVideoPlayURL(ctx context.Context, videoID int64, playURL string) error {
	if err := uc.repo.UpdateVideoPlayURL(ctx, videoID, playURL); err != nil {
		return err
	}

	// 清除缓存
	uc.cache.DeleteVideo(ctx, videoID)
	return nil
}

// 内部辅助方法

func (uc *VideoUsecase) uploadVideoToStorage(ctx context.Context, videoData []byte, filename string) (string, error) {
	objectName := utils.GenerateVideoFilename(filename)
	return uc.storage.UploadVideo(ctx, objectName, strings.NewReader(string(videoData)), int64(len(videoData)))
}

func (uc *VideoUsecase) generateAndUploadCover(ctx context.Context, videoData []byte, videoID int64) (string, error) {
	coverReader, err := uc.processor.GenerateDefaultThumbnail(ctx)
	if err != nil {
		return "", err
	}

	coverData, err := io.ReadAll(coverReader)
	if err != nil {
		return "", err
	}

	coverFilename := fmt.Sprintf("cover_%d.jpg", videoID)
	return uc.storage.UploadCover(ctx, coverFilename, strings.NewReader(string(coverData)), int64(len(coverData)))
}

func (uc *VideoUsecase) publishVideoUploadedEvent(ctx context.Context, video *domain.Video) {
	if uc.kafkaManager == nil {
		return
	}

	event := &messaging.VideoUploadEvent{
		VideoID:    video.ID,
		AuthorID:   video.AuthorID,
		Title:      video.Title,
		PlayURL:    video.PlayURL,
		UploadTime: video.CreatedAt.Unix(),
	}

	if err := uc.kafkaManager.SendVideoUploadEvent(ctx, uc.businessConfig.KafkaTopics.VideoUpload, event); err != nil {
		uc.log.WithContext(ctx).Errorf("send video upload event failed: %v", err)
	}
}

func (uc *VideoUsecase) publishVideoStatsUpdatedEvent(ctx context.Context, videoID int64, statsType string, delta int64) {
	if uc.kafkaManager == nil {
		return
	}

	event := &messaging.VideoStatsEvent{
		VideoID:   videoID,
		StatsType: statsType,
		Count:     delta,
		UserID:    0,
	}

	if err := uc.kafkaManager.SendVideoStatsEvent(ctx, uc.businessConfig.KafkaTopics.VideoStats, event); err != nil {
		uc.log.WithContext(ctx).Errorf("send video stats event failed: %v", err)
	}
}

func (uc *VideoUsecase) processVideoAsync(ctx context.Context, video *domain.Video) {
	if uc.kafkaManager == nil {
		return
	}

	event := &messaging.VideoProcessEvent{
		VideoID:     video.ID,
		ProcessType: "transcode",
		Status:      "processing",
	}

	if err := uc.kafkaManager.SendVideoProcessEvent(ctx, uc.businessConfig.KafkaTopics.VideoProcess, event); err != nil {
		uc.log.WithContext(ctx).Errorf("send video process event failed: %v", err)
	}
}

func (uc *VideoUsecase) cleanupUploadedFiles(ctx context.Context, playURL, coverURL string) {
	if playURL != "" {
		objectName := uc.extractObjectName(playURL)
		if err := uc.storage.Delete(ctx, objectName); err != nil {
			uc.log.WithContext(ctx).Warnf("cleanup video file failed: %v", err)
		}
	}

	if coverURL != "" {
		objectName := uc.extractObjectName(coverURL)
		if err := uc.storage.Delete(ctx, objectName); err != nil {
			uc.log.WithContext(ctx).Warnf("cleanup cover file failed: %v", err)
		}
	}
}

func (uc *VideoUsecase) extractObjectName(url string) string {
	parts := strings.Split(url, "/")
	if len(parts) >= 1 {
		return parts[len(parts)-1]
	}
	return url
}

func (uc *VideoUsecase) getNextTime(videos []*domain.Video, limit int) int64 {
	if len(videos) == 0 {
		return 0
	}

	lastIndex := len(videos) - 1
	if len(videos) > limit {
		lastIndex = limit - 1
	}

	return videos[lastIndex].CreatedAt.Unix()
}

// UploadConfig 上传配置
type UploadConfig struct {
	MaxFileSize      int64    `json:"max_file_size"`
	SupportedFormats []string `json:"supported_formats"`
	ChunkSize        int64    `json:"chunk_size"`
	EnableResume     bool     `json:"enable_resume"`
}

// UploadProgress 上传进度
type UploadProgress struct {
	UploadID      string `json:"upload_id"`
	Progress      int32  `json:"progress"`
	Status        string `json:"status"`
	TotalSize     int64  `json:"total_size"`
	UploadedSize  int64  `json:"uploaded_size"`
	ErrorMessage  string `json:"error_message,omitempty"`
	EstimatedTime int64  `json:"estimated_time"`
}
