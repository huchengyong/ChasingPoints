-- +goose Up
-- 熟人约球：扩展 challenges 保存约球选项与状态，matches.challenge_id 作为比赛来源权威关联，用户增加好友限制设置。

SET @col_exists = (SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'challenges' AND COLUMN_NAME = 'scheduled_date');
SET @ddl = IF(@col_exists = 0, 'ALTER TABLE `challenges` ADD COLUMN `scheduled_date` DATE NULL COMMENT ''预约日期（北京时间）'' AFTER `game_type`', 'SELECT 1');
PREPARE quick_match_stmt FROM @ddl;
EXECUTE quick_match_stmt;
DEALLOCATE PREPARE quick_match_stmt;

SET @col_exists = (SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'challenges' AND COLUMN_NAME = 'start_hour');
SET @ddl = IF(@col_exists = 0, 'ALTER TABLE `challenges` ADD COLUMN `start_hour` TINYINT NULL COMMENT ''预约开始小时 0-23'' AFTER `scheduled_date`', 'SELECT 1');
PREPARE quick_match_stmt FROM @ddl;
EXECUTE quick_match_stmt;
DEALLOCATE PREPARE quick_match_stmt;

SET @col_exists = (SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'challenges' AND COLUMN_NAME = 'end_hour');
SET @ddl = IF(@col_exists = 0, 'ALTER TABLE `challenges` ADD COLUMN `end_hour` TINYINT NULL COMMENT ''预约结束小时 1-24'' AFTER `start_hour`', 'SELECT 1');
PREPARE quick_match_stmt FROM @ddl;
EXECUTE quick_match_stmt;
DEALLOCATE PREPARE quick_match_stmt;

SET @col_exists = (SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'challenges' AND COLUMN_NAME = 'match_mode');
SET @ddl = IF(@col_exists = 0, 'ALTER TABLE `challenges` ADD COLUMN `match_mode` VARCHAR(20) NULL COMMENT ''排位/练习'' AFTER `end_hour`', 'SELECT 1');
PREPARE quick_match_stmt FROM @ddl;
EXECUTE quick_match_stmt;
DEALLOCATE PREPARE quick_match_stmt;

SET @col_exists = (SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'challenges' AND COLUMN_NAME = 'visibility');
SET @ddl = IF(@col_exists = 0, 'ALTER TABLE `challenges` ADD COLUMN `visibility` VARCHAR(20) NULL COMMENT ''公开范围'' AFTER `match_mode`', 'SELECT 1');
PREPARE quick_match_stmt FROM @ddl;
EXECUTE quick_match_stmt;
DEALLOCATE PREPARE quick_match_stmt;

SET @col_exists = (SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'challenges' AND COLUMN_NAME = 'match_format');
SET @ddl = IF(@col_exists = 0, 'ALTER TABLE `challenges` ADD COLUMN `match_format` VARCHAR(20) NULL COMMENT ''中八/美九赛制'' AFTER `visibility`', 'SELECT 1');
PREPARE quick_match_stmt FROM @ddl;
EXECUTE quick_match_stmt;
DEALLOCATE PREPARE quick_match_stmt;

SET @col_exists = (SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'challenges' AND COLUMN_NAME = 'target_wins');
SET @ddl = IF(@col_exists = 0, 'ALTER TABLE `challenges` ADD COLUMN `target_wins` INT NULL COMMENT ''中八/美九目标胜局'' AFTER `match_format`', 'SELECT 1');
PREPARE quick_match_stmt FROM @ddl;
EXECUTE quick_match_stmt;
DEALLOCATE PREPARE quick_match_stmt;

SET @col_exists = (SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'challenges' AND COLUMN_NAME = 'snooker_rules_version');
SET @ddl = IF(@col_exists = 0, 'ALTER TABLE `challenges` ADD COLUMN `snooker_rules_version` TINYINT NULL COMMENT ''斯诺克规则版本'' AFTER `target_wins`', 'SELECT 1');
PREPARE quick_match_stmt FROM @ddl;
EXECUTE quick_match_stmt;
DEALLOCATE PREPARE quick_match_stmt;

SET @col_exists = (SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'challenges' AND COLUMN_NAME = 'best_of_frames');
SET @ddl = IF(@col_exists = 0, 'ALTER TABLE `challenges` ADD COLUMN `best_of_frames` INT NULL COMMENT ''斯诺克总局数'' AFTER `snooker_rules_version`', 'SELECT 1');
PREPARE quick_match_stmt FROM @ddl;
EXECUTE quick_match_stmt;
DEALLOCATE PREPARE quick_match_stmt;

SET @col_exists = (SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'challenges' AND COLUMN_NAME = 'snooker_format');
SET @ddl = IF(@col_exists = 0, 'ALTER TABLE `challenges` ADD COLUMN `snooker_format` VARCHAR(20) NULL COMMENT ''斯诺克赛制'' AFTER `best_of_frames`', 'SELECT 1');
PREPARE quick_match_stmt FROM @ddl;
EXECUTE quick_match_stmt;
DEALLOCATE PREPARE quick_match_stmt;

SET @col_exists = (SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'challenges' AND COLUMN_NAME = 'snooker_target_wins');
SET @ddl = IF(@col_exists = 0, 'ALTER TABLE `challenges` ADD COLUMN `snooker_target_wins` INT NULL COMMENT ''斯诺克目标胜局'' AFTER `snooker_format`', 'SELECT 1');
PREPARE quick_match_stmt FROM @ddl;
EXECUTE quick_match_stmt;
DEALLOCATE PREPARE quick_match_stmt;

SET @col_exists = (SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'challenges' AND COLUMN_NAME = 'starting_actor');
SET @ddl = IF(@col_exists = 0, 'ALTER TABLE `challenges` ADD COLUMN `starting_actor` TINYINT NULL COMMENT ''斯诺克首局开球方'' AFTER `snooker_target_wins`', 'SELECT 1');
PREPARE quick_match_stmt FROM @ddl;
EXECUTE quick_match_stmt;
DEALLOCATE PREPARE quick_match_stmt;

SET @col_exists = (SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'challenges' AND COLUMN_NAME = 'waiting_user_id');
SET @ddl = IF(@col_exists = 0, 'ALTER TABLE `challenges` ADD COLUMN `waiting_user_id` BIGINT UNSIGNED NULL COMMENT ''当前等待进入的用户'' AFTER `starting_actor`', 'SELECT 1');
PREPARE quick_match_stmt FROM @ddl;
EXECUTE quick_match_stmt;
DEALLOCATE PREPARE quick_match_stmt;

SET @col_exists = (SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'challenges' AND COLUMN_NAME = 'waiting_entered_at');
SET @ddl = IF(@col_exists = 0, 'ALTER TABLE `challenges` ADD COLUMN `waiting_entered_at` DATETIME NULL COMMENT ''进入等待时间'' AFTER `waiting_user_id`', 'SELECT 1');
PREPARE quick_match_stmt FROM @ddl;
EXECUTE quick_match_stmt;
DEALLOCATE PREPARE quick_match_stmt;

SET @col_exists = (SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'challenges' AND COLUMN_NAME = 'merged_into_id');
SET @ddl = IF(@col_exists = 0, 'ALTER TABLE `challenges` ADD COLUMN `merged_into_id` BIGINT UNSIGNED NULL COMMENT ''互邀归并指向的主记录'' AFTER `waiting_entered_at`', 'SELECT 1');
PREPARE quick_match_stmt FROM @ddl;
EXECUTE quick_match_stmt;
DEALLOCATE PREPARE quick_match_stmt;

SET @col_exists = (SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'challenges' AND COLUMN_NAME = 'close_reason');
SET @ddl = IF(@col_exists = 0, 'ALTER TABLE `challenges` ADD COLUMN `close_reason` VARCHAR(50) NULL COMMENT ''终态原因'' AFTER `merged_into_id`', 'SELECT 1');
PREPARE quick_match_stmt FROM @ddl;
EXECUTE quick_match_stmt;
DEALLOCATE PREPARE quick_match_stmt;

SET @idx_exists = (SELECT COUNT(*) FROM information_schema.STATISTICS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'challenges' AND INDEX_NAME = 'idx_challenges_merged_into');
SET @ddl = IF(@idx_exists = 0, 'ALTER TABLE `challenges` ADD INDEX `idx_challenges_merged_into` (`merged_into_id`)', 'SELECT 1');
PREPARE quick_match_stmt FROM @ddl;
EXECUTE quick_match_stmt;
DEALLOCATE PREPARE quick_match_stmt;

SET @idx_exists = (SELECT COUNT(*) FROM information_schema.STATISTICS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'challenges' AND INDEX_NAME = 'idx_challenges_status_expires');
SET @ddl = IF(@idx_exists = 0, 'ALTER TABLE `challenges` ADD INDEX `idx_challenges_status_expires` (`status`, `expires_at`)', 'SELECT 1');
PREPARE quick_match_stmt FROM @ddl;
EXECUTE quick_match_stmt;
DEALLOCATE PREPARE quick_match_stmt;

SET @col_exists = (SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'matches' AND COLUMN_NAME = 'challenge_id');
SET @ddl = IF(@col_exists = 0, 'ALTER TABLE `matches` ADD COLUMN `challenge_id` BIGINT UNSIGNED NULL COMMENT ''来源约球ID'' AFTER `referee_joined_at`', 'SELECT 1');
PREPARE quick_match_stmt FROM @ddl;
EXECUTE quick_match_stmt;
DEALLOCATE PREPARE quick_match_stmt;

SET @idx_exists = (SELECT COUNT(*) FROM information_schema.STATISTICS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'matches' AND INDEX_NAME = 'uk_matches_challenge_id');
SET @ddl = IF(@idx_exists = 0, 'ALTER TABLE `matches` ADD UNIQUE INDEX `uk_matches_challenge_id` (`challenge_id`)', 'SELECT 1');
PREPARE quick_match_stmt FROM @ddl;
EXECUTE quick_match_stmt;
DEALLOCATE PREPARE quick_match_stmt;

SET @col_exists = (SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'users' AND COLUMN_NAME = 'friends_only_challenges');
SET @ddl = IF(@col_exists = 0, 'ALTER TABLE `users` ADD COLUMN `friends_only_challenges` TINYINT(1) NOT NULL DEFAULT 0 COMMENT ''仅允许好友约球'' AFTER `hide_match_record`', 'SELECT 1');
PREPARE quick_match_stmt FROM @ddl;
EXECUTE quick_match_stmt;
DEALLOCATE PREPARE quick_match_stmt;

-- +goose Down
SET @idx_exists = (SELECT COUNT(*) FROM information_schema.STATISTICS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'matches' AND INDEX_NAME = 'uk_matches_challenge_id');
SET @ddl = IF(@idx_exists > 0, 'ALTER TABLE `matches` DROP INDEX `uk_matches_challenge_id`', 'SELECT 1');
PREPARE quick_match_stmt FROM @ddl;
EXECUTE quick_match_stmt;
DEALLOCATE PREPARE quick_match_stmt;

SET @col_exists = (SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'matches' AND COLUMN_NAME = 'challenge_id');
SET @ddl = IF(@col_exists = 1, 'ALTER TABLE `matches` DROP COLUMN `challenge_id`', 'SELECT 1');
PREPARE quick_match_stmt FROM @ddl;
EXECUTE quick_match_stmt;
DEALLOCATE PREPARE quick_match_stmt;

SET @idx_exists = (SELECT COUNT(*) FROM information_schema.STATISTICS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'challenges' AND INDEX_NAME = 'idx_challenges_status_expires');
SET @ddl = IF(@idx_exists > 0, 'ALTER TABLE `challenges` DROP INDEX `idx_challenges_status_expires`', 'SELECT 1');
PREPARE quick_match_stmt FROM @ddl;
EXECUTE quick_match_stmt;
DEALLOCATE PREPARE quick_match_stmt;

SET @idx_exists = (SELECT COUNT(*) FROM information_schema.STATISTICS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'challenges' AND INDEX_NAME = 'idx_challenges_merged_into');
SET @ddl = IF(@idx_exists > 0, 'ALTER TABLE `challenges` DROP INDEX `idx_challenges_merged_into`', 'SELECT 1');
PREPARE quick_match_stmt FROM @ddl;
EXECUTE quick_match_stmt;
DEALLOCATE PREPARE quick_match_stmt;

SET @col_exists = (SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'users' AND COLUMN_NAME = 'friends_only_challenges');
SET @ddl = IF(@col_exists = 1, 'ALTER TABLE `users` DROP COLUMN `friends_only_challenges`', 'SELECT 1');
PREPARE quick_match_stmt FROM @ddl;
EXECUTE quick_match_stmt;
DEALLOCATE PREPARE quick_match_stmt;

SET @col_exists = (SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'challenges' AND COLUMN_NAME = 'close_reason');
SET @ddl = IF(@col_exists = 1, 'ALTER TABLE `challenges` DROP COLUMN `close_reason`', 'SELECT 1');
PREPARE quick_match_stmt FROM @ddl;
EXECUTE quick_match_stmt;
DEALLOCATE PREPARE quick_match_stmt;

SET @col_exists = (SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'challenges' AND COLUMN_NAME = 'merged_into_id');
SET @ddl = IF(@col_exists = 1, 'ALTER TABLE `challenges` DROP COLUMN `merged_into_id`', 'SELECT 1');
PREPARE quick_match_stmt FROM @ddl;
EXECUTE quick_match_stmt;
DEALLOCATE PREPARE quick_match_stmt;

SET @col_exists = (SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'challenges' AND COLUMN_NAME = 'waiting_entered_at');
SET @ddl = IF(@col_exists = 1, 'ALTER TABLE `challenges` DROP COLUMN `waiting_entered_at`', 'SELECT 1');
PREPARE quick_match_stmt FROM @ddl;
EXECUTE quick_match_stmt;
DEALLOCATE PREPARE quick_match_stmt;

SET @col_exists = (SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'challenges' AND COLUMN_NAME = 'waiting_user_id');
SET @ddl = IF(@col_exists = 1, 'ALTER TABLE `challenges` DROP COLUMN `waiting_user_id`', 'SELECT 1');
PREPARE quick_match_stmt FROM @ddl;
EXECUTE quick_match_stmt;
DEALLOCATE PREPARE quick_match_stmt;

SET @col_exists = (SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'challenges' AND COLUMN_NAME = 'starting_actor');
SET @ddl = IF(@col_exists = 1, 'ALTER TABLE `challenges` DROP COLUMN `starting_actor`', 'SELECT 1');
PREPARE quick_match_stmt FROM @ddl;
EXECUTE quick_match_stmt;
DEALLOCATE PREPARE quick_match_stmt;

SET @col_exists = (SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'challenges' AND COLUMN_NAME = 'snooker_target_wins');
SET @ddl = IF(@col_exists = 1, 'ALTER TABLE `challenges` DROP COLUMN `snooker_target_wins`', 'SELECT 1');
PREPARE quick_match_stmt FROM @ddl;
EXECUTE quick_match_stmt;
DEALLOCATE PREPARE quick_match_stmt;

SET @col_exists = (SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'challenges' AND COLUMN_NAME = 'snooker_format');
SET @ddl = IF(@col_exists = 1, 'ALTER TABLE `challenges` DROP COLUMN `snooker_format`', 'SELECT 1');
PREPARE quick_match_stmt FROM @ddl;
EXECUTE quick_match_stmt;
DEALLOCATE PREPARE quick_match_stmt;

SET @col_exists = (SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'challenges' AND COLUMN_NAME = 'best_of_frames');
SET @ddl = IF(@col_exists = 1, 'ALTER TABLE `challenges` DROP COLUMN `best_of_frames`', 'SELECT 1');
PREPARE quick_match_stmt FROM @ddl;
EXECUTE quick_match_stmt;
DEALLOCATE PREPARE quick_match_stmt;

SET @col_exists = (SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'challenges' AND COLUMN_NAME = 'snooker_rules_version');
SET @ddl = IF(@col_exists = 1, 'ALTER TABLE `challenges` DROP COLUMN `snooker_rules_version`', 'SELECT 1');
PREPARE quick_match_stmt FROM @ddl;
EXECUTE quick_match_stmt;
DEALLOCATE PREPARE quick_match_stmt;

SET @col_exists = (SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'challenges' AND COLUMN_NAME = 'target_wins');
SET @ddl = IF(@col_exists = 1, 'ALTER TABLE `challenges` DROP COLUMN `target_wins`', 'SELECT 1');
PREPARE quick_match_stmt FROM @ddl;
EXECUTE quick_match_stmt;
DEALLOCATE PREPARE quick_match_stmt;

SET @col_exists = (SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'challenges' AND COLUMN_NAME = 'match_format');
SET @ddl = IF(@col_exists = 1, 'ALTER TABLE `challenges` DROP COLUMN `match_format`', 'SELECT 1');
PREPARE quick_match_stmt FROM @ddl;
EXECUTE quick_match_stmt;
DEALLOCATE PREPARE quick_match_stmt;

SET @col_exists = (SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'challenges' AND COLUMN_NAME = 'visibility');
SET @ddl = IF(@col_exists = 1, 'ALTER TABLE `challenges` DROP COLUMN `visibility`', 'SELECT 1');
PREPARE quick_match_stmt FROM @ddl;
EXECUTE quick_match_stmt;
DEALLOCATE PREPARE quick_match_stmt;

SET @col_exists = (SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'challenges' AND COLUMN_NAME = 'match_mode');
SET @ddl = IF(@col_exists = 1, 'ALTER TABLE `challenges` DROP COLUMN `match_mode`', 'SELECT 1');
PREPARE quick_match_stmt FROM @ddl;
EXECUTE quick_match_stmt;
DEALLOCATE PREPARE quick_match_stmt;

SET @col_exists = (SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'challenges' AND COLUMN_NAME = 'end_hour');
SET @ddl = IF(@col_exists = 1, 'ALTER TABLE `challenges` DROP COLUMN `end_hour`', 'SELECT 1');
PREPARE quick_match_stmt FROM @ddl;
EXECUTE quick_match_stmt;
DEALLOCATE PREPARE quick_match_stmt;

SET @col_exists = (SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'challenges' AND COLUMN_NAME = 'start_hour');
SET @ddl = IF(@col_exists = 1, 'ALTER TABLE `challenges` DROP COLUMN `start_hour`', 'SELECT 1');
PREPARE quick_match_stmt FROM @ddl;
EXECUTE quick_match_stmt;
DEALLOCATE PREPARE quick_match_stmt;

SET @col_exists = (SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'challenges' AND COLUMN_NAME = 'scheduled_date');
SET @ddl = IF(@col_exists = 1, 'ALTER TABLE `challenges` DROP COLUMN `scheduled_date`', 'SELECT 1');
PREPARE quick_match_stmt FROM @ddl;
EXECUTE quick_match_stmt;
DEALLOCATE PREPARE quick_match_stmt;
