# 排位规则 V2：自动化验收记录

对应变更：`openspec/changes/rebalance-ranked-rating`（任务 9.x）。记录日期：2026-10-19。

## 1. 执行环境

- MySQL：本机 Homebrew MySQL 8.0.42，在会话临时目录新建一次性数据目录，`--no-defaults` 启动，仅监听 `127.0.0.1:33099`，root 无密码；验收结束后停止实例并删除数据目录。未连接开发库或生产库，未读取 `backend/.env`。
- 本机已在运行的 MySQL（开发者自己的数据目录）未被访问。
- `cmd/rank_rebuild` 用占位凭据的干净环境运行（`env -i`，工作目录不含 `.env`），`MYSQL_DSN` 指向上述临时库，Redis 指向不存在的端口（命令会打印连接失败日志，读取会员权益配置时回退默认值）。

## 2. 自动化结果

| 范围 | 命令（工作目录） | 结果 |
| --- | --- | --- |
| 后端全量 | `go vet ./...`、`go build .`、`go test ./...`（`backend/`） | 通过，55 个包 ok |
| 计分纯函数 | `go test ./internal/ranking/` | 通过；规格中的分值示例（±45、+68/−21、定级 +90/−23、同对手 ±14、封顶 +20）均有断言 |
| 线上结算 = 离线重放 | `go test ./internal/logic/ -run TestRatingV2LiveSettlementMatchesReplay` | 通过；覆盖同对手衰减、第 7 场不计分、每日封顶、定级、跨天、跨球种、平局计入同对手场次 |
| MySQL 执行计划 | `RUN_MYSQL_EXPLAIN_TESTS=1 … go test ./migrations -run TestMySQLHotReadPlans` | 15 个用例通过；赛季榜与名次计数走 `idx_season_game_qualified_rank` 且无 filesort，切换快路径走 `idx_user_ranking_rating_version` |
| MySQL 读模型回建 | `RUN_MYSQL_REBUILD_TESTS=1 … -run TestMySQLCompetitiveReadRebuildCheckpointAndAudit` | 通过（未配置 V2 时投影新增的达标逻辑不生效，回建结果与之前一致） |
| MySQL V2 验收 | `RUN_MYSQL_RATING_V2_TESTS=1 … -run TestMySQLRatingV2SettlementQualificationAndAudit` | 通过：行锁结算、就地切换只写一次 `rating_reset`、达标维护、达标优先榜单、Worker 切换、重放审计差异 0 |
| 迁移可回滚 | 临时库 `goose up` → `goose down` → `goose up` | 回滚后新列与新索引消失、`idx_season_game_rank` 恢复、`rank_reset_ratio` 默认值回到 0.70；重新执行后恢复 V2 结构；`rank_rating_seeds` 未指定字符集，沿用库默认 |
| 种子命令 | `rank_rebuild -mode=seed`（预览）与 `-apply` | 1,200 场合成历史中重放截至当前时间的 1,140 场，生成 40 条初始分；预览不写库，`-apply` 只写 `rank_rating_seeds`，`rank_change_logs` 与 `user_ranking` 不变 |
| 命令拒绝路径 | `-mode=audit`（起点前）、`-mode=seed`（起点后）、起点为月中 | 分别拒绝："尚未生效""切换可能已经开始""不是赛季起点" |
| UniApp | `node --test tests/*.test.mjs`（`uniapp/`） | 780 项通过 |
| 管理后台 | `node --experimental-strip-types --test tests/*.test.mjs`、`npm run build`（`admin/`） | 6 项通过，构建成功（既有的 chunk 体积警告） |
| 空白检查 | `git diff --check` | 通过 |

## 3. 未执行 / 需要人工确认

- 生产种子预览与写入已在 2026-10-19 执行：纳入 1 场已完成排位，写入 2 条种子。当前数据量太小，无法据此校准段位分布的阈值，需在数据较多时复核。
- 未做 App 与微信小程序编译及真机检查：段位说明页（规则版本、定级提示）、赛季页（达标进度、未达标标注、本季净变化）、历届赛季与报告的"未达标"、战报高光卡片、会员中心排位权益区，浅色/深色都需要人工确认。
- 原生 iOS 无改动、未构建：战报按 `label/value` 通用渲染排位明细，新增字段不参与解码。
- 上线后首日需在目标环境执行 `-mode=audit` 并确认 `differences=0`。

## 4. 线上库版本与 MySQL 5.7 兼容验证

- 线上库为 MySQL 5.7.44（只读确认）。开发库为 8.0.45，前面第 2 节的 8.0 验收不能代替 5.7 验收。
- 本地演练：`mysql:5.7` 容器（5.7.44，默认 sql_mode，仅绑定 127.0.0.1），goose 使用发布脚本同款 v3.28.0。
- 从空库迁移到线上当前版本 `20261016100000`：成功，说明整条迁移链在 5.7 上可执行。
- 执行 `20261019120000`：成功。结构核对通过：`user_ranking.rating_version`（默认 1）、`rated_matches`（默认 0）、`season_records.distinct_opponents` 与 `qualified`、`idx_season_game_qualified_rank` 存在且 `idx_season_game_rank` 已删除、`rank_rating_seeds` 存在、`seasons.rank_reset_ratio` 默认 1.00。
- `down` 后结构恢复（`rating_version` 列消失、`idx_season_game_rank` 恢复、`rank_reset_ratio` 默认回到 0.70）；再次 `up` 成功。
- 仓库 MySQL 验收测试在 5.7 上的结果：
  - `TestMySQLRatingV2SettlementQualificationAndAudit`：通过。
  - `TestMySQLCompetitiveReadRebuildCheckpointAndAudit`：通过。
  - `TestMySQLHotReadPlans`：失败在造数据步骤。该测试的造数据使用 `cte_max_recursion_depth` 与 `WITH RECURSIVE`，这是 MySQL 8.0 语法，5.7 无法执行；HEAD（53eb244）中已存在，不是本次引入。
- 5.7 上的执行计划（1 万行合成数据，造数据不使用递归 CTE，手工 EXPLAIN）：
  - 赛季榜单（`ORDER BY qualified DESC, end_rank_score DESC, wins DESC, id DESC LIMIT 20`）：`idx_season_game_qualified_rank`，无 filesort。
  - 名次计数：`idx_season_game_qualified_rank` 范围扫描，`Using index`。
  - 待切换段位行（`rating_version<2 ORDER BY rating_version, id LIMIT 1`）：`idx_user_ranking_rating_version`，读取 1 行。
  - 历届赛季本人记录：`idx_user_id`（按用户有界，合成数据中每人 1 条）。

## 5. 测试环境（开发服务器）发布与即时切换演练

- 发布：`backend/deploy-dev.sh`，发布目录对应源码 `a8600cb`，迁移 `20261019120000` 执行成功（360 ms），健康检查 200。
- 管理后台测试站：`20261019162819-c4a05cd` 发布到 `admin-zhuifen.kekemate.cn`，旧版备份在 `/www/wwwroot/.admin-backups/admin-zhuifen.kekemate.cn/20261019162819-c4a05cd`。入口页与含 API 地址的 chunk 与本地构建一致，线上 API 指向 `api-zhuifen.kekemate.cn`。
- 发布前备份：开发库 mysqldump（约 2.4 MB，61 张表），存放在服务器备份目录，权限 600。
- 切换演练：在共享 `.env` 写入 `RANKING_RATING_V2_START_AT=2026-10-19T16:08:34+08:00` 并重启服务；切换前为旧规则（`rules_version=1`）。
  - 种子预览：重放 0 场（开发库无比赛记录），写入 0 条种子。
  - 切换时刻到达后：`rules_version=2`，12/12 段位行切到 V2（分数 1250，已计分 0），每行一条 `rating_reset` 记录，共 12 条。
  - 只读审计：`differences=0`。
  - 公开接口（排行榜、排行榜摘要、配置、赛季、健康检查）返回 200；未登录访问受保护接口返回 401。
  - 切换前后没有新增排位相关错误日志。
- 局限：开发库没有比赛数据，HTTP 层面的 V2 结算未在开发环境触发。该路径由第 4 节的 MySQL 验收测试覆盖（8.0 与 5.7 均通过）。

## 6. 正式环境

- 发布前预检（只读）：线上版本 `53eb244`；待执行迁移 1 个（`20261019120000`）；线上库 4 个用户、1 场已完成的排位赛、0 场进行中。
- 后端：人工确认后由 `backend/deploy-prod.sh` 发布 `20261019162334-c4a05cdf7217`（提交 `c4a05cd`）。发布后只读核对：二进制 `vcs.revision` 为 `c4a05cdf7217…`；goose 中 `20261019120000` 已应用、待执行迁移 0；`restart-prod.sh --status` 显示进程运行中，本机与公网健康检查均为 200。
- 管理后台：`20261019162819-c4a05cd` 发布到 `admin.zhuifen.cn`，旧版备份在 `/www/wwwroot/.admin-backups/admin.zhuifen.cn/20261019162819-c4a05cd`。入口页与含 API 地址的 chunk 与本地构建一致，线上 API 指向 `api.zhuifen.cn`；`/`、`/settings/member-ranking-rights`、`/member/list` 返回 200。
- 已排期：线上 `.env` 写入 `RANKING_RATING_V2_START_AT=2026-10-20T00:00:00+08:00`（改动前备份在 `/www/wwwroot/api.zhuifen.cn/deploy-tools/env-before-v2-20261019165823`），API 已重启加载，健康检查 200。公开接口 `/api/public/rank/configs` 返回 `rules_version=1`、`rules_effective_at` 为该时刻。`rank_rating_seeds` 写入 2 条（用户 3 球种 3 → 1290，用户 4 球种 3 → 1210），其余段位行切换时取 1250。
- 未执行：切换本身由后台 worker 在 2026-10-20 00:00（Asia/Shanghai）自动完成。若在 T 之前又有排位完成，需在 T 之前重新执行 `-mode=seed -apply`（整表替换，可重复执行）。
- 切换后待办：执行 `-mode=audit` 并确认 `differences=0`，核对 `rules_version` 与 `rating_reset` 数量，并把结果补到本节。
