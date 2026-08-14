# 客户端/API 读性能运维手册

> 适用于 OpenSpec 变更 `optimize-client-api-read-performance`。所有正式环境操作必须先确认目标主机、数据库名和变更窗口；禁止直接复制本地 `.env` 凭据到日志或工单。

## 1. 权威配置与日志

- 服务配置：`backend/etc/chasing_points-api.yaml`
- 环境变量：`backend/.env` 或部署平台注入
- 启动：

```bash
cd backend
go run chasing_points.go -f etc/chasing_points-api.yaml
```

- HTTP 完成日志事件：`http_request_completed`
- 关键字段：`method`、规范化 `route`、`status`、`duration_ms`、`response_bytes`、`request_id`、`sql_count`、`sql_duration_ms`、`slow_sql_count`、`cache_hits`、`cache_misses`、`cache_decode_errors`、`cache_redis_errors`、`cache_write_errors`、`cache_fallbacks`
- SQL 慢查询事件：`slow_sql`，使用 `request_id` 与 HTTP 日志关联。
- 缓存领域统计统一包含：`hits`、`misses`、`decode_errors`、`redis_errors`、`write_errors`、`fallbacks`。Redis 错误不应改变业务响应的成功语义。

## 2. 数据库迁移

```bash
cd backend
./goose.sh status
./goose.sh up
./goose.sh version
```

本变更末端迁移：

- `20260811100000_add_match_completed_at.sql`
- `20260811101000_add_competitive_read_models.sql`
- `20260811102000_add_hot_read_indexes.sql`
- `20260811103000_add_user_opponent_snapshot_aggregates.sql`
- `20260811104000_add_finish_request_expiry_index.sql`
- `20260811105000_add_achievement_sync_pending_index.sql`
- `20260811106000_add_season_settlement_phases.sql`
- `20260811107000_add_season_challenge_archive_index.sql`
- `20260811108000_add_ordered_competitive_rebuild_checkpoint.sql`
- `20260811109000_add_competitive_profile_fallback_indexes.sql`

回滚阶段字段前必须先回滚到不读取阶段列的应用版本：

```bash
./goose.sh down-to 20260811105000
```

`20260811107000` 仅增加归档索引，可单独回滚；不要在高峰期执行大型表 DDL。

## 3. Redis 缓存域

| 域 | 主要 key/version | TTL/语义 | 失效源 |
| --- | --- | --- | --- |
| 赛讯 | `eventnews:version` | 列表 5 分钟、详情 3 分钟 | 赛讯管理/WST 写入 bump |
| 公共排行榜 | `leaderboard:version:{game_type}` | 20 秒，共享部分不含 `my_ranking` | 对应球种排位结算；资料更新 bump 四球种 |
| 静态数据 | `static-read:v1:*` | 24 小时 | App/schema/config 版本升级 |
| 用户概览运行时配置 | `static-read:v1:runtime-config:{reputation,member-rights,favorite-venue-reward}` | 5 分钟；只缓存共享配置，不缓存用户响应 | 对应 admin 配置写入后精确删除；TTL 兜底 |
| 球馆 | `venue:version` | 列表/详情 2 分钟、附近 1 分钟 | 创建、审核、资料可见性变化 |
| 赛季信息 | `season:info:version` | 当前窗口 5 分钟且 key 含 policy window | 排期创建、repair、换季发布 |
| 赛季榜单 | `season:leaderboard:version:{season}:{game}`、`season:leaderboard:profile-version` | 30 秒 | 比赛结算、资料更新 |
| 完成对局核心 | `completed-match-core:v1:{match_id}` | 24 小时，只缓存 achievement ready 的不可变事实 | 无常规失效；pending 不写入 |

附近球馆 key 使用离散经纬度桶、`radius`、`limit` 和领域版本，不使用完整坐标。完成对局当前胜率、最高分和用户资料不在长期核心中，每次从当前用户/竞技快照组装。

### Redis 故障处理

1. 确认 `redis_errors` / `write_errors` 增长，同时 HTTP 是否仍成功。
2. 缓存读取、JSON 解码或写入失败会回源 MySQL/静态真源；禁止因 Redis 故障回滚已提交比赛。
3. 损坏 JSON 会计入 `decode_errors`、删除坏 key 并回源。
4. 同进程相同热点 key 使用 single-flight，避免冷失效时同实例重复回源。
5. Redis 恢复后无需清库；版本 key 和 TTL 会自然收敛。仅在确认 key 污染时按域删除，不要执行全库 `FLUSHALL`。

## 4. 竞技读模型切换、rebuild / audit

`CompetitiveReadModel.ReadMode` 是强制读开关，默认和正式 YAML 都必须保持 `disabled`。空快照表、未完成回建或审计有差异时，线上读路径继续使用旧聚合查询，绝不能返回成功的零值快照。

按以下顺序切换，不能跳步：

1. 备份并在只读窗口确认 Goose 当前版本；执行迁移至 `20260811109000`，此时旧应用仍可继续提供旧读路径。
2. 部署包含兼容读路径和事务内双写投影的应用，保持 `COMPETITIVE_READ_MODEL_READ_MODE=disabled`。确认新完成对局会写入投影，但在线读仍来自旧实现。
3. 执行下方 dry-run、分批 rebuild；checkpoint 先按 ID 回填 `completed_at`，再按 `(completed_at,id)` 回放，只有单场投影成功后才推进。
4. 在稳定窗口重复 rebuild 直到无待处理记录，运行 audit，要求 `differences=0`；若有并发完成对局，先继续双写并重跑 catch-up rebuild/audit。
5. 记录审计输出、checkpoint 和指标后，才将 `COMPETITIVE_READ_MODEL_READ_MODE=enabled` 做滚动发布。切读后观察错误率、SQL 数和空快照比例。
6. 任意异常立即将开关恢复为 `disabled` 并滚动发布；双写和事实表不回滚，定位后从 checkpoint 重建并再次审计。

先 dry-run：

```bash
cd backend
go run ./cmd/competitive_read_model_rebuild -f etc/chasing_points-api.yaml \
  --dry-run --batch-size 100 --max-batches 0
```

正式分批执行：

```bash
go run ./cmd/competitive_read_model_rebuild -f etc/chasing_points-api.yaml \
  --batch-size 100 --max-batches 20 --rate-limit 200ms
```

暂停和恢复：

```bash
go run ./cmd/competitive_read_model_rebuild -f etc/chasing_points-api.yaml --pause
go run ./cmd/competitive_read_model_rebuild -f etc/chasing_points-api.yaml --resume --batch-size 100
```

checkpoint 存于 `competitive_read_model_rebuild_checkpoints`，job name 为 `competitive-read-model-v1`。完成后重复执行应扫描 0 场且不产生重复投影。

审计：

```bash
go run ./cmd/competitive_read_model_audit -f etc/chasing_points-api.yaml --sample-size 1000
```

验收标准：`differences=0`。若存在差异，保留输出中的 `kind`、`match_id`、`user_id`，不要直接手改投影表；先定位事实表、`completed_at` 或结算日志，再从 checkpoint 安全重建。

## 5. 连续赛季 repair

Dry-run 按固定窗口批次读取，不加载全部历史：

```bash
cd backend
go run ./cmd/season_lifecycle_repair -f etc/chasing_points-api.yaml \
  --dry-run --batch-size 100
```

正式执行并在每个成功窗口后原子写 checkpoint：

```bash
go run ./cmd/season_lifecycle_repair -f etc/chasing_points-api.yaml \
  --batch-size 100 --checkpoint tmp/season-repair.checkpoint
```

手工从排他 cursor 继续：

```bash
go run ./cmd/season_lifecycle_repair -f etc/chasing_points-api.yaml \
  --batch-size 100 --after-start-date 2026-08-01
```

checkpoint 内容是最后完成窗口的 `YYYY-MM-DD` 开始日期。中断后可重复处理上一批；赛季结算阶段、挑战快照、称号、通知均使用幂等键。

赛季结算阶段字段：

- `records_completed_at`
- `challenges_completed_at`
- `titles_completed_at`
- `notifications_completed_at`
- 最终 `status=completed` / `completed_at`

阶段失败时已完成 checkpoint 保留，但赛季不得对外标记 completed。最终发布在小事务中关闭旧赛季、激活下一赛季并写 completed；Push/WebSocket 只在提交后发送。

## 6. MySQL EXPLAIN 验证

本地或隔离验证库：

```bash
cd backend
export $(grep -v '^#' .env | grep -v '^$' | xargs)
RUN_MYSQL_EXPLAIN_TESTS=1 go test ./migrations -run TestMySQLHotReadPlans -count=1 -v
# 此验收同时覆盖 rebuild/audit 和热读索引的 up → down-to → up 恢复：
RUN_MYSQL_REBUILD_TESTS=1 go test ./migrations -run TestMySQLCompetitiveReadRebuildCheckpointAndAudit -count=1 -v
```

测试会创建并删除临时数据库。不得将正式库名配置为测试临时库名，也不得在无 `DROP DATABASE` 权限边界审查时运行。

## 7. 发布后验收

按 `docs/performance-baseline.md` 中相同页面路径采样：

1. App 前台恢复、首页、观赛、“我的”、排行榜、stats、H2H 请求数。
2. Bootstrap、overview、H2H、列表类接口 SQL 次数。
3. 每个 route 的 P50/P95/P99、平均/最大响应字节、QPS 和错误率。
4. Redis 冷 miss 与热 hit 分开统计；同时记录 fallback 和错误计数。
5. 赛季 Worker 稳定 tick 不得调用无范围 `ListAll`；换季 SQL 不得随用户球种 pair 线性增长。

若 P95/P99 回退但 SQL 次数正常，先区分冷缓存、Redis 网络、MySQL 锁等待和响应体增长，不要通过延长所有 TTL 掩盖根因。

### 7.1 聚合固定生产探针日志

发布后探针使用 `X-Request-ID: prodperf-<批次>-<场景>-<YYYYMMDD>-<序号>`，不会记录 Token 或响应正文。聚合脚本按日期和批次通用解析，例如：

```bash
cd backend
python3 ../docs/scripts/summarize-performance-log.py /www/wwwroot/api.zhuifen.cn/logs/access.log
```

若采样跨过日志轮转，同时传入对应的 `access.log*`。脚本只读取 `http_request_completed`，输出分组样本数、2xx/错误数、服务端 P50/P95/P99、平均/最大响应字节、平均/最大 SQL 次数、SQL 总耗时、慢 SQL，以及六类请求级缓存计数。代表性批次至少包含：

- `bootstrap`、`user-overview`、`stats-overview`、`h2h-overview`、`opponent-candidates`、`season-overview`、`notification-list`
- `leaderboard-summary-auth`、`rank-configs`、`public-matches`
- 多次自然 TTL 到期后的 `leaderboard-cold`，以及每次 cold 后紧随的 `leaderboard-hot`

2026-08-14 最终批次为上述常规场景各 20 条、leaderboard cold 5 条、hot 25 条，共 230 条。不要复制完整生产 access log 到工单；只保存经固定前缀过滤、验证无敏感字段的证据或聚合表。

### 7.2 缓存指标读取边界

各领域仍保留进程级原子累计计数；同时，在真实 HTTP Context 内发生的 hit/miss/decode error/Redis error/write error/fallback 会累加到该请求的 `http_request_completed`。因此发布验收应优先使用固定 request id 的完成日志直接计算命中率、冷 miss 和错误，不得再根据公网延迟反推。

正式环境未配置公开 Prometheus agent，公网 `/metrics` 保持不可用。若以后需要进程总量，指标端点只能绑定 loopback、内网或受认证监控网络，不得为了验收直接暴露到互联网。
