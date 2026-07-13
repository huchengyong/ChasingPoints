-- +goose Up
-- 用户段位表
CREATE TABLE IF NOT EXISTS `user_ranking` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `user_id` bigint unsigned NOT NULL COMMENT '用户ID',
  `game_type` TINYINT NOT NULL DEFAULT 3 COMMENT '球种 1=斯诺克 2=九球追分 3=中式八球 4=美式九球',
  `rank_score` int NOT NULL DEFAULT 0 COMMENT '排位分',
  `rank_level` tinyint NOT NULL DEFAULT 1 COMMENT '段位等级 1-5',
  `total_wins` int NOT NULL DEFAULT 0 COMMENT '总胜场',
  `total_losses` int NOT NULL DEFAULT 0 COMMENT '总败场',
  `current_streak` int NOT NULL DEFAULT 0 COMMENT '当前连胜(负则为负数)',
  `max_streak` int NOT NULL DEFAULT 0 COMMENT '最高连胜',
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uniq_user_game_type` (`user_id`, `game_type`),
  KEY `idx_game_type_rank_score` (`game_type`, `rank_score`, `total_wins`, `updated_at`, `user_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='用户段位表';

-- 段位配置表 (存储段位基础配置)
CREATE TABLE IF NOT EXISTS `rank_config` (
  `id` tinyint unsigned NOT NULL AUTO_INCREMENT,
  `level` tinyint NOT NULL COMMENT '段位等级',
  `name` varchar(20) NOT NULL COMMENT '段位名称',
  `icon` varchar(100) NOT NULL COMMENT '段位图标',
  `min_score` int NOT NULL COMMENT '晋级最低分',
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_level` (`level`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='段位配置表';

-- 初始化段位配置数据
INSERT INTO `rank_config` (`level`, `name`, `icon`, `min_score`) VALUES
(1, '青铜球手', '/static/images/ranks/rank_bronze.png', 0),
(2, '白银球手', '/static/images/ranks/rank_silver.png', 500),
(3, '黄金球手', '/static/images/ranks/rank_gold.png', 1000),
(4, '铂金大师', '/static/images/ranks/rank_platinum.png', 1500),
(5, '钻石王者', '/static/images/ranks/rank_diamond.png', 2000);

-- 特殊战绩奖励配置表
CREATE TABLE IF NOT EXISTS `achievement_reward_config` (
  `id` int unsigned NOT NULL AUTO_INCREMENT,
  `game_type` tinyint NOT NULL COMMENT '游戏类型 1=斯诺克 2=九球追分 3=中式八球 4=美式九球',
  `achievement_type` varchar(30) NOT NULL COMMENT '成就类型代码',
  `name` varchar(50) NOT NULL COMMENT '成就名称',
  `reward_score` int NOT NULL COMMENT '奖励排位分',
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_game_achievement` (`game_type`, `achievement_type`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='特殊战绩奖励配置表';

-- 初始化特殊战绩奖励配置
INSERT INTO `achievement_reward_config` (`game_type`, `achievement_type`, `name`, `reward_score`) VALUES
-- 斯诺克
(1, 'break_50', '单杆50+', 5),
(1, 'break_100', '单杆100+', 10),
(1, 'break_147', '单杆147', 50),
-- 九球
(2, 'golden_break', '小金', 10),
(2, 'nine_on_break', '大金', 20),
-- 中式八球
(3, 'run_out', '接清', 10),
(3, 'break_and_run', '炸清', 15),
-- 美式九球
(4, 'golden_break', '小金', 10),
(4, 'nine_on_break', '大金', 20);

-- +goose Down
DROP TABLE IF EXISTS `achievement_reward_config`;
DROP TABLE IF EXISTS `rank_config`;
DROP TABLE IF EXISTS `user_ranking`;
