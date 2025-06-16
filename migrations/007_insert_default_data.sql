-- +migrate Up
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

-- 插入测试用户（可选，用于开发测试）
-- 注意：生产环境不应该包含测试数据

-- +migrate Down
DELETE FROM `role_permissions` WHERE `role_id` IN (1, 2, 3);