-- +goose Up
-- 为 matches 表增加完成归因字段

SELECT COUNT(*) INTO @completed_by_user_id_exists
FROM information_schema.COLUMNS
WHERE TABLE_SCHEMA = DATABASE()
  AND TABLE_NAME = 'matches'
  AND COLUMN_NAME = 'completed_by_user_id';

SET @ddl = IF(
  @completed_by_user_id_exists = 0,
  'ALTER TABLE `matches` ADD COLUMN `completed_by_user_id` bigint unsigned DEFAULT NULL COMMENT ''完成人用户ID'' AFTER `referee_joined_at`',
  'SELECT 1'
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SELECT COUNT(*) INTO @completion_source_exists
FROM information_schema.COLUMNS
WHERE TABLE_SCHEMA = DATABASE()
  AND TABLE_NAME = 'matches'
  AND COLUMN_NAME = 'completion_source';

SET @ddl = IF(
  @completion_source_exists = 0,
  'ALTER TABLE `matches` ADD COLUMN `completion_source` varchar(20) NOT NULL DEFAULT ''unknown'' COMMENT ''完成方式: referee/player_direct/player_confirmed/unknown'' AFTER `completed_by_user_id`',
  'SELECT 1'
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

-- +goose Down

SELECT COUNT(*) INTO @completion_source_exists
FROM information_schema.COLUMNS
WHERE TABLE_SCHEMA = DATABASE()
  AND TABLE_NAME = 'matches'
  AND COLUMN_NAME = 'completion_source';

SET @ddl = IF(
  @completion_source_exists > 0,
  'ALTER TABLE `matches` DROP COLUMN `completion_source`',
  'SELECT 1'
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SELECT COUNT(*) INTO @completed_by_user_id_exists
FROM information_schema.COLUMNS
WHERE TABLE_SCHEMA = DATABASE()
  AND TABLE_NAME = 'matches'
  AND COLUMN_NAME = 'completed_by_user_id';

SET @ddl = IF(
  @completed_by_user_id_exists > 0,
  'ALTER TABLE `matches` DROP COLUMN `completed_by_user_id`',
  'SELECT 1'
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;
