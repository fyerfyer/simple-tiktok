package domain

import (
	"context"
	"fmt"
	"time"
)

// Video 视频领域模型
type Video struct {
	ID            int64     `json:"id"`
	AuthorID      int64     `json:"author_id"`
	Title         string    `json:"title"`
	PlayURL       string    `json:"play_url"`
	CoverURL      string    `json:"cover_url"`
	FavoriteCount int64     `json:"favorite_count"`
	CommentCount  int64     `json:"comment_count"`
	PlayCount     int64     `json:"play_count"`
	Status        int32     `json:"status"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// VideoFile 视频文件信息
type VideoFile struct {
	Data        []byte `json:"data"`
	Filename    string `json:"filename"`
	Size        int64  `json:"size"`
	ContentType string `json:"content_type"`
}

// VideoRepository 视频数据仓储接口
type VideoRepository interface {
	CreateVideo(ctx context.Context, video *Video) error
	GetVideo(ctx context.Context, videoID int64) (*Video, error)
	GetVideos(ctx context.Context, videoIDs []int64) ([]*Video, error)
	GetUserVideos(ctx context.Context, userID int64, limit int) ([]*Video, error)
	GetFeedVideos(ctx context.Context, latestTime time.Time, limit int) ([]*Video, error)
	UpdateVideoStats(ctx context.Context, videoID int64, field string, delta int64) error
	UpdateVideo(ctx context.Context, video *Video) error
}

// VideoStorageRepository 视频存储仓储接口
type VideoStorageRepository interface {
	UploadVideo(ctx context.Context, file *VideoFile) (string, error)
	UploadCover(ctx context.Context, file *VideoFile) (string, error)
	DeleteFile(ctx context.Context, url string) error
	GetPreviewURL(ctx context.Context, url string) (string, error)
}

// VideoProcessor 视频处理接口
type VideoProcessor interface {
	GenerateCover(ctx context.Context, videoURL string) ([]byte, error)
	ValidateFormat(ctx context.Context, file *VideoFile) error
	GetMetadata(ctx context.Context, file *VideoFile) (*VideoMetadata, error)
}

// VideoMetadata 视频元信息
type VideoMetadata struct {
	Duration  int64  `json:"duration"`  // 时长(秒)
	Width     int32  `json:"width"`     // 宽度
	Height    int32  `json:"height"`    // 高度
	Bitrate   int64  `json:"bitrate"`   // 比特率
	Format    string `json:"format"`    // 格式
	Size      int64  `json:"size"`      // 文件大小
	Framerate string `json:"framerate"` // 帧率
}

// VideoEventPublisher 视频事件发布器接口
type VideoEventPublisher interface {
	PublishVideoUploadedEvent(ctx context.Context, event *VideoUploadedEvent) error
	PublishVideoProcessedEvent(ctx context.Context, event *VideoProcessedEvent) error
	PublishVideoStatsUpdatedEvent(ctx context.Context, event *VideoStatsUpdatedEvent) error
	PublishVideoDeletedEvent(ctx context.Context, event *VideoDeletedEvent) error
}

// VideoUploadedEvent 视频上传事件
type VideoUploadedEvent struct {
	VideoID    int64     `json:"video_id"`
	AuthorID   int64     `json:"author_id"`
	Title      string    `json:"title"`
	PlayURL    string    `json:"play_url"`
	CoverURL   string    `json:"cover_url"`
	Size       int64     `json:"size"`
	Format     string    `json:"format"`
	UploadedAt time.Time `json:"uploaded_at"`
	EventID    string    `json:"event_id"`
	EventTime  time.Time `json:"event_time"`
}

// VideoProcessedEvent 视频处理事件
type VideoProcessedEvent struct {
	VideoID      int64     `json:"video_id"`
	ProcessType  string    `json:"process_type"` // transcode, thumbnail, audit
	Status       string    `json:"status"`       // success, failed
	Result       string    `json:"result,omitempty"`
	ErrorMessage string    `json:"error_message,omitempty"`
	ProcessedAt  time.Time `json:"processed_at"`
	EventID      string    `json:"event_id"`
	EventTime    time.Time `json:"event_time"`
}

// VideoStatsUpdatedEvent 视频统计更新事件
type VideoStatsUpdatedEvent struct {
	VideoID   int64     `json:"video_id"`
	StatsType string    `json:"stats_type"` // play, favorite, comment
	OldValue  int64     `json:"old_value"`
	NewValue  int64     `json:"new_value"`
	Delta     int64     `json:"delta"`
	UserID    int64     `json:"user_id,omitempty"`
	UpdatedAt time.Time `json:"updated_at"`
	EventID   string    `json:"event_id"`
	EventTime time.Time `json:"event_time"`
}

// VideoDeletedEvent 视频删除事件
type VideoDeletedEvent struct {
	VideoID   int64     `json:"video_id"`
	AuthorID  int64     `json:"author_id"`
	Title     string    `json:"title"`
	PlayURL   string    `json:"play_url"`
	CoverURL  string    `json:"cover_url"`
	DeletedAt time.Time `json:"deleted_at"`
	EventID   string    `json:"event_id"`
	EventTime time.Time `json:"event_time"`
}

// VideoAuditEvent 视频审核事件
type VideoAuditEvent struct {
	VideoID     int64     `json:"video_id"`
	AuthorID    int64     `json:"author_id"`
	AuditStatus string    `json:"audit_status"` // pending, approved, rejected
	AuditReason string    `json:"audit_reason,omitempty"`
	AuditorID   int64     `json:"auditor_id,omitempty"`
	AuditedAt   time.Time `json:"audited_at"`
	EventID     string    `json:"event_id"`
	EventTime   time.Time `json:"event_time"`
}

// 视频状态常量
const (
	VideoStatusPending   = 0 // 处理中
	VideoStatusPublished = 1 // 已发布
	VideoStatusPrivate   = 2 // 私密
	VideoStatusDeleted   = 3 // 已删除
	VideoStatusFailed    = 4 // 处理失败
	VideoStatusAuditing  = 5 // 审核中
	VideoStatusRejected  = 6 // 审核拒绝
)

// 视频处理类型常量
const (
	ProcessTypeTranscode = "transcode"
	ProcessTypeThumbnail = "thumbnail"
	ProcessTypeAudit     = "audit"
	ProcessTypeWatermark = "watermark"
)

// 视频处理状态常量
const (
	ProcessStatusPending    = "pending"
	ProcessStatusProcessing = "processing"
	ProcessStatusSuccess    = "success"
	ProcessStatusFailed     = "failed"
)

// 视频统计类型常量
const (
	StatsTypePlay     = "play"
	StatsTypeFavorite = "favorite"
	StatsTypeComment  = "comment"
	StatsTypeShare    = "share"
)

// 视频审核状态常量
const (
	AuditStatusPending  = "pending"
	AuditStatusApproved = "approved"
	AuditStatusRejected = "rejected"
)

// 视频文件限制
const (
	MaxVideoSize      = 100 * 1024 * 1024 // 100MB
	MaxTitleLength    = 50                // 标题最大长度
	DefaultVideoLimit = 30                // 默认视频列表数量
)

// 支持的视频格式
var SupportedVideoFormats = []string{
	"video/mp4",
	"video/avi",
	"video/quicktime", // .mov
}

// GetEventType 获取事件类型
func (e *VideoUploadedEvent) GetEventType() string {
	return "video.uploaded"
}

func (e *VideoProcessedEvent) GetEventType() string {
	return "video.processed"
}

func (e *VideoStatsUpdatedEvent) GetEventType() string {
	return "video.stats.updated"
}

func (e *VideoDeletedEvent) GetEventType() string {
	return "video.deleted"
}

func (e *VideoAuditEvent) GetEventType() string {
	return "video.audited"
}

// GetAggregateID 获取聚合根ID
func (e *VideoUploadedEvent) GetAggregateID() string {
	return fmt.Sprintf("video:%d", e.VideoID)
}

func (e *VideoProcessedEvent) GetAggregateID() string {
	return fmt.Sprintf("video:%d", e.VideoID)
}

func (e *VideoStatsUpdatedEvent) GetAggregateID() string {
	return fmt.Sprintf("video:%d", e.VideoID)
}

func (e *VideoDeletedEvent) GetAggregateID() string {
	return fmt.Sprintf("video:%d", e.VideoID)
}

func (e *VideoAuditEvent) GetAggregateID() string {
	return fmt.Sprintf("video:%d", e.VideoID)
}
