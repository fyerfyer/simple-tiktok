package data

import (
	"context"
	"testing"

	"go-backend/internal/domain"
	"go-backend/testutils"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupPermissionRepo(t *testing.T) (*PermissionRepo, *testutils.TestEnv, func()) {
	env, cleanup, err := testutils.SetupTestWithCleanup()
	require.NoError(t, err)

	data := &Data{
		db:  env.DB.DB,
		rdb: env.Redis.Client,
	}

	// 创建角色仓储
	roleRepo := NewRoleRepo(data, log.DefaultLogger)

	repo := NewPermissionRepo(data, roleRepo, log.DefaultLogger)

	return repo, env, cleanup
}

func TestPermissionRepo_GetPermission(t *testing.T) {
	repo, env, cleanup := setupPermissionRepo(t)
	defer cleanup()

	ctx := context.Background()

	// 创建测试权限
	permissions, err := env.DataManager.CreateTestPermissions()
	require.NoError(t, err)
	testPerm := permissions[0]

	// 获取权限
	perm, err := repo.GetPermission(ctx, testPerm.ID)
	require.NoError(t, err)
	assert.Equal(t, testPerm.ID, perm.ID)
	assert.Equal(t, testPerm.Name, perm.Name)
	assert.Equal(t, testPerm.Resource, perm.Resource)
	assert.Equal(t, testPerm.Action, perm.Action)

	// 测试权限不存在
	_, err = repo.GetPermission(ctx, 99999)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "permission not found")
}

func TestPermissionRepo_GetRolePermissions(t *testing.T) {
	repo, env, cleanup := setupPermissionRepo(t)
	defer cleanup()

	ctx := context.Background()

	// 创建测试数据
	roles, err := env.DataManager.CreateTestRoles()
	require.NoError(t, err)
	permissions, err := env.DataManager.CreateTestPermissions()
	require.NoError(t, err)

	role := roles[0]
	perm1, perm2 := permissions[0], permissions[1]

	// 创建角色权限关联
	err = env.DB.DB.Create(&RolePermission{
		RoleID:       role.ID,
		PermissionID: perm1.ID,
	}).Error
	require.NoError(t, err)

	err = env.DB.DB.Create(&RolePermission{
		RoleID:       role.ID,
		PermissionID: perm2.ID,
	}).Error
	require.NoError(t, err)

	// 获取角色权限
	rolePerms, err := repo.GetRolePermissions(ctx, role.ID)
	require.NoError(t, err)
	assert.Len(t, rolePerms, 2)

	// 验证权限内容
	permMap := make(map[string]*domain.Permission)
	for _, perm := range rolePerms {
		permMap[perm.Name] = perm
	}
	assert.NotNil(t, permMap[perm1.Name])
	assert.NotNil(t, permMap[perm2.Name])
}

func TestPermissionRepo_GetUserPermissions(t *testing.T) {
	repo, env, cleanup := setupPermissionRepo(t)
	defer cleanup()

	ctx := context.Background()

	// 创建测试数据
	users, err := env.DataManager.CreateTestUsers(1)
	require.NoError(t, err)
	user := users[0]

	roles, err := env.DataManager.CreateTestRoles()
	require.NoError(t, err)
	role := roles[0]

	permissions, err := env.DataManager.CreateTestPermissions()
	require.NoError(t, err)
	perm1, perm2 := permissions[0], permissions[1]

	// 分配角色给用户
	err = env.DataManager.AssignRoleToUser(user.ID, role.ID)
	require.NoError(t, err)

	// 分配权限给角色
	err = env.DB.DB.Create(&RolePermission{
		RoleID:       role.ID,
		PermissionID: perm1.ID,
	}).Error
	require.NoError(t, err)

	err = env.DB.DB.Create(&RolePermission{
		RoleID:       role.ID,
		PermissionID: perm2.ID,
	}).Error
	require.NoError(t, err)

	// 获取用户权限
	userPerms, err := repo.GetUserPermissions(ctx, user.ID)
	require.NoError(t, err)
	assert.Len(t, userPerms, 2)

	// 验证权限内容
	permNames := make([]string, 0, len(userPerms))
	for _, perm := range userPerms {
		permNames = append(permNames, perm.Name)
	}
	assert.Contains(t, permNames, perm1.Name)
	assert.Contains(t, permNames, perm2.Name)
}

func TestPermissionRepo_HasPermission(t *testing.T) {
	repo, env, cleanup := setupPermissionRepo(t)
	defer cleanup()

	ctx := context.Background()

	// 创建测试数据
	users, err := env.DataManager.CreateTestUsers(1)
	require.NoError(t, err)
	user := users[0]

	roles, err := env.DataManager.CreateTestRoles()
	require.NoError(t, err)
	role := roles[0]

	permissions, err := env.DataManager.CreateTestPermissions()
	require.NoError(t, err)
	perm := permissions[0] // user:read, /user, GET

	// 分配角色和权限
	err = env.DataManager.AssignRoleToUser(user.ID, role.ID)
	require.NoError(t, err)

	err = env.DB.DB.Create(&RolePermission{
		RoleID:       role.ID,
		PermissionID: perm.ID,
	}).Error
	require.NoError(t, err)

	// 检查用户是否有权限
	hasPermission, err := repo.HasPermission(ctx, user.ID, "/user", "GET")
	require.NoError(t, err)
	assert.True(t, hasPermission)

	// 检查用户没有的权限
	hasPermission, err = repo.HasPermission(ctx, user.ID, "/admin", "POST")
	require.NoError(t, err)
	assert.False(t, hasPermission)
}

func TestPermissionRepo_ConvertToPermission(t *testing.T) {
	repo, env, cleanup := setupPermissionRepo(t)
	defer cleanup()

	// 创建测试权限
	permissions, err := env.DataManager.CreateTestPermissions()
	require.NoError(t, err)
	testPerm := permissions[0]

	// 创建数据库模型
	dbPerm := &Permission{
		ID:          testPerm.ID,
		Name:        testPerm.Name,
		Resource:    testPerm.Resource,
		Action:      testPerm.Action,
		Description: testPerm.Description,
		Status:      testPerm.Status,
		CreatedAt:   testPerm.CreatedAt,
		UpdatedAt:   testPerm.UpdatedAt,
	}

	// 转换为领域模型
	domainPerm := repo.convertToPermission(dbPerm)

	// 验证转换结果
	assert.Equal(t, dbPerm.ID, domainPerm.ID)
	assert.Equal(t, dbPerm.Name, domainPerm.Name)
	assert.Equal(t, dbPerm.Resource, domainPerm.Resource)
	assert.Equal(t, dbPerm.Action, domainPerm.Action)
	assert.Equal(t, dbPerm.Description, domainPerm.Description)
	assert.Equal(t, dbPerm.Status, domainPerm.Status)
}

func TestPermissionRepo_EmptyResults(t *testing.T) {
	repo, env, cleanup := setupPermissionRepo(t)
	defer cleanup()

	ctx := context.Background()

	// 创建没有权限的用户
	users, err := env.DataManager.CreateTestUsers(1)
	require.NoError(t, err)
	user := users[0]

	// 获取用户权限（应该为空）
	userPerms, err := repo.GetUserPermissions(ctx, user.ID)
	require.NoError(t, err)
	assert.Empty(t, userPerms)

	// 检查不存在的角色权限
	rolePerms, err := repo.GetRolePermissions(ctx, 99999)
	require.NoError(t, err)
	assert.Empty(t, rolePerms)
}

func TestPermissionRepo_MultipleRolesAndPermissions(t *testing.T) {
	repo, env, cleanup := setupPermissionRepo(t)
	defer cleanup()

	ctx := context.Background()

	// 创建测试数据
	users, err := env.DataManager.CreateTestUsers(1)
	require.NoError(t, err)
	user := users[0]

	roles, err := env.DataManager.CreateTestRoles()
	require.NoError(t, err)
	role1, role2 := roles[0], roles[1]

	permissions, err := env.DataManager.CreateTestPermissions()
	require.NoError(t, err)
	perm1, perm2, perm3 := permissions[0], permissions[1], permissions[2]

	// 分配多个角色给用户
	err = env.DataManager.AssignRoleToUser(user.ID, role1.ID)
	require.NoError(t, err)
	err = env.DataManager.AssignRoleToUser(user.ID, role2.ID)
	require.NoError(t, err)

	// 分配权限给角色
	err = env.DB.DB.Create(&RolePermission{
		RoleID:       role1.ID,
		PermissionID: perm1.ID,
	}).Error
	require.NoError(t, err)

	err = env.DB.DB.Create(&RolePermission{
		RoleID:       role1.ID,
		PermissionID: perm2.ID,
	}).Error
	require.NoError(t, err)

	err = env.DB.DB.Create(&RolePermission{
		RoleID:       role2.ID,
		PermissionID: perm3.ID,
	}).Error
	require.NoError(t, err)

	// 获取用户权限（应该包含所有权限）
	userPerms, err := repo.GetUserPermissions(ctx, user.ID)
	require.NoError(t, err)
	assert.Len(t, userPerms, 3)

	// 验证所有权限都存在
	permNames := make(map[string]bool)
	for _, perm := range userPerms {
		permNames[perm.Name] = true
	}
	assert.True(t, permNames[perm1.Name])
	assert.True(t, permNames[perm2.Name])
	assert.True(t, permNames[perm3.Name])
}
