-- +goose Up
SET @ranking_has_game_type := (
  SELECT COUNT(*)
  FROM information_schema.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE()
    AND TABLE_NAME = 'user_ranking'
    AND COLUMN_NAME = 'game_type'
);

SET @ranking_add_game_type_sql := IF(
  @ranking_has_game_type = 0,
  'ALTER TABLE `user_ranking` ADD COLUMN `game_type` TINYINT NOT NULL DEFAULT 3 COMMENT ''球种 1=斯诺克 2=九球追分 3=中式八球 4=美式九球'' AFTER `user_id`',
  'SELECT 1'
);
PREPARE stmt FROM @ranking_add_game_type_sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @ranking_has_unique_index := (
  SELECT COUNT(*)
  FROM information_schema.STATISTICS
  WHERE TABLE_SCHEMA = DATABASE()
    AND TABLE_NAME = 'user_ranking'
    AND INDEX_NAME = 'uniq_user_game_type'
);

SET @ranking_add_unique_index_sql := IF(
  @ranking_has_unique_index = 0,
  'ALTER TABLE `user_ranking` ADD UNIQUE KEY `uniq_user_game_type` (`user_id`, `game_type`)',
  'SELECT 1'
);
PREPARE stmt FROM @ranking_add_unique_index_sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @ranking_has_rank_index := (
  SELECT COUNT(*)
  FROM information_schema.STATISTICS
  WHERE TABLE_SCHEMA = DATABASE()
    AND TABLE_NAME = 'user_ranking'
    AND INDEX_NAME = 'idx_game_type_rank_score'
);

SET @ranking_add_rank_index_sql := IF(
  @ranking_has_rank_index = 0,
  'ALTER TABLE `user_ranking` ADD KEY `idx_game_type_rank_score` (`game_type`, `rank_score`, `total_wins`, `updated_at`, `user_id`)',
  'SELECT 1'
);
PREPARE stmt FROM @ranking_add_rank_index_sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @ranking_has_old_user_index := (
  SELECT COUNT(*)
  FROM information_schema.STATISTICS
  WHERE TABLE_SCHEMA = DATABASE()
    AND TABLE_NAME = 'user_ranking'
    AND INDEX_NAME = 'idx_user_id'
);

SET @ranking_drop_old_user_index_sql := IF(
  @ranking_has_old_user_index > 0,
  'ALTER TABLE `user_ranking` DROP INDEX `idx_user_id`',
  'SELECT 1'
);
PREPARE stmt FROM @ranking_drop_old_user_index_sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @rank_logs_has_game_type := (
  SELECT COUNT(*)
  FROM information_schema.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE()
    AND TABLE_NAME = 'rank_change_logs'
    AND COLUMN_NAME = 'game_type'
);

SET @rank_logs_add_game_type_sql := IF(
  @rank_logs_has_game_type = 0,
  'ALTER TABLE `rank_change_logs` ADD COLUMN `game_type` TINYINT NOT NULL DEFAULT 3 COMMENT ''球种 1=斯诺克 2=九球追分 3=中式八球 4=美式九球'' AFTER `change_type`',
  'SELECT 1'
);
PREPARE stmt FROM @rank_logs_add_game_type_sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

UPDATE `rank_change_logs` rcl
LEFT JOIN `matches` m ON m.id = rcl.match_id
SET rcl.game_type = COALESCE(m.game_type, 3);

SET @rank_logs_has_unique_index := (
  SELECT COUNT(*)
  FROM information_schema.STATISTICS
  WHERE TABLE_SCHEMA = DATABASE()
    AND TABLE_NAME = 'rank_change_logs'
    AND INDEX_NAME = 'uniq_user_match_type_game'
);

SET @rank_logs_add_unique_index_sql := IF(
  @rank_logs_has_unique_index = 0,
  'ALTER TABLE `rank_change_logs` ADD UNIQUE KEY `uniq_user_match_type_game` (`user_id`, `match_id`, `change_type`, `game_type`)',
  'SELECT 1'
);
PREPARE stmt FROM @rank_logs_add_unique_index_sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @rank_logs_has_effective_index := (
  SELECT COUNT(*)
  FROM information_schema.STATISTICS
  WHERE TABLE_SCHEMA = DATABASE()
    AND TABLE_NAME = 'rank_change_logs'
    AND INDEX_NAME = 'idx_user_game_effective_at'
);

SET @rank_logs_add_effective_index_sql := IF(
  @rank_logs_has_effective_index = 0,
  'ALTER TABLE `rank_change_logs` ADD KEY `idx_user_game_effective_at` (`user_id`, `game_type`, `effective_at`)',
  'SELECT 1'
);
PREPARE stmt FROM @rank_logs_add_effective_index_sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @rank_logs_has_old_unique_index := (
  SELECT COUNT(*)
  FROM information_schema.STATISTICS
  WHERE TABLE_SCHEMA = DATABASE()
    AND TABLE_NAME = 'rank_change_logs'
    AND INDEX_NAME = 'uniq_user_match_type'
);

SET @rank_logs_drop_old_unique_index_sql := IF(
  @rank_logs_has_old_unique_index > 0,
  'ALTER TABLE `rank_change_logs` DROP INDEX `uniq_user_match_type`',
  'SELECT 1'
);
PREPARE stmt FROM @rank_logs_drop_old_unique_index_sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @season_has_game_type := (
  SELECT COUNT(*)
  FROM information_schema.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE()
    AND TABLE_NAME = 'season_records'
    AND COLUMN_NAME = 'game_type'
);

SET @season_add_game_type_sql := IF(
  @season_has_game_type = 0,
  'ALTER TABLE `season_records` ADD COLUMN `game_type` TINYINT NOT NULL DEFAULT 3 COMMENT ''球种 1=斯诺克 2=九球追分 3=中式八球 4=美式九球'' AFTER `user_id`',
  'SELECT 1'
);
PREPARE stmt FROM @season_add_game_type_sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @season_has_user_game_index := (
  SELECT COUNT(*)
  FROM information_schema.STATISTICS
  WHERE TABLE_SCHEMA = DATABASE()
    AND TABLE_NAME = 'season_records'
    AND INDEX_NAME = 'uk_season_user_game'
);

SET @season_add_user_game_index_sql := IF(
  @season_has_user_game_index = 0,
  'ALTER TABLE `season_records` ADD UNIQUE KEY `uk_season_user_game` (`season_id`, `user_id`, `game_type`)',
  'SELECT 1'
);
PREPARE stmt FROM @season_add_user_game_index_sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @season_has_rank_index := (
  SELECT COUNT(*)
  FROM information_schema.STATISTICS
  WHERE TABLE_SCHEMA = DATABASE()
    AND TABLE_NAME = 'season_records'
    AND INDEX_NAME = 'idx_season_game_rank'
);

SET @season_add_rank_index_sql := IF(
  @season_has_rank_index = 0,
  'ALTER TABLE `season_records` ADD KEY `idx_season_game_rank` (`season_id`, `game_type`, `end_rank_score`, `wins`, `id`)',
  'SELECT 1'
);
PREPARE stmt FROM @season_add_rank_index_sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @season_has_old_user_index := (
  SELECT COUNT(*)
  FROM information_schema.STATISTICS
  WHERE TABLE_SCHEMA = DATABASE()
    AND TABLE_NAME = 'season_records'
    AND INDEX_NAME = 'uk_season_user'
);

SET @season_drop_old_user_index_sql := IF(
  @season_has_old_user_index > 0,
  'ALTER TABLE `season_records` DROP INDEX `uk_season_user`',
  'SELECT 1'
);
PREPARE stmt FROM @season_drop_old_user_index_sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

-- +goose Down
ALTER TABLE `season_records`
  DROP INDEX `idx_season_game_rank`,
  DROP INDEX `uk_season_user_game`,
  ADD UNIQUE KEY `uk_season_user` (`season_id`, `user_id`),
  DROP COLUMN `game_type`;

ALTER TABLE `rank_change_logs`
  DROP INDEX `idx_user_game_effective_at`,
  DROP INDEX `uniq_user_match_type_game`,
  ADD UNIQUE KEY `uniq_user_match_type` (`user_id`, `match_id`, `change_type`),
  DROP COLUMN `game_type`;

ALTER TABLE `user_ranking`
  DROP INDEX `idx_game_type_rank_score`,
  DROP INDEX `uniq_user_game_type`,
  ADD UNIQUE KEY `idx_user_id` (`user_id`),
  DROP COLUMN `game_type`;
