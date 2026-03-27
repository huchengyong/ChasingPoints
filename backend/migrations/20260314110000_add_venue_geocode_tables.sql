-- +goose Up
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
