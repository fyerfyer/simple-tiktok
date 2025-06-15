package service

import (
	"context"

	commonv1 "go-backend/api/common/v1"
	v1 "go-backend/api/user/v1"
	"go-backend/internal/biz"
	"go-backend/internal/middleware"
	"go-backend/pkg/auth"
	"go-backend/pkg/security"

	"github.com/go-kratos/kratos/v2/log"
	"google.golang.org/protobuf/types/known/emptypb"
)

// UserService is a user service.
type UserService struct {
	v1.UnimplementedUserServiceServer

	userUc       *biz.UserUsecase
	relationUc   *biz.RelationUsecase
	authUc       *biz.AuthUsecase
	permissionUc *biz.PermissionUsecase
	jwtManager   *auth.JWTManager
	validator    *security.Validator
	log          *log.Helper
}

// NewUserService new a user service.
func NewUserService(
	userUc *biz.UserUsecase,
	relationUc *biz.RelationUsecase,
	authUc *biz.AuthUsecase,
	permissionUc *biz.PermissionUsecase,
	jwtManager *auth.JWTManager,
	validator *security.Validator,
	logger log.Logger,
) *UserService {
	return &UserService{
		userUc:       userUc,
		relationUc:   relationUc,
		authUc:       authUc,
		permissionUc: permissionUc,
		jwtManager:   jwtManager,
		validator:    validator,
		log:          log.NewHelper(logger),
	}
}

// Register 用户注册
func (s *UserService) Register(ctx context.Context, req *v1.RegisterRequest) (*v1.RegisterResponse, error) {
	// 参数验证
	if err := s.validator.ValidateUsername(req.Username); err != nil {
		return &v1.RegisterResponse{
			Base: &commonv1.BaseResponse{
				StatusCode: int32(commonv1.ErrorCode_PARAM_ERROR),
				StatusMsg:  err.Error(),
			},
		}, nil
	}

	if err := s.validator.ValidatePassword(req.Password); err != nil {
		return &v1.RegisterResponse{
			Base: &commonv1.BaseResponse{
				StatusCode: int32(commonv1.ErrorCode_PARAM_ERROR),
				StatusMsg:  err.Error(),
			},
		}, nil
	}

	// 注册用户
	user, err := s.userUc.Register(ctx, req.Username, req.Password)
	if err != nil {
		if err == biz.ErrUserExist {
			return &v1.RegisterResponse{
				Base: &commonv1.BaseResponse{
					StatusCode: int32(commonv1.ErrorCode_USER_EXIST),
					StatusMsg:  "user already exists",
				},
			}, nil
		}
		s.log.WithContext(ctx).Errorf("register user failed: %v", err)
		return &v1.RegisterResponse{
			Base: &commonv1.BaseResponse{
				StatusCode: int32(commonv1.ErrorCode_SERVER_ERROR),
				StatusMsg:  "register failed",
			},
		}, nil
	}

	// 生成Token对
	tokenPair, err := s.jwtManager.GenerateTokenPair(user.ID, user.Username)
	if err != nil {
		s.log.WithContext(ctx).Errorf("generate token failed: %v", err)
		return &v1.RegisterResponse{
			Base: &commonv1.BaseResponse{
				StatusCode: int32(commonv1.ErrorCode_SERVER_ERROR),
				StatusMsg:  "generate token failed",
			},
		}, nil
	}

	// 为新用户分配默认角色
	if err := s.permissionUc.InitUserDefaultRole(ctx, user.ID); err != nil {
		s.log.WithContext(ctx).Errorf("init user default role failed: %v", err)
	}

	return &v1.RegisterResponse{
		Base: &commonv1.BaseResponse{
			StatusCode: 0,
			StatusMsg:  "success",
		},
		Data: &v1.RegisterData{
			UserId: user.ID,
			Token:  tokenPair.AccessToken,
		},
	}, nil
}

// Login 用户登录
func (s *UserService) Login(ctx context.Context, req *v1.LoginRequest) (*v1.LoginResponse, error) {
	// 参数验证
	if req.Username == "" || req.Password == "" {
		return &v1.LoginResponse{
			Base: &commonv1.BaseResponse{
				StatusCode: int32(commonv1.ErrorCode_PARAM_ERROR),
				StatusMsg:  "username and password required",
			},
		}, nil
	}

	// 使用认证服务登录
	tokenPair, user, err := s.authUc.LoginWithToken(ctx, req.Username, req.Password)
	if err != nil {
		if err == biz.ErrUserNotFound {
			return &v1.LoginResponse{
				Base: &commonv1.BaseResponse{
					StatusCode: int32(commonv1.ErrorCode_USER_NOT_EXIST),
					StatusMsg:  "user not found",
				},
			}, nil
		}
		if err == biz.ErrPasswordError {
			return &v1.LoginResponse{
				Base: &commonv1.BaseResponse{
					StatusCode: int32(commonv1.ErrorCode_PASSWORD_ERROR),
					StatusMsg:  "invalid password",
				},
			}, nil
		}
		s.log.WithContext(ctx).Errorf("login failed: %v", err)
		return &v1.LoginResponse{
			Base: &commonv1.BaseResponse{
				StatusCode: int32(commonv1.ErrorCode_SERVER_ERROR),
				StatusMsg:  "login failed",
			},
		}, nil
	}

	return &v1.LoginResponse{
		Base: &commonv1.BaseResponse{
			StatusCode: 0,
			StatusMsg:  "success",
		},
		Data: &v1.LoginData{
			UserId: user.ID,
			Token:  tokenPair.AccessToken,
		},
	}, nil
}

// GetUser 获取用户信息
func (s *UserService) GetUser(ctx context.Context, req *v1.GetUserRequest) (*v1.GetUserResponse, error) {
	// 验证用户ID
	if err := s.validator.ValidateUserID(req.UserId); err != nil {
		return &v1.GetUserResponse{
			Base: &commonv1.BaseResponse{
				StatusCode: int32(commonv1.ErrorCode_PARAM_ERROR),
				StatusMsg:  err.Error(),
			},
		}, nil
	}

	// 获取当前用户ID
	currentUserID, _ := middleware.GetUserIDFromContext(ctx)

	// 获取用户信息
	user, err := s.userUc.GetUser(ctx, req.UserId)
	if err != nil {
		if err == biz.ErrUserNotFound {
			return &v1.GetUserResponse{
				Base: &commonv1.BaseResponse{
					StatusCode: int32(commonv1.ErrorCode_USER_NOT_EXIST),
					StatusMsg:  "user not found",
				},
			}, nil
		}
		s.log.WithContext(ctx).Errorf("get user failed: %v", err)
		return &v1.GetUserResponse{
			Base: &commonv1.BaseResponse{
				StatusCode: int32(commonv1.ErrorCode_SERVER_ERROR),
				StatusMsg:  "get user failed",
			},
		}, nil
	}

	// 检查关注关系
	isFollow := false
	if currentUserID > 0 && currentUserID != req.UserId {
		isFollow, _ = s.relationUc.IsFollowing(ctx, currentUserID, req.UserId)
	}

	return &v1.GetUserResponse{
		Base: &commonv1.BaseResponse{
			StatusCode: 0,
			StatusMsg:  "success",
		},
		Data: &v1.GetUserData{
			User: s.convertToCommonUser(user, isFollow),
		},
	}, nil
}

// RelationAction 关注操作
func (s *UserService) RelationAction(ctx context.Context, req *v1.RelationActionRequest) (*v1.RelationActionResponse, error) {
	// 获取当前用户ID
	userID, ok := middleware.GetUserIDFromContext(ctx)
	if !ok {
		return &v1.RelationActionResponse{
			Base: &commonv1.BaseResponse{
				StatusCode: int32(commonv1.ErrorCode_TOKEN_INVALID),
				StatusMsg:  "invalid token",
			},
		}, nil
	}

	// 验证参数
	if err := s.validator.ValidateUserID(req.ToUserId); err != nil {
		return &v1.RelationActionResponse{
			Base: &commonv1.BaseResponse{
				StatusCode: int32(commonv1.ErrorCode_PARAM_ERROR),
				StatusMsg:  err.Error(),
			},
		}, nil
	}

	if req.ActionType != 1 && req.ActionType != 2 {
		return &v1.RelationActionResponse{
			Base: &commonv1.BaseResponse{
				StatusCode: int32(commonv1.ErrorCode_PARAM_ERROR),
				StatusMsg:  "invalid action type",
			},
		}, nil
	}

	var err error
	if req.ActionType == 1 {
		// 关注
		err = s.relationUc.Follow(ctx, userID, req.ToUserId)
	} else {
		// 取消关注
		err = s.relationUc.Unfollow(ctx, userID, req.ToUserId)
	}

	if err != nil {
		if err == biz.ErrAlreadyFollow || err == biz.ErrNotFollow {
			return &v1.RelationActionResponse{
				Base: &commonv1.BaseResponse{
					StatusCode: int32(commonv1.ErrorCode_ALREADY_FOLLOW),
					StatusMsg:  err.Error(),
				},
			}, nil
		}
		s.log.WithContext(ctx).Errorf("relation action failed: %v", err)
		return &v1.RelationActionResponse{
			Base: &commonv1.BaseResponse{
				StatusCode: int32(commonv1.ErrorCode_SERVER_ERROR),
				StatusMsg:  "operation failed",
			},
		}, nil
	}

	return &v1.RelationActionResponse{
		Base: &commonv1.BaseResponse{
			StatusCode: 0,
			StatusMsg:  "success",
		},
	}, nil
}

// GetFollowList 获取关注列表
func (s *UserService) GetFollowList(ctx context.Context, req *v1.GetFollowListRequest) (*v1.GetFollowListResponse, error) {
	// 验证用户ID
	if err := s.validator.ValidateUserID(req.UserId); err != nil {
		return &v1.GetFollowListResponse{
			Base: &commonv1.BaseResponse{
				StatusCode: int32(commonv1.ErrorCode_PARAM_ERROR),
				StatusMsg:  err.Error(),
			},
		}, nil
	}

	// 获取关注列表
	users, _, err := s.relationUc.GetFollowList(ctx, req.UserId, 1, 50)
	if err != nil {
		s.log.WithContext(ctx).Errorf("get follow list failed: %v", err)
		return &v1.GetFollowListResponse{
			Base: &commonv1.BaseResponse{
				StatusCode: int32(commonv1.ErrorCode_SERVER_ERROR),
				StatusMsg:  "get follow list failed",
			},
		}, nil
	}

	// 转换为响应格式
	userList := make([]*commonv1.User, 0, len(users))
	for _, user := range users {
		userList = append(userList, s.convertToCommonUser(user, user.IsFollow))
	}

	return &v1.GetFollowListResponse{
		Base: &commonv1.BaseResponse{
			StatusCode: 0,
			StatusMsg:  "success",
		},
		Data: &v1.GetFollowListData{
			UserList: userList,
		},
	}, nil
}

// GetFollowerList 获取粉丝列表
func (s *UserService) GetFollowerList(ctx context.Context, req *v1.GetFollowerListRequest) (*v1.GetFollowerListResponse, error) {
	// 验证用户ID
	if err := s.validator.ValidateUserID(req.UserId); err != nil {
		return &v1.GetFollowerListResponse{
			Base: &commonv1.BaseResponse{
				StatusCode: int32(commonv1.ErrorCode_PARAM_ERROR),
				StatusMsg:  err.Error(),
			},
		}, nil
	}

	// 获取粉丝列表
	users, _, err := s.relationUc.GetFollowerList(ctx, req.UserId, 1, 50)
	if err != nil {
		s.log.WithContext(ctx).Errorf("get follower list failed: %v", err)
		return &v1.GetFollowerListResponse{
			Base: &commonv1.BaseResponse{
				StatusCode: int32(commonv1.ErrorCode_SERVER_ERROR),
				StatusMsg:  "get follower list failed",
			},
		}, nil
	}

	// 转换为响应格式
	userList := make([]*commonv1.User, 0, len(users))
	for _, user := range users {
		userList = append(userList, s.convertToCommonUser(user, user.IsFollow))
	}

	return &v1.GetFollowerListResponse{
		Base: &commonv1.BaseResponse{
			StatusCode: 0,
			StatusMsg:  "success",
		},
		Data: &v1.GetFollowerListData{
			UserList: userList,
		},
	}, nil
}

// GetFriendList 获取好友列表
func (s *UserService) GetFriendList(ctx context.Context, req *v1.GetFriendListRequest) (*v1.GetFriendListResponse, error) {
	// 验证用户ID
	if err := s.validator.ValidateUserID(req.UserId); err != nil {
		return &v1.GetFriendListResponse{
			Base: &commonv1.BaseResponse{
				StatusCode: int32(commonv1.ErrorCode_PARAM_ERROR),
				StatusMsg:  err.Error(),
			},
		}, nil
	}

	// 获取好友列表
	users, err := s.relationUc.GetFriendList(ctx, req.UserId)
	if err != nil {
		s.log.WithContext(ctx).Errorf("get friend list failed: %v", err)
		return &v1.GetFriendListResponse{
			Base: &commonv1.BaseResponse{
				StatusCode: int32(commonv1.ErrorCode_SERVER_ERROR),
				StatusMsg:  "get friend list failed",
			},
		}, nil
	}

	// 转换为响应格式
	userList := make([]*v1.FriendUser, 0, len(users))
	for _, user := range users {
		friendUser := &v1.FriendUser{
			Id:              user.ID,
			Name:            user.Nickname,
			FollowCount:     int64(user.FollowCount),
			FollowerCount:   int64(user.FollowerCount),
			IsFollow:        user.IsFollow,
			Avatar:          user.Avatar,
			BackgroundImage: user.BackgroundImage,
			Signature:       user.Signature,
			TotalFavorited:  user.TotalFavorited,
			WorkCount:       int64(user.WorkCount),
			FavoriteCount:   int64(user.FavoriteCount),
			Message:         "暂无消息",
			MsgType:         1,
		}
		userList = append(userList, friendUser)
	}

	return &v1.GetFriendListResponse{
		Base: &commonv1.BaseResponse{
			StatusCode: 0,
			StatusMsg:  "success",
		},
		Data: &v1.GetFriendListData{
			UserList: userList,
		},
	}, nil
}

// GetUserInfo 获取用户信息
func (s *UserService) GetUserInfo(ctx context.Context, req *v1.GetUserInfoRequest) (*v1.GetUserInfoResponse, error) {
	user, err := s.userUc.GetUser(ctx, req.UserId)
	if err != nil {
		return nil, err
	}

	return &v1.GetUserInfoResponse{
		User: s.convertToCommonUser(user, false),
	}, nil
}

// GetUsersInfo 批量获取用户信息
func (s *UserService) GetUsersInfo(ctx context.Context, req *v1.GetUsersInfoRequest) (*v1.GetUsersInfoResponse, error) {
	users, err := s.userUc.GetUsers(ctx, req.UserIds)
	if err != nil {
		return nil, err
	}

	userList := make([]*commonv1.User, 0, len(users))
	for _, user := range users {
		userList = append(userList, s.convertToCommonUser(user, false))
	}

	return &v1.GetUsersInfoResponse{
		Users: userList,
	}, nil
}

// VerifyToken 验证Token
func (s *UserService) VerifyToken(ctx context.Context, req *v1.VerifyTokenRequest) (*v1.VerifyTokenResponse, error) {
	claims, err := s.authUc.VerifyToken(ctx, req.Token)
	if err != nil {
		return &v1.VerifyTokenResponse{
			Valid: false,
		}, nil
	}

	return &v1.VerifyTokenResponse{
		Valid:    true,
		UserId:   claims.UserID,
		Username: claims.Username,
	}, nil
}

// UpdateUserStats 更新用户统计
func (s *UserService) UpdateUserStats(ctx context.Context, req *v1.UpdateUserStatsRequest) (*emptypb.Empty, error) {
	stats := &biz.UserStats{}

	switch req.Type {
	case v1.UpdateStatsType_UPDATE_STATS_FOLLOW_COUNT:
		stats.FollowCountDelta = int(req.Count)
	case v1.UpdateStatsType_UPDATE_STATS_FOLLOWER_COUNT:
		stats.FollowerCountDelta = int(req.Count)
	case v1.UpdateStatsType_UPDATE_STATS_WORK_COUNT:
		stats.WorkCountDelta = int(req.Count)
	case v1.UpdateStatsType_UPDATE_STATS_FAVORITE_COUNT:
		stats.FavoriteCountDelta = int(req.Count)
	case v1.UpdateStatsType_UPDATE_STATS_TOTAL_FAVORITED:
		stats.TotalFavoritedDelta = req.Count
	}

	err := s.userUc.UpdateUserStats(ctx, req.UserId, stats)
	if err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

// convertToCommonUser 转换为通用用户信息
func (s *UserService) convertToCommonUser(user *biz.User, isFollow bool) *commonv1.User {
	return &commonv1.User{
		Id:              user.ID,
		Name:            user.Nickname,
		FollowCount:     int64(user.FollowCount),
		FollowerCount:   int64(user.FollowerCount),
		IsFollow:        isFollow,
		Avatar:          user.Avatar,
		BackgroundImage: user.BackgroundImage,
		Signature:       user.Signature,
		TotalFavorited:  user.TotalFavorited,
		WorkCount:       int64(user.WorkCount),
		FavoriteCount:   int64(user.FavoriteCount),
	}
}
