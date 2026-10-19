# 赛季档案闭环：MySQL 执行计划回归

对应变更：`openspec/changes/close-season-archive-loop`（任务 8.2）。
范围：名次计数、已结束赛季分页、本人历届记录三条新读路径的执行计划，以及既有热读取计划回归。

## 1. 执行环境

- 实例：本机 Homebrew MySQL 8.0.42，一次性数据目录，仅监听 `127.0.0.1:3307`，验收后停止。
- 库：`MYSQL_HOST=127.0.0.1 MYSQL_PORT=3307 MYSQL_USER=root`，测试自行创建并在结束时删除 `chasing_points_explain_*` 临时库。
- 命令（工作目录 `backend/`）：
  `RUN_MYSQL_EXPLAIN_TESTS=1 MYSQL_HOST=127.0.0.1 MYSQL_PORT=3307 MYSQL_USER=root MYSQL_PASSWORD= go test ./migrations -run TestMySQLHotReadPlans -count=1 -v`
- 未使用开发或生产库；未写入任何真实凭据。

## 2. 结果

| 用例 | 修复前 | 修复后 | 说明 |
| --- | --- | --- | --- |
| season live rank count | 失败：`uk_season_user_game` 扫描整个赛季 | 通过：`idx_season_game_rank` 覆盖范围扫描 | 代码改为 MySQL 下 `FORCE INDEX (idx_season_game_rank)` |
| settled season history | 失败：`idx_status` + filesort | 通过：`idx_seasons_end_id_status_start` 无 filesort | 代码改为 MySQL 下 `FORCE INDEX (idx_seasons_end_id_status_start)` |
| season history viewer records | 失败：`idx_user_id`（1 行） | 通过 | 两种索引均按本人记录有界，用例接受 `uk_season_user_game` 或 `idx_user_id` |
| challenge expiry | 失败：`idx_challenges_status_expires` | 通过 | 既有用例，非本次变更引入；两索引前缀相同，见第 3 节 |
| 其余 10 个既有热读取用例 | 通过 | 通过 | 包括 `due seasons` |

修复后：14 个用例全部通过，测试结束后临时库已删除。

## 3. 需要记录的事项

- `idx_challenges_status_expires_id`（`20260811102000_add_hot_read_indexes.sql` 新增）与既有 `idx_challenges_status_expires` 前缀相同，属于重复索引。用例现在两者都接受，但重复索引仍会增加写入成本，建议另起变更清理。
- 代码与测试使用相同的 `FORCE INDEX` 文本，仅在 MySQL 方言下生效，SQLite 单测不受影响。
- 通过 gorm 在该实例上验证：`ListSettledPage` 与 `CountAheadInSeason` 生成的 SQL 带有对应 hint；名次计数结果与原始 SQL 一致（同一测试数据下均为 5997）。
- EXPLAIN ANALYZE（season 1 含 10000 条记录）：名次计数修复前约 6.1ms，修复后约 1.6ms；已结束赛季分页修复前约 0.39ms（排序 500 行），修复后约 0.02ms。小表上的绝对收益很小，hint 主要保证计划稳定、成本不随赛季人数增长。

## 4. 未覆盖

- 本记录只覆盖执行计划与语义一致性，不替代任务 8.4（App 与微信小程序编译、真机浅色/深色检查）。
