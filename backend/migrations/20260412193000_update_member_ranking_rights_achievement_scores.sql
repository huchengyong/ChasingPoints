-- +goose Up

INSERT INTO `achievement_reward_config` (`game_type`, `achievement_type`, `name`, `reward_score`) VALUES
(1, 'break_50', '单杆50+', 8),
(1, 'break_100', '单杆100+', 16),
(1, 'break_147', '单杆147', 30),
(2, 'golden_break', '小金', 4),
(2, 'nine_on_break', '大金', 6),
(3, 'run_out', '接清', 4),
(3, 'break_and_run', '炸清', 6),
(4, 'golden_break', '小金', 4),
(4, 'nine_on_break', '大金', 6)
ON DUPLICATE KEY UPDATE
`name` = VALUES(`name`),
`reward_score` = VALUES(`reward_score`);

-- +goose Down

INSERT INTO `achievement_reward_config` (`game_type`, `achievement_type`, `name`, `reward_score`) VALUES
(1, 'break_50', '单杆50+', 5),
(1, 'break_100', '单杆100+', 10),
(1, 'break_147', '单杆147', 50),
(2, 'golden_break', '小金', 10),
(2, 'nine_on_break', '大金', 20),
(3, 'run_out', '接清', 10),
(3, 'break_and_run', '炸清', 15),
(4, 'golden_break', '小金', 10),
(4, 'nine_on_break', '大金', 20)
ON DUPLICATE KEY UPDATE
`name` = VALUES(`name`),
`reward_score` = VALUES(`reward_score`);
