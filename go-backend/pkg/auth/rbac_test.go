package auth

import (
	"context"
	"testing"

	"go-backend/internal/domain"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMemoryRBACManager(t *testing.T) {
	manager := NewMemoryRBACManager()
	ctx := context.Background()

	userID := int64(12345)
	adminUserID := int64(54321)

	t.Run("InitialState", func(t *testing.T) {
		// 检查默认角色和权限是否正确初始化
		roles, err := manager.GetUserRoles(userID)
		require.NoError(t, err)
		assert.Empty(t, roles)

		permissions, err := manager.GetUserPermissions(userID)
		require.NoError(t, err)
		assert.Empty(t, permissions)
	})

	t.Run("AssignRole", func(t *testing.T) {
		// 分配普通用户角色
		err := manager.AssignRole(userID, 1) // user role
		require.NoError(t, err)

		roles, err := manager.GetUserRoles(userID)
		require.NoError(t, err)
		assert.Len(t, roles, 1)
		assert.Equal(t, "user", roles[0].Name)
	})

	t.Run("HasPermission_UserRole", func(t *testing.T) {
		// 用户应该有基本权限
		hasPermission := manager.HasPermission(ctx, userID, "/user", "GET")
		assert.True(t, hasPermission)

		// 用户不应该有管理员权限
		hasPermission = manager.HasPermission(ctx, userID, "/*", "*")
		assert.False(t, hasPermission)
	})

	t.Run("AssignAdminRole", func(t *testing.T) {
		// 分配管理员角色
		err := manager.AssignRole(adminUserID, 2) // admin role
		require.NoError(t, err)

		roles, err := manager.GetUserRoles(adminUserID)
		require.NoError(t, err)
		assert.Len(t, roles, 1)
		assert.Equal(t, "admin", roles[0].Name)
	})

	t.Run("HasPermission_AdminRole", func(t *testing.T) {
		// 管理员应该有所有权限
		hasPermission := manager.HasPermission(ctx, adminUserID, "/user", "GET")
		assert.True(t, hasPermission)

		hasPermission = manager.HasPermission(ctx, adminUserID, "/video", "DELETE")
		assert.True(t, hasPermission)

		hasPermission = manager.HasPermission(ctx, adminUserID, "/*", "*")
		assert.True(t, hasPermission)
	})

	t.Run("IsAdmin", func(t *testing.T) {
		assert.False(t, manager.IsAdmin(userID))
		assert.True(t, manager.IsAdmin(adminUserID))
	})

	t.Run("IsModerator", func(t *testing.T) {
		moderatorUserID := int64(99999)

		err := manager.AssignRole(moderatorUserID, 3) // moderator role
		require.NoError(t, err)

		assert.True(t, manager.IsModerator(moderatorUserID))
		assert.False(t, manager.IsModerator(userID))
		assert.False(t, manager.IsModerator(adminUserID))
	})

	t.Run("RemoveRole", func(t *testing.T) {
		err := manager.RemoveRole(userID, 1) // remove user role
		require.NoError(t, err)

		roles, err := manager.GetUserRoles(userID)
		require.NoError(t, err)
		assert.Empty(t, roles)

		// 权限也应该被清除
		hasPermission := manager.HasPermission(ctx, userID, "/user", "GET")
		assert.False(t, hasPermission)
	})

	t.Run("MultipleRoles", func(t *testing.T) {
		multiRoleUserID := int64(77777)

		// 分配多个角色
		err := manager.AssignRole(multiRoleUserID, 1) // user
		require.NoError(t, err)

		err = manager.AssignRole(multiRoleUserID, 3) // moderator
		require.NoError(t, err)

		roles, err := manager.GetUserRoles(multiRoleUserID)
		require.NoError(t, err)
		assert.Len(t, roles, 2)

		// 应该有两个角色的权限
		hasUserPermission := manager.HasPermission(ctx, multiRoleUserID, "/user", "GET")
		assert.True(t, hasUserPermission)

		hasModeratorPermission := manager.HasPermission(ctx, multiRoleUserID, "/video", "DELETE")
		assert.True(t, hasModeratorPermission)
	})

	t.Run("ClearUserCache", func(t *testing.T) {
		// 先获取权限，确保缓存被建立
		permissions, err := manager.GetUserPermissions(adminUserID)
		require.NoError(t, err)
		assert.NotEmpty(t, permissions)

		// 清除缓存
		manager.ClearUserCache(adminUserID)

		// 再次获取权限应该重新计算
		permissions, err = manager.GetUserPermissions(adminUserID)
		require.NoError(t, err)
		assert.NotEmpty(t, permissions)
	})

	t.Run("AssignRole_NonExistentRole", func(t *testing.T) {
		err := manager.AssignRole(userID, 999) // non-existent role
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "role not found")
	})

	t.Run("AssignRole_Duplicate", func(t *testing.T) {
		err := manager.AssignRole(userID, 1) // user role
		require.NoError(t, err)

		// 重复分配相同角色应该成功（不报错）
		err = manager.AssignRole(userID, 1)
		assert.NoError(t, err)

		roles, err := manager.GetUserRoles(userID)
		require.NoError(t, err)
		assert.Len(t, roles, 1) // 应该只有一个角色
	})

	t.Run("SetDefaultUserRole", func(t *testing.T) {
		newUserID := int64(88888)

		err := manager.SetDefaultUserRole(newUserID)
		require.NoError(t, err)

		roles, err := manager.GetUserRoles(newUserID)
		require.NoError(t, err)
		assert.Len(t, roles, 1)
		assert.Equal(t, "user", roles[0].Name)
	})
}

func TestSimplePermissionChecker(t *testing.T) {
	manager := NewMemoryRBACManager()
	checker := NewSimplePermissionChecker(manager)
	ctx := context.Background()

	userID := int64(12345)
	adminUserID := int64(54321)

	// 设置角色
	manager.AssignRole(userID, 1)      // user
	manager.AssignRole(adminUserID, 2) // admin

	t.Run("CheckPermission", func(t *testing.T) {
		hasPermission, err := checker.CheckPermission(ctx, userID, "/user", "GET")
		require.NoError(t, err)
		assert.True(t, hasPermission)

		hasPermission, err = checker.CheckPermission(ctx, userID, "/admin", "POST")
		require.NoError(t, err)
		assert.False(t, hasPermission)
	})

	t.Run("IsAdmin", func(t *testing.T) {
		isAdmin, err := checker.IsAdmin(ctx, userID)
		require.NoError(t, err)
		assert.False(t, isAdmin)

		isAdmin, err = checker.IsAdmin(ctx, adminUserID)
		require.NoError(t, err)
		assert.True(t, isAdmin)
	})

	t.Run("IsModerator", func(t *testing.T) {
		moderatorUserID := int64(99999)
		manager.AssignRole(moderatorUserID, 3) // moderator

		isModerator, err := checker.IsModerator(ctx, moderatorUserID)
		require.NoError(t, err)
		assert.True(t, isModerator)

		isModerator, err = checker.IsModerator(ctx, userID)
		require.NoError(t, err)
		assert.False(t, isModerator)
	})

	t.Run("CanModerateContent", func(t *testing.T) {
		moderatorUserID := int64(99999)
		manager.AssignRole(moderatorUserID, 3) // moderator

		// 管理员可以审核内容
		canModerate, err := checker.CanModerateContent(ctx, adminUserID)
		require.NoError(t, err)
		assert.True(t, canModerate)

		// 审核员可以审核内容
		canModerate, err = checker.CanModerateContent(ctx, moderatorUserID)
		require.NoError(t, err)
		assert.True(t, canModerate)

		// 普通用户不能审核内容
		canModerate, err = checker.CanModerateContent(ctx, userID)
		require.NoError(t, err)
		assert.False(t, canModerate)
	})
}

func TestPermissionMatching(t *testing.T) {
	t.Run("PermissionMatch", func(t *testing.T) {
		// 测试权限匹配逻辑
		perm := &domain.Permission{
			Resource: "/user",
			Action:   "GET",
			Status:   int8(domain.PermissionStatusActive), // 设置为激活状态
		}

		assert.True(t, perm.Match("/user", "GET"))
		assert.False(t, perm.Match("/user", "POST"))
		assert.False(t, perm.Match("/video", "GET"))

		// 测试通配符权限
		adminPerm := &domain.Permission{
			Resource: "/*",
			Action:   "*",
			Status:   int8(domain.PermissionStatusActive), // 设置为激活状态
		}

		assert.True(t, adminPerm.Match("/user", "GET"))
		assert.True(t, adminPerm.Match("/video", "DELETE"))
		assert.True(t, adminPerm.Match("/anything", "POST"))
	})
}
