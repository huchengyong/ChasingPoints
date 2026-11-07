-- +goose Up

-- 每段 DDL 都先检查索引，兼容已经由人工运维补过索引的旧库。
SET @idx_exists = (SELECT COUNT(*) FROM information_schema.STATISTICS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'friend_requests' AND INDEX_NAME = 'idx_friend_requests_to_status_id');
SET @ddl = IF(@idx_exists = 0, 'ALTER TABLE `friend_requests` ADD KEY `idx_friend_requests_to_status_id` (`to_user_id`, `status`, `id`)', 'SELECT 1');
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @idx_exists = (SELECT COUNT(*) FROM information_schema.STATISTICS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'challenges' AND INDEX_NAME = 'idx_challenges_status_expires_id');
SET @ddl = IF(@idx_exists = 0, 'ALTER TABLE `challenges` ADD KEY `idx_challenges_status_expires_id` (`status`, `expires_at`, `id`)', 'SELECT 1');
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @idx_exists = (SELECT COUNT(*) FROM information_schema.STATISTICS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'challenges' AND INDEX_NAME = 'idx_challenges_to_status_expires_id');
SET @ddl = IF(@idx_exists = 0, 'ALTER TABLE `challenges` ADD KEY `idx_challenges_to_status_expires_id` (`to_user_id`, `status`, `expires_at`, `id`)', 'SELECT 1');
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @idx_exists = (SELECT COUNT(*) FROM information_schema.STATISTICS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'seasons' AND INDEX_NAME = 'idx_seasons_end_id_status_start');
SET @ddl = IF(@idx_exists = 0, 'ALTER TABLE `seasons` ADD KEY `idx_seasons_end_id_status_start` (`end_date`, `id`, `status`, `start_date`)', 'SELECT 1');
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @idx_exists = (SELECT COUNT(*) FROM information_schema.STATISTICS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'season_settlements' AND INDEX_NAME = 'idx_season_settlements_status_completed_id');
SET @ddl = IF(@idx_exists = 0, 'ALTER TABLE `season_settlements` ADD KEY `idx_season_settlements_status_completed_id` (`status`, `completed_at`, `id`)', 'SELECT 1');
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @idx_exists = (SELECT COUNT(*) FROM information_schema.STATISTICS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'achievement_progress_events' AND INDEX_NAME = 'idx_achievement_events_user_game_metric');
SET @ddl = IF(@idx_exists = 0, 'ALTER TABLE `achievement_progress_events` ADD KEY `idx_achievement_events_user_game_metric` (`user_id`, `game_type`, `metric_key`, `created_at`, `id`)', 'SELECT 1');
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @idx_exists = (SELECT COUNT(*) FROM information_schema.STATISTICS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'venues' AND INDEX_NAME = 'idx_venues_nearby_candidates');
SET @ddl = IF(@idx_exists = 0, 'ALTER TABLE `venues` ADD KEY `idx_venues_nearby_candidates` (`status`, `geo_status`, `duplicate_of_venue_id`, `latitude`, `longitude`, `id`)', 'SELECT 1');
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @idx_exists = (SELECT COUNT(*) FROM information_schema.STATISTICS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'venues' AND INDEX_NAME = 'idx_status_geo_status');
SET @ddl = IF(@idx_exists > 0, 'ALTER TABLE `venues` DROP INDEX `idx_status_geo_status`', 'SELECT 1');
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @idx_exists = (SELECT COUNT(*) FROM information_schema.STATISTICS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'matches' AND INDEX_NAME = 'idx_matches_referee_history');
SET @ddl = IF(@idx_exists = 0, 'ALTER TABLE `matches` ADD KEY `idx_matches_referee_history` (`referee_user_id`, `deleted_at`, `end_time`, `id`, `status`)', 'SELECT 1');
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @idx_exists = (SELECT COUNT(*) FROM information_schema.STATISTICS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'matches' AND INDEX_NAME = 'idx_matches_public_status_time');
SET @ddl = IF(@idx_exists = 0, 'ALTER TABLE `matches` ADD KEY `idx_matches_public_status_time` (`visibility`, `status`, `match_time`, `id`)', 'SELECT 1');
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

-- +goose Down

SET @idx_exists = (SELECT COUNT(*) FROM information_schema.STATISTICS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'matches' AND INDEX_NAME = 'idx_matches_public_status_time');
SET @ddl = IF(@idx_exists > 0, 'ALTER TABLE `matches` DROP INDEX `idx_matches_public_status_time`', 'SELECT 1');
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @idx_exists = (SELECT COUNT(*) FROM information_schema.STATISTICS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'matches' AND INDEX_NAME = 'idx_matches_referee_history');
SET @ddl = IF(@idx_exists > 0, 'ALTER TABLE `matches` DROP INDEX `idx_matches_referee_history`', 'SELECT 1');
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @idx_exists = (SELECT COUNT(*) FROM information_schema.STATISTICS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'venues' AND INDEX_NAME = 'idx_venues_nearby_candidates');
SET @ddl = IF(@idx_exists > 0, 'ALTER TABLE `venues` DROP INDEX `idx_venues_nearby_candidates`', 'SELECT 1');
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @idx_exists = (SELECT COUNT(*) FROM information_schema.STATISTICS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'venues' AND INDEX_NAME = 'idx_status_geo_status');
SET @ddl = IF(@idx_exists = 0, 'ALTER TABLE `venues` ADD KEY `idx_status_geo_status` (`status`, `geo_status`)', 'SELECT 1');
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @idx_exists = (SELECT COUNT(*) FROM information_schema.STATISTICS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'achievement_progress_events' AND INDEX_NAME = 'idx_achievement_events_user_game_metric');
SET @ddl = IF(@idx_exists > 0, 'ALTER TABLE `achievement_progress_events` DROP INDEX `idx_achievement_events_user_game_metric`', 'SELECT 1');
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @idx_exists = (SELECT COUNT(*) FROM information_schema.STATISTICS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'season_settlements' AND INDEX_NAME = 'idx_season_settlements_status_completed_id');
SET @ddl = IF(@idx_exists > 0, 'ALTER TABLE `season_settlements` DROP INDEX `idx_season_settlements_status_completed_id`', 'SELECT 1');
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @idx_exists = (SELECT COUNT(*) FROM information_schema.STATISTICS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'seasons' AND INDEX_NAME = 'idx_seasons_end_id_status_start');
SET @ddl = IF(@idx_exists > 0, 'ALTER TABLE `seasons` DROP INDEX `idx_seasons_end_id_status_start`', 'SELECT 1');
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @idx_exists = (SELECT COUNT(*) FROM information_schema.STATISTICS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'challenges' AND INDEX_NAME = 'idx_challenges_to_status_expires_id');
SET @ddl = IF(@idx_exists > 0, 'ALTER TABLE `challenges` DROP INDEX `idx_challenges_to_status_expires_id`', 'SELECT 1');
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @idx_exists = (SELECT COUNT(*) FROM information_schema.STATISTICS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'challenges' AND INDEX_NAME = 'idx_challenges_status_expires_id');
SET @ddl = IF(@idx_exists > 0, 'ALTER TABLE `challenges` DROP INDEX `idx_challenges_status_expires_id`', 'SELECT 1');
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @idx_exists = (SELECT COUNT(*) FROM information_schema.STATISTICS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'friend_requests' AND INDEX_NAME = 'idx_friend_requests_to_status_id');
SET @ddl = IF(@idx_exists > 0, 'ALTER TABLE `friend_requests` DROP INDEX `idx_friend_requests_to_status_id`', 'SELECT 1');
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;
