package domain

import "time"

// User 用户领域模型
type User struct {
	ID              int64      `json:"id"`
	Username        string     `json:"username"`
	PasswordHash    string     `json:"-"`
	Salt            string     `json:"-"`
	Nickname        string     `json:"nickname"`
	Avatar          string     `json:"avatar"`
	BackgroundImage string     `json:"background_image"`
	Signature       string     `json:"signature"`
	FollowCount     int        `json:"follow_count"`
	FollowerCount   int        `json:"follower_count"`
	TotalFavorited  int64      `json:"total_favorited"`
	WorkCount       int        `json:"work_count"`
	FavoriteCount   int        `json:"favorite_count"`
	Status          int8       `json:"status"`
	LastLoginAt     *time.Time `json:"last_login_at"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

// UserStats 用户统计信息
type UserStats struct {
	FollowCountDelta    int
	FollowerCountDelta  int
	WorkCountDelta      int
	FavoriteCountDelta  int
	TotalFavoritedDelta int64
}

// UserStatus 用户状态枚举
type UserStatus int8

const (
	UserStatusActive   UserStatus = 1 // 正常
	UserStatusInactive UserStatus = 2 // 禁用
)

// IsActive 检查用户是否激活
func (u *User) IsActive() bool {
	return u.Status == int8(UserStatusActive)
}

// UpdateLoginTime 更新登录时间
func (u *User) UpdateLoginTime() {
	now := time.Now()
	u.LastLoginAt = &now
}
