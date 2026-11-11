-- +goose Up
-- 删除 match_achievements 残留的旧唯一键并修复被记错的按局特殊战绩。

-- 旧唯一键 uk_match_achievement (match_id, achievement_type) 必须删除。
-- 注意：information_schema.STATISTICS 按索引列展开，多列索引会有多行，
-- 所以这里只能用 COUNT(*) > 0 判断存在；旧迁移用 = 1 判断永远不会成立，导致旧键残留。
SET @idx_exists = (
  SELECT COUNT(*)
  FROM information_schema.STATISTICS
  WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'match_achievements' AND INDEX_NAME = 'uk_match_achievement'
);
SET @ddl = IF(
  @idx_exists > 0,
  'ALTER TABLE `match_achievements` DROP INDEX `uk_match_achievement`',
  'SELECT 1'
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

-- 防御性补建按 (match_id, actor, achievement_type) 的唯一键。
SET @idx_exists = (
  SELECT COUNT(*)
  FROM information_schema.STATISTICS
  WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'match_achievements' AND INDEX_NAME = 'uk_match_actor_achievement'
);
SET @ddl = IF(
  @idx_exists = 0,
  'ALTER TABLE `match_achievements` ADD UNIQUE KEY `uk_match_actor_achievement` (`match_id`, `actor`, `achievement_type`)',
  'SELECT 1'
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

-- 数据修复只覆盖按局记胜法的球种，并且只处理 match_rounds 里确实有特殊胜局的对局；
-- 没有特殊胜局记录的对局一行都不动，避免误删历史数据。
-- 先删除这些对局的特殊战绩行，再按局记录重新插入，可重复执行得到同样结果。
DELETE ma FROM `match_achievements` ma
JOIN `matches` m ON m.id = ma.match_id
WHERE m.game_type IN (2, 3, 4)
  AND ma.achievement_type IN ('break_clear', 'break_and_run', 'continue_clear', 'run_out', 'small_gold', 'golden_break', 'big_gold', 'nine_on_break')
  AND EXISTS (
    SELECT 1 FROM `match_rounds` mr
    WHERE mr.match_id = m.id
      AND mr.winner IN (1, 2)
      AND mr.win_type IN ('break_clear', 'break_and_run', 'continue_clear', 'run_out', 'small_gold', 'golden_break', 'big_gold', 'nine_on_break')
  );

-- 归一规则与代码里的 normalizeAchievementType 一致；斯诺克 break_50 / break_100 / break_147 不在本次修复范围。
INSERT INTO `match_achievements` (`match_id`, `actor`, `achievement_type`, `count`)
SELECT
  mr.match_id,
  mr.winner,
  CASE mr.win_type
    WHEN 'break_clear' THEN 'break_and_run'
    WHEN 'continue_clear' THEN 'run_out'
    WHEN 'small_gold' THEN 'golden_break'
    WHEN 'big_gold' THEN 'nine_on_break'
    ELSE mr.win_type
  END AS achievement_type,
  COUNT(*)
FROM `match_rounds` mr
JOIN `matches` m ON m.id = mr.match_id
WHERE m.game_type IN (2, 3, 4)
  AND mr.winner IN (1, 2)
  AND mr.win_type IN ('break_clear', 'break_and_run', 'continue_clear', 'run_out', 'small_gold', 'golden_break', 'big_gold', 'nine_on_break')
GROUP BY
  mr.match_id,
  mr.winner,
  CASE mr.win_type
    WHEN 'break_clear' THEN 'break_and_run'
    WHEN 'continue_clear' THEN 'run_out'
    WHEN 'small_gold' THEN 'golden_break'
    WHEN 'big_gold' THEN 'nine_on_break'
    ELSE mr.win_type
  END;

-- +goose Down
-- 不恢复旧唯一键 uk_match_achievement：恢复会让“同类型战绩记到错误选手行”的缺陷回来，
-- 且修复后的数据（同场同类型、不同 actor）可能直接违反旧唯一键。
SELECT 1;
