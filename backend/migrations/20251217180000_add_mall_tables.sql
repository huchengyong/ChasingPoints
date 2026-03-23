-- +goose Up
-- 台球商城模块数据库表
-- 创建时间: 2024-12-17

-- 1. 商品分类表
CREATE TABLE IF NOT EXISTS `product_categories` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `name` varchar(50) NOT NULL COMMENT '分类名称',
  `icon` varchar(255) DEFAULT '' COMMENT '分类图标',
  `sort` int NOT NULL DEFAULT 0 COMMENT '排序权重',
  `status` tinyint NOT NULL DEFAULT 1 COMMENT '状态:1启用 0禁用',
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='商品分类表';

-- 初始分类数据
INSERT INTO `product_categories` (`id`, `name`, `sort`, `status`) VALUES
(1, '球杆', 1, 1),
(2, '冲杆', 2, 1),
(3, '跳杆', 3, 1),
(4, '球盒/球袋', 4, 1);

-- 2. 商品表
CREATE TABLE IF NOT EXISTS `products` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `category_id` bigint unsigned NOT NULL COMMENT '分类ID',
  `name` varchar(100) NOT NULL COMMENT '商品名称',
  `subtitle` varchar(200) DEFAULT '' COMMENT '副标题',
  `cover_image` varchar(255) NOT NULL COMMENT '封面图',
  `images` json DEFAULT NULL COMMENT '商品图片列表',
  `description` text COMMENT '商品描述',
  `price` decimal(10,2) NOT NULL COMMENT '最低销售价',
  `original_price` decimal(10,2) DEFAULT NULL COMMENT '原价',
  `stock` int NOT NULL DEFAULT 0 COMMENT '总库存',
  `sales` int NOT NULL DEFAULT 0 COMMENT '销量',
  `rating` decimal(2,1) DEFAULT 5.0 COMMENT '评分',
  `review_count` int NOT NULL DEFAULT 0 COMMENT '评价数量',
  `is_hot` tinyint NOT NULL DEFAULT 0 COMMENT '是否热销',
  `is_new` tinyint NOT NULL DEFAULT 0 COMMENT '是否新品',
  `is_recommend` tinyint NOT NULL DEFAULT 0 COMMENT '是否推荐',
  `status` tinyint NOT NULL DEFAULT 1 COMMENT '状态:1上架 0下架',
  `sort` int NOT NULL DEFAULT 0 COMMENT '排序',
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `deleted_at` datetime DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_category` (`category_id`),
  KEY `idx_status_sort` (`status`, `sort`),
  KEY `idx_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='商品表';

-- 3. 商品规格属性表
CREATE TABLE IF NOT EXISTS `product_specs` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `product_id` bigint unsigned NOT NULL COMMENT '商品ID',
  `name` varchar(50) NOT NULL COMMENT '规格名称',
  `sort` int NOT NULL DEFAULT 0 COMMENT '排序',
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_product` (`product_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='商品规格属性表';

-- 4. 商品规格值表
CREATE TABLE IF NOT EXISTS `product_spec_values` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `spec_id` bigint unsigned NOT NULL COMMENT '规格ID',
  `value` varchar(50) NOT NULL COMMENT '规格值',
  `sort` int NOT NULL DEFAULT 0 COMMENT '排序',
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_spec` (`spec_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='商品规格值表';

-- 5. 商品SKU表
CREATE TABLE IF NOT EXISTS `product_skus` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `product_id` bigint unsigned NOT NULL COMMENT '商品ID',
  `spec_value_ids` varchar(100) NOT NULL DEFAULT '' COMMENT '规格值ID组合',
  `spec_text` varchar(200) NOT NULL DEFAULT '' COMMENT '规格文本',
  `price` decimal(10,2) NOT NULL COMMENT 'SKU价格',
  `original_price` decimal(10,2) DEFAULT NULL COMMENT '原价',
  `stock` int NOT NULL DEFAULT 0 COMMENT '库存',
  `sales` int NOT NULL DEFAULT 0 COMMENT '销量',
  `status` tinyint NOT NULL DEFAULT 1 COMMENT '状态:1启用 0禁用',
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_product` (`product_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='商品SKU表';

-- 6. 商品参数表
CREATE TABLE IF NOT EXISTS `product_attributes` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `product_id` bigint unsigned NOT NULL COMMENT '商品ID',
  `name` varchar(50) NOT NULL COMMENT '参数名',
  `value` varchar(200) NOT NULL COMMENT '参数值',
  `sort` int NOT NULL DEFAULT 0 COMMENT '排序',
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_product` (`product_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='商品参数表';

-- 7. 轮播Banner表
CREATE TABLE IF NOT EXISTS `banners` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `title` varchar(100) NOT NULL COMMENT '标题',
  `subtitle` varchar(200) DEFAULT '' COMMENT '副标题',
  `image` varchar(255) NOT NULL COMMENT '图片地址',
  `link_type` varchar(20) DEFAULT 'product' COMMENT '链接类型:product/category/url',
  `link_value` varchar(255) DEFAULT '' COMMENT '链接值',
  `sort` int NOT NULL DEFAULT 0 COMMENT '排序',
  `status` tinyint NOT NULL DEFAULT 1 COMMENT '状态:1显示 0隐藏',
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='轮播Banner表';

-- 8. 必备装备表
CREATE TABLE IF NOT EXISTS `essential_equipments` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `product_id` bigint unsigned NOT NULL COMMENT '关联商品ID',
  `sort` int NOT NULL DEFAULT 0 COMMENT '排序',
  `status` tinyint NOT NULL DEFAULT 1 COMMENT '状态:1显示 0隐藏',
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_product` (`product_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='必备装备表';

-- 9. 购物车表
CREATE TABLE IF NOT EXISTS `carts` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `user_id` bigint unsigned NOT NULL COMMENT '用户ID',
  `product_id` bigint unsigned NOT NULL COMMENT '商品ID',
  `sku_id` bigint unsigned NOT NULL COMMENT 'SKU ID',
  `quantity` int NOT NULL DEFAULT 1 COMMENT '数量',
  `selected` tinyint NOT NULL DEFAULT 1 COMMENT '是否选中',
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_user_sku` (`user_id`, `sku_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='购物车表';

-- 10. 用户收藏表
CREATE TABLE IF NOT EXISTS `user_favorites` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `user_id` bigint unsigned NOT NULL COMMENT '用户ID',
  `product_id` bigint unsigned NOT NULL COMMENT '商品ID',
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_user_product` (`user_id`, `product_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='用户收藏表';

-- 11. 优惠券表
CREATE TABLE IF NOT EXISTS `coupons` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `name` varchar(100) NOT NULL COMMENT '优惠券名称',
  `type` tinyint NOT NULL DEFAULT 1 COMMENT '类型:1满减 2折扣',
  `value` decimal(10,2) NOT NULL COMMENT '优惠值',
  `min_amount` decimal(10,2) DEFAULT 0 COMMENT '最低消费金额',
  `start_time` datetime NOT NULL COMMENT '开始时间',
  `end_time` datetime NOT NULL COMMENT '结束时间',
  `total_count` int NOT NULL DEFAULT 0 COMMENT '发放总量（0无限制）',
  `used_count` int NOT NULL DEFAULT 0 COMMENT '已使用数量',
  `status` tinyint NOT NULL DEFAULT 1 COMMENT '状态:1启用 0禁用',
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='优惠券表';

-- 12. 用户优惠券表
CREATE TABLE IF NOT EXISTS `user_coupons` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `user_id` bigint unsigned NOT NULL COMMENT '用户ID',
  `coupon_id` bigint unsigned NOT NULL COMMENT '优惠券ID',
  `status` tinyint NOT NULL DEFAULT 0 COMMENT '状态:0未使用 1已使用 2已过期',
  `used_order_id` bigint unsigned DEFAULT NULL COMMENT '使用的订单ID',
  `used_at` datetime DEFAULT NULL COMMENT '使用时间',
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_user_status` (`user_id`, `status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='用户优惠券表';

-- 13. 订单表
CREATE TABLE IF NOT EXISTS `orders` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `order_no` varchar(32) NOT NULL COMMENT '订单编号',
  `user_id` bigint unsigned NOT NULL COMMENT '用户ID',
  `status` tinyint NOT NULL DEFAULT 0 COMMENT '状态:0待付款 1待发货 2待收货 3已完成 4已取消',
  `total_amount` decimal(10,2) NOT NULL COMMENT '商品总金额',
  `freight_amount` decimal(10,2) NOT NULL DEFAULT 0 COMMENT '运费',
  `coupon_amount` decimal(10,2) NOT NULL DEFAULT 0 COMMENT '优惠券抵扣金额',
  `pay_amount` decimal(10,2) NOT NULL COMMENT '实付金额',
  `coupon_id` bigint unsigned DEFAULT NULL COMMENT '使用的优惠券ID',
  `pay_type` tinyint DEFAULT NULL COMMENT '支付方式:1支付宝',
  `pay_time` datetime DEFAULT NULL COMMENT '支付时间',
  `ship_time` datetime DEFAULT NULL COMMENT '发货时间',
  `receive_time` datetime DEFAULT NULL COMMENT '收货时间',
  `receiver_name` varchar(50) NOT NULL COMMENT '收货人姓名',
  `receiver_phone` varchar(20) NOT NULL COMMENT '收货人电话',
  `receiver_address` varchar(255) NOT NULL COMMENT '收货地址(完整)',
  `remark` varchar(500) DEFAULT '' COMMENT '订单备注',
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `deleted_at` datetime DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_order_no` (`order_no`),
  KEY `idx_user_status` (`user_id`, `status`),
  KEY `idx_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='订单表';

-- 14. 订单商品表
CREATE TABLE IF NOT EXISTS `order_items` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `order_id` bigint unsigned NOT NULL COMMENT '订单ID',
  `product_id` bigint unsigned NOT NULL COMMENT '商品ID',
  `sku_id` bigint unsigned NOT NULL COMMENT 'SKU ID',
  `product_name` varchar(100) NOT NULL COMMENT '商品名称(快照)',
  `product_image` varchar(255) NOT NULL COMMENT '商品图片(快照)',
  `spec_text` varchar(200) DEFAULT '' COMMENT '规格文本(快照)',
  `price` decimal(10,2) NOT NULL COMMENT '单价(快照)',
  `quantity` int NOT NULL COMMENT '数量',
  `total_amount` decimal(10,2) NOT NULL COMMENT '小计金额',
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_order` (`order_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='订单商品表';

-- 15. 商品评价表
CREATE TABLE IF NOT EXISTS `product_reviews` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `order_item_id` bigint unsigned NOT NULL COMMENT '订单商品ID',
  `product_id` bigint unsigned NOT NULL COMMENT '商品ID',
  `user_id` bigint unsigned NOT NULL COMMENT '用户ID',
  `rating` tinyint NOT NULL DEFAULT 5 COMMENT '评分(1-5)',
  `content` text COMMENT '评价内容',
  `images` json DEFAULT NULL COMMENT '评价图片',
  `is_anonymous` tinyint NOT NULL DEFAULT 0 COMMENT '是否匿名',
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_order_item` (`order_item_id`),
  KEY `idx_product` (`product_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='商品评价表';

-- +goose Down
DROP TABLE IF EXISTS `product_reviews`;
DROP TABLE IF EXISTS `order_items`;
DROP TABLE IF EXISTS `orders`;
DROP TABLE IF EXISTS `user_coupons`;
DROP TABLE IF EXISTS `coupons`;
DROP TABLE IF EXISTS `user_favorites`;
DROP TABLE IF EXISTS `carts`;
DROP TABLE IF EXISTS `essential_equipments`;
DROP TABLE IF EXISTS `banners`;
DROP TABLE IF EXISTS `product_attributes`;
DROP TABLE IF EXISTS `product_skus`;
DROP TABLE IF EXISTS `product_spec_values`;
DROP TABLE IF EXISTS `product_specs`;
DROP TABLE IF EXISTS `products`;
DROP TABLE IF EXISTS `product_categories`;
