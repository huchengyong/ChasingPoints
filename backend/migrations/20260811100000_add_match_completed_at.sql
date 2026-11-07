-- +goose Up

SET @column_exists = (
  SELECT COUNT(*)
  FROM information_schema.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE()
    AND TABLE_NAME = 'matches'
    AND COLUMN_NAME = 'completed_at'
);
SET @ddl = IF(
  @column_exists = 0,
  'ALTER TABLE `matches` ADD COLUMN `completed_at` DATETIME NULL COMMENT ''完成业务时间，读模型排序真源'' AFTER `end_time`',
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
    AND INDEX_NAME = 'idx_matches_status_mode_game_completed'
);
SET @ddl = IF(
  @idx_exists = 0,
  'ALTER TABLE `matches` ADD KEY `idx_matches_status_mode_game_completed` (`status`, `match_mode`, `game_type`, `completed_at`, `id`)',
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
    AND INDEX_NAME = 'idx_matches_completed_at'
);
SET @ddl = IF(
  @idx_exists = 0,
  'ALTER TABLE `matches` ADD KEY `idx_matches_completed_at` (`completed_at`, `id`)',
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
    AND INDEX_NAME = 'idx_matches_completed_at'
);
SET @ddl = IF(
  @idx_exists > 0,
  'ALTER TABLE `matches` DROP INDEX `idx_matches_completed_at`',
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
    AND INDEX_NAME = 'idx_matches_status_mode_game_completed'
);
SET @ddl = IF(
  @idx_exists > 0,
  'ALTER TABLE `matches` DROP INDEX `idx_matches_status_mode_game_completed`',
  'SELECT 1'
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @column_exists = (
  SELECT COUNT(*)
  FROM information_schema.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE()
    AND TABLE_NAME = 'matches'
    AND COLUMN_NAME = 'completed_at'
);
SET @ddl = IF(
  @column_exists = 1,
  'ALTER TABLE `matches` DROP COLUMN `completed_at`',
  'SELECT 1'
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;
