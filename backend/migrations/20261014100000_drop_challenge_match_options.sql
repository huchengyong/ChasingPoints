-- +goose Up
-- 约球只保存球种与比赛类型：赛制与公开范围在开局时按类型生成，移除冗余的赛制快照列。

SET @col_exists = (SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'challenges' AND COLUMN_NAME = 'visibility');
SET @ddl = IF(@col_exists > 0, 'ALTER TABLE `challenges` DROP COLUMN `visibility`', 'SELECT 1');
PREPARE challenge_options_stmt FROM @ddl;
EXECUTE challenge_options_stmt;
DEALLOCATE PREPARE challenge_options_stmt;

SET @col_exists = (SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'challenges' AND COLUMN_NAME = 'match_format');
SET @ddl = IF(@col_exists > 0, 'ALTER TABLE `challenges` DROP COLUMN `match_format`', 'SELECT 1');
PREPARE challenge_options_stmt FROM @ddl;
EXECUTE challenge_options_stmt;
DEALLOCATE PREPARE challenge_options_stmt;

SET @col_exists = (SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'challenges' AND COLUMN_NAME = 'target_wins');
SET @ddl = IF(@col_exists > 0, 'ALTER TABLE `challenges` DROP COLUMN `target_wins`', 'SELECT 1');
PREPARE challenge_options_stmt FROM @ddl;
EXECUTE challenge_options_stmt;
DEALLOCATE PREPARE challenge_options_stmt;

SET @col_exists = (SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'challenges' AND COLUMN_NAME = 'snooker_rules_version');
SET @ddl = IF(@col_exists > 0, 'ALTER TABLE `challenges` DROP COLUMN `snooker_rules_version`', 'SELECT 1');
PREPARE challenge_options_stmt FROM @ddl;
EXECUTE challenge_options_stmt;
DEALLOCATE PREPARE challenge_options_stmt;

SET @col_exists = (SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'challenges' AND COLUMN_NAME = 'best_of_frames');
SET @ddl = IF(@col_exists > 0, 'ALTER TABLE `challenges` DROP COLUMN `best_of_frames`', 'SELECT 1');
PREPARE challenge_options_stmt FROM @ddl;
EXECUTE challenge_options_stmt;
DEALLOCATE PREPARE challenge_options_stmt;

SET @col_exists = (SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'challenges' AND COLUMN_NAME = 'snooker_format');
SET @ddl = IF(@col_exists > 0, 'ALTER TABLE `challenges` DROP COLUMN `snooker_format`', 'SELECT 1');
PREPARE challenge_options_stmt FROM @ddl;
EXECUTE challenge_options_stmt;
DEALLOCATE PREPARE challenge_options_stmt;

SET @col_exists = (SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'challenges' AND COLUMN_NAME = 'snooker_target_wins');
SET @ddl = IF(@col_exists > 0, 'ALTER TABLE `challenges` DROP COLUMN `snooker_target_wins`', 'SELECT 1');
PREPARE challenge_options_stmt FROM @ddl;
EXECUTE challenge_options_stmt;
DEALLOCATE PREPARE challenge_options_stmt;

SET @col_exists = (SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'challenges' AND COLUMN_NAME = 'starting_actor');
SET @ddl = IF(@col_exists > 0, 'ALTER TABLE `challenges` DROP COLUMN `starting_actor`', 'SELECT 1');
PREPARE challenge_options_stmt FROM @ddl;
EXECUTE challenge_options_stmt;
DEALLOCATE PREPARE challenge_options_stmt;

-- +goose Down
SET @col_exists = (SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'challenges' AND COLUMN_NAME = 'visibility');
SET @ddl = IF(@col_exists = 0, 'ALTER TABLE `challenges` ADD COLUMN `visibility` VARCHAR(20) NULL COMMENT ''公开范围'' AFTER `match_mode`', 'SELECT 1');
PREPARE challenge_options_stmt FROM @ddl;
EXECUTE challenge_options_stmt;
DEALLOCATE PREPARE challenge_options_stmt;

SET @col_exists = (SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'challenges' AND COLUMN_NAME = 'match_format');
SET @ddl = IF(@col_exists = 0, 'ALTER TABLE `challenges` ADD COLUMN `match_format` VARCHAR(20) NULL COMMENT ''中八/美九赛制'' AFTER `visibility`', 'SELECT 1');
PREPARE challenge_options_stmt FROM @ddl;
EXECUTE challenge_options_stmt;
DEALLOCATE PREPARE challenge_options_stmt;

SET @col_exists = (SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'challenges' AND COLUMN_NAME = 'target_wins');
SET @ddl = IF(@col_exists = 0, 'ALTER TABLE `challenges` ADD COLUMN `target_wins` INT NULL COMMENT ''中八/美九目标胜局'' AFTER `match_format`', 'SELECT 1');
PREPARE challenge_options_stmt FROM @ddl;
EXECUTE challenge_options_stmt;
DEALLOCATE PREPARE challenge_options_stmt;

SET @col_exists = (SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'challenges' AND COLUMN_NAME = 'snooker_rules_version');
SET @ddl = IF(@col_exists = 0, 'ALTER TABLE `challenges` ADD COLUMN `snooker_rules_version` TINYINT NULL COMMENT ''斯诺克规则版本'' AFTER `target_wins`', 'SELECT 1');
PREPARE challenge_options_stmt FROM @ddl;
EXECUTE challenge_options_stmt;
DEALLOCATE PREPARE challenge_options_stmt;

SET @col_exists = (SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'challenges' AND COLUMN_NAME = 'best_of_frames');
SET @ddl = IF(@col_exists = 0, 'ALTER TABLE `challenges` ADD COLUMN `best_of_frames` INT NULL COMMENT ''斯诺克总局数'' AFTER `snooker_rules_version`', 'SELECT 1');
PREPARE challenge_options_stmt FROM @ddl;
EXECUTE challenge_options_stmt;
DEALLOCATE PREPARE challenge_options_stmt;

SET @col_exists = (SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'challenges' AND COLUMN_NAME = 'snooker_format');
SET @ddl = IF(@col_exists = 0, 'ALTER TABLE `challenges` ADD COLUMN `snooker_format` VARCHAR(20) NULL COMMENT ''斯诺克赛制'' AFTER `best_of_frames`', 'SELECT 1');
PREPARE challenge_options_stmt FROM @ddl;
EXECUTE challenge_options_stmt;
DEALLOCATE PREPARE challenge_options_stmt;

SET @col_exists = (SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'challenges' AND COLUMN_NAME = 'snooker_target_wins');
SET @ddl = IF(@col_exists = 0, 'ALTER TABLE `challenges` ADD COLUMN `snooker_target_wins` INT NULL COMMENT ''斯诺克目标胜局'' AFTER `snooker_format`', 'SELECT 1');
PREPARE challenge_options_stmt FROM @ddl;
EXECUTE challenge_options_stmt;
DEALLOCATE PREPARE challenge_options_stmt;

SET @col_exists = (SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'challenges' AND COLUMN_NAME = 'starting_actor');
SET @ddl = IF(@col_exists = 0, 'ALTER TABLE `challenges` ADD COLUMN `starting_actor` TINYINT NULL COMMENT ''斯诺克首局开球方'' AFTER `snooker_target_wins`', 'SELECT 1');
PREPARE challenge_options_stmt FROM @ddl;
EXECUTE challenge_options_stmt;
DEALLOCATE PREPARE challenge_options_stmt;
