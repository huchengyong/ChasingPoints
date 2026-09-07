-- +goose Up
-- 扩大 flag_emoji 列以容纳 tag sequence 国旗 emoji（如英格兰 🏴󠁧󠁢󠁥󠁮󠁧󠁿 最长约 28 字节）

SET @col_exists = (
  SELECT COUNT(*)
  FROM information_schema.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'players' AND COLUMN_NAME = 'flag_emoji' AND CHARACTER_MAXIMUM_LENGTH < 32
);
SET @ddl = IF(
  @col_exists = 1,
  'ALTER TABLE `players` MODIFY COLUMN `flag_emoji` VARCHAR(32) NOT NULL DEFAULT '''' COMMENT ''国旗emoji''',
  'SELECT 1'
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

-- +goose Down
SET @col_exists = (
  SELECT COUNT(*)
  FROM information_schema.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'players' AND COLUMN_NAME = 'flag_emoji' AND CHARACTER_MAXIMUM_LENGTH = 32
);
SET @ddl = IF(
  @col_exists = 1,
  'ALTER TABLE `players` MODIFY COLUMN `flag_emoji` VARCHAR(16) NOT NULL DEFAULT '''' COMMENT ''国旗emoji''',
  'SELECT 1'
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;