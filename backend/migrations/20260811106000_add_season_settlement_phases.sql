-- +goose Up

SET @column_exists = (SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'season_settlements' AND COLUMN_NAME = 'records_completed_at');
SET @ddl = IF(@column_exists = 0, 'ALTER TABLE `season_settlements` ADD COLUMN `records_completed_at` DATETIME NULL AFTER `started_at`', 'SELECT 1');
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @column_exists = (SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'season_settlements' AND COLUMN_NAME = 'challenges_completed_at');
SET @ddl = IF(@column_exists = 0, 'ALTER TABLE `season_settlements` ADD COLUMN `challenges_completed_at` DATETIME NULL AFTER `records_completed_at`', 'SELECT 1');
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @column_exists = (SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'season_settlements' AND COLUMN_NAME = 'titles_completed_at');
SET @ddl = IF(@column_exists = 0, 'ALTER TABLE `season_settlements` ADD COLUMN `titles_completed_at` DATETIME NULL AFTER `challenges_completed_at`', 'SELECT 1');
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @column_exists = (SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'season_settlements' AND COLUMN_NAME = 'notifications_completed_at');
SET @ddl = IF(@column_exists = 0, 'ALTER TABLE `season_settlements` ADD COLUMN `notifications_completed_at` DATETIME NULL AFTER `titles_completed_at`', 'SELECT 1');
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

-- +goose Down

SET @column_exists = (SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'season_settlements' AND COLUMN_NAME = 'notifications_completed_at');
SET @ddl = IF(@column_exists > 0, 'ALTER TABLE `season_settlements` DROP COLUMN `notifications_completed_at`', 'SELECT 1');
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @column_exists = (SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'season_settlements' AND COLUMN_NAME = 'titles_completed_at');
SET @ddl = IF(@column_exists > 0, 'ALTER TABLE `season_settlements` DROP COLUMN `titles_completed_at`', 'SELECT 1');
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @column_exists = (SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'season_settlements' AND COLUMN_NAME = 'challenges_completed_at');
SET @ddl = IF(@column_exists > 0, 'ALTER TABLE `season_settlements` DROP COLUMN `challenges_completed_at`', 'SELECT 1');
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @column_exists = (SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'season_settlements' AND COLUMN_NAME = 'records_completed_at');
SET @ddl = IF(@column_exists > 0, 'ALTER TABLE `season_settlements` DROP COLUMN `records_completed_at`', 'SELECT 1');
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;
