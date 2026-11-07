-- +goose Up

SET @idx_exists = (SELECT COUNT(*) FROM information_schema.STATISTICS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'achievement_progress_events' AND INDEX_NAME = 'idx_achievement_events_season_archive');
SET @ddl = IF(@idx_exists = 0, 'ALTER TABLE `achievement_progress_events` ADD KEY `idx_achievement_events_season_archive` (`occurred_at`, `metric_key`, `user_id`, `game_type`, `id`)', 'SELECT 1');
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

-- +goose Down

SET @idx_exists = (SELECT COUNT(*) FROM information_schema.STATISTICS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'achievement_progress_events' AND INDEX_NAME = 'idx_achievement_events_season_archive');
SET @ddl = IF(@idx_exists > 0, 'ALTER TABLE `achievement_progress_events` DROP INDEX `idx_achievement_events_season_archive`', 'SELECT 1');
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;
