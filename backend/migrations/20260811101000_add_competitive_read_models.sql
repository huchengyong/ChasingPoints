-- +goose Up

CREATE TABLE IF NOT EXISTS `match_participant_results` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `match_id` BIGINT UNSIGNED NOT NULL,
  `user_id` BIGINT UNSIGNED NOT NULL,
  `opponent_user_id` BIGINT UNSIGNED NOT NULL DEFAULT 0,
  `opponent_name_key` VARCHAR(128) NOT NULL DEFAULT '',
  `opponent_name` VARCHAR(128) NOT NULL DEFAULT '',
  `opponent_avatar` VARCHAR(512) NOT NULL DEFAULT '',
  `game_type` TINYINT NOT NULL,
  `match_mode` VARCHAR(20) NOT NULL DEFAULT 'ranked',
  `result` TINYINT NOT NULL DEFAULT 3 COMMENT '当前用户视角：1胜2负3平',
  `my_score` INT NOT NULL DEFAULT 0,
  `opponent_score` INT NOT NULL DEFAULT 0,
  `completed_at` DATETIME NOT NULL,
  `duration_seconds` BIGINT NOT NULL DEFAULT 0,
  `match_high_score` INT NOT NULL DEFAULT 0,
  `best_break` INT NOT NULL DEFAULT 0,
  `opponent_rank_bucket` VARCHAR(32) NOT NULL DEFAULT '',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_match_participant_result` (`match_id`, `user_id`),
  KEY `idx_mpr_user_mode_game_completed` (`user_id`, `match_mode`, `game_type`, `completed_at`, `match_id`),
  KEY `idx_mpr_user_opponent_game_completed` (`user_id`, `opponent_user_id`, `game_type`, `completed_at`, `match_id`),
  KEY `idx_mpr_user_opponent_name_game_completed` (`user_id`, `opponent_name_key`, `game_type`, `completed_at`, `match_id`),
  KEY `idx_mpr_completed_at` (`completed_at`, `match_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='已完成比赛的参与者视角读投影';

CREATE TABLE IF NOT EXISTS `user_competitive_stats` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `user_id` BIGINT UNSIGNED NOT NULL,
  `game_type` TINYINT NOT NULL DEFAULT 0 COMMENT '0=总体，1-4=球种',
  `total_matches` INT NOT NULL DEFAULT 0,
  `wins` INT NOT NULL DEFAULT 0,
  `losses` INT NOT NULL DEFAULT 0,
  `draws` INT NOT NULL DEFAULT 0,
  `current_win_streak` INT NOT NULL DEFAULT 0,
  `max_win_streak` INT NOT NULL DEFAULT 0,
  `highest_score` INT NOT NULL DEFAULT 0,
  `highest_break` INT NOT NULL DEFAULT 0,
  `duration_count` INT NOT NULL DEFAULT 0,
  `duration_sum_seconds` BIGINT NOT NULL DEFAULT 0,
  `duration_min_seconds` BIGINT NOT NULL DEFAULT 0,
  `duration_max_seconds` BIGINT NOT NULL DEFAULT 0,
  `last_match_id` BIGINT UNSIGNED NOT NULL DEFAULT 0,
  `last_match_at` DATETIME DEFAULT NULL,
  `revision` BIGINT UNSIGNED NOT NULL DEFAULT 0,
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_user_competitive_stats_user_game` (`user_id`, `game_type`),
  KEY `idx_user_competitive_stats_last_match` (`user_id`, `last_match_at`, `last_match_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='用户竞技统计快照';

CREATE TABLE IF NOT EXISTS `user_opponent_stats` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `user_id` BIGINT UNSIGNED NOT NULL,
  `opponent_user_id` BIGINT UNSIGNED NOT NULL DEFAULT 0,
  `opponent_name_key` VARCHAR(128) NOT NULL DEFAULT '',
  `opponent_name` VARCHAR(128) NOT NULL DEFAULT '',
  `opponent_avatar` VARCHAR(512) NOT NULL DEFAULT '',
  `game_type` TINYINT NOT NULL DEFAULT 0 COMMENT '0=总体，1-4=球种',
  `total_matches` INT NOT NULL DEFAULT 0,
  `wins` INT NOT NULL DEFAULT 0,
  `losses` INT NOT NULL DEFAULT 0,
  `draws` INT NOT NULL DEFAULT 0,
  `last_match_id` BIGINT UNSIGNED NOT NULL DEFAULT 0,
  `last_match_at` DATETIME DEFAULT NULL,
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_user_opponent_stats_identity_game` (`user_id`, `opponent_user_id`, `opponent_name_key`, `game_type`),
  KEY `idx_user_opponent_stats_recent` (`user_id`, `game_type`, `last_match_at`, `last_match_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='用户对手交锋统计快照';

CREATE TABLE IF NOT EXISTS `user_opponent_strength_buckets` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `user_id` BIGINT UNSIGNED NOT NULL,
  `game_type` TINYINT NOT NULL,
  `rank_bucket` VARCHAR(32) NOT NULL,
  `matches` INT NOT NULL DEFAULT 0,
  `wins` INT NOT NULL DEFAULT 0,
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_user_opponent_strength_bucket` (`user_id`, `game_type`, `rank_bucket`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='按结算时对手段位归档的强度统计';

CREATE TABLE IF NOT EXISTS `competitive_read_model_rebuild_checkpoints` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `job_name` VARCHAR(64) NOT NULL,
  `cursor_match_id` BIGINT UNSIGNED NOT NULL DEFAULT 0,
  `cursor_completed_at` DATETIME NULL,
  `backfill_cursor_match_id` BIGINT UNSIGNED NOT NULL DEFAULT 0,
  `backfill_completed` TINYINT NOT NULL DEFAULT 0,
  `paused` TINYINT NOT NULL DEFAULT 0,
  `last_error` VARCHAR(512) NOT NULL DEFAULT '',
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_competitive_read_model_rebuild_job` (`job_name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='竞技读模型回建检查点';

-- +goose Down

DROP TABLE IF EXISTS `competitive_read_model_rebuild_checkpoints`;
DROP TABLE IF EXISTS `user_opponent_strength_buckets`;
DROP TABLE IF EXISTS `user_opponent_stats`;
DROP TABLE IF EXISTS `user_competitive_stats`;
DROP TABLE IF EXISTS `match_participant_results`;
