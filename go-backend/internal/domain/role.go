package domain

import "time"

// Role 角色领域模型
type Role struct {
	ID          int64     `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Status      int8      `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// UserRole 用户角色关联
type UserRole struct {
	ID        int64     `json:"id"`
	UserID    int64     `json:"user_id"`
	RoleID    int64     `json:"role_id"`
	CreatedAt time.Time `json:"created_at"`
}

// RoleStatus 角色状态枚举
type RoleStatus int8

const (
	RoleStatusActive   RoleStatus = 1 // 正常
	RoleStatusInactive RoleStatus = 2 // 禁用
)

// IsActive 检查角色是否激活
func (r *Role) IsActive() bool {
	return r.Status == int8(RoleStatusActive)
}
