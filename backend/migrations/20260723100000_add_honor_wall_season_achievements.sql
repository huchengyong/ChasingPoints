-- +goose Up
-- 荣誉墙、赛季挑战、换季结算与本场奖励归因

SET @col_exists = (
  SELECT COUNT(*)
  FROM information_schema.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE()
    AND TABLE_NAME = 'achievement_progress_events'
    AND COLUMN_NAME = 'occurred_at'
);
SET @ddl = IF(
  @col_exists = 0,
  'ALTER TABLE `achievement_progress_events` ADD COLUMN `occurred_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT ''业务发生时间'' AFTER `metric_value`',
  'SELECT 1'
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @dml = IF(
  @col_exists = 0,
  'UPDATE `achievement_progress_events` SET `occurred_at` = `created_at` WHERE `created_at` IS NOT NULL',
  'SELECT 1'
);
PREPARE stmt FROM @dml;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @idx_exists = (
  SELECT COUNT(*)
  FROM information_schema.STATISTICS
  WHERE TABLE_SCHEMA = DATABASE()
    AND TABLE_NAME = 'achievement_progress_events'
    AND INDEX_NAME = 'idx_achievement_progress_events_season_scope'
);
SET @ddl = IF(
  @idx_exists = 0,
  'ALTER TABLE `achievement_progress_events` ADD KEY `idx_achievement_progress_events_season_scope` (`user_id`, `game_type`, `metric_key`, `occurred_at`)',
  'SELECT 1'
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @col_exists = (
  SELECT COUNT(*)
  FROM information_schema.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE()
    AND TABLE_NAME = 'user_achievements'
    AND COLUMN_NAME = 'unlocked_source_type'
);
SET @ddl = IF(
  @col_exists = 0,
  'ALTER TABLE `user_achievements` ADD COLUMN `unlocked_source_type` VARCHAR(32) NOT NULL DEFAULT '''' COMMENT ''首次解锁来源类型'' AFTER `unlocked_at`',
  'SELECT 1'
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @col_exists = (
  SELECT COUNT(*)
  FROM information_schema.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE()
    AND TABLE_NAME = 'user_achievements'
    AND COLUMN_NAME = 'unlocked_source_id'
);
SET @ddl = IF(
  @col_exists = 0,
  'ALTER TABLE `user_achievements` ADD COLUMN `unlocked_source_id` BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT ''首次解锁来源ID'' AFTER `unlocked_source_type`',
  'SELECT 1'
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @idx_exists = (
  SELECT COUNT(*)
  FROM information_schema.STATISTICS
  WHERE TABLE_SCHEMA = DATABASE()
    AND TABLE_NAME = 'user_achievements'
    AND INDEX_NAME = 'idx_user_achievement_unlock_source'
);
SET @ddl = IF(
  @idx_exists = 0,
  'ALTER TABLE `user_achievements` ADD KEY `idx_user_achievement_unlock_source` (`unlocked_source_type`, `unlocked_source_id`, `user_id`)',
  'SELECT 1'
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @col_exists = (
  SELECT COUNT(*)
  FROM information_schema.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE()
    AND TABLE_NAME = 'matches'
    AND COLUMN_NAME = 'achievement_synced_at'
);
SET @ddl = IF(
  @col_exists = 0,
  'ALTER TABLE `matches` ADD COLUMN `achievement_synced_at` DATETIME DEFAULT NULL COMMENT ''竞技成就同步完成时间'' AFTER `end_time`',
  'SELECT 1'
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @col_exists = (
  SELECT COUNT(*)
  FROM information_schema.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE()
    AND TABLE_NAME = 'notifications'
    AND COLUMN_NAME = 'dedupe_key'
);
SET @ddl = IF(
  @col_exists = 0,
  'ALTER TABLE `notifications` ADD COLUMN `dedupe_key` VARCHAR(128) DEFAULT NULL COMMENT ''通知幂等键'' AFTER `type`',
  'SELECT 1'
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @idx_exists = (
  SELECT COUNT(*)
  FROM information_schema.STATISTICS
  WHERE TABLE_SCHEMA = DATABASE()
    AND TABLE_NAME = 'notifications'
    AND INDEX_NAME = 'uk_notifications_user_type_dedupe'
);
SET @ddl = IF(
  @idx_exists = 0,
  'ALTER TABLE `notifications` ADD UNIQUE KEY `uk_notifications_user_type_dedupe` (`user_id`, `type`, `dedupe_key`)',
  'SELECT 1'
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

CREATE TABLE IF NOT EXISTS `season_challenge_snapshots` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `season_id` BIGINT UNSIGNED NOT NULL COMMENT '赛季ID',
  `user_id` BIGINT UNSIGNED NOT NULL COMMENT '用户ID',
  `game_type` TINYINT NOT NULL COMMENT '球种',
  `challenge_key` VARCHAR(64) NOT NULL COMMENT '挑战键',
  `challenge_name` VARCHAR(128) NOT NULL COMMENT '挑战名称快照',
  `threshold` INT NOT NULL DEFAULT 0 COMMENT '目标值快照',
  `progress` INT NOT NULL DEFAULT 0 COMMENT '最终进度',
  `completed` TINYINT NOT NULL DEFAULT 0 COMMENT '是否完成',
  `archived_at` DATETIME NOT NULL COMMENT '归档时间',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_season_challenge_snapshot` (`season_id`, `user_id`, `game_type`, `challenge_key`),
  KEY `idx_season_challenge_snapshots_user_season` (`user_id`, `season_id`, `game_type`),
  KEY `idx_season_challenge_snapshots_season` (`season_id`, `game_type`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='赛季挑战归档快照';

CREATE TABLE IF NOT EXISTS `season_settlements` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `season_id` BIGINT UNSIGNED NOT NULL COMMENT '待结算赛季ID',
  `next_season_id` BIGINT UNSIGNED DEFAULT NULL COMMENT '结算时启用的下一赛季ID',
  `status` VARCHAR(16) NOT NULL DEFAULT 'running' COMMENT 'running/completed/failed',
  `attempts` INT NOT NULL DEFAULT 0 COMMENT '执行次数',
  `started_at` DATETIME DEFAULT NULL,
  `completed_at` DATETIME DEFAULT NULL,
  `last_error` VARCHAR(512) NOT NULL DEFAULT '',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_season_settlements_season` (`season_id`),
  KEY `idx_season_settlements_status` (`status`, `updated_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='赛季结算状态';

-- +goose Down

DROP TABLE IF EXISTS `season_settlements`;
DROP TABLE IF EXISTS `season_challenge_snapshots`;

SET @idx_exists = (
  SELECT COUNT(*)
  FROM information_schema.STATISTICS
  WHERE TABLE_SCHEMA = DATABASE()
    AND TABLE_NAME = 'notifications'
    AND INDEX_NAME = 'uk_notifications_user_type_dedupe'
);
SET @ddl = IF(
  @idx_exists = 1,
  'ALTER TABLE `notifications` DROP INDEX `uk_notifications_user_type_dedupe`',
  'SELECT 1'
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @col_exists = (
  SELECT COUNT(*)
  FROM information_schema.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE()
    AND TABLE_NAME = 'notifications'
    AND COLUMN_NAME = 'dedupe_key'
);
SET @ddl = IF(
  @col_exists = 1,
  'ALTER TABLE `notifications` DROP COLUMN `dedupe_key`',
  'SELECT 1'
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @col_exists = (
  SELECT COUNT(*)
  FROM information_schema.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE()
    AND TABLE_NAME = 'matches'
    AND COLUMN_NAME = 'achievement_synced_at'
);
SET @ddl = IF(
  @col_exists = 1,
  'ALTER TABLE `matches` DROP COLUMN `achievement_synced_at`',
  'SELECT 1'
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @idx_exists = (
  SELECT COUNT(*)
  FROM information_schema.STATISTICS
  WHERE TABLE_SCHEMA = DATABASE()
    AND TABLE_NAME = 'user_achievements'
    AND INDEX_NAME = 'idx_user_achievement_unlock_source'
);
SET @ddl = IF(
  @idx_exists = 1,
  'ALTER TABLE `user_achievements` DROP INDEX `idx_user_achievement_unlock_source`',
  'SELECT 1'
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @col_exists = (
  SELECT COUNT(*)
  FROM information_schema.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE()
    AND TABLE_NAME = 'user_achievements'
    AND COLUMN_NAME = 'unlocked_source_id'
);
SET @ddl = IF(
  @col_exists = 1,
  'ALTER TABLE `user_achievements` DROP COLUMN `unlocked_source_id`',
  'SELECT 1'
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @col_exists = (
  SELECT COUNT(*)
  FROM information_schema.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE()
    AND TABLE_NAME = 'user_achievements'
    AND COLUMN_NAME = 'unlocked_source_type'
);
SET @ddl = IF(
  @col_exists = 1,
  'ALTER TABLE `user_achievements` DROP COLUMN `unlocked_source_type`',
  'SELECT 1'
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @idx_exists = (
  SELECT COUNT(*)
  FROM information_schema.STATISTICS
  WHERE TABLE_SCHEMA = DATABASE()
    AND TABLE_NAME = 'achievement_progress_events'
    AND INDEX_NAME = 'idx_achievement_progress_events_season_scope'
);
SET @ddl = IF(
  @idx_exists = 1,
  'ALTER TABLE `achievement_progress_events` DROP INDEX `idx_achievement_progress_events_season_scope`',
  'SELECT 1'
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @col_exists = (
  SELECT COUNT(*)
  FROM information_schema.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE()
    AND TABLE_NAME = 'achievement_progress_events'
    AND COLUMN_NAME = 'occurred_at'
);
SET @ddl = IF(
  @col_exists = 1,
  'ALTER TABLE `achievement_progress_events` DROP COLUMN `occurred_at`',
  'SELECT 1'
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;
