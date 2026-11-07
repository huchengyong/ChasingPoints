-- +goose Up
-- 新斯诺克对局支持自由局数与抢 N 局制，同时保留 best_of_frames 兼容语义

SELECT COUNT(*) INTO @snooker_format_exists
FROM information_schema.COLUMNS
WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'matches' AND COLUMN_NAME = 'snooker_format';
SET @ddl = IF(
  @snooker_format_exists = 0,
  'ALTER TABLE `matches` ADD COLUMN `snooker_format` varchar(20) NOT NULL DEFAULT ''legacy'' COMMENT ''斯诺克赛制：legacy/free/race_to'' AFTER `best_of_frames`',
  'SELECT 1'
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SELECT COUNT(*) INTO @snooker_target_wins_exists
FROM information_schema.COLUMNS
WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'matches' AND COLUMN_NAME = 'snooker_target_wins';
SET @ddl = IF(
  @snooker_target_wins_exists = 0,
  'ALTER TABLE `matches` ADD COLUMN `snooker_target_wins` smallint unsigned NOT NULL DEFAULT 0 COMMENT ''抢 N 局目标胜局，自由局数为0'' AFTER `snooker_format`',
  'SELECT 1'
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

UPDATE `matches`
SET
  `snooker_format` = 'race_to',
  `snooker_target_wins` = FLOOR(`best_of_frames` / 2) + 1
WHERE `game_type` = 1
  AND `snooker_rules_version` = 2
  AND `best_of_frames` > 0
  AND MOD(`best_of_frames`, 2) = 1;

-- +goose Down

SELECT COUNT(*) INTO @snooker_target_wins_exists
FROM information_schema.COLUMNS
WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'matches' AND COLUMN_NAME = 'snooker_target_wins';
SET @ddl = IF(@snooker_target_wins_exists > 0, 'ALTER TABLE `matches` DROP COLUMN `snooker_target_wins`', 'SELECT 1');
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SELECT COUNT(*) INTO @snooker_format_exists
FROM information_schema.COLUMNS
WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'matches' AND COLUMN_NAME = 'snooker_format';
SET @ddl = IF(@snooker_format_exists > 0, 'ALTER TABLE `matches` DROP COLUMN `snooker_format`', 'SELECT 1');
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;
