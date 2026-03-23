-- +goose Up
-- 创建用户收货地址表
CREATE TABLE IF NOT EXISTS `user_addresses` (
    `id` bigint NOT NULL AUTO_INCREMENT,
    `user_id` bigint NOT NULL COMMENT '用户ID',
    `name` varchar(50) NOT NULL COMMENT '收货人姓名',
    `phone` varchar(20) NOT NULL COMMENT '手机号',
    `province` varchar(50) NOT NULL COMMENT '省',
    `city` varchar(50) NOT NULL COMMENT '市',
    `district` varchar(50) NOT NULL COMMENT '区',
    `detail` varchar(255) NOT NULL COMMENT '详细地址',
    `post_code` varchar(10) NOT NULL DEFAULT '' COMMENT '邮编',
    `tag` varchar(20) NOT NULL DEFAULT '' COMMENT '标签：home/company/default',
    `is_default` tinyint(1) NOT NULL DEFAULT 0 COMMENT '是否默认地址',
    `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    `deleted_at` datetime DEFAULT NULL,
    PRIMARY KEY (`id`),
    INDEX `idx_user_id` (`user_id`),
    INDEX `idx_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='用户收货地址表';

-- +goose Down
DROP TABLE IF EXISTS `user_addresses`;
