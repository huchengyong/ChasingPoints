# RB8：好友（好友列表、好友申请、添加好友、好友主页、黑名单）

## 先修正一个测试缺陷（改完再做本轮任务）

`iOS/iOSAuthTests/ActivityStoreTests.swift` 的 `testMarkDirtyDuringReadSurvivesResponseAndRefreshIfDirtyReloads` 每次要跑 30 秒：桩里的 `gate.wait()` 没有超时、只被 `signal()` 一次，`refreshIfDirty()` 发出的第二个请求也进了同一个桩，一直卡到 `APIClient` 的 30 秒请求超时才失败。断言只看了请求次数，所以测试通过了，但「补读成功」其实没有被验证。

改法：用 `LockedCounter`（`TestSupport.swift` 里已有）区分第几次请求——只有第一次请求等闸门，第二次直接返回一个不同的未读数（例如 5）；在测试末尾补断言 `activity.unreadCount == 5`。同一文件里其他用 `DispatchSemaphore` 当闸门的测试也检查一遍，确保没有请求会卡到超时（`gate.wait(timeout: .now() + 5)` 这种带超时的写法可以保留）。改完后这个测试类的总耗时应在 1 秒以内，在交付说明里贴出 `ActivityStoreTests` 的耗时。

## 再补一处首页与小程序的差异（同样先做）

首页「附近球房」空态现在固定显示「添加附近球馆」。小程序 `uniapp/utils/home-index.js` 的 `resolveHomeVenueEmptyAction` 会在常玩球馆奖励可领取时（`isFavoriteVenueRewardClaimable`，原生已有 `FavoriteVenueReward.isClaimable`）显示「添加球馆领取VIP」和对应说明。改法：`HomeModel` 在已登录时读取常玩球馆奖励状态（复用 `ProfileAPI.overview()` 的 `availableFavoriteVenueReward`，与首页其他区块一样独立读取、失败不影响别的区块，失败或未登录时按不可领取处理），空态文案按该规则选择；把 `resolveHomeVenueEmptyAction` 迁成纯函数并搬运 `home-index.test.mjs` 里对应的用例。

## 还有一处消息跳转要改（同样先做）

`iOS/iOS/Features/Messages/MessageCenterRules.swift` 把通知目标 `/subPages/season/report` 映射成了 `.seasonIndex`（当时赛季报告页还没有路由）。现在已有 `case seasonReport(seasonId: Int64, gameType: Int)`：小程序 `uniapp/subPages/season/report.vue` 的 `onLoad` 用查询参数 `id` 作为赛季 ID（球种参数名以同一个 `onLoad` 为准，没有就按 0），从 target 里取出，映射到 `.seasonReport(...)`；赛季 ID 取不到或不大于 0 时退回 `.seasonIndex`。补对应的映射测试。

## 本轮要读的小程序源码

- `uniapp/subPages/social/friendList.vue`、`friendRequests.vue`、`addFriend.vue`、`friendHomepage.vue`、`blacklist.vue`（含各自样式）
- `uniapp/utils/friend-entry.js`、`friend-swipe.js`、`friend-homepage.js` 及测试 `friend-entry.test.mjs`、`friend-homepage.test.mjs`（以及 `uniapp/tests/` 里其他以 `friend` 开头的测试）
- `uniapp/api/friend.js`、`uniapp/store/friendRequest.js`；好友主页用到的 `api/match.js` 接口、`opponent-record.js`（已迁到 `Features/Opponents/`）、`gear-display.js`、`honor-wall.js`（如已迁到 `Features/Gear/`、`Features/Achievement/` 则复用；未迁则只迁本页用到的函数到 `Features/Friends/` 并在交付说明里列出，后面去重）
- 契约：`backend/chasing_points.api` 的 `/api/friend/*` 全部接口及类型；被拉黑 / 隐藏战绩 / 非好友等情况下各接口的实际返回读 `backend/internal/logic/friend/` 确认。
- 合规：页面里受 `canShowSocialPublishing()` 等开关控制的入口同步隐藏（`AppFlags`）。

## 任务

`iOS/iOS/Features/Friends/`：

1. **好友列表**：`.friendList` → `FriendListView`，标题「好友列表」。搜索 / 分组 / 在线或状态标记、左滑操作（删除好友、拉黑；`friend-swipe.js` 的规则用系统 `swipeActions` 表达，确认弹窗文案照搬）、顶部入口（好友申请带计数、添加好友）、空态、下拉刷新、分页，以 `friendList.vue` 为准。头像用 `MemberAvatarView`。点好友 → `.friendHomepage(userId:)`。
2. **好友申请**：`.friendRequests` → `FriendRequestsView`，标题「好友申请」。收到 / 发出的申请、同意 / 拒绝、处理后的本地更新，以 `friendRequests.vue` 为准。处理成功后同步 `ActivityStore.setPendingFriendRequestCount` / `markDirty()`。
3. **添加好友**：`.addFriend` → `AddFriendView`，标题「添加好友」。搜索用户（`GET /api/friend/search`）、发送申请及各种结果提示（已是好友、已申请、对方拉黑等）以 `addFriend.vue` 为准。小程序里「扫码加好友 / 分享邀请」如依赖微信能力则不显示；用系统能力能等价实现的（如通过匹配二维码）按页面实际逻辑判断并在交付说明里写明取舍。
4. **好友主页**：`.friendHomepage(userId:)` → `FriendHomepageView`，标题「好友主页」。资料头（`MemberAvatarView`）、段位 / 战绩概览、对战记录入口、装备与荣誉入口、发起约球 / 删除 / 拉黑等操作，以 `friendHomepage.vue` 和 `friend-homepage.js` 为准。
   - 「查看 TA 的过往对手 / 与我的交锋」：给 `.opponentRecord` 增加可选的目标用户参数（`case opponentRecord(target: OpponentRecordTarget?)`，`OpponentRecordRules` 里已经迁了 target 模式的规则），并同步所有引用处（「我的」页传 `nil`）；交锋记录走 `.h2hRecord(...)`。
   - TA 的装备 → `.gearFriend(userId:)`（如路由表里还没有，新增占位 case，标题「TA 的装备」）；TA 的荣誉墙入口按小程序的跳转参数处理。
5. **黑名单**：`.blacklist` → `BlacklistView`，标题「黑名单」。列表、移出黑名单（确认文案照搬）、空态，以 `blacklist.vue` 为准。

所有写操作：进行中禁用、失败提示照搬、成功后更新本页并按需 `ActivityStore.markDirty()`。

## 测试

搬运 JS 用例；接口门面；各页面模型（读取、分页、搜索防抖 / 只采纳最新、写操作成功与失败、身份变化）；快照：`friend-list`、`friend-list-empty`、`friend-requests`、`add-friend-results`、`friend-homepage`、`friend-homepage-hidden`（对方隐藏战绩）、`blacklist`、`blacklist-empty`。

## 并行开发提醒

其他开发线正在修改 `iOS/iOS/Features/Profile/**`、`Features/Achievement/**`、`Features/Season/**`、`Features/Gear/**`、`Features/Member/**`、`Features/Stats/**`、`Features/Match/**`、`iOS/iOS/SessionStore.swift` 及其测试。本轮除了给 `.opponentRecord` 加参数时必须同步的那一行引用（`Features/Profile/MyPageSections.swift` 里的路由值）外，不要碰这些文件。
