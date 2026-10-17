-- +goose Up

CREATE TABLE IF NOT EXISTS `llm_configs` (
  `id` BIGINT NOT NULL AUTO_INCREMENT,
  `config_key` VARCHAR(64) NOT NULL COMMENT '配置唯一键',
  `api_format` VARCHAR(32) NOT NULL COMMENT '接口格式: anthropic/openai',
  `endpoint_url` VARCHAR(512) NOT NULL COMMENT '大模型接口完整地址',
  `api_key` VARCHAR(512) NOT NULL COMMENT '接口鉴权 API Key（Bearer）',
  `model` VARCHAR(128) NOT NULL COMMENT '模型名称',
  `max_tokens` INT NOT NULL COMMENT '单次调用 max_tokens',
  `updated_by` BIGINT NOT NULL DEFAULT 0 COMMENT '最后更新管理员ID',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uniq_llm_configs_key` (`config_key`)
) ENGINE=InnoDB COMMENT='大模型调用配置表';

-- +goose Down

DROP TABLE IF EXISTS `llm_configs`;
