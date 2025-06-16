package biz

import (
	"context"
	"testing"
	"time"

	"go-backend/internal/domain"
	"go-backend/pkg/auth"
	"go-backend/testutils"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func setupAuthUsecase(t *testing.T) (*AuthUsecase, *MockAuthRepo, *MockUserRepo, *testutils.TestEnv, func()) {
	env, cleanup, err := testutils.SetupTestWithCleanup()
	require.NoError(t, err)

	authRepo := NewMockAuthRepo(t)
	userRepo := NewMockUserRepo(t)
	jwtManager := auth.NewJWTManager("test-secret", time.Hour)
	sessionMgr := auth.NewMemorySessionManager()
	logger := log.DefaultLogger

	uc := NewAuthUsecase(authRepo, userRepo, jwtManager, sessionMgr, logger)

	return uc, authRepo, userRepo, env, cleanup
}

func TestAuthUsecase_LoginWithToken(t *testing.T) {
	uc, authRepo, userRepo, env, cleanup := setupAuthUsecase(t)
	defer cleanup()

	ctx := context.Background()

	// 创建测试用户
	users, err := env.DataManager.CreateTestUsers(1)
	require.NoError(t, err)
	testUser := users[0]

	user := &User{
		ID:           testUser.ID,
		Username:     testUser.Username,
		PasswordHash: testUser.PasswordHash,
		Salt:         testUser.Salt,
		Nickname:     testUser.Nickname,
	}

	t.Run("LoginWithToken_Success", func(t *testing.T) {
		userRepo.EXPECT().VerifyPassword(ctx, "testuser1", "password1").Return(user, nil)
		userRepo.EXPECT().UpdateUser(ctx, mock.AnythingOfType("*biz.User")).Return(nil)
		authRepo.EXPECT().DeleteSession(ctx, testUser.ID).Return(nil)
		authRepo.EXPECT().CreateSession(ctx, mock.AnythingOfType("*domain.UserSession")).Return(nil)

		tokenPair, returnedUser, err := uc.LoginWithToken(ctx, "testuser1", "password1")

		require.NoError(t, err)
		assert.NotNil(t, tokenPair)
		assert.NotEmpty(t, tokenPair.AccessToken)
		assert.NotEmpty(t, tokenPair.RefreshToken)
		assert.Equal(t, user.ID, returnedUser.ID)
		assert.Equal(t, user.Username, returnedUser.Username)
	})

	t.Run("LoginWithToken_InvalidCredentials", func(t *testing.T) {
		userRepo.EXPECT().VerifyPassword(ctx, "testuser1", "wrongpassword").Return(nil, ErrPasswordError)

		tokenPair, returnedUser, err := uc.LoginWithToken(ctx, "testuser1", "wrongpassword")

		assert.Error(t, err)
		assert.Nil(t, tokenPair)
		assert.Nil(t, returnedUser)
		assert.Equal(t, ErrPasswordError, err)
	})

	t.Run("LoginWithToken_UserNotFound", func(t *testing.T) {
		userRepo.EXPECT().VerifyPassword(ctx, "nonexistent", "password").Return(nil, ErrUserNotFound)

		tokenPair, returnedUser, err := uc.LoginWithToken(ctx, "nonexistent", "password")

		assert.Error(t, err)
		assert.Nil(t, tokenPair)
		assert.Nil(t, returnedUser)
		assert.Equal(t, ErrUserNotFound, err)
	})
}

func TestAuthUsecase_RefreshToken(t *testing.T) {
	uc, authRepo, _, env, cleanup := setupAuthUsecase(t)
	defer cleanup()

	ctx := context.Background()

	// 创建测试用户
	users, err := env.DataManager.CreateTestUsers(1)
	require.NoError(t, err)
	testUser := users[0]

	t.Run("RefreshToken_Success", func(t *testing.T) {
		// 生成初始Token对
		jwtManager := auth.NewJWTManager("test-secret", time.Hour)
		tokenPair, err := jwtManager.GenerateTokenPair(testUser.ID, testUser.Username)
		require.NoError(t, err)

		// Mock session存在
		session := &domain.UserSession{
			UserID:       testUser.ID,
			RefreshToken: tokenPair.RefreshToken,
			ExpiresAt:    tokenPair.RefreshExpiry,
		}

		authRepo.EXPECT().GetSessionByToken(ctx, tokenPair.RefreshToken).Return(session, nil)
		authRepo.EXPECT().AddTokenToBlacklist(ctx, mock.AnythingOfType("string"), mock.AnythingOfType("time.Time")).Return(nil)
		authRepo.EXPECT().UpdateSession(ctx, testUser.ID, mock.AnythingOfType("string"), mock.AnythingOfType("time.Duration")).Return(nil)

		newTokenPair, err := uc.RefreshToken(ctx, tokenPair.RefreshToken)

		require.NoError(t, err)
		assert.NotNil(t, newTokenPair)
		assert.NotEmpty(t, newTokenPair.AccessToken)
		assert.NotEmpty(t, newTokenPair.RefreshToken)
		assert.NotEqual(t, tokenPair.AccessToken, newTokenPair.AccessToken)
		assert.NotEqual(t, tokenPair.RefreshToken, newTokenPair.RefreshToken)
	})

	t.Run("RefreshToken_InvalidToken", func(t *testing.T) {
		invalidToken := "invalid-refresh-token"

		newTokenPair, err := uc.RefreshToken(ctx, invalidToken)

		assert.Error(t, err)
		assert.Nil(t, newTokenPair)
	})

	t.Run("RefreshToken_SessionNotFound", func(t *testing.T) {
		jwtManager := auth.NewJWTManager("test-secret", time.Hour)
		tokenPair, err := jwtManager.GenerateTokenPair(testUser.ID, testUser.Username)
		require.NoError(t, err)

		authRepo.EXPECT().GetSessionByToken(ctx, tokenPair.RefreshToken).Return(nil, ErrSessionExpired)

		newTokenPair, err := uc.RefreshToken(ctx, tokenPair.RefreshToken)

		assert.Error(t, err)
		assert.Nil(t, newTokenPair)
		assert.Equal(t, ErrSessionExpired, err)
	})
}

func TestAuthUsecase_Logout(t *testing.T) {
	uc, authRepo, _, env, cleanup := setupAuthUsecase(t)
	defer cleanup()

	ctx := context.Background()

	// 创建测试用户
	users, err := env.DataManager.CreateTestUsers(1)
	require.NoError(t, err)
	testUser := users[0]

	t.Run("Logout_Success", func(t *testing.T) {
		jwtManager := auth.NewJWTManager("test-secret", time.Hour)
		accessToken, err := jwtManager.GenerateToken(testUser.ID, testUser.Username)
		require.NoError(t, err)

		tokenPair, err := jwtManager.GenerateTokenPair(testUser.ID, testUser.Username)
		require.NoError(t, err)

		authRepo.EXPECT().AddTokenToBlacklist(ctx, mock.AnythingOfType("string"), mock.AnythingOfType("time.Time")).Return(nil)
		authRepo.EXPECT().DeleteSession(ctx, testUser.ID).Return(nil)

		err = uc.Logout(ctx, testUser.ID, accessToken, tokenPair.RefreshToken)

		assert.NoError(t, err)
	})

	t.Run("Logout_WithoutRefreshToken", func(t *testing.T) {
		jwtManager := auth.NewJWTManager("test-secret", time.Hour)
		accessToken, err := jwtManager.GenerateToken(testUser.ID, testUser.Username)
		require.NoError(t, err)

		authRepo.EXPECT().AddTokenToBlacklist(ctx, mock.AnythingOfType("string"), mock.AnythingOfType("time.Time")).Return(nil)
		authRepo.EXPECT().DeleteSession(ctx, testUser.ID).Return(nil)

		err = uc.Logout(ctx, testUser.ID, accessToken, "")

		assert.NoError(t, err)
	})
}

func TestAuthUsecase_VerifyToken(t *testing.T) {
	uc, _, _, env, cleanup := setupAuthUsecase(t)
	defer cleanup()

	ctx := context.Background()

	// 创建测试用户
	users, err := env.DataManager.CreateTestUsers(1)
	require.NoError(t, err)
	testUser := users[0]

	t.Run("VerifyToken_Success", func(t *testing.T) {
		jwtManager := auth.NewJWTManager("test-secret", time.Hour)
		token, err := jwtManager.GenerateToken(testUser.ID, testUser.Username)
		require.NoError(t, err)

		claims, err := uc.VerifyToken(ctx, token)

		require.NoError(t, err)
		assert.Equal(t, testUser.ID, claims.UserID)
		assert.Equal(t, testUser.Username, claims.Username)
	})

	t.Run("VerifyToken_InvalidToken", func(t *testing.T) {
		invalidToken := "invalid-token"

		claims, err := uc.VerifyToken(ctx, invalidToken)

		assert.Error(t, err)
		assert.Nil(t, claims)
	})
}

func TestAuthUsecase_RevokeToken(t *testing.T) {
	uc, _, _, env, cleanup := setupAuthUsecase(t)
	defer cleanup()

	ctx := context.Background()

	// 创建测试用户
	users, err := env.DataManager.CreateTestUsers(1)
	require.NoError(t, err)
	testUser := users[0]

	t.Run("RevokeToken_Success", func(t *testing.T) {
		jwtManager := auth.NewJWTManager("test-secret", time.Hour)
		token, err := jwtManager.GenerateToken(testUser.ID, testUser.Username)
		require.NoError(t, err)

		err = uc.RevokeToken(ctx, token)

		assert.NoError(t, err)

		// 验证Token已被撤销
		_, err = uc.VerifyToken(ctx, token)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "blacklisted")
	})
}

func TestAuthUsecase_GetUserSession(t *testing.T) {
	uc, authRepo, _, env, cleanup := setupAuthUsecase(t)
	defer cleanup()

	ctx := context.Background()

	// 创建测试用户
	users, err := env.DataManager.CreateTestUsers(1)
	require.NoError(t, err)
	testUser := users[0]

	t.Run("GetUserSession_Success", func(t *testing.T) {
		expectedSession := &domain.UserSession{
			UserID:       testUser.ID,
			RefreshToken: "test-refresh-token",
			ExpiresAt:    time.Now().Add(time.Hour),
		}

		authRepo.EXPECT().GetSession(ctx, testUser.ID).Return(expectedSession, nil)

		session, err := uc.GetUserSession(ctx, testUser.ID)

		require.NoError(t, err)
		assert.Equal(t, expectedSession.UserID, session.UserID)
		assert.Equal(t, expectedSession.RefreshToken, session.RefreshToken)
	})

	t.Run("GetUserSession_NotFound", func(t *testing.T) {
		authRepo.EXPECT().GetSession(ctx, testUser.ID).Return(nil, ErrSessionExpired)

		session, err := uc.GetUserSession(ctx, testUser.ID)

		assert.Error(t, err)
		assert.Nil(t, session)
		assert.Equal(t, ErrSessionExpired, err)
	})
}

func TestAuthUsecase_ValidateSession(t *testing.T) {
	uc, authRepo, _, env, cleanup := setupAuthUsecase(t)
	defer cleanup()

	ctx := context.Background()

	// 创建测试用户
	users, err := env.DataManager.CreateTestUsers(1)
	require.NoError(t, err)
	testUser := users[0]

	t.Run("ValidateSession_Success", func(t *testing.T) {
		refreshToken := "valid-refresh-token"
		session := &domain.UserSession{
			UserID:       testUser.ID,
			RefreshToken: refreshToken,
			ExpiresAt:    time.Now().Add(time.Hour),
		}

		authRepo.EXPECT().GetSession(ctx, testUser.ID).Return(session, nil)

		isValid, err := uc.ValidateSession(ctx, testUser.ID, refreshToken)

		require.NoError(t, err)
		assert.True(t, isValid)
	})

	t.Run("ValidateSession_TokenMismatch", func(t *testing.T) {
		refreshToken := "valid-refresh-token"
		wrongToken := "wrong-refresh-token"
		session := &domain.UserSession{
			UserID:       testUser.ID,
			RefreshToken: refreshToken,
			ExpiresAt:    time.Now().Add(time.Hour),
		}

		authRepo.EXPECT().GetSession(ctx, testUser.ID).Return(session, nil)

		isValid, err := uc.ValidateSession(ctx, testUser.ID, wrongToken)

		require.NoError(t, err)
		assert.False(t, isValid)
	})

	t.Run("ValidateSession_SessionNotFound", func(t *testing.T) {
		authRepo.EXPECT().GetSession(ctx, testUser.ID).Return(nil, ErrSessionExpired)

		isValid, err := uc.ValidateSession(ctx, testUser.ID, "any-token")

		assert.Error(t, err)
		assert.False(t, isValid)
	})
}

func TestAuthUsecase_CheckTokenBlacklist(t *testing.T) {
	uc, authRepo, _, _, cleanup := setupAuthUsecase(t)
	defer cleanup()

	ctx := context.Background()

	t.Run("CheckTokenBlacklist_IsBlacklisted", func(t *testing.T) {
		tokenID := "blacklisted-token"

		authRepo.EXPECT().IsTokenBlacklisted(ctx, tokenID).Return(true, nil)

		isBlacklisted, err := uc.CheckTokenBlacklist(ctx, tokenID)

		require.NoError(t, err)
		assert.True(t, isBlacklisted)
	})

	t.Run("CheckTokenBlacklist_NotBlacklisted", func(t *testing.T) {
		tokenID := "clean-token"

		authRepo.EXPECT().IsTokenBlacklisted(ctx, tokenID).Return(false, nil)

		isBlacklisted, err := uc.CheckTokenBlacklist(ctx, tokenID)

		require.NoError(t, err)
		assert.False(t, isBlacklisted)
	})
}

func TestAuthUsecase_RevokeAllUserTokens(t *testing.T) {
	uc, authRepo, _, env, cleanup := setupAuthUsecase(t)
	defer cleanup()

	ctx := context.Background()

	// 创建测试用户
	users, err := env.DataManager.CreateTestUsers(1)
	require.NoError(t, err)
	testUser := users[0]

	t.Run("RevokeAllUserTokens_Success", func(t *testing.T) {
		authRepo.EXPECT().DeleteSession(ctx, testUser.ID).Return(nil)

		err := uc.RevokeAllUserTokens(ctx, testUser.ID)

		assert.NoError(t, err)
	})
}
