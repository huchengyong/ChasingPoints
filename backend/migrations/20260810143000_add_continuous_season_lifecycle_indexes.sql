-- +goose Up
-- 连续赛季依赖唯一的开始日期。若存在 duplicate season start dates，使用带明确
-- 人工处理提示的唯一键名触发 MySQL DDL 失败，绝不静默改写历史赛季。

SET @duplicate_start_date_count = (
  SELECT COUNT(*)
  FROM (
    SELECT `start_date`
    FROM `seasons`
    GROUP BY `start_date`
    HAVING COUNT(*) > 1
  ) AS duplicate_season_start_dates
);
SET @idx_exists = (
  SELECT COUNT(*)
  FROM information_schema.STATISTICS
  WHERE TABLE_SCHEMA = DATABASE()
    AND TABLE_NAME = 'seasons'
    AND INDEX_NAME = 'uk_seasons_start_date'
);
SET @ddl = IF(
  @idx_exists = 1,
  'SELECT 1',
  IF(
    @duplicate_start_date_count > 0,
    'ALTER TABLE `seasons` ADD UNIQUE KEY `duplicate_season_start_dates_must_be_resolved` (`start_date`)',
    'ALTER TABLE `seasons` ADD UNIQUE KEY `uk_seasons_start_date` (`start_date`)'
  )
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @idx_exists = (
  SELECT COUNT(*)
  FROM information_schema.STATISTICS
  WHERE TABLE_SCHEMA = DATABASE()
    AND TABLE_NAME = 'seasons'
    AND INDEX_NAME = 'idx_seasons_window'
);
SET @ddl = IF(
  @idx_exists = 0,
  'ALTER TABLE `seasons` ADD KEY `idx_seasons_window` (`start_date`, `end_date`)',
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
    AND TABLE_NAME = 'seasons'
    AND INDEX_NAME = 'idx_seasons_window'
);
SET @ddl = IF(
  @idx_exists = 1,
  'ALTER TABLE `seasons` DROP INDEX `idx_seasons_window`',
  'SELECT 1'
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @idx_exists = (
  SELECT COUNT(*)
  FROM information_schema.STATISTICS
  WHERE TABLE_SCHEMA = DATABASE()
    AND TABLE_NAME = 'seasons'
    AND INDEX_NAME = 'uk_seasons_start_date'
);
SET @ddl = IF(
  @idx_exists = 1,
  'ALTER TABLE `seasons` DROP INDEX `uk_seasons_start_date`',
  'SELECT 1'
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;
