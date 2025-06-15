package biz

import (
	"context"

	"go-backend/internal/domain"
	"go-backend/pkg/auth"

	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/log"
)

// 权限相关错误
var (
	ErrPermissionDenied = errors.Forbidden("PERMISSION_DENIED", "permission denied")
	ErrRoleNotFound     = errors.NotFound("ROLE_NOT_FOUND", "role not found")
	ErrInvalidRole      = errors.BadRequest("INVALID_ROLE", "invalid role")
)

// RoleRepo 角色仓储接口
type RoleRepo interface {
	GetRole(ctx context.Context, roleID int64) (*domain.Role, error)
	GetRoleByName(ctx context.Context, name string) (*domain.Role, error)
	GetUserRoles(ctx context.Context, userID int64) ([]*domain.Role, error)
	AssignRole(ctx context.Context, userID, roleID int64) error
	RemoveRole(ctx context.Context, userID, roleID int64) error
	HasRole(ctx context.Context, userID, roleID int64) (bool, error)
}

// PermissionRepo 权限仓储接口
type PermissionRepo interface {
	GetPermission(ctx context.Context, permissionID int64) (*domain.Permission, error)
	GetRolePermissions(ctx context.Context, roleID int64) ([]*domain.Permission, error)
	GetUserPermissions(ctx context.Context, userID int64) ([]*domain.Permission, error)
	HasPermission(ctx context.Context, userID int64, resource, action string) (bool, error)
}

// PermissionUsecase 权限用例
type PermissionUsecase struct {
	roleRepo       RoleRepo
	permissionRepo PermissionRepo
	rbacManager    auth.RBACManager
	log            *log.Helper
}

// NewPermissionUsecase 创建权限用例
func NewPermissionUsecase(
	roleRepo RoleRepo,
	permissionRepo PermissionRepo,
	rbacManager auth.RBACManager,
	logger log.Logger,
) *PermissionUsecase {
	return &PermissionUsecase{
		roleRepo:       roleRepo,
		permissionRepo: permissionRepo,
		rbacManager:    rbacManager,
		log:            log.NewHelper(logger),
	}
}

// CheckPermission 检查用户权限
func (uc *PermissionUsecase) CheckPermission(ctx context.Context, userID int64, resource, action string) (bool, error) {
	// 先从内存RBAC管理器检查
	if uc.rbacManager.HasPermission(ctx, userID, resource, action) {
		return true, nil
	}

	// 再从数据库检查
	return uc.permissionRepo.HasPermission(ctx, userID, resource, action)
}

// GetUserRoles 获取用户角色
func (uc *PermissionUsecase) GetUserRoles(ctx context.Context, userID int64) ([]*domain.Role, error) {
	// 优先从数据库获取
	roles, err := uc.roleRepo.GetUserRoles(ctx, userID)
	if err != nil {
		return nil, err
	}

	// 同步到内存管理器
	for _, role := range roles {
		uc.rbacManager.AssignRole(userID, role.ID)
	}

	return roles, nil
}

// GetUserPermissions 获取用户权限
func (uc *PermissionUsecase) GetUserPermissions(ctx context.Context, userID int64) ([]*domain.Permission, error) {
	return uc.permissionRepo.GetUserPermissions(ctx, userID)
}

// AssignRole 分配角色
func (uc *PermissionUsecase) AssignRole(ctx context.Context, userID, roleID int64) error {
	uc.log.WithContext(ctx).Infof("Assign role %d to user %d", roleID, userID)

	// 数据库操作
	if err := uc.roleRepo.AssignRole(ctx, userID, roleID); err != nil {
		return err
	}

	// 同步到内存管理器
	uc.rbacManager.AssignRole(userID, roleID)

	return nil
}

// RemoveRole 移除角色
func (uc *PermissionUsecase) RemoveRole(ctx context.Context, userID, roleID int64) error {
	uc.log.WithContext(ctx).Infof("Remove role %d from user %d", roleID, userID)

	// 数据库操作
	if err := uc.roleRepo.RemoveRole(ctx, userID, roleID); err != nil {
		return err
	}

	// 同步到内存管理器
	uc.rbacManager.RemoveRole(userID, roleID)

	return nil
}

// HasRole 检查用户是否有指定角色
func (uc *PermissionUsecase) HasRole(ctx context.Context, userID, roleID int64) (bool, error) {
	return uc.roleRepo.HasRole(ctx, userID, roleID)
}

// InitUserDefaultRole 为新用户初始化默认角色
func (uc *PermissionUsecase) InitUserDefaultRole(ctx context.Context, userID int64) error {
	uc.log.WithContext(ctx).Infof("Init default role for user: %d", userID)

	// 获取默认用户角色
	defaultRole, err := uc.roleRepo.GetRoleByName(ctx, "user")
	if err != nil {
		return err
	}

	// 分配默认角色
	return uc.AssignRole(ctx, userID, defaultRole.ID)
}

// IsAdmin 检查用户是否为管理员
func (uc *PermissionUsecase) IsAdmin(ctx context.Context, userID int64) (bool, error) {
	// 检查是否有管理员角色
	adminRole, err := uc.roleRepo.GetRoleByName(ctx, "admin")
	if err != nil {
		return false, err
	}

	return uc.HasRole(ctx, userID, adminRole.ID)
}

// IsModerator 检查用户是否为内容审核员
func (uc *PermissionUsecase) IsModerator(ctx context.Context, userID int64) (bool, error) {
	// 检查是否有审核员角色
	modRole, err := uc.roleRepo.GetRoleByName(ctx, "moderator")
	if err != nil {
		return false, err
	}

	return uc.HasRole(ctx, userID, modRole.ID)
}

// ClearUserPermissionCache 清除用户权限缓存
func (uc *PermissionUsecase) ClearUserPermissionCache(ctx context.Context, userID int64) {
	uc.log.WithContext(ctx).Infof("Clear permission cache for user: %d", userID)
	uc.rbacManager.ClearUserCache(userID)
}

// CheckVideoPermission 检查视频相关权限
func (uc *PermissionUsecase) CheckVideoPermission(ctx context.Context, userID int64, action string) (bool, error) {
	resource := "/video"
	return uc.CheckPermission(ctx, userID, resource, action)
}

// CheckCommentPermission 检查评论相关权限
func (uc *PermissionUsecase) CheckCommentPermission(ctx context.Context, userID int64, action string) (bool, error) {
	resource := "/comment"
	return uc.CheckPermission(ctx, userID, resource, action)
}

// CheckUserPermission 检查用户信息相关权限
func (uc *PermissionUsecase) CheckUserPermission(ctx context.Context, userID int64, action string) (bool, error) {
	resource := "/user"
	return uc.CheckPermission(ctx, userID, resource, action)
}

// ValidateResourceAccess 验证资源访问权限
func (uc *PermissionUsecase) ValidateResourceAccess(ctx context.Context, userID int64, resource, action string) error {
	hasPermission, err := uc.CheckPermission(ctx, userID, resource, action)
	if err != nil {
		return err
	}

	if !hasPermission {
		return ErrPermissionDenied
	}

	return nil
}
