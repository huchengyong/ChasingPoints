-- +goose Up
-- 初始化首版闭环所需的最小成就定义

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

-- +goose Down
DELETE FROM `achievements`
WHERE `key` IN (
  'match_10',
  'match_100',
  'wins_10',
  'wins_100',
  'streak_5',
  'streak_10',
  'break_clear_1',
  'break_clear_20',
  'break_147_1',
  'tournament_join_1',
  'tournament_finish_10',
  'tournament_champion_1'
);
