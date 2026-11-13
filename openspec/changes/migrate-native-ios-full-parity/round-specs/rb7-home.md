# RB7：首页

本轮迁移首页（`uniapp/pages/index/index.vue`），替换 `MainTabView` 里首页 Tab 的占位根页面。

## 本轮要读的小程序源码

- `uniapp/pages/index/index.vue`（773 行）+ `index.scss`
- `uniapp/utils/home-index.js` + `home-index.test.mjs`；`challenge-view.js` + `challenge-view.test.mjs`（约球卡用到的部分）；`uniapp/utils/user-homepage.js` 的 `resolveStatusCardContent` / `formatDuration`（如首页用到）；`match-core-flow.js`、`ongoing-match-guard.js`（继续当前对局的去向）
- 已迁移可直接复用的：`ActivityStore`（当前对局、当前约球、`challengeAvailable`）、`SpectatorLobbyRules`（首页「正在进行」预览，`homeLiveMatchLimit`）、`SaiXunRules.normalizeSaiXunCard`（赛事卡）、`RankingAPI` / `LeaderboardItem`、`VenueAPI` + `LocationProviding`（附近球房）、`MemberAvatarView`
- 数据来源以 `index.vue` 的实际调用为准：公开大厅进行中对局（`GET /api/public/matches`）、赛讯列表首条、排行榜摘要（`GET /api/public/rank/leaderboard-summary`，`GetLeaderboardSummaryReq/Resp`）、附近球馆（`GET /api/venue/nearby`）、活动状态（`ActivityStore`）

## 任务

`iOS/iOS/Features/Home/`：`HomeView` + `HomeModel`，导航标题「追分」。小程序是自定义导航栏（标题「追分」+ 问候语），原生端用大标题或在内容顶部放同样的标题与问候语，问候语规则照搬。

自上而下的区块、每个区块的加载 / 空 / 失败 / 重试文案，全部以模板为准：

1. **开局卡**（首页唯一的主操作）：加载骨架；有进行中 / 待确认对局 → 对局卡（状态、球种与类型、时长、双方头像昵称比分、「我」标记、继续按钮），点击 → `.liveMatch(matchId:)`（实时计分页未迁移，先走占位路由）；已登录无对局 →「来一局」+「发起对局」（→ `navigator.requestScan`）+「约球」（→ `.challengeCompose`）；未登录 →「登录后开始记分」+「登录 / 注册」。
2. **约球卡**：按 `challenge-view.js` 的规则显示当前约球状态与动作；`ActivityStore.challengeAvailable == false` 时显示「约球状态读取失败，点击重试」。
3. **正在进行**：公开大厅进行中对局预览 +「观赛大厅」→ `.spectatorLobby`；点对局的去向与观赛大厅一致。
4. **赛事**：置顶赛讯卡 +「全部赛讯」→ 切到赛讯 Tab（`navigator.select(.news, ...)`）；点卡片 → `.tournamentDetail(id:)`。
5. **排行榜**：中式八球前三名 +「完整榜单」→ `.ranking`；已登录显示「我」的名次文案（规则照搬）。
6. **附近球房**：定位成功显示附近球馆；定位失败 / 拒绝的提示与重试照搬；「查看球房」→ `.venueList`；点球馆 → `.venueDetail(id:)`。
7. 其余模板里有的区块（如快捷入口、分享 / 二维码相关）逐个对照：依赖微信专属能力的不显示，其余迁移。

读取规则：

- 各区块互不阻塞，单个区块失败只影响自己（对应模板里每个区块各自的 error 分支）；全部失败且没有任何内容时显示整页错误态（`homePageError`）。
- Tab 根页面常驻：页面出现时按新鲜度重读（公开数据 60 秒；活动状态走 `ActivityStore.refreshIfNeeded()`）；下拉刷新强制重读全部。
- 登录 / 退出后重读与身份相关的区块（开局卡、约球卡、「我」的名次）。
- 定位只在首页需要附近球房时请求一次；用户拒绝后不反复弹授权。

## 测试

搬运 `home-index.test.mjs`、`challenge-view.test.mjs` 里适用的用例；`HomeModel`：各区块独立成功 / 失败、整页错误的判定、访客与登录态差异、新鲜度、下拉刷新、身份变化、定位被拒的降级；快照：`home-guest`、`home-signed-in-idle`（无对局、有约球邀请）、`home-ongoing-match`、`home-partial-error`（两个区块失败）、`home-all-error`。快照里定位用替身，不渲染真实地图。

## 并行开发提醒

其他开发线正在修改 `iOS/iOS/Features/Profile/**`、`Features/Achievement/**`、`Features/Season/**`、`Features/Gear/**`、`Features/Member/**`、`Features/Stats/**`、`Features/Match/**`、`iOS/iOS/SessionStore.swift` 及其测试，本轮不要碰这些文件。`MainTabView.swift` 只替换首页 Tab 的根页面。
