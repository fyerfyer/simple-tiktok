package auth

import (
	"context"
	"sync"

	"go-backend/internal/domain"
)

// RBACManager RBAC权限管理器接口
type RBACManager interface {
	// 权限检查
	HasPermission(ctx context.Context, userID int64, resource, action string) bool
	// 角色管理
	AssignRole(userID, roleID int64) error
	RemoveRole(userID, roleID int64) error
	GetUserRoles(userID int64) ([]*domain.Role, error)
	// 权限管理
	GetUserPermissions(userID int64) ([]*domain.Permission, error)
	// 管理员和角色检查
	IsAdmin(userID int64) bool
	IsModerator(userID int64) bool
	// 缓存管理
	ClearUserCache(userID int64)
}

// MemoryRBACManager 内存RBAC管理器
type MemoryRBACManager struct {
	// 用户角色映射
	userRoles map[int64][]int64
	// 角色权限映射
	rolePermissions map[int64][]int64
	// 角色信息
	roles map[int64]*domain.Role
	// 权限信息
	permissions map[int64]*domain.Permission
	// 用户权限缓存
	userPermissionCache map[int64][]*domain.Permission
	mutex               sync.RWMutex
}

// NewMemoryRBACManager 创建内存RBAC管理器
func NewMemoryRBACManager() *MemoryRBACManager {
	manager := &MemoryRBACManager{
		userRoles:           make(map[int64][]int64),
		rolePermissions:     make(map[int64][]int64),
		roles:               make(map[int64]*domain.Role),
		permissions:         make(map[int64]*domain.Permission),
		userPermissionCache: make(map[int64][]*domain.Permission),
	}

	// 初始化基础角色和权限
	manager.initDefaultRolesAndPermissions()

	return manager
}

// initDefaultRolesAndPermissions 初始化默认角色和权限
func (r *MemoryRBACManager) initDefaultRolesAndPermissions() {
	// 创建基础权限
	permissions := []*domain.Permission{
		{ID: 1, Name: "user:read", Resource: "/user", Action: "GET", Status: 1},
		{ID: 2, Name: "user:update", Resource: "/user", Action: "PUT", Status: 1},
		{ID: 3, Name: "video:create", Resource: "/video", Action: "POST", Status: 1},
		{ID: 4, Name: "video:read", Resource: "/video", Action: "GET", Status: 1},
		{ID: 5, Name: "video:delete", Resource: "/video", Action: "DELETE", Status: 1},
		{ID: 6, Name: "comment:create", Resource: "/comment", Action: "POST", Status: 1},
		{ID: 7, Name: "comment:delete", Resource: "/comment", Action: "DELETE", Status: 1},
		{ID: 8, Name: "admin:all", Resource: "/*", Action: "*", Status: 1},
	}

	for _, perm := range permissions {
		r.permissions[perm.ID] = perm
	}

	// 创建基础角色
	roles := []*domain.Role{
		{ID: 1, Name: "user", Description: "Regular user", Status: 1},
		{ID: 2, Name: "admin", Description: "Administrator", Status: 1},
		{ID: 3, Name: "moderator", Description: "Content moderator", Status: 1},
	}

	for _, role := range roles {
		r.roles[role.ID] = role
	}

	// 分配权限给角色
	// 普通用户权限
	r.rolePermissions[1] = []int64{1, 2, 3, 4, 6} // user:read, user:update, video:create, video:read, comment:create
	// 管理员权限
	r.rolePermissions[2] = []int64{8} // admin:all
	// 内容审核员权限
	r.rolePermissions[3] = []int64{1, 4, 5, 7} // user:read, video:read, video:delete, comment:delete
}

// HasPermission 检查用户是否有权限
func (r *MemoryRBACManager) HasPermission(ctx context.Context, userID int64, resource, action string) bool {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// 获取用户权限
	permissions := r.getUserPermissionsFromCache(userID)

	// 检查权限
	for _, perm := range permissions {
		if perm.Match(resource, action) {
			return true
		}
	}

	return false
}

// AssignRole 给用户分配角色
func (r *MemoryRBACManager) AssignRole(userID, roleID int64) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	// 检查角色是否存在
	if _, exists := r.roles[roleID]; !exists {
		return NewAuthError("role not found")
	}

	// 检查是否已经有该角色
	userRoleList := r.userRoles[userID]
	for _, existingRoleID := range userRoleList {
		if existingRoleID == roleID {
			return nil // 已经有该角色
		}
	}

	// 添加角色
	r.userRoles[userID] = append(userRoleList, roleID)

	// 清除缓存
	delete(r.userPermissionCache, userID)

	return nil
}

// RemoveRole 移除用户角色
func (r *MemoryRBACManager) RemoveRole(userID, roleID int64) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	userRoleList := r.userRoles[userID]
	newRoleList := make([]int64, 0)

	for _, existingRoleID := range userRoleList {
		if existingRoleID != roleID {
			newRoleList = append(newRoleList, existingRoleID)
		}
	}

	r.userRoles[userID] = newRoleList

	// 清除缓存
	delete(r.userPermissionCache, userID)

	return nil
}

// GetUserRoles 获取用户角色
func (r *MemoryRBACManager) GetUserRoles(userID int64) ([]*domain.Role, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	userRoleIDs := r.userRoles[userID]
	roles := make([]*domain.Role, 0, len(userRoleIDs))

	for _, roleID := range userRoleIDs {
		if role, exists := r.roles[roleID]; exists && role.IsActive() {
			roles = append(roles, role)
		}
	}

	return roles, nil
}

// GetUserPermissions 获取用户权限
func (r *MemoryRBACManager) GetUserPermissions(userID int64) ([]*domain.Permission, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	return r.getUserPermissionsFromCache(userID), nil
}

// getUserPermissionsFromCache 从缓存获取用户权限
func (r *MemoryRBACManager) getUserPermissionsFromCache(userID int64) []*domain.Permission {
	// 检查缓存
	if cachedPermissions, exists := r.userPermissionCache[userID]; exists {
		return cachedPermissions
	}

	// 计算权限
	permissionMap := make(map[int64]*domain.Permission)
	userRoleIDs := r.userRoles[userID]

	for _, roleID := range userRoleIDs {
		if role, exists := r.roles[roleID]; exists && role.IsActive() {
			permissionIDs := r.rolePermissions[roleID]
			for _, permID := range permissionIDs {
				if perm, exists := r.permissions[permID]; exists && perm.IsActive() {
					permissionMap[permID] = perm
				}
			}
		}
	}

	// 转换为数组
	permissions := make([]*domain.Permission, 0, len(permissionMap))
	for _, perm := range permissionMap {
		permissions = append(permissions, perm)
	}

	// 缓存结果
	r.userPermissionCache[userID] = permissions

	return permissions
}

// ClearUserCache 清除用户权限缓存
func (r *MemoryRBACManager) ClearUserCache(userID int64) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	delete(r.userPermissionCache, userID)
}

// SetDefaultUserRole 为新用户设置默认角色
func (r *MemoryRBACManager) SetDefaultUserRole(userID int64) error {
	return r.AssignRole(userID, 1) // 默认分配普通用户角色
}

// IsAdmin 检查用户是否为管理员
func (r *MemoryRBACManager) IsAdmin(userID int64) bool {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	userRoleIDs := r.userRoles[userID]
	for _, roleID := range userRoleIDs {
		if roleID == 2 { // 管理员角色ID为2
			return true
		}
	}
	return false
}

// AddPermission 添加权限 (管理用)
func (r *MemoryRBACManager) AddPermission(perm *domain.Permission) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.permissions[perm.ID] = perm
}

// AddRole 添加角色 (管理用)
func (r *MemoryRBACManager) AddRole(role *domain.Role) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.roles[role.ID] = role
}

// AssignPermissionToRole 给角色分配权限 (管理用)
func (r *MemoryRBACManager) AssignPermissionToRole(roleID, permissionID int64) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if _, exists := r.roles[roleID]; !exists {
		return NewAuthError("role not found")
	}

	if _, exists := r.permissions[permissionID]; !exists {
		return NewAuthError("permission not found")
	}

	permList := r.rolePermissions[roleID]
	// 检查是否已存在
	for _, existingPermID := range permList {
		if existingPermID == permissionID {
			return nil
		}
	}

	r.rolePermissions[roleID] = append(permList, permissionID)

	// 清除所有用户权限缓存
	r.userPermissionCache = make(map[int64][]*domain.Permission)

	return nil
}

// IsModerator 检查用户是否为审核员
func (r *MemoryRBACManager) IsModerator(userID int64) bool {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	userRoleIDs := r.userRoles[userID]
	for _, roleID := range userRoleIDs {
		if roleID == 3 { // 审核员角色ID为3
			return true
		}
	}
	return false
}
