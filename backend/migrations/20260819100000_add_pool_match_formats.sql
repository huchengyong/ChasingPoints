-- +goose Up
-- 中式八球/美式九球逐局赛制字段，存量和旧客户端对局保持 legacy

SELECT COUNT(*) INTO @match_format_exists
FROM information_schema.COLUMNS
WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'matches' AND COLUMN_NAME = 'match_format';
SET @ddl = IF(
  @match_format_exists = 0,
  'ALTER TABLE `matches` ADD COLUMN `match_format` varchar(20) NOT NULL DEFAULT ''legacy'' COMMENT ''逐局赛制：legacy/free/race_to'' AFTER `snooker_target_wins`',
  'SELECT 1'
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SELECT COUNT(*) INTO @target_wins_exists
FROM information_schema.COLUMNS
WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'matches' AND COLUMN_NAME = 'target_wins';
SET @ddl = IF(
  @target_wins_exists = 0,
  'ALTER TABLE `matches` ADD COLUMN `target_wins` smallint unsigned NOT NULL DEFAULT 0 COMMENT ''逐局赛制目标胜局，自由局数为0'' AFTER `match_format`',
  'SELECT 1'
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

-- +goose Down

SELECT COUNT(*) INTO @target_wins_exists
FROM information_schema.COLUMNS
WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'matches' AND COLUMN_NAME = 'target_wins';
SET @ddl = IF(@target_wins_exists > 0, 'ALTER TABLE `matches` DROP COLUMN `target_wins`', 'SELECT 1');
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SELECT COUNT(*) INTO @match_format_exists
FROM information_schema.COLUMNS
WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'matches' AND COLUMN_NAME = 'match_format';
SET @ddl = IF(@match_format_exists > 0, 'ALTER TABLE `matches` DROP COLUMN `match_format`', 'SELECT 1');
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;
