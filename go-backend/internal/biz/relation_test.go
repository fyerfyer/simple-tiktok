package biz

import (
	"context"
	"testing"

	"go-backend/testutils"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRelationUsecase_Follow(t *testing.T) {
	_, cleanup, err := testutils.SetupTestWithCleanup()
	require.NoError(t, err)
	defer cleanup()

	ctx := context.Background()

	t.Run("Follow_Success", func(t *testing.T) {
		// 创建独立的mock和usecase
		relationRepo := NewMockRelationRepo(t)
		uc := NewRelationUsecase(relationRepo, log.DefaultLogger)

		userID := int64(1)
		followUserID := int64(2)

		relationRepo.EXPECT().Follow(ctx, userID, followUserID).Return(nil)

		err := uc.Follow(ctx, userID, followUserID)

		assert.NoError(t, err)
	})

	t.Run("Follow_SelfFollow", func(t *testing.T) {
		// 创建独立的mock和usecase
		relationRepo := NewMockRelationRepo(t)
		uc := NewRelationUsecase(relationRepo, log.DefaultLogger)

		userID := int64(1)

		err := uc.Follow(ctx, userID, userID)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot follow yourself")
	})

	t.Run("Follow_AlreadyFollowing", func(t *testing.T) {
		// 创建独立的mock和usecase
		relationRepo := NewMockRelationRepo(t)
		uc := NewRelationUsecase(relationRepo, log.DefaultLogger)

		userID := int64(1)
		followUserID := int64(2)

		relationRepo.EXPECT().Follow(ctx, userID, followUserID).Return(ErrAlreadyFollow)

		err := uc.Follow(ctx, userID, followUserID)

		assert.Error(t, err)
		assert.Equal(t, ErrAlreadyFollow, err)
	})

	t.Run("Follow_DatabaseError", func(t *testing.T) {
		// 创建独立的mock和usecase
		relationRepo := NewMockRelationRepo(t)
		uc := NewRelationUsecase(relationRepo, log.DefaultLogger)

		userID := int64(1)
		followUserID := int64(2)

		relationRepo.EXPECT().Follow(ctx, userID, followUserID).Return(assert.AnError)

		err := uc.Follow(ctx, userID, followUserID)

		assert.Error(t, err)
		assert.Equal(t, assert.AnError, err)
	})
}

func TestRelationUsecase_Unfollow(t *testing.T) {
	_, cleanup, err := testutils.SetupTestWithCleanup()
	require.NoError(t, err)
	defer cleanup()

	ctx := context.Background()

	t.Run("Unfollow_Success", func(t *testing.T) {
		// 创建独立的mock和usecase
		relationRepo := NewMockRelationRepo(t)
		uc := NewRelationUsecase(relationRepo, log.DefaultLogger)

		userID := int64(1)
		followUserID := int64(2)

		relationRepo.EXPECT().Unfollow(ctx, userID, followUserID).Return(nil)

		err := uc.Unfollow(ctx, userID, followUserID)

		assert.NoError(t, err)
	})

	t.Run("Unfollow_NotFollowing", func(t *testing.T) {
		// 创建独立的mock和usecase
		relationRepo := NewMockRelationRepo(t)
		uc := NewRelationUsecase(relationRepo, log.DefaultLogger)

		userID := int64(1)
		followUserID := int64(2)

		relationRepo.EXPECT().Unfollow(ctx, userID, followUserID).Return(ErrNotFollow)

		err := uc.Unfollow(ctx, userID, followUserID)

		assert.Error(t, err)
		assert.Equal(t, ErrNotFollow, err)
	})

	t.Run("Unfollow_DatabaseError", func(t *testing.T) {
		// 创建独立的mock和usecase
		relationRepo := NewMockRelationRepo(t)
		uc := NewRelationUsecase(relationRepo, log.DefaultLogger)

		userID := int64(1)
		followUserID := int64(2)

		relationRepo.EXPECT().Unfollow(ctx, userID, followUserID).Return(assert.AnError)

		err := uc.Unfollow(ctx, userID, followUserID)

		assert.Error(t, err)
		assert.Equal(t, assert.AnError, err)
	})
}

func TestRelationUsecase_IsFollowing(t *testing.T) {
	_, cleanup, err := testutils.SetupTestWithCleanup()
	require.NoError(t, err)
	defer cleanup()

	ctx := context.Background()

	t.Run("IsFollowing_True", func(t *testing.T) {
		// 创建独立的mock和usecase
		relationRepo := NewMockRelationRepo(t)
		uc := NewRelationUsecase(relationRepo, log.DefaultLogger)

		userID := int64(1)
		followUserID := int64(2)

		relationRepo.EXPECT().IsFollowing(ctx, userID, followUserID).Return(true, nil)

		isFollowing, err := uc.IsFollowing(ctx, userID, followUserID)

		require.NoError(t, err)
		assert.True(t, isFollowing)
	})

	t.Run("IsFollowing_False", func(t *testing.T) {
		// 创建独立的mock和usecase
		relationRepo := NewMockRelationRepo(t)
		uc := NewRelationUsecase(relationRepo, log.DefaultLogger)

		userID := int64(1)
		followUserID := int64(2)

		relationRepo.EXPECT().IsFollowing(ctx, userID, followUserID).Return(false, nil)

		isFollowing, err := uc.IsFollowing(ctx, userID, followUserID)

		require.NoError(t, err)
		assert.False(t, isFollowing)
	})

	t.Run("IsFollowing_DatabaseError", func(t *testing.T) {
		// 创建独立的mock和usecase
		relationRepo := NewMockRelationRepo(t)
		uc := NewRelationUsecase(relationRepo, log.DefaultLogger)

		userID := int64(1)
		followUserID := int64(2)

		relationRepo.EXPECT().IsFollowing(ctx, userID, followUserID).Return(false, assert.AnError)

		isFollowing, err := uc.IsFollowing(ctx, userID, followUserID)

		assert.Error(t, err)
		assert.False(t, isFollowing)
	})
}

func TestRelationUsecase_GetFollowList(t *testing.T) {
	_, cleanup, err := testutils.SetupTestWithCleanup()
	require.NoError(t, err)
	defer cleanup()

	ctx := context.Background()

	t.Run("GetFollowList_Success", func(t *testing.T) {
		// 创建独立的mock和usecase
		relationRepo := NewMockRelationRepo(t)
		uc := NewRelationUsecase(relationRepo, log.DefaultLogger)

		userID := int64(1)
		page := int32(1)
		size := int32(20)

		expectedUsers := []*User{
			{ID: 2, Username: "user2", Nickname: "User 2", IsFollow: true},
			{ID: 3, Username: "user3", Nickname: "User 3", IsFollow: true},
		}
		expectedTotal := int64(2)

		relationRepo.EXPECT().GetFollowList(ctx, userID, page, size).Return(expectedUsers, expectedTotal, nil)

		users, total, err := uc.GetFollowList(ctx, userID, page, size)

		require.NoError(t, err)
		assert.Len(t, users, 2)
		assert.Equal(t, expectedTotal, total)
		assert.Equal(t, expectedUsers[0].ID, users[0].ID)
		assert.True(t, users[0].IsFollow)
	})

	t.Run("GetFollowList_DefaultPagination", func(t *testing.T) {
		// 创建独立的mock和usecase
		relationRepo := NewMockRelationRepo(t)
		uc := NewRelationUsecase(relationRepo, log.DefaultLogger)

		userID := int64(1)
		page := int32(0) // 应该被修正为1
		size := int32(0) // 应该被修正为20

		expectedUsers := []*User{}
		expectedTotal := int64(0)

		relationRepo.EXPECT().GetFollowList(ctx, userID, int32(1), int32(20)).Return(expectedUsers, expectedTotal, nil)

		users, total, err := uc.GetFollowList(ctx, userID, page, size)

		require.NoError(t, err)
		assert.Empty(t, users)
		assert.Equal(t, expectedTotal, total)
	})

	t.Run("GetFollowList_LargePageSize", func(t *testing.T) {
		// 创建独立的mock和usecase
		relationRepo := NewMockRelationRepo(t)
		uc := NewRelationUsecase(relationRepo, log.DefaultLogger)

		userID := int64(1)
		page := int32(1)
		size := int32(100) // 应该被修正为20

		expectedUsers := []*User{}
		expectedTotal := int64(0)

		relationRepo.EXPECT().GetFollowList(ctx, userID, page, int32(20)).Return(expectedUsers, expectedTotal, nil)

		users, total, err := uc.GetFollowList(ctx, userID, page, size)

		require.NoError(t, err)
		assert.Empty(t, users)
		assert.Equal(t, expectedTotal, total)
	})

	t.Run("GetFollowList_DatabaseError", func(t *testing.T) {
		// 创建独立的mock和usecase
		relationRepo := NewMockRelationRepo(t)
		uc := NewRelationUsecase(relationRepo, log.DefaultLogger)

		userID := int64(1)
		page := int32(1)
		size := int32(20)

		relationRepo.EXPECT().GetFollowList(ctx, userID, page, size).Return(nil, int64(0), assert.AnError)

		users, total, err := uc.GetFollowList(ctx, userID, page, size)

		assert.Error(t, err)
		assert.Nil(t, users)
		assert.Equal(t, int64(0), total)
	})
}

func TestRelationUsecase_GetFollowerList(t *testing.T) {
	_, cleanup, err := testutils.SetupTestWithCleanup()
	require.NoError(t, err)
	defer cleanup()

	ctx := context.Background()

	t.Run("GetFollowerList_Success", func(t *testing.T) {
		// 创建独立的mock和usecase
		relationRepo := NewMockRelationRepo(t)
		uc := NewRelationUsecase(relationRepo, log.DefaultLogger)

		userID := int64(1)
		page := int32(1)
		size := int32(20)

		expectedUsers := []*User{
			{ID: 2, Username: "user2", Nickname: "User 2", IsFollow: false},
			{ID: 3, Username: "user3", Nickname: "User 3", IsFollow: true},
		}
		expectedTotal := int64(2)

		relationRepo.EXPECT().GetFollowerList(ctx, userID, page, size).Return(expectedUsers, expectedTotal, nil)

		users, total, err := uc.GetFollowerList(ctx, userID, page, size)

		require.NoError(t, err)
		assert.Len(t, users, 2)
		assert.Equal(t, expectedTotal, total)
		assert.Equal(t, expectedUsers[0].ID, users[0].ID)
		assert.False(t, users[0].IsFollow)
		assert.True(t, users[1].IsFollow)
	})

	t.Run("GetFollowerList_DefaultPagination", func(t *testing.T) {
		// 创建独立的mock和usecase
		relationRepo := NewMockRelationRepo(t)
		uc := NewRelationUsecase(relationRepo, log.DefaultLogger)

		userID := int64(1)
		page := int32(-1) // 应该被修正为1
		size := int32(0)  // 应该被修正为20

		expectedUsers := []*User{}
		expectedTotal := int64(0)

		relationRepo.EXPECT().GetFollowerList(ctx, userID, int32(1), int32(20)).Return(expectedUsers, expectedTotal, nil)

		users, total, err := uc.GetFollowerList(ctx, userID, page, size)

		require.NoError(t, err)
		assert.Empty(t, users)
		assert.Equal(t, expectedTotal, total)
	})

	t.Run("GetFollowerList_DatabaseError", func(t *testing.T) {
		// 创建独立的mock和usecase
		relationRepo := NewMockRelationRepo(t)
		uc := NewRelationUsecase(relationRepo, log.DefaultLogger)

		userID := int64(1)
		page := int32(1)
		size := int32(20)

		relationRepo.EXPECT().GetFollowerList(ctx, userID, page, size).Return(nil, int64(0), assert.AnError)

		users, total, err := uc.GetFollowerList(ctx, userID, page, size)

		assert.Error(t, err)
		assert.Nil(t, users)
		assert.Equal(t, int64(0), total)
	})
}

func TestRelationUsecase_GetFriendList(t *testing.T) {
	_, cleanup, err := testutils.SetupTestWithCleanup()
	require.NoError(t, err)
	defer cleanup()

	ctx := context.Background()

	t.Run("GetFriendList_Success", func(t *testing.T) {
		// 创建独立的mock和usecase
		relationRepo := NewMockRelationRepo(t)
		uc := NewRelationUsecase(relationRepo, log.DefaultLogger)

		userID := int64(1)

		expectedUsers := []*User{
			{ID: 2, Username: "user2", Nickname: "User 2", IsFollow: true},
			{ID: 3, Username: "user3", Nickname: "User 3", IsFollow: true},
		}

		relationRepo.EXPECT().GetFriendList(ctx, userID).Return(expectedUsers, nil)

		users, err := uc.GetFriendList(ctx, userID)

		require.NoError(t, err)
		assert.Len(t, users, 2)
		assert.Equal(t, expectedUsers[0].ID, users[0].ID)
		assert.True(t, users[0].IsFollow)
		assert.True(t, users[1].IsFollow)
	})

	t.Run("GetFriendList_Empty", func(t *testing.T) {
		// 创建独立的mock和usecase
		relationRepo := NewMockRelationRepo(t)
		uc := NewRelationUsecase(relationRepo, log.DefaultLogger)

		userID := int64(1)

		relationRepo.EXPECT().GetFriendList(ctx, userID).Return([]*User{}, nil)

		users, err := uc.GetFriendList(ctx, userID)

		require.NoError(t, err)
		assert.Empty(t, users)
	})

	t.Run("GetFriendList_DatabaseError", func(t *testing.T) {
		// 创建独立的mock和usecase
		relationRepo := NewMockRelationRepo(t)
		uc := NewRelationUsecase(relationRepo, log.DefaultLogger)

		userID := int64(1)

		relationRepo.EXPECT().GetFriendList(ctx, userID).Return(nil, assert.AnError)

		users, err := uc.GetFriendList(ctx, userID)

		assert.Error(t, err)
		assert.Nil(t, users)
	})
}

func TestRelationUsecase_ErrorTypes(t *testing.T) {
	t.Run("ErrAlreadyFollow", func(t *testing.T) {
		err := ErrAlreadyFollow
		assert.NotNil(t, err)
		assert.Contains(t, err.Error(), "already followed")
	})

	t.Run("ErrNotFollow", func(t *testing.T) {
		err := ErrNotFollow
		assert.NotNil(t, err)
		assert.Contains(t, err.Error(), "not followed")
	})
}

func TestRelationUsecase_PaginationValidation(t *testing.T) {
	_, cleanup, err := testutils.SetupTestWithCleanup()
	require.NoError(t, err)
	defer cleanup()

	ctx := context.Background()
	userID := int64(1)

	testCases := []struct {
		name         string
		inputPage    int32
		inputSize    int32
		expectedPage int32
		expectedSize int32
	}{
		{"NormalValues", 2, 10, 2, 10},
		{"ZeroPage", 0, 10, 1, 10},
		{"NegativePage", -1, 10, 1, 10},
		{"ZeroSize", 1, 0, 1, 20},
		{"NegativeSize", 1, -1, 1, 20},
		{"LargeSize", 1, 100, 1, 20},
		{"MaxSize", 1, 50, 1, 50},
		{"OverMaxSize", 1, 51, 1, 20},
	}

	for _, tc := range testCases {
		t.Run(tc.name+"_FollowList", func(t *testing.T) {
			// 为每个测试用例创建独立的mock
			relationRepo := NewMockRelationRepo(t)
			uc := NewRelationUsecase(relationRepo, log.DefaultLogger)

			relationRepo.EXPECT().GetFollowList(ctx, userID, tc.expectedPage, tc.expectedSize).Return([]*User{}, int64(0), nil)

			_, _, err := uc.GetFollowList(ctx, userID, tc.inputPage, tc.inputSize)
			assert.NoError(t, err)
		})

		t.Run(tc.name+"_FollowerList", func(t *testing.T) {
			// 为每个测试用例创建独立的mock
			relationRepo := NewMockRelationRepo(t)
			uc := NewRelationUsecase(relationRepo, log.DefaultLogger)

			relationRepo.EXPECT().GetFollowerList(ctx, userID, tc.expectedPage, tc.expectedSize).Return([]*User{}, int64(0), nil)

			_, _, err := uc.GetFollowerList(ctx, userID, tc.inputPage, tc.inputSize)
			assert.NoError(t, err)
		})
	}
}
