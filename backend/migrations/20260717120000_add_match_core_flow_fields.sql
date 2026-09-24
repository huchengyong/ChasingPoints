-- +goose Up
-- 对局核心链路：模式、公开范围与排位结束确认状态

SELECT COUNT(*) INTO @match_mode_exists
FROM information_schema.COLUMNS
WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'matches' AND COLUMN_NAME = 'match_mode';
SET @ddl = IF(
  @match_mode_exists = 0,
  'ALTER TABLE `matches` ADD COLUMN `match_mode` varchar(20) NOT NULL DEFAULT ''ranked'' COMMENT ''对局模式：practice/ranked'' AFTER `game_mode`',
  'SELECT 1'
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SELECT COUNT(*) INTO @visibility_exists
FROM information_schema.COLUMNS
WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'matches' AND COLUMN_NAME = 'visibility';
SET @ddl = IF(
  @visibility_exists = 0,
  'ALTER TABLE `matches` ADD COLUMN `visibility` varchar(20) NOT NULL DEFAULT ''public'' COMMENT ''公开范围：private/public'' AFTER `match_mode`',
  'SELECT 1'
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SELECT COUNT(*) INTO @finish_state_exists
FROM information_schema.COLUMNS
WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'matches' AND COLUMN_NAME = 'finish_state';

SELECT COUNT(*) INTO @finish_confirmation_required_exists
FROM information_schema.COLUMNS
WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'matches' AND COLUMN_NAME = 'finish_confirmation_required';
SET @ddl = IF(
  @finish_confirmation_required_exists = 0,
  'ALTER TABLE `matches` ADD COLUMN `finish_confirmation_required` tinyint(1) NOT NULL DEFAULT 0 COMMENT ''是否要求结束确认'' AFTER `visibility`',
  'SELECT 1'
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @ddl = IF(
  @finish_state_exists = 0,
  'ALTER TABLE `matches` ADD COLUMN `finish_state` varchar(32) NOT NULL DEFAULT ''none'' COMMENT ''结束确认状态'' AFTER `finish_confirmation_required`',
  'SELECT 1'
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SELECT COUNT(*) INTO @finish_requested_by_exists
FROM information_schema.COLUMNS
WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'matches' AND COLUMN_NAME = 'finish_requested_by';
SET @ddl = IF(
  @finish_requested_by_exists = 0,
  'ALTER TABLE `matches` ADD COLUMN `finish_requested_by` bigint unsigned DEFAULT NULL COMMENT ''结束请求发起方'' AFTER `finish_state`',
  'SELECT 1'
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SELECT COUNT(*) INTO @finish_requested_at_exists
FROM information_schema.COLUMNS
WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'matches' AND COLUMN_NAME = 'finish_requested_at';
SET @ddl = IF(
  @finish_requested_at_exists = 0,
  'ALTER TABLE `matches` ADD COLUMN `finish_requested_at` datetime DEFAULT NULL COMMENT ''结束请求时间'' AFTER `finish_requested_by`',
  'SELECT 1'
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SELECT COUNT(*) INTO @finish_request_revision_exists
FROM information_schema.COLUMNS
WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'matches' AND COLUMN_NAME = 'finish_request_revision';
SET @ddl = IF(
  @finish_request_revision_exists = 0,
  'ALTER TABLE `matches` ADD COLUMN `finish_request_revision` bigint unsigned NOT NULL DEFAULT 0 COMMENT ''结束请求冻结版本'' AFTER `finish_requested_at`',
  'SELECT 1'
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

UPDATE `matches`
SET `match_mode` = 'ranked'
WHERE `match_mode` IS NULL OR `match_mode` = '';

UPDATE `matches`
SET `visibility` = 'public'
WHERE `visibility` IS NULL OR `visibility` = '';

UPDATE `matches`
SET `finish_state` = 'none'
WHERE `finish_state` IS NULL OR `finish_state` = '';

SELECT COUNT(*) INTO @match_mode_index_exists
FROM information_schema.STATISTICS
WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'matches' AND INDEX_NAME = 'idx_match_mode';
SET @ddl = IF(@match_mode_index_exists = 0, 'ALTER TABLE `matches` ADD KEY `idx_match_mode` (`match_mode`)', 'SELECT 1');
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SELECT COUNT(*) INTO @visibility_index_exists
FROM information_schema.STATISTICS
WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'matches' AND INDEX_NAME = 'idx_visibility';
SET @ddl = IF(@visibility_index_exists = 0, 'ALTER TABLE `matches` ADD KEY `idx_visibility` (`visibility`)', 'SELECT 1');
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SELECT COUNT(*) INTO @finish_state_index_exists
FROM information_schema.STATISTICS
WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'matches' AND INDEX_NAME = 'idx_finish_state';
SET @ddl = IF(@finish_state_index_exists = 0, 'ALTER TABLE `matches` ADD KEY `idx_finish_state` (`finish_state`)', 'SELECT 1');
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SELECT COUNT(*) INTO @finish_requested_by_index_exists
FROM information_schema.STATISTICS
WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'matches' AND INDEX_NAME = 'idx_finish_requested_by';
SET @ddl = IF(@finish_requested_by_index_exists = 0, 'ALTER TABLE `matches` ADD KEY `idx_finish_requested_by` (`finish_requested_by`)', 'SELECT 1');
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

-- +goose Down

SELECT COUNT(*) INTO @finish_requested_by_index_exists
FROM information_schema.STATISTICS
WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'matches' AND INDEX_NAME = 'idx_finish_requested_by';
SET @ddl = IF(@finish_requested_by_index_exists > 0, 'ALTER TABLE `matches` DROP INDEX `idx_finish_requested_by`', 'SELECT 1');
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SELECT COUNT(*) INTO @finish_state_index_exists
FROM information_schema.STATISTICS
WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'matches' AND INDEX_NAME = 'idx_finish_state';
SET @ddl = IF(@finish_state_index_exists > 0, 'ALTER TABLE `matches` DROP INDEX `idx_finish_state`', 'SELECT 1');
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SELECT COUNT(*) INTO @visibility_index_exists
FROM information_schema.STATISTICS
WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'matches' AND INDEX_NAME = 'idx_visibility';
SET @ddl = IF(@visibility_index_exists > 0, 'ALTER TABLE `matches` DROP INDEX `idx_visibility`', 'SELECT 1');
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SELECT COUNT(*) INTO @match_mode_index_exists
FROM information_schema.STATISTICS
WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'matches' AND INDEX_NAME = 'idx_match_mode';
SET @ddl = IF(@match_mode_index_exists > 0, 'ALTER TABLE `matches` DROP INDEX `idx_match_mode`', 'SELECT 1');
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SELECT COUNT(*) INTO @finish_request_revision_exists
FROM information_schema.COLUMNS
WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'matches' AND COLUMN_NAME = 'finish_request_revision';
SET @ddl = IF(@finish_request_revision_exists > 0, 'ALTER TABLE `matches` DROP COLUMN `finish_request_revision`', 'SELECT 1');
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SELECT COUNT(*) INTO @finish_requested_at_exists
FROM information_schema.COLUMNS
WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'matches' AND COLUMN_NAME = 'finish_requested_at';
SET @ddl = IF(@finish_requested_at_exists > 0, 'ALTER TABLE `matches` DROP COLUMN `finish_requested_at`', 'SELECT 1');
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SELECT COUNT(*) INTO @finish_requested_by_exists
FROM information_schema.COLUMNS
WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'matches' AND COLUMN_NAME = 'finish_requested_by';
SET @ddl = IF(@finish_requested_by_exists > 0, 'ALTER TABLE `matches` DROP COLUMN `finish_requested_by`', 'SELECT 1');
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SELECT COUNT(*) INTO @finish_state_exists
FROM information_schema.COLUMNS
WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'matches' AND COLUMN_NAME = 'finish_state';
SET @ddl = IF(@finish_state_exists > 0, 'ALTER TABLE `matches` DROP COLUMN `finish_state`', 'SELECT 1');
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SELECT COUNT(*) INTO @finish_confirmation_required_exists
FROM information_schema.COLUMNS
WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'matches' AND COLUMN_NAME = 'finish_confirmation_required';
SET @ddl = IF(@finish_confirmation_required_exists > 0, 'ALTER TABLE `matches` DROP COLUMN `finish_confirmation_required`', 'SELECT 1');
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SELECT COUNT(*) INTO @visibility_exists
FROM information_schema.COLUMNS
WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'matches' AND COLUMN_NAME = 'visibility';
SET @ddl = IF(@visibility_exists > 0, 'ALTER TABLE `matches` DROP COLUMN `visibility`', 'SELECT 1');
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SELECT COUNT(*) INTO @match_mode_exists
FROM information_schema.COLUMNS
WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'matches' AND COLUMN_NAME = 'match_mode';
SET @ddl = IF(@match_mode_exists > 0, 'ALTER TABLE `matches` DROP COLUMN `match_mode`', 'SELECT 1');
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;
