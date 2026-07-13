-- +goose Up
ALTER TABLE `social_posts`
  ADD COLUMN `status` TINYINT NOT NULL DEFAULT 1 COMMENT '状态 1=已发布 2=待审核 3=已拒绝' AFTER `comments_count`,
  ADD COLUMN `reject_reason` VARCHAR(255) NOT NULL DEFAULT '' COMMENT '审核拒绝原因' AFTER `status`,
  ADD COLUMN `reviewed_at` DATETIME NULL COMMENT '审核时间' AFTER `reject_reason`,
  ADD COLUMN `reviewed_by` BIGINT UNSIGNED NULL COMMENT '审核管理员ID' AFTER `reviewed_at`,
  ADD KEY `idx_status` (`status`);

ALTER TABLE `social_posts`
  MODIFY COLUMN `status` TINYINT NOT NULL DEFAULT 2 COMMENT '状态 1=已发布 2=待审核 3=已拒绝';

-- +goose Down
ALTER TABLE `social_posts`
  DROP INDEX `idx_status`,
  DROP COLUMN `reviewed_by`,
  DROP COLUMN `reviewed_at`,
  DROP COLUMN `reject_reason`,
  DROP COLUMN `status`;
