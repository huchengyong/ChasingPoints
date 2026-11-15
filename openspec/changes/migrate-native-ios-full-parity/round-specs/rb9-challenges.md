# RB9：约球（发起约球、等待对手、约球列表）

## 先修正上一轮的一个界面问题（改完再做本轮任务）

好友列表（`iOS/iOS/Features/Friends/FriendListView.swift`）每一行卡片里画了一个箭头，卡片外右侧又多出一个灰色箭头——这是 `List` 里的 `NavigationLink` 自动加的系统指示符，同时把卡片挤窄了（看 `friend-list-data` 快照）。去掉系统指示符（常用做法：行内容放在 `ZStack` 里，`NavigationLink(value:) { EmptyView() }.opacity(0)` 叠在下面；或改用 `Button { navigator.open(...) }` 加 `.buttonStyle(.plain)`），保留卡片里自绘的那一个，并让卡片左右撑满。`Features/Friends/` 里其他列表页（好友申请、黑名单、添加好友）同样检查一遍。改完自己导出快照看一眼。

## 本轮要读的小程序源码

- `uniapp/subPages/match/challengeCompose.vue`、`challengeWaiting.vue`、`uniapp/subPages/social/challenges.vue`（863 行）及各自样式
- `uniapp/utils/challenge-view.js`、`challenge-time.js`、`challenge-entry.js` 及测试 `challenge-view.test.mjs`、`challenge-time.test.mjs`、`challenge-entry.test.mjs`、`challenges-auth.test.mjs`（能对应到规则 / 页面模型的都搬）
- `uniapp/api/challenge.js`；页面对 `store/activity.js` 的读写（当前约球、收到的邀请数、`challengeServerTime`、标脏）
- 契约：`backend/chasing_points.api` 的 `/api/challenge/*` 全部接口与类型；各状态流转与失败原因读 `backend/internal/logic/challenge/` 确认。规格 `openspec/specs/quick-match-invites/spec.md`。
- 好友选择：`challengeCompose.vue` 怎么选对手（好友列表 / 传入的对手参数）按页面实际代码。

## 任务

`iOS/iOS/Features/Challenge/`：

1. 规则迁移：三个 utils 的全部导出（`challenge-view.js` 如已在首页那一轮迁过一部分，补齐其余并去重），搬运测试。涉及时间的规则（约球时间选项、过期判断、与服务端时间的校准 `challengeServerTime`）把 `now` 和服务端时间作为参数传入。
2. **发起约球**：`.challengeCompose` → `ChallengeComposeView`，标题「发起约球」。如果小程序进入时可带对手参数，给路由加可选关联值并同步引用处。对手选择、球种、对局类型、时间 / 地点 / 留言等字段与校验、发送（`POST /api/challenge/send`）及各失败提示（对方仅允许好友约球、已有进行中约球等）以页面为准。成功 → 等待页。
3. **等待对手**：新增路由 `case challengeWaiting(challengeId: Int64)`，标题「等待对手」。状态展示、倒计时、取消 / 放弃 / 进入对局（`/enter`、`/leave`、`/cancel`、`/abandon`）、被接受 / 拒绝 / 过期后的去向，以 `challengeWaiting.vue` 为准。状态变化来自轮询还是实时通道按页面实际实现（用户实时通道的通知更新会让 `ActivityStore` 标脏——可据此触发重读）；计时用可注入时钟。
4. **约球列表**：`.challenges(...)` → `ChallengesView`，标题「约球」。收到 / 发出 / 历史三个 Tab（路由参数决定初始 Tab）、接受 / 拒绝 / 取消等操作、历史分页、空态，以 `challenges.vue` 为准。接受后进入对局的去向与小程序一致（实时计分页走 `.liveMatch(matchId:)`）。
5. 所有写操作成功后 `ActivityStore.markDirty()` 并触发补读，使首页约球卡和消息角标同步。

## 测试

搬运 JS 用例；接口门面；三个页面模型（各状态流转、倒计时到期、写操作成功 / 失败 / 进行中锁、身份变化、迟到结果丢弃）；快照：`challenge-compose`、`challenge-compose-errors`、`challenge-waiting`、`challenge-waiting-accepted`、`challenges-received`、`challenges-sent`、`challenges-history`、`challenges-empty`。

## 并行开发提醒

其他开发线正在修改 `iOS/iOS/Features/Profile/**`、`Features/Achievement/**`、`Features/Season/**`、`Features/Gear/**`、`Features/Member/**`、`Features/Stats/**`、`Features/Match/**`、`iOS/iOS/SessionStore.swift` 及其测试，本轮不要碰这些文件。
