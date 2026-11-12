-- +goose Up
-- 用户提供的两位业余选手中文名（此前只查到英文赛果，未能核实）。只写仍为空的行，可重复执行。
UPDATE `players` SET `name_zh` = '刘洋' WHERE `source_type` = 'official' AND `name_zh` = '' AND (`display_name` = 'Liu Yang' OR CONCAT(`first_name`, ' ', `last_name`) = 'Liu Yang');
UPDATE `players` SET `name_zh` = '李俊' WHERE `source_type` = 'official' AND `name_zh` = '' AND (`display_name` = 'Li Jun' OR CONCAT(`first_name`, ' ', `last_name`) = 'Li Jun');

-- +goose Down
UPDATE `players` SET `name_zh` = '' WHERE `source_type` = 'official' AND `name_zh` IN ('刘洋', '李俊') AND `display_name` IN ('Liu Yang', 'Li Jun');
