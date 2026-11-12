-- +goose Up
-- 补填首批回填时未核实的中文名，并更正首批里凭记忆填错的三个（均已查到中文报道）。
-- 补填只写仍为空的行，更正只改仍是旧值的行，不覆盖后台改动；可重复执行。
-- Liu Yang、Li Jun 为业余选手，只见英文赛果，未查到中文写法，继续留空由后台补填。
UPDATE `players` SET `name_zh` = '高阳' WHERE `source_type` = 'official' AND `name_zh` = '' AND (`display_name` = 'Gao Yang' OR CONCAT(`first_name`, ' ', `last_name`) = 'Gao Yang');
UPDATE `players` SET `name_zh` = '赵翰洋' WHERE `source_type` = 'official' AND `name_zh` = '' AND (`display_name` = 'Zhao Hanyang' OR CONCAT(`first_name`, ' ', `last_name`) = 'Zhao Hanyang');
UPDATE `players` SET `name_zh` = '邓壕辉' WHERE `source_type` = 'official' AND `name_zh` = '' AND (`display_name` = 'Deng Haohui' OR CONCAT(`first_name`, ' ', `last_name`) = 'Deng Haohui');
UPDATE `players` SET `name_zh` = '巩晨智' WHERE `source_type` = 'official' AND `name_zh` = '龚晨志';
UPDATE `players` SET `name_zh` = '王信伯' WHERE `source_type` = 'official' AND `name_zh` = '王欣博';
UPDATE `players` SET `name_zh` = '江俊' WHERE `source_type` = 'official' AND `name_zh` = '蒋俊';

-- +goose Down
UPDATE `players` SET `name_zh` = '龚晨志' WHERE `source_type` = 'official' AND `name_zh` = '巩晨智';
UPDATE `players` SET `name_zh` = '王欣博' WHERE `source_type` = 'official' AND `name_zh` = '王信伯';
UPDATE `players` SET `name_zh` = '蒋俊' WHERE `source_type` = 'official' AND `name_zh` = '江俊';
UPDATE `players` SET `name_zh` = '' WHERE `source_type` = 'official' AND `name_zh` IN ('高阳', '赵翰洋', '邓壕辉');
