-- +goose Up

SET @column_exists = (
  SELECT COUNT(*)
  FROM information_schema.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE()
    AND TABLE_NAME = 'competitive_read_model_rebuild_checkpoints'
    AND COLUMN_NAME = 'cursor_completed_at'
);
SET @ddl = IF(
  @column_exists = 0,
  'ALTER TABLE `competitive_read_model_rebuild_checkpoints` ADD COLUMN `cursor_completed_at` DATETIME NULL AFTER `cursor_match_id`',
  'SELECT 1'
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @column_exists = (
  SELECT COUNT(*)
  FROM information_schema.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE()
    AND TABLE_NAME = 'competitive_read_model_rebuild_checkpoints'
    AND COLUMN_NAME = 'backfill_cursor_match_id'
);
SET @ddl = IF(
  @column_exists = 0,
  'ALTER TABLE `competitive_read_model_rebuild_checkpoints` ADD COLUMN `backfill_cursor_match_id` BIGINT UNSIGNED NOT NULL DEFAULT 0 AFTER `cursor_completed_at`',
  'SELECT 1'
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @column_exists = (
  SELECT COUNT(*)
  FROM information_schema.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE()
    AND TABLE_NAME = 'competitive_read_model_rebuild_checkpoints'
    AND COLUMN_NAME = 'backfill_completed'
);
SET @ddl = IF(
  @column_exists = 0,
  'ALTER TABLE `competitive_read_model_rebuild_checkpoints` ADD COLUMN `backfill_completed` TINYINT NOT NULL DEFAULT 0 AFTER `backfill_cursor_match_id`',
  'SELECT 1'
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @idx_exists = (
  SELECT COUNT(*)
  FROM information_schema.STATISTICS
  WHERE TABLE_SCHEMA = DATABASE()
    AND TABLE_NAME = 'matches'
    AND INDEX_NAME = 'idx_matches_status_deleted_completed_at_id'
);
SET @ddl = IF(
  @idx_exists = 0,
  'ALTER TABLE `matches` ADD KEY `idx_matches_status_deleted_completed_at_id` (`status`, `deleted_at`, `completed_at`, `id`)',
  'SELECT 1'
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

-- +goose Down

SET @idx_exists = (
  SELECT COUNT(*)
  FROM information_schema.STATISTICS
  WHERE TABLE_SCHEMA = DATABASE()
    AND TABLE_NAME = 'matches'
    AND INDEX_NAME = 'idx_matches_status_deleted_completed_at_id'
);
SET @ddl = IF(
  @idx_exists > 0,
  'ALTER TABLE `matches` DROP INDEX `idx_matches_status_deleted_completed_at_id`',
  'SELECT 1'
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @column_exists = (
  SELECT COUNT(*)
  FROM information_schema.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE()
    AND TABLE_NAME = 'competitive_read_model_rebuild_checkpoints'
    AND COLUMN_NAME = 'backfill_completed'
);
SET @ddl = IF(
  @column_exists = 1,
  'ALTER TABLE `competitive_read_model_rebuild_checkpoints` DROP COLUMN `backfill_completed`',
  'SELECT 1'
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @column_exists = (
  SELECT COUNT(*)
  FROM information_schema.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE()
    AND TABLE_NAME = 'competitive_read_model_rebuild_checkpoints'
    AND COLUMN_NAME = 'backfill_cursor_match_id'
);
SET @ddl = IF(
  @column_exists = 1,
  'ALTER TABLE `competitive_read_model_rebuild_checkpoints` DROP COLUMN `backfill_cursor_match_id`',
  'SELECT 1'
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @column_exists = (
  SELECT COUNT(*)
  FROM information_schema.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE()
    AND TABLE_NAME = 'competitive_read_model_rebuild_checkpoints'
    AND COLUMN_NAME = 'cursor_completed_at'
);
SET @ddl = IF(
  @column_exists = 1,
  'ALTER TABLE `competitive_read_model_rebuild_checkpoints` DROP COLUMN `cursor_completed_at`',
  'SELECT 1'
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;
