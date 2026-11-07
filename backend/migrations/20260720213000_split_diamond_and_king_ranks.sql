-- +goose Up

UPDATE `rank_config`
SET
  `name` = '钻石',
  `icon` = '/static/images/ranks/rank_diamond.png',
  `min_score` = 2000
WHERE `level` = 5;

INSERT INTO `rank_config` (`level`, `name`, `icon`, `min_score`) VALUES
(6, '王者', '/static/images/ranks/rank_king.png', 2500)
ON DUPLICATE KEY UPDATE
  `name` = VALUES(`name`),
  `icon` = VALUES(`icon`),
  `min_score` = VALUES(`min_score`);

UPDATE `user_ranking`
SET `rank_level` = CASE
  WHEN `rank_score` >= 2500 THEN 6
  WHEN `rank_score` >= 2000 THEN 5
  WHEN `rank_score` >= 1500 THEN 4
  WHEN `rank_score` >= 1000 THEN 3
  WHEN `rank_score` >= 500 THEN 2
  ELSE 1
END;

-- +goose Down

UPDATE `user_ranking`
SET `rank_level` = CASE
  WHEN `rank_score` >= 2000 THEN 5
  WHEN `rank_score` >= 1500 THEN 4
  WHEN `rank_score` >= 1000 THEN 3
  WHEN `rank_score` >= 500 THEN 2
  ELSE 1
END;

DELETE FROM `rank_config`
WHERE `level` = 6;

UPDATE `rank_config`
SET
  `name` = '钻石王者',
  `icon` = '/static/images/ranks/rank_diamond.png',
  `min_score` = 2000
WHERE `level` = 5;
