-- +goose Up
-- 成就/好友/社交相关表

-- 成就定义表
CREATE TABLE `achievements` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `key` VARCHAR(64) NOT NULL COMMENT '唯一标识如 first_win',
  `name` VARCHAR(128) NOT NULL COMMENT '成就名称',
  `description` VARCHAR(512) NOT NULL DEFAULT '' COMMENT '成就描述',
  `icon` VARCHAR(512) NOT NULL DEFAULT '' COMMENT '成就图标URL',
  `category` VARCHAR(32) NOT NULL DEFAULT '' COMMENT '分类: wins/streak/special/match/social/tournament',
  `threshold` INT NOT NULL DEFAULT 1 COMMENT '达成条件值',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_key` (`key`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='成就定义表';

-- 用户成就进度表
CREATE TABLE `user_achievements` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `user_id` BIGINT UNSIGNED NOT NULL COMMENT '用户ID',
  `achievement_id` BIGINT UNSIGNED NOT NULL COMMENT '成就ID',
  `progress` INT NOT NULL DEFAULT 0 COMMENT '当前进度',
  `unlocked` TINYINT NOT NULL DEFAULT 0 COMMENT '是否已解锁 0=否 1=是',
  `unlocked_at` DATETIME DEFAULT NULL COMMENT '解锁时间',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_user_achievement` (`user_id`, `achievement_id`),
  KEY `idx_user_id` (`user_id`),
  KEY `idx_achievement_id` (`achievement_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='用户成就进度表';

-- 用户称号表
CREATE TABLE `user_titles` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `user_id` BIGINT UNSIGNED NOT NULL COMMENT '用户ID',
  `title_name` VARCHAR(64) NOT NULL COMMENT '称号名称',
  `source` VARCHAR(32) NOT NULL DEFAULT '' COMMENT '来源: achievement/season/tournament',
  `equipped` TINYINT NOT NULL DEFAULT 0 COMMENT '是否装备中 0=否 1=是',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_user_id` (`user_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='用户称号表';

-- 好友表（双向记录）
CREATE TABLE `friends` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `user_id` BIGINT UNSIGNED NOT NULL COMMENT '用户ID',
  `friend_id` BIGINT UNSIGNED NOT NULL COMMENT '好友ID',
  `status` TINYINT NOT NULL DEFAULT 1 COMMENT '状态 1=正常',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_user_friend` (`user_id`, `friend_id`),
  KEY `idx_user_id` (`user_id`),
  KEY `idx_friend_id` (`friend_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='好友关系表';

-- 好友请求表
CREATE TABLE `friend_requests` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `from_user_id` BIGINT UNSIGNED NOT NULL COMMENT '发起方用户ID',
  `to_user_id` BIGINT UNSIGNED NOT NULL COMMENT '接收方用户ID',
  `status` TINYINT NOT NULL DEFAULT 0 COMMENT '状态 0=待处理 1=已接受 2=已拒绝',
  `message` VARCHAR(256) NOT NULL DEFAULT '' COMMENT '附加消息',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_from_user` (`from_user_id`),
  KEY `idx_to_user` (`to_user_id`),
  KEY `idx_status` (`status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='好友请求表';

-- 关注表
CREATE TABLE `follows` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `follower_id` BIGINT UNSIGNED NOT NULL COMMENT '关注者ID',
  `following_id` BIGINT UNSIGNED NOT NULL COMMENT '被关注者ID',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_follower_following` (`follower_id`, `following_id`),
  KEY `idx_follower_id` (`follower_id`),
  KEY `idx_following_id` (`following_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='关注关系表';

-- 社交动态表
CREATE TABLE `social_posts` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `user_id` BIGINT UNSIGNED NOT NULL COMMENT '发布者ID',
  `content` TEXT NOT NULL COMMENT '动态内容',
  `images` JSON DEFAULT NULL COMMENT '图片URL数组',
  `post_type` TINYINT NOT NULL DEFAULT 3 COMMENT '类型 1=战绩 2=打卡 3=日常',
  `match_id` BIGINT UNSIGNED DEFAULT NULL COMMENT '关联对局ID',
  `likes_count` INT NOT NULL DEFAULT 0 COMMENT '点赞数',
  `comments_count` INT NOT NULL DEFAULT 0 COMMENT '评论数',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_user_id` (`user_id`),
  KEY `idx_created_at` (`created_at`),
  KEY `idx_post_type` (`post_type`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='社交动态表';

-- 动态点赞表
CREATE TABLE `social_post_likes` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `post_id` BIGINT UNSIGNED NOT NULL COMMENT '动态ID',
  `user_id` BIGINT UNSIGNED NOT NULL COMMENT '点赞用户ID',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_post_user` (`post_id`, `user_id`),
  KEY `idx_post_id` (`post_id`),
  KEY `idx_user_id` (`user_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='动态点赞表';

-- 动态评论表
CREATE TABLE `social_post_comments` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `post_id` BIGINT UNSIGNED NOT NULL COMMENT '动态ID',
  `user_id` BIGINT UNSIGNED NOT NULL COMMENT '评论用户ID',
  `content` VARCHAR(1024) NOT NULL COMMENT '评论内容',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_post_id` (`post_id`),
  KEY `idx_user_id` (`user_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='动态评论表';

-- +goose Down
DROP TABLE IF EXISTS `social_post_comments`;
DROP TABLE IF EXISTS `social_post_likes`;
DROP TABLE IF EXISTS `social_posts`;
DROP TABLE IF EXISTS `follows`;
DROP TABLE IF EXISTS `friend_requests`;
DROP TABLE IF EXISTS `friends`;
DROP TABLE IF EXISTS `user_titles`;
DROP TABLE IF EXISTS `user_achievements`;
DROP TABLE IF EXISTS `achievements`;
