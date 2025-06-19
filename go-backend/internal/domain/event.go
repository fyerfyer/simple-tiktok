package domain

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// DomainEvent 领域事件接口
type DomainEvent interface {
	GetEventID() string
	GetEventType() string
	GetAggregateID() string
	GetEventTime() time.Time
	GetVersion() int
	ToJSON() ([]byte, error)
}

// BaseEvent 基础事件结构
type BaseEvent struct {
	EventID     string    `json:"event_id"`
	EventType   string    `json:"event_type"`
	AggregateID string    `json:"aggregate_id"`
	EventTime   time.Time `json:"event_time"`
	Version     int       `json:"version"`
}

// GetEventID 获取事件ID
func (e *BaseEvent) GetEventID() string {
	return e.EventID
}

// GetEventType 获取事件类型
func (e *BaseEvent) GetEventType() string {
	return e.EventType
}

// GetAggregateID 获取聚合根ID
func (e *BaseEvent) GetAggregateID() string {
	return e.AggregateID
}

// GetEventTime 获取事件时间
func (e *BaseEvent) GetEventTime() time.Time {
	return e.EventTime
}

// GetVersion 获取版本号
func (e *BaseEvent) GetVersion() int {
	return e.Version
}

// ToJSON 转换为JSON
func (e *BaseEvent) ToJSON() ([]byte, error) {
	return json.Marshal(e)
}

// EventPublisher 事件发布器接口
type EventPublisher interface {
	Publish(ctx context.Context, topic string, event DomainEvent) error
	PublishBatch(ctx context.Context, topic string, events []DomainEvent) error
}

// EventHandler 事件处理器接口
type EventHandler interface {
	Handle(ctx context.Context, event DomainEvent) error
	GetEventType() string
}

// EventStore 事件存储接口
type EventStore interface {
	SaveEvent(ctx context.Context, event DomainEvent) error
	SaveEvents(ctx context.Context, events []DomainEvent) error
	GetEvents(ctx context.Context, aggregateID string, fromVersion int) ([]DomainEvent, error)
	GetEventsByType(ctx context.Context, eventType string, limit int) ([]DomainEvent, error)
}

// UserRelatedEvent 用户相关事件
type UserRelatedEvent interface {
	DomainEvent
	GetUserID() int64
}

// UserRegisteredEvent 用户注册事件
type UserRegisteredEvent struct {
	BaseEvent
	UserID       int64     `json:"user_id"`
	Username     string    `json:"username"`
	Nickname     string    `json:"nickname"`
	RegisterIP   string    `json:"register_ip"`
	RegisteredAt time.Time `json:"registered_at"`
}

// GetUserID 获取用户ID
func (e *UserRegisteredEvent) GetUserID() int64 {
	return e.UserID
}

// UserFollowedEvent 用户关注事件
type UserFollowedEvent struct {
	BaseEvent
	UserID       int64     `json:"user_id"`
	FollowUserID int64     `json:"follow_user_id"`
	FollowedAt   time.Time `json:"followed_at"`
}

// GetUserID 获取用户ID
func (e *UserFollowedEvent) GetUserID() int64 {
	return e.UserID
}

// UserUnfollowedEvent 用户取消关注事件
type UserUnfollowedEvent struct {
	BaseEvent
	UserID         int64     `json:"user_id"`
	UnfollowUserID int64     `json:"unfollow_user_id"`
	UnfollowedAt   time.Time `json:"unfollowed_at"`
}

// GetUserID 获取用户ID
func (e *UserUnfollowedEvent) GetUserID() int64 {
	return e.UserID
}

// SocialInteractionEvent 社交互动事件
type SocialInteractionEvent interface {
	DomainEvent
	GetUserID() int64
	GetTargetID() int64
	GetInteractionType() string
}

// VideoLikedEvent 视频点赞事件
type VideoLikedEvent struct {
	BaseEvent
	UserID   int64     `json:"user_id"`
	VideoID  int64     `json:"video_id"`
	AuthorID int64     `json:"author_id"`
	LikedAt  time.Time `json:"liked_at"`
}

// GetUserID 获取用户ID
func (e *VideoLikedEvent) GetUserID() int64 {
	return e.UserID
}

// GetTargetID 获取目标ID
func (e *VideoLikedEvent) GetTargetID() int64 {
	return e.VideoID
}

// GetInteractionType 获取互动类型
func (e *VideoLikedEvent) GetInteractionType() string {
	return "like"
}

// VideoUnlikedEvent 视频取消点赞事件
type VideoUnlikedEvent struct {
	BaseEvent
	UserID    int64     `json:"user_id"`
	VideoID   int64     `json:"video_id"`
	AuthorID  int64     `json:"author_id"`
	UnlikedAt time.Time `json:"unliked_at"`
}

// GetUserID 获取用户ID
func (e *VideoUnlikedEvent) GetUserID() int64 {
	return e.UserID
}

// GetTargetID 获取目标ID
func (e *VideoUnlikedEvent) GetTargetID() int64 {
	return e.VideoID
}

// GetInteractionType 获取互动类型
func (e *VideoUnlikedEvent) GetInteractionType() string {
	return "unlike"
}

// CommentCreatedEvent 评论创建事件
type CommentCreatedEvent struct {
	BaseEvent
	CommentID       int64     `json:"comment_id"`
	VideoID         int64     `json:"video_id"`
	UserID          int64     `json:"user_id"`
	AuthorID        int64     `json:"author_id"`
	Content         string    `json:"content"`
	ParentCommentID int64     `json:"parent_comment_id,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
}

// GetUserID 获取用户ID
func (e *CommentCreatedEvent) GetUserID() int64 {
	return e.UserID
}

// GetTargetID 获取目标ID
func (e *CommentCreatedEvent) GetTargetID() int64 {
	return e.VideoID
}

// GetInteractionType 获取互动类型
func (e *CommentCreatedEvent) GetInteractionType() string {
	return "comment"
}

// CommentDeletedEvent 评论删除事件
type CommentDeletedEvent struct {
	BaseEvent
	CommentID int64     `json:"comment_id"`
	VideoID   int64     `json:"video_id"`
	UserID    int64     `json:"user_id"`
	DeletedAt time.Time `json:"deleted_at"`
}

// GetUserID 获取用户ID
func (e *CommentDeletedEvent) GetUserID() int64 {
	return e.UserID
}

// GetTargetID 获取目标ID
func (e *CommentDeletedEvent) GetTargetID() int64 {
	return e.VideoID
}

// GetInteractionType 获取互动类型
func (e *CommentDeletedEvent) GetInteractionType() string {
	return "delete_comment"
}

// MessageSentEvent 消息发送事件
type MessageSentEvent struct {
	BaseEvent
	MessageID   int64     `json:"message_id"`
	FromUserID  int64     `json:"from_user_id"`
	ToUserID    int64     `json:"to_user_id"`
	Content     string    `json:"content"`
	MessageType string    `json:"message_type"` // text, image, video
	SentAt      time.Time `json:"sent_at"`
}

// GetUserID 获取用户ID
func (e *MessageSentEvent) GetUserID() int64 {
	return e.FromUserID
}

// GetTargetID 获取目标ID
func (e *MessageSentEvent) GetTargetID() int64 {
	return e.ToUserID
}

// GetInteractionType 获取互动类型
func (e *MessageSentEvent) GetInteractionType() string {
	return "message"
}

// SystemEvent 系统事件接口
type SystemEvent interface {
	DomainEvent
	GetSeverity() string
}

// SystemErrorEvent 系统错误事件
type SystemErrorEvent struct {
	BaseEvent
	ErrorCode    string `json:"error_code"`
	ErrorMessage string `json:"error_message"`
	ServiceName  string `json:"service_name"`
	Severity     string `json:"severity"` // low, medium, high, critical
	StackTrace   string `json:"stack_trace,omitempty"`
}

// GetSeverity 获取严重程度
func (e *SystemErrorEvent) GetSeverity() string {
	return e.Severity
}

// CacheInvalidationEvent 缓存失效事件
type CacheInvalidationEvent struct {
	BaseEvent
	CacheKey      string    `json:"cache_key"`
	CacheType     string    `json:"cache_type"` // user, video, feed
	InvalidatedAt time.Time `json:"invalidated_at"`
}

// EventFactory 事件工厂
type EventFactory struct{}

// NewEventFactory 创建事件工厂
func NewEventFactory() *EventFactory {
	return &EventFactory{}
}

// CreateVideoUploadedEvent 创建视频上传事件
func (f *EventFactory) CreateVideoUploadedEvent(video *Video) *VideoUploadedEvent {
	return &VideoUploadedEvent{
		VideoID:    video.ID,
		AuthorID:   video.AuthorID,
		Title:      video.Title,
		PlayURL:    video.PlayURL,
		CoverURL:   video.CoverURL,
		UploadedAt: video.CreatedAt,
		EventID:    generateEventID(),
		EventTime:  time.Now(),
	}
}

// CreateVideoStatsUpdatedEvent 创建视频统计更新事件
func (f *EventFactory) CreateVideoStatsUpdatedEvent(videoID int64, statsType string, oldValue, newValue int64, userID int64) *VideoStatsUpdatedEvent {
	return &VideoStatsUpdatedEvent{
		VideoID:   videoID,
		StatsType: statsType,
		OldValue:  oldValue,
		NewValue:  newValue,
		Delta:     newValue - oldValue,
		UserID:    userID,
		UpdatedAt: time.Now(),
		EventID:   generateEventID(),
		EventTime: time.Now(),
	}
}

// CreateVideoLikedEvent 创建视频点赞事件
func (f *EventFactory) CreateVideoLikedEvent(userID, videoID, authorID int64) *VideoLikedEvent {
	return &VideoLikedEvent{
		BaseEvent: BaseEvent{
			EventID:     generateEventID(),
			EventType:   "video.liked",
			AggregateID: fmt.Sprintf("video:%d", videoID),
			EventTime:   time.Now(),
			Version:     1,
		},
		UserID:   userID,
		VideoID:  videoID,
		AuthorID: authorID,
		LikedAt:  time.Now(),
	}
}

// CreateCommentCreatedEvent 创建评论创建事件
func (f *EventFactory) CreateCommentCreatedEvent(commentID, videoID, userID, authorID int64, content string, parentCommentID int64) *CommentCreatedEvent {
	return &CommentCreatedEvent{
		BaseEvent: BaseEvent{
			EventID:     generateEventID(),
			EventType:   "comment.created",
			AggregateID: fmt.Sprintf("comment:%d", commentID),
			EventTime:   time.Now(),
			Version:     1,
		},
		CommentID:       commentID,
		VideoID:         videoID,
		UserID:          userID,
		AuthorID:        authorID,
		Content:         content,
		ParentCommentID: parentCommentID,
		CreatedAt:       time.Now(),
	}
}

// EventBus 事件总线接口
type EventBus interface {
	Subscribe(eventType string, handler EventHandler) error
	Unsubscribe(eventType string, handler EventHandler) error
	Publish(ctx context.Context, event DomainEvent) error
	PublishAsync(ctx context.Context, event DomainEvent) error
}

// generateEventID 生成事件ID
func generateEventID() string {
	return fmt.Sprintf("evt_%d_%s", time.Now().UnixNano(), randomString(8))
}

// randomString 生成随机字符串
func randomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[time.Now().UnixNano()%int64(len(charset))]
	}
	return string(b)
}

// 事件类型常量
const (
	EventTypeVideoUploaded     = "video.uploaded"
	EventTypeVideoProcessed    = "video.processed"
	EventTypeVideoStatsUpdated = "video.stats.updated"
	EventTypeVideoDeleted      = "video.deleted"
	EventTypeVideoAudited      = "video.audited"

	EventTypeUserRegistered = "user.registered"
	EventTypeUserFollowed   = "user.followed"
	EventTypeUserUnfollowed = "user.unfollowed"

	EventTypeVideoLiked     = "video.liked"
	EventTypeVideoUnliked   = "video.unliked"
	EventTypeCommentCreated = "comment.created"
	EventTypeCommentDeleted = "comment.deleted"
	EventTypeMessageSent    = "message.sent"

	EventTypeSystemError       = "system.error"
	EventTypeCacheInvalidation = "cache.invalidation"
)

// 事件优先级常量
const (
	EventPriorityLow      = "low"
	EventPriorityMedium   = "medium"
	EventPriorityHigh     = "high"
	EventPriorityCritical = "critical"
)
