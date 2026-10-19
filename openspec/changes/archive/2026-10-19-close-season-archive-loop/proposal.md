## Why

「赛季档案」目前只能看进行中的赛季，「打排位 → 换季结算 → 回顾」在回顾这一环断开：报告按钮要求 `season.status === 2`，但页面只渲染进行中赛季（在线解析固定返回 `status = 1`），赛季报告页在产品上不可达；也没有历届赛季列表，换季通知只跳荣誉墙。同时进行中赛季的「当前排名」依赖只在结算时写入的 `final_rank`，整季恒为 0，用户无法知道自己本赛季排第几。

## What Changes

- 新增认证接口 `GET /api/season/history`：按结束时间倒序有界分页返回已结算赛季，并附带当前用户在指定球种的赛季记录（含最终排名）；未参加该球种排位时记录为空，不伪装成零值。
- 新增 UniApp「历届赛季」分包页：可切换球种，展示每届名次、场次、胜场与胜率；有记录的赛季可进入既有赛季报告页。
- 「赛季档案」页移除永远不可见的「赛季报告」按钮，改为在任意赛季状态下都可进入的「历届赛季」入口；进行中赛季展示实时排名，本人尚无记录时给出上榜指引。
- 赛季 overview 增加 `my_rank`：按与榜单、结算一致的排序计算本人在当前赛季该球种的实时名次，并通过 `availability.my_rank` 独立表达读取失败。
- 新生成的换季通知：对在该届有赛季记录的用户，跳转目标改为该届赛季报告（携带其主要参赛球种）；无赛季记录的用户保持跳荣誉墙赛季 Tab。「我的」页换季弹窗在目标为赛季报告时提供「查看回顾」。已持久化的历史通知不回写。
- 不改变赛季排期、结算阶段、榜单排序与 `final_rank` 的写入时机；不新增表或索引。

## Capabilities

### New Capabilities

- `season-archive`: 历届赛季列表、赛季回顾入口（赛季档案页与换季通知）、进行中赛季实时排名的客户端展示与无记录引导。

### Modified Capabilities

- `read-api-composition`: 「赛季首屏必须提供概览接口」增加本人实时名次及其 partial 语义。

## Impact

- 契约：`backend/chasing_points.api` 新增 `GetSeasonHistory` 及类型、`GetSeasonOverviewResp.my_rank`；修改后立即运行 goctl，不手改生成产物。均为新增字段/接口，不破坏既有客户端。
- 后端：`internal/logic/season/`（overview、history）、`internal/model/season.go`（已结算赛季分页、本人记录批量读取、领先人数计数）、`internal/logic/season_settlement_service.go`（换季通知载荷与目标）；查询复用现有 `idx_season_game_rank`、`idx_seasons_end_id_status_start` 与 `uk_season_user_game`，需补 query-count 与 MySQL EXPLAIN 回归。
- UniApp：`subPages/season/index.vue`、新增 `subPages/season/history.vue` 并注册到 `pages.json` 赛季分包、`api/season.js`、`utils/honor-wall.js` 换季弹窗及对应测试。
- 原生 iOS 当前没有赛季页面，本变更只做新增字段，不影响 iOS；管理后台不涉及。
- 不包含：段位软重置（`rank_reset_ratio` 仍为独立产品决策）、历届赛季榜单浏览、报告页内容改版、结算后迟到写入的一致性问题。
