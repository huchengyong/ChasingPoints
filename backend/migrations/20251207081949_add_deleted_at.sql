-- +goose Up
-- 为所有表添加 deleted_at 字段以支持软删除

ALTER TABLE `users` ADD COLUMN `deleted_at` datetime DEFAULT NULL COMMENT '删除时间';
ALTER TABLE `users` ADD INDEX `idx_deleted_at` (`deleted_at`);

ALTER TABLE `user_oauth` ADD COLUMN `deleted_at` datetime DEFAULT NULL COMMENT '删除时间';
ALTER TABLE `user_oauth` ADD INDEX `idx_deleted_at` (`deleted_at`);

ALTER TABLE `verification_codes` ADD COLUMN `deleted_at` datetime DEFAULT NULL COMMENT '删除时间';
ALTER TABLE `verification_codes` ADD INDEX `idx_deleted_at` (`deleted_at`);

-- +goose Down
ALTER TABLE `users` DROP INDEX `idx_deleted_at`;
ALTER TABLE `users` DROP COLUMN `deleted_at`;

ALTER TABLE `user_oauth` DROP INDEX `idx_deleted_at`;
ALTER TABLE `user_oauth` DROP COLUMN `deleted_at`;

ALTER TABLE `verification_codes` DROP INDEX `idx_deleted_at`;
ALTER TABLE `verification_codes` DROP COLUMN `deleted_at`;
