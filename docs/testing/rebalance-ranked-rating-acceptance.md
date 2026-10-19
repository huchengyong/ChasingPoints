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

- 未用生产数据运行种子预览：段位分布与阈值是否需要校准，需在旧赛季最后一晚用 `-mode=seed` 的输出确认。
- 未做 App 与微信小程序编译及真机检查：段位说明页（规则版本、定级提示）、赛季页（达标进度、未达标标注、本季净变化）、历届赛季与报告的"未达标"、战报高光卡片、会员中心排位权益区，浅色/深色都需要人工确认。
- 原生 iOS 无改动、未构建：战报按 `label/value` 通用渲染排位明细，新增字段不参与解码。
- 上线后首日需在目标环境执行 `-mode=audit` 并确认 `differences=0`。
