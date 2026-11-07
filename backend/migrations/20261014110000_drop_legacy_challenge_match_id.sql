-- +goose Up
-- 约球比赛关联以 matches.challenge_id 为唯一权威来源，移除不再写入的旧 challenges.match_id 列（其单列索引随列一并删除）。

SET @col_exists = (SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'challenges' AND COLUMN_NAME = 'match_id');
SET @ddl = IF(@col_exists > 0, 'ALTER TABLE `challenges` DROP COLUMN `match_id`', 'SELECT 1');
PREPARE challenge_legacy_stmt FROM @ddl;
EXECUTE challenge_legacy_stmt;
DEALLOCATE PREPARE challenge_legacy_stmt;

-- +goose Down
SET @col_exists = (SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'challenges' AND COLUMN_NAME = 'match_id');
SET @ddl = IF(@col_exists = 0, 'ALTER TABLE `challenges` ADD COLUMN `match_id` BIGINT UNSIGNED NULL COMMENT ''旧关联对局ID，已停用'' AFTER `status`', 'SELECT 1');
PREPARE challenge_legacy_stmt FROM @ddl;
EXECUTE challenge_legacy_stmt;
DEALLOCATE PREPARE challenge_legacy_stmt;

SET @idx_exists = (SELECT COUNT(*) FROM information_schema.STATISTICS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'challenges' AND INDEX_NAME = 'idx_match_id');
SET @ddl = IF(@idx_exists = 0, 'ALTER TABLE `challenges` ADD INDEX `idx_match_id` (`match_id`)', 'SELECT 1');
PREPARE challenge_legacy_stmt FROM @ddl;
EXECUTE challenge_legacy_stmt;
DEALLOCATE PREPARE challenge_legacy_stmt;
