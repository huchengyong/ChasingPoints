## Context

- 赛季页 `uniapp/subPages/season/index.vue` 只消费 `GET /api/season/overview`，overview 只解析当前窗口；在线解析会把当前赛季 `status` 固定为 1，因此页面上 `season.status === 2` 的「赛季报告」按钮永远不渲染，`subPages/season/report.vue` 没有可达入口。
- 历届数据已经齐备：换季发布事务把旧赛季置为 `status = 2`，`season_records` 在结算阶段写入 `final_rank`；`GET /api/season/report` 可按 `season_id + game_type` 读取任一赛季报告。缺的是"列出历届赛季"的读接口与客户端入口。
- 进行中赛季的 `final_rank` 恒为 0，页面「当前排名」不显示。`season_records` 已有 `idx_season_game_rank (season_id, game_type, end_rank_score, wins, id)`，榜单与结算都按 `end_rank_score DESC, wins DESC, id DESC` 排序；全站榜单已有「领先人数 + 1」的本人名次先例（`higherRankingCondition`）。
- 换季通知在 `persistRolloverNotificationsWithTx` 中按用户批量生成，`url` 固定为荣誉墙赛季 Tab；通知中心按 `data.url` 通用跳转，「我的」页通过 `presentLatestSeasonRollover` 弹窗展示最近一条未读换季通知。
- 约束：读取保持纯读、有界、SQL 次数固定；用户级读取按 `userId + authGeneration` 隔离；契约以 `.api` 为准并由 goctl 生成；原生 iOS 当前没有赛季页面。

## Goals / Non-Goals

**Goals:**
- 用户可以从「赛季档案」进入历届赛季列表，并进入任一届自己有记录的赛季报告。
- 进行中赛季展示本人实时名次，无记录时给出上榜指引。
- 新的换季通知与「我的」页弹窗直达该届回顾。
- 全部为新增接口/字段，旧客户端行为不变；不新增表、索引或迁移。

**Non-Goals:**
- 段位软重置与 `rank_reset_ratio` 的应用。
- 历届赛季榜单浏览、赛季报告页内容改版（趋势图、文案等）。
- `leaderboard_total`、「第 N 名 / 共 M 人」展示。
- 结算后迟到投影导致的 `final_rank` 一致性问题（另行评估）。
- 原生 iOS 赛季页面。

## Decisions

### D1：历届列表使用独立接口与独立分包页，不并入 overview
- 新增 `GET /api/season/history`（`jwt: Auth` 分组），请求 `game_type`（默认 3）、`page`（默认 1）、`page_size`（默认 10，上限 20）；响应 `success`、`total`、`list: [{ season: SeasonInfo, record: SeasonRecordInfo | null }]`。
- 新增 `subPages/season/history.vue` 并注册到 `pages.json` 的 `subPages/season` 分包。
- 理由：overview 是高频首屏聚合接口，历届回顾是低频操作；入口本身不依赖数据，可以静态展示，不需要为它在首屏增加 SQL。
- 备选：overview 附带「上一届」摘要。否决：每次进入赛季页多 1～2 条 SQL，只为一个入口文案。

### D2：已结算以 `seasons.status = 2` 判定
- 发布事务在同一事务内写 `status = 2` 与 `season_settlements.status = completed`；连续与 legacy 两种换季路径都经过发布。
- 备选：JOIN `season_settlements` 判定 completed。否决：多一张表、无额外正确性收益。

### D3：历届列表固定 3 条 SQL，不加共享缓存
1. `COUNT(*) FROM seasons WHERE status = 2`
2. `SELECT ... FROM seasons WHERE status = 2 ORDER BY end_date DESC, id DESC LIMIT ? OFFSET ?`（`idx_seasons_end_id_status_start`）
3. `SELECT ... FROM season_records WHERE season_id IN (<本页 ID>) AND user_id = ? AND game_type = ?`（`uk_season_user_game`），在内存按 `season_id` 回填。
- 本页无赛季时跳过第 3 条。SeasonInfo 复用 `buildSeasonInfo` 计算权威边界，SeasonRecordInfo 复用 `buildSeasonRecordInfo`。
- 不加 Redis：响应含 viewer 个性化记录，不适合公共缓存；`seasons` 按月增长、数据量小。
- 读取失败返回 `success: false` 并记日志，与现有赛季榜单接口一致。

### D4：本人实时名次放在 overview 顶层 `my_rank`
- 计算：`1 + COUNT(*)`，条件为同 `season_id`、`game_type` 且排序在本人之前，展开为 `end_rank_score > s OR (end_rank_score = s AND wins > w) OR (end_rank_score = s AND wins = w AND id > i)`，与 `higherRankingCondition` 同形，走 `idx_season_game_rank` 范围扫描。
- 仅在本人记录存在时执行；记录为空则 `my_rank = 0` 且 `availability.my_rank = true`；记录区块不可用时 `availability.my_rank = false`；计数失败单独 partial。
- 放在 overview 顶层而非 `SeasonRecordInfo`：`SeasonRecordInfo` 同时被 my-record、report、history 复用，加入只在 overview 填充的字段会形成"部分接口为 0"的歧义。
- 备选：Redis ZSET 维护赛季名次。否决：新增写路径与一致性负担，当前规模不需要。备选：从首屏榜单推算。否决：只覆盖前 20 名。

### D5：换季通知目标在生成时按记录决定，不回写历史
- `persistRolloverNotificationsWithTx` 已加载该届全部 `records`（结算阶段本就读取），在内存为每个用户选出主要球种：`matches_played` 最多，相同取较小 `game_type`。有记录的用户 `url = /subPages/season/report?id={from}&game_type={g}`，载荷新增 `report_game_type`；无记录用户保持荣誉墙 URL。
- 幂等键 `season_rollover:{from}:{to}`、阶段 checkpoint 与提交后发送 Push/WS 的时机均不变；不新增 SQL。
- 通知中心已按 `data.url` 通用跳转，无需修改；`honor-wall.js` 的换季弹窗在 `url` 指向赛季报告时显示「查看回顾 / 知道了」，两者都先标记已读，确认后再跳转。跳转函数通过参数注入，保持纯函数可测。

### D6：赛季档案页改动保持在现有读取链路内
- 删除 `season.status === 2` 的报告按钮与 `goReport`。
- 在 `season-content` 内、与赛季卡同级处放「历届赛季」入口（不放在 `hasActiveSeason` 模板中），携带当前球种跳转；加载中与整页错误态不显示。
- 「当前排名」改用 `my_rank`；记录区块可用且 `record` 为空时展示上榜指引卡；`my_rank` 不可用时隐藏名次行，其余照常。
- 历届页沿用 `async-page-state` 与 `userId:authGeneration` 身份键，球种切换与加载更多通过 requestId 丢弃迟到响应；`onShow` 只在身份变化时重载（历届只在换季时变化，每次进入页面都会重新创建并请求）。

## Risks / Trade-offs

- [低名次用户在大赛季中计数扫描较多索引项，成本 O(名次)] → 与榜单已有的 `COUNT(*) total` 同量级；以 MySQL EXPLAIN 回归约束必须走 `idx_season_game_rank` 范围且无 filesort，并依赖既有慢 SQL 观测；规模增长后再评估缓存。
- [首屏榜单有 30 秒 Redis 缓存而 `my_rank` 直读数据库，短时间内名次与榜单位置可能不一致] → 接受；比赛结算会使榜单缓存版本失效，赛季 scope 失效后页面重载。
- [「主要球种」规则未必符合用户预期] → 报告页本身支持切换球种；规则集中在一个纯函数，便于调整。
- [legacy 手工赛季中 `status = 2` 的旧数据会出现在历届列表] → 属于真实历届数据，按预期展示。
- [OR 展开的计数条件在 SQLite 测试与 MySQL 执行计划不同] → query-count 测试用 SQLite，执行计划以 MySQL EXPLAIN 测试为准。

## Migration Plan

1. 无数据库迁移。修改 `.api` 后立即运行 goctl，补齐 logic、model 与测试。
2. 先部署后端：新增字段与接口对旧客户端透明；部署后新生成的换季通知携带报告 URL，旧客户端的通知中心同样可以打开既有报告页。
3. 再发布 UniApp；App 与微信小程序编译和真机验收后上线。
4. 回滚：客户端可独立回退；后端改动均为新增，回退后已生成的通知仍指向已存在的报告页。

## Open Questions

- 「主要球种」按场次最多是否符合产品预期，还是应改为名次最好？
- 历届赛季页是否需要展示该届榜单前几名（当前列为非目标）？
