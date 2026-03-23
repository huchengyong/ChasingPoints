-- +goose Up
INSERT INTO `achievement_reward_config` (`game_type`, `achievement_type`, `name`, `reward_score`) VALUES
(4, 'golden_break', '小金', 10),
(4, 'nine_on_break', '大金', 20);

-- +goose Down
DELETE FROM `achievement_reward_config`
WHERE `game_type` = 4
  AND `achievement_type` IN ('golden_break', 'nine_on_break');
