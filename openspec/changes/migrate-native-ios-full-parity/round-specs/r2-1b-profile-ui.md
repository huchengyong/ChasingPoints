# R2.1b：「我的」主页界面（对应小程序 `pages/user/index`）

你在一个 git worktree 的根目录里工作（当前目录就是仓库根）。总目标是把微信小程序（`uniapp/`）几乎 1:1 迁移到原生 SwiftUI（`iOS/`）。上一轮（R2.1a）已经把「我的」页需要的模型、接口和规则迁好：`ProfileAPI`、`UserOverview`、`RankInfoMap`、`UserHomepageRules`、`MemberGrowthCard`、`FavoriteVenueReward`、`GameTypeCatalog`、`GameTypePreference`、`QRCodeImage`、`ResourceLoader.loadIfNeeded(maxAge:)`。本轮用它们把「我的」页的界面做出来。

先读：`AGENTS.md`、`iOS/AGENTS.md`、`DESIGN.md`；`iOS/iOS/` 下全部 Swift 文件（重点 `App/`、`Shared/`、`Shared/UI/`、`Features/Profile/`）；`iOS/iOSAuthTests/` 的 `SnapshotSupport.swift`、`SharedUISnapshotTests.swift`、`ProfileFixtures.swift`、`TestSupport.swift`。
小程序侧（界面的唯一依据）：`uniapp/pages/user/index.vue`（模板 + script）与 `uniapp/pages/user/index.scss`。

## 硬性约束（违反任何一条都算失败）

1. 只允许新增/修改 `iOS/iOS/**` 与 `iOS/iOSAuthTests/**`。不得修改 `iOS/iOS.xcodeproj/**`、`uniapp/**`、`backend/**` 及其他目录。
2. 不执行任何 git 写操作，只允许 `git status`、`git diff` 等只读命令。
3. 不联网、不 ssh、不 curl；不读取任何 `.env`、密钥、Keychain、真实会话；不访问当前目录之外的路径（Xcode 默认 DerivedData 除外）。
4. 不引入第三方依赖。最低部署目标 iOS 17.0。
5. 现有全部测试必须继续通过。除本规格点名的文件外不改现有代码；上一轮的规则函数如果发现有错，在交付说明里指出，不要悄悄改。
6. 颜色一律走 `AppTheme` / `AppPalette`；段位卡上的白色及其透明度变体（卡片是深色渐变表面）可以直接写 `Color.white.opacity(...)`。文案与小程序逐字一致。
7. 不创建 Markdown / 文档文件。

## 工程须知

- App target 默认 MainActor 隔离、Swift 5 语言模式；测试类标 `@MainActor`。
- rpx 换算：小程序 750rpx 对应屏宽，原生端大致按 `pt ≈ rpx / 2` 取整（例：`32rpx` 圆角 → 16，`24rpx` 内边距 → 12，`38rpx` 字号 → 19）。字号用 `.font(.system(size:weight:))`。不要引入 rpx 换算函数。
- 页面之间的跳转只用 `NavigationLink(value: AppRoute.xxx)` 或 `navigator.open(.xxx)`。
- bash 工具跑 xcodebuild 时超时设为 600000 毫秒，输出必须用 grep 过滤。

## 任务

### 0. 修正上一轮的一个小问题（`iOS/iOS/Shared/UI/MemberAvatarView.swift`）

VIP 标识「标准」档的文字色现在写死 `#C69200`。小程序用的是随主题变化的 `--ui-brand-strong`（暗色下是 `#F0C542`）。改成取 `palette.brandStrong`，删掉 `BadgeColor.standardText`。其余档位不动。

**并行开发提醒**：另一条开发线正在修改 `iOS/iOS/Shared/ResourceLoader.swift`、`iOS/iOS/Shared/UI/LoadableContent.swift` 和新增 `iOS/iOS/Features/Rules/**`。本轮不要改这三处；`AppRoute.swift` 只改下面点名的四个 case，不动其他行。

### 1. 路由参数（修改 `iOS/iOS/App/AppRoute.swift` 及其测试）

这几个路由需要带参数，改成：

- `case rankExplain(gameType: Int)`
- `case statsDetail(gameType: Int)`
- `case achievementWall(gameType: Int)`
- `case challenges(showHistory: Bool)`

标题不变，目的地仍是占位页。同步修正引用它们的测试。

### 2. 页面模型（新建 `iOS/iOS/Features/Profile/MyPageModel.swift`）

```swift
/// 「我的」页的读取与展示状态。概览与段位各自独立读取，互不阻塞。
@MainActor
final class MyPageModel: ObservableObject {
    static let freshness: TimeInterval = 60   // 对应小程序 USER_OVERVIEW_CACHE_TTL

    let overview: ResourceLoader<UserOverview>
    let ranks: ResourceLoader<RankInfoMap>

    /// 页内临时切换的球种；0 表示跟随默认。
    @Published var selectedRankGameType = 0
    @Published private(set) var preferredGameType = 0
    @Published private(set) var floatRewardVisible = false

    init(session: SessionStore, defaults: UserDefaults = .standard, now: @escaping () -> Date = Date.init)

    /// 页面出现：同步常用球种偏好；已登录时按新鲜度读取概览与段位。
    func onAppear() async
    /// 下拉刷新：两个资源都强制刷新。
    func refresh() async
    /// 身份变化：清空页内状态并重新同步偏好。
    func onIdentityChanged() async
    /// 在段位卡里选定常用球种（与设置页共用同一偏好）。
    func pickPreferredGameType(_ gameType: Int)
    /// 关闭悬浮奖励入口：按当前状态记 3 天稍后提醒。
    func snoozeFloatReward()

    // —— 以下全部是由上面的状态 + UserHomepageRules 推导出的只读展示值 ——
    var rankGameView: RankGameTypeView { get }
    var currentRankGameType: Int { get }          // selectedRankGameType 在 tabs 里才生效，否则取 primaryGameType
    var showRankGamePicker: Bool { get }          // 段位已读到 && needsPick
    var rankCard: RankCardView { get }
    var isRankSkeleton: Bool { get }              // 段位还没有任何数据且正在读取
    var memberIdentity: MemberIdentity { get }
    var profileLevelText: String { get }
    var reputationChip: ReputationChip { get }
    var careerState: CareerState { get }          // loading / ready / empty / error
    var coreMetrics: [CoreMetric] { get }
    var memberEntry: MemberEntry { get }
    var compactRewardEntry: CompactRewardEntry { get }

    enum CareerState { case loading, ready, empty, error }
}
```

推导规则逐条对应 `pages/user/index.vue` 的 computed：

- `rankGameView`：段位未读到时 `rankInfoMap` 传空字典。
- `careerState`：概览里 `availableStats` 有值 → `totalMatches > 0 ? .ready : .empty`；没有值 → 概览已结束读取（失败，或成功但该分区不可用）→ `.error`，仍在读 → `.loading`。
- `coreMetrics`：`isLoading` = 还没有可用的 stats。
- `reputationChip` / `profileLevelText` 的 `isLoading` = 概览正在首次读取或刷新。
- `memberIdentity`：`rankLoading` = 段位正在读取且还没有数据。
- `floatRewardVisible`：照搬 `syncFloatRewardVisibility`——奖励状态可用且 `enabled`、状态是 `not_started` 或 `rejected`、没有处在稍后提醒期内才显示；过期的稍后提醒记录要删除。在概览每次读取完成后重新计算。存储用注入的 `defaults`，键用 `FavoriteVenueReward.floatSnoozeKey`，值存时间（ISO 8601 字符串或 `Date` 都行，自己读写一致即可）。
- 没有身份时所有推导值回到初始（访客）状态，不残留上一个账号的数据。
- `ResourceLoader` 本身不是 `@Published` 的嵌套对象：`MyPageModel` 需要把两个 loader 的 `objectWillChange` 转发到自己（用 Combine `sink`），否则界面不会刷新。

### 3. 页面（重写 `iOS/iOS/Features/Profile/MyPage.swift`，区块拆到 `MyPageSections.swift`）

整体是一个带 `.refreshable` 的 `ScrollView`，背景 `palette.page`，导航标题「我的」。内容自上而下，对照小程序模板与 scss：

**会话恢复中**（`session.isRestoring`）与**恢复失败**（`session.restoreError`）：保留现有的「正在恢复登录状态」和错误 + 「重试」两块。

**访客**（没有身份）：

- 访客卡：灰色人像图标、「未登录」、「登录后同步你的段位、战绩和好友」；三条权益（「记录每一局真实比分」「四个球种分别积累排位分」「查看胜率、连胜和过往对手」），每条前面一个对勾；主按钮「登录 / 注册」→ 登录页。
- 列表卡：「排行榜 / 各球种高手的段位与积分」→ `.ranking`；「规则百科 / 常见台球规则与术语」→ `.rulesIndex`；「帮助、投诉与举报 / 无需登录即可提交问题与反馈」→ `.feedback`。
- 另加一行「设置」→ `.settings`（原生端访客也要能改主题、看协议，这是既有规格的要求，小程序没有这一行）。

**已登录**：

1. 资料头：左侧头像（56pt；会员且有段位头像框时显示头像框，用 `MemberAvatarView`，`showBadge: false`，`reserveFrame: true`；`vipLevel` / `rankLevel` 取自 `memberIdentity` 对应的来源），昵称（19pt，heavy，单行截断，空昵称显示「用户」），下面两个胶囊标签：等级标签（`profileLevelText`，会员时用 `brandTint` 底 + `brandStrong` 字）→ 点击去 `.memberCenter`；信誉标签（`reputationChip.text`，受限时用危险色浅底 + `danger` 字）→ 点击去 `.reputation`。点击头像/昵称区域 → `.editNickname`。右侧两个 36pt 圆形图标按钮：二维码（SF Symbol `qrcode`，无障碍标签「出示我的二维码」）→ 打开二维码弹层；设置（`gearshape`，无障碍标签「设置」）→ `.settings`。
2. 段位卡（高价值表面）：`LinearGradient(colors: palette.heroColors, startPoint: .topLeading, endPoint: .bottomTrailing)`，圆角 16，白字。
   - `rankGameView.tabs` 非空 → 顶部一排分段胶囊（半透明黑底，选中项白底深棕字），点击只改 `selectedRankGameType`；为空且段位已读到 → 显示一行球种名（`primaryLabel`）。
   - `isRankSkeleton` → 骨架占位（一个圆 + 两条横条，半透明白）；否则是段位主体：段位图标（48pt，`RemoteImage`）、段位名（20pt heavy）、`ranked` 时「排位分 {score}」、右侧「段位说明 ›」；`ranked` 时显示进度条（6pt 高）；最下面 `hint`。点击主体 → `.rankExplain(gameType: currentRankGameType)`。
   - `showRankGamePicker` → 卡片底部显示「你主要打哪种？选好后发起 PK 也会默认选中」和四个球种选项，点了调 `pickPreferredGameType` 并 Toast「已设为常用球种」。
3. 生涯数据：`SectionHeader(title: "生涯数据", caption: "全部球种", moreTitle: "竞技分析")` → `.statsDetail(gameType: currentRankGameType)`。
   - `.empty` → 卡片内「还没有对局记录」+「打完第一场，场次、胜率和连胜都会记在这里」。
   - 其他 → 四格指标卡（数值 22pt bold，胜率用 `brandStrong`，标签 12pt `textSecondary`），整卡点击同上。
   - `.error` 时在下面加一行可点击的「生涯数据读取失败，点击重试」→ `model.refresh()`。
4. 快捷入口（一张卡四等分，图标在上文字在下）：对局记录 → `.matchHistory`；过往对手 → `.opponentRecord`；荣誉墙 → `.achievementWall(gameType: currentRankGameType)`；好友 → `.friendList`。图标用 SF Symbols（`list.bullet.rectangle`、`person.2`、`medal`、`person`），放在 `brandTint` 圆底上，颜色 `brandStrong`。
5. 奖励入口（仅 `compactRewardEntry.inline` 时显示）：标题 + 右侧状态文字 + 描述，纯展示。
6. 服务列表（一张卡，行高 52，行间分隔线）：约球记录 → `.challenges(showHistory: true)`；VIP中心（右侧显示 `memberEntry.text`）→ `.memberCenter`；装备 → `.gearIndex`；赛季 → `.seasonIndex`；规则百科 → `.rulesIndex`；附近球馆 → `.venueList`。图标同样用 SF Symbols。
7. 悬浮奖励入口（`floatRewardVisible` 时，用 `.safeAreaInset(edge: .bottom)` 贴在内容底部、Tab 栏之上）：标题、描述、右侧动作文字（`actionText`）和一个关闭按钮（无障碍标签「稍后提醒」）。点主体 → `.venueSubmit`；点关闭 → `model.snoozeFloatReward()`。

生命周期：`.task { await model.onAppear() }`（每次页面出现都会触发——从子页面返回、切回 Tab）、`.refreshable { await model.refresh() }`、`.onChange(of: session.identity) { await model.onIdentityChanged() }`。`MyPageModel` 用 `@StateObject` 持有，`init(session:)` 的写法参照 `MatchHistoryView`；`MainTabView` / `AppRouteDestination` 里创建 `MyPage` 的地方相应调整。

本轮**不做**：换季结果弹窗（`showLatestSeasonRollover`）、消息/好友角标相关的 store——后面的轮次再做。

### 4. 我的匹配二维码（新建 `iOS/iOS/Features/Profile/MatchQRCodeSheet.swift`）

对应模板里的 `qr-mask` 弹层，原生端用 `.sheet` + `presentationDetents([.medium])`：

- 标题「出示我的匹配二维码」、副标题「请让对手使用发起对局扫码入口识别」、右上角关闭按钮。
- 三种状态：生成中（`ProgressView` +「二维码生成中...」）、失败（「二维码加载失败」+「重试」）、成功（`QRCodeImage.make` 的图片，240pt 见方，白底，四周留白——暗色主题下也必须是白底黑码，否则扫不出来）。
- 底部按钮「关闭」。
- 读取用 `ProfileAPI.matchQRCode()`。弹层关闭后或身份变化后到达的结果必须丢弃；关闭时清空已生成的二维码内容（凭证不滞留）。读取逻辑放进一个小的 `@MainActor final class MatchQRCodeModel: ObservableObject`（`phase: loading / failed / ready(UIImage)`，`load()`、`reset()`），便于测试。
- 不要把 `qrcode_data` 打日志、存本地或放进路由。

## 必须新增的测试

- `MyPageModelTests.swift`（`MockHTTPServer` + `ProfileFixtures`，偏好用独立的 `UserDefaults` suite）：
  1. 访客 `onAppear()` 不发任何请求，推导值是访客初始状态；
  2. 已登录 `onAppear()` 各请求一次概览和段位；60 秒内再次 `onAppear()` 不请求，超过 60 秒（注入的 `now` 前进）各再请求一次；
  3. `careerState` 的四种结果（读取中 / 有对局 / 无对局 / 概览失败或 stats 分区不可用）；
  4. 段位切换：`selectedRankGameType` 在 tabs 里生效、不在 tabs 里回落到 `primaryGameType`；`pickPreferredGameType` 写入偏好并使 `showRankGamePicker` 变为 false；
  5. 概览失败不影响段位卡，段位失败不影响生涯数据；
  6. 悬浮奖励：`not_started` 且 enabled 时显示；`snoozeFloatReward()` 后隐藏，3 天内保持隐藏，3 天后再次显示；`pending_review` 不显示悬浮、`compactRewardEntry.inline == true`；
  7. 退出登录后 `onIdentityChanged()`：推导值回到访客状态；读取中退出，迟到的结果不改变状态。
- `MatchQRCodeModelTests.swift`：成功生成；`qrcode_data` 为空或请求失败 → `failed`；`reset()` 后到达的结果被丢弃且 phase 保持初始；重试成功。
- `MyPageSnapshotTests.swift`（用 `renderSnapshot`，亮暗各一张，页面放在带 `AppNavigator`、`ToastCenter`、`SessionStore` 环境的 `NavigationStack` 里，等数据读完再渲染）：
  1. `my-guest`：访客；
  2. `my-member`：已登录、会员有效（VIP3）、中式八球黄金段位、打过两个球种（有切换）、有对局数据、奖励 `pending_review`；
  3. `my-newcomer`：已登录、非会员、四个球种都没打过排位（显示球种选择）、无对局（生涯空态）、奖励 `not_started`（显示悬浮入口）；
  4. `my-qrcode`：`MatchQRCodeSheet` 的成功状态单独渲染（不通过 sheet 弹出，直接渲染其内容视图，尺寸 393×420）。
- 更新 `SharedUISnapshotTests` 里用到 `MyPage` / `MainTabView` 的地方，使其继续通过。

## 自检（在 `iOS/` 目录执行，超时 600000 毫秒，输出必须过滤）

```bash
cd iOS
xcodebuild build -project iOS.xcodeproj -scheme iOS -configuration Debug \
  -destination 'generic/platform=iOS Simulator' CODE_SIGNING_ALLOWED=NO 2>&1 | grep -E "error:|BUILD (SUCCEEDED|FAILED)" | head -60

xcodebuild test -project iOS.xcodeproj -scheme iOS -configuration Debug \
  -destination "platform=iOS Simulator,id=1A59DDBE-8E27-488F-864C-E7D44C52B490" \
  -parallel-testing-enabled NO 2>&1 | grep -E "error:|: error|failed|Executed [0-9]+ tests|TEST (SUCCEEDED|FAILED)" | tail -60

xcodebuild build -project iOS.xcodeproj -scheme iOS -configuration Release \
  -destination 'generic/platform=iOS Simulator' CODE_SIGNING_ALLOWED=NO 2>&1 | grep -E "error:|BUILD (SUCCEEDED|FAILED)" | head -20
```

要求：Debug 与 Release 构建成功；全部测试通过。测试失败就修到通过，不允许靠删除、跳过或弱化测试来通过。

## 交付

完成后用中文给出：新增/修改的文件清单、测试总数与结果（贴出 `Executed N tests` 那一行）、与本规格或小程序界面不一致的地方及原因。没做到的写明没做到。
