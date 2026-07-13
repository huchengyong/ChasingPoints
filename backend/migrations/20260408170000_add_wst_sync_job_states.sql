-- +goose Up

CREATE TABLE IF NOT EXISTS `wst_sync_job_states` (
  `id` BIGINT NOT NULL AUTO_INCREMENT,
  `job_name` VARCHAR(64) NOT NULL COMMENT '同步任务名',
  `last_successful_sync_at` DATETIME DEFAULT NULL COMMENT '最近一次成功同步时间',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_wst_sync_job_states_job_name` (`job_name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- +goose Down

DROP TABLE IF EXISTS `wst_sync_job_states`;
