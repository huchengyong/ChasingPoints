-- +goose Up
CREATE TABLE IF NOT EXISTS `rank_change_logs` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `user_id` BIGINT UNSIGNED NOT NULL COMMENT '用户ID',
  `match_id` BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '关联对局ID，非对局变更时为0',
  `change_type` VARCHAR(32) NOT NULL DEFAULT 'match_result' COMMENT '变更类型: match_result/season_reset/manual_adjust',
  `game_type` TINYINT NOT NULL DEFAULT 3 COMMENT '球种 1=斯诺克 2=九球追分 3=中式八球 4=美式九球',
  `result` VARCHAR(16) NOT NULL DEFAULT '' COMMENT '结果: win/lose/draw/adjust',
  `base_score` INT NOT NULL DEFAULT 0 COMMENT '基础分变化',
  `achievement_score` INT NOT NULL DEFAULT 0 COMMENT '成就奖励分变化',
  `final_change` INT NOT NULL DEFAULT 0 COMMENT '最终分数变化',
  `before_score` INT NOT NULL DEFAULT 0 COMMENT '变更前分数',
  `after_score` INT NOT NULL DEFAULT 0 COMMENT '变更后分数',
  `before_level` TINYINT NOT NULL DEFAULT 1 COMMENT '变更前段位',
  `after_level` TINYINT NOT NULL DEFAULT 1 COMMENT '变更后段位',
  `operator_user_id` BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '操作人ID，系统自动结算时为0',
  `remark` VARCHAR(255) NOT NULL DEFAULT '' COMMENT '备注',
  `effective_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '业务生效时间',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uniq_user_match_type_game` (`user_id`, `match_id`, `change_type`, `game_type`),
  KEY `idx_user_game_effective_at` (`user_id`, `game_type`, `effective_at`),
  KEY `idx_match_id` (`match_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='段位分变更明细表';

-- +goose Down
DROP TABLE IF EXISTS `rank_change_logs`;
