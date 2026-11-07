# 前后端读性能基线与回归门槛

> 记录时间：2026-08-10  
> 基线提交：`0549ac2`（实施前的业务读路径；本文件随性能变更维护）  
> 采集范围：登录后的移动端核心路径与其后端读接口

## 1. 采集状态

当前工作树此前只有空的 `backend/tmp/logs/access.log`，且 `Log.Stat` 为 `false`，因此不存在可用于比较的生产 QPS、P50/P95/P99、响应字节或 SQL 耗时历史样本。本次变更已启用：

- go-zero Prometheus/Stat；
- JSON `http_request_completed` 日志：`method`、规范化 `route`、`status`、`duration_ms`、`response_bytes`、`request_id`；
- 可选的请求上下文 SQL `sql_count`、`sql_duration_ms`、`slow_sql_count`；
- 测试/调试环境的移动端 `METHOD route` 请求计数。

上线或压测后，必须先使用同一账号、同一数据库快照和相同网络条件连续采样，再将本表中的“待采集”值补为实测值。不要把缺失数据写成 0。

## 2. 实施前静态请求基线

下表按当前页面源码审计；“首次”不含 App-Plus 可选 push-token 上传，不含用户主动下拉和后续分页。

| 场景 | 已知 HTTP 请求 | 静态数量 | 已知重复/问题 |
| --- | --- | ---: | --- |
| App 前台恢复 | `user/info`、`match/current` | 2 | 每次恢复重新验证；随后首页可能再请求当前对局 |
| 首页（已登录） | `match/current`、`public/rank/leaderboard`、`event-news/list`、`user/favorite-venue-reward-status`、`venue/nearby` | 5 | 与 App 的当前对局重复；公共资源每次 `onShow` 刷新 |
| 观赛页 | `match/current`、`public/matches` | 2 | 当前对局与 App/首页/我的重复 |
| “我的”页 | 未读通知、好友申请、信誉、球馆奖励、会员、用户统计、四球种段位、当前对局、换季通知 | 9 | 每次 `onShow` 重拉；好友申请仅为 total 仍加载列表 |
| 排行榜首次进入 | `public/rank/leaderboard` | 2 | `onMounted` 与 `onShow` 连续请求相同参数 |
| 竞技统计详情 | 分球种、近期趋势、段位趋势、最高分、时长、对手强度 | 6 | 切球种时六项都重新请求；多项扫描历史 |
| H2H 首屏 | H2H 统计、月历史（10 条）、月历史（100 条） | 3 | 同一筛选条件重复下载两份历史 |

## 3. 实施前静态 SQL 风险基线

以下为代码审计结论，需在压测环境用 `sql_count`、慢 SQL 日志与 `EXPLAIN` 补充实际耗时和扫描行数。

| 路径 | 当前风险 | 目标读模型/查询约束 |
| --- | --- | --- |
| 用户统计 | 全部历史比赛至少两次扫描（胜负与最大连胜） | 固定数量 `user_competitive_stats` 快照行 |
| 对局列表、H2H 历史、对手统计 | `user_id OR opponent_id` 后全量读取并在 Go 翻转/分页 | `match_participant_results` 有界索引分页 |
| 单杆最高分 | 全部比赛和动作加载后排序 | 结算时增量快照 |
| 好友/申请/挑战/球馆/赛季/裁判/公开比赛 | N+1 用户、段位、签到或局数读取 | JOIN、聚合子查询或有限 `IN` 批量读取 |
| 当前赛季、荣誉墙、赛季 Worker | 在线/每分钟多次 `ListAll` | 根据 policy 使用当前/相邻有限窗口索引读取 |
| 换季结算、挑战归档 | 整季比赛扫描与按用户球种 N+1 | 增量 season records 与批量 GROUP BY |
| 附近球馆 | 候选范围不受索引限制时需逐条距离计算 | 经纬度边界框预筛选后有限 Haversine |

## 4. 回归门槛

以下为本变更完成前必须实现的上限；后续基于实测基线收紧 P95/P99 数值，但不得放宽“数量不随历史或列表规模增长”的约束。

### 移动端网络请求上限

| 场景 | 完成后的首次/恢复上限 | 说明 |
| --- | ---: | --- |
| 已登录 App 前台恢复 | 1 | Bootstrap；可选 push-token 上传不计入业务首屏 |
| 首页首屏 | 4 | Bootstrap 已提供当前对局与角标；定位失败时少于该值 |
| 观赛页首屏 | 1 | 仅公开比赛列表；当前对局来自 activity Store |
| “我的”页首屏 | 2 | user overview + 尚未加载的 rank 资源；正常缓存命中为 0 |
| 排行榜首次进入 | 1 | 同一球种只请求一次 |
| 竞技统计详情 | 1 | stats overview |
| H2H 首屏 | 1 | H2H overview；后续历史分页单独计数 |

### 后端查询次数上限

| 接口/能力 | 单请求目标上限 | 不变量 |
| --- | ---: | --- |
| Bootstrap | 5 | 不重复读取认证中间件已校验用户 |
| 用户概览 | 5（认证 1 + 业务 4） | 可选区块失败不触发额外重试风暴；业务查询门槛仍为 4 |
| stats overview | 6（认证 1 + 业务 5） | 不随用户历史场次增长 |
| H2H overview | 4 | 不随双方历史场次增长 |
| 好友、申请、挑战、球馆、赛季榜单、裁判、公开比赛列表 | 4 | 返回 1 条与 100 条的 SQL 次数保持上限内 |
| 对局历史、H2H 历史、对手列表 | 3 | 必须包含数据库分页/游标与稳定排序 |
| 稳定赛季 Worker tick | 6 | 不得使用无范围 `ListAll` |

### 延迟、响应体与缓存门槛

- 阶段 0 完成后，为 Bootstrap、用户概览、stats overview、H2H overview、榜单与历史列表补充同一环境下的 P50/P95/P99、平均/最大响应字节、QPS 和错误率。
- Redis 缓存领域同时报告 hit、miss、decode error、Redis error、write error 与 fallback；命中率与冷 miss 的 P95 必须一并评估。
- 任何新缓存不得仅以命中率达标视为完成；其冷 miss 查询也必须满足本表查询上限。

## 5. 采集操作

1. 启动权威配置：`cd backend && go run chasing_points.go -f etc/chasing_points-api.yaml`。
2. 在同一测试账号下执行各核心路径，客户端调试开关 `globalThis.__CHASING_POINTS_REQUEST_METRICS__ = true` 后读取 `getRequestCountSnapshot()`。
3. 解析 `backend/tmp/logs` 中 `http_request_completed` 记录，按 `route` 汇总状态、耗时、字节和 SQL 字段；慢 SQL 使用 `slow_sql` 日志关联 `request_id`。
4. 为热点 SQL 保存代表性 MySQL `EXPLAIN` 输出，确认键、扫描行数和排序策略。
5. 在实施后使用完全相同的路径和数据快照复测，并把对比结果附在本文件的后续“实施结果”小节。

## 6. 实施结果（本地验收）

> 采集时间：2026-08-11  
> 环境：本地 Go/Node 测试进程、SQLite query-count fixtures、MySQL 8.0.42 临时验收库、miniredis。以下延迟不能替代正式网络与生产数据量压测；正式环境按 `docs/performance-read-operations.md` 复测。

### 6.1 移动端请求图

`cd app && node --test tests/*.test.mjs` 共 591 项通过，其中包含使用 `@vue/compiler-sfc`、jsdom 和 `@vue/test-utils` 对赛季页执行的真实 SFC 编译/组件挂载测试。请求图回归结果：

| 场景 | 实施前 | 本地验收 | 结果 |
| --- | ---: | ---: | --- |
| App 冷启动/同一前台代次 | 2，且页面可能重复当前对局 | 1 次 Bootstrap | 达标 |
| 首页首屏 | 5 | 4（overview、榜单摘要、赛讯、附近球馆；当前对局复用 Bootstrap） | 达标 |
| 观赛页首屏 | 2 | 1（公开比赛） | 达标 |
| “我的”首屏 | 9 | 2（user overview + rank infos） | 达标 |
| 排行榜首次进入 | 2 | 1，同身份同球种 single-flight | 达标 |
| stats 详情 | 6 | 1 次 stats overview | 达标 |
| H2H 首屏 | 3 | 1 次 H2H overview | 达标 |

### 6.2 SQL 次数与规模不变量

`cd backend && go test ./...` 全量通过；代表性固定 SQL 断言：

| 能力 | 本地 SQL 次数/边界 |
| --- | --- |
| Bootstrap | 业务 4 条；完整 HTTP 含活动用户认证为 5 条 |
| 用户概览 | 运行时配置热缓存后业务 4 条；完整 HTTP 为 5 条；1/100 行固定不增长 |
| stats overview | schema capability 预热后业务 5 条；完整 HTTP 为 6 条；历史 1/100 行固定不增长 |
| H2H overview | 不超过 4，分页稳定 |
| 公共排行榜共享页 | 冷 miss 4 条共享 SQL + 3 条个人排名 SQL；热 hit 仅 3 条个人排名 SQL |
| 赛季榜单显式 season id | 冷 miss 3 条；热 hit 0 条 |
| 完成对局 ready 核心 | 冷 miss 5 条核心/资料查询（含 rank log 路径时 6）；热 hit 仅 1 条当前资料批量查询 |
| 赛季终榜记录读取 | 1 条，与 1/1000 条记录无关 |
| 挑战归档 | 1/100 pair 均 2 条 GROUP BY 读取；1200 pair 为 6 条固定批次读取 |
| 大赛季结算 | 2000 条相对 10 条仅增加固定写批次数，未出现逐用户 SQL |
| 稳定 lifecycle Worker | 固定窗口和到期页查询，不调用在线 `ListAll` |

### 6.3 MySQL 代表性验收

以下本地 MySQL 临时库测试通过：

```text
RUN_MYSQL_EXPLAIN_TESTS=1 go test ./migrations -run TestMySQLHotReadPlans
RUN_MYSQL_REBUILD_TESTS=1 go test ./migrations -run TestMySQLCompetitiveReadRebuildCheckpointAndAudit
```

- 约 2 万条热点代表数据的 completed matches、好友申请、挑战过期、赛季榜单、到期赛季、赛季挑战归档、附近球馆、裁判历史和公开比赛均命中预期索引。
- 500 场完成比赛读模型 rebuild：dry-run 按 ID 分 7 批检查；正式前 2 批先回填 146 场 `completed_at`；重启后继续完成剩余回填并按 `(completed_at,id)` 投影；再次执行投影扫描 0 场。
- `completed_at` 缺失数最终为 0，participant projection 为 1000 行，audit `differences=0`；MySQL 验收用例覆盖热读索引 `up → down-to → up` 恢复、投影失败 checkpoint 不前进以及 ID/完成时间逆序回放。
- MySQL 验收发现并修复了 `max_win_streak` 在 `ON DUPLICATE KEY UPDATE` 左到右赋值语义下多 1 的问题。

### 6.4 缓存冷 miss / 热 hit 延迟样本

本地 1000 行公共排行榜探针（响应 JSON 3083 bytes）：

| 状态 | P50 | P95 | P99 |
| --- | ---: | ---: | ---: |
| 冷 miss（100 次，每次 bump 球种版本） | 0.937 ms | 1.163 ms | 1.195 ms |
| 热 hit（500 次，同 key） | 0.340 ms | 0.450 ms | 0.577 ms |

探针指标：`hits=500`、`misses=100`、`fallbacks=100`，热 hit 相比冷 miss 的本地 P95 下降约 61%。损坏 JSON、Redis 不可用、写失败和热点并发失效均有独立回源/指标测试。

### 6.5 生产数据库 rollout 证据

> 执行时间：2026-08-12；目标：正式 MySQL 5.7.44 `chasing_points`。执行 108 前已生成 gzip 完整性、55 张表、dump footer 和 SHA-256 均通过的全量逻辑备份；执行 109 前另生成并验证 `matches + goose_db_version` 专项备份。备份文件均保存在操作者本机受限目录，未纳入仓库。

- Goose 从 `20260810143000` 成功迁移到 `20260811109000`；5 张竞技读模型表、`matches.completed_at`、4 个 settlement 阶段字段及关键索引均已核验。`20260811109000` 为快照缺失时的有界 profile fallback 增加创建者/对手两个覆盖排序索引，本地代表性 MySQL EXPLAIN 已确认分别命中 `idx_matches_user_game_status_mode_score` 和 `idx_matches_opponent_game_status_mode_score`。
- 109 执行前保留 108 前完整已验证备份，并额外生成含 `matches` DDL、`goose_db_version` DDL/数据的专项备份。正式账号缺少 `PROCESS` 权限且全量 dump 两次在 10 分钟内于 64 KiB 停滞，因此没有把不完整 dump 当作备份证据；109 仅对 0 行 `matches` 增加两条可逆索引。迁移后 Goose 版本、索引列序、用户/比赛/投影行数和旧公开比赛 API 均已复核。
- 迁移前正式库有 2 个用户、0 场已完成比赛；迁移后 `completed_at_missing=0`、participant projection=0、竞技统计快照=0。
- 正式环境 read-model dry-run、正式 rebuild 和幂等重跑均扫描 0 场；audit 输出 `matches=0 users=0 differences=0`。
- 2026-08-12 重新发布应用后，新 `/api/public/rank/configs`、`/api/public/rank/leaderboard-summary` 及认证聚合接口均已返回 200；旧用户信息、用户统计、分球种统计、当前对局和公开比赛接口继续可用。无效 Bearer Token 返回 401，匿名好友观赛返回明确未登录业务响应。
- 发布后再次执行 catch-up dry-run、正式 rebuild、幂等重跑和 audit：均扫描 0 场，`differences=0`；checkpoint 为 `backfill_completed=1`、`paused=0`、无错误。正式库仍为 2 个有效用户、0 场完成比赛、0 条参与者投影/竞技统计/对手统计/赛季记录。
- 应用继续保持 `ReadMode=disabled`。Bootstrap、user overview、stats overview、opponent candidates 和 season overview 对竞技 revision/新投影区块返回明确 `availability=false` / partial error，而不是把空快照伪装成成功零值；其他兼容区块正常返回。
- 2026-08-13 使用两个明确标记的性能验收账号完成生产比赛 `id=1`：确认接口原样重放后 revision 保持 4；`finish_request`、`finish_confirm`、`match_end`、`win` 各恰好一条。比赛生成 2 条参与者投影、4 条总体/球种竞技快照、4 条总体/球种双向对手快照、4 条强度桶、2 条段位日志和 2 条赛季记录，`season_records.rewards` 为 SQL NULL。
- 生产 catch-up 将 checkpoint 推进到 `cursor_match_id=1`；再次执行扫描 0 场。2026-08-14 最终兼容应用发布后重跑 audit，输出 `matches=1 users=2 differences=0`。线上仍为单进程且 `ReadMode=disabled`，未因审计通过自动切读。

### 6.6 生产公网黑盒采样

> 采集时间：2026-08-12；采集机到 `api.zhuifen.cn`，HTTP/2 持久连接、单路串行低负载；公开路由 60 次、认证路由 40 次。该数据包含公网链路和反向代理时间，不等于服务端处理耗时。

| Route | 样本 | P50 | P95 | P99 | 平均/最大响应字节 | 串行 QPS | 错误率 |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| rank configs | 60 | 19.563 ms | 516.660 ms | 528.476 ms | 703 / 703 | 11.832 | 0% |
| leaderboard summary | 60 | 21.305 ms | 443.849 ms | 452.020 ms | 78 / 78 | 9.475 | 0% |
| leaderboard page | 60 | 27.873 ms | 523.346 ms | 697.141 ms | 69 / 69 | 5.992 | 0% |
| public matches | 60 | 20.687 ms | 528.305 ms | 726.277 ms | 49 / 49 | 6.056 | 0% |
| Bootstrap | 40 | 23.799 ms | 526.062 ms | 532.925 ms | 532 / 532 | 6.265 | 0% |
| user overview | 40 | 28.473 ms | 531.441 ms | 536.738 ms | 999 / 999 | 4.479 | 0% |
| stats overview | 40 | 27.563 ms | 524.090 ms | 526.202 ms | 503 / 503 | 5.927 | 0% |
| opponent candidates | 40 | 21.354 ms | 457.010 ms | 508.326 ms | 278 / 278 | 8.382 | 0% |
| season overview | 40 | 60.360 ms | 525.237 ms | 531.845 ms | 606 / 606 | 5.062 | 0% |
| notification list | 40 | 85.835 ms | 523.822 ms | 525.102 ms | 53 / 53 | 4.867 | 0% |

所有样本均收到 `X-Request-ID`。链路存在明显双峰：HTTP/2 同一复用连接上的缓存 200、JWT 401 和未知路由 404 都出现约 500 ms 长尾，因此不能把公网 P95 全部归因于业务 SQL。稳定样本多在约 15～85 ms，后续需结合服务端 `duration_ms` 与 `sql_duration_ms` 区分网络/代理和应用耗时。

排行榜 20 秒 TTL 在同一复用连接上的 4 轮成对采样：冷 miss 分别为 595.891、92.318、92.242、127.499 ms，紧随其后的热 hit 为 64.089、26.137、28.228、18.500 ms，单轮改善约 69%～89%。首轮冷样本包含异常网络长尾；其余三轮冷 miss 为 92～127 ms、热 hit 为 18～28 ms。

### 6.7 最终生产服务端采样

> 采集时间：2026-08-14；生产二进制 SHA-256 `bc180a4730f48aeb3d2cc3d49ea66708a3dea767c6282a4a72f050fd371db840`；`ReadMode=disabled`。常规场景各 20 次，排行榜按 20 秒 TTL 采集 5 次冷 miss 和 25 次紧随其后的热 hit，共 230 条固定 `prodperf-v3-*` 请求。全部为 HTTP 200，无慢 SQL；服务端日志证据位于操作者本机受限文件 `prodperf-v3-access.log`，SHA-256 `00c8671bbf4c7f7a2a0de909edfd3a5831fdd146db15b8e09ae8eda437103f9d`，已验证不含 Token、Cookie、请求/响应正文或敏感字段。

| Route | n | 服务端 P50/P95/P99 | 平均/最大 bytes | SQL 平均/最大 | SQL ms 平均/最大 | 错误 |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| Bootstrap | 20 | 2 / 2 / 2 ms | 532 / 532 | 5.0 / 5 | 1.9 / 2 | 0 |
| user overview（disabled 热配置） | 20 | 3 / 3 / 3 ms | 1001 / 1001 | 6.0 / 6 | 2.0 / 3 | 0 |
| stats overview（disabled 稳态） | 20 | 3 / 5 / 5 ms | 972 / 972 | 8.0 / 8 | 7.5 / 12 | 0 |
| H2H overview | 20 | 1 / 1 / 1 ms | 146 / 146 | 2.0 / 2 | 0.5 / 1 | 0 |
| opponent candidates | 20 | 1 / 1 / 1 ms | 278 / 278 | 2.0 / 2 | 0.7 / 1 | 0 |
| season overview | 20 | 1 / 1 / 1 ms | 606 / 606 | 2.0 / 2 | 0.5 / 1 | 0 |
| notification list | 20 | 1 / 3 / 3 ms | 293 / 293 | 4.0 / 4 | 1.2 / 3 | 0 |
| leaderboard summary（认证） | 20 | 2 / 5 / 5 ms | 520 / 520 | 4.1 / 6 | 1.4 / 3 | 0 |
| rank configs | 20 | 0 / 0.1 / 1.6 ms | 703 / 703 | 0 / 0 | 0 / 0 | 0 |
| public matches | 20 | 1 / 1.1 / 1.8 ms | 540 / 540 | 2.0 / 2 | 1.0 / 1 | 0 |
| leaderboard cold | 5 | 2 / 2 / 2 ms | 366 / 366 | 4.0 / 4 | 1.0 / 1 | 0 |
| leaderboard hot | 25 | 0 / 0 / 0 ms | 366 / 366 | 0 / 0 | 0 / 0 | 0 |

缓存结果来自同一条 `http_request_completed`，不是用延迟推断：

| 缓存场景 | hits | misses | hit rate | fallbacks | decode/Redis/write errors |
| --- | ---: | ---: | ---: | ---: | ---: |
| leaderboard cold | 0 | 5 | 0% | 5 | 0 / 0 / 0 |
| leaderboard hot | 25 | 0 | 100% | 0 | 0 / 0 / 0 |
| leaderboard summary（认证） | 19 | 1 | 95% | 1 | 0 / 0 / 0 |
| rank configs | 20 | 0 | 100% | 0 | 0 / 0 / 0 |
| user overview 三项运行时配置 | 60 | 0 | 100% | 0 | 0 / 0 / 0 |

额外受控冷配置请求确认 user overview 的 3 个运行时配置同时过期时为 `duration=6 ms`、`sql_count=9`、`misses=3`、`fallbacks=3`；紧随其后的热请求为 `duration=3 ms`、`sql_count=6`、`hits=3`。冷 miss 仍为固定查询数且没有缓存错误。

### 6.8 最终生产公网网络采样

采集机以 HTTP/2 单路串行、约 3.47～3.86 offered QPS 执行常规场景；排行榜各轮约 3.75～4.58 offered QPS。全部 230 个响应均为 200、均回显固定 `X-Request-ID`。下表的串行传输 QPS 按请求实际传输时间计算，不包含显式 pacing 等待，不能视为容量上限。

| Route | n | 公网 P50/P95/P99 | 平均/最大 bytes | 串行传输 QPS | 错误 |
| --- | ---: | ---: | ---: | ---: | ---: |
| Bootstrap | 20 | 127.3 / 249.0 / 250.6 ms | 532 / 532 | 8.405 | 0 |
| user overview | 20 | 158.6 / 273.5 / 274.9 ms | 1001 / 1001 | 6.945 | 0 |
| stats overview | 20 | 157.0 / 267.7 / 269.9 ms | 972 / 972 | 6.954 | 0 |
| H2H overview | 20 | 36.0 / 307.8 / 680.7 ms | 146 / 146 | 6.717 | 0 |
| opponent candidates | 20 | 153.8 / 273.0 / 275.0 ms | 278 / 278 | 6.883 | 0 |
| season overview | 20 | 160.1 / 275.4 / 276.6 ms | 606 / 606 | 6.888 | 0 |
| notification list | 20 | 41.5 / 332.8 / 700.7 ms | 293 / 293 | 6.714 | 0 |
| leaderboard summary（认证） | 20 | 153.2 / 277.3 / 277.4 ms | 520 / 520 | 6.963 | 0 |
| rank configs | 20 | 141.9 / 275.5 / 276.3 ms | 703 / 703 | 7.089 | 0 |
| public matches | 20 | 52.4 / 278.9 / 279.5 ms | 540 / 540 | 7.990 | 0 |
| leaderboard cold | 5 | 72.5 / 455.6 / 506.3 ms | 366 / 366 | 5.399 | 0 |
| leaderboard hot | 25 | 39.3 / 232.1 / 235.0 ms | 366 / 366 | 9.672 | 0 |

服务端排行榜 cold/hot 分别稳定为 2 ms/0 ms 和 4 SQL/0 SQL，但公网仍有独立长尾，证明网络、代理和连接调度必须与应用 SQL 分开判断。

### 6.9 SQL 门槛口径收敛

- `sql_count` 现在覆盖真实 HTTP 链中的活动用户认证 SQL。原“用户概览 4 条”是业务 SQL 门槛；完整 HTTP 门槛应明确为 **认证 1 + 业务 4 = 5**，不是为了适配旧的 11 条而放宽。
- 用户概览优化前同数据为 11 SQL；最终兼容 `disabled` 模式在运行时配置热缓存时为 6，冷配置为 9。服务器本地只读探针仅在探针内强制 `enabled`、不修改线上 YAML，得到 `sql_count=5`、`sql_duration=2 ms`、3 次配置缓存 hit，达到完整 HTTP 门槛。
- stats overview 优化前同数据为 11 SQL。最终进程首次 schema capability 探测请求仍为 11，但 capability 由请求级 Model 共享后，后续 `disabled` 稳态固定为 8；服务器本地 `enabled` 只读探针得到 `sql_count=6`、`sql_duration=4 ms`，即认证 1 + 业务 5，达到门槛。
- 生产全局仍保持 `ReadMode=disabled`。上述 enabled 结果只用于在已审计的生产数据上验证目标查询图，不构成切读；真机/模拟器验收完成前不得把开关改为 enabled。

### 6.10 分阶段收益

| 阶段/场景 | 优化前 | 最终证据 | 收益 |
| --- | ---: | ---: | ---: |
| App 前台恢复请求 | 2 | 1 Bootstrap | -50% |
| 首页首屏请求 | 5 | 4 | -20% |
| 观赛首屏请求 | 2 | 1 | -50% |
| “我的”首屏请求 | 9 | 2 | -78% |
| 排行榜首次请求 | 2 | 1 | -50% |
| stats 首屏请求 | 6 | 1 | -83% |
| H2H 首屏请求 | 3 | 1 | -67% |
| user overview SQL（同数据，disabled 热配置） | 11 | 6 | -45% |
| user overview SQL（enabled 目标读图） | 11 | 5 | -55% |
| stats overview SQL（同数据，disabled 稳态） | 11 | 8 | -27% |
| stats overview SQL（enabled 目标读图） | 11 | 6 | -45% |
| leaderboard 热请求 SQL | 4 | 0 | -100% |

### 6.11 尚未完成的发布验收

服务端生产流量、网络、SQL、缓存、双写、catch-up 和 audit 证据均已补齐；唯一剩余阻断是按 `docs/client-read-performance-device-acceptance.md` 在 Android、iOS 和微信小程序/开发者工具完成脱敏真机或模拟器验收。完成前任务 13.4 保持未勾选，竞技新读模型保持 disabled，本 OpenSpec change 不归档。
