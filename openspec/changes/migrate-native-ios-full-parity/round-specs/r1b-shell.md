# R1b：原生 iOS 应用骨架（路由 / 五 Tab / 共享 UI 组件 / 快照测试支架）

你在一个 git worktree 的根目录里工作（当前目录就是仓库根）。总目标是把微信小程序（`uniapp/`）几乎 1:1 迁移到原生 SwiftUI（`iOS/`）。上一轮（R1a）已经加好：`AppPalette` 的完整语义色、`AppFlags`、`SessionStore.authorizedPost` / `optionalAuthRequest`、`ResourceLoader`、`APIQuery`、`CDNAsset`、`DefaultAvatar` / `MemberIdentity` / `MemberAvatarMetrics`。本轮搭应用骨架：统一路由、五个主 Tab、共享 UI 组件、可以把界面渲染成 PNG 的测试支架。**不新增业务页面**，没迁移的页面一律显示占位。

先读：`AGENTS.md`、`iOS/AGENTS.md`、`DESIGN.md`（第 3、5–9 节）、`iOS/iOS/` 下全部 Swift 文件（含 `Shared/`）、`iOS/iOSAuthTests/TestSupport.swift` 与 `MockHTTPServer.swift`、`uniapp/pages.json`、`uniapp/utils/main-tab-bar.js`、`uniapp/components/memberAvatar.vue`。

## 硬性约束（违反任何一条都算失败）

1. 只允许新增/修改 `iOS/iOS/**` 与 `iOS/iOSAuthTests/**`。不得修改 `iOS/iOS.xcodeproj/**`（文件系统同步组，新文件自动进 target）、`uniapp/**`、`backend/**` 及其他目录。从 `uniapp/static/` 复制图片到 asset catalog 是允许的（只复制，不改原图）。
2. 不执行任何 git 写操作（commit / add / checkout / reset / stash 等），只允许 `git status`、`git diff` 等只读命令。
3. 不联网、不 ssh、不 curl；不读取任何 `.env`、密钥、Keychain、真实会话；不访问当前目录之外的路径（Xcode 默认 DerivedData 除外）。
4. 不引入第三方依赖。只用 SwiftUI / UIKit / Foundation 等系统框架。最低部署目标 iOS 17.0。
5. 现有全部测试必须继续通过。除本规格点名的文件外，不重命名、不移动、不重构现有代码；`LoginView`、`SettingsView`、`NicknameEditView`、`MatchHistoryView`、`MatchReportView` 的界面和行为保持不变。
6. 注释与文案用简体中文，风格与现有文件一致。颜色一律走 `AppTheme` / `AppPalette`，不在页面里写死色值。
7. 不创建任何 Markdown / 文档文件。

## 工程须知

- App target 默认 MainActor 隔离、Swift 5 语言模式；测试类标 `@MainActor`。
- 测试宿主：共享 scheme 给测试注入了 `CHASING_POINTS_AUTH_TEST_HOST=1`，`iOSApp` 此时只显示 `Color.clear`、不创建 `SessionStore`。这段逻辑不能动。
- bash 工具跑 xcodebuild 时超时设为 600000 毫秒，输出必须用 grep 过滤。

## 任务

### 0. 修正上一轮的一个缺陷（`iOS/iOS/Shared/ResourceLoader.swift`）

现状：`fetch` 一开始就把 `isStale` 清掉。如果 `invalidate()` 之后的那次刷新失败了，过期标记已经丢失，页面下次出现时 `loadIfNeeded()` 不会再重读，旧数据会一直留着。

改成：过期标记只在**刷新成功**后清除；刷新失败或被取消时保留，下次 `loadIfNeeded()` 继续重读。首次加载的行为不变。在 `ResourceLoaderTests.swift` 新增一个用例：`invalidate()` → `loadIfNeeded()` 失败（数据不变、`refreshError` 有值）→ 再次 `loadIfNeeded()` 会重新请求并成功更新。只改这一处，不动其他行为。

### 1. 路由（新建 `iOS/iOS/App/AppRoute.swift`）

```swift
/// 应用内页面路由，与小程序 pages.json 的页面一一对应；未迁移的页面先落到占位页。
enum AppRoute: Hashable {
    case login
    case settings
    case editNickname
    case matchHistory
    case matchReport(matchId: Int64)
    case ranking
    case spectatorLobby
    case rankExplain
    case notificationSettings
    case opponentRecord
    case h2hRecord(opponentId: Int64)
    case memberCenter
    case reputation
    case statsDetail
    case performance
    case matchReplay(matchId: Int64)
    case challengeCompose
    case challenges
    case feedback
    case achievementWall
    case achievementDetail(achievementId: Int64)
    case friendList
    case friendHomepage(userId: Int64)
    case addFriend
    case friendRequests
    case blacklist
    case tournamentDetail(id: Int64)
    case venueList
    case venueDetail(id: Int64)
    case venueSubmit
    case rulesIndex
    case glossary
    case seasonIndex
    case seasonHistory
    case gearIndex

    /// 导航栏标题，取自 pages.json 的 navigationBarTitleText。
    var title: String { get }
}

/// 路由 → 页面。已迁移的返回真实页面，其余返回占位页。
struct AppRouteDestination: View {
    let route: AppRoute
}
```

- 标题对照 `uniapp/pages.json`：login「登录」、settings「设置」、editNickname「编辑资料」、matchHistory「对局记录」、matchReport「对局详情」、ranking「排行榜」、spectatorLobby「观赛大厅」、rankExplain「段位说明」、notificationSettings「通知管理」、opponentRecord「过往对手」、h2hRecord「交锋记录」、memberCenter「VIP中心」、reputation「信誉明细」、statsDetail「竞技分析」、performance「表现分析」、matchReplay「单场复盘」、challengeCompose「发起约球」、challenges「约球」、feedback「帮助、投诉与举报」、achievementWall「荣誉墙」、achievementDetail「成就详情」、friendList「好友列表」、friendHomepage「好友主页」、addFriend「添加好友」、friendRequests「好友申请」、blacklist「黑名单」、tournamentDetail「赛事详情」、venueList「附近球馆」、venueDetail「球馆详情」、venueSubmit「上传球馆」、rulesIndex「规则百科」、glossary「术语词典」、seasonIndex「赛季」、seasonHistory「历届赛季」、gearIndex「我的装备」。
- `AppRouteDestination`：`.login` → `LoginView()`；`.settings` → `SettingsView()`；`.editNickname` → `NicknameEditView()`；`.matchHistory` → `MatchHistoryView(session:)`；`.matchReport` → `MatchReportView(session:matchId:)`；其余全部 → `MigrationPlaceholderView(title: route.title)`（它自己设置 `navigationTitle`）。需要 `session` 的用 `@EnvironmentObject`。
- `MatchHistoryView` 内部现有的 `MatchReportRoute` 与 `.navigationDestination(for: MatchReportRoute.self)` 保持不动。

### 2. 导航状态（新建 `iOS/iOS/App/AppNavigator.swift`）

```swift
enum MainTab: Int, CaseIterable, Hashable {
    case home, news, scan, messages, me
    var title: String { get }   // 首页 / 赛讯 / 扫码 / 消息 / 我的
}

/// 主 Tab 选中态与各 Tab 的导航栈。扫码不是页面：点它不切换 Tab，只触发扫码流程。
@MainActor
final class AppNavigator: ObservableObject {
    @Published private(set) var selectedTab: MainTab = .home
    @Published var isScanPresented = false

    func path(for tab: MainTab) -> NavigationPath
    func setPath(_ path: NavigationPath, for tab: MainTab)

    /// 点击 Tab：`.scan` 走 requestScan，其余切换选中态（已选中的 Tab 再点一次回到该 Tab 根页面）。
    func select(_ tab: MainTab, isSignedIn: Bool)
    /// 在当前选中的 Tab 里打开页面。
    func open(_ route: AppRoute)
    /// 切到指定 Tab 并回到根页面后打开页面（用于跨 Tab 跳转）。
    func open(_ route: AppRoute, in tab: MainTab)
    /// 切到「我的」、回到根页面并打开登录页。
    func openLogin()
    /// 扫码入口：未登录 → openLogin()；已登录 → isScanPresented = true。
    func requestScan(isSignedIn: Bool)
    func popToRoot(_ tab: MainTab)
    /// 退出登录、会话失效或切换账号后调用：清空所有 Tab 的导航栈并关闭扫码。
    func reset()
}
```

- 各 Tab 的导航栈用 `NavigationPath` 保存（因为栈里还会出现 `MatchReportRoute` 这类非 `AppRoute` 的值），内部用 `@Published` 字典或四个属性都行。
- `selectedTab` 永远不会等于 `.scan`；`open(_:in:)` / `path(for:)` 传入 `.scan` 时按 no-op / 空路径处理。

### 3. 主 Tab（新建 `iOS/iOS/App/MainTabView.swift`，改写 `iOS/iOS/ContentView.swift`）

- `ContentView` 只保留：读取 `@AppStorage("theme_mode")` 并应用 `preferredColorScheme`，内容换成 `MainTabView()`。`MainTabs`、`FeaturePlaceholder` 删除；`MyPage`、`ProfileCard`、`ProfileAvatar` 搬到新文件 `iOS/iOS/Features/Profile/MyPage.swift`（见任务 6）。
- `MainTabView`：
  - `TabView(selection:)` 用自定义 `Binding`：get 返回 `navigator.selectedTab`，set 调 `navigator.select(newValue, isSignedIn: session.identity != nil)`，这样点「扫码」不会切走当前 Tab。
  - 五个 Tab 依次为 首页 / 赛讯 / 扫码 / 消息 / 我的，对应小程序主导航。首页、赛讯、消息的根页面先用 `MigrationPlaceholderView`（导航标题分别是「追分」「赛讯」「消息」）；「我的」是 `MyPage()`；扫码 Tab 的内容是 `Color.clear`。
  - 首页、赛讯、消息、我的各自一个 `NavigationStack(path:)`，path 绑定到 `navigator.path(for:)` / `setPath`，并在根视图上挂 `.navigationDestination(for: AppRoute.self) { AppRouteDestination(route: $0) }`。
  - Tab 图标：首页 `TabHome` / `TabHomeSelected`，赛讯 `TabNews` / `TabNewsSelected`，我的 `TabMe` / `TabMeSelected`（现有资源，`.renderingMode(.original)` 写法不变）；消息用新资源 `TabMessage` / `TabMessageSelected`；扫码用新资源 `TabScan`（亮暗共用）。每个 Tab 都带文字标题。
  - `.tint(AppTheme.primary)`、`toolbarBackground` 沿用现有写法。
  - 扫码占位：`.sheet(isPresented: $navigator.isScanPresented)` 里显示 `MigrationPlaceholderView(title: "扫码")` 外包一个带「关闭」按钮的 `NavigationStack`（真正的扫码后面迁移）。
  - 「登录状态已失效」的 alert 从 `MyPage` 挪到这里（全局只弹一次）：「重新登录」→ `session.acknowledgeSessionExpired()` + `navigator.openLogin()`；「暂不登录」→ `acknowledgeSessionExpired()`。文案与现有一致。
  - `.onChange(of: session.identity)`：旧身份非空，且新身份为空或 `userId` 不同 → `navigator.reset()`。从未登录变为已登录时不清栈。
  - 根上挂 Toast（任务 5）。
- `iOS/iOS/iOSApp.swift` 的 `ApplicationRootView`：新增 `@StateObject` 的 `AppNavigator` 与 `ToastCenter`，用 `.environmentObject` 注入。测试宿主分支（`Color.clear`）不动。
- 资源（`iOS/iOS/Assets.xcassets/`，`Contents.json` 格式照抄 `TabHome.imageset`）：
  - `TabMessage.imageset`：`uniapp/static/tabbar/message_light.png` + `message_dark.png`（3x，暗色用 appearances）。
  - `TabMessageSelected.imageset`：`message-selected_light.png` + `message-selected_dark.png`。
  - `TabScan.imageset`：把 `uniapp/static/tabbar/scan-button.png` 用 `sips -Z 90 <源> --out <目标>` 缩成 90px 后放入，标 3x（即 30pt），不分亮暗。
  - 删除不再使用的 `TabWatch.imageset`、`TabWatchSelected.imageset`。

### 4. 共享 UI 组件（新建到 `iOS/iOS/Shared/UI/`，每个组件一个文件）

对外接口如下（内部实现自定，颜色全部取 `AppTheme.palette(for: colorScheme)`）：

```swift
/// 空态 / 错误 / 登录引导：图标 → 标题 → 说明 → 可选操作。
struct StateMessageView: View {
    init(icon: String, title: String, detail: String? = nil, actionTitle: String? = nil, action: (() -> Void)? = nil)
}

/// 把 ResourceLoader 的各状态画出来：加载中、失败（带重试）、未登录引导、已加载内容 + 刷新失败提示。
struct LoadableContent<Value, Content: View>: View {
    init(
        loader: ResourceLoader<Value>,
        requiresLogin: Bool = true,
        loginPrompt: String = "登录后查看",
        @ViewBuilder content: @escaping (Value) -> Content
    )
}

/// 未迁移页面的占位。
struct MigrationPlaceholderView: View {
    init(title: String)
}

/// 区块标题行：标题 + 可选小字说明 + 可选右侧「更多」。
struct SectionHeader: View {
    init(title: String, caption: String? = nil, moreTitle: String? = nil, moreAction: (() -> Void)? = nil)
}

/// 远程图片：地址为空或加载失败时保持占位内容。
struct RemoteImage<Placeholder: View>: View {
    init(url: URL?, contentMode: ContentMode = .fill, @ViewBuilder placeholder: @escaping () -> Placeholder)
}

/// 普通头像：有 http(s) 头像地址用远程图，否则用按 userId 取的默认球头像。
struct AvatarView: View {
    init(avatar: String, userId: Int64, size: CGFloat)
}

/// 会员身份头像：头像 + 段位头像框 + VIP 标识，对应小程序 components/memberAvatar.vue。
struct MemberAvatarView: View {
    init(
        avatar: String, userId: Int64, vipLevel: Int = 0, rankLevel: Int = 0, size: CGFloat = 48,
        reserveFrame: Bool = false, fitFrame: Bool = false, showBadge: Bool = true
    )
}

extension View {
    /// 普通卡片：surface-card 背景 + 1px border-default 描边 + 16 圆角。
    func appCard(padding: CGFloat = 16) -> some View
}
```

按钮样式（`AppButtonStyles.swift`），圆角规则见 `DESIGN.md` 9.1：

- `PrimaryButtonStyle`：块级主按钮，撑满宽度、高 48、圆角 12、`AppTheme.primary` 背景、白字 16 semibold；禁用 opacity 0.5，按下 opacity 0.85。
- `SecondaryButtonStyle`：块级次按钮，`subtle` 背景 + `border` 描边 + `brandStrong` 文字，尺寸同上。
- `PillButtonStyle`：紧凑胶囊按钮，高 36、水平 padding 20、`AppTheme.primary` 背景、白字 14 semibold。
- `DangerButtonStyle`：块级破坏性按钮，`AppTheme.danger` 文字 + `card` 背景 + `border` 描边。
- 提供 `.buttonStyle(.appPrimary)` / `.appSecondary` / `.appPill` / `.appDanger` 这种静态成员写法。禁用态要读 `@Environment(\.isEnabled)`（`ButtonStyle` 里读不到，需要在内部包一层 `View`）。

细节要求：

- `StateMessageView`：整体居中；图标用 SF Symbol（约 40pt，`textMuted`）；标题 17 semibold `textPrimary`；说明 14 `textSecondary` 多行居中；操作按钮用 `.appPill`。无障碍上合并为一个元素朗读标题和说明，按钮单独可聚焦。
- `LoadableContent`：自己读 `@EnvironmentObject` 的 `SessionStore` 与 `AppNavigator`。`requiresLogin && session.identity == nil` → `StateMessageView(icon: "lock.fill", title: loginPrompt, actionTitle: "去登录") { navigator.openLogin() }`。`.idle` / `.loading` → 居中的 `ProgressView()`。`.failed(msg)` → `StateMessageView(icon: "exclamationmark.triangle", title: "加载失败", detail: msg, actionTitle: "重试")`，点了调 `loader.retry()`。`.loaded(v)` → `content(v)`，若 `loader.refreshError != nil` 在内容上方显示一行提示（`textSecondary` 14 号字 + 「重试」）。用 `.task { await loader.loadIfNeeded() }` 触发读取，并在 `.onChange(of: session.identity)` 时调用 `loader.syncIdentity()` 后再 `loadIfNeeded()`。它不包 `ScrollView`、不管下拉刷新，这些由调用方决定。
- `MigrationPlaceholderView`：沿用原 `FeaturePlaceholder` 的样子（图标 + 「{title}页面正在迁移」+ 「此功能尚未在原生版开放」），并设置 `.navigationTitle(title)`、inline 标题。
- `AvatarView`：圆形裁剪；底层始终是 `Image(DefaultAvatar.assetName(userId:))`，头像地址以 `http://` 或 `https://` 开头时在上面叠 `RemoteImage`；`accessibilityHidden(true)`。
- `MemberAvatarView`：尺寸全部来自 `MemberAvatarMetrics.resolve`，身份来自 `MemberIdentity.resolve`。布局盒 `boxSize`，头像 `avatarSize` 居中，有 `frameURL` 时叠 `frameSize` 的头像框（`RemoteImage`，`.fit`，允许溢出布局盒）；`showBadge && identity.showBadge` 时在头像底部居中叠 VIP 标识（胶囊，文字 `identity.badgeText`，字号 / 高度 / 左右内边距取 metrics）。标识三档的配色参照 `uniapp/components/memberAvatar.vue` 里 `.member-avatar-badge--standard/advanced/premium` 的样式，这三组色值作为「会员档位色」定义成 `MemberAvatarView` 文件内的私有常量（这是本规格唯一允许写死色值的地方）。非会员时外观与 `AvatarView` 完全一致。无障碍：会员时 `accessibilityLabel("VIP\(level) 会员")`，否则隐藏。

### 5. Toast（新建 `iOS/iOS/Shared/UI/Toast.swift`）

```swift
/// 轻提示，对应小程序的 uni.showToast({ icon: 'none' })。
@MainActor
final class ToastCenter: ObservableObject {
    @Published private(set) var message: String?
    init(duration: TimeInterval = 2)
    /// 显示一条提示；新提示替换旧提示并重新计时；空白字符串忽略。
    func show(_ message: String)
    func dismiss()
}

extension View {
    /// 在根视图上叠加 Toast 层。
    func toastOverlay(_ center: ToastCenter) -> some View
}
```

- 样式：底部居中（避开 Tab 栏，距底部约 96pt）、深色半透明胶囊、白字 14、最多两行、淡入淡出 0.2s；不拦截点击（`allowsHitTesting(false)`）。
- 显示时用 `AccessibilityNotification.Announcement(message).post()` 朗读。
- 计时用可取消的 `Task`；被新提示替换时旧计时不得把新提示提前清掉（要有测试）。

### 6. 「我的」页搬家（新建 `iOS/iOS/Features/Profile/MyPage.swift`）

- 把 `ContentView.swift` 里的 `MyPage`、`ProfileCard`、`ProfileAvatar` 原样搬过来（`MyPage` 改为 internal，另外两个保持 private），界面不变，只做三处调整：
  1. 三个 `NavigationLink { ... }` 改成 `NavigationLink(value: AppRoute.editNickname / .matchHistory / .settings)`；「去登录」改成 `NavigationLink(value: AppRoute.login)`。
  2. 删除 `showLogin` 状态、`.navigationDestination(isPresented:)` 和会话失效 alert（已挪到 `MainTabView`）。
  3. 其余不动。下一轮会按小程序的「我的」页整体重做，这轮不要提前做。

### 7. 快照测试支架（新建 `iOS/iOSAuthTests/SnapshotSupport.swift`）

目的：让人能看到界面长什么样。把任意 SwiftUI 视图渲染成 PNG 写到宿主机目录。

```swift
/// 把视图渲染成 PNG。设置了环境变量 SNAPSHOT_DIR 时写入 `<SNAPSHOT_DIR>/<name>-<light|dark>.png`，否则只渲染不落盘。
/// 返回渲染出的图片，调用方至少断言尺寸正确。
@MainActor
@discardableResult
func renderSnapshot<Content: View>(
    _ name: String,
    colorScheme: ColorScheme,
    size: CGSize = CGSize(width: 393, height: 852),
    @ViewBuilder content: () -> Content
) throws -> UIImage
```

- 实现方式：取当前活跃的 `UIWindowScene`（测试宿主 App 有一个），新建 `UIWindow(windowScene:)`，`frame` 设为 `size`，`rootViewController = UIHostingController(rootView: content().environment(\.colorScheme, colorScheme))`，同时设置 `window.overrideUserInterfaceStyle`；`isHidden = false`、`layoutIfNeeded()`，再让主 RunLoop 转 0.3 秒左右（等 SwiftUI 完成布局与异步状态），用 `UIGraphicsImageRenderer(size:)` + `window.drawHierarchy(in:afterScreenUpdates: true)` 出图；最后 `window.isHidden = true` 并释放。不要用 `ImageRenderer`（它画不出 `ScrollView`、`NavigationStack`）。
- 取不到 `UIWindowScene` 时用 `XCTSkip` 跳过而不是失败。
- 写文件用 `pngData()`，目录不存在时创建；`name` 里只允许字母、数字、`-`、`_`。

## 必须新增的测试（`iOS/iOSAuthTests/`）

- `AppNavigatorTests.swift`：
  1. `select(.scan, isSignedIn: true)` 不改变 `selectedTab`，`isScanPresented == true`；
  2. `select(.scan, isSignedIn: false)` → 选中 `.me`、栈顶是登录页（`path(for: .me).count == 1`）、`isScanPresented == false`；
  3. `open(_:)` 压入当前 Tab 的栈、不影响其他 Tab；
  4. `open(_:in:)` 切 Tab、清掉该 Tab 原有栈再压入；
  5. 再点已选中的 Tab 回到根页面；
  6. `reset()` 清空所有栈并关闭扫码；
  7. `openLogin()` 多次调用栈里只有一个登录页。
- `AppRouteTests.swift`：抽查 8 个以上路由的 `title` 与 `pages.json` 一致；带参数的路由参数不同则不相等。
- `ToastCenterTests.swift`（用很短的 `duration`，如 0.05）：显示后到时自动消失；空白字符串忽略；后一条替换前一条，且前一条的计时到点不会清掉后一条；`dismiss()` 立即清空。
- `SharedUISnapshotTests.swift`：用 `renderSnapshot` 渲染下面这些，亮色、暗色各一张，断言返回图片的 `size` 等于传入尺寸：
  1. `ui-kit`：一个 `ScrollView`，纵向依次摆出四种按钮样式（含一个禁用的主按钮）、`SectionHeader`（带 caption 和「更多」）、一张 `.appCard()` 卡片（里面放两行文字）、`AvatarView`（userId 3 和 12，size 48）、`MemberAvatarView`（vipLevel 0 / 2 / 4 / 5 配 rankLevel 3，size 56，`reserveFrame: true`）、`StateMessageView`（带操作按钮）。背景用 `palette.page`。
  2. `main-tabs-guest`：完整的 `MainTabView`，注入一个没有凭据的 `SessionStore`（用 `MockHTTPServer` + `InMemoryCredentialStore()`）、`AppNavigator`、`ToastCenter`。
  3. `my-page-signed-in`：`MainTabView`，`SessionStore` 用 `AuthFixtures` 的凭据并 `restore()` 成功后，`navigator.select(.me, isSignedIn: true)`。
  4. `toast`：任意背景上叠 `toastOverlay`，`center.show("已复制到剪贴板")` 后渲染。
- 不得删除、跳过或弱化现有测试。`ContentView` 的改动如果让现有测试编译不过，修测试引用即可，不改测试意图。

## 自检（在 `iOS/` 目录执行，超时 600000 毫秒，输出必须过滤）

```bash
cd iOS
xcodebuild build -project iOS.xcodeproj -scheme iOS -configuration Debug \
  -destination 'generic/platform=iOS Simulator' CODE_SIGNING_ALLOWED=NO 2>&1 | grep -E "error:|warning:|BUILD (SUCCEEDED|FAILED)" | head -60

xcodebuild test -project iOS.xcodeproj -scheme iOS -configuration Debug \
  -destination "platform=iOS Simulator,id=1A59DDBE-8E27-488F-864C-E7D44C52B490" \
  -parallel-testing-enabled NO 2>&1 | grep -E "error:|: error|failed|Executed [0-9]+ tests|TEST (SUCCEEDED|FAILED)" | tail -60

xcodebuild build -project iOS.xcodeproj -scheme iOS -configuration Release \
  -destination 'generic/platform=iOS Simulator' CODE_SIGNING_ALLOWED=NO 2>&1 | grep -E "error:|BUILD (SUCCEEDED|FAILED)" | head -20
```

要求：Debug 与 Release 构建成功；全部测试通过；本次新增代码不产生新的编译警告。测试失败就修到通过，不允许靠删除、跳过或弱化测试来通过。

## 交付

完成后用中文给出：新增/修改/删除的文件清单、测试总数与结果（贴出 `Executed N tests` 那一行）、与本规格不一致的地方及原因。没做到的写明没做到。
