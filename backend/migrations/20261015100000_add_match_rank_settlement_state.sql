-- +goose Up
-- 约球排位手动结束后的延后排位结算：状态 none/pending/settled/voided，pending 时记录生效时刻。
-- 仅回填约球关联的已完成排位胜负比赛（走 uk_matches_challenge_id），其余存量保持 none。

SELECT COUNT(*) INTO @rank_settlement_status_exists
FROM information_schema.COLUMNS
WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'matches' AND COLUMN_NAME = 'rank_settlement_status';
SET @ddl = IF(
  @rank_settlement_status_exists = 0,
  'ALTER TABLE `matches` ADD COLUMN `rank_settlement_status` varchar(20) NOT NULL DEFAULT ''none'' COMMENT ''排位结算状态: none/pending/settled/voided'' AFTER `completion_source`',
  'SELECT 1'
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SELECT COUNT(*) INTO @rank_settlement_due_at_exists
FROM information_schema.COLUMNS
WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'matches' AND COLUMN_NAME = 'rank_settlement_due_at';
SET @ddl = IF(
  @rank_settlement_due_at_exists = 0,
  'ALTER TABLE `matches` ADD COLUMN `rank_settlement_due_at` datetime DEFAULT NULL COMMENT ''排位结算生效时刻；pending时为到期时刻，生效后保留作为实际生效时刻'' AFTER `rank_settlement_status`',
  'SELECT 1'
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SELECT COUNT(*) INTO @idx_rank_settlement_exists
FROM information_schema.STATISTICS
WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'matches' AND INDEX_NAME = 'idx_matches_rank_settlement';
SET @ddl = IF(
  @idx_rank_settlement_exists = 0,
  'ALTER TABLE `matches` ADD INDEX `idx_matches_rank_settlement` (`rank_settlement_status`, `rank_settlement_due_at`)',
  'SELECT 1'
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SELECT COUNT(*) INTO @rank_excluded_exists
FROM information_schema.COLUMNS
WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'match_participant_results' AND COLUMN_NAME = 'rank_excluded';
SET @ddl = IF(
  @rank_excluded_exists = 0,
  'ALTER TABLE `match_participant_results` ADD COLUMN `rank_excluded` tinyint(1) NOT NULL DEFAULT 0 COMMENT ''排位结算待生效或已作废：历史可见，不计排位口径'' AFTER `opponent_rank_bucket`',
  'SELECT 1'
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

UPDATE `matches`
SET `rank_settlement_status` = 'settled'
WHERE `challenge_id` IS NOT NULL
  AND `status` = 2
  AND `match_mode` = 'ranked'
  AND `result` IN (1, 2)
  AND `rank_settlement_status` = 'none';

-- +goose Down

SELECT COUNT(*) INTO @rank_excluded_exists
FROM information_schema.COLUMNS
WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'match_participant_results' AND COLUMN_NAME = 'rank_excluded';
SET @ddl = IF(@rank_excluded_exists > 0, 'ALTER TABLE `match_participant_results` DROP COLUMN `rank_excluded`', 'SELECT 1');
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SELECT COUNT(*) INTO @idx_rank_settlement_exists
FROM information_schema.STATISTICS
WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'matches' AND INDEX_NAME = 'idx_matches_rank_settlement';
SET @ddl = IF(@idx_rank_settlement_exists > 0, 'ALTER TABLE `matches` DROP INDEX `idx_matches_rank_settlement`', 'SELECT 1');
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SELECT COUNT(*) INTO @rank_settlement_due_at_exists
FROM information_schema.COLUMNS
WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'matches' AND COLUMN_NAME = 'rank_settlement_due_at';
SET @ddl = IF(@rank_settlement_due_at_exists > 0, 'ALTER TABLE `matches` DROP COLUMN `rank_settlement_due_at`', 'SELECT 1');
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SELECT COUNT(*) INTO @rank_settlement_status_exists
FROM information_schema.COLUMNS
WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'matches' AND COLUMN_NAME = 'rank_settlement_status';
SET @ddl = IF(@rank_settlement_status_exists > 0, 'ALTER TABLE `matches` DROP COLUMN `rank_settlement_status`', 'SELECT 1');
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;
