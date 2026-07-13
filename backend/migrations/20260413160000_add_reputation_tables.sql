-- +goose Up

CREATE TABLE IF NOT EXISTS `reputation_configs` (
  `id` BIGINT NOT NULL AUTO_INCREMENT,
  `config_key` VARCHAR(64) NOT NULL COMMENT '配置唯一键',
  `base_rules_json` JSON NOT NULL COMMENT '基础信誉规则',
  `recovery_rules_json` JSON NOT NULL COMMENT '信誉恢复规则',
  `detection_rules_json` JSON NOT NULL COMMENT '异常比赛判定规则',
  `updated_by` BIGINT NOT NULL DEFAULT 0 COMMENT '最后更新管理员ID',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uniq_reputation_configs_key` (`config_key`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='信誉制度配置表';

CREATE TABLE IF NOT EXISTS `user_reputation_profiles` (
  `id` BIGINT NOT NULL AUTO_INCREMENT,
  `user_id` BIGINT NOT NULL COMMENT '用户ID',
  `reputation_score` INT NOT NULL COMMENT '当前信誉分',
  `last_recovered_at` DATETIME DEFAULT NULL COMMENT '最近一次恢复计算时间',
  `last_penalized_at` DATETIME DEFAULT NULL COMMENT '最近一次扣分时间',
  `ban_until` DATETIME DEFAULT NULL COMMENT '禁赛截止时间',
  `total_penalty_count` INT NOT NULL DEFAULT 0 COMMENT '累计处罚次数',
  `total_abnormal_match_count` INT NOT NULL DEFAULT 0 COMMENT '累计异常比赛数',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uniq_user_reputation_profiles_user_id` (`user_id`),
  KEY `idx_user_reputation_profiles_ban_until` (`ban_until`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='用户信誉快照表';

CREATE TABLE IF NOT EXISTS `user_reputation_logs` (
  `id` BIGINT NOT NULL AUTO_INCREMENT,
  `user_id` BIGINT NOT NULL COMMENT '用户ID',
  `match_id` BIGINT DEFAULT NULL COMMENT '对局ID，非对局变更为NULL',
  `change_type` VARCHAR(32) NOT NULL COMMENT '变更类型: penalty/recovery/manual_adjust',
  `change_score` INT NOT NULL COMMENT '本次变更分值',
  `before_score` INT NOT NULL COMMENT '变更前信誉分',
  `after_score` INT NOT NULL COMMENT '变更后信誉分',
  `reason_code` VARCHAR(64) NOT NULL COMMENT '变更原因编码',
  `reason_detail` TEXT COMMENT '变更原因详情',
  `operator_admin_id` BIGINT NOT NULL DEFAULT 0 COMMENT '操作管理员ID，系统自动为0',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uniq_user_match_change_type_reason_code` (`user_id`, `match_id`, `change_type`, `reason_code`),
  KEY `idx_user_reputation_logs_user_id` (`user_id`),
  KEY `idx_user_reputation_logs_created_at` (`created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='用户信誉变更日志表';

-- +goose Down
DROP TABLE IF EXISTS `user_reputation_logs`;
DROP TABLE IF EXISTS `user_reputation_profiles`;
DROP TABLE IF EXISTS `reputation_configs`;
