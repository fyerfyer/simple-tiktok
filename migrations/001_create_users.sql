-- +migrate Up
CREATE TABLE `users` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `username` varchar(32) NOT NULL COMMENT 'Username',
  `password_hash` varchar(128) NOT NULL COMMENT 'Argon2id hash',
  `salt` varchar(32) NOT NULL COMMENT 'Random salt',
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
  `last_login_at` timestamp NULL,
  `created_at` timestamp DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_username` (`username`),
  KEY `idx_created_at` (`created_at`),
  KEY `idx_status` (`status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- +migrate Down
DROP TABLE IF EXISTS `users`;