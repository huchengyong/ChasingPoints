-- +goose Up
-- 扩展商品分类表结构，新增Banner相关字段
-- 创建时间: 2024-12-19

ALTER TABLE `product_categories` 
ADD COLUMN `banner_image` varchar(255) DEFAULT '' COMMENT '分类横幅Banner图' AFTER `icon`,
ADD COLUMN `banner_title` varchar(100) DEFAULT '' COMMENT 'Banner标题' AFTER `banner_image`,
ADD COLUMN `banner_subtitle` varchar(200) DEFAULT '' COMMENT 'Banner副标题' AFTER `banner_title`;

-- 更新现有分类的Banner数据
UPDATE `product_categories` SET 
  `banner_title` = '专业球杆系列',
  `banner_subtitle` = '大师之选 · 精准击球'
WHERE `name` = '球杆';

UPDATE `product_categories` SET 
  `banner_title` = '冲杆系列',
  `banner_subtitle` = '力量迸发 · 一击必中'
WHERE `name` = '冲杆';

UPDATE `product_categories` SET 
  `banner_title` = '跳杆系列',
  `banner_subtitle` = '灵活跳跃 · 突破障碍'
WHERE `name` = '跳杆';

UPDATE `product_categories` SET 
  `banner_title` = '球盒球袋系列',
  `banner_subtitle` = '安全收纳 · 便携出行'
WHERE `name` = '球盒/球袋';

-- 新增3个分类
INSERT INTO `product_categories` (`id`, `name`, `sort`, `status`, `banner_title`, `banner_subtitle`) VALUES
(5, '巧克', 5, 1, '巧克系列', '细腻手感 · 稳定发挥'),
(6, '皮头', 6, 1, '皮头系列', '精准触感 · 持久耐用'),
(7, '手套', 7, 1, '手套系列', '舒适透气 · 专业之选');

-- +goose Down
DELETE FROM `product_categories` WHERE `name` IN ('巧克', '皮头', '手套');
ALTER TABLE `product_categories` 
DROP COLUMN `banner_image`,
DROP COLUMN `banner_title`,
DROP COLUMN `banner_subtitle`;
