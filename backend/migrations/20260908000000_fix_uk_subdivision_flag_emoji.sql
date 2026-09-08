-- 历史数据修复：修正英格兰、苏格兰、威尔士选手的 flag_emoji
--
-- 背景：旧版 buildFlagEmoji 对 gb-eng/gb-sct/gb-wls 生成的 tag 序列缺少 gb 前缀（如 eng 而非 gbeng），
-- 导致 Emoji 渲染为黑旗。前端已改为根据 country_code 使用本地图片，正常同步也会在下次更新时自动修复。
-- 本迁移目标：主动修复已有数据，确保 flag_emoji 列一致性。
--
-- 执行方式（通过 goose）：
--   cd backend && ./goose.sh up
-- 或直接执行 SQL（先 dry-run 预览）：
--   mysql> SOURCE backend/migrations/20260408000000_fix_uk_subdivision_flag_emoji.sql;
--
-- 幂等：已正确的行不会被更新（WHERE 条件排除已匹配的 flag_emoji）。
-- 仅修正目标地区（gb-eng / gb-sct / gb-wls），不触碰其他选手字段。
-- +goose Up
-- England: 🏴 + tag_gbeng + cancel
UPDATE players
SET flag_emoji = _utf8mb4 X'F09F8FB4F3A081A7F3A081A2F3A081A5F3A081AEF3A081A7F3A081BF'
WHERE LOWER(country_code) = 'gb-eng'
  AND (flag_emoji IS NULL OR flag_emoji = '' OR flag_emoji != _utf8mb4 X'F09F8FB4F3A081A7F3A081A2F3A081A5F3A081AEF3A081A7F3A081BF');

-- Scotland: 🏴 + tag_gbsct + cancel
UPDATE players
SET flag_emoji = _utf8mb4 X'F09F8FB4F3A081A7F3A081A2F3A081B3F3A081A3F3A081B4F3A081BF'
WHERE LOWER(country_code) = 'gb-sct'
  AND (flag_emoji IS NULL OR flag_emoji = '' OR flag_emoji != _utf8mb4 X'F09F8FB4F3A081A7F3A081A2F3A081B3F3A081A3F3A081B4F3A081BF');

-- Wales: 🏴 + tag_gbwls + cancel
UPDATE players
SET flag_emoji = _utf8mb4 X'F09F8FB4F3A081A7F3A081A2F3A081B7F3A081ACF3A081B3F3A081BF'
WHERE LOWER(country_code) = 'gb-wls'
  AND (flag_emoji IS NULL OR flag_emoji = '' OR flag_emoji != _utf8mb4 X'F09F8FB4F3A081A7F3A081A2F3A081B7F3A081ACF3A081B3F3A081BF');

-- 预览（dry-run）：执行以下 SELECT 查看受影响行数
-- SELECT COUNT(*) AS affected_rows FROM players
-- WHERE LOWER(country_code) IN ('gb-eng', 'gb-sct', 'gb-wls')
--   AND (flag_emoji IS NULL OR flag_emoji = ''
--     OR (LOWER(country_code) = 'gb-eng' AND flag_emoji != _utf8mb4 X'F09F8FB4F3A081A7F3A081A2F3A081A5F3A081AEF3A081A7F3A081BF')
--     OR (LOWER(country_code) = 'gb-sct' AND flag_emoji != _utf8mb4 X'F09F8FB4F3A081A7F3A081A2F3A081B3F3A081A3F3A081B4F3A081BF')
--     OR (LOWER(country_code) = 'gb-wls' AND flag_emoji != _utf8mb4 X'F09F8FB4F3A081A7F3A081A2F3A081B7F3A081ACF3A081B3F3A081BF')
--   );

-- +goose Down
-- 撤销：将英格兰、苏格兰、威尔士选手的 flag_emoji 清空（无法恢复旧错误值，清空为干净的撤销行为）
UPDATE players
SET flag_emoji = ''
WHERE LOWER(country_code) IN ('gb-eng', 'gb-sct', 'gb-wls');