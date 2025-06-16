-- +migrate Up
CREATE TABLE `roles` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `name` varchar(50) NOT NULL COMMENT 'Role name',
  `description` varchar(200) DEFAULT '' COMMENT 'Role description',
  `status` tinyint DEFAULT '1' COMMENT 'Role status: 1-active, 2-inactive',
  `created_at` timestamp DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_name` (`name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 插入基础角色数据
INSERT INTO `roles` (`name`, `description`) VALUES
('user', 'Regular user'),
('admin', 'Administrator'),
('moderator', 'Content moderator');

-- +migrate Down
DROP TABLE IF EXISTS `roles`;