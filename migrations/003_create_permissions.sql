-- +migrate Up
CREATE TABLE `permissions` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `name` varchar(50) NOT NULL COMMENT 'Permission name',
  `resource` varchar(100) NOT NULL COMMENT 'Resource path',
  `action` varchar(20) NOT NULL COMMENT 'Action type',
  `description` varchar(200) DEFAULT '' COMMENT 'Permission description',
  `status` tinyint DEFAULT '1' COMMENT 'Permission status: 1-active, 2-inactive',
  `created_at` timestamp DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_name` (`name`),
  KEY `idx_resource_action` (`resource`, `action`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 角色权限关联表
CREATE TABLE `role_permissions` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `role_id` bigint NOT NULL COMMENT 'Role ID',
  `permission_id` bigint NOT NULL COMMENT 'Permission ID',
  `created_at` timestamp DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_role_permission` (`role_id`, `permission_id`),
  KEY `idx_role_id` (`role_id`),
  KEY `idx_permission_id` (`permission_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 用户角色关联表
CREATE TABLE `user_roles` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `user_id` bigint NOT NULL COMMENT 'User ID',
  `role_id` bigint NOT NULL COMMENT 'Role ID',
  `created_at` timestamp DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_user_role` (`user_id`, `role_id`),
  KEY `idx_user_id` (`user_id`),
  KEY `idx_role_id` (`role_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 插入基础权限数据
INSERT INTO `permissions` (`name`, `resource`, `action`, `description`) VALUES
('user:read', '/user', 'GET', 'Read user info'),
('user:update', '/user', 'PUT', 'Update user info'),
('video:create', '/video', 'POST', 'Upload video'),
('video:read', '/video', 'GET', 'View video'),
('video:delete', '/video', 'DELETE', 'Delete video'),
('comment:create', '/comment', 'POST', 'Create comment'),
('comment:delete', '/comment', 'DELETE', 'Delete comment'),
('admin:all', '/*', '*', 'Admin full access');

-- 分配权限给角色
-- 普通用户权限 (role_id=1)
INSERT INTO `role_permissions` (`role_id`, `permission_id`) VALUES
(1, 1), -- user:read
(1, 2), -- user:update
(1, 3), -- video:create
(1, 4), -- video:read
(1, 6); -- comment:create

-- 管理员权限 (role_id=2)
INSERT INTO `role_permissions` (`role_id`, `permission_id`) VALUES
(2, 8); -- admin:all

-- 内容审核员权限 (role_id=3)
INSERT INTO `role_permissions` (`role_id`, `permission_id`) VALUES
(3, 1), -- user:read
(3, 4), -- video:read
(3, 5), -- video:delete
(3, 7); -- comment:delete

-- +migrate Down
DROP TABLE IF EXISTS `role_permissions`;
DROP TABLE IF EXISTS `user_roles`;
DROP TABLE IF EXISTS `permissions`;