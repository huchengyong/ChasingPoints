# 验证记录

## 1.2 WST 官方历史接口只读核实

- 观测日期：2026-09-08（UTC）
- 观测方式：直接只读请求代码中配置的三个公开 HTTPS 端点；未携带鉴权信息，未向仓库保存完整响应或球员 `history`。
- 结果边界：以下计数只代表该时点的接口快照。接口没有提供快照 ID 或游标，分页期间源数据变化仍可能造成跨页漂移。

### 端点与分页

| 资源 | 请求 | 实测结果 |
| --- | --- | --- |
| 赛季 | `GET https://seasons.snooker.web.gc.wstservices.co.uk/v2` | `data=5`，`meta.totalCount=5`、`meta.count=5`，目录为 2027/28 至 2023/24。使用 `page.number=1&page.size=1` 时 `links.last` 指向第 5 页，说明端点支持分页；默认请求在本次观测中一次返回全部数据。 |
| 2023 赛事 | `GET https://tournaments.snooker.web.gc.wstservices.co.uk/v2?season=2023&page.number=1&page.size=200` | 第 1 页 46 条，`meta.totalCount=46`；第 2 页 `data=[]`。用 `page.size=2` 复核时第 23 页 2 条且 `links.next=null`，第 24 页为空。 |
| 全局对阵 | `GET https://matches.snooker.web.gc.wstservices.co.uk/v2?page.number=1&page.size=200` | `meta.totalCount=8990`；按程序页大小抽查第 45 页为 190 条且 `links.next=null`，第 46 页为空。另以 `page.size=1000` 全扫 9 个数据页，得到 8990 个唯一来源 ID，第 10 页为空。 |

响应均为 HTTP 200、`application/json`，结构包含 `data`、`meta`、`links`。赛事和对阵的 `links` 提供 `first`、`prev`、`next`、`last`；最后一个非空页的 `next` 为 `null`，越界页返回空 `data`，而不是 404。`tournament`、`filter[tournamentID]` 和 `filter[tournament]` 参数均未缩小对阵结果，`meta.totalCount` 仍为 8990，因此对阵需要全局分页后按 `tournamentID` 本地过滤。

赛季目录默认按年份降序；2023 赛事默认按 `startDate` 升序。全局对阵默认按 `startDateTime` 降序，但相同时间存在大量并列。显式 `sort=startDateTime` 返回升序，`sort=-startDateTime` 本次仍返回升序；赛事的 `sort=-startDate` 也被响应链接归一为 `sort=startDate`。因此不能依赖负号方向或唯一稳定次序，ID 去重、原始页无进展检查和页数上限仍是必要保护。

### 2023/24 数据覆盖

赛季目录最早项为 `2023`，标注范围为 2023-06-26 至 2024-06-01；直接查询 `season=2022` 返回 `data=[]`、`meta.totalCount=0`。这只证明观测时点可取得的最早目录数据为 2023/24，不承诺源站永久保留范围。

- 赛事：46 个唯一 ID，最早开始日期 2023-06-26，最晚结束日期 2024-06-01。
- 对阵：按这 46 个赛事 ID 从全局响应筛得 2580 个唯一 ID，时间为 2023-06-26 11:00:00 至 2024-06-01 09:00:00。
- 45 个赛事有对阵；`88b77674…` 是 2024-04-19 的播客活动，官方 `matchCount=0`，没有对阵。
- 赛事响应没有 `status` 字段。2023/24 的 2580 个对阵均为 `Completed`；全局状态分布为 `Completed=8833`、`Scheduled=155`、`Live=2`。

2023/24 对阵的主客比分字段均存在且非 `null`。明确的 `0` 是有效值：未来 `Scheduled` 对阵也以 0:0 返回，不能仅凭比分判断状态。2023/24 的 `playersAllocated` 分布为 `true=2577`、`false=2`、`null=1`，三种情况必须区分。当前全量响应另提供了待定选手样例：

- `94649826…`：`Scheduled`、0:0、`playersAllocated=false`，名称含 `Winner of Match 4`，客方 ID 为 `null`，并用 `unallocatedAwayPlayerFixtureNumber=4` 表达待定选手。
- `e6d8508b…`：`Completed`、1:4、`playersAllocated=false`，名称含 `Winner of Match 84`，主方 ID 为 `null`，说明明确的 `false` 不等同于无效对阵。
- `a357b4f1…`：`Completed`、4:0、`playersAllocated=false`，名称含 `Withdrawn Player #2`，双方 ID 均为 `null`，是本次观测到的退赛占位样例。
- `52d5a835…`：`Completed`、0:0，`playersAllocated`、`fixtureNumber`、`numberOfFrames` 均为 `null`，双方 ID 有值。JSON `null` 不能按明确的 `false` 或整数零值处理。

本次 8990 条全量响应中没有观察到名为 `Bye`、`Walkover`、`Retired` 的状态、名称或专用字段，也没有足够证据确定轮空和中途退赛的统一编码规则。实现只能按已观测的待定和 `Withdrawn Player` 语义处理，其余情况必须保留为未知并进入核对清单。

### 完整性与运行保护结论

赛事接口提供的计数与全局对阵接口不完全一致：46 个赛事的 `matchCount` 合计 2637，`matchesWithStartDateCount` 合计 2590，而实际筛得 2580 条。已定位的有日期差异为：

- `9c4c0551…`（UK Championship 2023 Qualifiers）：官方计数 112，实际 111，差 1。
- `36ba53b4…`（Riyadh 2024）：`matchCount=40`、`matchesWithStartDateCount=20`，实际 11，较有日期计数少 9。

因此成功走到空页只能证明完成了本次接口扫描，不能证明官方历史完整。运行摘要应把累计唯一 ID 与 `meta.totalCount`、`links.last/next` 和赛事计数差异一并核对；上述差异应产生 `needs_review`，不能报告为无异常的 `completed`。

程序实际页大小为 200，本次对阵有 45 个数据页，固定上限 500 页约为当前体量的 11 倍，保留该上限合理，触发时必须失败。赛事当前每赛季只有一个数据页，但仍应按 `links`/空页遍历并检查无新增 ID。

本轮只读请求未遇到 429 或 5xx，响应也未提供限流头，无法从真实响应推导服务端限流窗口。现有最多 3 次请求、普通退避 1 秒和 2 秒、累计等待不超过 60 秒、单请求超时 2 分钟的预算可作为保守运行值；429 应继续尊重合法 `Retry-After`，超过累计预算则失败后重跑。

复核命令：

```bash
curl -sS 'https://seasons.snooker.web.gc.wstservices.co.uk/v2'
curl -sS 'https://tournaments.snooker.web.gc.wstservices.co.uk/v2?season=2023&page.number=1&page.size=200'
curl -sS 'https://matches.snooker.web.gc.wstservices.co.uk/v2?page.number=45&page.size=200'
curl -sS 'https://matches.snooker.web.gc.wstservices.co.uk/v2?page.number=46&page.size=200'
```

## 5.2 隔离 MySQL 验收

- 执行时间：2026-09-09 06:22:07–06:23:46 UTC。
- 环境：本机临时启动的隔离 MySQL 8.0.42，独立 data directory、socket、PID、日志和 TCP 端口 `43381`；未连接业务数据库。
- Schema：对随机临时数据库完整应用 `backend/migrations`，最终版本 `20260908000000`。
- Schema 约定：`testsupport` 的轻量 schema 仅服务 SQLite 单测，无法验证 MySQL 生成列；本验收直接应用真实迁移，避免复制一套测试表结构。
- 测试：当前源码包含 `soft-event-tournament` 赛讯软删除用例，明确使用 `-count=1` 禁用测试缓存。

执行形式（连接参数通过临时环境变量传入，未记录密码或 DSN）：

```bash
RUN_WST_MYSQL_ACCEPTANCE_TESTS=1 go test ./internal/logic/wstsync \
  -run '^TestWSTBackfillMySQLAcceptance$' -count=1 -v
```

Shell 退出码为 0，测试本身耗时 1.25 秒，package 耗时 3.051 秒。验收结果：

| 子项 | 结果 |
| --- | --- |
| 官方来源生成列唯一索引 | `players.official_source_player_id` / `uk_players_official_source_player_id`、`tournaments.official_source_tournament_id` / `uk_tournaments_official_source_tournament_id`、`tournament_matches.official_source_match_id` / `uk_tournament_matches_official_source_match_id` 均为 STORED GENERATED + UNIQUE；重复 official ID 均返回 MySQL 1062，manual 与空来源 ID 不被错误约束。 |
| 软删除冲突 | player、match 的 dry-run 与正式模式预检均失败并列出来源 ID，四表计数不变；对应来源 ID 直接重建仍由唯一索引返回 1062。软删除 official event news 同样在两个模式的投影前失败并列出 tournament ID，不创建替代行。 |
| 重复导入 | 连续两次导入后，players、tournaments、tournament_matches、event_news_events 的记录数量和全部业务主键保持一致。 |
| 事务回滚 | 在赛讯写入阶段注入唯一键冲突后得到 `failed`、`commit_state=rolled_back`；本轮四类事实新增均为 0。 |

当前 `tournaments` 表和 Gorm 模型没有 `deleted_at`，因此不存在“软删除赛事”可构造场景；赛事 official ID 由生成列唯一索引保护。验收后 MySQL 已正常关闭，data directory、socket、PID、日志全部删除，端口 `43381` 已确认关闭，未留下临时数据库或进程。

## 5.3 开发环境全范围 dry-run

- 执行日期：2026-09-09（UTC）
- 环境：用户指定的 `168.107.77.113:14379` 开发主机；`/home` HTTP 路径本身返回 404，因此通过同主机 shell 在实际发布配置下运行 CLI。
- 主机：Linux `aarch64`，MySQL 8.0.45；业务服务 `chasing-points.service` 在验收后仍为 `active`。
- 构建：本地 Go 1.25.0 以 `CGO_ENABLED=0 GOOS=linux GOARCH=arm64` 构建；Git HEAD 为 `96aa53d711adf46a66294ba284fb815f425d206f` 加当前变更工作树，验证二进制 SHA-256 为 `99fa238fbf044b061b3874d4d0b6142126dcb09639c1a059b6cd13a928cd7330`，远端哈希一致。
- 运行：以服务用户 `www`、共享 `.env` 和当前发布的 `chasing_points-api.yaml` 直接执行二进制 `--backfill --dry-run`；未执行正式导入。

执行形式：

```bash
./wst_sync_linux_arm64 -f /www/server/chasing-points/current/chasing_points-api.yaml --backfill --dry-run
```

外层使用 Python 3.12 标准库的 `time.monotonic` 和 `resource.getrusage(RUSAGE_CHILDREN)` 保留真实退出码、耗时和峰值 RSS。第二次复核还在执行前后对 `players`、`tournaments`、`tournament_matches`、`event_news_events`、`wst_sync_job_states` 执行只读 `CHECKSUM TABLE`。

### 运行摘要

| 指标 | 结果 |
| --- | ---: |
| 状态 / 退出码 | `needs_review` / `2` |
| 请求范围 | 2023-01-01 至 2026-09-09（UTC，闭区间） |
| 实际赛事覆盖 | 2023-06-26 至 2026-09-13；末日超出请求上界是因为赛事按日期相交纳入 |
| 赛季目录 / 候选赛季 | 5 / 4（2023 至 2026） |
| 拉取赛事 / 选中赛事 | 183 / 160 |
| 对阵数据页 / 扫描对阵 | 46 / 9038 |
| 选中 / 跳过 / 有效准备对阵 | 8865 / 1 / 8864 |
| 准备球员 / 赛事 / 赛讯 | 466 / 160 / 160 |
| 保留已有对阵 / 拟改变发布状态 | 0 / 0 |
| 数据库事务状态 | `not_started` |
| 复核运行耗时 | 53.065 秒（首次运行 84.949 秒） |
| 峰值 RSS | 65,624 KiB，约 64.1 MiB（首次运行 66,016 KiB） |
| 五张相关表前后指纹 | 完全一致 |

按赛事开始年份统计：

| 年份 | 赛事 | 对阵 |
| ---: | ---: | ---: |
| 2023 | 22 | 1503 |
| 2024 | 51 | 2800 |
| 2025 | 48 | 2605 |
| 2026 | 39 | 1957 |

### 分页与异常

同一环境、同一观测窗口的分页元数据为：赛季第 1 页 `data=5,total=5,next=null`；赛事 2023/2024/2025/2026 各自第 1 页为 `46/49/49/39` 条且均满足 `totalCount`、`next=null`；对阵第 46 页为 38 条、`total=9038,next=null`。因此本轮实际完成 1 个赛季数据页、4 个赛事数据页和 46 个对阵数据页，共 51 个列表请求，无需额外请求终止空页。封面页面请求不计入列表分页数。

源站全局对阵总数已由前一日只读核实的 8990 增至 9038，进一步证明接口没有稳定快照，以下结果只代表本次扫描。

共输出 34 条待核对 warning：

- 1 条必要字段缺失：`52d5a835…` 的 `playersAllocated` 为 `null`，被跳过，没有用零值覆盖。
- 5 条“实际对阵数与官方有日期对阵数不一致”。
- 24 条“官方总对阵数与有日期对阵数不一致”。
- 4 个赛事没有官方对阵。

退出码 2 符合预期：扫描和 dry-run 均成功，但官方计数差异、空赛事和一条必要字段缺失需要人工核对，不能把本次接口快照描述成完整历史。`commit_state=not_started`，五张相关表的前后校验和一致；定向测试同时覆盖 dry-run 在图片镜像和缓存失效之前返回。按本次约 53–85 秒、约 64 MiB 峰值 RSS 和 46 个对阵数据页评估，2023 年至今一次性运行可行，无需为本轮增加断点或分批机制。

## 5.4 开发环境正式导入与恢复验收

- 执行日期：2026-09-09（UTC）
- 目标：用户指定的 `168.107.77.113` 开发主机，数据库 `chasing_points_dev`；未连接生产环境。
- 成功运行编号：`20260909T093427Z`；远端证据目录为 `/www/server/chasing-points/backfill-runs/20260909T093427Z-harden-wst-history-backfill/`。
- 固定参数：`--backfill --from 2023-01-01 --to 2026-09-09 --publish=true --include-qualifiers=true --game-type=1`。
- 最终 Linux arm64 二进制 SHA-256：`ddeaa065e623da713b41754bd88d62f81732df800a6e23a0d8af0c23ab003d82`；维护脚本 SHA-256：`73342c41a558aa5810ff0fb0aec75a75f7ed8ef0563f2eb5d205f75635f7e84a`。

目标主机盘点只有一个 `chasing-points.service` 实例，普通和热点 WST worker 都内嵌在该进程中。维护脚本先取得独占文件锁并设置 10 分钟安全恢复 timer，再停止整个服务；`service-stopped.txt` 记录 `inactive` 且 `MainPID=0`，维护期间也没有其他 `wst_sync` 进程，因此两类自动任务均已暂停。

首次操作尝试在备份恢复演练的临时库创建阶段被应用数据库账号权限拒绝，发生在最终 dry-run 和任何正式写入之前。异常路径自动恢复服务并通过健康检查，失败运行和原始备份保留在 `20260909T070740Z-harden-wst-history-backfill/`。随后只调整维护脚本的备份验证连接：通过宝塔面板自身解密接口在进程内取得 MySQL 管理凭据，仅用于临时恢复库；没有把密码或 DSN 写入参数、日志或证据。该修复先用失败轮备份实测恢复并逐表核对 60 张基础表成功，才开始新的维护运行。

### 备份与正式执行

成功运行在写入前生成备份：

- 文件：`/www/backup/database/mysql/all_backup/chasing_points_dev_pre_wst_backfill_20260909T093427Z.sql.gz`
- 大小：85,429 bytes；权限：`0600 root:root`
- SHA-256：`6eefe06abbc360eef13aca37dd6d9fb4e1e2a09185c910ff46f1906ca0fb5164`
- `gzip -t` 和 `sha256sum -c` 均通过；备份恢复到临时数据库后，60/60 张 `BASE TABLE` 的精确 `COUNT(*)` 与原库一致，随后临时库已删除。

三次同范围运行结果：

| 运行 | 状态 / 退出码 | 耗时 | `commit_state` | committed：球员 / 赛事 / 对阵 / 赛讯 |
| --- | --- | ---: | --- | ---: |
| 最终 dry-run | `needs_review` / 2 | 39.946 秒 | `not_started` | 0 / 0 / 0 / 0 |
| 第一次正式导入 | `needs_review` / 2 | 210.289 秒 | `committed` | 466 / 160 / 8,864 / 160 |
| 第二次幂等重跑 | `needs_review` / 2 | 116.779 秒 | `committed` | 466 / 160 / 8,864 / 160 |

三次运行都扫描 5 个赛季目录、4 个候选赛季、183 个赛事和 9,038 个全局对阵，选中 160 个赛事及 8,865 个对阵；其中 1 个缺少必要字段的对阵按规则跳过，实际准备 8,864 个。每次的 34 条 warning 与 5.3 的已知基线逐条一致，三个 `*.warning-gate.json` 均为 `passed=true`、`missing=[]`、`unexpected=[]`。因此退出码 2 继续表示官方计数差异、无对阵赛事和已跳过异常，而不是事务失败；正式运行的有效数据已经提交。

写入前后的目标范围活动记录为：官方球员 139 → 468、赛事 4 → 160、对阵 208 → 8,864、赛讯 4 → 160。这里的 committed 数量是本次事务处理量，不等同于新增量；已有官方数据按来源 ID 更新，未返回的历史记录未因补采被清理。160 条目标赛讯均已发布。

按赛事开始年份核对最终数据库：

| 年份 | 赛事 | 活动对阵 | 完赛对阵 | 赛讯 / 已发布 |
| ---: | ---: | ---: | ---: | ---: |
| 2023 | 22 | 1,502 | 1,502 | 22 / 22 |
| 2024 | 51 | 2,800 | 2,800 | 51 / 51 |
| 2025 | 48 | 2,605 | 2,605 | 48 / 48 |
| 2026 | 39 | 1,957 | 1,941 | 39 / 39 |

`integrity-after-first.tsv` 与 `integrity-after-second.tsv` 的全部检查均为 0，覆盖来源 ID 非空及唯一、赛事/对阵/赛讯关联、球员关联、对阵双方与胜者字段、赛讯球种/发布状态，以及每个目标赛事恰有一条活动赛讯。两次正式运行的四类事实哈希相同（首结果集 SHA-256 均为 `1037122a…b98cc`），来源 ID 到本地主键及关联快照逐字节相同（SHA-256 均为 `a0495eb1…f2b9a`），年度统计逐字节相同（SHA-256 均为 `34094f89…3531b`）。`wst_sync_job_states` 在两次补采期间也逐字节不变，证明显式补采没有推进自动任务游标。

官方抽查重新完整扫描 46 页、9,038 个唯一对阵，并在 2023、2024、2025、2026 各抽取一场已完成比赛。四场的赛事 ID/名称/日期、对阵归属、双方球员、比分、`Completed` 状态和已发布赛讯全部与数据库一致，比分样本依次为 `0:1`、`5:4`、`6:0`、`4:5`，`official-sample-check.json` 的 `failures=[]`。

### 自动任务恢复

正式验收结束后服务恢复为新 `MainPID=702670`，systemd 状态为 `active/running`，内部健康接口返回 `code=0, success=true`，安全恢复 timer 已清理。启动即运行的普通 WST worker 随后成功完成一次真实近期同步：`wst-auto-sync.last_successful_sync_at=2026-09-09 09:45:45 UTC`，状态行在 `09:46:28 UTC` 提交更新。

自动任务完成后再次执行完整关联检查，所有失败数仍为 0；2023–2025 的 121 个赛事来源 ID、本地主键、对阵/赛讯关联、年度数量和逐年抽样均与恢复前一致。`post-auto-verification.json` 记录 `historical_identity_unchanged=true`、`historical_samples_unchanged=true`，证明恢复近期自动同步没有清理历史范围。收尾时 transient 补采单元已停止并卸载（`LoadState=not-found`），主服务 PID 仍为 `702670` 且健康接口继续通过；安全恢复 timer、维护进程和独立 `wst_sync` 进程均不存在。

正式日志另记录 2 个赛事封面来源域名不在现有 WST 图片允许列表，系统按既有规则仅保留已有受管地址，否则置空，没有写入未托管外链；Redis 客户端也记录了一次 `maint_notifications` 兼容回退。两者均未造成事务、缓存版本更新或服务健康失败。官方接口本身仍有 34 条已知待核对项，因此本次验收只证明当前接口返回的数据已完整扫描、关联存储且可幂等重跑，不把它表述为官方历史百分之百完整。
