-- +goose Up
ALTER TABLE `matches`
  ADD COLUMN `sync_revision` bigint unsigned NOT NULL DEFAULT 0 COMMENT '对局同步版本号';

ALTER TABLE `match_actions`
  ADD COLUMN `client_action_id` varchar(64) DEFAULT NULL COMMENT '客户端操作ID',
  ADD COLUMN `base_revision` bigint unsigned NOT NULL DEFAULT 0 COMMENT '客户端基线版本号',
  ADD COLUMN `server_revision` bigint unsigned NOT NULL DEFAULT 0 COMMENT '服务端确认版本号';

ALTER TABLE `match_actions`
  ADD UNIQUE KEY `uk_match_client_action` (`match_id`, `client_action_id`);

-- +goose Down
ALTER TABLE `match_actions`
  DROP INDEX `uk_match_client_action`;

ALTER TABLE `match_actions`
  DROP COLUMN `server_revision`,
  DROP COLUMN `base_revision`,
  DROP COLUMN `client_action_id`;

ALTER TABLE `matches`
  DROP COLUMN `sync_revision`;
