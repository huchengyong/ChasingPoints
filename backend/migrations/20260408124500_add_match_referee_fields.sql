-- +goose Up
-- 为 matches 表增加单场裁判字段

SELECT COUNT(*) INTO @referee_user_id_exists
FROM information_schema.COLUMNS
WHERE TABLE_SCHEMA = DATABASE()
  AND TABLE_NAME = 'matches'
  AND COLUMN_NAME = 'referee_user_id';

SET @ddl = IF(
  @referee_user_id_exists = 0,
  'ALTER TABLE `matches` ADD COLUMN `referee_user_id` bigint unsigned DEFAULT NULL COMMENT ''本场裁判用户ID'' AFTER `game_mode`',
  'SELECT 1'
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SELECT COUNT(*) INTO @referee_joined_at_exists
FROM information_schema.COLUMNS
WHERE TABLE_SCHEMA = DATABASE()
  AND TABLE_NAME = 'matches'
  AND COLUMN_NAME = 'referee_joined_at';

SET @ddl = IF(
  @referee_joined_at_exists = 0,
  'ALTER TABLE `matches` ADD COLUMN `referee_joined_at` datetime DEFAULT NULL COMMENT ''裁判加入时间'' AFTER `referee_user_id`',
  'SELECT 1'
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SELECT COUNT(*) INTO @referee_user_id_idx_exists
FROM information_schema.STATISTICS
WHERE TABLE_SCHEMA = DATABASE()
  AND TABLE_NAME = 'matches'
  AND INDEX_NAME = 'idx_referee_user_id';

SET @ddl = IF(
  @referee_user_id_idx_exists = 0,
  'ALTER TABLE `matches` ADD KEY `idx_referee_user_id` (`referee_user_id`)',
  'SELECT 1'
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

-- +goose Down

SELECT COUNT(*) INTO @referee_user_id_idx_exists
FROM information_schema.STATISTICS
WHERE TABLE_SCHEMA = DATABASE()
  AND TABLE_NAME = 'matches'
  AND INDEX_NAME = 'idx_referee_user_id';

SET @ddl = IF(
  @referee_user_id_idx_exists > 0,
  'ALTER TABLE `matches` DROP INDEX `idx_referee_user_id`',
  'SELECT 1'
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SELECT COUNT(*) INTO @referee_joined_at_exists
FROM information_schema.COLUMNS
WHERE TABLE_SCHEMA = DATABASE()
  AND TABLE_NAME = 'matches'
  AND COLUMN_NAME = 'referee_joined_at';

SET @ddl = IF(
  @referee_joined_at_exists > 0,
  'ALTER TABLE `matches` DROP COLUMN `referee_joined_at`',
  'SELECT 1'
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SELECT COUNT(*) INTO @referee_user_id_exists
FROM information_schema.COLUMNS
WHERE TABLE_SCHEMA = DATABASE()
  AND TABLE_NAME = 'matches'
  AND COLUMN_NAME = 'referee_user_id';

SET @ddl = IF(
  @referee_user_id_exists > 0,
  'ALTER TABLE `matches` DROP COLUMN `referee_user_id`',
  'SELECT 1'
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;
