-- +goose Up
-- Ensure one provider identity resolves to only one local account before verified OAuth login.

SELECT COUNT(*) INTO @user_oauth_duplicate_provider_open_id_count
FROM (
  SELECT `provider`, `open_id`
  FROM `user_oauth`
  GROUP BY `provider`, `open_id`
  HAVING COUNT(*) > 1
) AS duplicate_user_oauth_provider_open_id;

SET @ddl = IF(
  @user_oauth_duplicate_provider_open_id_count > 0,
  'SIGNAL SQLSTATE ''45000'' SET MESSAGE_TEXT = ''duplicate user_oauth provider/open_id rows exist; resolve them before adding the unique constraint''',
  'SELECT 1'
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SELECT COUNT(*) INTO @user_oauth_existing_unique_idx_provider_openid
FROM information_schema.STATISTICS
WHERE TABLE_SCHEMA = DATABASE()
  AND TABLE_NAME = 'user_oauth'
  AND INDEX_NAME = 'idx_provider_openid'
  AND NON_UNIQUE = 0;

SELECT COUNT(*) INTO @user_oauth_existing_unique_uk_provider_open_id
FROM information_schema.STATISTICS
WHERE TABLE_SCHEMA = DATABASE()
  AND TABLE_NAME = 'user_oauth'
  AND INDEX_NAME = 'uk_user_oauth_provider_open_id'
  AND NON_UNIQUE = 0;

SET @ddl = IF(
  @user_oauth_existing_unique_idx_provider_openid = 0 AND @user_oauth_existing_unique_uk_provider_open_id = 0,
  'ALTER TABLE `user_oauth` ADD UNIQUE KEY `uk_user_oauth_provider_open_id` (`provider`, `open_id`)',
  'SELECT 1'
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

-- +goose Down

SELECT COUNT(*) INTO @user_oauth_existing_unique_uk_provider_open_id
FROM information_schema.STATISTICS
WHERE TABLE_SCHEMA = DATABASE()
  AND TABLE_NAME = 'user_oauth'
  AND INDEX_NAME = 'uk_user_oauth_provider_open_id';

SET @ddl = IF(
  @user_oauth_existing_unique_uk_provider_open_id > 0,
  'ALTER TABLE `user_oauth` DROP INDEX `uk_user_oauth_provider_open_id`',
  'SELECT 1'
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;
