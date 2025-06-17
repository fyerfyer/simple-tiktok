SET NAMES utf8mb4;
SET FOREIGN_KEY_CHECKS = 0;

-- 创建数据库
CREATE DATABASE IF NOT EXISTS `tiktok` DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
USE `tiktok`;

-- 用户表
CREATE TABLE `users` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `username` varchar(32) NOT NULL COMMENT 'Username',
  `password_hash` varchar(128) NOT NULL COMMENT 'Password hash',
  `salt` varchar(32) NOT NULL COMMENT 'Password salt',
  `nickname` varchar(50) DEFAULT NULL COMMENT 'Display name',
  `avatar` varchar(255) DEFAULT 'https://example.com/default-avatar.jpg' COMMENT 'Avatar URL',
  `background_image` varchar(255) DEFAULT 'https://example.com/default-bg.jpg' COMMENT 'Background image URL',
  `signature` varchar(200) DEFAULT '' COMMENT 'User signature',
  `follow_count` int DEFAULT '0' COMMENT 'Following count',
  `follower_count` int DEFAULT '0' COMMENT 'Follower count',
  `total_favorited` bigint DEFAULT '0' COMMENT 'Total likes received',
  `work_count` int DEFAULT '0' COMMENT 'Video count',
  `favorite_count` int DEFAULT '0' COMMENT 'Liked video count',
  `status` tinyint DEFAULT '1' COMMENT 'User status: 1-active, 2-inactive',
  `last_login_at` timestamp NULL COMMENT 'Last login time',
  `created_at` timestamp DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_username` (`username`),
  KEY `idx_created_at` (`created_at`),
  KEY `idx_status` (`status`),
  KEY `idx_last_login` (`last_login_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 角色表
CREATE TABLE `roles` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `name` varchar(50) NOT NULL COMMENT 'Role name',
  `description` varchar(200) DEFAULT NULL COMMENT 'Role description',
  `status` tinyint DEFAULT '1' COMMENT 'Role status: 1-active, 2-inactive',
  `created_at` timestamp DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_name` (`name`),
  KEY `idx_status` (`status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 权限表
CREATE TABLE `permissions` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `name` varchar(50) NOT NULL COMMENT 'Permission name',
  `resource` varchar(100) NOT NULL COMMENT 'Resource path',
  `action` varchar(20) NOT NULL COMMENT 'Action type',
  `description` varchar(200) DEFAULT NULL COMMENT 'Permission description',
  `status` tinyint DEFAULT '1' COMMENT 'Permission status: 1-active, 2-inactive',
  `created_at` timestamp DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_name` (`name`),
  KEY `idx_status` (`status`),
  KEY `idx_resource_action` (`resource`, `action`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 用户角色关联表
CREATE TABLE `user_roles` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `user_id` bigint NOT NULL COMMENT 'User ID',
  `role_id` bigint NOT NULL COMMENT 'Role ID',
  `created_at` timestamp DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_user_role` (`user_id`,`role_id`),
  KEY `idx_user_id` (`user_id`),
  KEY `idx_role_id` (`role_id`),
  CONSTRAINT `fk_user_roles_user` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE,
  CONSTRAINT `fk_user_roles_role` FOREIGN KEY (`role_id`) REFERENCES `roles` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 角色权限关联表
CREATE TABLE `role_permissions` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `role_id` bigint NOT NULL COMMENT 'Role ID',
  `permission_id` bigint NOT NULL COMMENT 'Permission ID',
  `created_at` timestamp DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_role_permission` (`role_id`,`permission_id`),
  KEY `idx_role_id` (`role_id`),
  KEY `idx_permission_id` (`permission_id`),
  CONSTRAINT `fk_role_permissions_role` FOREIGN KEY (`role_id`) REFERENCES `roles` (`id`) ON DELETE CASCADE,
  CONSTRAINT `fk_role_permissions_permission` FOREIGN KEY (`permission_id`) REFERENCES `permissions` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 用户会话表 
CREATE TABLE `user_sessions` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `user_id` bigint NOT NULL COMMENT 'User ID',
  `refresh_token` varchar(255) NOT NULL COMMENT 'Refresh token',
  `expires_at` timestamp NOT NULL COMMENT 'Expiration time',
  `created_at` timestamp DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_refresh_token` (`refresh_token`),
  KEY `idx_user_id` (`user_id`),
  KEY `idx_expires_at` (`expires_at`),
  CONSTRAINT `fk_user_sessions_user` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Token黑名单表
CREATE TABLE `token_blacklist` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `token_id` varchar(255) NOT NULL COMMENT 'Token ID',
  `expires_at` timestamp NOT NULL COMMENT 'Expiration time',
  `created_at` timestamp DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_token_id` (`token_id`),
  KEY `idx_expires_at` (`expires_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 关注关系表 - 补全完整定义  
CREATE TABLE `user_follows` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `user_id` bigint NOT NULL COMMENT 'Follower user ID',
  `follow_user_id` bigint NOT NULL COMMENT 'Following user ID',
  `created_at` timestamp DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_user_follow` (`user_id`,`follow_user_id`),
  KEY `idx_user_id` (`user_id`),
  KEY `idx_follow_user_id` (`follow_user_id`),
  KEY `idx_created_at` (`created_at`),
  CONSTRAINT `fk_user_follows_user` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE,
  CONSTRAINT `fk_user_follows_follow_user` FOREIGN KEY (`follow_user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 视频表
CREATE TABLE `videos` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `author_id` bigint NOT NULL COMMENT 'Author user ID',
  `title` varchar(255) NOT NULL COMMENT 'Video title',
  `play_url` varchar(500) NOT NULL COMMENT 'Video play URL',
  `cover_url` varchar(500) DEFAULT NULL COMMENT 'Video cover URL',
  `favorite_count` int DEFAULT '0' COMMENT 'Like count',
  `comment_count` int DEFAULT '0' COMMENT 'Comment count',
  `play_count` bigint DEFAULT '0' COMMENT 'Play count',
  `status` tinyint DEFAULT '1' COMMENT 'Video status: 1-published, 2-private, 3-deleted',
  `created_at` timestamp DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_author_created` (`author_id`,`created_at` DESC),
  KEY `idx_created_at` (`created_at` DESC),
  KEY `idx_status` (`status`),
  CONSTRAINT `fk_videos_author` FOREIGN KEY (`author_id`) REFERENCES `users` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 点赞表
CREATE TABLE `user_favorites` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `user_id` bigint NOT NULL COMMENT 'User ID',
  `video_id` bigint NOT NULL COMMENT 'Video ID',
  `created_at` timestamp DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_user_video` (`user_id`,`video_id`),
  KEY `idx_user_id` (`user_id`),
  KEY `idx_video_id` (`video_id`),
  CONSTRAINT `fk_user_favorites_user` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE,
  CONSTRAINT `fk_user_favorites_video` FOREIGN KEY (`video_id`) REFERENCES `videos` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 评论表
CREATE TABLE `comments` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `video_id` bigint NOT NULL COMMENT 'Video ID',
  `user_id` bigint NOT NULL COMMENT 'Comment user ID',
  `parent_id` bigint DEFAULT '0' COMMENT 'Parent comment ID, 0 for root comment',
  `content` text NOT NULL COMMENT 'Comment content',
  `like_count` int DEFAULT '0' COMMENT 'Comment like count',
  `reply_count` int DEFAULT '0' COMMENT 'Reply count',
  `status` tinyint DEFAULT '1' COMMENT 'Comment status: 1-normal, 2-deleted',
  `created_at` timestamp DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_video_created` (`video_id`,`created_at` DESC),
  KEY `idx_user_id` (`user_id`),
  KEY `idx_parent_id` (`parent_id`),
  CONSTRAINT `fk_comments_video` FOREIGN KEY (`video_id`) REFERENCES `videos` (`id`) ON DELETE CASCADE,
  CONSTRAINT `fk_comments_user` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 消息表
CREATE TABLE `messages` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `from_user_id` bigint NOT NULL COMMENT 'Sender user ID',
  `to_user_id` bigint NOT NULL COMMENT 'Receiver user ID',
  `content` varchar(500) NOT NULL COMMENT 'Message content',
  `message_type` tinyint DEFAULT '1' COMMENT 'Message type: 1-text',
  `status` tinyint DEFAULT '1' COMMENT 'Message status: 1-sent, 2-read',
  `created_at` timestamp DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_from_to_created` (`from_user_id`,`to_user_id`,`created_at` DESC),
  KEY `idx_to_created` (`to_user_id`,`created_at` DESC),
  CONSTRAINT `fk_messages_from_user` FOREIGN KEY (`from_user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE,
  CONSTRAINT `fk_messages_to_user` FOREIGN KEY (`to_user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 插入基础角色数据
INSERT INTO `roles` (`name`, `description`) VALUES
('user', 'Regular user'),
('admin', 'Administrator'),
('moderator', 'Content moderator');

-- 插入基础权限数据
INSERT INTO `permissions` (`name`, `resource`, `action`, `description`) VALUES
('user:read', '/user', 'GET', 'Read user information'),
('user:update', '/user', 'PUT', 'Update user information'),
('user:create', '/user', 'POST', 'Create user'),
('user:delete', '/user', 'DELETE', 'Delete user'),
('video:create', '/video', 'POST', 'Create video'),
('video:read', '/video', 'GET', 'Read video information'),
('video:update', '/video', 'PUT', 'Update video'),
('video:delete', '/video', 'DELETE', 'Delete video'),
('comment:create', '/comment', 'POST', 'Create comment'),
('comment:read', '/comment', 'GET', 'Read comment'),
('comment:update', '/comment', 'PUT', 'Update comment'),
('comment:delete', '/comment', 'DELETE', 'Delete comment'),
('admin:all', '/*', '*', 'Administrator full access');

-- 分配权限给角色
-- 用户角色权限
INSERT INTO `role_permissions` (`role_id`, `permission_id`) 
SELECT r.id, p.id FROM `roles` r, `permissions` p 
WHERE r.name = 'user' AND p.name IN ('user:read', 'user:update', 'video:create', 'video:read', 'comment:create', 'comment:read');

-- 管理员角色权限
INSERT INTO `role_permissions` (`role_id`, `permission_id`) 
SELECT r.id, p.id FROM `roles` r, `permissions` p 
WHERE r.name = 'admin' AND p.name = 'admin:all';

-- 审核员角色权限
INSERT INTO `role_permissions` (`role_id`, `permission_id`) 
SELECT r.id, p.id FROM `roles` r, `permissions` p 
WHERE r.name = 'moderator' AND p.name IN ('video:read', 'video:update', 'video:delete', 'comment:read', 'comment:update', 'comment:delete');

SET FOREIGN_KEY_CHECKS = 1;