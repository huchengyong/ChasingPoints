-- +goose Up
-- 商城测试数据
-- 多品牌台球杆真实数据

-- 1. 插入轮播图数据
INSERT INTO `banners` (`title`, `subtitle`, `image`, `link_type`, `link_value`, `sort`, `status`) VALUES
('PREDATOR美洲豹球杆', '职业选手之选', 'https://cdn.dianzaozao.com/billiard/cues/predator/f89b532079a358d6.jpg', 'product', '4', 1, 1),
('Mezz美兹专业台球杆', '日本匠心工艺', 'https://cdn.dianzaozao.com/billiard/cues/mezz/98dcf5e2b8a138b1.jpg', 'category', '1', 2, 1),
('Riley莱利英式球杆', '英伦传统品质', 'https://cdn.dianzaozao.com/billiard/cues/riley/29f83cfa6acb9add.jpg', 'category', '1', 3, 1);

-- 2. 插入商品数据
INSERT INTO `products` (`category_id`, `name`, `subtitle`, `cover_image`, `images`, `description`, `price`, `original_price`, `stock`, `sales`, `rating`, `is_hot`, `is_new`, `is_recommend`, `status`, `sort`) VALUES
-- Predator 美洲豹 (ID 1-5)
(3, '美洲豹PREDATOR跳杆', '胶把 黄色空气跳 三段式台球杆正品台球用品', 'https://cdn.dianzaozao.com/billiard/cues/predator/a05280e1ceb0f734.jpg', '["https://cdn.dianzaozao.com/billiard/cues/predator/a05280e1ceb0f734.jpg"]', 'PREDATOR美洲豹跳杆，三段式设计。', 3014.00, 3500.00, 99, 128, 4.9, 1, 0, 1, 1, 100),
(3, 'PREDATOR美州豹台球跳杆', 'AIR2空气跳杆球杆 三节跳杆', 'https://cdn.dianzaozao.com/billiard/cues/predator/bbd0d50666c4c35a.jpg', '["https://cdn.dianzaozao.com/billiard/cues/predator/bbd0d50666c4c35a.jpg"]', 'PREDATOR AIR2空气跳杆。', 2550.00, 3000.00, 594, 256, 4.8, 1, 1, 1, 1, 99),
(3, 'PREDATOR美洲豹跳杆AIR RUSH', '碳纤维跳杆球杆 三节跳杆', 'https://cdn.dianzaozao.com/billiard/cues/predator/e12d6bb117b1bbc7.jpg', '["https://cdn.dianzaozao.com/billiard/cues/predator/e12d6bb117b1bbc7.jpg"]', 'PREDATOR AIR RUSH碳纤维跳杆。', 6171.00, 7000.00, 396, 89, 4.9, 1, 1, 1, 1, 98),
(1, 'PREDATOR美洲豹台球杆', '专业中八球杆 高端前节檀木枫木', 'https://cdn.dianzaozao.com/billiard/cues/predator/f59e71f8f35bc1ad.jpg', '["https://cdn.dianzaozao.com/billiard/cues/predator/f59e71f8f35bc1ad.jpg"]', 'PREDATOR美洲豹旗舰台球杆。', 19350.00, 22000.00, 297, 45, 5.0, 1, 0, 1, 1, 97),
(1, 'PREDATOR美洲豹入门级台球杆', '基础款训练杆 新手专用', 'https://cdn.dianzaozao.com/billiard/cues/predator/f70761ed91bc5f08.jpg', '["https://cdn.dianzaozao.com/billiard/cues/predator/f70761ed91bc5f08.jpg"]', 'PREDATOR入门级台球杆。', 235.00, 299.00, 198, 520, 4.6, 0, 0, 1, 1, 96),
-- Mezz 美兹 (ID 6-10)
(1, 'Mezz美兹专业台球杆', '日本进口 高端前节 加拿大枫木', 'https://cdn.dianzaozao.com/billiard/cues/mezz/98dcf5e2b8a138b1.jpg', '["https://cdn.dianzaozao.com/billiard/cues/mezz/98dcf5e2b8a138b1.jpg"]', 'Mezz美兹专业台球杆。', 4500.00, 5200.00, 297, 156, 4.9, 1, 0, 1, 1, 95),
(1, 'Mezz美兹EXTRA赛级台球杆', '比赛专用 碳纤维前节', 'https://cdn.dianzaozao.com/billiard/cues/mezz/7aee49c9411a0d36.jpg', '["https://cdn.dianzaozao.com/billiard/cues/mezz/7aee49c9411a0d36.jpg"]', 'Mezz美兹EXTRA赛级台球杆。', 6800.00, 7800.00, 297, 89, 5.0, 1, 1, 1, 1, 94),
(2, 'Mezz美兹Power Break冲杆', '冲杆专用 强化力量', 'https://cdn.dianzaozao.com/billiard/cues/mezz/a49634c4cb040b14.jpg', '["https://cdn.dianzaozao.com/billiard/cues/mezz/a49634c4cb040b14.jpg"]', 'Mezz美兹Power Break冲杆。', 3200.00, 3800.00, 396, 168, 4.8, 1, 0, 1, 1, 93),
(1, 'Mezz美兹经典款三节球杆', '传统工艺 舒适手柄', 'https://cdn.dianzaozao.com/billiard/cues/mezz/c35d7c3fba9d6662.jpg', '["https://cdn.dianzaozao.com/billiard/cues/mezz/c35d7c3fba9d6662.jpg"]', 'Mezz美兹经典款三节球杆。', 2800.00, 3300.00, 297, 210, 4.7, 0, 0, 1, 1, 92),
(1, 'Mezz美兹入门训练台球杆', '新手适用 性价比高', 'https://cdn.dianzaozao.com/billiard/cues/mezz/cf9cd9006fcfa0f6.jpg', '["https://cdn.dianzaozao.com/billiard/cues/mezz/cf9cd9006fcfa0f6.jpg"]', 'Mezz美兹入门训练台球杆。', 1200.00, 1500.00, 198, 380, 4.5, 0, 0, 0, 1, 91),
-- Riley 莱利 (ID 11-15)
(1, 'Riley莱利英式斯诺克球杆', '传统手工制作 非洲乌木', 'https://cdn.dianzaozao.com/billiard/cues/riley/2edcc64ac0d0bf4f.jpg', '["https://cdn.dianzaozao.com/billiard/cues/riley/2edcc64ac0d0bf4f.jpg"]', 'Riley莱利英式斯诺克球杆。', 3800.00, 4500.00, 297, 178, 4.9, 1, 0, 1, 1, 90),
(1, 'Riley莱利中式八球台球杆', '专业比赛用杆 高级檀木', 'https://cdn.dianzaozao.com/billiard/cues/riley/29f83cfa6acb9add.jpg', '["https://cdn.dianzaozao.com/billiard/cues/riley/29f83cfa6acb9add.jpg"]', 'Riley莱利中式八球台球杆。', 2500.00, 3000.00, 198, 245, 4.8, 1, 1, 1, 1, 89),
(2, 'Riley莱利冲杆专用球杆', '强化力量型 超硬前节', 'https://cdn.dianzaozao.com/billiard/cues/riley/48d0356d2e76d29f.jpg', '["https://cdn.dianzaozao.com/billiard/cues/riley/48d0356d2e76d29f.jpg"]', 'Riley莱利冲杆专用球杆。', 1800.00, 2200.00, 297, 156, 4.7, 0, 0, 1, 1, 88),
(1, 'Riley莱利新手入门套装', '两节球杆+球杆盒', 'https://cdn.dianzaozao.com/billiard/cues/riley/6fccad779c090f4d.jpg', '["https://cdn.dianzaozao.com/billiard/cues/riley/6fccad779c090f4d.jpg"]', 'Riley莱利新手入门套装。', 890.00, 1100.00, 198, 420, 4.6, 0, 0, 0, 1, 87),
(1, 'Riley莱利定制款签名球杆', '大师签名款 限量版', 'https://cdn.dianzaozao.com/billiard/cues/riley/de7b5c7929993fc4.jpg', '["https://cdn.dianzaozao.com/billiard/cues/riley/de7b5c7929993fc4.jpg"]', 'Riley莱利定制款签名球杆。', 12000.00, 15000.00, 297, 32, 5.0, 1, 1, 1, 1, 86),
-- Fury 威利 (ID 16-18)
(1, 'Fury(威利) 中式黑8中头杆', '波茨杆 中八 16彩 9.8mm', 'https://cdn.dianzaozao.com/billiard/cues/fury/612a25eac7174c7f.jpg', '["https://cdn.dianzaozao.com/billiard/cues/fury/612a25eac7174c7f.jpg"]', 'Fury威利中式黑8中头杆。', 719.00, 899.00, 198, 320, 4.7, 1, 0, 1, 1, 85),
(1, 'Fury(威利) 大头AWP球杆', '威力黑八中式八球 九球杆 11.75mm', 'https://cdn.dianzaozao.com/billiard/cues/fury/73636782b91d5fba.jpg', '["https://cdn.dianzaozao.com/billiard/cues/fury/73636782b91d5fba.jpg"]', 'Fury威利大头AWP球杆。', 728.00, 899.00, 198, 280, 4.6, 0, 0, 1, 1, 84),
(1, 'Fury(威利) AK入门级球杆', '威力中式八球黑八 九球杆 12.75mm', 'https://cdn.dianzaozao.com/billiard/cues/fury/9d5e54e0e6bba4af.jpg', '["https://cdn.dianzaozao.com/billiard/cues/fury/9d5e54e0e6bba4af.jpg"]', 'Fury威利AK入门级球杆。', 788.00, 999.00, 99, 156, 4.5, 0, 0, 0, 1, 83),
-- INVUI 英辉 (ID 19-21)
(1, 'INVUI(英辉) 中式黑8中头杆', '波茨杆 中八 16彩 11.5mm Z27', 'https://cdn.dianzaozao.com/billiard/cues/invui/399bf0b852dc657b.jpg', '["https://cdn.dianzaozao.com/billiard/cues/invui/399bf0b852dc657b.jpg"]', 'INVUI英辉中式黑8中头杆。', 1288.00, 1599.00, 99, 189, 4.8, 1, 0, 1, 1, 82),
(1, 'INVUI(英辉) 九球杆', '大头枫木 12.75mm 专业比赛款', 'https://cdn.dianzaozao.com/billiard/cues/other/c3b1dd05e3e70fde.jpg', '["https://cdn.dianzaozao.com/billiard/cues/other/c3b1dd05e3e70fde.jpg"]', 'INVUI英辉九球杆。', 1580.00, 1899.00, 99, 145, 4.7, 0, 1, 1, 1, 81),
(1, 'INVUI(英辉) 斯诺克球杆', '通杆小头 10mm 进口白蜡木', 'https://cdn.dianzaozao.com/billiard/cues/other/9e5a280e4f001678.jpg', '["https://cdn.dianzaozao.com/billiard/cues/other/9e5a280e4f001678.jpg"]', 'INVUI英辉斯诺克球杆。', 980.00, 1299.00, 99, 112, 4.6, 0, 0, 0, 1, 80),
-- NICHE 尼车 (ID 22-24)
(1, 'NICHE(尼车) 专业斯诺克球杆', '通杆 小头 10mm 进口白蜡木', 'https://cdn.dianzaozao.com/billiard/cues/niche/b7c181ab1e8d4866.jpg', '["https://cdn.dianzaozao.com/billiard/cues/niche/b7c181ab1e8d4866.jpg"]', 'NICHE尼车专业斯诺克球杆。', 1680.00, 1999.00, 198, 167, 4.8, 1, 0, 1, 1, 79),
(1, 'NICHE(尼车) 中式八球杆', '中头杆 11.5mm 红木后把', 'https://cdn.dianzaozao.com/billiard/cues/other/1316f2e5aa8c13f2.jpg', '["https://cdn.dianzaozao.com/billiard/cues/other/1316f2e5aa8c13f2.jpg"]', 'NICHE尼车中式八球杆。', 2280.00, 2699.00, 198, 178, 4.7, 0, 1, 1, 1, 78),
(1, 'NICHE(尼车) 九球杆', '大头枫木 12.75mm 专业比赛款', 'https://cdn.dianzaozao.com/billiard/cues/other/81fe96ca9cf89ed8.jpg', '["https://cdn.dianzaozao.com/billiard/cues/other/81fe96ca9cf89ed8.jpg"]', 'NICHE尼车九球杆。', 2680.00, 3199.00, 99, 89, 4.8, 0, 0, 1, 1, 77),
-- Dufferin 达芬尼 (ID 25-27)
(1, 'Dufferin(达芬尼) 九球杆', '大头枫木球杆 12.75mm 专业级', 'https://cdn.dianzaozao.com/billiard/cues/dufferin/c9d0f0ef1fed0fa3.jpg', '["https://cdn.dianzaozao.com/billiard/cues/dufferin/c9d0f0ef1fed0fa3.jpg"]', 'Dufferin达芬尼九球杆。', 2280.00, 2799.00, 99, 145, 4.8, 1, 0, 1, 1, 76),
(1, 'Dufferin(达芬尼) 中式八球杆', '中头杆 11mm 红木镶嵌', 'https://cdn.dianzaozao.com/billiard/cues/other/0a2a5714b034ddd8.jpg', '["https://cdn.dianzaozao.com/billiard/cues/other/0a2a5714b034ddd8.jpg"]', 'Dufferin达芬尼中式八球杆。', 1880.00, 2299.00, 99, 112, 4.7, 0, 0, 1, 1, 75),
(1, 'Dufferin(达芬尼) 斯诺克球杆', '通杆小头 10mm 乌木镶嵌', 'https://cdn.dianzaozao.com/billiard/cues/other/352853d9b5fd7ab4.jpg', '["https://cdn.dianzaozao.com/billiard/cues/other/352853d9b5fd7ab4.jpg"]', 'Dufferin达芬尼斯诺克球杆。', 3280.00, 3899.00, 198, 78, 4.9, 1, 1, 1, 1, 74),
-- John Parris JP (ID 28-30)
(1, 'John Parris(JP) 斯诺克球杆', '传统手工制作 小头 10mm', 'https://cdn.dianzaozao.com/billiard/cues/jp/4c9859f201ea5cda.jpg', '["https://cdn.dianzaozao.com/billiard/cues/jp/4c9859f201ea5cda.jpg"]', 'John Parris斯诺克球杆。', 5880.00, 6999.00, 99, 56, 5.0, 1, 0, 1, 1, 73),
(1, 'John Parris(JP) 经典系列球杆', '限量版 小头 9.8mm 乌木后把', 'https://cdn.dianzaozao.com/billiard/cues/other/b590744c3b45162c.jpg', '["https://cdn.dianzaozao.com/billiard/cues/other/b590744c3b45162c.jpg"]', 'John Parris经典系列球杆。', 8800.00, 9999.00, 99, 23, 5.0, 1, 1, 1, 1, 72),
(1, 'John Parris(JP) 入门级球杆', '学生款 小头 10mm 实用型', 'https://cdn.dianzaozao.com/billiard/cues/other/c7d8bea8961124ea.jpg', '["https://cdn.dianzaozao.com/billiard/cues/other/c7d8bea8961124ea.jpg"]', 'John Parris入门级球杆。', 2880.00, 3499.00, 198, 89, 4.7, 0, 0, 1, 1, 71),
-- Will Hunt 亨特 (ID 31-33)
(1, 'Will Hunt(亨特) 九球杆', '专业级 大头 12.75mm 进口枫木', 'https://cdn.dianzaozao.com/billiard/cues/will_hunt/9bf78d895a136373.jpg', '["https://cdn.dianzaozao.com/billiard/cues/will_hunt/9bf78d895a136373.jpg"]', 'Will Hunt亨特九球杆。', 3580.00, 4299.00, 99, 67, 4.8, 1, 0, 1, 1, 70),
(1, 'Will Hunt(亨特) 中式八球杆', '中头杆 11.5mm 白蜡木 专业款', 'https://cdn.dianzaozao.com/billiard/cues/other/cc44798b28d29881.jpg', '["https://cdn.dianzaozao.com/billiard/cues/other/cc44798b28d29881.jpg"]', 'Will Hunt亨特中式八球杆。', 2680.00, 3199.00, 99, 78, 4.7, 0, 0, 1, 1, 69),
(1, 'Will Hunt(亨特) 斯诺克球杆', '手工通杆 小头 10mm 乌木镶嵌', 'https://cdn.dianzaozao.com/billiard/cues/other/e5a9fc3c896abc40.jpg', '["https://cdn.dianzaozao.com/billiard/cues/other/e5a9fc3c896abc40.jpg"]', 'Will Hunt亨特斯诺克球杆。', 4280.00, 4999.00, 99, 45, 4.9, 1, 1, 1, 1, 68),
-- Stamford 斯坦福 (ID 34-36)
(1, 'Stamford(斯坦福) 经典球杆', '传统英式 小头 10mm 手工制作', 'https://cdn.dianzaozao.com/billiard/cues/stam_ford/ba66c2a348259e78.jpg', '["https://cdn.dianzaozao.com/billiard/cues/stam_ford/ba66c2a348259e78.jpg"]', 'Stamford斯坦福经典球杆。', 1880.00, 2299.00, 198, 134, 4.7, 0, 0, 1, 1, 67),
(1, 'Stamford(斯坦福) 九球杆', '大头枫木 12.75mm 专业比赛款', 'https://cdn.dianzaozao.com/billiard/cues/other/e901a458276aebf9.jpg', '["https://cdn.dianzaozao.com/billiard/cues/other/e901a458276aebf9.jpg"]', 'Stamford斯坦福九球杆。', 2480.00, 2999.00, 99, 67, 4.8, 0, 1, 1, 1, 66),
(1, 'Stamford(斯坦福) 入门球杆', '学生款 中头 11mm 经济实用', 'https://cdn.dianzaozao.com/billiard/cues/other/f185cce6712dc78e.jpg', '["https://cdn.dianzaozao.com/billiard/cues/other/f185cce6712dc78e.jpg"]', 'Stamford斯坦福入门球杆。', 1280.00, 1599.00, 99, 189, 4.5, 0, 0, 0, 1, 65),
-- 球盒/球袋类 (ID 37)
(4, 'Predator Urbain 球杆包', '高端硬壳设计 容纳2杆4前节', 'https://cdn.dianzaozao.com/billiard/cue_bag/00bf18b586e86873.jpg', '["https://cdn.dianzaozao.com/billiard/cue_bag/00bf18b586e86873.jpg"]', 'Predator Urbain系列球杆包。', 1280.00, 1580.00, 40, 89, 4.7, 1, 0, 1, 1, 64);

-- 3. 插入商品规格
INSERT INTO `product_specs` (`product_id`, `name`, `sort`) VALUES
(1, '颜色', 1), (2, '颜色', 1), (3, '颜色', 1), (4, '规格', 1), (5, '长度', 1),
(6, '前节直径', 1), (7, '后把材质', 1), (8, '重量', 1), (9, '皮头直径', 1), (10, '长度', 1),
(11, '长度', 1), (12, '前节直径', 1), (13, '重量', 1), (14, '套装内容', 1), (15, '版本', 1),
(16, '颜色', 1), (16, '皮头直径', 2), (17, '颜色', 1), (17, '皮头直径', 2), (18, '颜色', 1),
(19, '颜色', 1), (19, '皮头直径', 2), (20, '颜色', 1), (21, '颜色', 1),
(22, '颜色', 1), (22, '皮头直径', 2), (23, '颜色', 1), (24, '颜色', 1),
(25, '颜色', 1), (25, '皮头直径', 2), (26, '颜色', 1), (27, '颜色', 1),
(28, '颜色', 1), (28, '皮头直径', 2), (29, '颜色', 1), (30, '颜色', 1),
(31, '颜色', 1), (31, '皮头直径', 2), (32, '颜色', 1), (33, '颜色', 1),
(34, '颜色', 1), (34, '皮头直径', 2), (35, '颜色', 1), (36, '颜色', 1), (37, '颜色', 1);

-- 4. 插入规格值
INSERT INTO `product_spec_values` (`spec_id`, `value`, `sort`) VALUES
(1, '黑黄光把', 1), (1, '黑黄胶把', 2),
(2, '新款白色（光把）', 1), (2, '新款白色（胶把）', 2), (2, '红色（光把）', 3), (2, '红色胶把', 4), (2, '黄色（光把）', 5), (2, '黄色（胶把）', 6),
(3, 'AIR RUSH空气跳（光把）', 1), (3, 'AIR RUSH空气跳（胶把）', 2), (3, '30周年白色（光把）', 3), (3, '30周年白色（胶把）', 4),
(4, '11.5mm', 1), (4, '12mm', 2), (4, '13mm', 3),
(5, '107cm', 1), (5, '137cm', 2),
(6, '11.75mm', 1), (6, '12mm', 2), (6, '12.5mm', 3),
(7, '乌木', 1), (7, '紫檀木', 2), (7, '花梨木', 3),
(8, '18oz', 1), (8, '19oz', 2), (8, '20oz', 3), (8, '21oz', 4),
(9, '12mm', 1), (9, '12.5mm', 2), (9, '13mm', 3),
(10, '145cm', 1), (10, '148cm', 2),
(11, '57英寸', 1), (11, '58英寸', 2), (11, '59英寸', 3),
(12, '11.5mm', 1), (12, '12mm', 2),
(13, '19oz', 1), (13, '20oz', 2), (13, '21oz', 3),
(14, '球杆+硬盒', 1), (14, '球杆+软盒', 2),
(15, '大师签名版', 1), (15, '限量版', 2), (15, '珍藏版', 3),
(16, '黑色', 1), (16, '红色', 2), (17, '9.8mm', 1), (17, '10mm', 2),
(18, '线把', 1), (18, '皮把', 2), (19, '11.75mm', 1), (19, '12mm', 2),
(20, '线把', 1), (21, '黑色', 1), (21, '红色', 2), (22, '11.5mm', 1), (22, '12mm', 2),
(23, '棕色', 1), (23, '黑色', 2), (24, '10mm', 1),
(25, '红木色', 1), (25, '黑色', 2), (26, '11.5mm', 1), (26, '12mm', 2),
(27, '木色', 1), (27, '黑色', 2), (28, '12.75mm', 1),
(29, '木色', 1), (29, '黑色', 2), (30, '12.75mm', 1),
(31, '红木色', 1), (31, '黑色', 2), (32, '11mm', 1),
(33, '乌木色', 1), (33, '棕色', 2), (34, '10mm', 1),
(35, '棕色', 1), (35, '乌木色', 2), (36, '10mm', 1),
(37, '乌木色', 1), (38, '9.8mm', 1),
(39, '棕色', 1), (39, '黑色', 2), (40, '10mm', 1),
(41, '木色', 1), (41, '黑色', 2), (42, '12.75mm', 1),
(43, '棕色', 1), (43, '黑色', 2), (44, '11.5mm', 1),
(45, '乌木色', 1), (46, '10mm', 1),
(47, '棕色', 1), (47, '黑色', 2), (48, '10mm', 1),
(49, '木色', 1), (49, '黑色', 2), (50, '12.75mm', 1),
(51, '棕色', 1), (52, '11mm', 1),
(53, '黑色', 1);

-- 5. 插入商品SKU
INSERT INTO `product_skus` (`product_id`, `spec_value_ids`, `spec_text`, `price`, `original_price`, `stock`, `sales`, `status`) VALUES
(1, '1', '黑黄光把', 3014.00, 3500.00, 0, 45, 1),
(1, '2', '黑黄胶把', 3014.00, 3500.00, 99, 83, 1),
(2, '3', '新款白色（光把）', 2550.00, 3000.00, 99, 42, 1),
(2, '4', '新款白色（胶把）', 2550.00, 3000.00, 99, 45, 1),
(2, '5', '红色（光把）', 2550.00, 3000.00, 99, 38, 1),
(2, '6', '红色胶把', 2550.00, 3000.00, 99, 40, 1),
(2, '7', '黄色（光把）', 2550.00, 3000.00, 99, 48, 1),
(2, '8', '黄色（胶把）', 2550.00, 3000.00, 99, 43, 1),
(3, '9', 'AIR RUSH空气跳（光把）', 6171.00, 7000.00, 99, 25, 1),
(3, '10', 'AIR RUSH空气跳（胶把）', 6171.00, 7000.00, 99, 28, 1),
(3, '11', '30周年白色（光把）', 6171.00, 7000.00, 99, 18, 1),
(3, '12', '30周年白色（胶把）', 6171.00, 7000.00, 99, 18, 1),
(4, '13', '11.5mm', 19350.00, 22000.00, 99, 15, 1),
(4, '14', '12mm', 19350.00, 22000.00, 99, 18, 1),
(4, '15', '13mm', 19350.00, 22000.00, 99, 12, 1),
(5, '16', '107cm', 235.00, 299.00, 99, 280, 1),
(5, '17', '137cm', 235.00, 299.00, 99, 240, 1),
(6, '18', '11.75mm', 4500.00, 5200.00, 99, 52, 1),
(6, '19', '12mm', 4500.00, 5200.00, 99, 58, 1),
(6, '20', '12.5mm', 4500.00, 5200.00, 99, 46, 1),
(7, '21', '乌木', 6800.00, 7800.00, 99, 30, 1),
(7, '22', '紫檀木', 7200.00, 8200.00, 99, 28, 1),
(7, '23', '花梨木', 6500.00, 7500.00, 99, 31, 1),
(8, '24', '18oz', 3200.00, 3800.00, 99, 42, 1),
(8, '25', '19oz', 3200.00, 3800.00, 99, 45, 1),
(8, '26', '20oz', 3200.00, 3800.00, 99, 43, 1),
(8, '27', '21oz', 3200.00, 3800.00, 99, 38, 1),
(9, '28', '12mm', 2800.00, 3300.00, 99, 72, 1),
(9, '29', '12.5mm', 2800.00, 3300.00, 99, 68, 1),
(9, '30', '13mm', 2800.00, 3300.00, 99, 70, 1),
(10, '31', '145cm', 1200.00, 1500.00, 99, 195, 1),
(10, '32', '148cm', 1200.00, 1500.00, 99, 185, 1),
(11, '33', '57英寸', 3800.00, 4500.00, 99, 58, 1),
(11, '34', '58英寸', 3800.00, 4500.00, 99, 65, 1),
(11, '35', '59英寸', 3800.00, 4500.00, 99, 55, 1),
(12, '36', '11.5mm', 2500.00, 3000.00, 99, 125, 1),
(12, '37', '12mm', 2500.00, 3000.00, 99, 120, 1),
(13, '38', '19oz', 1800.00, 2200.00, 99, 52, 1),
(13, '39', '20oz', 1800.00, 2200.00, 99, 56, 1),
(13, '40', '21oz', 1800.00, 2200.00, 99, 48, 1),
(14, '41', '球杆+硬盒', 890.00, 1100.00, 99, 215, 1),
(14, '42', '球杆+软盒', 790.00, 1000.00, 99, 205, 1),
(15, '43', '大师签名版', 12000.00, 15000.00, 99, 10, 1),
(15, '44', '限量版', 12000.00, 15000.00, 99, 12, 1),
(15, '45', '珍藏版', 12000.00, 15000.00, 99, 10, 1),
(16, '46,48', '黑色,9.8mm', 719.00, 899.00, 99, 80, 1),
(16, '47,48', '红色,9.8mm', 719.00, 899.00, 99, 75, 1),
(16, '46,49', '黑色,10mm', 728.00, 899.00, 0, 45, 1),
(17, '50,52', '线把,11.75mm', 728.00, 899.00, 99, 68, 1),
(17, '51,52', '皮把,11.75mm', 788.00, 959.00, 99, 62, 1),
(18, '53', '线把,12.75mm', 788.00, 999.00, 99, 56, 1),
(19, '54,56', '黑色,11.5mm', 1288.00, 1599.00, 99, 95, 1),
(19, '55,56', '红色,11.5mm', 1288.00, 1599.00, 0, 45, 1),
(20, '57', '黑色,12.75mm', 1580.00, 1899.00, 99, 72, 1),
(21, '58', '棕色,10mm', 980.00, 1299.00, 99, 56, 1),
(22, '59,61', '棕色,10mm', 1680.00, 1999.00, 99, 84, 1),
(22, '60,61', '黑色,10mm', 1680.00, 1999.00, 99, 83, 1),
(23, '62,64', '红木色,11.5mm', 2280.00, 2699.00, 99, 89, 1),
(23, '63,64', '黑色,11.5mm', 2280.00, 2699.00, 99, 89, 1),
(24, '65', '木色,12.75mm', 2680.00, 3199.00, 99, 89, 1),
(25, '66,68', '木色,12.75mm', 2280.00, 2799.00, 99, 72, 1),
(25, '67,68', '黑色,12.75mm', 2280.00, 2799.00, 0, 35, 1),
(26, '69', '红木色,11mm', 1880.00, 2299.00, 99, 56, 1),
(27, '70,72', '乌木色,10mm', 3280.00, 3899.00, 99, 39, 1),
(27, '71,72', '棕色,10mm', 3280.00, 3899.00, 99, 39, 1),
(28, '73,75', '棕色,10mm', 5880.00, 6999.00, 99, 28, 1),
(28, '74,75', '乌木色,10mm', 6880.00, 7999.00, 99, 28, 1),
(29, '76', '乌木色,9.8mm', 8800.00, 9999.00, 99, 23, 1),
(30, '77,79', '棕色,10mm', 2880.00, 3499.00, 99, 45, 1),
(30, '78,79', '黑色,10mm', 2880.00, 3499.00, 99, 44, 1),
(31, '80,82', '木色,12.75mm', 3580.00, 4299.00, 99, 34, 1),
(31, '81,82', '黑色,12.75mm', 3580.00, 4299.00, 0, 16, 1),
(32, '83', '棕色,11.5mm', 2680.00, 3199.00, 99, 39, 1),
(33, '84', '乌木色,10mm', 4280.00, 4999.00, 99, 45, 1),
(34, '85,87', '棕色,10mm', 1880.00, 2299.00, 99, 67, 1),
(34, '86,87', '黑色,10mm', 1880.00, 2299.00, 99, 67, 1),
(35, '88,90', '木色,12.75mm', 2480.00, 2999.00, 99, 34, 1),
(36, '91', '棕色,11mm', 1280.00, 1599.00, 99, 95, 1),
(37, '92', '黑色', 1280.00, 1580.00, 40, 89, 1);

-- 6. 插入必备装备关联
INSERT INTO `essential_equipments` (`product_id`, `sort`, `status`) VALUES
(4, 1, 1), (7, 2, 1), (11, 3, 1), (15, 4, 1), (25, 5, 1), (28, 6, 1);

-- 7. 插入商品参数
INSERT INTO `product_attributes` (`product_id`, `name`, `value`, `sort`) VALUES
(1, '品牌', 'PREDATOR', 1), (1, '台球杆结构', '1/2分体球杆', 2), (1, '台球杆打法', '中式八球杆', 3),
(2, '品牌', 'PREDATOR', 1), (2, '产品系列', 'AIR2空气跳杆', 2),
(3, '品牌', 'PREDATOR', 1), (3, '材质', '碳纤维', 2),
(4, '品牌', 'PREDATOR', 1), (4, '前肢材质', '檀木枫木', 2), (4, '工艺', '镶嵌工艺', 3),
(5, '品牌', 'PREDATOR', 1), (5, '适用人群', '新手', 2),
(6, '品牌', 'Mezz', 1), (6, '产地', '日本', 2), (6, '前肢材质', '加拿大枫木', 3),
(7, '品牌', 'Mezz', 1), (7, '适用级别', '专业比赛', 2), (7, '前节材质', '碳纤维', 3),
(8, '品牌', 'Mezz', 1), (8, '用途', '冲杆', 2),
(9, '品牌', 'Mezz', 1), (9, '结构', '三节球杆', 2),
(10, '品牌', 'Mezz', 1), (10, '适用人群', '新手', 2),
(11, '品牌', 'Riley', 1), (11, '后把材质', '非洲乌木', 2), (11, '前节材质', '加拿大枫木', 3),
(12, '品牌', 'Riley', 1), (12, '材质', '高级檀木', 2),
(13, '品牌', 'Riley', 1), (13, '用途', '冲杆', 2),
(14, '品牌', 'Riley', 1), (14, '适用人群', '初学者', 2),
(15, '品牌', 'Riley', 1), (15, '特点', '大师签名', 2),
(16, '品牌', 'Fury(威利)', 1), (16, '台球杆结构', '3/4分体球杆', 2), (16, '台球杆打法', '中式八球', 3), (16, '前肢材质', '白蜡木', 4),
(17, '品牌', 'Fury(威利)', 1), (17, '台球杆结构', '1/2分体球杆', 2), (17, '台球杆打法', '九球', 3),
(18, '品牌', 'Fury(威利)', 1), (18, '台球杆结构', '分体球杆', 2), (18, '制作方式', '手工制作', 3),
(19, '品牌', 'INVUI(英辉)', 1), (19, '台球杆结构', '3/4分体球杆', 2), (19, '台球杆打法', '中式八球', 3),
(20, '品牌', 'INVUI(英辉)', 1), (20, '台球杆结构', '1/2分体球杆', 2), (20, '台球杆打法', '九球', 3),
(21, '品牌', 'INVUI(英辉)', 1), (21, '台球杆结构', '通杆', 2), (21, '台球杆打法', '斯诺克', 3),
(22, '品牌', 'NICHE(尼车)', 1), (22, '台球杆结构', '通杆', 2), (22, '台球杆打法', '斯诺克', 3),
(23, '品牌', 'NICHE(尼车)', 1), (23, '台球杆结构', '3/4分体球杆', 2), (23, '台球杆打法', '中式八球', 3),
(24, '品牌', 'NICHE(尼车)', 1), (24, '台球杆结构', '1/2分体球杆', 2), (24, '台球杆打法', '九球', 3),
(25, '品牌', 'Dufferin(达芬尼)', 1), (25, '台球杆结构', '1/2分体球杆', 2), (25, '台球杆打法', '九球', 3),
(26, '品牌', 'Dufferin(达芬尼)', 1), (26, '台球杆结构', '3/4分体球杆', 2), (26, '台球杆打法', '中式八球', 3),
(27, '品牌', 'Dufferin(达芬尼)', 1), (27, '台球杆结构', '通杆', 2), (27, '台球杆打法', '斯诺克', 3),
(28, '品牌', 'John Parris(JP)', 1), (28, '台球杆结构', '通杆', 2), (28, '台球杆打法', '斯诺克', 3), (28, '制作方式', '手工制作', 4),
(29, '品牌', 'John Parris(JP)', 1), (29, '台球杆结构', '通杆', 2), (29, '台球杆打法', '斯诺克', 3),
(30, '品牌', 'John Parris(JP)', 1), (30, '台球杆结构', '通杆', 2), (30, '台球杆打法', '斯诺克', 3),
(31, '品牌', 'Will Hunt(亨特)', 1), (31, '台球杆结构', '1/2分体球杆', 2), (31, '台球杆打法', '九球', 3),
(32, '品牌', 'Will Hunt(亨特)', 1), (32, '台球杆结构', '3/4分体球杆', 2), (32, '台球杆打法', '中式八球', 3),
(33, '品牌', 'Will Hunt(亨特)', 1), (33, '台球杆结构', '通杆', 2), (33, '台球杆打法', '斯诺克', 3),
(34, '品牌', 'Stamford(斯坦福)', 1), (34, '台球杆结构', '通杆', 2), (34, '台球杆打法', '斯诺克', 3),
(35, '品牌', 'Stamford(斯坦福)', 1), (35, '台球杆结构', '1/2分体球杆', 2), (35, '台球杆打法', '九球', 3),
(36, '品牌', 'Stamford(斯坦福)', 1), (36, '台球杆结构', '3/4分体球杆', 2), (36, '台球杆打法', '中式八球', 3),
(37, '品牌', 'Predator', 1), (37, '材质', '高端硬壳', 2), (37, '容量', '2杆4前节', 3);

-- +goose Down
TRUNCATE TABLE `product_attributes`;
TRUNCATE TABLE `essential_equipments`;
TRUNCATE TABLE `product_skus`;
TRUNCATE TABLE `product_spec_values`;
TRUNCATE TABLE `product_specs`;
TRUNCATE TABLE `products`;
TRUNCATE TABLE `banners`;
