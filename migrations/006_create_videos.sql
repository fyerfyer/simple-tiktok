-- +migrate Up
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
  KEY `idx_status` (`status`)
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
  KEY `idx_video_id` (`video_id`)
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
  KEY `idx_parent_id` (`parent_id`)
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
  KEY `idx_to_created` (`to_user_id`,`created_at` DESC)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- +migrate Down
DROP TABLE IF EXISTS `messages`;
DROP TABLE IF EXISTS `comments`;
DROP TABLE IF EXISTS `user_favorites`;
DROP TABLE IF EXISTS `videos`;