-- +goose Up
SET @db_name = DATABASE();

SET @hide_match_record_exists = (
	SELECT COUNT(*)
	FROM information_schema.COLUMNS
	WHERE TABLE_SCHEMA = @db_name
		AND TABLE_NAME = 'users'
		AND COLUMN_NAME = 'hide_match_record'
);
SET @add_hide_match_record_sql = IF(
	@hide_match_record_exists = 0,
	'ALTER TABLE `users` ADD COLUMN `hide_match_record` TINYINT(1) NOT NULL DEFAULT 0 COMMENT ''是否隐藏战绩'' AFTER `member_expires_at`',
	'SELECT 1'
);
PREPARE add_hide_match_record_stmt FROM @add_hide_match_record_sql;
EXECUTE add_hide_match_record_stmt;
DEALLOCATE PREPARE add_hide_match_record_stmt;

CREATE TABLE IF NOT EXISTS `member_subscription_orders` (
	`id` BIGINT NOT NULL AUTO_INCREMENT,
	`order_no` VARCHAR(64) NOT NULL COMMENT '订单号',
	`user_id` BIGINT NOT NULL COMMENT '用户ID',
	`plan_code` VARCHAR(64) NOT NULL COMMENT '套餐编码',
	`plan_name` VARCHAR(64) NOT NULL COMMENT '套餐名称',
	`duration_days` INT NOT NULL DEFAULT 30 COMMENT '会员天数',
	`amount_fen` INT NOT NULL COMMENT '金额(分)',
	`pay_channel` VARCHAR(32) NOT NULL COMMENT '支付渠道',
	`status` VARCHAR(32) NOT NULL DEFAULT 'pending' COMMENT '订单状态',
	`third_party_order_no` VARCHAR(128) NOT NULL DEFAULT '' COMMENT '第三方流水号',
	`paid_at` DATETIME DEFAULT NULL COMMENT '支付时间',
	`member_expires_at_before` DATETIME DEFAULT NULL COMMENT '发放前会员到期时间',
	`member_expires_at_after` DATETIME DEFAULT NULL COMMENT '发放后会员到期时间',
	`created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
	`updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
	PRIMARY KEY (`id`),
	UNIQUE KEY `uniq_member_subscription_orders_order_no` (`order_no`),
	KEY `idx_member_subscription_orders_user_id` (`user_id`),
	KEY `idx_member_subscription_orders_status` (`status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='会员订阅订单表';

-- +goose Down
DROP TABLE IF EXISTS `member_subscription_orders`;

SET @db_name = DATABASE();

SET @hide_match_record_exists = (
	SELECT COUNT(*)
	FROM information_schema.COLUMNS
	WHERE TABLE_SCHEMA = @db_name
		AND TABLE_NAME = 'users'
		AND COLUMN_NAME = 'hide_match_record'
);
SET @drop_hide_match_record_sql = IF(
	@hide_match_record_exists = 1,
	'ALTER TABLE `users` DROP COLUMN `hide_match_record`',
	'SELECT 1'
);
PREPARE drop_hide_match_record_stmt FROM @drop_hide_match_record_sql;
EXECUTE drop_hide_match_record_stmt;
DEALLOCATE PREPARE drop_hide_match_record_stmt;
