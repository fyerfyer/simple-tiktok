package data

import (
	"context"
	"testing"

	"go-backend/internal/biz"
	"go-backend/testutils"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupRelationRepo(t *testing.T) (*relationRepo, *testutils.TestEnv, func()) {
	env, cleanup, err := testutils.SetupTestWithCleanup()
	require.NoError(t, err)

	data := &Data{
		db:  env.DB.DB,
		rdb: env.Redis.Client,
	}

	repo := &relationRepo{
		data: data,
		log:  log.NewHelper(log.DefaultLogger),
	}

	return repo, env, cleanup
}

func TestRelationRepo_Follow(t *testing.T) {
	repo, env, cleanup := setupRelationRepo(t)
	defer cleanup()

	ctx := context.Background()

	// 创建测试用户
	users, err := env.DataManager.CreateTestUsers(2)
	require.NoError(t, err)
	user1, user2 := users[0], users[1]

	// 关注操作
	err = repo.Follow(ctx, user1.ID, user2.ID)
	require.NoError(t, err)

	// 验证关注关系
	isFollowing, err := repo.IsFollowing(ctx, user1.ID, user2.ID)
	require.NoError(t, err)
	assert.True(t, isFollowing)

	// 验证关注数和粉丝数
	var dbUser1, dbUser2 User
	err = env.DB.DB.Where("id = ?", user1.ID).First(&dbUser1).Error
	require.NoError(t, err)
	err = env.DB.DB.Where("id = ?", user2.ID).First(&dbUser2).Error
	require.NoError(t, err)

	assert.Equal(t, 1, dbUser1.FollowCount)
	assert.Equal(t, 1, dbUser2.FollowerCount)

	// 重复关注应该返回错误
	err = repo.Follow(ctx, user1.ID, user2.ID)
	assert.Equal(t, biz.ErrAlreadyFollow, err)
}

func TestRelationRepo_Unfollow(t *testing.T) {
	repo, env, cleanup := setupRelationRepo(t)
	defer cleanup()

	ctx := context.Background()

	// 创建测试用户
	users, err := env.DataManager.CreateTestUsers(2)
	require.NoError(t, err)
	user1, user2 := users[0], users[1]

	// 先建立关注关系
	err = env.DataManager.CreateFollowRelation(user1.ID, user2.ID)
	require.NoError(t, err)

	// 取消关注
	err = repo.Unfollow(ctx, user1.ID, user2.ID)
	require.NoError(t, err)

	// 验证关注关系已删除
	isFollowing, err := repo.IsFollowing(ctx, user1.ID, user2.ID)
	require.NoError(t, err)
	assert.False(t, isFollowing)

	// 重复取消关注应该返回错误
	err = repo.Unfollow(ctx, user1.ID, user2.ID)
	assert.Equal(t, biz.ErrNotFollow, err)
}

func TestRelationRepo_IsFollowing(t *testing.T) {
	repo, env, cleanup := setupRelationRepo(t)
	defer cleanup()

	ctx := context.Background()

	// 创建测试用户
	users, err := env.DataManager.CreateTestUsers(2)
	require.NoError(t, err)
	user1, user2 := users[0], users[1]

	// 初始状态：未关注
	isFollowing, err := repo.IsFollowing(ctx, user1.ID, user2.ID)
	require.NoError(t, err)
	assert.False(t, isFollowing)

	// 建立关注关系
	err = env.DataManager.CreateFollowRelation(user1.ID, user2.ID)
	require.NoError(t, err)

	// 验证关注状态
	isFollowing, err = repo.IsFollowing(ctx, user1.ID, user2.ID)
	require.NoError(t, err)
	assert.True(t, isFollowing)
}

func TestRelationRepo_GetFollowList(t *testing.T) {
	repo, env, cleanup := setupRelationRepo(t)
	defer cleanup()

	ctx := context.Background()

	// 创建测试用户
	users, err := env.DataManager.CreateTestUsers(4)
	require.NoError(t, err)
	user1 := users[0]

	// 建立关注关系：user1 关注 user2, user3, user4
	for i := 1; i < 4; i++ {
		err = env.DataManager.CreateFollowRelation(user1.ID, users[i].ID)
		require.NoError(t, err)
	}

	// 获取关注列表
	followList, total, err := repo.GetFollowList(ctx, user1.ID, 1, 10)
	require.NoError(t, err)
	assert.Equal(t, int64(3), total)
	assert.Len(t, followList, 3)

	// 验证所有关注的用户都标记为已关注
	for _, user := range followList {
		assert.True(t, user.IsFollow)
	}

	// 测试分页
	followList, total, err = repo.GetFollowList(ctx, user1.ID, 1, 2)
	require.NoError(t, err)
	assert.Equal(t, int64(3), total)
	assert.Len(t, followList, 2)
}

func TestRelationRepo_GetFollowerList(t *testing.T) {
	repo, env, cleanup := setupRelationRepo(t)
	defer cleanup()

	ctx := context.Background()

	// 创建测试用户
	users, err := env.DataManager.CreateTestUsers(4)
	require.NoError(t, err)
	user1 := users[0]

	// 建立关注关系：user2, user3, user4 关注 user1
	for i := 1; i < 4; i++ {
		err = env.DataManager.CreateFollowRelation(users[i].ID, user1.ID)
		require.NoError(t, err)
	}

	// 建立互相关注：user1 关注 user2
	err = env.DataManager.CreateFollowRelation(user1.ID, users[1].ID)
	require.NoError(t, err)

	// 获取粉丝列表
	followerList, total, err := repo.GetFollowerList(ctx, user1.ID, 1, 10)
	require.NoError(t, err)
	assert.Equal(t, int64(3), total)
	assert.Len(t, followerList, 3)

	// 验证互相关注的用户标记为已关注
	user2Found := false
	for _, user := range followerList {
		if user.ID == users[1].ID {
			assert.True(t, user.IsFollow) // user2应该被标记为已关注
			user2Found = true
		}
	}
	assert.True(t, user2Found)
}

func TestRelationRepo_GetFriendList(t *testing.T) {
	repo, env, cleanup := setupRelationRepo(t)
	defer cleanup()

	ctx := context.Background()

	// 创建测试用户
	users, err := env.DataManager.CreateTestUsers(4)
	require.NoError(t, err)
	user1 := users[0]

	// 建立互相关注关系：user1 和 user2, user3 互相关注
	for i := 1; i < 3; i++ {
		err = env.DataManager.CreateFollowRelation(user1.ID, users[i].ID)
		require.NoError(t, err)
		err = env.DataManager.CreateFollowRelation(users[i].ID, user1.ID)
		require.NoError(t, err)
	}

	// user4 只关注 user1，不是好友
	err = env.DataManager.CreateFollowRelation(users[3].ID, user1.ID)
	require.NoError(t, err)

	// 获取好友列表
	friendList, err := repo.GetFriendList(ctx, user1.ID)
	require.NoError(t, err)
	assert.Len(t, friendList, 2)

	// 验证所有好友都标记为已关注
	for _, user := range friendList {
		assert.True(t, user.IsFollow)
	}

	// 验证返回的是正确的好友
	friendIDs := make(map[int64]bool)
	for _, friend := range friendList {
		friendIDs[friend.ID] = true
	}
	assert.True(t, friendIDs[users[1].ID])
	assert.True(t, friendIDs[users[2].ID])
	assert.False(t, friendIDs[users[3].ID])
}

func TestRelationRepo_CacheOperations(t *testing.T) {
	repo, env, cleanup := setupRelationRepo(t)
	defer cleanup()

	ctx := context.Background()

	// 创建测试用户
	users, err := env.DataManager.CreateTestUsers(2)
	require.NoError(t, err)
	user1, user2 := users[0], users[1]

	// 设置关注缓存
	repo.setFollowCache(ctx, user1.ID, user2.ID, true)

	// 验证缓存
	cached := repo.getFollowCache(ctx, user1.ID, user2.ID)
	assert.Equal(t, "1", cached)

	// 清除缓存
	repo.clearRelationCache(ctx, user1.ID, user2.ID)

	// 验证缓存已清除
	cached = repo.getFollowCache(ctx, user1.ID, user2.ID)
	assert.Empty(t, cached)
}
