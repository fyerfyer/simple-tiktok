package messaging

import (
	"encoding/json"
	"time"
)

// MessageType 消息类型
type MessageType string

const (
	VideoUploadMessage  MessageType = "video_upload"
	VideoProcessMessage MessageType = "video_process"
	VideoStatsMessage   MessageType = "video_stats"
	UserActionMessage   MessageType = "user_action"
)

// BaseMessage 基础消息结构
type BaseMessage struct {
	ID        string      `json:"id"`
	Type      MessageType `json:"type"`
	Timestamp int64       `json:"timestamp"`
	Data      interface{} `json:"data"`
}

// NewBaseMessage 创建基础消息
func NewBaseMessage(msgType MessageType, data interface{}) *BaseMessage {
	return &BaseMessage{
		ID:        generateMessageID(),
		Type:      msgType,
		Timestamp: time.Now().Unix(),
		Data:      data,
	}
}

// ToJSON 转换为JSON字符串
func (m *BaseMessage) ToJSON() ([]byte, error) {
	return json.Marshal(m)
}

// FromJSON 从JSON字符串解析
func (m *BaseMessage) FromJSON(data []byte) error {
	return json.Unmarshal(data, m)
}

// VideoUploadEvent 视频上传事件
type VideoUploadEvent struct {
	VideoID    int64  `json:"video_id"`
	AuthorID   int64  `json:"author_id"`
	Title      string `json:"title"`
	PlayURL    string `json:"play_url"`
	UploadTime int64  `json:"upload_time"`
}

// VideoProcessEvent 视频处理事件
type VideoProcessEvent struct {
	VideoID     int64  `json:"video_id"`
	ProcessType string `json:"process_type"` // transcode, thumbnail, etc.
	Status      string `json:"status"`       // processing, completed, failed
	Result      string `json:"result,omitempty"`
	Error       string `json:"error,omitempty"`
}

// VideoStatsEvent 视频统计事件
type VideoStatsEvent struct {
	VideoID   int64  `json:"video_id"`
	StatsType string `json:"stats_type"` // play, like, comment, share
	Count     int64  `json:"count"`
	UserID    int64  `json:"user_id,omitempty"`
}

// UserActionEvent 用户行为事件
type UserActionEvent struct {
	UserID     int64  `json:"user_id"`
	ActionType string `json:"action_type"` // follow, unfollow, like, unlike
	TargetID   int64  `json:"target_id"`
	TargetType string `json:"target_type"` // user, video
	Timestamp  int64  `json:"timestamp"`
}

// generateMessageID 生成消息ID
func generateMessageID() string {
	return time.Now().Format("20060102150405") + randomString(6)
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
