-- +goose Up
CREATE TABLE IF NOT EXISTS `user_notification_preferences` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `user_id` BIGINT UNSIGNED NOT NULL COMMENT '用户ID',
  `match_result_enabled` TINYINT NOT NULL DEFAULT 1 COMMENT '是否接收对局结果通知',
  `friend_request_enabled` TINYINT NOT NULL DEFAULT 1 COMMENT '是否接收好友申请通知',
  `challenge_enabled` TINYINT NOT NULL DEFAULT 1 COMMENT '是否接收挑战通知',
  `tournament_enabled` TINYINT NOT NULL DEFAULT 1 COMMENT '是否接收赛事报名通知',
  `follow_enabled` TINYINT NOT NULL DEFAULT 1 COMMENT '是否接收关注通知',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_user_notification_preferences_user_id` (`user_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='用户通知偏好表';

-- +goose Down
DROP TABLE IF EXISTS `user_notification_preferences`;
