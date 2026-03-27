-- +goose Up
-- 赛事情报最终结构：赛事主表 + 阶段表

CREATE TABLE IF NOT EXISTS `event_news_events` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `title` VARCHAR(128) NOT NULL COMMENT '赛事标题',
  `game_type` TINYINT NOT NULL COMMENT '球种 1=斯诺克 2=中式九球 3=中式八球 4=美式九球',
  `source_type` VARCHAR(32) NOT NULL DEFAULT '' COMMENT '来源类型 manual/official/imported',
  `source_name` VARCHAR(64) NOT NULL DEFAULT '' COMMENT '来源名称',
  `source_url` VARCHAR(512) NOT NULL DEFAULT '' COMMENT '来源链接',
  `cover_image` VARCHAR(512) NOT NULL DEFAULT '' COMMENT '封面图',
  `summary` VARCHAR(512) NOT NULL DEFAULT '' COMMENT '摘要',
  `content` TEXT DEFAULT NULL COMMENT '正文',
  `country` VARCHAR(64) NOT NULL DEFAULT '' COMMENT '国家',
  `city` VARCHAR(64) NOT NULL DEFAULT '' COMMENT '城市',
  `venue` VARCHAR(128) NOT NULL DEFAULT '' COMMENT '场馆',
  `start_time` DATETIME DEFAULT NULL COMMENT '赛事开始时间',
  `end_time` DATETIME DEFAULT NULL COMMENT '赛事结束时间',
  `status` TINYINT NOT NULL DEFAULT 0 COMMENT '赛事状态 0=即将开始 1=进行中 2=已结束 3=已取消',
  `featured` TINYINT NOT NULL DEFAULT 0 COMMENT '是否首页焦点',
  `sort_time` DATETIME DEFAULT NULL COMMENT '排序时间',
  `published` TINYINT NOT NULL DEFAULT 0 COMMENT '是否发布 0=否 1=是',
  `published_at` DATETIME DEFAULT NULL COMMENT '发布时间',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `deleted_at` DATETIME DEFAULT NULL COMMENT '删除时间',
  PRIMARY KEY (`id`),
  KEY `idx_game_type` (`game_type`),
  KEY `idx_status` (`status`),
  KEY `idx_city` (`city`),
  KEY `idx_published` (`published`),
  KEY `idx_featured_published_sort_time` (`featured`, `published`, `sort_time`),
  KEY `idx_sort_time` (`sort_time`),
  KEY `idx_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='赛事情报赛事主表';

CREATE TABLE IF NOT EXISTS `event_news_stages` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `event_id` BIGINT UNSIGNED NOT NULL COMMENT '所属赛事ID',
  `stage_name` VARCHAR(128) NOT NULL COMMENT '阶段名称',
  `stage_order` INT NOT NULL DEFAULT 0 COMMENT '阶段排序',
  `start_time` DATETIME DEFAULT NULL COMMENT '阶段开始时间',
  `end_time` DATETIME DEFAULT NULL COMMENT '阶段结束时间',
  `status` TINYINT NOT NULL DEFAULT 0 COMMENT '阶段状态 0=即将开始 1=进行中 2=已结束 3=已取消',
  `result_text` VARCHAR(255) NOT NULL DEFAULT '' COMMENT '阶段赛果摘要',
  `sort_time` DATETIME DEFAULT NULL COMMENT '阶段排序时间',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `deleted_at` DATETIME DEFAULT NULL COMMENT '删除时间',
  PRIMARY KEY (`id`),
  KEY `idx_event_id` (`event_id`),
  KEY `idx_stage_order` (`stage_order`),
  KEY `idx_status` (`status`),
  KEY `idx_sort_time` (`sort_time`),
  KEY `idx_deleted_at` (`deleted_at`),
  CONSTRAINT `fk_event_news_stages_event_id`
    FOREIGN KEY (`event_id`) REFERENCES `event_news_events` (`id`)
    ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='赛事情报阶段表';

-- +goose Down

DROP TABLE IF EXISTS `event_news_stages`;
DROP TABLE IF EXISTS `event_news_events`;
