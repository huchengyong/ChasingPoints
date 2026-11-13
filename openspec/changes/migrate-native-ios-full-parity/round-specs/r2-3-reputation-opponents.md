# R2.3：信誉明细、过往对手、交锋记录

本轮迁移三个页面，以及它们共用的时间格式化工具。

## 本轮要读的小程序源码

- `uniapp/subPages/user/reputation.vue`（+ 同目录 scss）：信誉明细。数据：`useUserOverviewStore` 里的信誉分区（原生端用 `ProfileAPI.overview()` 的 `availableReputation`，或单独读 `GET /api/user/reputation`——以页面实际用法为准）+ `GET /api/user/reputation/logs`（分页）。
- `uniapp/subPages/user/opponentRecord.vue`（+ scss）、`uniapp/utils/opponent-record.js`、`uniapp/tests/opponent-record.test.mjs`：过往对手。接口见 `uniapp/api/match.js` 的 `getOpponentList`。
- `uniapp/subPages/user/h2hRecord.vue`（+ scss）、`uniapp/utils/h2h-record.js`、`h2h-view-model.js`、`h2h-copywriting.js` 及各自的测试：交锋记录。接口 `getH2HOverview`、`getH2HHistory`。
- `uniapp/utils/format.js` + `uniapp/tests/format.test.mjs`：时间格式化。
- 契约：`backend/chasing_points.api` 里上述接口对应的 Req / Resp 类型（自己按路径查）。

## 任务

1. **时间格式化**（新建 `iOS/iOS/Shared/AppFormat.swift`）：把 `format.js` 的全部导出迁成 `enum AppFormat` 的静态函数（`now: Date` 与 `Calendar` 可注入，默认北京时间公历），搬运 `format.test.mjs` 全部用例。不要改 `MatchHistoryModels.swift` 里现有的 `MatchTimeFormatting`。
2. **信誉明细**（`iOS/iOS/Features/Reputation/`）：`.reputation` → `ReputationView`，标题「信誉明细」。页面结构、信誉分 / 状态卡、流水列表每一行的内容、分页与空态、错误态全部以 `reputation.vue` 为准。流水分页用 `PagedHistoryList`。
3. **过往对手**（`iOS/iOS/Features/Opponents/`）：`.opponentRecord` → `OpponentRecordView`，标题「过往对手」。`opponent-record.js` 的规则逐条迁移并搬测试；页面的筛选 / 排序 / 分页 / 空态以 `opponentRecord.vue` 为准；点对手 → `.h2hRecord(opponentId:)`（如果小程序跳转时还带了别的参数，如球种，就给路由 case 增加相应的关联值）。
4. **交锋记录**（同目录）：`.h2hRecord` → `H2HRecordView`，标题「交锋记录」。`h2h-record.js`、`h2h-view-model.js`、`h2h-copywriting.js` 的规则逐条迁移并搬测试；页面的概览卡、文案、历史列表（分页）、球种切换以 `h2hRecord.vue` 为准；点历史对局 → `.matchReport(matchId:)`。
5. 小程序用 `useUserDataInvalidationStore` 做「写后失效」的地方，原生端本轮先不做跨页面失效（页面每次进入都重新读取），在交付说明里列出这些失效点，后面统一处理。

## 测试

除搬运 JS 用例外：每个接口门面的路径 / 参数测试；每个页面模型的首次加载、分页、空、失败重试、身份变化丢弃迟到结果；快照：`reputation`（有流水）、`reputation-empty`、`opponent-record`（5 个对手）、`opponent-record-empty`、`h2h-record`（有胜有负）、`h2h-record-error`。

## 并行开发提醒

其他开发线正在修改 `iOS/iOS/Features/Ranking/**`、`Features/Rank/**`、`Features/News/**` 及其测试，本轮不要碰这些文件。
