-- +goose Up
-- 标准斯诺克规则版本与赛制字段

SELECT COUNT(*) INTO @snooker_rules_version_exists
FROM information_schema.COLUMNS
WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'matches' AND COLUMN_NAME = 'snooker_rules_version';
SET @ddl = IF(
  @snooker_rules_version_exists = 0,
  'ALTER TABLE `matches` ADD COLUMN `snooker_rules_version` tinyint unsigned NOT NULL DEFAULT 1 COMMENT ''斯诺克规则版本：1=旧记分 2=WPBSA标准'' AFTER `game_mode`',
  'SELECT 1'
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SELECT COUNT(*) INTO @best_of_frames_exists
FROM information_schema.COLUMNS
WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'matches' AND COLUMN_NAME = 'best_of_frames';
SET @ddl = IF(
  @best_of_frames_exists = 0,
  'ALTER TABLE `matches` ADD COLUMN `best_of_frames` smallint unsigned NOT NULL DEFAULT 0 COMMENT ''总局数，版本2斯诺克为正奇数'' AFTER `snooker_rules_version`',
  'SELECT 1'
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SELECT COUNT(*) INTO @starting_actor_exists
FROM information_schema.COLUMNS
WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'matches' AND COLUMN_NAME = 'starting_actor';
SET @ddl = IF(
  @starting_actor_exists = 0,
  'ALTER TABLE `matches` ADD COLUMN `starting_actor` tinyint unsigned NOT NULL DEFAULT 0 COMMENT ''首局开球方：1=选手1 2=选手2'' AFTER `best_of_frames`',
  'SELECT 1'
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

UPDATE `matches`
SET `snooker_rules_version` = 1
WHERE `snooker_rules_version` IS NULL OR `snooker_rules_version` = 0;

-- +goose Down

SELECT COUNT(*) INTO @starting_actor_exists
FROM information_schema.COLUMNS
WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'matches' AND COLUMN_NAME = 'starting_actor';
SET @ddl = IF(@starting_actor_exists > 0, 'ALTER TABLE `matches` DROP COLUMN `starting_actor`', 'SELECT 1');
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SELECT COUNT(*) INTO @best_of_frames_exists
FROM information_schema.COLUMNS
WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'matches' AND COLUMN_NAME = 'best_of_frames';
SET @ddl = IF(@best_of_frames_exists > 0, 'ALTER TABLE `matches` DROP COLUMN `best_of_frames`', 'SELECT 1');
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SELECT COUNT(*) INTO @snooker_rules_version_exists
FROM information_schema.COLUMNS
WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'matches' AND COLUMN_NAME = 'snooker_rules_version';
SET @ddl = IF(@snooker_rules_version_exists > 0, 'ALTER TABLE `matches` DROP COLUMN `snooker_rules_version`', 'SELECT 1');
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;
