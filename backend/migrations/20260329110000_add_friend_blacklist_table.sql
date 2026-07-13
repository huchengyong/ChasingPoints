-- +goose Up
CREATE TABLE IF NOT EXISTS `friend_blacklists` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `user_id` BIGINT UNSIGNED NOT NULL COMMENT '发起拉黑的用户ID',
  `blocked_user_id` BIGINT UNSIGNED NOT NULL COMMENT '被拉黑的用户ID',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_user_blocked_user` (`user_id`, `blocked_user_id`),
  KEY `idx_blocked_user_id` (`blocked_user_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='好友黑名单表';

-- +goose Down
DROP TABLE IF EXISTS `friend_blacklists`;
