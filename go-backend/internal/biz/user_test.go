package biz

import (
	"context"
	"testing"

	"go-backend/testutils"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func setupUserUsecase(t *testing.T) (*UserUsecase, *MockUserRepo, *testutils.TestEnv, func()) {
	env, cleanup, err := testutils.SetupTestWithCleanup()
	require.NoError(t, err)

	userRepo := NewMockUserRepo(t)
	logger := log.DefaultLogger

	uc := NewUserUsecase(userRepo, logger)

	return uc, userRepo, env, cleanup
}

func TestUserUsecase_Register(t *testing.T) {
	uc, userRepo, _, cleanup := setupUserUsecase(t)
	defer cleanup()

	ctx := context.Background()

	t.Run("Register_Success", func(t *testing.T) {
		username := "testuser"
		password := "Password123!"

		expectedUser := &User{
			ID:       1,
			Username: username,
			Nickname: username,
			Avatar:   "https://example.com/default-avatar.jpg",
		}

		// Mock检查用户不存在
		userRepo.EXPECT().GetUserByUsername(ctx, username).Return(nil, ErrUserNotFound)
		// Mock创建用户成功
		userRepo.EXPECT().CreateUser(ctx, mock.AnythingOfType("*biz.User")).Return(expectedUser, nil)

		user, err := uc.Register(ctx, username, password)

		require.NoError(t, err)
		assert.Equal(t, expectedUser.ID, user.ID)
		assert.Equal(t, expectedUser.Username, user.Username)
		assert.Equal(t, expectedUser.Nickname, user.Nickname)
	})

	t.Run("Register_UserExists", func(t *testing.T) {
		username := "existinguser"
		password := "Password123!"

		existingUser := &User{
			ID:       1,
			Username: username,
		}

		userRepo.EXPECT().GetUserByUsername(ctx, username).Return(existingUser, nil)

		user, err := uc.Register(ctx, username, password)

		assert.Error(t, err)
		assert.Nil(t, user)
		assert.Equal(t, ErrUserExist, err)
	})

	t.Run("Register_CreateUserFailed", func(t *testing.T) {
		username := "newuser"
		password := "Password123!"

		userRepo.EXPECT().GetUserByUsername(ctx, username).Return(nil, ErrUserNotFound)
		userRepo.EXPECT().CreateUser(ctx, mock.AnythingOfType("*biz.User")).Return(nil, assert.AnError)

		user, err := uc.Register(ctx, username, password)

		assert.Error(t, err)
		assert.Nil(t, user)
	})
}

func TestUserUsecase_Login(t *testing.T) {
	uc, userRepo, _, cleanup := setupUserUsecase(t)
	defer cleanup()

	ctx := context.Background()

	t.Run("Login_Success", func(t *testing.T) {
		username := "testuser"
		password := "Password123!"

		expectedUser := &User{
			ID:       1,
			Username: username,
			Nickname: "Test User",
		}

		userRepo.EXPECT().VerifyPassword(ctx, username, password).Return(expectedUser, nil)
		userRepo.EXPECT().UpdateUser(ctx, mock.AnythingOfType("*biz.User")).Return(nil)

		user, err := uc.Login(ctx, username, password)

		require.NoError(t, err)
		assert.Equal(t, expectedUser.ID, user.ID)
		assert.Equal(t, expectedUser.Username, user.Username)
		assert.NotNil(t, user.LastLoginAt)
	})

	t.Run("Login_WrongPassword", func(t *testing.T) {
		username := "testuser"
		password := "wrongpassword"

		userRepo.EXPECT().VerifyPassword(ctx, username, password).Return(nil, ErrPasswordError)

		user, err := uc.Login(ctx, username, password)

		assert.Error(t, err)
		assert.Nil(t, user)
		assert.Equal(t, ErrPasswordError, err)
	})

	t.Run("Login_UserNotFound", func(t *testing.T) {
		username := "nonexistent"
		password := "Password123!"

		userRepo.EXPECT().VerifyPassword(ctx, username, password).Return(nil, ErrUserNotFound)

		user, err := uc.Login(ctx, username, password)

		assert.Error(t, err)
		assert.Nil(t, user)
		assert.Equal(t, ErrUserNotFound, err)
	})
}

func TestUserUsecase_GetUser(t *testing.T) {
	uc, userRepo, _, cleanup := setupUserUsecase(t)
	defer cleanup()

	ctx := context.Background()

	t.Run("GetUser_Success", func(t *testing.T) {
		userID := int64(1)

		expectedUser := &User{
			ID:       userID,
			Username: "testuser",
			Nickname: "Test User",
		}

		userRepo.EXPECT().GetUser(ctx, userID).Return(expectedUser, nil)

		user, err := uc.GetUser(ctx, userID)

		require.NoError(t, err)
		assert.Equal(t, expectedUser.ID, user.ID)
		assert.Equal(t, expectedUser.Username, user.Username)
	})

	t.Run("GetUser_NotFound", func(t *testing.T) {
		userID := int64(999)

		userRepo.EXPECT().GetUser(ctx, userID).Return(nil, ErrUserNotFound)

		user, err := uc.GetUser(ctx, userID)

		assert.Error(t, err)
		assert.Nil(t, user)
		assert.Equal(t, ErrUserNotFound, err)
	})
}

func TestUserUsecase_GetUsers(t *testing.T) {
	uc, userRepo, _, cleanup := setupUserUsecase(t)
	defer cleanup()

	ctx := context.Background()

	t.Run("GetUsers_Success", func(t *testing.T) {
		userIDs := []int64{1, 2, 3}

		expectedUsers := []*User{
			{ID: 1, Username: "user1", Nickname: "User 1"},
			{ID: 2, Username: "user2", Nickname: "User 2"},
			{ID: 3, Username: "user3", Nickname: "User 3"},
		}

		userRepo.EXPECT().GetUsers(ctx, userIDs).Return(expectedUsers, nil)

		users, err := uc.GetUsers(ctx, userIDs)

		require.NoError(t, err)
		assert.Len(t, users, 3)
		assert.Equal(t, expectedUsers[0].ID, users[0].ID)
		assert.Equal(t, expectedUsers[1].ID, users[1].ID)
		assert.Equal(t, expectedUsers[2].ID, users[2].ID)
	})

	t.Run("GetUsers_Empty", func(t *testing.T) {
		userIDs := []int64{}

		userRepo.EXPECT().GetUsers(ctx, userIDs).Return([]*User{}, nil)

		users, err := uc.GetUsers(ctx, userIDs)

		require.NoError(t, err)
		assert.Empty(t, users)
	})
}

func TestUserUsecase_UpdateUser(t *testing.T) {
	uc, userRepo, _, cleanup := setupUserUsecase(t)
	defer cleanup()

	ctx := context.Background()

	t.Run("UpdateUser_Success", func(t *testing.T) {
		user := &User{
			ID:       1,
			Username: "testuser",
			Nickname: "Updated User",
		}

		userRepo.EXPECT().UpdateUser(ctx, user).Return(nil)

		err := uc.UpdateUser(ctx, user)

		assert.NoError(t, err)
	})

	t.Run("UpdateUser_Failed", func(t *testing.T) {
		user := &User{
			ID:       999,
			Username: "testuser",
			Nickname: "Updated User",
		}

		userRepo.EXPECT().UpdateUser(ctx, user).Return(assert.AnError)

		err := uc.UpdateUser(ctx, user)

		assert.Error(t, err)
	})
}

func TestUserUsecase_GetUserByUsername(t *testing.T) {
	uc, userRepo, _, cleanup := setupUserUsecase(t)
	defer cleanup()

	ctx := context.Background()

	t.Run("GetUserByUsername_Success", func(t *testing.T) {
		username := "testuser"

		expectedUser := &User{
			ID:       1,
			Username: username,
			Nickname: "Test User",
		}

		userRepo.EXPECT().GetUserByUsername(ctx, username).Return(expectedUser, nil)

		user, err := uc.GetUserByUsername(ctx, username)

		require.NoError(t, err)
		assert.Equal(t, expectedUser.ID, user.ID)
		assert.Equal(t, expectedUser.Username, user.Username)
	})

	t.Run("GetUserByUsername_NotFound", func(t *testing.T) {
		username := "nonexistent"

		userRepo.EXPECT().GetUserByUsername(ctx, username).Return(nil, ErrUserNotFound)

		user, err := uc.GetUserByUsername(ctx, username)

		assert.Error(t, err)
		assert.Nil(t, user)
		assert.Equal(t, ErrUserNotFound, err)
	})
}

func TestUserUsecase_UpdateUserStats(t *testing.T) {
	uc, userRepo, _, cleanup := setupUserUsecase(t)
	defer cleanup()

	ctx := context.Background()

	t.Run("UpdateUserStats_Success", func(t *testing.T) {
		userID := int64(1)
		stats := &UserStats{
			FollowCountDelta:    1,
			FollowerCountDelta:  2,
			WorkCountDelta:      1,
			FavoriteCountDelta:  3,
			TotalFavoritedDelta: 5,
		}

		userRepo.EXPECT().UpdateUserStats(ctx, userID, stats).Return(nil)

		err := uc.UpdateUserStats(ctx, userID, stats)

		assert.NoError(t, err)
	})

	t.Run("UpdateUserStats_Failed", func(t *testing.T) {
		userID := int64(999)
		stats := &UserStats{
			FollowCountDelta: 1,
		}

		userRepo.EXPECT().UpdateUserStats(ctx, userID, stats).Return(assert.AnError)

		err := uc.UpdateUserStats(ctx, userID, stats)

		assert.Error(t, err)
	})
}

func TestUserUsecase_ChangePassword(t *testing.T) {
	uc, userRepo, _, cleanup := setupUserUsecase(t)
	defer cleanup()

	ctx := context.Background()

	t.Run("ChangePassword_Success", func(t *testing.T) {
		userID := int64(1)
		oldPassword := "OldPassword123!"
		newPassword := "NewPassword123!"

		user := &User{
			ID:       userID,
			Username: "testuser",
		}

		userRepo.EXPECT().GetUser(ctx, userID).Return(user, nil)
		userRepo.EXPECT().VerifyPassword(ctx, user.Username, oldPassword).Return(user, nil)
		userRepo.EXPECT().UpdateUser(ctx, mock.AnythingOfType("*biz.User")).Return(nil)

		err := uc.ChangePassword(ctx, userID, oldPassword, newPassword)

		assert.NoError(t, err)
	})

	t.Run("ChangePassword_UserNotFound", func(t *testing.T) {
		userID := int64(999)
		oldPassword := "OldPassword123!"
		newPassword := "NewPassword123!"

		userRepo.EXPECT().GetUser(ctx, userID).Return(nil, ErrUserNotFound)

		err := uc.ChangePassword(ctx, userID, oldPassword, newPassword)

		assert.Error(t, err)
		assert.Equal(t, ErrUserNotFound, err)
	})

	t.Run("ChangePassword_WrongOldPassword", func(t *testing.T) {
		userID := int64(1)
		oldPassword := "WrongPassword123!"
		newPassword := "NewPassword123!"

		user := &User{
			ID:       userID,
			Username: "testuser",
		}

		userRepo.EXPECT().GetUser(ctx, userID).Return(user, nil)
		userRepo.EXPECT().VerifyPassword(ctx, user.Username, oldPassword).Return(nil, ErrPasswordError)

		err := uc.ChangePassword(ctx, userID, oldPassword, newPassword)

		assert.Error(t, err)
		assert.Equal(t, ErrPasswordError, err)
	})
}

func TestUserUsecase_UpdateProfile(t *testing.T) {
	uc, userRepo, _, cleanup := setupUserUsecase(t)
	defer cleanup()

	ctx := context.Background()

	t.Run("UpdateProfile_Success", func(t *testing.T) {
		userID := int64(1)
		nickname := "New Nickname"
		avatar := "https://example.com/new-avatar.jpg"
		backgroundImage := "https://example.com/new-bg.jpg"
		signature := "New signature"

		user := &User{
			ID:       userID,
			Username: "testuser",
			Nickname: "Old Nickname",
		}

		userRepo.EXPECT().GetUser(ctx, userID).Return(user, nil)
		userRepo.EXPECT().UpdateUser(ctx, mock.MatchedBy(func(u *User) bool {
			return u.ID == userID &&
				u.Nickname == nickname &&
				u.Avatar == avatar &&
				u.BackgroundImage == backgroundImage &&
				u.Signature == signature
		})).Return(nil)

		err := uc.UpdateProfile(ctx, userID, nickname, avatar, backgroundImage, signature)

		assert.NoError(t, err)
	})

	t.Run("UpdateProfile_UserNotFound", func(t *testing.T) {
		userID := int64(999)

		userRepo.EXPECT().GetUser(ctx, userID).Return(nil, ErrUserNotFound)

		err := uc.UpdateProfile(ctx, userID, "nickname", "", "", "")

		assert.Error(t, err)
		assert.Equal(t, ErrUserNotFound, err)
	})

	t.Run("UpdateProfile_PartialUpdate", func(t *testing.T) {
		userID := int64(1)
		nickname := "New Nickname"

		user := &User{
			ID:              userID,
			Username:        "testuser",
			Nickname:        "Old Nickname",
			Avatar:          "old-avatar.jpg",
			BackgroundImage: "old-bg.jpg",
			Signature:       "old signature",
		}

		userRepo.EXPECT().GetUser(ctx, userID).Return(user, nil)
		userRepo.EXPECT().UpdateUser(ctx, mock.MatchedBy(func(u *User) bool {
			return u.ID == userID &&
				u.Nickname == nickname &&
				u.Avatar == "old-avatar.jpg" &&
				u.BackgroundImage == "old-bg.jpg" &&
				u.Signature == "old signature"
		})).Return(nil)

		err := uc.UpdateProfile(ctx, userID, nickname, "", "", "")

		assert.NoError(t, err)
	})
}

func TestUser_IsActive(t *testing.T) {
	user := &User{
		ID:       1,
		Username: "testuser",
	}

	// 当前实现总是返回true
	assert.True(t, user.IsActive())
}
