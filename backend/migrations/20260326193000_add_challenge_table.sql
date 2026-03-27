-- +goose Up
CREATE TABLE IF NOT EXISTS `challenges` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `from_user_id` BIGINT UNSIGNED NOT NULL COMMENT '发起挑战的用户ID',
  `to_user_id` BIGINT UNSIGNED NOT NULL COMMENT '接收挑战的用户ID',
  `game_type` TINYINT NOT NULL COMMENT '球种类型',
  `message` VARCHAR(200) NOT NULL DEFAULT '' COMMENT '挑战留言',
  `status` TINYINT NOT NULL DEFAULT 0 COMMENT '状态 0=待处理 1=已接受 2=已拒绝 3=已过期',
  `match_id` BIGINT UNSIGNED DEFAULT NULL COMMENT '关联对局ID',
  `expires_at` DATETIME NOT NULL COMMENT '过期时间',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`),
  KEY `idx_from_user_id` (`from_user_id`),
  KEY `idx_to_user_id` (`to_user_id`),
  KEY `idx_match_id` (`match_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='用户挑战表';

-- +goose Down
DROP TABLE IF EXISTS `challenges`;
