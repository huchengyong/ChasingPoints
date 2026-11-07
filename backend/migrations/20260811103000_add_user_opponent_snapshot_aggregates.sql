-- +goose Up

SET @column_exists = (
  SELECT COUNT(*) FROM information_schema.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'user_opponent_stats' AND COLUMN_NAME = 'opponent_name'
);
SET @ddl = IF(@column_exists = 0,
  'ALTER TABLE `user_opponent_stats` ADD COLUMN `opponent_name` VARCHAR(128) NOT NULL DEFAULT '''' AFTER `opponent_name_key`',
  'SELECT 1'
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @column_exists = (
  SELECT COUNT(*) FROM information_schema.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'user_opponent_stats' AND COLUMN_NAME = 'opponent_avatar'
);
SET @ddl = IF(@column_exists = 0,
  'ALTER TABLE `user_opponent_stats` ADD COLUMN `opponent_avatar` VARCHAR(512) NOT NULL DEFAULT '''' AFTER `opponent_name`',
  'SELECT 1'
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @column_exists = (
  SELECT COUNT(*) FROM information_schema.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'user_opponent_stats' AND COLUMN_NAME = 'score_diff_sum'
);
SET @ddl = IF(@column_exists = 0,
  'ALTER TABLE `user_opponent_stats` ADD COLUMN `score_diff_sum` BIGINT NOT NULL DEFAULT 0 AFTER `draws`',
  'SELECT 1'
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @column_exists = (
  SELECT COUNT(*) FROM information_schema.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'user_opponent_stats' AND COLUMN_NAME = 'current_win_streak'
);
SET @ddl = IF(@column_exists = 0,
  'ALTER TABLE `user_opponent_stats` ADD COLUMN `current_win_streak` INT NOT NULL DEFAULT 0 AFTER `score_diff_sum`',
  'SELECT 1'
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @column_exists = (
  SELECT COUNT(*) FROM information_schema.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'user_opponent_stats' AND COLUMN_NAME = 'max_win_streak'
);
SET @ddl = IF(@column_exists = 0,
  'ALTER TABLE `user_opponent_stats` ADD COLUMN `max_win_streak` INT NOT NULL DEFAULT 0 AFTER `current_win_streak`',
  'SELECT 1'
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

-- +goose Down

SET @column_exists = (
  SELECT COUNT(*) FROM information_schema.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'user_opponent_stats' AND COLUMN_NAME = 'max_win_streak'
);
SET @ddl = IF(@column_exists = 1,
  'ALTER TABLE `user_opponent_stats` DROP COLUMN `max_win_streak`',
  'SELECT 1'
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @column_exists = (
  SELECT COUNT(*) FROM information_schema.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'user_opponent_stats' AND COLUMN_NAME = 'current_win_streak'
);
SET @ddl = IF(@column_exists = 1,
  'ALTER TABLE `user_opponent_stats` DROP COLUMN `current_win_streak`',
  'SELECT 1'
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @column_exists = (
  SELECT COUNT(*) FROM information_schema.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'user_opponent_stats' AND COLUMN_NAME = 'score_diff_sum'
);
SET @ddl = IF(@column_exists = 1,
  'ALTER TABLE `user_opponent_stats` DROP COLUMN `score_diff_sum`',
  'SELECT 1'
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @column_exists = (
  SELECT COUNT(*) FROM information_schema.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'user_opponent_stats' AND COLUMN_NAME = 'opponent_avatar'
);
SET @ddl = IF(@column_exists = 1,
  'ALTER TABLE `user_opponent_stats` DROP COLUMN `opponent_avatar`',
  'SELECT 1'
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @column_exists = (
  SELECT COUNT(*) FROM information_schema.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'user_opponent_stats' AND COLUMN_NAME = 'opponent_name'
);
SET @ddl = IF(@column_exists = 1,
  'ALTER TABLE `user_opponent_stats` DROP COLUMN `opponent_name`',
  'SELECT 1'
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;
