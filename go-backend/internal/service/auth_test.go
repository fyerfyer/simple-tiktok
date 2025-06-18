package service

import (
	"context"
	"testing"
	"time"

	"go-backend/internal/biz"
	"go-backend/internal/conf"
	"go-backend/internal/data"
	"go-backend/internal/data/cache"
	"go-backend/pkg/auth"
	pkgcache "go-backend/pkg/cache"
	"go-backend/testutils"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-redis/redis/v8"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/durationpb"
)

func TestAuthService_RefreshToken(t *testing.T) {
	t.Run("RefreshToken_Success", func(t *testing.T) {
		// 创建独立的服务和环境
		service, env, cleanup := setupAuthServiceForTest(t)
		defer cleanup()

		ctx := context.Background()

		// 创建测试用户
		users, err := env.DataManager.CreateTestUsers(1)
		require.NoError(t, err)
		testUser := users[0]

		// 生成Token对
		jwtManager := auth.NewJWTManager("test-secret", time.Hour)
		tokenPair, err := jwtManager.GenerateTokenPair(testUser.ID, testUser.Username)
		require.NoError(t, err)

		// 创建会话
		err = env.DB.DB.Exec("INSERT INTO user_sessions (user_id, refresh_token, expires_at, created_at) VALUES (?, ?, ?, ?)",
			testUser.ID, tokenPair.RefreshToken, tokenPair.RefreshExpiry, time.Now()).Error
		require.NoError(t, err)

		// 刷新Token
		newTokenPair, err := service.RefreshToken(ctx, tokenPair.RefreshToken)

		require.NoError(t, err)
		assert.NotNil(t, newTokenPair)
		assert.NotEmpty(t, newTokenPair.AccessToken)
		assert.NotEmpty(t, newTokenPair.RefreshToken)
		assert.NotEqual(t, tokenPair.AccessToken, newTokenPair.AccessToken)
		assert.NotEqual(t, tokenPair.RefreshToken, newTokenPair.RefreshToken)
	})

	t.Run("RefreshToken_InvalidToken", func(t *testing.T) {
		// 创建独立的服务和环境
		service, _, cleanup := setupAuthServiceForTest(t)
		defer cleanup()

		ctx := context.Background()
		invalidToken := "invalid-refresh-token"

		tokenPair, err := service.RefreshToken(ctx, invalidToken)

		assert.Error(t, err)
		assert.Nil(t, tokenPair)
	})

	t.Run("RefreshToken_ExpiredToken", func(t *testing.T) {
		// 创建独立的服务和环境
		service, env, cleanup := setupAuthServiceForTest(t)
		defer cleanup()

		ctx := context.Background()

		// 创建测试用户
		users, err := env.DataManager.CreateTestUsers(1)
		require.NoError(t, err)
		testUser := users[0]

		// 生成过期Token对
		jwtManager := auth.NewJWTManager("test-secret", -time.Hour) // 负数表示已过期
		tokenPair, err := jwtManager.GenerateTokenPair(testUser.ID, testUser.Username)
		require.NoError(t, err)

		newTokenPair, err := service.RefreshToken(ctx, tokenPair.RefreshToken)

		assert.Error(t, err)
		assert.Nil(t, newTokenPair)
	})
}

func TestAuthService_Logout(t *testing.T) {
	t.Run("Logout_Success", func(t *testing.T) {
		// 创建独立的服务和环境
		service, env, cleanup := setupAuthServiceForTest(t)
		defer cleanup()

		ctx := context.Background()

		// 创建测试用户
		users, err := env.DataManager.CreateTestUsers(1)
		require.NoError(t, err)
		testUser := users[0]

		// 生成Token
		jwtManager := auth.NewJWTManager("test-secret", time.Hour)
		accessToken, err := jwtManager.GenerateToken(testUser.ID, testUser.Username)
		require.NoError(t, err)

		tokenPair, err := jwtManager.GenerateTokenPair(testUser.ID, testUser.Username)
		require.NoError(t, err)

		// 创建会话
		err = env.DB.DB.Exec("INSERT INTO user_sessions (user_id, refresh_token, expires_at, created_at) VALUES (?, ?, ?, ?)",
			testUser.ID, tokenPair.RefreshToken, tokenPair.RefreshExpiry, time.Now()).Error
		require.NoError(t, err)

		// 登出
		err = service.Logout(ctx, testUser.ID, accessToken, tokenPair.RefreshToken)

		assert.NoError(t, err)

		// 验证会话已删除
		var count int64
		err = env.DB.DB.Model(&data.UserSession{}).Where("user_id = ?", testUser.ID).Count(&count).Error
		require.NoError(t, err)
		assert.Equal(t, int64(0), count)
	})

	t.Run("Logout_WithoutRefreshToken", func(t *testing.T) {
		// 创建独立的服务和环境
		service, env, cleanup := setupAuthServiceForTest(t)
		defer cleanup()

		ctx := context.Background()

		// 创建测试用户
		users, err := env.DataManager.CreateTestUsers(1)
		require.NoError(t, err)
		testUser := users[0]

		// 生成Token
		jwtManager := auth.NewJWTManager("test-secret", time.Hour)
		accessToken, err := jwtManager.GenerateToken(testUser.ID, testUser.Username)
		require.NoError(t, err)

		// 只用Access Token登出
		err = service.Logout(ctx, testUser.ID, accessToken, "")

		assert.NoError(t, err)
	})
}

func TestAuthService_RevokeToken(t *testing.T) {
	t.Run("RevokeToken_Success", func(t *testing.T) {
		// 创建独立的服务和环境
		service, env, cleanup := setupAuthServiceForTest(t)
		defer cleanup()

		ctx := context.Background()

		// 创建测试用户
		users, err := env.DataManager.CreateTestUsers(1)
		require.NoError(t, err)
		testUser := users[0]

		// 生成Token
		jwtManager := auth.NewJWTManager("test-secret", time.Hour)
		token, err := jwtManager.GenerateToken(testUser.ID, testUser.Username)
		require.NoError(t, err)

		// 撤销Token
		err = service.RevokeToken(ctx, token)

		assert.NoError(t, err)
	})

	t.Run("RevokeToken_InvalidToken", func(t *testing.T) {
		// 创建独立的服务和环境
		service, _, cleanup := setupAuthServiceForTest(t)
		defer cleanup()

		ctx := context.Background()
		invalidToken := "invalid-token"

		err := service.RevokeToken(ctx, invalidToken)

		assert.Error(t, err)
	})
}

func TestAuthService_VerifyTokenInternal(t *testing.T) {
	t.Run("VerifyTokenInternal_Success", func(t *testing.T) {
		// 创建独立的服务和环境
		service, env, cleanup := setupAuthServiceForTest(t)
		defer cleanup()

		ctx := context.Background()

		// 创建测试用户
		users, err := env.DataManager.CreateTestUsers(1)
		require.NoError(t, err)
		testUser := users[0]

		// 生成Token
		jwtManager := auth.NewJWTManager("test-secret", time.Hour)
		token, err := jwtManager.GenerateToken(testUser.ID, testUser.Username)
		require.NoError(t, err)

		// 验证Token
		claims, err := service.VerifyTokenInternal(ctx, token)

		require.NoError(t, err)
		assert.NotNil(t, claims)
		assert.Equal(t, testUser.ID, claims.UserID)
		assert.Equal(t, testUser.Username, claims.Username)
	})

	t.Run("VerifyTokenInternal_InvalidToken", func(t *testing.T) {
		// 创建独立的服务和环境
		service, _, cleanup := setupAuthServiceForTest(t)
		defer cleanup()

		ctx := context.Background()
		invalidToken := "invalid-token"

		claims, err := service.VerifyTokenInternal(ctx, invalidToken)

		assert.Error(t, err)
		assert.Nil(t, claims)
	})

	t.Run("VerifyTokenInternal_ExpiredToken", func(t *testing.T) {
		// 创建独立的服务和环境
		service, env, cleanup := setupAuthServiceForTest(t)
		defer cleanup()

		ctx := context.Background()

		// 创建测试用户
		users, err := env.DataManager.CreateTestUsers(1)
		require.NoError(t, err)
		testUser := users[0]

		// 生成过期Token
		jwtManager := auth.NewJWTManager("test-secret", -time.Hour)
		token, err := jwtManager.GenerateToken(testUser.ID, testUser.Username)
		require.NoError(t, err)

		claims, err := service.VerifyTokenInternal(ctx, token)

		assert.Error(t, err)
		assert.Nil(t, claims)
	})
}

func TestAuthService_CheckTokenBlacklist(t *testing.T) {
	t.Run("CheckTokenBlacklist_NotBlacklisted", func(t *testing.T) {
		// 创建独立的服务和环境
		service, _, cleanup := setupAuthServiceForTest(t)
		defer cleanup()

		ctx := context.Background()
		tokenID := "clean-token-id"

		isBlacklisted, err := service.CheckTokenBlacklist(ctx, tokenID)

		require.NoError(t, err)
		assert.False(t, isBlacklisted)
	})

	t.Run("CheckTokenBlacklist_IsBlacklisted", func(t *testing.T) {
		// 创建独立的服务和环境
		service, env, cleanup := setupAuthServiceForTest(t)
		defer cleanup()

		ctx := context.Background()
		tokenID := "blacklisted-token-id"

		// 添加到黑名单
		err := env.DB.DB.Exec("INSERT INTO token_blacklist (token_id, expires_at, created_at) VALUES (?, ?, ?)",
			tokenID, time.Now().Add(time.Hour), time.Now()).Error
		require.NoError(t, err)

		isBlacklisted, err := service.CheckTokenBlacklist(ctx, tokenID)

		require.NoError(t, err)
		assert.True(t, isBlacklisted)
	})
}

func TestAuthService_GetUserSession(t *testing.T) {
	t.Run("GetUserSession_Success", func(t *testing.T) {
		// 创建独立的服务和环境
		service, env, cleanup := setupAuthServiceForTest(t)
		defer cleanup()

		ctx := context.Background()

		// 创建测试用户
		users, err := env.DataManager.CreateTestUsers(1)
		require.NoError(t, err)
		testUser := users[0]

		// 创建会话
		refreshToken := "test-refresh-token"
		expiresAt := time.Now().Add(time.Hour)
		err = env.DB.DB.Exec("INSERT INTO user_sessions (user_id, refresh_token, expires_at, created_at) VALUES (?, ?, ?, ?)",
			testUser.ID, refreshToken, expiresAt, time.Now()).Error
		require.NoError(t, err)

		// 获取会话
		err = service.GetUserSession(ctx, testUser.ID)

		assert.NoError(t, err)
	})

	t.Run("GetUserSession_NotFound", func(t *testing.T) {
		// 创建独立的服务和环境
		service, _, cleanup := setupAuthServiceForTest(t)
		defer cleanup()

		ctx := context.Background()

		err := service.GetUserSession(ctx, 99999)

		assert.Error(t, err)
	})

	t.Run("GetUserSession_Expired", func(t *testing.T) {
		// 创建独立的服务和环境
		service, env, cleanup := setupAuthServiceForTest(t)
		defer cleanup()

		ctx := context.Background()

		// 创建测试用户
		users, err := env.DataManager.CreateTestUsers(1)
		require.NoError(t, err)
		testUser := users[0]

		// 创建过期会话
		refreshToken := "expired-refresh-token"
		expiresAt := time.Now().Add(-time.Hour) // 已过期
		err = env.DB.DB.Exec("INSERT INTO user_sessions (user_id, refresh_token, expires_at, created_at) VALUES (?, ?, ?, ?)",
			testUser.ID, refreshToken, expiresAt, time.Now()).Error
		require.NoError(t, err)

		err = service.GetUserSession(ctx, testUser.ID)

		assert.Error(t, err)
	})
}

func TestAuthService_ValidateSession(t *testing.T) {
	t.Run("ValidateSession_Success", func(t *testing.T) {
		// 创建独立的服务和环境
		service, env, cleanup := setupAuthServiceForTest(t)
		defer cleanup()

		ctx := context.Background()

		// 创建测试用户
		users, err := env.DataManager.CreateTestUsers(1)
		require.NoError(t, err)
		testUser := users[0]

		refreshToken := "valid-refresh-token"
		expiresAt := time.Now().Add(time.Hour)

		// 创建会话
		err = env.DB.DB.Exec("INSERT INTO user_sessions (user_id, refresh_token, expires_at, created_at) VALUES (?, ?, ?, ?)",
			testUser.ID, refreshToken, expiresAt, time.Now()).Error
		require.NoError(t, err)

		// 验证会话
		isValid, err := service.ValidateSession(ctx, testUser.ID, refreshToken)

		require.NoError(t, err)
		assert.True(t, isValid)
	})

	t.Run("ValidateSession_TokenMismatch", func(t *testing.T) {
		// 创建独立的服务和环境
		service, env, cleanup := setupAuthServiceForTest(t)
		defer cleanup()

		ctx := context.Background()

		// 创建测试用户
		users, err := env.DataManager.CreateTestUsers(1)
		require.NoError(t, err)
		testUser := users[0]

		correctToken := "correct-token"
		wrongToken := "wrong-token"
		expiresAt := time.Now().Add(time.Hour)

		// 创建会话
		err = env.DB.DB.Exec("INSERT INTO user_sessions (user_id, refresh_token, expires_at, created_at) VALUES (?, ?, ?, ?)",
			testUser.ID, correctToken, expiresAt, time.Now()).Error
		require.NoError(t, err)

		// 验证错误Token
		isValid, err := service.ValidateSession(ctx, testUser.ID, wrongToken)

		require.NoError(t, err)
		assert.False(t, isValid)
	})

	t.Run("ValidateSession_SessionNotFound", func(t *testing.T) {
		// 创建独立的服务和环境
		service, _, cleanup := setupAuthServiceForTest(t)
		defer cleanup()

		ctx := context.Background()

		isValid, err := service.ValidateSession(ctx, 99999, "any-token")

		assert.Error(t, err)
		assert.False(t, isValid)
	})
}

func TestAuthService_RevokeAllUserTokens(t *testing.T) {
	t.Run("RevokeAllUserTokens_Success", func(t *testing.T) {
		// 创建独立的服务和环境
		service, env, cleanup := setupAuthServiceForTest(t)
		defer cleanup()

		ctx := context.Background()

		// 创建测试用户
		users, err := env.DataManager.CreateTestUsers(1)
		require.NoError(t, err)
		testUser := users[0]

		// 创建会话
		refreshToken := "test-refresh-token"
		expiresAt := time.Now().Add(time.Hour)
		err = env.DB.DB.Exec("INSERT INTO user_sessions (user_id, refresh_token, expires_at, created_at) VALUES (?, ?, ?, ?)",
			testUser.ID, refreshToken, expiresAt, time.Now()).Error
		require.NoError(t, err)

		// 撤销所有Token
		err = service.RevokeAllUserTokens(ctx, testUser.ID)

		assert.NoError(t, err)

		// 验证会话已删除
		var count int64
		err = env.DB.DB.Model(&data.UserSession{}).Where("user_id = ?", testUser.ID).Count(&count).Error
		require.NoError(t, err)
		assert.Equal(t, int64(0), count)
	})
}

func TestAuthService_GenerateTokenPair(t *testing.T) {
	t.Run("GenerateTokenPair_Success", func(t *testing.T) {
		// 创建独立的服务和环境
		service, env, cleanup := setupAuthServiceForTest(t)
		defer cleanup()

		ctx := context.Background()

		// 创建测试用户
		users, err := env.DataManager.CreateTestUsers(1)
		require.NoError(t, err)
		testUser := users[0]

		tokenPair, err := service.GenerateTokenPair(ctx, testUser.ID, testUser.Username)

		require.NoError(t, err)
		assert.NotNil(t, tokenPair)
		assert.NotEmpty(t, tokenPair.AccessToken)
		assert.NotEmpty(t, tokenPair.RefreshToken)
		assert.True(t, tokenPair.AccessExpiry.After(time.Now()))
		assert.True(t, tokenPair.RefreshExpiry.After(time.Now()))
	})
}

func TestAuthService_GetTokenExpiry(t *testing.T) {
	t.Run("GetTokenExpiry_Success", func(t *testing.T) {
		// 创建独立的服务和环境
		service, env, cleanup := setupAuthServiceForTest(t)
		defer cleanup()

		ctx := context.Background()

		// 创建测试用户
		users, err := env.DataManager.CreateTestUsers(1)
		require.NoError(t, err)
		testUser := users[0]

		// 生成Token
		jwtManager := auth.NewJWTManager("test-secret", time.Hour)
		token, err := jwtManager.GenerateToken(testUser.ID, testUser.Username)
		require.NoError(t, err)

		// 获取过期时间
		expiry, err := service.GetTokenExpiry(ctx, token)

		require.NoError(t, err)
		assert.True(t, expiry > time.Now().Unix())
	})

	t.Run("GetTokenExpiry_InvalidToken", func(t *testing.T) {
		// 创建独立的服务和环境
		service, _, cleanup := setupAuthServiceForTest(t)
		defer cleanup()

		ctx := context.Background()
		invalidToken := "invalid-token"

		expiry, err := service.GetTokenExpiry(ctx, invalidToken)

		assert.Error(t, err)
		assert.Equal(t, int64(0), expiry)
	})
}

// setupAuthServiceForTest 为每个测试创建独立的服务实例
func setupAuthServiceForTest(t *testing.T) (*AuthService, *testutils.TestEnv, func()) {
	env, cleanup, err := testutils.SetupTestWithCleanup()
	require.NoError(t, err)

	// 创建配置
	config := &conf.Data{
		Database: &conf.Data_Database{
			Driver:          "mysql",
			Source:          "tiktok:tiktok123@tcp(localhost:3307)/tiktok?charset=utf8mb4&parseTime=True&loc=Local",
			MaxIdleConns:    10,
			MaxOpenConns:    100,
			ConnMaxLifetime: durationpb.New(time.Hour),
		},
		Redis: &conf.Data_Redis{
			Addr:         "localhost:6381",
			Password:     "tiktok123",
			Db:           1,
			DialTimeout:  durationpb.New(5 * time.Second),
			ReadTimeout:  durationpb.New(3 * time.Second),
			WriteTimeout: durationpb.New(3 * time.Second),
			PoolSize:     100,
		},
	}

	// 创建Redis客户端
	rdb := redis.NewClient(&redis.Options{
		Addr:     config.Redis.Addr,
		Password: config.Redis.Password,
		DB:       int(config.Redis.Db),
	})

	// 重要：清理这个Redis客户端的数据
	ctx := context.Background()
	rdb.FlushDB(ctx)

	// 清理数据库
	env.DB.TruncateTable("user_sessions")
	env.DB.TruncateTable("token_blacklist")

	// 创建Data实例
	d, dataCleanup, err := data.NewData(config, log.DefaultLogger)
	require.NoError(t, err)

	// 创建缓存
	multiCache := pkgcache.NewMultiLevelCache(rdb, &pkgcache.CacheConfig{
		EnableL1: true,
		EnableL2: true,
	})
	userCache := cache.NewUserCache(multiCache, log.DefaultLogger)
	authCache := cache.NewAuthCache(multiCache, log.DefaultLogger)

	// 创建仓储
	passwordMgr := auth.NewPasswordManager()
	userRepo := data.NewUserRepo(d, userCache, passwordMgr, log.DefaultLogger)
	sessionRepo := data.NewSessionRepo(d, authCache, log.DefaultLogger)

	// 创建用例
	jwtManager := auth.NewJWTManager("test-secret", time.Hour)
	sessionMgr := auth.NewMemorySessionManager()
	authUc := biz.NewAuthUsecase(sessionRepo, userRepo, jwtManager, sessionMgr, log.DefaultLogger)

	// 创建服务
	service := NewAuthService(authUc, jwtManager, log.DefaultLogger)

	cleanupFunc := func() {
		// 清理Redis数据
		rdb.FlushDB(ctx)
		rdb.Close()

		// 清理数据库
		env.DB.TruncateTable("user_sessions")
		env.DB.TruncateTable("token_blacklist")

		dataCleanup()
		cleanup()
	}

	return service, env, cleanupFunc
}
