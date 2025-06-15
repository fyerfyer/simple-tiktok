package data

import (
	"context"
	"fmt"
	"time"

	"go-backend/internal/biz"
	"go-backend/internal/domain"

	"github.com/go-kratos/kratos/v2/log"
	"gorm.io/gorm"
)

// Permission 权限模型
type Permission struct {
	ID          int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	Name        string    `gorm:"uniqueIndex;size:50;not null" json:"name"`
	Resource    string    `gorm:"size:100;not null" json:"resource"`
	Action      string    `gorm:"size:20;not null" json:"action"`
	Description string    `gorm:"size:200" json:"description"`
	Status      int8      `gorm:"default:1" json:"status"`
	CreatedAt   time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

func (Permission) TableName() string {
	return "permissions"
}

// RolePermission 角色权限关联模型
type RolePermission struct {
	ID           int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	RoleID       int64     `gorm:"not null;index:uk_role_permission,priority:1" json:"role_id"`
	PermissionID int64     `gorm:"not null;index:uk_role_permission,priority:2" json:"permission_id"`
	CreatedAt    time.Time `gorm:"autoCreateTime" json:"created_at"`
}

func (RolePermission) TableName() string {
	return "role_permissions"
}

// PermissionRepo 权限仓储实现
type PermissionRepo struct {
	data     *Data
	roleRepo biz.RoleRepo
	log      *log.Helper
}

// NewPermissionRepo 创建权限仓储
func NewPermissionRepo(data *Data, roleRepo biz.RoleRepo, logger log.Logger) *PermissionRepo {
	return &PermissionRepo{
		data:     data,
		roleRepo: roleRepo,
		log:      log.NewHelper(logger),
	}
}

// 实现 biz.PermissionRepo 接口
func (r *PermissionRepo) GetPermission(ctx context.Context, permissionID int64) (*domain.Permission, error) {
	var perm Permission
	if err := r.data.db.WithContext(ctx).Where("id = ? AND status = 1", permissionID).First(&perm).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("permission not found")
		}
		return nil, err
	}

	return r.convertToPermission(&perm), nil
}

func (r *PermissionRepo) GetRolePermissions(ctx context.Context, roleID int64) ([]*domain.Permission, error) {
	var rolePerms []RolePermission
	if err := r.data.db.WithContext(ctx).Where("role_id = ?", roleID).Find(&rolePerms).Error; err != nil {
		return nil, err
	}

	if len(rolePerms) == 0 {
		return []*domain.Permission{}, nil
	}

	permissionIDs := make([]int64, 0, len(rolePerms))
	for _, rp := range rolePerms {
		permissionIDs = append(permissionIDs, rp.PermissionID)
	}

	var permissions []Permission
	if err := r.data.db.WithContext(ctx).Where("id IN ? AND status = 1", permissionIDs).Find(&permissions).Error; err != nil {
		return nil, err
	}

	result := make([]*domain.Permission, 0, len(permissions))
	for _, perm := range permissions {
		result = append(result, r.convertToPermission(&perm))
	}

	return result, nil
}

func (r *PermissionRepo) GetUserPermissions(ctx context.Context, userID int64) ([]*domain.Permission, error) {
	// 获取用户角色
	roles, err := r.roleRepo.GetUserRoles(ctx, userID)
	if err != nil {
		return nil, err
	}

	if len(roles) == 0 {
		return []*domain.Permission{}, nil
	}

	// 收集所有权限
	permissionMap := make(map[int64]*domain.Permission)
	for _, role := range roles {
		permissions, err := r.GetRolePermissions(ctx, role.ID)
		if err != nil {
			continue
		}

		for _, perm := range permissions {
			permissionMap[perm.ID] = perm
		}
	}

	// 转换为数组
	result := make([]*domain.Permission, 0, len(permissionMap))
	for _, perm := range permissionMap {
		result = append(result, perm)
	}

	return result, nil
}

func (r *PermissionRepo) HasPermission(ctx context.Context, userID int64, resource, action string) (bool, error) {
	permissions, err := r.GetUserPermissions(ctx, userID)
	if err != nil {
		return false, err
	}

	for _, perm := range permissions {
		if perm.Match(resource, action) {
			return true, nil
		}
	}

	return false, nil
}

func (r *PermissionRepo) convertToPermission(perm *Permission) *domain.Permission {
	return &domain.Permission{
		ID:          perm.ID,
		Name:        perm.Name,
		Resource:    perm.Resource,
		Action:      perm.Action,
		Description: perm.Description,
		Status:      perm.Status,
		CreatedAt:   perm.CreatedAt,
		UpdatedAt:   perm.UpdatedAt,
	}
}
