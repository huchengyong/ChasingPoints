-- +goose Up
-- 插入模拟用户数据
INSERT INTO `users` (`id`, `phone`, `nickname`, `avatar`, `status`, `created_at`, `updated_at`, `deleted_at`) VALUES (10001, '13800006501', '南阳周栋分栋', 'https://cdn.dianzaozao.com/avatars/224e6c82b9232f035a46e27aac04b719.jpg', 1, '2025-12-11 20:20:11', '2025-12-13 14:51:57', NULL);
INSERT INTO `users` (`id`, `phone`, `nickname`, `avatar`, `status`, `created_at`, `updated_at`, `deleted_at`) VALUES (10002, '13800003802', '奥沙利勇', 'https://cdn.dianzaozao.com/avatars/52fb4ac7de25c23bad14bd7dcde143a6.jpg', 1, '2025-12-11 20:20:11', '2025-12-13 14:52:03', NULL);
INSERT INTO `users` (`id`, `phone`, `nickname`, `avatar`, `status`, `created_at`, `updated_at`, `deleted_at`) VALUES (10003, '13800000293', '我不是菜鸟', 'https://cdn.dianzaozao.com/avatars/884d366623f697f7f08f205b320654f2.jpg', 1, '2025-12-11 20:20:11', '2025-12-13 14:52:14', NULL);
INSERT INTO `users` (`id`, `phone`, `nickname`, `avatar`, `status`, `created_at`, `updated_at`, `deleted_at`) VALUES (10004, '13800009014', 'TimesNewRoman', 'https://cdn.dianzaozao.com/avatars/b755bd5a52f713b21cc42f1b64c35699.jpg', 1, '2025-12-11 20:20:11', '2025-12-13 14:52:22', NULL);
INSERT INTO `users` (`id`, `phone`, `nickname`, `avatar`, `status`, `created_at`, `updated_at`, `deleted_at`) VALUES (10005, '13800000125', 'PerfectIsShit', 'https://cdn.dianzaozao.com/avatars/bade4fa34bde25c242a1ef172509147e.jpg', 1, '2025-12-11 20:20:11', '2025-12-13 14:52:29', NULL);
INSERT INTO `users` (`id`, `phone`, `nickname`, `avatar`, `status`, `created_at`, `updated_at`, `deleted_at`) VALUES (10006, '13800003306', '放你1金1', 'https://cdn.dianzaozao.com/avatars/bdf56f242b7f6918041f825318f108bb.jpg', 1, '2025-12-11 20:20:11', '2025-12-13 14:52:31', NULL);
INSERT INTO `users` (`id`, `phone`, `nickname`, `avatar`, `status`, `created_at`, `updated_at`, `deleted_at`) VALUES (10007, '13800001437', '别跟我来这套', 'https://cdn.dianzaozao.com/avatars/c7c960e3339078e65784f118f981cc5d.jpg', 1, '2025-12-11 20:20:11', '2025-12-13 14:52:45', NULL);
INSERT INTO `users` (`id`, `phone`, `nickname`, `avatar`, `status`, `created_at`, `updated_at`, `deleted_at`) VALUES (10008, '13800002908', '最强塞大师', 'https://cdn.dianzaozao.com/avatars/cc3d20d37ebc698210877b313d6ae5e7.jpg', 1, '2025-12-11 20:20:11', '2025-12-13 14:52:52', NULL);
INSERT INTO `users` (`id`, `phone`, `nickname`, `avatar`, `status`, `created_at`, `updated_at`, `deleted_at`) VALUES (10009, '13800006659', '浦东奥沙利文', 'https://cdn.dianzaozao.com/avatars/dd05babb41762ba2e71da161b286973d.jpg', 1, '2025-12-11 20:20:11', '2025-12-13 14:53:01', NULL);
INSERT INTO `users` (`id`, `phone`, `nickname`, `avatar`, `status`, `created_at`, `updated_at`, `deleted_at`) VALUES (10010, '13800003110', '高杆右塞', 'https://cdn.dianzaozao.com/avatars/e26245ce55dc958e9e610a69fa2aea34.jpg', 1, '2025-12-11 20:20:11', '2025-12-13 14:53:04', NULL);

-- 插入模拟用户段位数据
INSERT INTO `user_ranking` (`user_id`, `rank_score`, `rank_level`, `total_wins`, `total_losses`, `current_streak`, `max_streak`, `created_at`, `updated_at`) VALUES
(10001, 2300, 5, 80, 10, 5, 10, NOW(), NOW()), -- 钻石王者
(10002, 2100, 5, 75, 15, 3, 8, NOW(), NOW()),
(10003, 1900, 4, 60, 20, 2, 6, NOW(), NOW()), -- 铂金大师
(10004, 1700, 4, 55, 25, 1, 5, NOW(), NOW()),
(10005, 1200, 3, 40, 30, 0, 3, NOW(), NOW()), -- 黄金球手
(10006, 1050, 3, 35, 32, 0, 2, NOW(), NOW()),
(10007, 700, 2, 20, 40, 0, 1, NOW(), NOW()), -- 白银球手
(10008, 600, 2, 15, 45, 0, 1, NOW(), NOW()),
(10009, 300, 1, 10, 50, 0, 0, NOW(), NOW()), -- 青铜球手
(10010, 100, 1, 5, 55, 0, 0, NOW(), NOW());

-- +goose Down
-- 删除模拟用户段位数据
DELETE FROM `user_ranking` WHERE `user_id` BETWEEN 10001 AND 10010;
-- 删除模拟用户数据
DELETE FROM `users` WHERE `id` BETWEEN 10001 AND 10010;