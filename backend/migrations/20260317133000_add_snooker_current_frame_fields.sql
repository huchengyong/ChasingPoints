-- +goose Up
ALTER TABLE `matches`
  ADD COLUMN `current_frame_my_score` int NOT NULL DEFAULT 0 COMMENT '当前frame我方分数' AFTER `opponent_score`,
  ADD COLUMN `current_frame_opponent_score` int NOT NULL DEFAULT 0 COMMENT '当前frame对手分数' AFTER `current_frame_my_score`,
  ADD COLUMN `current_frame_started` tinyint(1) NOT NULL DEFAULT 0 COMMENT '当前frame是否已开始' AFTER `current_frame_opponent_score`;

INSERT INTO `match_rounds` (`match_id`, `round_no`, `my_score`, `opponent_score`, `winner`, `win_type`, `created_at`)
SELECT
  m.`id`,
  1,
  m.`my_score`,
  m.`opponent_score`,
  CASE
    WHEN m.`result` = 1 THEN 1
    WHEN m.`result` = 2 THEN 2
    ELSE NULL
  END,
  'legacy_single_frame',
  COALESCE(m.`end_time`, m.`match_time`)
FROM `matches` m
WHERE m.`game_type` = 1
  AND m.`status` = 2
  AND NOT EXISTS (
    SELECT 1 FROM `match_rounds` mr WHERE mr.`match_id` = m.`id`
  );

UPDATE `matches`
SET
  `current_frame_my_score` = `my_score`,
  `current_frame_opponent_score` = `opponent_score`,
  `current_frame_started` = CASE WHEN `status` = 1 THEN 1 ELSE 0 END,
  `my_score` = CASE
    WHEN `status` = 2 AND `result` = 1 THEN 1
    ELSE 0
  END,
  `opponent_score` = CASE
    WHEN `status` = 2 AND `result` = 2 THEN 1
    ELSE 0
  END
WHERE `game_type` = 1;

UPDATE `matches`
SET `current_frame_started` = CASE WHEN `status` = 1 THEN 1 ELSE 0 END
WHERE `game_type` <> 1;

-- +goose Down
UPDATE `matches` m
LEFT JOIN (
  SELECT
    mr.`match_id`,
    MAX(CASE WHEN mr.`win_type` IN ('legacy_single_frame', 'manual_single_frame') THEN mr.`my_score` END) AS `restore_my_score`,
    MAX(CASE WHEN mr.`win_type` IN ('legacy_single_frame', 'manual_single_frame') THEN mr.`opponent_score` END) AS `restore_opponent_score`
  FROM `match_rounds` mr
  GROUP BY mr.`match_id`
) r ON r.`match_id` = m.`id`
SET
  m.`my_score` = CASE
    WHEN m.`status` = 1 THEN m.`current_frame_my_score`
    WHEN r.`restore_my_score` IS NOT NULL THEN r.`restore_my_score`
    ELSE m.`my_score`
  END,
  m.`opponent_score` = CASE
    WHEN m.`status` = 1 THEN m.`current_frame_opponent_score`
    WHEN r.`restore_opponent_score` IS NOT NULL THEN r.`restore_opponent_score`
    ELSE m.`opponent_score`
  END
WHERE m.`game_type` = 1;

DELETE mr
FROM `match_rounds` mr
INNER JOIN `matches` m ON m.`id` = mr.`match_id`
WHERE m.`game_type` = 1
  AND mr.`win_type` = 'legacy_single_frame';

ALTER TABLE `matches`
  DROP COLUMN `current_frame_started`,
  DROP COLUMN `current_frame_opponent_score`,
  DROP COLUMN `current_frame_my_score`;
