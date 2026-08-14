-- +goose Up

SET @idx_exists = (SELECT COUNT(*) FROM information_schema.STATISTICS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'matches' AND INDEX_NAME = 'idx_matches_achievement_sync_pending');
SET @ddl = IF(@idx_exists = 0, 'ALTER TABLE `matches` ADD KEY `idx_matches_achievement_sync_pending` (`status`, `achievement_synced_at`, `id`)', 'SELECT 1');
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

-- +goose Down

SET @idx_exists = (SELECT COUNT(*) FROM information_schema.STATISTICS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'matches' AND INDEX_NAME = 'idx_matches_achievement_sync_pending');
SET @ddl = IF(@idx_exists > 0, 'ALTER TABLE `matches` DROP INDEX `idx_matches_achievement_sync_pending`', 'SELECT 1');
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;
