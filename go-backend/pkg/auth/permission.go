package auth

import "context"

// PermissionChecker 权限检查器接口
type PermissionChecker interface {
	CheckPermission(ctx context.Context, userID int64, resource, action string) (bool, error)
	IsAdmin(ctx context.Context, userID int64) (bool, error)
	IsModerator(ctx context.Context, userID int64) (bool, error)
	CanModerateContent(ctx context.Context, userID int64) (bool, error)
}

// SimplePermissionChecker 简单权限检查器（基于内存RBAC）
type SimplePermissionChecker struct {
	rbacManager RBACManager
}

// NewSimplePermissionChecker 创建简单权限检查器
func NewSimplePermissionChecker(rbacManager RBACManager) *SimplePermissionChecker {
	return &SimplePermissionChecker{
		rbacManager: rbacManager,
	}
}

func (c *SimplePermissionChecker) CheckPermission(ctx context.Context, userID int64, resource, action string) (bool, error) {
	return c.rbacManager.HasPermission(ctx, userID, resource, action), nil
}

func (c *SimplePermissionChecker) IsAdmin(ctx context.Context, userID int64) (bool, error) {
	return c.rbacManager.IsAdmin(userID), nil
}

func (c *SimplePermissionChecker) IsModerator(ctx context.Context, userID int64) (bool, error) {
	// 检查是否有审核员角色（角色ID为3）
	roles, _ := c.rbacManager.GetUserRoles(userID)
	for _, role := range roles {
		if role.ID == 3 { // moderator role
			return true, nil
		}
	}
	return false, nil
}

func (c *SimplePermissionChecker) CanModerateContent(ctx context.Context, userID int64) (bool, error) {
	// 管理员或审核员都可以审核内容
	isAdmin, _ := c.IsAdmin(ctx, userID)
	if isAdmin {
		return true, nil
	}
	return c.IsModerator(ctx, userID)
}
