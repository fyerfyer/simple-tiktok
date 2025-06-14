package biz

import (
	"context"

	v1 "go-backend/api/common/v1"

	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/log"
)

var (
	ErrAlreadyFollow = errors.BadRequest(v1.ErrorCode_ALREADY_FOLLOW.String(), "already followed")
	ErrNotFollow     = errors.BadRequest(v1.ErrorCode_NOT_FOLLOW.String(), "not followed")
)

// RelationRepo is a Relation repo.
type RelationRepo interface {
	Follow(context.Context, int64, int64) error
	Unfollow(context.Context, int64, int64) error
	IsFollowing(context.Context, int64, int64) (bool, error)
	GetFollowList(context.Context, int64, int32, int32) ([]*User, int64, error)
	GetFollowerList(context.Context, int64, int32, int32) ([]*User, int64, error)
	GetFriendList(context.Context, int64) ([]*User, error)
}

// RelationUsecase is a Relation usecase.
type RelationUsecase struct {
	repo RelationRepo
	log  *log.Helper
}

// NewRelationUsecase new a Relation usecase.
func NewRelationUsecase(repo RelationRepo, logger log.Logger) *RelationUsecase {
	return &RelationUsecase{repo: repo, log: log.NewHelper(logger)}
}

// Follow follows a user.
func (uc *RelationUsecase) Follow(ctx context.Context, userID, followUserID int64) error {
	uc.log.WithContext(ctx).Infof("User %d follows user %d", userID, followUserID)

	if userID == followUserID {
		return errors.BadRequest("INVALID_FOLLOW", "cannot follow yourself")
	}

	return uc.repo.Follow(ctx, userID, followUserID)
}

// Unfollow unfollows a user.
func (uc *RelationUsecase) Unfollow(ctx context.Context, userID, followUserID int64) error {
	uc.log.WithContext(ctx).Infof("User %d unfollows user %d", userID, followUserID)

	return uc.repo.Unfollow(ctx, userID, followUserID)
}

// IsFollowing checks if user is following another user.
func (uc *RelationUsecase) IsFollowing(ctx context.Context, userID, followUserID int64) (bool, error) {
	return uc.repo.IsFollowing(ctx, userID, followUserID)
}

// GetFollowList gets user's follow list.
func (uc *RelationUsecase) GetFollowList(ctx context.Context, userID int64, page, size int32) ([]*User, int64, error) {
	if page <= 0 {
		page = 1
	}
	if size <= 0 || size > 50 {
		size = 20
	}

	return uc.repo.GetFollowList(ctx, userID, page, size)
}

// GetFollowerList gets user's follower list.
func (uc *RelationUsecase) GetFollowerList(ctx context.Context, userID int64, page, size int32) ([]*User, int64, error) {
	if page <= 0 {
		page = 1
	}
	if size <= 0 || size > 50 {
		size = 20
	}

	return uc.repo.GetFollowerList(ctx, userID, page, size)
}

// GetFriendList gets user's friend list.
func (uc *RelationUsecase) GetFriendList(ctx context.Context, userID int64) ([]*User, error) {
	return uc.repo.GetFriendList(ctx, userID)
}
