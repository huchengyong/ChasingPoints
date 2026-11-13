# R2.4：竞技分析、表现分析

本轮迁移两个页面。

## 本轮要读的小程序源码

- `uniapp/subPages/user/statsDetail.vue`（1150 行，+ 同目录 scss）、`uniapp/utils/stats-detail.js`、`uniapp/tests/stats-detail.test.mjs`、`uniapp/api/stats.js`：竞技分析。主接口 `GET /api/stats/overview`（`GetStatsOverviewReq/Resp` 及其嵌套类型）。
- `uniapp/subPages/user/performance.vue`（+ scss）、`uniapp/utils/member-performance-view.js`、`uniapp/store/memberPerformance.js`、`uniapp/utils/member-performance-cache.js`、`uniapp/tests/member-performance-view.test.mjs`、`uniapp/tests/member-analysis-pages.test.mjs`、`uniapp/api/member.js`：表现分析（会员分析页）。接口 `GET /api/member/performance`（`GetMemberPerformanceReq/Resp`）。
- 契约：`backend/chasing_points.api` 里上述类型；锁定态 / 非会员时服务端实际返回什么，去读 `backend/internal/logic/member/` 的实现确认。

## 任务

1. **竞技分析**（`iOS/iOS/Features/Stats/`）：`.statsDetail(gameType:)` → `StatsDetailView`，标题「竞技分析」。
   - `stats-detail.js` 的规则逐条迁移并搬测试。
   - 页面的球种 Tab、各统计区块（总览、趋势、对手强度、时长、单杆 / 高分等，以模板为准，一个都不能少）、空态、失败态、未登录态以 `statsDetail.vue` 为准；传入的 `gameType` 决定初始 Tab。
   - 图表：用系统 `Charts` 框架（Swift Charts，iOS 16+，属于系统框架）画折线 / 柱状图；图表配色用 `AppTheme.primary`、`palette.brandStrong`、`palette.success`、`AppTheme.danger`、`palette.info`。小程序里用 CSS 画的进度条 / 占比条，原生端用 SwiftUI 形状画，不用 Charts。
   - 「表现分析」入口 → `.performance`（如果小程序跳转带球种，就给 `.performance` 增加 `gameType` 关联值，并同步其他引用处）。
2. **表现分析**（同目录）：`.performance` → `PerformanceView`，标题「表现分析」。
   - `member-performance-view.js` 的规则逐条迁移并搬测试（含 `resolveLockedView`）。
   - 会员可见内容、非会员的锁定态以 `performance.vue` 为准。开通入口只由 `AppFlags.canShowPaidMembershipEntry` 控制（合规模式下不显示开通按钮，其余锁定说明照常显示）——对应小程序 `canShowPurchase`。
   - 小程序的 `memberPerformance` store 是为了跨页面缓存；原生端不做跨页面缓存，页面模型里按球种保留本页已读结果即可。
   - 点对局 → `.matchReport(matchId:)`。

## 测试

搬运 JS 用例；接口门面的路径 / 参数；页面模型：首次加载、切换球种（旧结果丢弃、已读球种不重复请求）、空数据、失败重试、未登录不请求、身份变化；表现分析的锁定态在 `canShowPaidMembershipEntry` 为 false 时不含开通动作。快照：`stats-detail`（数据丰富）、`stats-detail-empty`、`stats-detail-error`、`performance-member`、`performance-locked`。

## 并行开发提醒

其他开发线正在修改 `iOS/iOS/Features/Ranking/**`、`Features/Rank/**`、`Features/News/**`、`Features/Venue/**` 及其测试，本轮不要碰这些文件。
