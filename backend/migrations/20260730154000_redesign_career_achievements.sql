-- +goose Up
-- 生涯成就 V2：15 项通用里程碑 + 10 项球种专属绝技

INSERT INTO `achievements` (
  `key`,
  `name`,
  `description`,
  `icon`,
  `category`,
  `game_type`,
  `metric_key`,
  `progress_mode`,
  `threshold`,
  `reward_title_key`,
  `reward_title_name`,
  `sort`,
  `status`
) VALUES
  ('match_1', '初入战局', '累计完成 1 场有效比赛', '/static/images/achievements/match_1.png', 'match', 0, 'matches_total', 'sum', 1, '', '', 100, 1),
  ('match_10', '渐入佳境', '累计完成 10 场有效比赛', '/static/images/achievements/match_10.png', 'match', 0, 'matches_total', 'sum', 10, '', '', 110, 1),
  ('match_50', '五十征程', '累计完成 50 场有效比赛', '/static/images/achievements/match_50.png', 'match', 0, 'matches_total', 'sum', 50, '', '', 120, 1),
  ('match_100', '百战磨砺', '累计完成 100 场有效比赛', '/static/images/achievements/match_100.png', 'match', 0, 'matches_total', 'sum', 100, 'title_match_100', '资深球手', 130, 1),
  ('wins_1', '首战告捷', '累计赢下 1 场有效比赛', '/static/images/achievements/wins_1.png', 'wins', 0, 'wins_total', 'sum', 1, '', '', 200, 1),
  ('wins_10', '十胜起步', '累计赢下 10 场有效比赛', '/static/images/achievements/wins_10.png', 'wins', 0, 'wins_total', 'sum', 10, 'title_wins_10', '胜场新星', 210, 1),
  ('wins_50', '五十胜将', '累计赢下 50 场有效比赛', '/static/images/achievements/wins_50.png', 'wins', 0, 'wins_total', 'sum', 50, '', '', 220, 1),
  ('wins_100', '百胜丰碑', '累计赢下 100 场有效比赛', '/static/images/achievements/wins_100.png', 'wins', 0, 'wins_total', 'sum', 100, 'title_wins_100', '百胜名将', 230, 1),
  ('streak_3', '状态正佳', '个人最高连胜达到 3 场', '/static/images/achievements/streak_3.png', 'streak', 0, 'max_win_streak', 'max', 3, '', '', 300, 1),
  ('streak_5', '势如破竹', '个人最高连胜达到 5 场', '/static/images/achievements/streak_5.png', 'streak', 0, 'max_win_streak', 'max', 5, 'title_streak_5', '连胜猎手', 310, 1),
  ('streak_10', '十连制霸', '个人最高连胜达到 10 场', '/static/images/achievements/streak_10.png', 'streak', 0, 'max_win_streak', 'max', 10, 'title_streak_10', '连胜主宰', 320, 1),
  ('tournament_join_1', '赛事启程', '累计报名 1 场正式赛事', '/static/images/achievements/tournament_join_1.png', 'tournament', 0, 'tournament_join_total', 'sum', 1, '', '', 400, 1),
  ('tournament_finish_1', '初登赛场', '累计完成 1 场正式赛事', '/static/images/achievements/tournament_finish_1.png', 'tournament', 0, 'tournament_finish_total', 'sum', 1, '', '', 410, 1),
  ('tournament_finish_10', '久经赛场', '累计完成 10 场正式赛事', '/static/images/achievements/tournament_finish_10.png', 'tournament', 0, 'tournament_finish_total', 'sum', 10, 'title_tournament_finish_10', '赛事常客', 420, 1),
  ('tournament_champion_1', '初次登顶', '累计获得 1 次正式赛事冠军', '/static/images/achievements/tournament_champion_1.png', 'tournament', 0, 'tournament_champion_total', 'sum', 1, 'title_tournament_champion_1', '冠军球手', 430, 1),
  ('snooker_break_50_1', '半百一杆', '在斯诺克对局中打出 1 次单杆 50+', '/static/images/achievements/snooker_break_50_1.png', 'special', 1, 'break_50_total', 'sum', 1, '', '', 500, 1),
  ('snooker_break_100_1', '破百时刻', '在斯诺克对局中打出 1 次单杆 100+', '/static/images/achievements/snooker_break_100_1.png', 'special', 1, 'break_100_total', 'sum', 1, '', '', 510, 1),
  ('break_147_1', '满分时刻', '在斯诺克对局中打出 1 次单杆 147', '/static/images/achievements/break_147_1.png', 'special', 1, 'break_147_total', 'sum', 1, 'title_break_147_1', '满分王', 520, 1),
  ('chasing_golden_break_1', '小金初现', '在九球追分对局中打出 1 次小金', '/static/images/achievements/chasing_golden_break_1.png', 'special', 2, 'golden_break_total', 'sum', 1, '', '', 600, 1),
  ('chasing_nine_on_break_1', '大金降临', '在九球追分对局中打出 1 次大金', '/static/images/achievements/chasing_nine_on_break_1.png', 'special', 2, 'nine_on_break_total', 'sum', 1, '', '', 610, 1),
  ('continue_clear_1', '初次接清', '在中式八球对局中打出 1 次接清', '/static/images/achievements/continue_clear_1.png', 'special', 3, 'continue_clear_total', 'sum', 1, '', '', 700, 1),
  ('break_clear_1', '初次炸清', '在中式八球对局中打出 1 次炸清', '/static/images/achievements/break_clear_1.png', 'special', 3, 'break_clear_total', 'sum', 1, 'title_break_clear_1', '清台猎手', 710, 1),
  ('break_clear_20', '全台掌控', '在中式八球对局中累计打出 20 次炸清', '/static/images/achievements/break_clear_20.png', 'special', 3, 'break_clear_total', 'sum', 20, 'title_break_clear_20', '炸清大师', 720, 1),
  ('american_golden_break_1', '小金初现', '在美式九球对局中打出 1 次小金', '/static/images/achievements/american_golden_break_1.png', 'special', 4, 'golden_break_total', 'sum', 1, '', '', 800, 1),
  ('american_nine_on_break_1', '大金降临', '在美式九球对局中打出 1 次大金', '/static/images/achievements/american_nine_on_break_1.png', 'special', 4, 'nine_on_break_total', 'sum', 1, '', '', 810, 1)
ON DUPLICATE KEY UPDATE
  `name` = VALUES(`name`),
  `description` = VALUES(`description`),
  `icon` = VALUES(`icon`),
  `category` = VALUES(`category`),
  `game_type` = VALUES(`game_type`),
  `metric_key` = VALUES(`metric_key`),
  `progress_mode` = VALUES(`progress_mode`),
  `threshold` = VALUES(`threshold`),
  `reward_title_key` = VALUES(`reward_title_key`),
  `reward_title_name` = VALUES(`reward_title_name`),
  `sort` = VALUES(`sort`),
  `status` = VALUES(`status`);

UPDATE `user_titles` AS `ut`
JOIN `achievements` AS `a`
  ON `a`.`id` = `ut`.`source_ref_id`
  AND `a`.`reward_title_key` = `ut`.`title_key`
SET
  `ut`.`title_name` = `a`.`reward_title_name`,
  `ut`.`source_ref_name` = `a`.`name`
WHERE `ut`.`source_type` = 'achievement'
  AND `a`.`reward_title_key` <> '';

-- +goose Down
-- 保留已被用户资产引用的定义：新增项只停用，旧项恢复 V1 元数据。

UPDATE `achievements`
SET `status` = 0
WHERE `key` IN (
  'match_1',
  'match_50',
  'wins_1',
  'wins_50',
  'streak_3',
  'tournament_finish_1',
  'snooker_break_50_1',
  'snooker_break_100_1',
  'continue_clear_1',
  'chasing_golden_break_1',
  'chasing_nine_on_break_1',
  'american_golden_break_1',
  'american_nine_on_break_1'
);

INSERT INTO `achievements` (
  `key`,
  `name`,
  `description`,
  `icon`,
  `category`,
  `game_type`,
  `metric_key`,
  `progress_mode`,
  `threshold`,
  `reward_title_key`,
  `reward_title_name`,
  `sort`,
  `status`
) VALUES
  ('match_10', '初入战局', '累计完成 10 场有效比赛', '', 'match', 0, 'matches_total', 'sum', 10, '', '', 100, 1),
  ('match_100', '百战磨砺', '累计完成 100 场有效比赛', '', 'match', 0, 'matches_total', 'sum', 100, 'title_match_100', '资深球手', 110, 1),
  ('wins_10', '十胜起步', '累计赢下 10 场有效比赛', '', 'wins', 0, 'wins_total', 'sum', 10, 'title_wins_10', '胜场新星', 200, 1),
  ('wins_100', '百胜之路', '累计赢下 100 场有效比赛', '', 'wins', 0, 'wins_total', 'sum', 100, 'title_wins_100', '百胜球手', 210, 1),
  ('streak_5', '连胜猎手', '个人最高连胜达到 5 场', '', 'streak', 0, 'max_win_streak', 'max', 5, 'title_streak_5', '连胜猎手', 300, 1),
  ('streak_10', '不败锋芒', '个人最高连胜达到 10 场', '', 'streak', 0, 'max_win_streak', 'max', 10, 'title_streak_10', '不败王者', 310, 1),
  ('break_clear_1', '首次清台', '累计打出 1 次炸清', '', 'special', 0, 'break_clear_total', 'sum', 1, 'title_break_clear_1', '清台猎手', 400, 1),
  ('break_clear_20', '炸清大师', '累计打出 20 次炸清', '', 'special', 0, 'break_clear_total', 'sum', 20, 'title_break_clear_20', '炸清大师', 410, 1),
  ('break_147_1', '满分时刻', '累计打出 1 次单杆 147', '', 'special', 0, 'break_147_total', 'sum', 1, 'title_break_147_1', '满分王', 420, 1),
  ('tournament_join_1', '赛事初体验', '累计报名 1 场赛事', '', 'tournament', 0, 'tournament_join_total', 'sum', 1, '', '', 500, 1),
  ('tournament_finish_10', '赛事常客', '累计完成 10 场赛事', '', 'tournament', 0, 'tournament_finish_total', 'sum', 10, 'title_tournament_finish_10', '赛事常客', 510, 1),
  ('tournament_champion_1', '冠军球手', '累计获得 1 次赛事冠军', '', 'tournament', 0, 'tournament_champion_total', 'sum', 1, 'title_tournament_champion_1', '冠军球手', 520, 1)
ON DUPLICATE KEY UPDATE
  `name` = VALUES(`name`),
  `description` = VALUES(`description`),
  `icon` = VALUES(`icon`),
  `category` = VALUES(`category`),
  `game_type` = VALUES(`game_type`),
  `metric_key` = VALUES(`metric_key`),
  `progress_mode` = VALUES(`progress_mode`),
  `threshold` = VALUES(`threshold`),
  `reward_title_key` = VALUES(`reward_title_key`),
  `reward_title_name` = VALUES(`reward_title_name`),
  `sort` = VALUES(`sort`),
  `status` = VALUES(`status`);

UPDATE `user_titles` AS `ut`
JOIN `achievements` AS `a`
  ON `a`.`id` = `ut`.`source_ref_id`
  AND `a`.`reward_title_key` = `ut`.`title_key`
SET
  `ut`.`title_name` = `a`.`reward_title_name`,
  `ut`.`source_ref_name` = `a`.`name`
WHERE `ut`.`source_type` = 'achievement'
  AND `a`.`reward_title_key` <> '';
