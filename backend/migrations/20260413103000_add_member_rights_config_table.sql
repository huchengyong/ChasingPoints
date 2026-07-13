-- +goose Up

CREATE TABLE IF NOT EXISTS `member_rights_configs` (
  `id` BIGINT NOT NULL AUTO_INCREMENT,
  `config_key` VARCHAR(64) NOT NULL COMMENT '配置唯一键',
  `growth_rules_json` JSON NOT NULL COMMENT '会员成长规则',
  `ranking_rights_rules_json` JSON NOT NULL COMMENT '会员排位权益规则',
  `updated_by` BIGINT NOT NULL DEFAULT 0 COMMENT '最后更新管理员ID',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uniq_member_rights_configs_key` (`config_key`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='会员体系配置表';

-- +goose Down
DROP TABLE IF EXISTS `member_rights_configs`;
