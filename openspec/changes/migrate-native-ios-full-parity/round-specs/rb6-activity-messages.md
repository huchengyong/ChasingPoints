# RB6：应用级活动状态、用户实时通道接线、消息 Tab 与角标

小程序有一份全应用共享的「活动状态」（`uniapp/store/activity.js`）：当前对局、当前约球、收到的约球数、未读通知数、待处理好友申请数、最近一条换季通知。首页、消息 Tab 角标、「我的」页都读它，用户实时通道（WebSocket）负责把它标脏。本轮在原生端建立这层状态，把用户实时通道接到应用生命周期，并迁移消息 Tab。

## 本轮要读的小程序源码

- `uniapp/store/activity.js`（重点 `applyBootstrap`、`markDirty` / `dirtySeq`、`refreshIfDirty`）、`store/notification.js`、`store/friendRequest.js`
- `uniapp/App.vue`：`userWS` 的连接 / 断开时机、前后台与网络状态处理、三个消息处理函数（`handleRankInfoUpdated`、`handleUserDataUpdated`、`handleNotificationUpdate`）、会话失效处理
- `uniapp/pages/message/index.vue` + 同目录 scss、`uniapp/utils/message-center.js`、`use-message-counts.js`、`main-tab-bar.js` 的 `resolveMessageBadge`；测试 `message-center.test.mjs`、`main-tab-bar.test.mjs`（角标规则部分）
- `uniapp/api/notification.js`；契约：`GET /api/user/bootstrap`（`GetUserBootstrapResp`、`CurrentMatchInfo`、`ChallengeInfo`、`NotificationInfo`）、`/api/notification/*`（`GetNotificationListReq/Resp`、`NotificationIdReq`、`GetUnreadCountResp`）。`availability` 各分区的键名与失败时的返回读 `backend/internal/logic/user/` 里 bootstrap 的实现确认。
- 原生已有：`iOS/iOS/Shared/Realtime/`（`UserRealtime`、`RealtimeMessage` 的消息类型常量；上一轮的交付说明里列了接入时要调用的方法：`connect(authGeneration:)`、`setForeground`、`setNetworkOnline`、`disconnect`、`sessionInvalidHandler`、`on` / `off`）。

## 任务

### 1. 活动状态（新建 `iOS/iOS/App/ActivityStore.swift`）

```swift
/// 全应用共享的活动状态，按身份隔离。读取来源是聚合接口 GET /api/user/bootstrap。
@MainActor
final class ActivityStore: ObservableObject {
    static let freshness: TimeInterval = 30      // 对应 ACTIVITY_CACHE_TTL

    @Published private(set) var currentMatch: CurrentMatchInfo?
    @Published private(set) var currentChallenge: ChallengeInfo?
    @Published private(set) var receivedChallengeCount = 0
    @Published private(set) var challengeServerTime = ""
    /// 约球分区读不到时为 false：页面不得把未知当成「没有约球」。
    @Published private(set) var challengeAvailable = true
    @Published private(set) var unreadCount = 0
    @Published private(set) var pendingFriendRequestCount = 0
    @Published private(set) var latestSeasonRollover: NotificationInfo?
    /// 用户数据版本：实时通道通知「战绩 / 段位等已变化」时 +1，页面据此决定是否重读。
    @Published private(set) var userDataRevision = 0
    @Published private(set) var isLoaded = false

    init(session: SessionStore, now: @escaping () -> Date = Date.init)

    /// 身份变化时清空；没有身份时保持清空状态。
    func syncIdentity()
    /// 未读过、已标脏或超过 30 秒时读取；同一时刻只有一个在途读取（并发调用共用它）。
    func refreshIfNeeded() async
    /// 强制读取。
    func refresh() async
    /// 写操作成功或实时通道通知后调用：标脏并递增序号；读取期间发生的标脏不会被该次响应清掉。
    func markDirty()
    /// 标脏后的补读：有在途读取先等它结束，仍然是脏的再读一次。
    func refreshIfDirty() async
    func setUnreadCount(_ count: Int)
    func decrementUnread()
    func setPendingFriendRequestCount(_ count: Int)
    func clearLatestSeasonRollover()
    func bumpUserDataRevision()

    /// 消息 Tab 角标。
    var messageBadge: MessageBadge { get }
}

enum MessageBadge: Equatable { case none, dot, text(String) }
```

- 应用规则逐条对应 `applyBootstrap`：某个分区在 `availability` 里为 false 时保留旧值；约球两个分区任一不可用 → `challengeAvailable = false`；有任何分区不可用 → 读取后仍保持「脏」。读取整体失败（网络 / 5xx）→ 状态不变、保持脏。
- `messageBadge` 对应 `resolveMessageBadge`：未登录 none；未读 > 0 显示数字（超过 99 为「99+」）；否则好友申请 + 约球邀请 > 0 显示红点；否则 none。
- `CurrentMatchInfo`、`ChallengeInfo`、`NotificationInfo` 模型字段与 `.api` 对齐（放 `iOS/iOS/App/ActivityModels.swift`；`optional` 字段用可选类型）。
- 计数不得为负。

### 2. 用户实时通道接线（新建 `iOS/iOS/App/AppRealtime.swift`，修改 `iOS/iOS/iOSApp.swift`）

```swift
/// 把用户实时通道接到应用生命周期；只在正式启动路径里创建，测试宿主不创建。
@MainActor
final class AppRealtime: ObservableObject {
    init(session: SessionStore, activity: ActivityStore, user: UserRealtime? = nil)
    /// 身份出现 → 连接；身份消失或变化 → 断开后按新身份重连。
    func handleIdentityChange() async
    func setForeground(_ foreground: Bool)
    func setNetworkOnline(_ online: Bool)
}
```

- 连接 / 断开 / 前后台 / 网络的时机逐条对应 `App.vue`（注意小程序进入后台时除了 `setForeground(false)` 还会 `disconnect()`，回到前台重新连接——照搬）。
- 三类消息的处理对应 `App.vue` 的三个 handler：通知更新 → `activity.markDirty()` 后 `refreshIfDirty()`（防抖间隔照搬 `NOTIFICATION_REFRESH_DEBOUNCE_MS` 如果 handler 用到）；用户数据 / 段位更新 → `activity.bumpUserDataRevision()` + 标脏（小程序在这里做的各 store 失效，原生端用这个版本号统一表达）。
- 实时通道报告会话失效时：调用 `activity.refresh()`——它是一个带凭据的普通请求，会经由 `SessionStore.authorizedRequest` 走标准的 401 / `SESSION_INVALID` 清理流程。不要给 `SessionStore` 新增「强制失效」入口。
- `iOSApp.swift` 的 `ApplicationRootView`：创建 `ActivityStore` 与 `AppRealtime`（`@StateObject`），注入环境；用 `@Environment(\.scenePhase)` 驱动 `setForeground`；用 `NWPathMonitor`（Network 框架）驱动 `setNetworkOnline`；`session.identity` 变化时调用 `handleIdentityChange()` 与 `activity.syncIdentity()` + `refreshIfNeeded()`。测试宿主分支（`CHASING_POINTS_AUTH_TEST_HOST`）保持只显示 `Color.clear`，不创建这些对象。
- `NWPathMonitor` 包在一个小的可注入协议后面，测试不依赖真实网络状态。

### 3. 消息角标（修改 `iOS/iOS/App/MainTabView.swift`，只动消息 Tab 相关的行）

- `MainTabView` 读取 `@EnvironmentObject ActivityStore`，消息 Tab 用 `.badge(...)` 显示：数字 / 「99+」/ 红点（红点用 `.badge("")` 之类系统能显示成小圆点的方式；如果系统不支持纯红点，用空格文本并在交付说明里写明实际效果）。
- 页面出现和回到前台时调用 `activity.refreshIfNeeded()`。
- 现有用到 `MainTabView` 的快照 / 单元测试需要补上 `ActivityStore` 环境对象（用 `MockHTTPServer` 的会话创建）。

### 4. 消息 Tab（新建 `iOS/iOS/Features/Messages/`）

替换 `MainTabView` 里消息 Tab 的占位根页面 → `MessageCenterView`（导航标题「消息」）。

- `message-center.js` 的规则逐条迁移并搬运测试（`buildMessageEntries` 里的小程序路径换成路由：好友申请 → `.friendRequests`；约球邀请 → `.challenges(...)` 的「收到」视图——如果现有 `challenges` 路由的参数表达不了 `tab=received`，把关联值改成能表达三种 Tab 的枚举，并同步其他引用处）。
- 顶部两个入口（好友申请、约球邀请，带计数与说明）——计数来自 `ActivityStore`。
- 通知列表：分页（`PagedHistoryList`）、未读样式、下拉刷新、空态、失败态、未登录引导，以 `index.vue` 为准。
- 操作：点通知（标记已读 `POST /api/notification/read` + 按通知类型跳转）、全部已读（`POST /api/notification/read-all`）、删除（`POST /api/notification/delete`，原生端用左滑删除）。成功后同步 `ActivityStore` 的未读数（已读一条 → `decrementUnread()`；全部已读 → `setUnreadCount(0)`；列表接口返回的 `unread_count` → `setUnreadCount`）。
- 通知点击后的跳转规则以 `index.vue` 里按 `type` / `data` 分流的代码为准，逐个类型映射到 `AppRoute`（目标页面未迁移的走已有占位路由；`data` 的 JSON 解析失败时不跳转）。把「通知 → 路由」做成纯函数并逐类型测试。
- 何时重新加载列表：对应 `resolveNotificationReload`——身份变化或 `ActivityStore` 标脏后才回源，普通切回 Tab 不重复请求。

## 测试

- `ActivityStoreTests.swift`：bootstrap 全部可用 / 部分不可用（保留旧值、保持脏、`challengeAvailable`）/ 整体失败；30 秒新鲜度（注入时钟）；并发 `refreshIfNeeded` 只发一次请求；读取期间 `markDirty` 后响应到达仍为脏且 `refreshIfDirty` 会再读一次；身份变化清空且迟到响应丢弃；计数操作不为负；`messageBadge` 的全部分支（搬 `main-tab-bar.test.mjs` 的角标用例）。
- `AppRealtimeTests.swift`（用上一轮的替身传输层 `FakeRealtimeTransport` 等）：登录后连接、退出后断开、切号重连；进入后台断开、回到前台重连；离线不连、恢复后连；三类消息各自的效果；会话失效触发一次 `activity.refresh()`。
- `MessageCenterTests.swift`：规则搬运；通知 → 路由的逐类型映射；列表模型（分页、已读 / 全部已读 / 删除后的本地更新与未读数同步、失败回滚、身份变化）。
- 快照：`messages`（两个入口有计数、通知 5 条含已读未读）、`messages-empty`、`messages-guest`、`main-tabs-badge-count`（消息 Tab 显示数字角标）、`main-tabs-badge-dot`。

## 并行开发提醒

其他开发线正在修改 `iOS/iOS/Features/Profile/**`、`Features/Reputation/**`、`Features/Opponents/**`、`Features/Stats/**`、`Features/Achievement/**`、`Features/Season/**`、`Features/Gear/**`、`Features/Member/**`、`Features/Match/**`、`iOS/iOS/SettingsView.swift`、`iOS/iOS/SessionStore.swift` 及其测试，本轮不要碰这些文件。`MainTabView.swift` 只改消息 Tab 与角标相关的行；`iOSApp.swift` 只改 `ApplicationRootView`。
