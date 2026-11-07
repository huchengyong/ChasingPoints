-- +goose Up

CREATE TABLE IF NOT EXISTS `gear_brands` (
  `id` BIGINT NOT NULL AUTO_INCREMENT,
  `name` VARCHAR(64) NOT NULL COMMENT '品牌名称',
  `aliases` VARCHAR(255) NOT NULL DEFAULT '' COMMENT '搜索别名，逗号分隔',
  `sort` INT NOT NULL DEFAULT 0 COMMENT '排序值，越小越靠前',
  `status` TINYINT NOT NULL DEFAULT 1 COMMENT '状态 1=启用 0=停用',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uniq_gear_brands_name` (`name`)
) ENGINE=InnoDB COMMENT='装备品牌目录表';

CREATE TABLE IF NOT EXISTS `gear_products` (
  `id` BIGINT NOT NULL AUTO_INCREMENT,
  `category` VARCHAR(16) NOT NULL COMMENT '类别 cue=球杆 break_jump=冲杆跳杆 tip=皮头 chalk=巧克',
  `brand_id` BIGINT NOT NULL DEFAULT 0 COMMENT '品牌ID，0=无品牌（球房公用条目）',
  `name` VARCHAR(128) NOT NULL COMMENT '型号名称',
  `house` TINYINT NOT NULL DEFAULT 0 COMMENT '是否球房公用条目 1=是',
  `cue_style` VARCHAR(16) NOT NULL DEFAULT '' COMMENT '球杆杆型 snooker=斯诺克/中式 pool=美式',
  `roles` VARCHAR(32) NOT NULL DEFAULT '' COMMENT '冲跳身份，逗号分隔 break,jump',
  `length_mm` INT NOT NULL DEFAULT 0 COMMENT '官方长度毫米，0=未提供',
  `weight_g` INT NOT NULL DEFAULT 0 COMMENT '官方重量克，0=未提供',
  `tip_diameter_mm` DECIMAL(5,2) NOT NULL DEFAULT 0 COMMENT '官方皮头直径毫米，0=未提供',
  `image_path` VARCHAR(255) NOT NULL DEFAULT '' COMMENT '目录图路径，空=使用默认图',
  `sort` INT NOT NULL DEFAULT 0 COMMENT '排序值，越小越靠前',
  `status` TINYINT NOT NULL DEFAULT 1 COMMENT '状态 1=启用 0=停用',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uniq_gear_products_identity` (`category`, `brand_id`, `name`),
  KEY `idx_gear_products_list` (`category`, `status`, `brand_id`, `sort`, `id`)
) ENGINE=InnoDB COMMENT='装备型号目录表';

CREATE TABLE IF NOT EXISTS `user_gear_items` (
  `id` BIGINT NOT NULL AUTO_INCREMENT,
  `user_id` BIGINT NOT NULL COMMENT '用户ID',
  `category` VARCHAR(16) NOT NULL COMMENT '类别 cue=球杆 break_jump=冲杆跳杆',
  `product_id` BIGINT NOT NULL DEFAULT 0 COMMENT '目录型号ID，0=未收录型号',
  `custom_brand_id` BIGINT NOT NULL DEFAULT 0 COMMENT '未收录登记时选择的目录品牌ID',
  `custom_brand_name` VARCHAR(32) NOT NULL DEFAULT '' COMMENT '未收录登记时自填的品牌名',
  `custom_model_name` VARCHAR(64) NOT NULL DEFAULT '' COMMENT '未收录登记时自填的型号名',
  `custom_key` VARCHAR(255) NOT NULL DEFAULT '' COMMENT '未收录登记的归并键',
  `cue_style` VARCHAR(16) NOT NULL DEFAULT '' COMMENT '未收录球杆的杆型',
  `roles` VARCHAR(32) NOT NULL DEFAULT '' COMMENT '未收录冲跳装备的身份，逗号分隔',
  `tip_product_id` BIGINT NOT NULL DEFAULT 0 COMMENT '当前皮头型号ID，0=未填',
  `chalk_product_id` BIGINT NOT NULL DEFAULT 0 COMMENT '当前巧克型号ID，0=未填',
  `status` TINYINT NOT NULL DEFAULT 1 COMMENT '状态 1=在用 2=退役 3=已删除',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_user_gear_items_user_status` (`user_id`, `status`, `id`),
  KEY `idx_user_gear_items_pending` (`product_id`, `custom_key`)
) ENGINE=InnoDB COMMENT='用户装备表';

CREATE TABLE IF NOT EXISTS `user_gear_defaults` (
  `id` BIGINT NOT NULL AUTO_INCREMENT,
  `user_id` BIGINT NOT NULL COMMENT '用户ID',
  `game_type` TINYINT NOT NULL COMMENT '球种 1=斯诺克 2=九球追分 3=中式八球 4=美式九球',
  `slot` VARCHAR(8) NOT NULL COMMENT '槽位 cue=主力球杆 break=主力冲杆',
  `item_id` BIGINT NOT NULL COMMENT '装备ID',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uniq_user_gear_defaults` (`user_id`, `game_type`, `slot`)
) ENGINE=InnoDB COMMENT='用户主力装备设置表';

CREATE TABLE IF NOT EXISTS `match_gear_snapshots` (
  `id` BIGINT NOT NULL AUTO_INCREMENT,
  `match_id` BIGINT NOT NULL COMMENT '对局ID',
  `user_id` BIGINT NOT NULL COMMENT '选手用户ID',
  `cue_item_id` BIGINT NOT NULL DEFAULT 0 COMMENT '球杆装备ID，0=未记录',
  `tip_product_id` BIGINT NOT NULL DEFAULT 0 COMMENT '记录当时的皮头型号ID，0=未填',
  `chalk_product_id` BIGINT NOT NULL DEFAULT 0 COMMENT '记录当时的巧克型号ID，0=未填',
  `break_item_id` BIGINT NOT NULL DEFAULT 0 COMMENT '冲杆装备ID，0=未记录',
  `source` VARCHAR(16) NOT NULL DEFAULT 'auto' COMMENT '来源 auto=完成时按主力设置自动记录 manual=选手手动修改',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uniq_match_gear_snapshots` (`match_id`, `user_id`),
  KEY `idx_match_gear_snapshots_user` (`user_id`, `match_id`)
) ENGINE=InnoDB COMMENT='对局用杆记录表';

INSERT IGNORE INTO `gear_products` (`category`, `brand_id`, `name`, `house`, `sort`, `status`) VALUES
  ('cue', 0, '球房公杆', 1, 0, 1),
  ('chalk', 0, '球房巧克', 1, 0, 1);

-- +goose Down

DROP TABLE IF EXISTS `match_gear_snapshots`;
DROP TABLE IF EXISTS `user_gear_defaults`;
DROP TABLE IF EXISTS `user_gear_items`;
DROP TABLE IF EXISTS `gear_products`;
DROP TABLE IF EXISTS `gear_brands`;
