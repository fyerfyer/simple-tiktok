package data

import (
	"context"
	"testing"

	"go-backend/internal/biz"
	"go-backend/internal/data/cache"
	"go-backend/pkg/auth"
	pkgcache "go-backend/pkg/cache"
	"go-backend/testutils"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupUserRepo(t *testing.T) (*userRepo, *testutils.TestEnv, func()) {
	env, cleanup, err := testutils.SetupTestWithCleanup()
	require.NoError(t, err)

	// 创建数据结构
	data := &Data{
		db:  env.DB.DB,
		rdb: env.Redis.Client,
	}

	// 创建缓存
	multiCache := pkgcache.NewMultiLevelCache(env.Redis.Client, &pkgcache.CacheConfig{
		EnableL1: true,
		EnableL2: true,
	})
	userCache := cache.NewUserCache(multiCache, log.DefaultLogger)
	passwordMgr := auth.NewPasswordManager()

	repo := &userRepo{
		data:        data,
		log:         log.NewHelper(log.DefaultLogger),
		userCache:   userCache,
		passwordMgr: passwordMgr,
	}

	return repo, env, cleanup
}

func TestUserRepo_CreateUser(t *testing.T) {
	repo, env, cleanup := setupUserRepo(t)
	defer cleanup()

	ctx := context.Background()

	user := &biz.User{
		Username:        "testuser",
		PasswordHash:    "Password123!",
		Nickname:        "Test User",
		Avatar:          "https://example.com/avatar.jpg",
		BackgroundImage: "https://example.com/bg.jpg",
		Signature:       "Test signature",
	}

	// 创建用户
	created, err := repo.CreateUser(ctx, user)
	require.NoError(t, err)
	assert.NotZero(t, created.ID)
	assert.Equal(t, user.Username, created.Username)
	assert.Equal(t, user.Nickname, created.Nickname)
	assert.NotEqual(t, user.PasswordHash, created.PasswordHash) // 密码应该被加密

	// 验证数据库中的数据
	var dbUser User
	err = env.DB.DB.Where("username = ?", user.Username).First(&dbUser).Error
	require.NoError(t, err)
	assert.Equal(t, user.Username, dbUser.Username)
}

func TestUserRepo_GetUser(t *testing.T) {
	repo, env, cleanup := setupUserRepo(t)
	defer cleanup()

	ctx := context.Background()

	// 创建测试数据
	users, err := env.DataManager.CreateTestUsers(1)
	require.NoError(t, err)
	testUser := users[0]

	// 获取用户
	user, err := repo.GetUser(ctx, testUser.ID)
	require.NoError(t, err)
	assert.Equal(t, testUser.ID, user.ID)
	assert.Equal(t, testUser.Username, user.Username)
	assert.Equal(t, testUser.Nickname, user.Nickname)

	// 测试用户不存在
	_, err = repo.GetUser(ctx, 99999)
	assert.Equal(t, biz.ErrUserNotFound, err)
}

func TestUserRepo_GetUserByUsername(t *testing.T) {
	repo, env, cleanup := setupUserRepo(t)
	defer cleanup()

	ctx := context.Background()

	// 创建测试数据
	users, err := env.DataManager.CreateTestUsers(1)
	require.NoError(t, err)
	testUser := users[0]

	// 根据用户名获取用户
	user, err := repo.GetUserByUsername(ctx, testUser.Username)
	require.NoError(t, err)
	assert.Equal(t, testUser.ID, user.ID)
	assert.Equal(t, testUser.Username, user.Username)

	// 测试用户不存在
	_, err = repo.GetUserByUsername(ctx, "nonexistent")
	assert.Equal(t, biz.ErrUserNotFound, err)
}

func TestUserRepo_GetUsers(t *testing.T) {
	repo, env, cleanup := setupUserRepo(t)
	defer cleanup()

	ctx := context.Background()

	// 创建测试数据
	testUsers, err := env.DataManager.CreateTestUsers(3)
	require.NoError(t, err)

	userIDs := make([]int64, 0, len(testUsers))
	for _, user := range testUsers {
		userIDs = append(userIDs, user.ID)
	}

	// 批量获取用户
	users, err := repo.GetUsers(ctx, userIDs)
	require.NoError(t, err)
	assert.Len(t, users, 3)

	// 验证返回的用户数据
	userMap := make(map[int64]*biz.User)
	for _, user := range users {
		userMap[user.ID] = user
	}

	for _, testUser := range testUsers {
		user, exists := userMap[testUser.ID]
		assert.True(t, exists)
		assert.Equal(t, testUser.Username, user.Username)
	}
}

func TestUserRepo_UpdateUser(t *testing.T) {
	repo, env, cleanup := setupUserRepo(t)
	defer cleanup()

	ctx := context.Background()

	// 创建测试数据
	users, err := env.DataManager.CreateTestUsers(1)
	require.NoError(t, err)
	testUser := users[0]

	// 获取用户
	user, err := repo.GetUser(ctx, testUser.ID)
	require.NoError(t, err)

	// 更新用户信息
	user.Nickname = "Updated Nickname"
	user.Signature = "Updated signature"

	err = repo.UpdateUser(ctx, user)
	require.NoError(t, err)

	// 验证更新结果
	updated, err := repo.GetUser(ctx, testUser.ID)
	require.NoError(t, err)
	assert.Equal(t, "Updated Nickname", updated.Nickname)
	assert.Equal(t, "Updated signature", updated.Signature)
}

func TestUserRepo_UpdateUserStats(t *testing.T) {
	repo, env, cleanup := setupUserRepo(t)
	defer cleanup()

	ctx := context.Background()

	// 创建测试数据
	users, err := env.DataManager.CreateTestUsers(1)
	require.NoError(t, err)
	testUser := users[0]

	// 更新用户统计
	stats := &biz.UserStats{
		FollowCountDelta:    1,
		FollowerCountDelta:  2,
		WorkCountDelta:      1,
		FavoriteCountDelta:  3,
		TotalFavoritedDelta: 5,
	}

	err = repo.UpdateUserStats(ctx, testUser.ID, stats)
	require.NoError(t, err)

	// 验证统计更新
	updated, err := repo.GetUser(ctx, testUser.ID)
	require.NoError(t, err)
	assert.Equal(t, 1, updated.FollowCount)
	assert.Equal(t, 2, updated.FollowerCount)
	assert.Equal(t, 1, updated.WorkCount)
	assert.Equal(t, 3, updated.FavoriteCount)
	assert.Equal(t, int64(5), updated.TotalFavorited)
}

func TestUserRepo_VerifyPassword(t *testing.T) {
	repo, _, cleanup := setupUserRepo(t)
	defer cleanup()

	ctx := context.Background()

	// 创建用户
	user := &biz.User{
		Username:     "testuser",
		PasswordHash: "password123!",
		Nickname:     "Test User",
	}

	created, err := repo.CreateUser(ctx, user)
	require.NoError(t, err)

	// 验证正确密码
	verified, err := repo.VerifyPassword(ctx, created.Username, "password123!")
	require.NoError(t, err)
	assert.Equal(t, created.ID, verified.ID)

	// 验证错误密码
	_, err = repo.VerifyPassword(ctx, created.Username, "wrongpassword")
	assert.Equal(t, biz.ErrPasswordError, err)

	// 验证不存在的用户
	_, err = repo.VerifyPassword(ctx, "nonexistent", "password123!")
	assert.Equal(t, biz.ErrUserNotFound, err)
}
