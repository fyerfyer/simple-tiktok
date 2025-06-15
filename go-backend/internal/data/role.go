package data

import (
	"context"
	"fmt"
	"time"

	"go-backend/internal/domain"

	"github.com/go-kratos/kratos/v2/log"
	"gorm.io/gorm"
)

// Role 角色模型
type Role struct {
	ID          int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	Name        string    `gorm:"uniqueIndex;size:50;not null" json:"name"`
	Description string    `gorm:"size:200" json:"description"`
	Status      int8      `gorm:"default:1" json:"status"`
	CreatedAt   time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

func (Role) TableName() string {
	return "roles"
}

// UserRole 用户角色关联模型
type UserRole struct {
	ID        int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID    int64     `gorm:"not null;index:uk_user_role,priority:1" json:"user_id"`
	RoleID    int64     `gorm:"not null;index:uk_user_role,priority:2" json:"role_id"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
}

func (UserRole) TableName() string {
	return "user_roles"
}

// RoleRepo 角色仓储实现
type RoleRepo struct {
	data *Data
	log  *log.Helper
}

// NewRoleRepo 创建角色仓储
func NewRoleRepo(data *Data, logger log.Logger) *RoleRepo {
	return &RoleRepo{
		data: data,
		log:  log.NewHelper(logger),
	}
}

// 实现 biz.RoleRepo 接口
func (r *RoleRepo) GetRole(ctx context.Context, roleID int64) (*domain.Role, error) {
	var role Role
	if err := r.data.db.WithContext(ctx).Where("id = ? AND status = 1", roleID).First(&role).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("role not found")
		}
		return nil, err
	}

	return r.convertToRole(&role), nil
}

func (r *RoleRepo) GetRoleByName(ctx context.Context, name string) (*domain.Role, error) {
	var role Role
	if err := r.data.db.WithContext(ctx).Where("name = ? AND status = 1", name).First(&role).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("role not found")
		}
		return nil, err
	}

	return r.convertToRole(&role), nil
}

func (r *RoleRepo) GetUserRoles(ctx context.Context, userID int64) ([]*domain.Role, error) {
	var userRoles []UserRole
	if err := r.data.db.WithContext(ctx).Where("user_id = ?", userID).Find(&userRoles).Error; err != nil {
		return nil, err
	}

	if len(userRoles) == 0 {
		return []*domain.Role{}, nil
	}

	roleIDs := make([]int64, 0, len(userRoles))
	for _, ur := range userRoles {
		roleIDs = append(roleIDs, ur.RoleID)
	}

	var roles []Role
	if err := r.data.db.WithContext(ctx).Where("id IN ? AND status = 1", roleIDs).Find(&roles).Error; err != nil {
		return nil, err
	}

	result := make([]*domain.Role, 0, len(roles))
	for _, role := range roles {
		result = append(result, r.convertToRole(&role))
	}

	return result, nil
}

func (r *RoleRepo) AssignRole(ctx context.Context, userID, roleID int64) error {
	// 检查是否已存在
	var count int64
	r.data.db.WithContext(ctx).Model(&UserRole{}).
		Where("user_id = ? AND role_id = ?", userID, roleID).
		Count(&count)

	if count > 0 {
		return nil // 已存在，不重复添加
	}

	userRole := &UserRole{
		UserID: userID,
		RoleID: roleID,
	}

	return r.data.db.WithContext(ctx).Create(userRole).Error
}

func (r *RoleRepo) RemoveRole(ctx context.Context, userID, roleID int64) error {
	return r.data.db.WithContext(ctx).
		Where("user_id = ? AND role_id = ?", userID, roleID).
		Delete(&UserRole{}).Error
}

func (r *RoleRepo) HasRole(ctx context.Context, userID, roleID int64) (bool, error) {
	var count int64
	err := r.data.db.WithContext(ctx).Model(&UserRole{}).
		Where("user_id = ? AND role_id = ?", userID, roleID).
		Count(&count).Error

	return count > 0, err
}

func (r *RoleRepo) convertToRole(role *Role) *domain.Role {
	return &domain.Role{
		ID:          role.ID,
		Name:        role.Name,
		Description: role.Description,
		Status:      role.Status,
		CreatedAt:   role.CreatedAt,
		UpdatedAt:   role.UpdatedAt,
	}
}
