package data

import (
	"context"
	"testing"
	"time"

	"go-backend/internal/data/cache"
	"go-backend/internal/domain"
	pkgcache "go-backend/pkg/cache"
	"go-backend/testutils"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupSessionRepo(t *testing.T) (*SessionRepo, *testutils.TestEnv, func()) {
	env, cleanup, err := testutils.SetupTestWithCleanup()
	require.NoError(t, err)

	data := &Data{
		db:  env.DB.DB,
		rdb: env.Redis.Client,
	}

	// 创建缓存
	multiCache := pkgcache.NewMultiLevelCache(env.Redis.Client, &pkgcache.CacheConfig{
		EnableL1: true,
		EnableL2: true,
	})
	authCache := cache.NewAuthCache(multiCache, log.DefaultLogger)

	repo := NewSessionRepo(data, authCache, log.DefaultLogger)

	return repo, env, cleanup
}

func TestSessionRepo_CreateSession(t *testing.T) {
	repo, env, cleanup := setupSessionRepo(t)
	defer cleanup()

	ctx := context.Background()

	// 创建测试用户
	users, err := env.DataManager.CreateTestUsers(1)
	require.NoError(t, err)
	user := users[0]

	// 创建会话
	session := &domain.UserSession{
		UserID:       user.ID,
		RefreshToken: "test-refresh-token",
		ExpiresAt:    time.Now().Add(time.Hour),
	}

	err = repo.CreateSession(ctx, session)
	require.NoError(t, err)
	assert.NotZero(t, session.ID)
	assert.NotZero(t, session.CreatedAt)

	// 验证数据库中的数据
	var dbSession UserSession
	err = env.DB.DB.Where("user_id = ?", user.ID).First(&dbSession).Error
	require.NoError(t, err)
	assert.Equal(t, user.ID, dbSession.UserID)
	assert.Equal(t, "test-refresh-token", dbSession.RefreshToken)
}

func TestSessionRepo_GetSession(t *testing.T) {
	repo, env, cleanup := setupSessionRepo(t)
	defer cleanup()

	ctx := context.Background()

	// 创建测试用户
	users, err := env.DataManager.CreateTestUsers(1)
	require.NoError(t, err)
	user := users[0]

	// 创建会话
	session := &domain.UserSession{
		UserID:       user.ID,
		RefreshToken: "test-refresh-token",
		ExpiresAt:    time.Now().Add(time.Hour),
	}

	err = repo.CreateSession(ctx, session)
	require.NoError(t, err)

	// 获取会话
	retrieved, err := repo.GetSession(ctx, user.ID)
	require.NoError(t, err)
	assert.Equal(t, session.UserID, retrieved.UserID)
	assert.Equal(t, session.RefreshToken, retrieved.RefreshToken)

	// 测试会话不存在
	_, err = repo.GetSession(ctx, 99999)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "session not found")
}

func TestSessionRepo_GetSessionByToken(t *testing.T) {
	repo, env, cleanup := setupSessionRepo(t)
	defer cleanup()

	ctx := context.Background()

	// 创建测试用户
	users, err := env.DataManager.CreateTestUsers(1)
	require.NoError(t, err)
	user := users[0]

	// 创建会话
	refreshToken := "test-refresh-token-123"
	session := &domain.UserSession{
		UserID:       user.ID,
		RefreshToken: refreshToken,
		ExpiresAt:    time.Now().Add(time.Hour),
	}

	err = repo.CreateSession(ctx, session)
	require.NoError(t, err)

	// 根据Token获取会话
	retrieved, err := repo.GetSessionByToken(ctx, refreshToken)
	require.NoError(t, err)
	assert.Equal(t, session.UserID, retrieved.UserID)
	assert.Equal(t, refreshToken, retrieved.RefreshToken)

	// 测试Token不存在
	_, err = repo.GetSessionByToken(ctx, "nonexistent-token")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "session not found")
}

func TestSessionRepo_UpdateSession(t *testing.T) {
	repo, env, cleanup := setupSessionRepo(t)
	defer cleanup()

	ctx := context.Background()

	// 创建测试用户
	users, err := env.DataManager.CreateTestUsers(1)
	require.NoError(t, err)
	user := users[0]

	// 创建会话
	session := &domain.UserSession{
		UserID:       user.ID,
		RefreshToken: "old-refresh-token",
		ExpiresAt:    time.Now().Add(time.Hour),
	}

	err = repo.CreateSession(ctx, session)
	require.NoError(t, err)

	// 更新会话
	newToken := "new-refresh-token"
	newExpiry := 2 * time.Hour
	err = repo.UpdateSession(ctx, user.ID, newToken, newExpiry)
	require.NoError(t, err)

	// 验证更新结果
	updated, err := repo.GetSession(ctx, user.ID)
	require.NoError(t, err)
	assert.Equal(t, newToken, updated.RefreshToken)
	assert.True(t, updated.ExpiresAt.After(session.ExpiresAt))
}

func TestSessionRepo_DeleteSession(t *testing.T) {
	repo, env, cleanup := setupSessionRepo(t)
	defer cleanup()

	ctx := context.Background()

	// 创建测试用户
	users, err := env.DataManager.CreateTestUsers(1)
	require.NoError(t, err)
	user := users[0]

	// 创建会话
	session := &domain.UserSession{
		UserID:       user.ID,
		RefreshToken: "test-refresh-token",
		ExpiresAt:    time.Now().Add(time.Hour),
	}

	err = repo.CreateSession(ctx, session)
	require.NoError(t, err)

	// 删除会话
	err = repo.DeleteSession(ctx, user.ID)
	require.NoError(t, err)

	// 验证会话已删除
	_, err = repo.GetSession(ctx, user.ID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "session not found")
}

func TestSessionRepo_AddTokenToBlacklist(t *testing.T) {
	repo, env, cleanup := setupSessionRepo(t)
	defer cleanup()

	ctx := context.Background()

	tokenID := "test-token-id"
	expiresAt := time.Now().Add(time.Hour)

	// 添加Token到黑名单
	err := repo.AddTokenToBlacklist(ctx, tokenID, expiresAt)
	require.NoError(t, err)

	// 验证数据库中的数据
	var blacklist TokenBlacklist
	err = env.DB.DB.Where("token_id = ?", tokenID).First(&blacklist).Error
	require.NoError(t, err)
	assert.Equal(t, tokenID, blacklist.TokenID)
}

func TestSessionRepo_IsTokenBlacklisted(t *testing.T) {
	repo, _, cleanup := setupSessionRepo(t)
	defer cleanup()

	ctx := context.Background()

	tokenID := "blacklisted-token"
	expiresAt := time.Now().Add(time.Hour)

	// 添加Token到黑名单
	err := repo.AddTokenToBlacklist(ctx, tokenID, expiresAt)
	require.NoError(t, err)

	// 检查Token是否在黑名单中
	isBlacklisted, err := repo.IsTokenBlacklisted(ctx, tokenID)
	require.NoError(t, err)
	assert.True(t, isBlacklisted)

	// 检查不在黑名单中的Token
	isBlacklisted, err = repo.IsTokenBlacklisted(ctx, "not-blacklisted")
	require.NoError(t, err)
	assert.False(t, isBlacklisted)
}

func TestSessionRepo_ExpiredSession(t *testing.T) {
	repo, env, cleanup := setupSessionRepo(t)
	defer cleanup()

	ctx := context.Background()

	// 创建测试用户
	users, err := env.DataManager.CreateTestUsers(1)
	require.NoError(t, err)
	user := users[0]

	// 创建过期会话
	session := &domain.UserSession{
		UserID:       user.ID,
		RefreshToken: "expired-token",
		ExpiresAt:    time.Now().Add(-time.Hour), // 已过期
	}

	err = repo.CreateSession(ctx, session)
	require.NoError(t, err)

	// 尝试获取过期会话
	_, err = repo.GetSession(ctx, user.ID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "session not found")

	// 尝试根据过期Token获取会话
	_, err = repo.GetSessionByToken(ctx, "expired-token")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "session not found")
}

func TestSessionRepo_ExpiredTokenBlacklist(t *testing.T) {
	repo, _, cleanup := setupSessionRepo(t)
	defer cleanup()

	ctx := context.Background()

	tokenID := "expired-blacklist-token"
	expiresAt := time.Now().Add(-time.Hour) // 已过期

	// 添加过期Token到黑名单
	err := repo.AddTokenToBlacklist(ctx, tokenID, expiresAt)
	require.NoError(t, err)

	// 检查过期Token（应该不在黑名单中）
	isBlacklisted, err := repo.IsTokenBlacklisted(ctx, tokenID)
	require.NoError(t, err)
	assert.False(t, isBlacklisted)
}

func TestSessionRepo_ConvertToSession(t *testing.T) {
	repo, _, cleanup := setupSessionRepo(t)
	defer cleanup()

	// 创建数据库会话模型
	dbSession := &UserSession{
		ID:           123,
		UserID:       456,
		RefreshToken: "test-token",
		ExpiresAt:    time.Now().Add(time.Hour),
		CreatedAt:    time.Now(),
	}

	// 转换为领域模型
	domainSession := repo.convertToSession(dbSession)

	// 验证转换结果
	assert.Equal(t, dbSession.ID, domainSession.ID)
	assert.Equal(t, dbSession.UserID, domainSession.UserID)
	assert.Equal(t, dbSession.RefreshToken, domainSession.RefreshToken)
	assert.Equal(t, dbSession.ExpiresAt, domainSession.ExpiresAt)
	assert.Equal(t, dbSession.CreatedAt, domainSession.CreatedAt)
}

func TestSessionRepo_CacheIntegration(t *testing.T) {
	repo, env, cleanup := setupSessionRepo(t)
	defer cleanup()

	ctx := context.Background()

	// 创建测试用户
	users, err := env.DataManager.CreateTestUsers(1)
	require.NoError(t, err)
	user := users[0]

	// 创建会话
	session := &domain.UserSession{
		UserID:       user.ID,
		RefreshToken: "cached-token",
		ExpiresAt:    time.Now().Add(time.Hour),
	}

	err = repo.CreateSession(ctx, session)
	require.NoError(t, err)

	// 第一次获取（从数据库）
	retrieved1, err := repo.GetSession(ctx, user.ID)
	require.NoError(t, err)
	assert.Equal(t, session.RefreshToken, retrieved1.RefreshToken)

	// 第二次获取（应该从缓存）
	retrieved2, err := repo.GetSession(ctx, user.ID)
	require.NoError(t, err)
	assert.Equal(t, session.RefreshToken, retrieved2.RefreshToken)

	// 删除会话应该清除缓存
	err = repo.DeleteSession(ctx, user.ID)
	require.NoError(t, err)

	// 再次获取应该失败
	_, err = repo.GetSession(ctx, user.ID)
	assert.Error(t, err)
}
