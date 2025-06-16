package service

import (
	"context"
	"testing"
	"time"

	"go-backend/internal/biz"
	"go-backend/internal/conf"
	"go-backend/internal/data"
	"go-backend/pkg/auth"
	"go-backend/testutils"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/durationpb"
)

func setupPermissionService(t *testing.T) (*PermissionService, *testutils.TestEnv, func()) {
	env, cleanup, err := testutils.SetupTestWithCleanup()
	require.NoError(t, err)

	// 创建配置
	config := &conf.Data{
		Database: &conf.Data_Database{
			Driver:          "mysql",
			Source:          "tiktok:tiktok123@tcp(localhost:3306)/tiktok?charset=utf8mb4&parseTime=True&loc=Local",
			MaxIdleConns:    10,
			MaxOpenConns:    100,
			ConnMaxLifetime: durationpb.New(time.Hour),
		},
		Redis: &conf.Data_Redis{
			Addr:         "localhost:6380",
			Password:     "tiktok123",
			Db:           1,
			DialTimeout:  durationpb.New(5 * time.Second),
			ReadTimeout:  durationpb.New(3 * time.Second),
			WriteTimeout: durationpb.New(3 * time.Second),
			PoolSize:     100,
		},
	}

	d, cleanup, err := data.NewData(config, log.DefaultLogger)
	assert.NoError(t, err, "Failed to create data instance")

	// 创建仓储
	roleRepo := data.NewRoleRepo(d, log.DefaultLogger)
	permissionRepo := data.NewPermissionRepo(d, roleRepo, log.DefaultLogger)

	// 创建用例
	rbacManager := auth.NewMemoryRBACManager()
	permissionUc := biz.NewPermissionUsecase(roleRepo, permissionRepo, rbacManager, log.DefaultLogger)

	// 创建服务
	service := NewPermissionService(permissionUc, log.DefaultLogger)

	return service, env, cleanup
}

func TestPermissionService_CheckPermission(t *testing.T) {
	service, env, cleanup := setupPermissionService(t)
	defer cleanup()

	ctx := context.Background()

	// 创建测试用户
	users, err := env.DataManager.CreateTestUsers(1)
	require.NoError(t, err)
	testUser := users[0]

	// 创建测试角色和权限
	roles, err := env.DataManager.CreateTestRoles()
	require.NoError(t, err)
	permissions, err := env.DataManager.CreateTestPermissions()
	require.NoError(t, err)

	t.Run("CheckPermission_Success", func(t *testing.T) {
		// 分配角色给用户
		err := env.DataManager.AssignRoleToUser(testUser.ID, roles[0].ID)
		require.NoError(t, err)

		// 分配权限给角色
		err = env.DB.DB.Exec("INSERT INTO role_permissions (role_id, permission_id, created_at) VALUES (?, ?, ?)",
			roles[0].ID, permissions[0].ID, time.Now()).Error
		require.NoError(t, err)

		hasPermission, err := service.CheckPermission(ctx, testUser.ID, "/user", "GET")

		require.NoError(t, err)
		assert.True(t, hasPermission)
	})

	t.Run("CheckPermission_NoPermission", func(t *testing.T) {
		hasPermission, err := service.CheckPermission(ctx, testUser.ID, "/admin", "DELETE")

		require.NoError(t, err)
		assert.False(t, hasPermission)
	})
}

func TestPermissionService_GetUserRoles(t *testing.T) {
	service, env, cleanup := setupPermissionService(t)
	defer cleanup()

	ctx := context.Background()

	// 创建测试用户
	users, err := env.DataManager.CreateTestUsers(1)
	require.NoError(t, err)
	testUser := users[0]

	// 创建测试角色
	roles, err := env.DataManager.CreateTestRoles()
	require.NoError(t, err)

	t.Run("GetUserRoles_Success", func(t *testing.T) {
		// 分配角色给用户
		err := env.DataManager.AssignRoleToUser(testUser.ID, roles[0].ID)
		require.NoError(t, err)
		err = env.DataManager.AssignRoleToUser(testUser.ID, roles[1].ID)
		require.NoError(t, err)

		userRoles, err := service.GetUserRoles(ctx, testUser.ID)

		require.NoError(t, err)
		assert.Len(t, userRoles, 2)
	})

	t.Run("GetUserRoles_Empty", func(t *testing.T) {
		// 创建新用户（没有角色）
		newUsers, err := env.DataManager.CreateTestUsers(1)
		require.NoError(t, err)

		userRoles, err := service.GetUserRoles(ctx, newUsers[0].ID)

		require.NoError(t, err)
		assert.Empty(t, userRoles)
	})
}

func TestPermissionService_GetUserPermissions(t *testing.T) {
	service, env, cleanup := setupPermissionService(t)
	defer cleanup()

	ctx := context.Background()

	// 创建测试用户
	users, err := env.DataManager.CreateTestUsers(1)
	require.NoError(t, err)
	testUser := users[0]

	// 创建测试角色和权限
	roles, err := env.DataManager.CreateTestRoles()
	require.NoError(t, err)
	permissions, err := env.DataManager.CreateTestPermissions()
	require.NoError(t, err)

	t.Run("GetUserPermissions_Success", func(t *testing.T) {
		// 分配角色给用户
		err := env.DataManager.AssignRoleToUser(testUser.ID, roles[0].ID)
		require.NoError(t, err)

		// 分配权限给角色
		err = env.DB.DB.Exec("INSERT INTO role_permissions (role_id, permission_id, created_at) VALUES (?, ?, ?)",
			roles[0].ID, permissions[0].ID, time.Now()).Error
		require.NoError(t, err)

		userPermissions, err := service.GetUserPermissions(ctx, testUser.ID)

		require.NoError(t, err)
		assert.Len(t, userPermissions, 1)
		assert.Equal(t, permissions[0].Name, userPermissions[0].Name)
	})
}

func TestPermissionService_AssignRole(t *testing.T) {
	service, env, cleanup := setupPermissionService(t)
	defer cleanup()

	ctx := context.Background()

	// 创建测试用户和角色
	users, err := env.DataManager.CreateTestUsers(1)
	require.NoError(t, err)
	testUser := users[0]

	roles, err := env.DataManager.CreateTestRoles()
	require.NoError(t, err)

	t.Run("AssignRole_Success", func(t *testing.T) {
		err := service.AssignRole(ctx, testUser.ID, roles[0].ID)

		assert.NoError(t, err)

		// 验证角色已分配
		hasRole, err := service.HasRole(ctx, testUser.ID, roles[0].ID)
		require.NoError(t, err)
		assert.True(t, hasRole)
	})
}

func TestPermissionService_RemoveRole(t *testing.T) {
	service, env, cleanup := setupPermissionService(t)
	defer cleanup()

	ctx := context.Background()

	// 创建测试用户和角色
	users, err := env.DataManager.CreateTestUsers(1)
	require.NoError(t, err)
	testUser := users[0]

	roles, err := env.DataManager.CreateTestRoles()
	require.NoError(t, err)

	t.Run("RemoveRole_Success", func(t *testing.T) {
		// 先分配角色
		err := env.DataManager.AssignRoleToUser(testUser.ID, roles[0].ID)
		require.NoError(t, err)

		// 移除角色
		err = service.RemoveRole(ctx, testUser.ID, roles[0].ID)

		assert.NoError(t, err)

		// 验证角色已移除
		hasRole, err := service.HasRole(ctx, testUser.ID, roles[0].ID)
		require.NoError(t, err)
		assert.False(t, hasRole)
	})
}

func TestPermissionService_HasRole(t *testing.T) {
	service, env, cleanup := setupPermissionService(t)
	defer cleanup()

	ctx := context.Background()

	// 创建测试用户和角色
	users, err := env.DataManager.CreateTestUsers(1)
	require.NoError(t, err)
	testUser := users[0]

	roles, err := env.DataManager.CreateTestRoles()
	require.NoError(t, err)

	t.Run("HasRole_True", func(t *testing.T) {
		// 分配角色
		err := env.DataManager.AssignRoleToUser(testUser.ID, roles[0].ID)
		require.NoError(t, err)

		hasRole, err := service.HasRole(ctx, testUser.ID, roles[0].ID)

		require.NoError(t, err)
		assert.True(t, hasRole)
	})

	t.Run("HasRole_False", func(t *testing.T) {
		hasRole, err := service.HasRole(ctx, testUser.ID, roles[1].ID)

		require.NoError(t, err)
		assert.False(t, hasRole)
	})
}

func TestPermissionService_IsAdmin(t *testing.T) {
	service, env, cleanup := setupPermissionService(t)
	defer cleanup()

	ctx := context.Background()

	// 创建测试用户
	users, err := env.DataManager.CreateTestUsers(1)
	require.NoError(t, err)
	testUser := users[0]

	// 创建测试角色
	roles, err := env.DataManager.CreateTestRoles()
	require.NoError(t, err)

	t.Run("IsAdmin_True", func(t *testing.T) {
		// 找到admin角色并分配
		var adminRole *testutils.TestRole
		for _, role := range roles {
			if role.Name == "admin" {
				adminRole = role
				break
			}
		}
		require.NotNil(t, adminRole)

		err := env.DataManager.AssignRoleToUser(testUser.ID, adminRole.ID)
		require.NoError(t, err)

		isAdmin, err := service.IsAdmin(ctx, testUser.ID)

		require.NoError(t, err)
		assert.True(t, isAdmin)
	})

	t.Run("IsAdmin_False", func(t *testing.T) {
		// 创建新用户（非管理员）
		newUsers, err := env.DataManager.CreateTestUsers(1)
		require.NoError(t, err)

		isAdmin, err := service.IsAdmin(ctx, newUsers[0].ID)

		require.NoError(t, err)
		assert.False(t, isAdmin)
	})
}

func TestPermissionService_IsModerator(t *testing.T) {
	service, env, cleanup := setupPermissionService(t)
	defer cleanup()

	ctx := context.Background()

	// 创建测试用户
	users, err := env.DataManager.CreateTestUsers(1)
	require.NoError(t, err)
	testUser := users[0]

	// 创建测试角色
	roles, err := env.DataManager.CreateTestRoles()
	require.NoError(t, err)

	t.Run("IsModerator_True", func(t *testing.T) {
		// 找到moderator角色并分配
		var modRole *testutils.TestRole
		for _, role := range roles {
			if role.Name == "moderator" {
				modRole = role
				break
			}
		}
		require.NotNil(t, modRole)

		err := env.DataManager.AssignRoleToUser(testUser.ID, modRole.ID)
		require.NoError(t, err)

		isModerator, err := service.IsModerator(ctx, testUser.ID)

		require.NoError(t, err)
		assert.True(t, isModerator)
	})

	t.Run("IsModerator_False", func(t *testing.T) {
		// 创建新用户（非审核员）
		newUsers, err := env.DataManager.CreateTestUsers(1)
		require.NoError(t, err)

		isModerator, err := service.IsModerator(ctx, newUsers[0].ID)

		require.NoError(t, err)
		assert.False(t, isModerator)
	})
}

func TestPermissionService_ValidateResourceAccess(t *testing.T) {
	service, env, cleanup := setupPermissionService(t)
	defer cleanup()

	ctx := context.Background()

	// 创建测试用户
	users, err := env.DataManager.CreateTestUsers(1)
	require.NoError(t, err)
	testUser := users[0]

	// 创建测试角色和权限
	roles, err := env.DataManager.CreateTestRoles()
	require.NoError(t, err)
	permissions, err := env.DataManager.CreateTestPermissions()
	require.NoError(t, err)

	t.Run("ValidateResourceAccess_Success", func(t *testing.T) {
		// 分配角色和权限
		err := env.DataManager.AssignRoleToUser(testUser.ID, roles[0].ID)
		require.NoError(t, err)

		err = env.DB.DB.Exec("INSERT INTO role_permissions (role_id, permission_id, created_at) VALUES (?, ?, ?)",
			roles[0].ID, permissions[0].ID, time.Now()).Error
		require.NoError(t, err)

		err = service.ValidateResourceAccess(ctx, testUser.ID, "/user", "GET")

		assert.NoError(t, err)
	})

	t.Run("ValidateResourceAccess_PermissionDenied", func(t *testing.T) {
		err := service.ValidateResourceAccess(ctx, testUser.ID, "/admin", "DELETE")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "permission denied")
	})
}

func TestPermissionService_CheckSpecificPermissions(t *testing.T) {
	service, env, cleanup := setupPermissionService(t)
	defer cleanup()

	ctx := context.Background()

	// 创建测试用户
	users, err := env.DataManager.CreateTestUsers(1)
	require.NoError(t, err)
	testUser := users[0]

	// 创建测试角色和权限
	roles, err := env.DataManager.CreateTestRoles()
	require.NoError(t, err)
	permissions, err := env.DataManager.CreateTestPermissions()
	require.NoError(t, err)

	// 分配用户角色和权限
	err = env.DataManager.AssignRoleToUser(testUser.ID, roles[0].ID)
	require.NoError(t, err)

	// 分配video权限
	var videoPermission *testutils.TestPermission
	for _, perm := range permissions {
		if perm.Resource == "/video" {
			videoPermission = perm
			break
		}
	}
	require.NotNil(t, videoPermission)

	err = env.DB.DB.Exec("INSERT INTO role_permissions (role_id, permission_id, created_at) VALUES (?, ?, ?)",
		roles[0].ID, videoPermission.ID, time.Now()).Error
	require.NoError(t, err)

	t.Run("CheckVideoPermission", func(t *testing.T) {
		hasPermission, err := service.CheckVideoPermission(ctx, testUser.ID, "POST")

		require.NoError(t, err)
		assert.True(t, hasPermission)
	})

	t.Run("CheckCommentPermission", func(t *testing.T) {
		hasPermission, err := service.CheckCommentPermission(ctx, testUser.ID, "POST")

		require.NoError(t, err)
		// 没有分配comment权限，应该返回false
		assert.False(t, hasPermission)
	})

	t.Run("CheckUserPermission", func(t *testing.T) {
		// 分配用户权限
		var userPermission *testutils.TestPermission
		for _, perm := range permissions {
			if perm.Resource == "/user" {
				userPermission = perm
				break
			}
		}
		require.NotNil(t, userPermission)

		err = env.DB.DB.Exec("INSERT INTO role_permissions (role_id, permission_id, created_at) VALUES (?, ?, ?)",
			roles[0].ID, userPermission.ID, time.Now()).Error
		require.NoError(t, err)

		hasPermission, err := service.CheckUserPermission(ctx, testUser.ID, "GET")

		require.NoError(t, err)
		assert.True(t, hasPermission)
	})
}

func TestPermissionService_ClearUserPermissionCache(t *testing.T) {
	service, env, cleanup := setupPermissionService(t)
	defer cleanup()

	ctx := context.Background()

	// 创建测试用户
	users, err := env.DataManager.CreateTestUsers(1)
	require.NoError(t, err)
	testUser := users[0]

	t.Run("ClearUserPermissionCache_Success", func(t *testing.T) {
		// 调用清除缓存方法
		service.ClearUserPermissionCache(ctx, testUser.ID)

		// 这里主要测试方法调用不会出错
		assert.True(t, true)
	})
}

func TestPermissionService_InitUserRole(t *testing.T) {
	service, env, cleanup := setupPermissionService(t)
	defer cleanup()

	ctx := context.Background()

	// 创建测试用户
	users, err := env.DataManager.CreateTestUsers(1)
	require.NoError(t, err)
	testUser := users[0]

	t.Run("InitUserRole_Success", func(t *testing.T) {
		err := service.InitUserRole(ctx, testUser.ID)

		require.NoError(t, err)

		// 验证默认角色已分配
		roles, err := service.GetUserRoles(ctx, testUser.ID)
		require.NoError(t, err)
		assert.NotEmpty(t, roles)

		// 验证有"user"角色
		hasUserRole := false
		for _, role := range roles {
			if role.Name == "user" {
				hasUserRole = true
				break
			}
		}
		assert.True(t, hasUserRole)
	})
}

func TestPermissionService_AdvancedPermissionChecks(t *testing.T) {
	service, env, cleanup := setupPermissionService(t)
	defer cleanup()

	ctx := context.Background()

	// 创建测试用户
	users, err := env.DataManager.CreateTestUsers(2)
	require.NoError(t, err)
	user1, user2 := users[0], users[1]

	// 创建测试角色
	roles, err := env.DataManager.CreateTestRoles()
	require.NoError(t, err)

	t.Run("CanAccessVideo", func(t *testing.T) {
		// 分配用户角色
		err := env.DataManager.AssignRoleToUser(user1.ID, roles[0].ID)
		require.NoError(t, err)

		canAccess, err := service.CanAccessVideo(ctx, user1.ID, 1)

		require.NoError(t, err)
		assert.True(t, canAccess)
	})

	t.Run("CanModerateContent_Admin", func(t *testing.T) {
		// 分配admin角色
		var adminRole *testutils.TestRole
		for _, role := range roles {
			if role.Name == "admin" {
				adminRole = role
				break
			}
		}
		require.NotNil(t, adminRole)

		err := env.DataManager.AssignRoleToUser(user1.ID, adminRole.ID)
		require.NoError(t, err)

		canModerate, err := service.CanModerateContent(ctx, user1.ID)

		require.NoError(t, err)
		assert.True(t, canModerate)
	})

	t.Run("CanModerateContent_Moderator", func(t *testing.T) {
		// 分配moderator角色
		var modRole *testutils.TestRole
		for _, role := range roles {
			if role.Name == "moderator" {
				modRole = role
				break
			}
		}
		require.NotNil(t, modRole)

		err := env.DataManager.AssignRoleToUser(user2.ID, modRole.ID)
		require.NoError(t, err)

		canModerate, err := service.CanModerateContent(ctx, user2.ID)

		require.NoError(t, err)
		assert.True(t, canModerate)
	})

	t.Run("CanDeleteComment_SelfComment", func(t *testing.T) {
		// 用户可以删除自己的评论
		canDelete, err := service.CanDeleteComment(ctx, user1.ID, user1.ID)

		require.NoError(t, err)
		assert.True(t, canDelete)
	})

	t.Run("CanDeleteComment_AdminCanDeleteAny", func(t *testing.T) {
		// 管理员可以删除任何评论
		canDelete, err := service.CanDeleteComment(ctx, user1.ID, user2.ID)

		require.NoError(t, err)
		assert.True(t, canDelete) // user1是admin
	})

	t.Run("CanManageUser", func(t *testing.T) {
		canManage, err := service.CanManageUser(ctx, user1.ID)

		require.NoError(t, err)
		assert.True(t, canManage) // user1是admin
	})
}
