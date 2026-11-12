-- +goose Up
-- 更正首批回填的中文名：Thepchaiya Un-Nooh 用国内常见译名「塔猜亚·乌诺」。
-- 只改仍是旧译名的行，不覆盖后台已改过的值；可重复执行。
UPDATE `players` SET `name_zh` = '塔猜亚·乌诺' WHERE `source_type` = 'official' AND `name_zh` = '特普查亚·乌诺';

-- +goose Down
UPDATE `players` SET `name_zh` = '特普查亚·乌诺' WHERE `source_type` = 'official' AND `name_zh` = '塔猜亚·乌诺';
