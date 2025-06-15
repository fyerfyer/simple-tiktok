package service

import (
	"context"

	"go-backend/internal/biz"
	"go-backend/internal/domain"

	"github.com/go-kratos/kratos/v2/log"
)

// PermissionService 权限服务
type PermissionService struct {
	permissionUc *biz.PermissionUsecase
	log          *log.Helper
}

// NewPermissionService 创建权限服务
func NewPermissionService(
	permissionUc *biz.PermissionUsecase,
	logger log.Logger,
) *PermissionService {
	return &PermissionService{
		permissionUc: permissionUc,
		log:          log.NewHelper(logger),
	}
}

// CheckPermission 检查用户权限
func (s *PermissionService) CheckPermission(ctx context.Context, userID int64, resource, action string) (bool, error) {
	return s.permissionUc.CheckPermission(ctx, userID, resource, action)
}

// GetUserRoles 获取用户角色
func (s *PermissionService) GetUserRoles(ctx context.Context, userID int64) ([]*domain.Role, error) {
	return s.permissionUc.GetUserRoles(ctx, userID)
}

// GetUserPermissions 获取用户权限
func (s *PermissionService) GetUserPermissions(ctx context.Context, userID int64) ([]*domain.Permission, error) {
	return s.permissionUc.GetUserPermissions(ctx, userID)
}

// AssignRole 分配角色
func (s *PermissionService) AssignRole(ctx context.Context, userID, roleID int64) error {
	s.log.WithContext(ctx).Infof("Assign role %d to user %d", roleID, userID)

	err := s.permissionUc.AssignRole(ctx, userID, roleID)
	if err != nil {
		s.log.WithContext(ctx).Errorf("assign role failed: %v", err)
		return err
	}

	return nil
}

// RemoveRole 移除角色
func (s *PermissionService) RemoveRole(ctx context.Context, userID, roleID int64) error {
	s.log.WithContext(ctx).Infof("Remove role %d from user %d", roleID, userID)

	err := s.permissionUc.RemoveRole(ctx, userID, roleID)
	if err != nil {
		s.log.WithContext(ctx).Errorf("remove role failed: %v", err)
		return err
	}

	return nil
}

// HasRole 检查用户是否有指定角色
func (s *PermissionService) HasRole(ctx context.Context, userID, roleID int64) (bool, error) {
	return s.permissionUc.HasRole(ctx, userID, roleID)
}

// IsAdmin 检查用户是否为管理员
func (s *PermissionService) IsAdmin(ctx context.Context, userID int64) (bool, error) {
	return s.permissionUc.IsAdmin(ctx, userID)
}

// IsModerator 检查用户是否为内容审核员
func (s *PermissionService) IsModerator(ctx context.Context, userID int64) (bool, error) {
	return s.permissionUc.IsModerator(ctx, userID)
}

// ValidateResourceAccess 验证资源访问权限
func (s *PermissionService) ValidateResourceAccess(ctx context.Context, userID int64, resource, action string) error {
	s.log.WithContext(ctx).Infof("Validate access: user=%d, resource=%s, action=%s", userID, resource, action)

	err := s.permissionUc.ValidateResourceAccess(ctx, userID, resource, action)
	if err != nil {
		s.log.WithContext(ctx).Warnf("access denied: user=%d, resource=%s, action=%s", userID, resource, action)
		return err
	}

	return nil
}

// CheckVideoPermission 检查视频相关权限
func (s *PermissionService) CheckVideoPermission(ctx context.Context, userID int64, action string) (bool, error) {
	return s.permissionUc.CheckVideoPermission(ctx, userID, action)
}

// CheckCommentPermission 检查评论相关权限
func (s *PermissionService) CheckCommentPermission(ctx context.Context, userID int64, action string) (bool, error) {
	return s.permissionUc.CheckCommentPermission(ctx, userID, action)
}

// CheckUserPermission 检查用户信息相关权限
func (s *PermissionService) CheckUserPermission(ctx context.Context, userID int64, action string) (bool, error) {
	return s.permissionUc.CheckUserPermission(ctx, userID, action)
}

// ClearUserPermissionCache 清除用户权限缓存
func (s *PermissionService) ClearUserPermissionCache(ctx context.Context, userID int64) {
	s.log.WithContext(ctx).Infof("Clear permission cache for user: %d", userID)
	s.permissionUc.ClearUserPermissionCache(ctx, userID)
}

// InitUserRole 为新用户初始化角色
func (s *PermissionService) InitUserRole(ctx context.Context, userID int64) error {
	s.log.WithContext(ctx).Infof("Init role for new user: %d", userID)

	err := s.permissionUc.InitUserDefaultRole(ctx, userID)
	if err != nil {
		s.log.WithContext(ctx).Errorf("init user role failed: %v", err)
		return err
	}

	return nil
}

// CanAccessVideo 检查是否可以访问视频
func (s *PermissionService) CanAccessVideo(ctx context.Context, userID int64, videoID int64) (bool, error) {
	// 简化实现：普通用户可以查看视频，管理员可以删除视频
	canView, err := s.CheckVideoPermission(ctx, userID, "GET")
	if err != nil {
		return false, err
	}

	return canView, nil
}

// CanModerateContent 检查是否可以审核内容
func (s *PermissionService) CanModerateContent(ctx context.Context, userID int64) (bool, error) {
	// 检查是否为管理员或审核员
	isAdmin, err := s.IsAdmin(ctx, userID)
	if err != nil {
		return false, err
	}

	if isAdmin {
		return true, nil
	}

	return s.IsModerator(ctx, userID)
}

// CanDeleteComment 检查是否可以删除评论
func (s *PermissionService) CanDeleteComment(ctx context.Context, userID int64, commentUserID int64) (bool, error) {
	// 用户可以删除自己的评论
	if userID == commentUserID {
		return true, nil
	}

	// 管理员和审核员可以删除任何评论
	return s.CanModerateContent(ctx, userID)
}

// CanManageUser 检查是否可以管理用户
func (s *PermissionService) CanManageUser(ctx context.Context, userID int64) (bool, error) {
	return s.IsAdmin(ctx, userID)
}
