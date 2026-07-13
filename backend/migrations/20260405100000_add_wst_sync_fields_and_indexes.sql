-- +goose Up
-- 为 WST 官方同步补齐赛事来源字段与官方唯一索引

SET @col_exists = (
  SELECT COUNT(*)
  FROM information_schema.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'tournaments' AND COLUMN_NAME = 'source_type'
);
SET @ddl = IF(
  @col_exists = 0,
  'ALTER TABLE `tournaments` ADD COLUMN `source_type` VARCHAR(32) NOT NULL DEFAULT '''' COMMENT ''来源类型 manual/official/imported'' AFTER `end_time`',
  'SELECT 1'
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @col_exists = (
  SELECT COUNT(*)
  FROM information_schema.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'tournaments' AND COLUMN_NAME = 'source_tournament_id'
);
SET @ddl = IF(
  @col_exists = 0,
  'ALTER TABLE `tournaments` ADD COLUMN `source_tournament_id` VARCHAR(128) NOT NULL DEFAULT '''' COMMENT ''官方赛事ID'' AFTER `source_type`',
  'SELECT 1'
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @col_exists = (
  SELECT COUNT(*)
  FROM information_schema.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'tournaments' AND COLUMN_NAME = 'source_season_id'
);
SET @ddl = IF(
  @col_exists = 0,
  'ALTER TABLE `tournaments` ADD COLUMN `source_season_id` VARCHAR(64) NOT NULL DEFAULT '''' COMMENT ''官方赛季ID'' AFTER `source_tournament_id`',
  'SELECT 1'
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @col_exists = (
  SELECT COUNT(*)
  FROM information_schema.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'tournaments' AND COLUMN_NAME = 'information_page'
);
SET @ddl = IF(
  @col_exists = 0,
  'ALTER TABLE `tournaments` ADD COLUMN `information_page` VARCHAR(512) NOT NULL DEFAULT '''' COMMENT ''官方信息页'' AFTER `source_season_id`',
  'SELECT 1'
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @col_exists = (
  SELECT COUNT(*)
  FROM information_schema.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'tournaments' AND COLUMN_NAME = 'ticketing_link'
);
SET @ddl = IF(
  @col_exists = 0,
  'ALTER TABLE `tournaments` ADD COLUMN `ticketing_link` VARCHAR(512) NOT NULL DEFAULT '''' COMMENT ''官方购票链接'' AFTER `information_page`',
  'SELECT 1'
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @col_exists = (
  SELECT COUNT(*)
  FROM information_schema.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'tournaments' AND COLUMN_NAME = 'last_synced_at'
);
SET @ddl = IF(
  @col_exists = 0,
  'ALTER TABLE `tournaments` ADD COLUMN `last_synced_at` DATETIME DEFAULT NULL COMMENT ''最近同步时间'' AFTER `ticketing_link`',
  'SELECT 1'
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @col_exists = (
  SELECT COUNT(*)
  FROM information_schema.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'tournaments' AND COLUMN_NAME = 'official_source_tournament_id'
);
SET @ddl = IF(
  @col_exists = 0,
  'ALTER TABLE `tournaments` ADD COLUMN `official_source_tournament_id` VARCHAR(128) GENERATED ALWAYS AS (CASE WHEN `source_type` = ''official'' AND `source_tournament_id` <> '''' THEN `source_tournament_id` ELSE NULL END) STORED',
  'SELECT 1'
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @idx_exists = (
  SELECT COUNT(*)
  FROM information_schema.STATISTICS
  WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'tournaments' AND INDEX_NAME = 'idx_source_season_id'
);
SET @ddl = IF(
  @idx_exists = 0,
  'ALTER TABLE `tournaments` ADD KEY `idx_source_season_id` (`source_season_id`)',
  'SELECT 1'
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @idx_exists = (
  SELECT COUNT(*)
  FROM information_schema.STATISTICS
  WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'tournaments' AND INDEX_NAME = 'idx_last_synced_at'
);
SET @ddl = IF(
  @idx_exists = 0,
  'ALTER TABLE `tournaments` ADD KEY `idx_last_synced_at` (`last_synced_at`)',
  'SELECT 1'
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @idx_exists = (
  SELECT COUNT(*)
  FROM information_schema.STATISTICS
  WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'tournaments' AND INDEX_NAME = 'uk_tournaments_official_source_tournament_id'
);
SET @ddl = IF(
  @idx_exists = 0,
  'ALTER TABLE `tournaments` ADD UNIQUE KEY `uk_tournaments_official_source_tournament_id` (`official_source_tournament_id`)',
  'SELECT 1'
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @col_exists = (
  SELECT COUNT(*)
  FROM information_schema.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'players' AND COLUMN_NAME = 'official_source_player_id'
);
SET @ddl = IF(
  @col_exists = 0,
  'ALTER TABLE `players` ADD COLUMN `official_source_player_id` VARCHAR(128) GENERATED ALWAYS AS (CASE WHEN `source_type` = ''official'' AND `source_player_id` <> '''' THEN `source_player_id` ELSE NULL END) STORED',
  'SELECT 1'
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @idx_exists = (
  SELECT COUNT(*)
  FROM information_schema.STATISTICS
  WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'players' AND INDEX_NAME = 'uk_players_official_source_player_id'
);
SET @ddl = IF(
  @idx_exists = 0,
  'ALTER TABLE `players` ADD UNIQUE KEY `uk_players_official_source_player_id` (`official_source_player_id`)',
  'SELECT 1'
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @col_exists = (
  SELECT COUNT(*)
  FROM information_schema.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'tournament_matches' AND COLUMN_NAME = 'official_source_match_id'
);
SET @ddl = IF(
  @col_exists = 0,
  'ALTER TABLE `tournament_matches` ADD COLUMN `official_source_match_id` VARCHAR(128) GENERATED ALWAYS AS (CASE WHEN `source_type` = ''official'' AND `source_match_id` <> '''' THEN `source_match_id` ELSE NULL END) STORED',
  'SELECT 1'
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @idx_exists = (
  SELECT COUNT(*)
  FROM information_schema.STATISTICS
  WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'tournament_matches' AND INDEX_NAME = 'uk_tournament_matches_official_source_match_id'
);
SET @ddl = IF(
  @idx_exists = 0,
  'ALTER TABLE `tournament_matches` ADD UNIQUE KEY `uk_tournament_matches_official_source_match_id` (`official_source_match_id`)',
  'SELECT 1'
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

-- +goose Down

SET @idx_exists = (
  SELECT COUNT(*)
  FROM information_schema.STATISTICS
  WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'tournament_matches' AND INDEX_NAME = 'uk_tournament_matches_official_source_match_id'
);
SET @ddl = IF(
  @idx_exists = 1,
  'ALTER TABLE `tournament_matches` DROP INDEX `uk_tournament_matches_official_source_match_id`',
  'SELECT 1'
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @col_exists = (
  SELECT COUNT(*)
  FROM information_schema.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'tournament_matches' AND COLUMN_NAME = 'official_source_match_id'
);
SET @ddl = IF(
  @col_exists = 1,
  'ALTER TABLE `tournament_matches` DROP COLUMN `official_source_match_id`',
  'SELECT 1'
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @idx_exists = (
  SELECT COUNT(*)
  FROM information_schema.STATISTICS
  WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'players' AND INDEX_NAME = 'uk_players_official_source_player_id'
);
SET @ddl = IF(
  @idx_exists = 1,
  'ALTER TABLE `players` DROP INDEX `uk_players_official_source_player_id`',
  'SELECT 1'
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @col_exists = (
  SELECT COUNT(*)
  FROM information_schema.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'players' AND COLUMN_NAME = 'official_source_player_id'
);
SET @ddl = IF(
  @col_exists = 1,
  'ALTER TABLE `players` DROP COLUMN `official_source_player_id`',
  'SELECT 1'
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @idx_exists = (
  SELECT COUNT(*)
  FROM information_schema.STATISTICS
  WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'tournaments' AND INDEX_NAME = 'uk_tournaments_official_source_tournament_id'
);
SET @ddl = IF(
  @idx_exists = 1,
  'ALTER TABLE `tournaments` DROP INDEX `uk_tournaments_official_source_tournament_id`',
  'SELECT 1'
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @idx_exists = (
  SELECT COUNT(*)
  FROM information_schema.STATISTICS
  WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'tournaments' AND INDEX_NAME = 'idx_last_synced_at'
);
SET @ddl = IF(
  @idx_exists = 1,
  'ALTER TABLE `tournaments` DROP INDEX `idx_last_synced_at`',
  'SELECT 1'
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @idx_exists = (
  SELECT COUNT(*)
  FROM information_schema.STATISTICS
  WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'tournaments' AND INDEX_NAME = 'idx_source_season_id'
);
SET @ddl = IF(
  @idx_exists = 1,
  'ALTER TABLE `tournaments` DROP INDEX `idx_source_season_id`',
  'SELECT 1'
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @col_exists = (
  SELECT COUNT(*)
  FROM information_schema.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'tournaments' AND COLUMN_NAME = 'official_source_tournament_id'
);
SET @ddl = IF(
  @col_exists = 1,
  'ALTER TABLE `tournaments` DROP COLUMN `official_source_tournament_id`',
  'SELECT 1'
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @col_exists = (
  SELECT COUNT(*)
  FROM information_schema.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'tournaments' AND COLUMN_NAME = 'last_synced_at'
);
SET @ddl = IF(
  @col_exists = 1,
  'ALTER TABLE `tournaments` DROP COLUMN `last_synced_at`',
  'SELECT 1'
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @col_exists = (
  SELECT COUNT(*)
  FROM information_schema.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'tournaments' AND COLUMN_NAME = 'ticketing_link'
);
SET @ddl = IF(
  @col_exists = 1,
  'ALTER TABLE `tournaments` DROP COLUMN `ticketing_link`',
  'SELECT 1'
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @col_exists = (
  SELECT COUNT(*)
  FROM information_schema.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'tournaments' AND COLUMN_NAME = 'information_page'
);
SET @ddl = IF(
  @col_exists = 1,
  'ALTER TABLE `tournaments` DROP COLUMN `information_page`',
  'SELECT 1'
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @col_exists = (
  SELECT COUNT(*)
  FROM information_schema.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'tournaments' AND COLUMN_NAME = 'source_season_id'
);
SET @ddl = IF(
  @col_exists = 1,
  'ALTER TABLE `tournaments` DROP COLUMN `source_season_id`',
  'SELECT 1'
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @col_exists = (
  SELECT COUNT(*)
  FROM information_schema.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'tournaments' AND COLUMN_NAME = 'source_tournament_id'
);
SET @ddl = IF(
  @col_exists = 1,
  'ALTER TABLE `tournaments` DROP COLUMN `source_tournament_id`',
  'SELECT 1'
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @col_exists = (
  SELECT COUNT(*)
  FROM information_schema.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'tournaments' AND COLUMN_NAME = 'source_type'
);
SET @ddl = IF(
  @col_exists = 1,
  'ALTER TABLE `tournaments` DROP COLUMN `source_type`',
  'SELECT 1'
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;
