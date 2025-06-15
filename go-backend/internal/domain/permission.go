package domain

import "time"

// Permission 权限领域模型
type Permission struct {
	ID          int64     `json:"id"`
	Name        string    `json:"name"`
	Resource    string    `json:"resource"`
	Action      string    `json:"action"`
	Description string    `json:"description"`
	Status      int8      `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// RolePermission 角色权限关联
type RolePermission struct {
	ID           int64     `json:"id"`
	RoleID       int64     `json:"role_id"`
	PermissionID int64     `json:"permission_id"`
	CreatedAt    time.Time `json:"created_at"`
}

// PermissionStatus 权限状态枚举
type PermissionStatus int8

const (
	PermissionStatusActive   PermissionStatus = 1 // 正常
	PermissionStatusInactive PermissionStatus = 2 // 禁用
)

// ActionType 操作类型常量
const (
	ActionGet    = "GET"
	ActionPost   = "POST"
	ActionPut    = "PUT"
	ActionDelete = "DELETE"
	ActionAll    = "*"
)

// IsActive 检查权限是否激活
func (p *Permission) IsActive() bool {
	return p.Status == int8(PermissionStatusActive)
}

// Match 检查权限是否匹配资源和操作
func (p *Permission) Match(resource, action string) bool {
	if !p.IsActive() {
		return false
	}

	resourceMatch := p.Resource == resource || p.Resource == "/*"
	actionMatch := p.Action == action || p.Action == ActionAll

	return resourceMatch && actionMatch
}
