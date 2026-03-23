-- +goose Up
SET @venues_has_full_address_unique_index := (
  SELECT COUNT(*)
  FROM information_schema.STATISTICS
  WHERE TABLE_SCHEMA = DATABASE()
    AND TABLE_NAME = 'venues'
    AND INDEX_NAME = 'uniq_full_address'
);

SET @venues_add_full_address_unique_index_sql := IF(
  @venues_has_full_address_unique_index = 0,
  'ALTER TABLE `venues` ADD UNIQUE KEY `uniq_full_address` (`full_address`)',
  'SELECT 1'
);
PREPARE stmt FROM @venues_add_full_address_unique_index_sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

-- +goose Down
ALTER TABLE `venues`
  DROP INDEX `uniq_full_address`;
