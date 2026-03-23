-- +goose Up
-- 对局记录相关表

-- 对局记录表
CREATE TABLE IF NOT EXISTS `matches` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '对局ID',
  `user_id` bigint unsigned NOT NULL COMMENT '用户ID',
  `opponent_id` bigint unsigned DEFAULT NULL COMMENT '对手ID（已注册用户）',
  `opponent_name` varchar(50) NOT NULL COMMENT '对手昵称',
  `game_type` tinyint NOT NULL COMMENT '比赛类型：1=斯诺克 2=九球追分 3=中式八球',
  `game_mode` varchar(20) DEFAULT NULL COMMENT '比赛模式：让球数/单局决胜/目标分等',
  `my_score` int NOT NULL DEFAULT 0 COMMENT '我的总得分（局数或分数）',
  `opponent_score` int NOT NULL DEFAULT 0 COMMENT '对手总得分',
  `current_frame_my_score` int NOT NULL DEFAULT 0 COMMENT '当前frame我方分数',
  `current_frame_opponent_score` int NOT NULL DEFAULT 0 COMMENT '当前frame对手分数',
  `current_frame_started` tinyint(1) NOT NULL DEFAULT 0 COMMENT '当前frame是否已开始',
  `status` tinyint NOT NULL DEFAULT 1 COMMENT '状态：1=进行中 2=已完成 3=已取消',
  `result` tinyint DEFAULT NULL COMMENT '比赛结果：1=胜利 2=失败 3=平局',
  `match_time` datetime NOT NULL COMMENT '比赛开始时间',
  `end_time` datetime DEFAULT NULL COMMENT '比赛结束时间',
  `remark` varchar(500) DEFAULT NULL COMMENT '备注',
  `sync_revision` bigint unsigned NOT NULL DEFAULT 0 COMMENT '对局同步版本号',
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `deleted_at` datetime DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_user_id` (`user_id`),
  KEY `idx_opponent_id` (`opponent_id`),
  KEY `idx_status` (`status`),
  KEY `idx_match_time` (`match_time`),
  KEY `idx_deleted_at` (`deleted_at`),
  KEY `idx_user_opponent_status` (`user_id`, `opponent_id`, `status`),
  KEY `idx_status_deleted` (`status`, `deleted_at`),
  KEY `idx_opponent_status` (`opponent_id`, `status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='对局记录表';

-- 局记录表
CREATE TABLE IF NOT EXISTS `match_rounds` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '记录ID',
  `match_id` bigint unsigned NOT NULL COMMENT '对局ID',
  `round_no` int NOT NULL COMMENT '局数/回合数',
  `my_score` int NOT NULL DEFAULT 0 COMMENT '我的得分',
  `opponent_score` int NOT NULL DEFAULT 0 COMMENT '对手得分',
  `winner` tinyint DEFAULT NULL COMMENT '本局获胜方：1=我 2=对手',
  `win_type` varchar(20) DEFAULT NULL COMMENT '获胜方式：normal/break_clear/run_out等',
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_match_id` (`match_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='局记录表';

-- 操作日志表（支持撤销）
CREATE TABLE IF NOT EXISTS `match_actions` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '记录ID',
  `match_id` bigint unsigned NOT NULL COMMENT '对局ID',
  `round_no` int NOT NULL COMMENT '当前局数',
  `action_type` varchar(20) NOT NULL COMMENT '操作类型：score/foul/win等',
  `actor` tinyint NOT NULL COMMENT '操作方：1=我 2=对手',
  `score_change` int NOT NULL DEFAULT 0 COMMENT '分数变化',
  `extra_data` json DEFAULT NULL COMMENT '额外数据',
  `is_undone` tinyint NOT NULL DEFAULT 0 COMMENT '是否已撤销',
  `client_action_id` varchar(64) DEFAULT NULL COMMENT '客户端操作ID',
  `base_revision` bigint unsigned NOT NULL DEFAULT 0 COMMENT '客户端基线版本号',
  `server_revision` bigint unsigned NOT NULL DEFAULT 0 COMMENT '服务端确认版本号',
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_match_id` (`match_id`),
  KEY `idx_is_undone` (`is_undone`),
  UNIQUE KEY `uk_match_client_action` (`match_id`, `client_action_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='操作日志表';

-- 特殊成绩表
CREATE TABLE IF NOT EXISTS `match_achievements` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '记录ID',
  `match_id` bigint unsigned NOT NULL COMMENT '对局ID',
  `achievement_type` varchar(20) NOT NULL COMMENT '成绩类型',
  `count` int NOT NULL DEFAULT 0 COMMENT '次数',
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_match_id` (`match_id`),
  UNIQUE KEY `uk_match_achievement` (`match_id`, `achievement_type`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='特殊成绩表';

-- 对手表
CREATE TABLE IF NOT EXISTS `opponents` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '对手ID',
  `user_id` bigint unsigned NOT NULL COMMENT '创建者用户ID',
  `name` varchar(50) NOT NULL COMMENT '对手昵称',
  `avatar` varchar(255) DEFAULT NULL COMMENT '对手头像',
  `linked_user_id` bigint unsigned DEFAULT NULL COMMENT '关联的注册用户ID',
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `deleted_at` datetime DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_user_id` (`user_id`),
  KEY `idx_linked_user_id` (`linked_user_id`),
  KEY `idx_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='对手表';

-- +goose Down
DROP TABLE IF EXISTS `opponents`;
DROP TABLE IF EXISTS `match_achievements`;
DROP TABLE IF EXISTS `match_actions`;
DROP TABLE IF EXISTS `match_rounds`;
DROP TABLE IF EXISTS `matches`;
