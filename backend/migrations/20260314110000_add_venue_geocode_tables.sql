-- +goose Up
SET @venues_has_full_address := (
  SELECT COUNT(*)
  FROM information_schema.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE()
    AND TABLE_NAME = 'venues'
    AND COLUMN_NAME = 'full_address'
);

SET @venues_add_full_address_sql := IF(
  @venues_has_full_address = 0,
  'ALTER TABLE `venues` ADD COLUMN `full_address` VARCHAR(512) NOT NULL DEFAULT '''' COMMENT ''完整地址'' AFTER `district`',
  'SELECT 1'
);
PREPARE stmt FROM @venues_add_full_address_sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @venues_has_geo_status := (
  SELECT COUNT(*)
  FROM information_schema.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE()
    AND TABLE_NAME = 'venues'
    AND COLUMN_NAME = 'geo_status'
);

SET @venues_add_geo_status_sql := IF(
  @venues_has_geo_status = 0,
  'ALTER TABLE `venues` ADD COLUMN `geo_status` TINYINT NOT NULL DEFAULT 0 COMMENT ''地理解析状态 0=待解析 1=成功 2=重试中 3=失败'' AFTER `status`',
  'SELECT 1'
);
PREPARE stmt FROM @venues_add_geo_status_sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @venues_has_geo_source := (
  SELECT COUNT(*)
  FROM information_schema.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE()
    AND TABLE_NAME = 'venues'
    AND COLUMN_NAME = 'geo_source'
);

SET @venues_add_geo_source_sql := IF(
  @venues_has_geo_source = 0,
  'ALTER TABLE `venues` ADD COLUMN `geo_source` VARCHAR(32) NOT NULL DEFAULT '''' COMMENT ''地理解析来源'' AFTER `geo_status`',
  'SELECT 1'
);
PREPARE stmt FROM @venues_add_geo_source_sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @venues_has_geo_score := (
  SELECT COUNT(*)
  FROM information_schema.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE()
    AND TABLE_NAME = 'venues'
    AND COLUMN_NAME = 'geo_score'
);

SET @venues_add_geo_score_sql := IF(
  @venues_has_geo_score = 0,
  'ALTER TABLE `venues` ADD COLUMN `geo_score` INT NOT NULL DEFAULT 0 COMMENT ''地理解析匹配分'' AFTER `geo_source`',
  'SELECT 1'
);
PREPARE stmt FROM @venues_add_geo_score_sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @venues_has_geo_level := (
  SELECT COUNT(*)
  FROM information_schema.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE()
    AND TABLE_NAME = 'venues'
    AND COLUMN_NAME = 'geo_level'
);

SET @venues_add_geo_level_sql := IF(
  @venues_has_geo_level = 0,
  'ALTER TABLE `venues` ADD COLUMN `geo_level` VARCHAR(64) NOT NULL DEFAULT '''' COMMENT ''地理解析级别'' AFTER `geo_score`',
  'SELECT 1'
);
PREPARE stmt FROM @venues_add_geo_level_sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @venues_has_geo_attempts := (
  SELECT COUNT(*)
  FROM information_schema.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE()
    AND TABLE_NAME = 'venues'
    AND COLUMN_NAME = 'geo_attempts'
);

SET @venues_add_geo_attempts_sql := IF(
  @venues_has_geo_attempts = 0,
  'ALTER TABLE `venues` ADD COLUMN `geo_attempts` INT NOT NULL DEFAULT 0 COMMENT ''地理解析尝试次数'' AFTER `geo_level`',
  'SELECT 1'
);
PREPARE stmt FROM @venues_add_geo_attempts_sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @venues_has_geo_error := (
  SELECT COUNT(*)
  FROM information_schema.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE()
    AND TABLE_NAME = 'venues'
    AND COLUMN_NAME = 'geo_error'
);

SET @venues_add_geo_error_sql := IF(
  @venues_has_geo_error = 0,
  'ALTER TABLE `venues` ADD COLUMN `geo_error` VARCHAR(255) NOT NULL DEFAULT '''' COMMENT ''最近一次地理解析错误'' AFTER `geo_attempts`',
  'SELECT 1'
);
PREPARE stmt FROM @venues_add_geo_error_sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @venues_has_geo_updated_at := (
  SELECT COUNT(*)
  FROM information_schema.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE()
    AND TABLE_NAME = 'venues'
    AND COLUMN_NAME = 'geo_updated_at'
);

SET @venues_add_geo_updated_at_sql := IF(
  @venues_has_geo_updated_at = 0,
  'ALTER TABLE `venues` ADD COLUMN `geo_updated_at` DATETIME DEFAULT NULL COMMENT ''最近一次地理解析时间'' AFTER `geo_error`',
  'SELECT 1'
);
PREPARE stmt FROM @venues_add_geo_updated_at_sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @venues_has_duplicate_of_venue_id := (
  SELECT COUNT(*)
  FROM information_schema.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE()
    AND TABLE_NAME = 'venues'
    AND COLUMN_NAME = 'duplicate_of_venue_id'
);

SET @venues_add_duplicate_of_venue_id_sql := IF(
  @venues_has_duplicate_of_venue_id = 0,
  'ALTER TABLE `venues` ADD COLUMN `duplicate_of_venue_id` BIGINT UNSIGNED DEFAULT NULL COMMENT ''重复球馆归并目标ID'' AFTER `geo_updated_at`',
  'SELECT 1'
);
PREPARE stmt FROM @venues_add_duplicate_of_venue_id_sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @venues_has_status_geo_index := (
  SELECT COUNT(*)
  FROM information_schema.STATISTICS
  WHERE TABLE_SCHEMA = DATABASE()
    AND TABLE_NAME = 'venues'
    AND INDEX_NAME = 'idx_status_geo_status'
);

SET @venues_add_status_geo_index_sql := IF(
  @venues_has_status_geo_index = 0,
  'ALTER TABLE `venues` ADD KEY `idx_status_geo_status` (`status`, `geo_status`)',
  'SELECT 1'
);
PREPARE stmt FROM @venues_add_status_geo_index_sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

CREATE TABLE IF NOT EXISTS `venue_geocode_tasks` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `venue_id` BIGINT UNSIGNED NOT NULL COMMENT '球馆ID',
  `status` TINYINT NOT NULL DEFAULT 0 COMMENT '任务状态 0=pending 1=processing 2=success 3=retry 4=dead',
  `attempts` INT NOT NULL DEFAULT 0 COMMENT '已尝试次数',
  `max_attempts` INT NOT NULL DEFAULT 5 COMMENT '最大尝试次数',
  `next_retry_at` DATETIME DEFAULT NULL COMMENT '下次重试时间',
  `locked_at` DATETIME DEFAULT NULL COMMENT '任务领取时间',
  `locked_by` VARCHAR(64) NOT NULL DEFAULT '' COMMENT '任务领取者',
  `last_error` VARCHAR(255) NOT NULL DEFAULT '' COMMENT '最近一次错误',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uniq_venue_id` (`venue_id`),
  KEY `idx_status_next_retry_at` (`status`, `next_retry_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='球馆地理解析任务表';

CREATE TABLE IF NOT EXISTS `geocode_accounts` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `provider` VARCHAR(32) NOT NULL DEFAULT 'apihz' COMMENT '供应商',
  `provider_app_id` VARCHAR(64) NOT NULL DEFAULT '' COMMENT '供应商APP ID',
  `provider_key` VARCHAR(255) NOT NULL DEFAULT '' COMMENT '供应商密钥',
  `per_minute_limit` INT NOT NULL DEFAULT 10 COMMENT '每分钟限频',
  `status` TINYINT NOT NULL DEFAULT 1 COMMENT '账号状态 0=禁用 1=启用',
  `priority` INT NOT NULL DEFAULT 100 COMMENT '优先级，越小越优先',
  `cool_down_until` DATETIME DEFAULT NULL COMMENT '冷却截止时间',
  `fail_streak` INT NOT NULL DEFAULT 0 COMMENT '连续失败次数',
  `last_success_at` DATETIME DEFAULT NULL COMMENT '最近成功时间',
  `last_error_at` DATETIME DEFAULT NULL COMMENT '最近失败时间',
  `remark` VARCHAR(255) NOT NULL DEFAULT '' COMMENT '备注',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_provider_status` (`provider`, `status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='地理解析账号池';

-- +goose Down
DROP TABLE IF EXISTS `geocode_accounts`;
DROP TABLE IF EXISTS `venue_geocode_tasks`;
ALTER TABLE `venues`
  DROP INDEX `idx_status_geo_status`,
  DROP COLUMN `duplicate_of_venue_id`,
  DROP COLUMN `geo_updated_at`,
  DROP COLUMN `geo_error`,
  DROP COLUMN `geo_attempts`,
  DROP COLUMN `geo_level`,
  DROP COLUMN `geo_score`,
  DROP COLUMN `geo_source`,
  DROP COLUMN `geo_status`,
  DROP COLUMN `full_address`;
