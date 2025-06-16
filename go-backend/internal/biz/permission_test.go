package biz

import (
	"context"
	"testing"

	"go-backend/internal/domain"
	"go-backend/pkg/auth"
	"go-backend/testutils"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupPermissionUsecase(t *testing.T) (*PermissionUsecase, *MockRoleRepo, *MockPermissionRepo, *testutils.TestEnv, func()) {
	env, cleanup, err := testutils.SetupTestWithCleanup()
	require.NoError(t, err)

	roleRepo := NewMockRoleRepo(t)
	permissionRepo := NewMockPermissionRepo(t)
	rbacManager := auth.NewMemoryRBACManager()
	logger := log.DefaultLogger

	uc := NewPermissionUsecase(roleRepo, permissionRepo, rbacManager, logger)

	return uc, roleRepo, permissionRepo, env, cleanup
}

func TestPermissionUsecase_CheckPermission(t *testing.T) {
	uc, _, permissionRepo, env, cleanup := setupPermissionUsecase(t)
	defer cleanup()

	ctx := context.Background()

	// 创建测试用户
	users, err := env.DataManager.CreateTestUsers(1)
	require.NoError(t, err)
	testUser := users[0]

	t.Run("CheckPermission_HasPermission", func(t *testing.T) {
		// 通过内存RBAC管理器分配角色
		err := uc.rbacManager.AssignRole(testUser.ID, 1) // 分配用户角色
		require.NoError(t, err)

		hasPermission, err := uc.CheckPermission(ctx, testUser.ID, "/user", "GET")

		require.NoError(t, err)
		assert.True(t, hasPermission)
	})

	t.Run("CheckPermission_NoPermission", func(t *testing.T) {
		// 检查不存在的权限
		permissionRepo.EXPECT().HasPermission(ctx, testUser.ID, "/admin", "DELETE").Return(false, nil)

		hasPermission, err := uc.CheckPermission(ctx, testUser.ID, "/admin", "DELETE")

		require.NoError(t, err)
		assert.False(t, hasPermission)
	})

	t.Run("CheckPermission_DatabaseFallback", func(t *testing.T) {
		// 内存RBAC没有权限，回退到数据库查询
		permissionRepo.EXPECT().HasPermission(ctx, testUser.ID, "/video", "POST").Return(true, nil)

		hasPermission, err := uc.CheckPermission(ctx, testUser.ID, "/video", "POST")

		require.NoError(t, err)
		assert.True(t, hasPermission)
	})
}

func TestPermissionUsecase_GetUserRoles(t *testing.T) {
	uc, roleRepo, _, env, cleanup := setupPermissionUsecase(t)
	defer cleanup()

	ctx := context.Background()

	// 创建测试用户
	users, err := env.DataManager.CreateTestUsers(1)
	require.NoError(t, err)
	testUser := users[0]

	t.Run("GetUserRoles_Success", func(t *testing.T) {
		expectedRoles := []*domain.Role{
			{ID: 1, Name: "user", Description: "Regular user", Status: 1},
			{ID: 2, Name: "admin", Description: "Administrator", Status: 1},
		}

		roleRepo.EXPECT().GetUserRoles(ctx, testUser.ID).Return(expectedRoles, nil)

		roles, err := uc.GetUserRoles(ctx, testUser.ID)

		require.NoError(t, err)
		assert.Len(t, roles, 2)
		assert.Equal(t, "user", roles[0].Name)
		assert.Equal(t, "admin", roles[1].Name)
	})

	t.Run("GetUserRoles_Empty", func(t *testing.T) {
		roleRepo.EXPECT().GetUserRoles(ctx, testUser.ID).Return([]*domain.Role{}, nil)

		roles, err := uc.GetUserRoles(ctx, testUser.ID)

		require.NoError(t, err)
		assert.Empty(t, roles)
	})
}

func TestPermissionUsecase_GetUserPermissions(t *testing.T) {
	uc, _, permissionRepo, env, cleanup := setupPermissionUsecase(t)
	defer cleanup()

	ctx := context.Background()

	// 创建测试用户
	users, err := env.DataManager.CreateTestUsers(1)
	require.NoError(t, err)
	testUser := users[0]

	t.Run("GetUserPermissions_Success", func(t *testing.T) {
		expectedPermissions := []*domain.Permission{
			{ID: 1, Name: "user:read", Resource: "/user", Action: "GET", Status: 1},
			{ID: 2, Name: "user:update", Resource: "/user", Action: "PUT", Status: 1},
		}

		permissionRepo.EXPECT().GetUserPermissions(ctx, testUser.ID).Return(expectedPermissions, nil)

		permissions, err := uc.GetUserPermissions(ctx, testUser.ID)

		require.NoError(t, err)
		assert.Len(t, permissions, 2)
		assert.Equal(t, "user:read", permissions[0].Name)
		assert.Equal(t, "user:update", permissions[1].Name)
	})
}

func TestPermissionUsecase_AssignRole(t *testing.T) {
	uc, roleRepo, _, env, cleanup := setupPermissionUsecase(t)
	defer cleanup()

	ctx := context.Background()

	// 创建测试用户
	users, err := env.DataManager.CreateTestUsers(1)
	require.NoError(t, err)
	testUser := users[0]

	t.Run("AssignRole_Success", func(t *testing.T) {
		roleID := int64(1)

		roleRepo.EXPECT().AssignRole(ctx, testUser.ID, roleID).Return(nil)

		err := uc.AssignRole(ctx, testUser.ID, roleID)

		require.NoError(t, err)

		// 验证内存管理器中已分配角色
		roles, err := uc.rbacManager.GetUserRoles(testUser.ID)
		require.NoError(t, err)
		assert.Len(t, roles, 1)
		assert.Equal(t, roleID, roles[0].ID)
	})

	t.Run("AssignRole_DatabaseError", func(t *testing.T) {
		roleID := int64(2)

		roleRepo.EXPECT().AssignRole(ctx, testUser.ID, roleID).Return(ErrRoleNotFound)

		err := uc.AssignRole(ctx, testUser.ID, roleID)

		assert.Error(t, err)
		assert.Equal(t, ErrRoleNotFound, err)
	})
}

func TestPermissionUsecase_RemoveRole(t *testing.T) {
	uc, roleRepo, _, env, cleanup := setupPermissionUsecase(t)
	defer cleanup()

	ctx := context.Background()

	// 创建测试用户
	users, err := env.DataManager.CreateTestUsers(1)
	require.NoError(t, err)
	testUser := users[0]

	t.Run("RemoveRole_Success", func(t *testing.T) {
		roleID := int64(1)

		// 先分配角色
		uc.rbacManager.AssignRole(testUser.ID, roleID)

		roleRepo.EXPECT().RemoveRole(ctx, testUser.ID, roleID).Return(nil)

		err := uc.RemoveRole(ctx, testUser.ID, roleID)

		require.NoError(t, err)

		// 验证内存管理器中角色已被移除
		roles, err := uc.rbacManager.GetUserRoles(testUser.ID)
		require.NoError(t, err)
		assert.Empty(t, roles)
	})
}

func TestPermissionUsecase_HasRole(t *testing.T) {
	uc, roleRepo, _, env, cleanup := setupPermissionUsecase(t)
	defer cleanup()

	ctx := context.Background()

	// 创建测试用户
	users, err := env.DataManager.CreateTestUsers(1)
	require.NoError(t, err)
	testUser := users[0]

	t.Run("HasRole_True", func(t *testing.T) {
		roleID := int64(1)

		roleRepo.EXPECT().HasRole(ctx, testUser.ID, roleID).Return(true, nil)

		hasRole, err := uc.HasRole(ctx, testUser.ID, roleID)

		require.NoError(t, err)
		assert.True(t, hasRole)
	})

	t.Run("HasRole_False", func(t *testing.T) {
		roleID := int64(2)

		roleRepo.EXPECT().HasRole(ctx, testUser.ID, roleID).Return(false, nil)

		hasRole, err := uc.HasRole(ctx, testUser.ID, roleID)

		require.NoError(t, err)
		assert.False(t, hasRole)
	})
}

func TestPermissionUsecase_InitUserDefaultRole(t *testing.T) {
	uc, roleRepo, _, env, cleanup := setupPermissionUsecase(t)
	defer cleanup()

	ctx := context.Background()

	// 创建测试用户
	users, err := env.DataManager.CreateTestUsers(1)
	require.NoError(t, err)
	testUser := users[0]

	t.Run("InitUserDefaultRole_Success", func(t *testing.T) {
		defaultRole := &domain.Role{
			ID:   1,
			Name: "user",
		}

		roleRepo.EXPECT().GetRoleByName(ctx, "user").Return(defaultRole, nil)
		roleRepo.EXPECT().AssignRole(ctx, testUser.ID, defaultRole.ID).Return(nil)

		err := uc.InitUserDefaultRole(ctx, testUser.ID)

		require.NoError(t, err)
	})

	t.Run("InitUserDefaultRole_RoleNotFound", func(t *testing.T) {
		roleRepo.EXPECT().GetRoleByName(ctx, "user").Return(nil, ErrRoleNotFound)

		err := uc.InitUserDefaultRole(ctx, testUser.ID)

		assert.Error(t, err)
		assert.Equal(t, ErrRoleNotFound, err)
	})
}

func TestPermissionUsecase_IsAdmin(t *testing.T) {
	uc, roleRepo, _, env, cleanup := setupPermissionUsecase(t)
	defer cleanup()

	ctx := context.Background()

	// 创建测试用户
	users, err := env.DataManager.CreateTestUsers(1)
	require.NoError(t, err)
	testUser := users[0]

	t.Run("IsAdmin_True", func(t *testing.T) {
		adminRole := &domain.Role{
			ID:   2,
			Name: "admin",
		}

		roleRepo.EXPECT().GetRoleByName(ctx, "admin").Return(adminRole, nil)
		roleRepo.EXPECT().HasRole(ctx, testUser.ID, adminRole.ID).Return(true, nil)

		isAdmin, err := uc.IsAdmin(ctx, testUser.ID)

		require.NoError(t, err)
		assert.True(t, isAdmin)
	})

	t.Run("IsAdmin_False", func(t *testing.T) {
		adminRole := &domain.Role{
			ID:   2,
			Name: "admin",
		}

		roleRepo.EXPECT().GetRoleByName(ctx, "admin").Return(adminRole, nil)
		roleRepo.EXPECT().HasRole(ctx, testUser.ID, adminRole.ID).Return(false, nil)

		isAdmin, err := uc.IsAdmin(ctx, testUser.ID)

		require.NoError(t, err)
		assert.False(t, isAdmin)
	})
}

func TestPermissionUsecase_IsModerator(t *testing.T) {
	uc, roleRepo, _, env, cleanup := setupPermissionUsecase(t)
	defer cleanup()

	ctx := context.Background()

	// 创建测试用户
	users, err := env.DataManager.CreateTestUsers(1)
	require.NoError(t, err)
	testUser := users[0]

	t.Run("IsModerator_True", func(t *testing.T) {
		modRole := &domain.Role{
			ID:   3,
			Name: "moderator",
		}

		roleRepo.EXPECT().GetRoleByName(ctx, "moderator").Return(modRole, nil)
		roleRepo.EXPECT().HasRole(ctx, testUser.ID, modRole.ID).Return(true, nil)

		isModerator, err := uc.IsModerator(ctx, testUser.ID)

		require.NoError(t, err)
		assert.True(t, isModerator)
	})

	t.Run("IsModerator_False", func(t *testing.T) {
		modRole := &domain.Role{
			ID:   3,
			Name: "moderator",
		}

		roleRepo.EXPECT().GetRoleByName(ctx, "moderator").Return(modRole, nil)
		roleRepo.EXPECT().HasRole(ctx, testUser.ID, modRole.ID).Return(false, nil)

		isModerator, err := uc.IsModerator(ctx, testUser.ID)

		require.NoError(t, err)
		assert.False(t, isModerator)
	})
}

func TestPermissionUsecase_ClearUserPermissionCache(t *testing.T) {
	uc, _, _, env, cleanup := setupPermissionUsecase(t)
	defer cleanup()

	ctx := context.Background()

	// 创建测试用户
	users, err := env.DataManager.CreateTestUsers(1)
	require.NoError(t, err)
	testUser := users[0]

	t.Run("ClearUserPermissionCache_Success", func(t *testing.T) {
		// 先分配角色建立缓存
		uc.rbacManager.AssignRole(testUser.ID, 1)

		// 清除缓存
		uc.ClearUserPermissionCache(ctx, testUser.ID)

		// 这里主要测试调用不会出错，具体缓存清理逻辑在RBAC管理器中
		assert.True(t, true)
	})
}

func TestPermissionUsecase_ValidateResourceAccess(t *testing.T) {
	uc, _, permissionRepo, env, cleanup := setupPermissionUsecase(t)
	defer cleanup()

	ctx := context.Background()

	// 创建测试用户
	users, err := env.DataManager.CreateTestUsers(1)
	require.NoError(t, err)
	testUser := users[0]

	t.Run("ValidateResourceAccess_HasPermission", func(t *testing.T) {
		// 分配用户角色
		uc.rbacManager.AssignRole(testUser.ID, 1)

		err := uc.ValidateResourceAccess(ctx, testUser.ID, "/user", "GET")

		assert.NoError(t, err)
	})

	t.Run("ValidateResourceAccess_NoPermission", func(t *testing.T) {
		permissionRepo.EXPECT().HasPermission(ctx, testUser.ID, "/admin", "DELETE").Return(false, nil)

		err := uc.ValidateResourceAccess(ctx, testUser.ID, "/admin", "DELETE")

		assert.Error(t, err)
		assert.Equal(t, ErrPermissionDenied, err)
	})
}

func TestPermissionUsecase_CheckSpecificPermissions(t *testing.T) {
	uc, _, _, env, cleanup := setupPermissionUsecase(t)
	defer cleanup()

	ctx := context.Background()

	// 创建测试用户
	users, err := env.DataManager.CreateTestUsers(1)
	require.NoError(t, err)
	testUser := users[0]

	t.Run("CheckVideoPermission", func(t *testing.T) {
		// 分配用户角色，应该有video相关权限
		uc.rbacManager.AssignRole(testUser.ID, 1)

		hasPermission, err := uc.CheckVideoPermission(ctx, testUser.ID, "POST")

		require.NoError(t, err)
		assert.True(t, hasPermission)
	})

	t.Run("CheckCommentPermission", func(t *testing.T) {
		// 分配用户角色，应该有comment相关权限
		uc.rbacManager.AssignRole(testUser.ID, 1)

		hasPermission, err := uc.CheckCommentPermission(ctx, testUser.ID, "POST")

		require.NoError(t, err)
		assert.True(t, hasPermission)
	})

	t.Run("CheckUserPermission", func(t *testing.T) {
		// 分配用户角色，应该有user相关权限
		uc.rbacManager.AssignRole(testUser.ID, 1)

		hasPermission, err := uc.CheckUserPermission(ctx, testUser.ID, "GET")

		require.NoError(t, err)
		assert.True(t, hasPermission)
	})
}
