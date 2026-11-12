-- +goose Up
-- 球员中文名：WST 只提供英文名，中文名由管理后台维护；WST 同步更新球员时保留该列。
-- 先查 information_schema 再加列，可重复执行。

SELECT COUNT(*) INTO @player_name_zh_exists
FROM information_schema.COLUMNS
WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'players' AND COLUMN_NAME = 'name_zh';
SET @ddl = IF(@player_name_zh_exists = 0, 'ALTER TABLE `players` ADD COLUMN `name_zh` VARCHAR(64) NOT NULL DEFAULT '''' AFTER `display_name`', 'SELECT 1');
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

-- +goose Down
SELECT COUNT(*) INTO @player_name_zh_exists
FROM information_schema.COLUMNS
WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'players' AND COLUMN_NAME = 'name_zh';
SET @ddl = IF(@player_name_zh_exists > 0, 'ALTER TABLE `players` DROP COLUMN `name_zh`', 'SELECT 1');
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;
