## 1. API 契约与生成代码

- [x] 1.1 在 `backend/chasing_points.api` 赛季需登录分组新增 `GetSeasonHistory`（`get /history`）及 `GetSeasonHistoryReq`（`game_type` 默认 3、`page` 默认 1、`page_size` 默认 10）、`SeasonHistoryItem`、`GetSeasonHistoryResp`（`success`、`total`、`list`），并为 `GetSeasonOverviewResp` 增加 `my_rank`。
- [x] 1.2 修改 `.api` 后立即在 `backend/` 运行 goctl，检查生成 diff 只包含新增类型、路由与 handler，确认没有遗留 todo logic。

## 2. 后端：本人实时名次

- [x] 2.1 先为 `SeasonRecordModel` 增加领先人数计数测试，覆盖分数领先、同分胜场领先、同分同胜场按 `id` 降序、不同球种与不同赛季隔离。
- [x] 2.2 实现 `CountAheadInSeason`（或同义方法），条件与榜单排序 `end_rank_score DESC, wins DESC, id DESC` 一致，并限定 `season_id`、`game_type`。
- [x] 2.3 在 overview 中仅在本人记录存在时计算 `my_rank`；无记录返回 0 且 `availability.my_rank = true`；记录区块不可用或计数失败时 `availability.my_rank = false` 并追加 partial error，不影响记录与榜单。
- [x] 2.4 补 overview 测试：名次与 `GetSeasonLeaderboard` 分页位置一致、无记录、计数失败 partial、`ReadMode = disabled` 时仍返回名次；更新固定 SQL 数测试，确认 1 行与 100 行的 SQL 次数相同。
- [x] 2.5 在 `migrations/mysql_explain_test.go` 增加名次计数用例，断言使用 `idx_season_game_rank` 且无 filesort。

## 3. 后端：历届赛季列表

- [x] 3.1 先为 `SeasonModel` 增加已结算赛季分页与计数测试，覆盖只含 `status = 2`、`end_date DESC, id DESC` 稳定排序与分页边界。
- [x] 3.2 实现已结算赛季的计数与分页方法，以及按 `season_id IN (...) AND user_id AND game_type` 批量读取本人记录的方法。
- [x] 3.3 实现 `GetSeasonHistoryLogic`：归一化分页（非正页码取 1，`page_size` 上限 20），复用 `buildSeasonInfo` 与 `buildSeasonRecordInfo`，无记录项返回 `record = null`，本页为空时跳过记录查询，读取失败返回 `success: false` 并记日志。
- [x] 3.4 补 history logic 测试：未认证拒绝、进行中/未开始赛季不出现、无记录为 null、分页越界归一、固定 SQL 数（本页 1 条与 20 条相同）。
- [x] 3.5 在 `migrations/mysql_explain_test.go` 增加已结算赛季分页用例，断言使用 `idx_seasons_end_id_status_start` 且无 filesort。

## 4. 后端：换季通知直达回顾

- [x] 4.1 先扩展换季结算测试：有记录用户的通知 `url` 指向 `/subPages/season/report?id={from}&game_type={g}` 并带 `report_game_type`；多球种按场次最多、同场次取较小球种；只有挑战快照的用户保持荣誉墙 URL；重复执行不产生重复通知。
- [x] 4.2 在 `persistRolloverNotificationsWithTx` 中基于已加载的 `records` 计算每个用户的主要球种并写入载荷与 URL，不新增 SQL，不改变幂等键与提交后发送时机。

## 5. UniApp：API 门面与赛季档案页

- [x] 5.1 在 `uniapp/api/season.js` 新增 `getSeasonHistory`，只组织路径与参数。
- [x] 5.2 `subPages/season/index.vue` 删除 `season.status === 2` 的报告按钮与 `goReport`，新增在所有赛季状态下可见的「历届赛季」入口，并携带当前球种跳转。
- [x] 5.3 「当前排名」改用 overview 的 `my_rank`；记录区块可用且无记录时展示上榜指引；`availability.my_rank = false` 时隐藏名次行，不显示第 0 名。
- [x] 5.4 更新 `tests/season-page.test.mjs` 与 `tests/season-page-mount.test.mjs`：入口在 active / not_started / unavailable 下渲染、名次展示与隐藏、无记录指引，以及原有单次请求与 scope 失效行为保持不变。

## 6. UniApp：历届赛季页

- [x] 6.1 新增 `subPages/season/history.vue` 并在 `pages.json` 的 `subPages/season` 分包注册，标题为「历届赛季」，使用系统导航栏，样式与深色模式遵循 DESIGN.md。
- [x] 6.2 实现球种切换、首屏加载、加载更多、空状态与失败重试；有记录项进入 `subPages/season/report?id=&game_type=`，无记录项标注未参加且不可点击。
- [x] 6.3 使用 `async-page-state` 与 `userId:authGeneration` 身份键隔离请求；切球种、加载更多与切号时丢弃迟到响应；`onShow` 只在身份变化时重载。
- [x] 6.4 新增历届页测试，覆盖首次进入单次请求、切球种、加载更多与 `total` 截止、空列表、失败重试、切号与迟到响应；并纳入 `page-state-contract` 与 `page-theme-contract` 检查。

## 7. UniApp：换季弹窗

- [x] 7.1 扩展 `utils/honor-wall.js` 的 `buildSeasonRolloverModal`：目标为赛季报告时显示「查看回顾 / 知道了」，否则保持单按钮。
- [x] 7.2 扩展 `presentLatestSeasonRollover`：两种选择都先标记已读，确认「查看回顾」后再通过注入的跳转函数打开报告；`pages/user/index.vue` 传入跳转函数。
- [x] 7.3 更新 `tests/honor-wall.test.mjs`，覆盖新旧载荷、确认与取消路径、标记已读失败时不跳转。

## 8. 验证与收尾

- [x] 8.1 在 `backend/` 运行 `go test ./internal/logic/season/ ./internal/logic/ ./internal/model/` 后运行 `go test ./...` 与 `go build .`，并清理构建产物。
- [x] 8.2 在已授权的隔离 MySQL 运行 EXPLAIN 回归；记录见 `docs/testing/season-archive-loop-mysql-explain.md`。
- [x] 8.3 在 `uniapp/` 运行 `node --test tests/*.test.mjs`。
- [x] 8.4 App 与微信小程序编译、真机检查赛季档案、历届赛季、报告跳转与换季弹窗（浅色/深色），结果按「自动化 / 人工确认 / 未执行」记录到 `docs/testing/`。
- [x] 8.5 核对原生 iOS 不受新增字段影响，`git diff --check` 通过，并确认改动未使指南失效。
