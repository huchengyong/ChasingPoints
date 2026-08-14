-- +goose Up

SET @idx_exists = (
  SELECT COUNT(*)
  FROM information_schema.STATISTICS
  WHERE TABLE_SCHEMA = DATABASE()
    AND TABLE_NAME = 'matches'
    AND INDEX_NAME = 'idx_matches_user_game_status_mode_score'
);
SET @ddl = IF(
  @idx_exists = 0,
  'ALTER TABLE `matches` ADD KEY `idx_matches_user_game_status_mode_score` (`user_id`, `game_type`, `status`, `match_mode`, `my_score`, `completed_at`, `id`)',
  'SELECT 1'
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @idx_exists = (
  SELECT COUNT(*)
  FROM information_schema.STATISTICS
  WHERE TABLE_SCHEMA = DATABASE()
    AND TABLE_NAME = 'matches'
    AND INDEX_NAME = 'idx_matches_opponent_game_status_mode_score'
);
SET @ddl = IF(
  @idx_exists = 0,
  'ALTER TABLE `matches` ADD KEY `idx_matches_opponent_game_status_mode_score` (`opponent_id`, `game_type`, `status`, `match_mode`, `opponent_score`, `completed_at`, `id`)',
  'SELECT 1'
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

-- +goose Down

SET @idx_exists = (
  SELECT COUNT(*)
  FROM information_schema.STATISTICS
  WHERE TABLE_SCHEMA = DATABASE()
    AND TABLE_NAME = 'matches'
    AND INDEX_NAME = 'idx_matches_opponent_game_status_mode_score'
);
SET @ddl = IF(
  @idx_exists > 0,
  'ALTER TABLE `matches` DROP INDEX `idx_matches_opponent_game_status_mode_score`',
  'SELECT 1'
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @idx_exists = (
  SELECT COUNT(*)
  FROM information_schema.STATISTICS
  WHERE TABLE_SCHEMA = DATABASE()
    AND TABLE_NAME = 'matches'
    AND INDEX_NAME = 'idx_matches_user_game_status_mode_score'
);
SET @ddl = IF(
  @idx_exists > 0,
  'ALTER TABLE `matches` DROP INDEX `idx_matches_user_game_status_mode_score`',
  'SELECT 1'
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;
