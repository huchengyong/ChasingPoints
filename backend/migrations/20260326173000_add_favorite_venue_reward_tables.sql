-- +goose Up
CREATE TABLE IF NOT EXISTS `favorite_venue_reward_configs` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `activity_key` VARCHAR(64) NOT NULL COMMENT '活动唯一标识',
  `enabled` TINYINT(1) NOT NULL DEFAULT 0 COMMENT '是否开启奖励',
  `popup_enabled` TINYINT(1) NOT NULL DEFAULT 0 COMMENT '是否开启我的页弹窗',
  `reward_days` INT NOT NULL DEFAULT 30 COMMENT '奖励会员天数',
  `new_user_window_days` INT NOT NULL DEFAULT 7 COMMENT '新用户有效期天数',
  `start_at` DATETIME DEFAULT NULL COMMENT '活动开始时间',
  `end_at` DATETIME DEFAULT NULL COMMENT '活动结束时间',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uniq_activity_key` (`activity_key`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='常玩球馆会员奖励配置表';

CREATE TABLE IF NOT EXISTS `favorite_venue_reward_records` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `activity_key` VARCHAR(64) NOT NULL COMMENT '活动唯一标识',
  `user_id` BIGINT UNSIGNED NOT NULL COMMENT '获奖用户ID',
  `venue_id` BIGINT UNSIGNED NOT NULL COMMENT '触发奖励的球馆ID',
  `reward_days` INT NOT NULL DEFAULT 30 COMMENT '发放会员天数',
  `member_expires_at_before` DATETIME DEFAULT NULL COMMENT '发放前会员到期时间',
  `member_expires_at_after` DATETIME NOT NULL COMMENT '发放后会员到期时间',
  `granted_at` DATETIME NOT NULL COMMENT '发放时间',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uniq_activity_user` (`activity_key`, `user_id`),
  UNIQUE KEY `uniq_activity_venue` (`activity_key`, `venue_id`),
  KEY `idx_user_id` (`user_id`),
  KEY `idx_venue_id` (`venue_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='常玩球馆会员奖励发放记录表';

INSERT INTO `favorite_venue_reward_configs` (
  `activity_key`,
  `enabled`,
  `popup_enabled`,
  `reward_days`,
  `new_user_window_days`
) VALUES (
  'favorite_venue_member_reward',
  1,
  1,
  30,
  7
)
ON DUPLICATE KEY UPDATE
  `reward_days` = VALUES(`reward_days`),
  `new_user_window_days` = VALUES(`new_user_window_days`);

-- +goose Down
DROP TABLE IF EXISTS `favorite_venue_reward_records`;
DROP TABLE IF EXISTS `favorite_venue_reward_configs`;
