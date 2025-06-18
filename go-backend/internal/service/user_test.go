package service

import (
	"context"
	"testing"
	"time"

	v1 "go-backend/api/user/v1"
	"go-backend/internal/biz"
	"go-backend/internal/conf"
	"go-backend/internal/data"
	"go-backend/internal/data/cache"
	"go-backend/pkg/auth"
	pkgcache "go-backend/pkg/cache"
	"go-backend/pkg/security"
	"go-backend/testutils"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-redis/redis/v8"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/durationpb"
)

func TestUserService_Register(t *testing.T) {
	t.Run("Register_Success", func(t *testing.T) {
		service, _, cleanup := setupUserServiceForTest(t)
		defer cleanup()

		ctx := context.Background()

		req := &v1.RegisterRequest{
			Username: "testuser",
			Password: "Password123!",
		}

		resp, err := service.Register(ctx, req)

		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, int32(0), resp.Base.StatusCode)
		assert.Equal(t, "success", resp.Base.StatusMsg)
		assert.NotNil(t, resp.Data)
		assert.NotZero(t, resp.Data.UserId)
		assert.NotEmpty(t, resp.Data.Token)
	})

	t.Run("Register_InvalidUsername", func(t *testing.T) {
		service, _, cleanup := setupUserServiceForTest(t)
		defer cleanup()

		ctx := context.Background()

		req := &v1.RegisterRequest{
			Username: "ab", // 太短
			Password: "Password123!",
		}

		resp, err := service.Register(ctx, req)

		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.NotEqual(t, int32(0), resp.Base.StatusCode)
		assert.Contains(t, resp.Base.StatusMsg, "username")
	})

	t.Run("Register_InvalidPassword", func(t *testing.T) {
		service, _, cleanup := setupUserServiceForTest(t)
		defer cleanup()

		ctx := context.Background()

		req := &v1.RegisterRequest{
			Username: "testuser2",
			Password: "weak", // 太弱
		}

		resp, err := service.Register(ctx, req)

		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.NotEqual(t, int32(0), resp.Base.StatusCode)
		assert.Contains(t, resp.Base.StatusMsg, "password")
	})

	t.Run("Register_UserExists", func(t *testing.T) {
		service, env, cleanup := setupUserServiceForTest(t)
		defer cleanup()

		ctx := context.Background()

		// 先创建用户
		users, err := env.DataManager.CreateTestUsers(1)
		require.NoError(t, err)

		req := &v1.RegisterRequest{
			Username: users[0].Username,
			Password: "Password123!",
		}

		resp, err := service.Register(ctx, req)

		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.NotEqual(t, int32(0), resp.Base.StatusCode)
		assert.Contains(t, resp.Base.StatusMsg, "exists")
	})
}

func TestUserService_Login(t *testing.T) {
	t.Run("Login_Success", func(t *testing.T) {
		service, _, cleanup := setupUserServiceForTest(t)
		defer cleanup()

		ctx := context.Background()

		// 先注册用户
		registerReq := &v1.RegisterRequest{
			Username: "loginuser",
			Password: "Password123!",
		}
		_, err := service.Register(ctx, registerReq)
		require.NoError(t, err)

		// 登录
		loginReq := &v1.LoginRequest{
			Username: "loginuser",
			Password: "Password123!",
		}

		resp, err := service.Login(ctx, loginReq)

		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, int32(0), resp.Base.StatusCode)
		assert.Equal(t, "success", resp.Base.StatusMsg)
		assert.NotNil(t, resp.Data)
		assert.NotZero(t, resp.Data.UserId)
		assert.NotEmpty(t, resp.Data.Token)
	})

	t.Run("Login_WrongPassword", func(t *testing.T) {
		service, _, cleanup := setupUserServiceForTest(t)
		defer cleanup()

		ctx := context.Background()

		// 先注册用户
		registerReq := &v1.RegisterRequest{
			Username: "loginuser2",
			Password: "Password123!",
		}
		_, err := service.Register(ctx, registerReq)
		require.NoError(t, err)

		req := &v1.LoginRequest{
			Username: "loginuser2",
			Password: "WrongPassword123!",
		}

		resp, err := service.Login(ctx, req)

		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.NotEqual(t, int32(0), resp.Base.StatusCode)
		assert.Contains(t, resp.Base.StatusMsg, "password")
	})

	t.Run("Login_UserNotFound", func(t *testing.T) {
		service, _, cleanup := setupUserServiceForTest(t)
		defer cleanup()

		ctx := context.Background()

		req := &v1.LoginRequest{
			Username: "nonexistent",
			Password: "Password123!",
		}

		resp, err := service.Login(ctx, req)

		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.NotEqual(t, int32(0), resp.Base.StatusCode)
		assert.Contains(t, resp.Base.StatusMsg, "not found")
	})

	t.Run("Login_EmptyCredentials", func(t *testing.T) {
		service, _, cleanup := setupUserServiceForTest(t)
		defer cleanup()

		ctx := context.Background()

		req := &v1.LoginRequest{
			Username: "",
			Password: "",
		}

		resp, err := service.Login(ctx, req)

		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.NotEqual(t, int32(0), resp.Base.StatusCode)
		assert.Contains(t, resp.Base.StatusMsg, "required")
	})
}

func TestUserService_GetUser(t *testing.T) {
	t.Run("GetUser_Success", func(t *testing.T) {
		service, env, cleanup := setupUserServiceForTest(t)
		defer cleanup()

		ctx := context.Background()

		// 创建测试用户
		users, err := env.DataManager.CreateTestUsers(1)
		require.NoError(t, err)
		testUser := users[0]

		req := &v1.GetUserRequest{
			UserId: testUser.ID,
		}

		resp, err := service.GetUser(ctx, req)

		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, int32(0), resp.Base.StatusCode)
		assert.NotNil(t, resp.Data)
		assert.NotNil(t, resp.Data.User)
		assert.Equal(t, testUser.ID, resp.Data.User.Id)
		assert.Equal(t, testUser.Nickname, resp.Data.User.Name)
	})

	t.Run("GetUser_InvalidID", func(t *testing.T) {
		service, _, cleanup := setupUserServiceForTest(t)
		defer cleanup()

		ctx := context.Background()

		req := &v1.GetUserRequest{
			UserId: 0,
		}

		resp, err := service.GetUser(ctx, req)

		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.NotEqual(t, int32(0), resp.Base.StatusCode)
	})

	t.Run("GetUser_NotFound", func(t *testing.T) {
		service, _, cleanup := setupUserServiceForTest(t)
		defer cleanup()

		ctx := context.Background()

		req := &v1.GetUserRequest{
			UserId: 99999,
		}

		resp, err := service.GetUser(ctx, req)

		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.NotEqual(t, int32(0), resp.Base.StatusCode)
		assert.Contains(t, resp.Base.StatusMsg, "not found")
	})
}

func TestUserService_RelationAction(t *testing.T) {
	t.Run("RelationAction_Follow", func(t *testing.T) {
		service, env, cleanup := setupUserServiceForTest(t)
		defer cleanup()

		ctx := context.Background()

		// 创建测试用户
		users, err := env.DataManager.CreateTestUsers(2)
		require.NoError(t, err)
		user1, user2 := users[0], users[1]

		// 生成Token
		jwtManager := auth.NewJWTManager("test-secret", time.Hour)
		token, err := jwtManager.GenerateToken(user1.ID, user1.Username)
		require.NoError(t, err)

		// 在上下文中设置用户信息
		ctx = context.WithValue(ctx, "user_id", user1.ID)

		req := &v1.RelationActionRequest{
			Token:      token,
			ToUserId:   user2.ID,
			ActionType: 1, // 关注
		}

		resp, err := service.RelationAction(ctx, req)

		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, int32(0), resp.Base.StatusCode)
		assert.Equal(t, "success", resp.Base.StatusMsg)
	})

	t.Run("RelationAction_Unfollow", func(t *testing.T) {
		service, env, cleanup := setupUserServiceForTest(t)
		defer cleanup()

		ctx := context.Background()

		// 创建测试用户
		users, err := env.DataManager.CreateTestUsers(2)
		require.NoError(t, err)
		user1, user2 := users[0], users[1]

		// 先建立关注关系
		err = env.DataManager.CreateFollowRelation(user1.ID, user2.ID)
		require.NoError(t, err)

		// 生成Token
		jwtManager := auth.NewJWTManager("test-secret", time.Hour)
		token, err := jwtManager.GenerateToken(user1.ID, user1.Username)
		require.NoError(t, err)

		// 在上下文中设置用户信息
		ctx = context.WithValue(ctx, "user_id", user1.ID)

		req := &v1.RelationActionRequest{
			Token:      token,
			ToUserId:   user2.ID,
			ActionType: 2, // 取消关注
		}

		resp, err := service.RelationAction(ctx, req)

		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, int32(0), resp.Base.StatusCode)
	})

	t.Run("RelationAction_InvalidActionType", func(t *testing.T) {
		service, env, cleanup := setupUserServiceForTest(t)
		defer cleanup()

		ctx := context.Background()

		// 创建测试用户
		users, err := env.DataManager.CreateTestUsers(2)
		require.NoError(t, err)
		user1, user2 := users[0], users[1]

		// 生成Token
		jwtManager := auth.NewJWTManager("test-secret", time.Hour)
		token, err := jwtManager.GenerateToken(user1.ID, user1.Username)
		require.NoError(t, err)

		// 在上下文中设置用户信息
		ctx = context.WithValue(ctx, "user_id", user1.ID)

		req := &v1.RelationActionRequest{
			Token:      token,
			ToUserId:   user2.ID,
			ActionType: 3, // 无效类型
		}

		resp, err := service.RelationAction(ctx, req)

		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.NotEqual(t, int32(0), resp.Base.StatusCode)
		assert.Contains(t, resp.Base.StatusMsg, "invalid")
	})

	t.Run("RelationAction_InvalidUserID", func(t *testing.T) {
		service, env, cleanup := setupUserServiceForTest(t)
		defer cleanup()

		ctx := context.Background()

		// 创建测试用户
		users, err := env.DataManager.CreateTestUsers(1)
		require.NoError(t, err)
		user1 := users[0]

		// 生成Token
		jwtManager := auth.NewJWTManager("test-secret", time.Hour)
		token, err := jwtManager.GenerateToken(user1.ID, user1.Username)
		require.NoError(t, err)

		// 在上下文中设置用户信息
		ctx = context.WithValue(ctx, "user_id", user1.ID)

		req := &v1.RelationActionRequest{
			Token:      token,
			ToUserId:   0,
			ActionType: 1,
		}

		resp, err := service.RelationAction(ctx, req)

		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.NotEqual(t, int32(0), resp.Base.StatusCode)
	})
}

func TestUserService_GetFollowList(t *testing.T) {
	t.Run("GetFollowList_Success", func(t *testing.T) {
		service, env, cleanup := setupUserServiceForTest(t)
		defer cleanup()

		ctx := context.Background()

		// 创建测试用户
		users, err := env.DataManager.CreateTestUsers(3)
		require.NoError(t, err)
		user1 := users[0]

		// 建立关注关系
		for i := 1; i < 3; i++ {
			err = env.DataManager.CreateFollowRelation(user1.ID, users[i].ID)
			require.NoError(t, err)
		}

		req := &v1.GetFollowListRequest{
			UserId: user1.ID,
		}

		resp, err := service.GetFollowList(ctx, req)

		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, int32(0), resp.Base.StatusCode)
		assert.NotNil(t, resp.Data)
		assert.Len(t, resp.Data.UserList, 2)

		// 验证关注状态
		for _, user := range resp.Data.UserList {
			assert.True(t, user.IsFollow)
		}
	})

	t.Run("GetFollowList_InvalidUserID", func(t *testing.T) {
		service, _, cleanup := setupUserServiceForTest(t)
		defer cleanup()

		ctx := context.Background()

		req := &v1.GetFollowListRequest{
			UserId: 0,
		}

		resp, err := service.GetFollowList(ctx, req)

		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.NotEqual(t, int32(0), resp.Base.StatusCode)
	})
}

func TestUserService_GetFollowerList(t *testing.T) {
	t.Run("GetFollowerList_Success", func(t *testing.T) {
		service, env, cleanup := setupUserServiceForTest(t)
		defer cleanup()

		ctx := context.Background()

		// 创建测试用户
		users, err := env.DataManager.CreateTestUsers(3)
		require.NoError(t, err)
		user1 := users[0]

		// 建立关注关系：其他用户关注user1
		for i := 1; i < 3; i++ {
			err = env.DataManager.CreateFollowRelation(users[i].ID, user1.ID)
			require.NoError(t, err)
		}

		req := &v1.GetFollowerListRequest{
			UserId: user1.ID,
		}

		resp, err := service.GetFollowerList(ctx, req)

		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, int32(0), resp.Base.StatusCode)
		assert.NotNil(t, resp.Data)
		assert.Len(t, resp.Data.UserList, 2)
	})
}

func TestUserService_GetFriendList(t *testing.T) {
	t.Run("GetFriendList_Success", func(t *testing.T) {
		service, env, cleanup := setupUserServiceForTest(t)
		defer cleanup()

		ctx := context.Background()

		// 创建测试用户
		users, err := env.DataManager.CreateTestUsers(3)
		require.NoError(t, err)
		user1 := users[0]

		// 建立互相关注关系
		for i := 1; i < 3; i++ {
			err = env.DataManager.CreateFollowRelation(user1.ID, users[i].ID)
			require.NoError(t, err)
			err = env.DataManager.CreateFollowRelation(users[i].ID, user1.ID)
			require.NoError(t, err)
		}

		req := &v1.GetFriendListRequest{
			UserId: user1.ID,
		}

		resp, err := service.GetFriendList(ctx, req)

		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, int32(0), resp.Base.StatusCode)
		assert.NotNil(t, resp.Data)
		assert.Len(t, resp.Data.UserList, 2)

		// 验证好友状态
		for _, user := range resp.Data.UserList {
			assert.True(t, user.IsFollow)
			assert.Equal(t, "暂无消息", user.Message)
			assert.Equal(t, int64(1), user.MsgType)
		}
	})
}

func TestUserService_GetUserInfo(t *testing.T) {
	t.Run("GetUserInfo_Success", func(t *testing.T) {
		service, env, cleanup := setupUserServiceForTest(t)
		defer cleanup()

		ctx := context.Background()

		// 创建测试用户
		users, err := env.DataManager.CreateTestUsers(1)
		require.NoError(t, err)
		testUser := users[0]

		req := &v1.GetUserInfoRequest{
			UserId: testUser.ID,
		}

		resp, err := service.GetUserInfo(ctx, req)

		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.NotNil(t, resp.User)
		assert.Equal(t, testUser.ID, resp.User.Id)
		assert.False(t, resp.User.IsFollow) // 默认不关注
	})
}

func TestUserService_GetUsersInfo(t *testing.T) {
	t.Run("GetUsersInfo_Success", func(t *testing.T) {
		service, env, cleanup := setupUserServiceForTest(t)
		defer cleanup()

		ctx := context.Background()

		// 创建测试用户
		users, err := env.DataManager.CreateTestUsers(3)
		require.NoError(t, err)

		userIDs := make([]int64, 0, len(users))
		for _, user := range users {
			userIDs = append(userIDs, user.ID)
		}

		req := &v1.GetUsersInfoRequest{
			UserIds: userIDs,
		}

		resp, err := service.GetUsersInfo(ctx, req)

		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Len(t, resp.Users, 3)
	})
}

func TestUserService_VerifyToken(t *testing.T) {
	t.Run("VerifyToken_Success", func(t *testing.T) {
		service, env, cleanup := setupUserServiceForTest(t)
		defer cleanup()

		ctx := context.Background()

		// 创建测试用户
		users, err := env.DataManager.CreateTestUsers(1)
		require.NoError(t, err)
		testUser := users[0]

		// 生成有效Token
		jwtManager := auth.NewJWTManager("test-secret", time.Hour)
		token, err := jwtManager.GenerateToken(testUser.ID, testUser.Username)
		require.NoError(t, err)

		req := &v1.VerifyTokenRequest{
			Token: token,
		}

		resp, err := service.VerifyToken(ctx, req)

		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.True(t, resp.Valid)
		assert.Equal(t, testUser.ID, resp.UserId)
		assert.Equal(t, testUser.Username, resp.Username)
	})

	t.Run("VerifyToken_Invalid", func(t *testing.T) {
		service, _, cleanup := setupUserServiceForTest(t)
		defer cleanup()

		ctx := context.Background()

		req := &v1.VerifyTokenRequest{
			Token: "invalid-token",
		}

		resp, err := service.VerifyToken(ctx, req)

		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.False(t, resp.Valid)
	})
}

func TestUserService_UpdateUserStats(t *testing.T) {
	t.Run("UpdateUserStats_Success", func(t *testing.T) {
		service, env, cleanup := setupUserServiceForTest(t)
		defer cleanup()

		ctx := context.Background()

		// 创建测试用户
		users, err := env.DataManager.CreateTestUsers(1)
		require.NoError(t, err)
		testUser := users[0]

		req := &v1.UpdateUserStatsRequest{
			UserId: testUser.ID,
			Type:   v1.UpdateStatsType_UPDATE_STATS_FOLLOW_COUNT,
			Count:  1,
		}

		resp, err := service.UpdateUserStats(ctx, req)

		require.NoError(t, err)
		assert.NotNil(t, resp)
	})
}

// setupUserServiceForTest 为每个测试创建独立的服务实例
func setupUserServiceForTest(t *testing.T) (*UserService, *testutils.TestEnv, func()) {
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

	rdb := redis.NewClient(&redis.Options{
		Addr:     config.Redis.Addr,
		Password: config.Redis.Password,
		DB:       int(config.Redis.Db),
	})

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
	relationRepo := data.NewRelationRepo(d, log.DefaultLogger)
	roleRepo := data.NewRoleRepo(d, log.DefaultLogger)
	permissionRepo := data.NewPermissionRepo(d, roleRepo, log.DefaultLogger)
	sessionRepo := data.NewSessionRepo(d, authCache, log.DefaultLogger)

	// 创建用例
	userUc := biz.NewUserUsecase(userRepo, log.DefaultLogger)
	relationUc := biz.NewRelationUsecase(relationRepo, log.DefaultLogger)
	jwtManager := auth.NewJWTManager("test-secret", time.Hour)
	sessionMgr := auth.NewMemorySessionManager()
	authUc := biz.NewAuthUsecase(sessionRepo, userRepo, jwtManager, sessionMgr, log.DefaultLogger)
	rbacManager := auth.NewMemoryRBACManager()
	permissionUc := biz.NewPermissionUsecase(roleRepo, permissionRepo, rbacManager, log.DefaultLogger)

	// 创建服务
	validator := security.NewValidator()
	service := NewUserService(userUc, relationUc, authUc, permissionUc, jwtManager, validator, log.DefaultLogger)

	cleanupFunc := func() {
		dataCleanup()
		cleanup()
	}

	return service, env, cleanupFunc
}
