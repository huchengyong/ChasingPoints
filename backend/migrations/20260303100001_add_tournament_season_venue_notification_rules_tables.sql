-- +goose Up
-- 赛事/赛季/球馆/通知/规则相关表

-- 赛事表
CREATE TABLE IF NOT EXISTS `tournaments` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `creator_id` BIGINT UNSIGNED NOT NULL COMMENT '创建者ID',
  `name` VARCHAR(128) NOT NULL COMMENT '赛事名称',
  `description` TEXT DEFAULT NULL COMMENT '赛事描述',
  `game_type` TINYINT NOT NULL COMMENT '球种 1=斯诺克 2=九球追分 3=中式八球',
  `format` TINYINT NOT NULL DEFAULT 1 COMMENT '赛制 1=单败淘汰 2=双败淘汰 3=循环赛 4=瑞士轮',
  `max_players` INT NOT NULL DEFAULT 16 COMMENT '最大参赛人数(最多64)',
  `current_players` INT NOT NULL DEFAULT 0 COMMENT '当前报名人数',
  `status` TINYINT NOT NULL DEFAULT 0 COMMENT '状态 0=报名中 1=进行中 2=已结束 3=已取消',
  `city` VARCHAR(64) NOT NULL DEFAULT '' COMMENT '城市',
  `venue_name` VARCHAR(128) NOT NULL DEFAULT '' COMMENT '场馆名称',
  `start_time` DATETIME DEFAULT NULL COMMENT '开始时间',
  `end_time` DATETIME DEFAULT NULL COMMENT '结束时间',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_creator_id` (`creator_id`),
  KEY `idx_status` (`status`),
  KEY `idx_game_type` (`game_type`),
  KEY `idx_city` (`city`),
  KEY `idx_start_time` (`start_time`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='赛事表';

-- 赛事参赛者表
CREATE TABLE IF NOT EXISTS `tournament_participants` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `tournament_id` BIGINT UNSIGNED NOT NULL COMMENT '赛事ID',
  `user_id` BIGINT UNSIGNED NOT NULL COMMENT '用户ID',
  `seed` INT NOT NULL DEFAULT 0 COMMENT '种子排名',
  `status` TINYINT NOT NULL DEFAULT 0 COMMENT '状态 0=已报名 1=已签到 2=已淘汰 3=冠军',
  `final_rank` INT NOT NULL DEFAULT 0 COMMENT '最终排名',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_tournament_user` (`tournament_id`, `user_id`),
  KEY `idx_tournament_id` (`tournament_id`),
  KEY `idx_user_id` (`user_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='赛事参赛者表';

-- 赛事对阵表
CREATE TABLE IF NOT EXISTS `tournament_matches` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `tournament_id` BIGINT UNSIGNED NOT NULL COMMENT '赛事ID',
  `round_number` INT NOT NULL DEFAULT 1 COMMENT '轮次',
  `match_order` INT NOT NULL DEFAULT 0 COMMENT '本轮内顺序',
  `player1_id` BIGINT UNSIGNED DEFAULT NULL COMMENT '选手1 ID',
  `player2_id` BIGINT UNSIGNED DEFAULT NULL COMMENT '选手2 ID',
  `winner_id` BIGINT UNSIGNED DEFAULT NULL COMMENT '胜者ID',
  `match_id` BIGINT UNSIGNED DEFAULT NULL COMMENT '关联 matches 表的对局ID',
  `bracket_position` VARCHAR(32) NOT NULL DEFAULT '' COMMENT '对阵位置标识',
  `status` TINYINT NOT NULL DEFAULT 0 COMMENT '状态 0=待开始 1=进行中 2=已完成',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_tournament_id` (`tournament_id`),
  KEY `idx_round_number` (`tournament_id`, `round_number`),
  KEY `idx_match_id` (`match_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='赛事对阵表';

-- 赛季表
CREATE TABLE IF NOT EXISTS `seasons` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `name` VARCHAR(64) NOT NULL COMMENT '赛季名称',
  `start_date` DATE NOT NULL COMMENT '开始日期',
  `end_date` DATE NOT NULL COMMENT '结束日期',
  `status` TINYINT NOT NULL DEFAULT 0 COMMENT '状态 0=未开始 1=进行中 2=已结束',
  `rank_reset_ratio` DECIMAL(3,2) NOT NULL DEFAULT 0.70 COMMENT '段位重置保留比例',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_status` (`status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='赛季表';

-- 赛季记录表
CREATE TABLE IF NOT EXISTS `season_records` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `season_id` BIGINT UNSIGNED NOT NULL COMMENT '赛季ID',
  `user_id` BIGINT UNSIGNED NOT NULL COMMENT '用户ID',
  `start_rank_score` INT NOT NULL DEFAULT 0 COMMENT '赛季初始排位分',
  `end_rank_score` INT NOT NULL DEFAULT 0 COMMENT '赛季结束排位分',
  `peak_rank_score` INT NOT NULL DEFAULT 0 COMMENT '赛季峰值排位分',
  `matches_played` INT NOT NULL DEFAULT 0 COMMENT '赛季对局数',
  `wins` INT NOT NULL DEFAULT 0 COMMENT '赛季胜场',
  `final_rank` INT NOT NULL DEFAULT 0 COMMENT '赛季最终排名',
  `rewards` JSON DEFAULT NULL COMMENT '赛季奖励JSON',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_season_user` (`season_id`, `user_id`),
  KEY `idx_season_id` (`season_id`),
  KEY `idx_user_id` (`user_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='赛季记录表';

-- 球馆表
CREATE TABLE IF NOT EXISTS `venues` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `name` VARCHAR(128) NOT NULL COMMENT '球馆名称',
  `address` VARCHAR(512) NOT NULL DEFAULT '' COMMENT '详细地址',
  `city` VARCHAR(64) NOT NULL DEFAULT '' COMMENT '城市',
  `district` VARCHAR(64) NOT NULL DEFAULT '' COMMENT '区县',
  `latitude` DECIMAL(10,7) DEFAULT NULL COMMENT '纬度',
  `longitude` DECIMAL(10,7) DEFAULT NULL COMMENT '经度',
  `phone` VARCHAR(32) NOT NULL DEFAULT '' COMMENT '联系电话',
  `images` JSON DEFAULT NULL COMMENT '球馆照片URL数组',
  `business_hours` VARCHAR(128) NOT NULL DEFAULT '' COMMENT '营业时间',
  `table_count` INT NOT NULL DEFAULT 0 COMMENT '球台数量',
  `price_range` VARCHAR(64) NOT NULL DEFAULT '' COMMENT '台费范围',
  `description` TEXT DEFAULT NULL COMMENT '球馆描述',
  `owner_user_id` BIGINT UNSIGNED DEFAULT NULL COMMENT '认证球馆店主用户ID',
  `status` TINYINT NOT NULL DEFAULT 0 COMMENT '状态 0=待审核 1=已通过',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_city` (`city`),
  KEY `idx_district` (`city`, `district`),
  KEY `idx_status` (`status`),
  KEY `idx_owner` (`owner_user_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='球馆表';

-- 球馆签到表
CREATE TABLE IF NOT EXISTS `venue_checkins` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `venue_id` BIGINT UNSIGNED NOT NULL COMMENT '球馆ID',
  `user_id` BIGINT UNSIGNED NOT NULL COMMENT '用户ID',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_venue_id` (`venue_id`),
  KEY `idx_user_id` (`user_id`),
  KEY `idx_venue_user_date` (`venue_id`, `user_id`, `created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='球馆签到表';

-- 通知表
CREATE TABLE IF NOT EXISTS `notifications` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `user_id` BIGINT UNSIGNED NOT NULL COMMENT '接收用户ID',
  `type` VARCHAR(32) NOT NULL DEFAULT '' COMMENT '类型: challenge/tournament/rank_change/friend_request/system',
  `title` VARCHAR(128) NOT NULL DEFAULT '' COMMENT '标题',
  `content` VARCHAR(512) NOT NULL DEFAULT '' COMMENT '内容',
  `data` JSON DEFAULT NULL COMMENT '扩展数据',
  `is_read` TINYINT NOT NULL DEFAULT 0 COMMENT '是否已读 0=未读 1=已读',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_user_id` (`user_id`),
  KEY `idx_user_read` (`user_id`, `is_read`),
  KEY `idx_type` (`type`),
  KEY `idx_created_at` (`created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='通知表';

-- 规则内容表
CREATE TABLE IF NOT EXISTS `rules_content` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `category` VARCHAR(32) NOT NULL COMMENT '球种分类: snooker/nine_ball/chinese_eight',
  `content_type` VARCHAR(16) NOT NULL COMMENT '内容类型: rule/foul/glossary',
  `title` VARCHAR(128) NOT NULL COMMENT '标题',
  `content` TEXT NOT NULL COMMENT '正文(富文本)',
  `sort_order` INT NOT NULL DEFAULT 0 COMMENT '排序',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_category_type` (`category`, `content_type`),
  KEY `idx_sort_order` (`sort_order`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='规则内容表';

-- +goose Down
DROP TABLE IF EXISTS `rules_content`;
DROP TABLE IF EXISTS `notifications`;
DROP TABLE IF EXISTS `venue_checkins`;
DROP TABLE IF EXISTS `venues`;
DROP TABLE IF EXISTS `season_records`;
DROP TABLE IF EXISTS `seasons`;
DROP TABLE IF EXISTS `tournament_matches`;
DROP TABLE IF EXISTS `tournament_participants`;
DROP TABLE IF EXISTS `tournaments`;
