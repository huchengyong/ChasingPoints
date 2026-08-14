-- +goose Up

SET @idx_exists = (SELECT COUNT(*) FROM information_schema.STATISTICS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'matches' AND INDEX_NAME = 'idx_matches_finish_request_expiry');
SET @ddl = IF(@idx_exists = 0, 'ALTER TABLE `matches` ADD KEY `idx_matches_finish_request_expiry` (`status`, `finish_state`, `finish_requested_at`, `id`)', 'SELECT 1');
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

-- +goose Down

SET @idx_exists = (SELECT COUNT(*) FROM information_schema.STATISTICS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'matches' AND INDEX_NAME = 'idx_matches_finish_request_expiry');
SET @ddl = IF(@idx_exists > 0, 'ALTER TABLE `matches` DROP INDEX `idx_matches_finish_request_expiry`', 'SELECT 1');
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;
