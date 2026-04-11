-- +goose Up

CREATE TABLE IF NOT EXISTS `member_growth_profiles` (
  `id` BIGINT NOT NULL AUTO_INCREMENT,
  `user_id` BIGINT NOT NULL COMMENT '用户ID',
  `growth_points` INT NOT NULL DEFAULT 0 COMMENT '总成长值',
  `growth_level` INT NOT NULL DEFAULT 1 COMMENT '当前成长等级',
  `today_growth_count` INT NOT NULL DEFAULT 0 COMMENT '今日已计入成长次数',
  `today_growth_date` DATE DEFAULT NULL COMMENT '今日成长统计日期(UTC+8)',
  `last_growth_at` DATETIME DEFAULT NULL COMMENT '最近一次成长发放时间',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uniq_member_growth_profiles_user_id` (`user_id`),
  KEY `idx_member_growth_profiles_level` (`growth_level`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='会员成长快照表';

CREATE TABLE IF NOT EXISTS `member_growth_logs` (
  `id` BIGINT NOT NULL AUTO_INCREMENT,
  `user_id` BIGINT NOT NULL COMMENT '用户ID',
  `match_id` BIGINT NOT NULL COMMENT '对局ID',
  `growth_points` INT NOT NULL DEFAULT 1 COMMENT '本次发放成长值',
  `source` VARCHAR(32) NOT NULL COMMENT '成长来源',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uniq_member_growth_log` (`user_id`, `match_id`, `source`),
  KEY `idx_member_growth_logs_user_id` (`user_id`),
  KEY `idx_member_growth_logs_created_at` (`created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='会员成长发放日志表';

-- +goose Down
DROP TABLE IF EXISTS `member_growth_logs`;
DROP TABLE IF EXISTS `member_growth_profiles`;
