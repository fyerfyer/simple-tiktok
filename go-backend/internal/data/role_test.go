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

func setupRoleRepo(t *testing.T) (*RoleRepo, *testutils.TestEnv, func()) {
	env, cleanup, err := testutils.SetupTestWithCleanup()
	require.NoError(t, err)

	data := &Data{
		db:  env.DB.DB,
		rdb: env.Redis.Client,
	}

	repo := NewRoleRepo(data, log.DefaultLogger)

	return repo, env, cleanup
}

func TestRoleRepo_GetRole(t *testing.T) {
	repo, env, cleanup := setupRoleRepo(t)
	defer cleanup()

	ctx := context.Background()

	// 创建测试角色
	roles, err := env.DataManager.CreateTestRoles()
	require.NoError(t, err)
	testRole := roles[0]

	// 获取角色
	role, err := repo.GetRole(ctx, testRole.ID)
	require.NoError(t, err)
	assert.Equal(t, testRole.ID, role.ID)
	assert.Equal(t, testRole.Name, role.Name)
	assert.Equal(t, testRole.Description, role.Description)

	// 测试角色不存在
	_, err = repo.GetRole(ctx, 99999)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "role not found")
}

func TestRoleRepo_GetRoleByName(t *testing.T) {
	repo, env, cleanup := setupRoleRepo(t)
	defer cleanup()

	ctx := context.Background()

	// 创建测试角色
	roles, err := env.DataManager.CreateTestRoles()
	require.NoError(t, err)
	testRole := roles[0]

	// 根据名称获取角色
	role, err := repo.GetRoleByName(ctx, testRole.Name)
	require.NoError(t, err)
	assert.Equal(t, testRole.ID, role.ID)
	assert.Equal(t, testRole.Name, role.Name)

	// 测试角色不存在
	_, err = repo.GetRoleByName(ctx, "nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "role not found")
}

func TestRoleRepo_AssignRole(t *testing.T) {
	repo, env, cleanup := setupRoleRepo(t)
	defer cleanup()

	ctx := context.Background()

	// 创建测试数据
	users, err := env.DataManager.CreateTestUsers(1)
	require.NoError(t, err)
	user := users[0]

	roles, err := env.DataManager.CreateTestRoles()
	require.NoError(t, err)
	role := roles[0]

	// 分配角色
	err = repo.AssignRole(ctx, user.ID, role.ID)
	require.NoError(t, err)

	// 验证角色分配
	hasRole, err := repo.HasRole(ctx, user.ID, role.ID)
	require.NoError(t, err)
	assert.True(t, hasRole)

	// 重复分配应该成功（不报错）
	err = repo.AssignRole(ctx, user.ID, role.ID)
	assert.NoError(t, err)
}

func TestRoleRepo_RemoveRole(t *testing.T) {
	repo, env, cleanup := setupRoleRepo(t)
	defer cleanup()

	ctx := context.Background()

	// 创建测试数据
	users, err := env.DataManager.CreateTestUsers(1)
	require.NoError(t, err)
	user := users[0]

	roles, err := env.DataManager.CreateTestRoles()
	require.NoError(t, err)
	role := roles[0]

	// 先分配角色
	err = env.DataManager.AssignRoleToUser(user.ID, role.ID)
	require.NoError(t, err)

	// 验证角色存在
	hasRole, err := repo.HasRole(ctx, user.ID, role.ID)
	require.NoError(t, err)
	assert.True(t, hasRole)

	// 移除角色
	err = repo.RemoveRole(ctx, user.ID, role.ID)
	require.NoError(t, err)

	// 验证角色已移除
	hasRole, err = repo.HasRole(ctx, user.ID, role.ID)
	require.NoError(t, err)
	assert.False(t, hasRole)
}

func TestRoleRepo_GetUserRoles(t *testing.T) {
	repo, env, cleanup := setupRoleRepo(t)
	defer cleanup()

	ctx := context.Background()

	// 创建测试数据
	users, err := env.DataManager.CreateTestUsers(1)
	require.NoError(t, err)
	user := users[0]

	roles, err := env.DataManager.CreateTestRoles()
	require.NoError(t, err)

	// 分配多个角色
	err = env.DataManager.AssignRoleToUser(user.ID, roles[0].ID)
	require.NoError(t, err)
	err = env.DataManager.AssignRoleToUser(user.ID, roles[1].ID)
	require.NoError(t, err)

	// 获取用户角色
	userRoles, err := repo.GetUserRoles(ctx, user.ID)
	require.NoError(t, err)
	assert.Len(t, userRoles, 2)

	// 验证角色信息
	roleMap := make(map[string]*domain.Role)
	for _, role := range userRoles {
		roleMap[role.Name] = role
	}

	assert.NotNil(t, roleMap["user"])
	assert.NotNil(t, roleMap["admin"])
}

func TestRoleRepo_HasRole(t *testing.T) {
	repo, env, cleanup := setupRoleRepo(t)
	defer cleanup()

	ctx := context.Background()

	// 创建测试数据
	users, err := env.DataManager.CreateTestUsers(1)
	require.NoError(t, err)
	user := users[0]

	roles, err := env.DataManager.CreateTestRoles()
	require.NoError(t, err)
	role := roles[0]

	// 初始状态：用户没有角色
	hasRole, err := repo.HasRole(ctx, user.ID, role.ID)
	require.NoError(t, err)
	assert.False(t, hasRole)

	// 分配角色
	err = env.DataManager.AssignRoleToUser(user.ID, role.ID)
	require.NoError(t, err)

	// 验证用户有角色
	hasRole, err = repo.HasRole(ctx, user.ID, role.ID)
	require.NoError(t, err)
	assert.True(t, hasRole)

	// 验证用户没有其他角色
	hasRole, err = repo.HasRole(ctx, user.ID, roles[1].ID)
	require.NoError(t, err)
	assert.False(t, hasRole)
}

func TestRoleRepo_ConvertToRole(t *testing.T) {
	repo, env, cleanup := setupRoleRepo(t)
	defer cleanup()

	// 创建测试角色数据
	roles, err := env.DataManager.CreateTestRoles()
	require.NoError(t, err)
	testRole := roles[0]

	// 创建数据库模型
	dbRole := &Role{
		ID:          testRole.ID,
		Name:        testRole.Name,
		Description: testRole.Description,
		Status:      testRole.Status,
		CreatedAt:   testRole.CreatedAt,
		UpdatedAt:   testRole.UpdatedAt,
	}

	// 转换为领域模型
	domainRole := repo.convertToRole(dbRole)

	// 验证转换结果
	assert.Equal(t, dbRole.ID, domainRole.ID)
	assert.Equal(t, dbRole.Name, domainRole.Name)
	assert.Equal(t, dbRole.Description, domainRole.Description)
	assert.Equal(t, dbRole.Status, domainRole.Status)
	assert.Equal(t, dbRole.CreatedAt, domainRole.CreatedAt)
	assert.Equal(t, dbRole.UpdatedAt, domainRole.UpdatedAt)
}

func TestRoleRepo_EmptyUserRoles(t *testing.T) {
	repo, env, cleanup := setupRoleRepo(t)
	defer cleanup()

	ctx := context.Background()

	// 创建测试用户（不分配任何角色）
	users, err := env.DataManager.CreateTestUsers(1)
	require.NoError(t, err)
	user := users[0]

	// 获取用户角色
	userRoles, err := repo.GetUserRoles(ctx, user.ID)
	require.NoError(t, err)
	assert.Empty(t, userRoles)
}
