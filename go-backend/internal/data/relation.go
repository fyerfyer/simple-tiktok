package data

import (
	"context"
	"fmt"
	"time"

	"go-backend/internal/biz"

	"github.com/go-kratos/kratos/v2/log"
	"gorm.io/gorm"
)

// UserFollow 关注关系模型
type UserFollow struct {
	ID           int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID       int64     `gorm:"not null;index:uk_user_follow,priority:1" json:"user_id"`
	FollowUserID int64     `gorm:"not null;index:uk_user_follow,priority:2" json:"follow_user_id"`
	CreatedAt    time.Time `gorm:"autoCreateTime" json:"created_at"`
}

func (UserFollow) TableName() string {
	return "user_follows"
}

type relationRepo struct {
	data *Data
	log  *log.Helper
}

// NewRelationRepo .
func NewRelationRepo(data *Data, logger log.Logger) biz.RelationRepo {
	return &relationRepo{
		data: data,
		log:  log.NewHelper(logger),
	}
}

func (r *relationRepo) Follow(ctx context.Context, userID, followUserID int64) error {
	// 检查是否已关注
	var count int64
	r.data.db.WithContext(ctx).Model(&UserFollow{}).
		Where("user_id = ? AND follow_user_id = ?", userID, followUserID).
		Count(&count)

	if count > 0 {
		return biz.ErrAlreadyFollow
	}

	// 创建关注关系
	follow := &UserFollow{
		UserID:       userID,
		FollowUserID: followUserID,
	}

	err := r.data.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 插入关注记录
		if err := tx.Create(follow).Error; err != nil {
			return err
		}

		// 更新关注数
		if err := tx.Model(&User{}).Where("id = ?", userID).
			Update("follow_count", gorm.Expr("follow_count + 1")).Error; err != nil {
			return err
		}

		// 更新粉丝数
		if err := tx.Model(&User{}).Where("id = ?", followUserID).
			Update("follower_count", gorm.Expr("follower_count + 1")).Error; err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return err
	}

	// 清除缓存
	r.clearRelationCache(ctx, userID, followUserID)

	return nil
}

func (r *relationRepo) Unfollow(ctx context.Context, userID, followUserID int64) error {
	// 检查是否已关注
	var follow UserFollow
	err := r.data.db.WithContext(ctx).
		Where("user_id = ? AND follow_user_id = ?", userID, followUserID).
		First(&follow).Error

	if err == gorm.ErrRecordNotFound {
		return biz.ErrNotFollow
	}
	if err != nil {
		return err
	}

	err = r.data.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 删除关注记录
		if err := tx.Delete(&follow).Error; err != nil {
			return err
		}

		// 更新关注数
		if err := tx.Model(&User{}).Where("id = ?", userID).
			Update("follow_count", gorm.Expr("follow_count - 1")).Error; err != nil {
			return err
		}

		// 更新粉丝数
		if err := tx.Model(&User{}).Where("id = ?", followUserID).
			Update("follower_count", gorm.Expr("follower_count - 1")).Error; err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return err
	}

	// 清除缓存
	r.clearRelationCache(ctx, userID, followUserID)

	return nil
}

func (r *relationRepo) IsFollowing(ctx context.Context, userID, followUserID int64) (bool, error) {
	// 先从缓存检查
	if cached := r.getFollowCache(ctx, userID, followUserID); cached != "" {
		return cached == "1", nil
	}

	var count int64
	err := r.data.db.WithContext(ctx).Model(&UserFollow{}).
		Where("user_id = ? AND follow_user_id = ?", userID, followUserID).
		Count(&count).
		Error

	if err != nil {
		return false, err
	}

	isFollowing := count > 0
	// 设置缓存
	r.setFollowCache(ctx, userID, followUserID, isFollowing)

	return isFollowing, nil
}

func (r *relationRepo) GetFollowList(ctx context.Context, userID int64, page, size int32) ([]*biz.User, int64, error) {
	offset := (page - 1) * size

	// 获取总数
	var total int64
	if err := r.data.db.WithContext(ctx).Model(&UserFollow{}).
		Where("user_id = ?", userID).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 获取关注的用户ID列表
	var follows []UserFollow
	if err := r.data.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Offset(int(offset)).Limit(int(size)).
		Find(&follows).Error; err != nil {
		return nil, 0, err
	}

	if len(follows) == 0 {
		return []*biz.User{}, total, nil
	}

	// 获取用户信息
	userIDs := make([]int64, 0, len(follows))
	for _, f := range follows {
		userIDs = append(userIDs, f.FollowUserID)
	}

	var users []User
	if err := r.data.db.WithContext(ctx).
		Where("id IN ? AND status = 1", userIDs).
		Find(&users).Error; err != nil {
		return nil, 0, err
	}

	// 转换为业务模型
	result := make([]*biz.User, 0, len(users))
	for _, u := range users {
		user := &biz.User{
			ID:              u.ID,
			Username:        u.Username,
			Nickname:        u.Nickname,
			Avatar:          u.Avatar,
			BackgroundImage: u.BackgroundImage,
			Signature:       u.Signature,
			FollowCount:     u.FollowCount,
			FollowerCount:   u.FollowerCount,
			TotalFavorited:  u.TotalFavorited,
			WorkCount:       u.WorkCount,
			FavoriteCount:   u.FavoriteCount,
			IsFollow:        true, // 关注列表中的都是已关注
		}
		result = append(result, user)
	}

	return result, total, nil
}

func (r *relationRepo) GetFollowerList(ctx context.Context, userID int64, page, size int32) ([]*biz.User, int64, error) {
	offset := (page - 1) * size

	// 获取总数
	var total int64
	if err := r.data.db.WithContext(ctx).Model(&UserFollow{}).
		Where("follow_user_id = ?", userID).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 获取粉丝的用户ID列表
	var follows []UserFollow
	if err := r.data.db.WithContext(ctx).
		Where("follow_user_id = ?", userID).
		Order("created_at DESC").
		Offset(int(offset)).Limit(int(size)).
		Find(&follows).Error; err != nil {
		return nil, 0, err
	}

	if len(follows) == 0 {
		return []*biz.User{}, total, nil
	}

	// 获取用户信息
	userIDs := make([]int64, 0, len(follows))
	for _, f := range follows {
		userIDs = append(userIDs, f.UserID)
	}

	var users []User
	if err := r.data.db.WithContext(ctx).
		Where("id IN ? AND status = 1", userIDs).
		Find(&users).Error; err != nil {
		return nil, 0, err
	}

	// 检查互相关注关系
	followMap := make(map[int64]bool)
	var reverseFollows []UserFollow
	if err := r.data.db.WithContext(ctx).
		Where("user_id = ? AND follow_user_id IN ?", userID, userIDs).
		Find(&reverseFollows).Error; err == nil {
		for _, f := range reverseFollows {
			followMap[f.FollowUserID] = true
		}
	}

	// 转换为业务模型
	result := make([]*biz.User, 0, len(users))
	for _, u := range users {
		user := &biz.User{
			ID:              u.ID,
			Username:        u.Username,
			Nickname:        u.Nickname,
			Avatar:          u.Avatar,
			BackgroundImage: u.BackgroundImage,
			Signature:       u.Signature,
			FollowCount:     u.FollowCount,
			FollowerCount:   u.FollowerCount,
			TotalFavorited:  u.TotalFavorited,
			WorkCount:       u.WorkCount,
			FavoriteCount:   u.FavoriteCount,
			IsFollow:        followMap[u.ID],
		}
		result = append(result, user)
	}

	return result, total, nil
}

func (r *relationRepo) GetFriendList(ctx context.Context, userID int64) ([]*biz.User, error) {
	// 获取互相关注的用户ID
	var friendIDs []int64
	err := r.data.db.WithContext(ctx).Raw(`
        SELECT f1.follow_user_id 
        FROM user_follows f1 
        INNER JOIN user_follows f2 ON f1.follow_user_id = f2.user_id 
        WHERE f1.user_id = ? AND f2.follow_user_id = ?
    `, userID, userID).Scan(&friendIDs).Error

	if err != nil {
		return nil, err
	}

	if len(friendIDs) == 0 {
		return []*biz.User{}, nil
	}

	// 获取好友信息
	var users []User
	if err := r.data.db.WithContext(ctx).
		Where("id IN ? AND status = 1", friendIDs).
		Find(&users).Error; err != nil {
		return nil, err
	}

	// 转换为业务模型
	result := make([]*biz.User, 0, len(users))
	for _, u := range users {
		user := &biz.User{
			ID:              u.ID,
			Username:        u.Username,
			Nickname:        u.Nickname,
			Avatar:          u.Avatar,
			BackgroundImage: u.BackgroundImage,
			Signature:       u.Signature,
			FollowCount:     u.FollowCount,
			FollowerCount:   u.FollowerCount,
			TotalFavorited:  u.TotalFavorited,
			WorkCount:       u.WorkCount,
			FavoriteCount:   u.FavoriteCount,
			IsFollow:        true, // 好友列表中的都是互相关注
		}
		result = append(result, user)
	}

	return result, nil
}

// 缓存相关方法
func (r *relationRepo) getFollowCache(ctx context.Context, userID, followUserID int64) string {
	key := fmt.Sprintf("follow:%d:%d", userID, followUserID)
	val, _ := r.data.rdb.Get(ctx, key).Result()
	return val
}

func (r *relationRepo) setFollowCache(ctx context.Context, userID, followUserID int64, isFollowing bool) {
	key := fmt.Sprintf("follow:%d:%d", userID, followUserID)
	val := "0"
	if isFollowing {
		val = "1"
	}
	r.data.rdb.Set(ctx, key, val, 10*time.Minute)
}

func (r *relationRepo) clearRelationCache(ctx context.Context, userID, followUserID int64) {
	keys := []string{
		fmt.Sprintf("follow:%d:%d", userID, followUserID),
		fmt.Sprintf("follow:%d:*", userID),
		fmt.Sprintf("follow:*:%d", followUserID),
	}
	r.data.rdb.Del(ctx, keys...)
}
