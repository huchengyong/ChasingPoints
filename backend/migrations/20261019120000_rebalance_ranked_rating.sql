-- +goose Up
-- 排位规则 V2：段位行记录规则版本与 V2 已计分场次；初始分表；赛季记录维护不同对手数与达标状态；
-- 赛季榜索引改为达标优先；rank_reset_ratio 取 1 表达不做赛季软重置。

SELECT COUNT(*) INTO @rating_version_exists
FROM information_schema.COLUMNS
WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'user_ranking' AND COLUMN_NAME = 'rating_version';
SET @ddl = IF(
  @rating_version_exists = 0,
  'ALTER TABLE `user_ranking` ADD COLUMN `rating_version` TINYINT NOT NULL DEFAULT 1 COMMENT ''排位规则版本: 1=旧规则 2=V2'' AFTER `max_streak`',
  'SELECT 1'
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SELECT COUNT(*) INTO @rated_matches_exists
FROM information_schema.COLUMNS
WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'user_ranking' AND COLUMN_NAME = 'rated_matches';
SET @ddl = IF(
  @rated_matches_exists = 0,
  'ALTER TABLE `user_ranking` ADD COLUMN `rated_matches` INT NOT NULL DEFAULT 0 COMMENT ''V2 已计分场次（含初始分重放场次）'' AFTER `rating_version`',
  'SELECT 1'
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SELECT COUNT(*) INTO @idx_rating_version_exists
FROM information_schema.STATISTICS
WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'user_ranking' AND INDEX_NAME = 'idx_user_ranking_rating_version';
SET @ddl = IF(
  @idx_rating_version_exists = 0,
  'ALTER TABLE `user_ranking` ADD KEY `idx_user_ranking_rating_version` (`rating_version`, `id`)',
  'SELECT 1'
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

CREATE TABLE IF NOT EXISTS `rank_rating_seeds` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `user_id` BIGINT UNSIGNED NOT NULL COMMENT '用户ID',
  `game_type` TINYINT NOT NULL COMMENT '球种 1=斯诺克 2=九球追分 3=中式八球 4=美式九球',
  `seed_score` INT NOT NULL COMMENT 'V2 初始排位分',
  `seed_matches` INT NOT NULL DEFAULT 0 COMMENT '重放中已计分场次',
  `cutoff_at` DATETIME NOT NULL COMMENT '参与重放的对局完成时间上限（不含）',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_rank_rating_seed_user_game` (`user_id`, `game_type`)
) ENGINE=InnoDB COMMENT='排位规则 V2 初始分';

SELECT COUNT(*) INTO @distinct_opponents_exists
FROM information_schema.COLUMNS
WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'season_records' AND COLUMN_NAME = 'distinct_opponents';
SET @ddl = IF(
  @distinct_opponents_exists = 0,
  'ALTER TABLE `season_records` ADD COLUMN `distinct_opponents` INT NOT NULL DEFAULT 0 COMMENT ''本季该球种不同对手数'' AFTER `wins`',
  'SELECT 1'
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SELECT COUNT(*) INTO @qualified_exists
FROM information_schema.COLUMNS
WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'season_records' AND COLUMN_NAME = 'qualified';
SET @ddl = IF(
  @qualified_exists = 0,
  'ALTER TABLE `season_records` ADD COLUMN `qualified` TINYINT NOT NULL DEFAULT 0 COMMENT ''是否达到 V2 赛季名次门槛'' AFTER `distinct_opponents`',
  'SELECT 1'
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SELECT COUNT(*) INTO @idx_qualified_rank_exists
FROM information_schema.STATISTICS
WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'season_records' AND INDEX_NAME = 'idx_season_game_qualified_rank';
SET @ddl = IF(
  @idx_qualified_rank_exists = 0,
  'ALTER TABLE `season_records` ADD KEY `idx_season_game_qualified_rank` (`season_id`, `game_type`, `qualified`, `end_rank_score`, `wins`, `id`)',
  'SELECT 1'
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SELECT COUNT(*) INTO @idx_season_game_rank_exists
FROM information_schema.STATISTICS
WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'season_records' AND INDEX_NAME = 'idx_season_game_rank';
SET @ddl = IF(
  @idx_season_game_rank_exists > 0,
  'ALTER TABLE `season_records` DROP INDEX `idx_season_game_rank`',
  'SELECT 1'
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

ALTER TABLE `seasons` ALTER COLUMN `rank_reset_ratio` SET DEFAULT 1.00;
UPDATE `seasons` SET `rank_reset_ratio` = 1.00 WHERE `rank_reset_ratio` <> 1.00;

-- +goose Down

UPDATE `seasons` SET `rank_reset_ratio` = 0.70 WHERE `rank_reset_ratio` = 1.00;
ALTER TABLE `seasons` ALTER COLUMN `rank_reset_ratio` SET DEFAULT 0.70;

SELECT COUNT(*) INTO @idx_season_game_rank_exists
FROM information_schema.STATISTICS
WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'season_records' AND INDEX_NAME = 'idx_season_game_rank';
SET @ddl = IF(
  @idx_season_game_rank_exists = 0,
  'ALTER TABLE `season_records` ADD KEY `idx_season_game_rank` (`season_id`, `game_type`, `end_rank_score`, `wins`, `id`)',
  'SELECT 1'
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SELECT COUNT(*) INTO @idx_qualified_rank_exists
FROM information_schema.STATISTICS
WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'season_records' AND INDEX_NAME = 'idx_season_game_qualified_rank';
SET @ddl = IF(
  @idx_qualified_rank_exists > 0,
  'ALTER TABLE `season_records` DROP INDEX `idx_season_game_qualified_rank`',
  'SELECT 1'
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SELECT COUNT(*) INTO @qualified_exists
FROM information_schema.COLUMNS
WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'season_records' AND COLUMN_NAME = 'qualified';
SET @ddl = IF(@qualified_exists > 0, 'ALTER TABLE `season_records` DROP COLUMN `qualified`', 'SELECT 1');
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SELECT COUNT(*) INTO @distinct_opponents_exists
FROM information_schema.COLUMNS
WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'season_records' AND COLUMN_NAME = 'distinct_opponents';
SET @ddl = IF(@distinct_opponents_exists > 0, 'ALTER TABLE `season_records` DROP COLUMN `distinct_opponents`', 'SELECT 1');
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

DROP TABLE IF EXISTS `rank_rating_seeds`;

SELECT COUNT(*) INTO @idx_rating_version_exists
FROM information_schema.STATISTICS
WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'user_ranking' AND INDEX_NAME = 'idx_user_ranking_rating_version';
SET @ddl = IF(@idx_rating_version_exists > 0, 'ALTER TABLE `user_ranking` DROP INDEX `idx_user_ranking_rating_version`', 'SELECT 1');
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SELECT COUNT(*) INTO @rated_matches_exists
FROM information_schema.COLUMNS
WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'user_ranking' AND COLUMN_NAME = 'rated_matches';
SET @ddl = IF(@rated_matches_exists > 0, 'ALTER TABLE `user_ranking` DROP COLUMN `rated_matches`', 'SELECT 1');
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SELECT COUNT(*) INTO @rating_version_exists
FROM information_schema.COLUMNS
WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'user_ranking' AND COLUMN_NAME = 'rating_version';
SET @ddl = IF(@rating_version_exists > 0, 'ALTER TABLE `user_ranking` DROP COLUMN `rating_version`', 'SELECT 1');
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;
