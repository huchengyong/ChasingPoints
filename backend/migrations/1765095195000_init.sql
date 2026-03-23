-- +goose Up
-- 台球助手数据库初始化脚本
-- 创建时间: 2024-12-06

-- 用户表
CREATE TABLE IF NOT EXISTS `users` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '用户ID',
  `phone` varchar(20) DEFAULT NULL COMMENT '手机号',
  `nickname` varchar(50) NOT NULL DEFAULT '' COMMENT '昵称',
  `avatar` varchar(255) NOT NULL DEFAULT '' COMMENT '头像URL',
  `status` tinyint NOT NULL DEFAULT 1 COMMENT '状态:1正常 0禁用',
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_phone` (`phone`)
) ENGINE=InnoDB AUTO_INCREMENT=143713 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='用户表';

-- OAuth关联表
CREATE TABLE IF NOT EXISTS `user_oauth` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '记录ID',
  `user_id` bigint unsigned NOT NULL COMMENT '用户ID',
  `provider` varchar(20) NOT NULL COMMENT 'OAuth提供商:huawei',
  `open_id` varchar(128) NOT NULL COMMENT 'OpenID',
  `union_id` varchar(128) DEFAULT NULL COMMENT 'UnionID',
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_provider_openid` (`provider`, `open_id`),
  KEY `idx_user_id` (`user_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='OAuth关联表';

-- 验证码表（用于存储短信验证码）
CREATE TABLE IF NOT EXISTS `verification_codes` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '记录ID',
  `phone` varchar(20) NOT NULL COMMENT '手机号',
  `code` varchar(10) NOT NULL COMMENT '验证码',
  `scene` varchar(20) NOT NULL DEFAULT 'login' COMMENT '场景:login/bind',
  `expires_at` datetime NOT NULL COMMENT '过期时间',
  `used` tinyint NOT NULL DEFAULT 0 COMMENT '是否已使用:0未使用 1已使用',
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  PRIMARY KEY (`id`),
  KEY `idx_phone_scene` (`phone`, `scene`),
  KEY `idx_expires_at` (`expires_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='验证码表';

-- +goose Down
DROP TABLE IF EXISTS `verification_codes`;
DROP TABLE IF EXISTS `user_oauth`;
DROP TABLE IF EXISTS `users`;
