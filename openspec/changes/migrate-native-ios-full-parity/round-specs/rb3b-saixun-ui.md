# RB3b：赛讯 Tab 与赛事详情（界面）

上一轮已经把赛讯的模型、接口和规则迁好（`iOS/iOS/Features/News/`：`EventNewsAPI`、`EventNewsFilter`、`SaiXunFilter`、`SaiXunRules` 及本地化对照表）。本轮用它们做出两个页面。

## 本轮要读的小程序源码

- `uniapp/pages/tournament/index.vue` + `index.scss`：赛讯 Tab 根页面（导航标题「赛讯」）
- `uniapp/subPages/tournament/detail.vue` + `detail.scss`：赛事详情
- `uniapp/tests/tournament-page.test.mjs`：页面级行为的期望（能对应到页面模型的用例都搬过来）
- `uniapp/store/publicRead.js` 的 `loadEventNews` / `loadEventNewsView`：只为了解读取参数；原生端不做跨页面缓存

## 任务

1. **赛讯 Tab**（`iOS/iOS/Features/News/NewsHomeView.swift` 等）：替换 `MainTabView` 里赛讯 Tab 的占位根页面（只改那一处）。
   - 年份筛选、分区（进行中 / 即将开始 / 已结束等，按 `buildSaiXunListSections`）、赛事卡片（封面、状态徽标、标题、时间地点、倒计时等，按 `normalizeSaiXunCard`）、顶部统计（如模板里有，按 `buildSaiXunHeroStats`）、下拉刷新、滚动到底加载更多、空态、失败态，全部以 `index.vue` 为准。
   - Tab 根页面常驻：页面出现时按 60 秒新鲜度重读第一页（已有数据时静默刷新）。
   - 封面图：`SaiXunRules` 给出的图片来源是远程地址时用 `RemoteImage`；是「默认封面」时用 CDN 默认封面地址；加载失败保留占位底色。
   - 点卡片 → `.tournamentDetail(id:)`。
   - 受 `AppFlags.canShowUserTournamentActions` 控制的入口（如「创建赛事」）与小程序同步隐藏。
2. **赛事详情**（`NewsDetailView.swift` 等）：`.tournamentDetail(id:)` → 该页面，标题「赛事详情」。
   - 头部（封面、标题、状态、时间地点）、冠军（`resolveSaiXunChampion`）、赛程 Tab（`buildSaiXunScheduleTabs` / `resolveSaiXunScheduleTabKey`）、按天分组的对阵列表（`buildSaiXunScheduleDays`，每次多显示 `SAIXUN_SCHEDULE_PAGE_SIZE` 场，滚动到底继续显示）、球员名 / 国旗 / 头像（官方球员占位图：把 `uniapp/static/images/wst-player-placeholder.png` 复制进 asset catalog，命名 `WSTPlayerPlaceholder`）、比分与局数（`formatBestOfText`）、空态、失败态、下拉刷新，全部以 `detail.vue` 为准。
   - 小程序在详情页对进行中的赛事有定时刷新的话（看 `onShow` / `onHide` / `onUnload` 里的定时器），原生端用 `.task` + `Task.sleep` 实现同样的间隔，页面离开即停止；间隔与触发条件照搬。
   - 如果模板里有跳到「对阵图」（`bracket`）或其他受合规开关控制的入口，按开关同步；未迁移的目的地走占位路由。

## 测试

页面模型：首次加载、年份切换重读并丢弃旧结果、分页追加、刷新失败保留数据、空、失败重试、60 秒新鲜度；详情模型：读取、Tab 切换、分批显示、定时刷新只在进行中触发且页面离开后停止（用可注入的间隔与时钟）。搬运 `tournament-page.test.mjs` 里适用的用例。快照：`news-home`（三个分区都有内容）、`news-home-empty`、`news-home-error`、`news-detail-live`（进行中、两天赛程）、`news-detail-finished`（有冠军）、`news-detail-error`。

## 并行开发提醒

其他开发线正在修改 `iOS/iOS/Features/Profile/**`、`Features/Settings/**`、`Features/Reputation/**`、`Features/Opponents/**`、`Features/Stats/**`、`Features/Venue/**`、`Features/Spectate/**`、`iOS/iOS/SettingsView.swift`、`iOS/iOS/SessionStore.swift` 及其测试，本轮不要碰这些文件。`MainTabView.swift` 只替换赛讯 Tab 的根页面，不动其他行。
