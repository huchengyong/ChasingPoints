-- +goose Up
-- 补齐成就-称号闭环所需表结构

SET @col_exists = (
  SELECT COUNT(*)
  FROM information_schema.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'match_achievements' AND COLUMN_NAME = 'actor'
);
SET @ddl = IF(
  @col_exists = 0,
  'ALTER TABLE `match_achievements` ADD COLUMN `actor` TINYINT NOT NULL DEFAULT 1 COMMENT ''触发方 1=本人 2=对手'' AFTER `match_id`',
  'SELECT 1'
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @idx_exists = (
  SELECT COUNT(*)
  FROM information_schema.STATISTICS
  WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'match_achievements' AND INDEX_NAME = 'uk_match_achievement'
);
SET @ddl = IF(
  @idx_exists = 1,
  'ALTER TABLE `match_achievements` DROP INDEX `uk_match_achievement`',
  'SELECT 1'
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @idx_exists = (
  SELECT COUNT(*)
  FROM information_schema.STATISTICS
  WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'match_achievements' AND INDEX_NAME = 'uk_match_actor_achievement'
);
SET @ddl = IF(
  @idx_exists = 0,
  'ALTER TABLE `match_achievements` ADD UNIQUE KEY `uk_match_actor_achievement` (`match_id`, `actor`, `achievement_type`)',
  'SELECT 1'
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @col_exists = (
  SELECT COUNT(*)
  FROM information_schema.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'achievements' AND COLUMN_NAME = 'game_type'
);
SET @ddl = IF(
  @col_exists = 0,
  'ALTER TABLE `achievements` ADD COLUMN `game_type` TINYINT NOT NULL DEFAULT 0 COMMENT ''适用球种 0=通用 1=斯诺克 2=九球追分 3=中式八球'' AFTER `category`',
  'SELECT 1'
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @col_exists = (
  SELECT COUNT(*)
  FROM information_schema.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'achievements' AND COLUMN_NAME = 'metric_key'
);
SET @ddl = IF(
  @col_exists = 0,
  'ALTER TABLE `achievements` ADD COLUMN `metric_key` VARCHAR(64) NOT NULL DEFAULT '''' COMMENT ''进度指标键'' AFTER `game_type`',
  'SELECT 1'
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @col_exists = (
  SELECT COUNT(*)
  FROM information_schema.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'achievements' AND COLUMN_NAME = 'progress_mode'
);
SET @ddl = IF(
  @col_exists = 0,
  'ALTER TABLE `achievements` ADD COLUMN `progress_mode` VARCHAR(32) NOT NULL DEFAULT ''sum'' COMMENT ''进度模式 sum/max'' AFTER `metric_key`',
  'SELECT 1'
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @col_exists = (
  SELECT COUNT(*)
  FROM information_schema.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'achievements' AND COLUMN_NAME = 'reward_title_key'
);
SET @ddl = IF(
  @col_exists = 0,
  'ALTER TABLE `achievements` ADD COLUMN `reward_title_key` VARCHAR(64) NOT NULL DEFAULT '''' COMMENT ''奖励称号key'' AFTER `threshold`',
  'SELECT 1'
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @col_exists = (
  SELECT COUNT(*)
  FROM information_schema.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'achievements' AND COLUMN_NAME = 'reward_title_name'
);
SET @ddl = IF(
  @col_exists = 0,
  'ALTER TABLE `achievements` ADD COLUMN `reward_title_name` VARCHAR(64) NOT NULL DEFAULT '''' COMMENT ''奖励称号名称'' AFTER `reward_title_key`',
  'SELECT 1'
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @col_exists = (
  SELECT COUNT(*)
  FROM information_schema.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'achievements' AND COLUMN_NAME = 'sort'
);
SET @ddl = IF(
  @col_exists = 0,
  'ALTER TABLE `achievements` ADD COLUMN `sort` INT NOT NULL DEFAULT 0 COMMENT ''排序值'' AFTER `reward_title_name`',
  'SELECT 1'
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @col_exists = (
  SELECT COUNT(*)
  FROM information_schema.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'achievements' AND COLUMN_NAME = 'status'
);
SET @ddl = IF(
  @col_exists = 0,
  'ALTER TABLE `achievements` ADD COLUMN `status` TINYINT NOT NULL DEFAULT 1 COMMENT ''状态 1=启用 0=停用'' AFTER `sort`',
  'SELECT 1'
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @col_exists = (
  SELECT COUNT(*)
  FROM information_schema.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'achievements' AND COLUMN_NAME = 'updated_at'
);
SET @ddl = IF(
  @col_exists = 0,
  'ALTER TABLE `achievements` ADD COLUMN `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP AFTER `created_at`',
  'SELECT 1'
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @col_exists = (
  SELECT COUNT(*)
  FROM information_schema.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'user_achievements' AND COLUMN_NAME = 'reward_granted'
);
SET @ddl = IF(
  @col_exists = 0,
  'ALTER TABLE `user_achievements` ADD COLUMN `reward_granted` TINYINT NOT NULL DEFAULT 0 COMMENT ''奖励是否已发放'' AFTER `unlocked_at`',
  'SELECT 1'
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @col_exists = (
  SELECT COUNT(*)
  FROM information_schema.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'user_achievements' AND COLUMN_NAME = 'reward_granted_at'
);
SET @ddl = IF(
  @col_exists = 0,
  'ALTER TABLE `user_achievements` ADD COLUMN `reward_granted_at` DATETIME DEFAULT NULL COMMENT ''奖励发放时间'' AFTER `reward_granted`',
  'SELECT 1'
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @col_exists = (
  SELECT COUNT(*)
  FROM information_schema.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'user_achievements' AND COLUMN_NAME = 'updated_at'
);
SET @ddl = IF(
  @col_exists = 0,
  'ALTER TABLE `user_achievements` ADD COLUMN `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP AFTER `created_at`',
  'SELECT 1'
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @col_exists = (
  SELECT COUNT(*)
  FROM information_schema.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'user_titles' AND COLUMN_NAME = 'title_key'
);
SET @ddl = IF(
  @col_exists = 0,
  'ALTER TABLE `user_titles` ADD COLUMN `title_key` VARCHAR(64) NOT NULL DEFAULT '''' COMMENT ''称号唯一键'' AFTER `user_id`',
  'SELECT 1'
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @col_exists = (
  SELECT COUNT(*)
  FROM information_schema.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'user_titles' AND COLUMN_NAME = 'source_type'
);
SET @ddl = IF(
  @col_exists = 0,
  'ALTER TABLE `user_titles` ADD COLUMN `source_type` VARCHAR(32) NOT NULL DEFAULT '''' COMMENT ''来源类型 achievement/season/tournament/manual'' AFTER `source`',
  'SELECT 1'
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @col_exists = (
  SELECT COUNT(*)
  FROM information_schema.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'user_titles' AND COLUMN_NAME = 'source_ref_id'
);
SET @ddl = IF(
  @col_exists = 0,
  'ALTER TABLE `user_titles` ADD COLUMN `source_ref_id` BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT ''来源引用ID'' AFTER `source_type`',
  'SELECT 1'
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @needs_fix = (
  SELECT COUNT(*)
  FROM information_schema.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE()
    AND TABLE_NAME = 'user_titles'
    AND COLUMN_NAME = 'source_ref_id'
    AND (
      IS_NULLABLE = 'YES'
      OR COLUMN_DEFAULT IS NULL
      OR DATA_TYPE <> 'bigint'
      OR COLUMN_TYPE NOT LIKE '%unsigned%'
    )
);
SET @ddl = IF(
  @needs_fix = 1,
  'ALTER TABLE `user_titles` MODIFY COLUMN `source_ref_id` BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT ''来源引用ID'' AFTER `source_type`',
  'SELECT 1'
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @col_exists = (
  SELECT COUNT(*)
  FROM information_schema.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'user_titles' AND COLUMN_NAME = 'source_ref_name'
);
SET @ddl = IF(
  @col_exists = 0,
  'ALTER TABLE `user_titles` ADD COLUMN `source_ref_name` VARCHAR(128) NOT NULL DEFAULT '''' COMMENT ''来源引用名称'' AFTER `source_ref_id`',
  'SELECT 1'
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @col_exists = (
  SELECT COUNT(*)
  FROM information_schema.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'user_titles' AND COLUMN_NAME = 'granted_by_achievement_id'
);
SET @ddl = IF(
  @col_exists = 0,
  'ALTER TABLE `user_titles` ADD COLUMN `granted_by_achievement_id` BIGINT UNSIGNED DEFAULT NULL COMMENT ''授予该称号的成就ID'' AFTER `source_ref_name`',
  'SELECT 1'
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @col_exists = (
  SELECT COUNT(*)
  FROM information_schema.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'user_titles' AND COLUMN_NAME = 'equipped_at'
);
SET @ddl = IF(
  @col_exists = 0,
  'ALTER TABLE `user_titles` ADD COLUMN `equipped_at` DATETIME DEFAULT NULL COMMENT ''装备时间'' AFTER `equipped`',
  'SELECT 1'
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @col_exists = (
  SELECT COUNT(*)
  FROM information_schema.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'user_titles' AND COLUMN_NAME = 'granted_at'
);
SET @ddl = IF(
  @col_exists = 0,
  'ALTER TABLE `user_titles` ADD COLUMN `granted_at` DATETIME DEFAULT NULL COMMENT ''授予时间'' AFTER `equipped_at`',
  'SELECT 1'
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @idx_exists = (
  SELECT COUNT(*)
  FROM information_schema.STATISTICS
  WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'user_titles' AND INDEX_NAME = 'uk_user_title_source'
);
SET @ddl = IF(
  @idx_exists = 0,
  'ALTER TABLE `user_titles` ADD UNIQUE KEY `uk_user_title_source` (`user_id`, `title_key`, `source_type`, `source_ref_id`)',
  'SELECT 1'
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

CREATE TABLE IF NOT EXISTS `achievement_progress_events` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `user_id` BIGINT UNSIGNED NOT NULL COMMENT '用户ID',
  `source_type` VARCHAR(32) NOT NULL DEFAULT '' COMMENT '来源类型 match/achievement/manual',
  `source_id` BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '来源ID',
  `game_type` TINYINT NOT NULL DEFAULT 0 COMMENT '球种 0=通用 1=斯诺克 2=九球追分 3=中式八球',
  `metric_key` VARCHAR(64) NOT NULL DEFAULT '' COMMENT '进度指标键',
  `metric_value` INT NOT NULL DEFAULT 0 COMMENT '指标值',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_user_source_metric` (`user_id`, `source_type`, `source_id`, `metric_key`),
  KEY `idx_achievement_progress_events_user_id` (`user_id`),
  KEY `idx_achievement_progress_events_game_type` (`game_type`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='成就进度事件表';

-- +goose Down

DROP TABLE IF EXISTS `achievement_progress_events`;

SET @idx_exists = (
  SELECT COUNT(*)
  FROM information_schema.STATISTICS
  WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'user_titles' AND INDEX_NAME = 'uk_user_title_source'
);
SET @ddl = IF(
  @idx_exists = 1,
  'ALTER TABLE `user_titles` DROP INDEX `uk_user_title_source`',
  'SELECT 1'
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @col_exists = (
  SELECT COUNT(*)
  FROM information_schema.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'user_titles' AND COLUMN_NAME = 'granted_at'
);
SET @ddl = IF(
  @col_exists = 1,
  'ALTER TABLE `user_titles` DROP COLUMN `granted_at`',
  'SELECT 1'
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @col_exists = (
  SELECT COUNT(*)
  FROM information_schema.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'user_titles' AND COLUMN_NAME = 'equipped_at'
);
SET @ddl = IF(
  @col_exists = 1,
  'ALTER TABLE `user_titles` DROP COLUMN `equipped_at`',
  'SELECT 1'
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @col_exists = (
  SELECT COUNT(*)
  FROM information_schema.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'user_titles' AND COLUMN_NAME = 'granted_by_achievement_id'
);
SET @ddl = IF(
  @col_exists = 1,
  'ALTER TABLE `user_titles` DROP COLUMN `granted_by_achievement_id`',
  'SELECT 1'
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @col_exists = (
  SELECT COUNT(*)
  FROM information_schema.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'user_titles' AND COLUMN_NAME = 'source_ref_name'
);
SET @ddl = IF(
  @col_exists = 1,
  'ALTER TABLE `user_titles` DROP COLUMN `source_ref_name`',
  'SELECT 1'
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @col_exists = (
  SELECT COUNT(*)
  FROM information_schema.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'user_titles' AND COLUMN_NAME = 'source_ref_id'
);
SET @ddl = IF(
  @col_exists = 1,
  'ALTER TABLE `user_titles` DROP COLUMN `source_ref_id`',
  'SELECT 1'
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @col_exists = (
  SELECT COUNT(*)
  FROM information_schema.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'user_titles' AND COLUMN_NAME = 'source_type'
);
SET @ddl = IF(
  @col_exists = 1,
  'ALTER TABLE `user_titles` DROP COLUMN `source_type`',
  'SELECT 1'
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @col_exists = (
  SELECT COUNT(*)
  FROM information_schema.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'user_titles' AND COLUMN_NAME = 'title_key'
);
SET @ddl = IF(
  @col_exists = 1,
  'ALTER TABLE `user_titles` DROP COLUMN `title_key`',
  'SELECT 1'
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @col_exists = (
  SELECT COUNT(*)
  FROM information_schema.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'user_achievements' AND COLUMN_NAME = 'updated_at'
);
SET @ddl = IF(
  @col_exists = 1,
  'ALTER TABLE `user_achievements` DROP COLUMN `updated_at`',
  'SELECT 1'
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @col_exists = (
  SELECT COUNT(*)
  FROM information_schema.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'user_achievements' AND COLUMN_NAME = 'reward_granted_at'
);
SET @ddl = IF(
  @col_exists = 1,
  'ALTER TABLE `user_achievements` DROP COLUMN `reward_granted_at`',
  'SELECT 1'
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @col_exists = (
  SELECT COUNT(*)
  FROM information_schema.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'user_achievements' AND COLUMN_NAME = 'reward_granted'
);
SET @ddl = IF(
  @col_exists = 1,
  'ALTER TABLE `user_achievements` DROP COLUMN `reward_granted`',
  'SELECT 1'
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @col_exists = (
  SELECT COUNT(*)
  FROM information_schema.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'achievements' AND COLUMN_NAME = 'updated_at'
);
SET @ddl = IF(
  @col_exists = 1,
  'ALTER TABLE `achievements` DROP COLUMN `updated_at`',
  'SELECT 1'
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @col_exists = (
  SELECT COUNT(*)
  FROM information_schema.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'achievements' AND COLUMN_NAME = 'status'
);
SET @ddl = IF(
  @col_exists = 1,
  'ALTER TABLE `achievements` DROP COLUMN `status`',
  'SELECT 1'
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @col_exists = (
  SELECT COUNT(*)
  FROM information_schema.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'achievements' AND COLUMN_NAME = 'sort'
);
SET @ddl = IF(
  @col_exists = 1,
  'ALTER TABLE `achievements` DROP COLUMN `sort`',
  'SELECT 1'
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @col_exists = (
  SELECT COUNT(*)
  FROM information_schema.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'achievements' AND COLUMN_NAME = 'reward_title_name'
);
SET @ddl = IF(
  @col_exists = 1,
  'ALTER TABLE `achievements` DROP COLUMN `reward_title_name`',
  'SELECT 1'
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @col_exists = (
  SELECT COUNT(*)
  FROM information_schema.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'achievements' AND COLUMN_NAME = 'reward_title_key'
);
SET @ddl = IF(
  @col_exists = 1,
  'ALTER TABLE `achievements` DROP COLUMN `reward_title_key`',
  'SELECT 1'
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @col_exists = (
  SELECT COUNT(*)
  FROM information_schema.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'achievements' AND COLUMN_NAME = 'progress_mode'
);
SET @ddl = IF(
  @col_exists = 1,
  'ALTER TABLE `achievements` DROP COLUMN `progress_mode`',
  'SELECT 1'
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @col_exists = (
  SELECT COUNT(*)
  FROM information_schema.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'achievements' AND COLUMN_NAME = 'metric_key'
);
SET @ddl = IF(
  @col_exists = 1,
  'ALTER TABLE `achievements` DROP COLUMN `metric_key`',
  'SELECT 1'
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @col_exists = (
  SELECT COUNT(*)
  FROM information_schema.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'achievements' AND COLUMN_NAME = 'game_type'
);
SET @ddl = IF(
  @col_exists = 1,
  'ALTER TABLE `achievements` DROP COLUMN `game_type`',
  'SELECT 1'
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @idx_exists = (
  SELECT COUNT(*)
  FROM information_schema.STATISTICS
  WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'match_achievements' AND INDEX_NAME = 'uk_match_actor_achievement'
);
SET @ddl = IF(
  @idx_exists = 1,
  'ALTER TABLE `match_achievements` DROP INDEX `uk_match_actor_achievement`',
  'SELECT 1'
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @idx_exists = (
  SELECT COUNT(*)
  FROM information_schema.STATISTICS
  WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'match_achievements' AND INDEX_NAME = 'uk_match_achievement'
);
SET @ddl = IF(
  @idx_exists = 0,
  'ALTER TABLE `match_achievements` ADD UNIQUE KEY `uk_match_achievement` (`match_id`, `achievement_type`)',
  'SELECT 1'
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @col_exists = (
  SELECT COUNT(*)
  FROM information_schema.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'match_achievements' AND COLUMN_NAME = 'actor'
);
SET @ddl = IF(
  @col_exists = 1,
  'ALTER TABLE `match_achievements` DROP COLUMN `actor`',
  'SELECT 1'
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;
