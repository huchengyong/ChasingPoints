# R2.1a：「我的」主页的数据模型、接口与业务规则（纯逻辑，不含界面）

你在一个 git worktree 的根目录里工作（当前目录就是仓库根）。总目标是把微信小程序（`uniapp/`）几乎 1:1 迁移到原生 SwiftUI（`iOS/`）。骨架已就绪（`iOS/iOS/App/`、`iOS/iOS/Shared/`）。本轮为小程序「我的」页（`uniapp/pages/user/index.vue`）迁移**数据模型、接口门面和业务规则**，并把小程序对应的 Node 测试用例迁成 XCTest。**本轮不写任何 SwiftUI 界面，不改 `MyPage.swift`**，界面在下一轮做。

先读：`AGENTS.md`、`iOS/AGENTS.md`、`iOS/iOS/` 下全部 Swift 文件（重点 `SessionStore.swift`、`Shared/ResourceLoader.swift`、`Shared/MemberIdentity.swift`、`Shared/CDNAsset.swift`、`MatchHistoryAPI.swift`、`MatchHistoryModels.swift`），`iOS/iOSAuthTests/` 下的 `TestSupport.swift`、`MockHTTPServer.swift`、`AuthFixtures.swift`、`ResourceLoaderTests.swift`。
小程序侧：`uniapp/pages/user/index.vue`（script 部分）、`uniapp/utils/user-homepage.js`、`game-types.js`、`game-type-preference.js`、`member-center.js`、`favorite-venue-reward.js`、`rank-cache.js`、`member-identity.js`，以及 `uniapp/tests/` 下的 `user-homepage.test.mjs`、`game-types.test.mjs`、`game-type-preference.test.mjs`、`member-growth-center.test.mjs`、`member-center.test.mjs`、`favorite-venue-reward.test.mjs`、`rank-info-cache.test.mjs`。
接口契约：`backend/chasing_points.api` 里的 `GetUserOverviewResp`、`UserStatsResp`、`GetUserReputationResp`、`GetMemberStatusResp`、`FavoriteVenueRewardStatusResp`、`GetUserRankInfosResp`、`UserRankInfoItem`、`RankInfo`、`GetMatchQRCodeResp`。

## 硬性约束（违反任何一条都算失败）

1. 只允许新增/修改 `iOS/iOS/**` 与 `iOS/iOSAuthTests/**`。不得修改 `iOS/iOS.xcodeproj/**`、`uniapp/**`、`backend/**` 及其他目录。
2. 不执行任何 git 写操作，只允许 `git status`、`git diff` 等只读命令。
3. 不联网、不 ssh、不 curl；不读取任何 `.env`、密钥、Keychain、真实会话；不访问当前目录之外的路径（Xcode 默认 DerivedData 除外）。
4. 不引入第三方依赖。最低部署目标 iOS 17.0。
5. 现有全部测试必须继续通过。不重命名、不移动、不重构现有代码（任务 1 点名的 `ResourceLoader` 扩展除外）。
6. 注释与文案用简体中文；文案必须与小程序源码**逐字一致**（包括标点和空格，例如 `距 Lv3 还差 5 场`、`1 个月VIP`）。
7. 不创建 Markdown / 文档文件。不写 SwiftUI 视图。

## 工程须知

- App target 默认 MainActor 隔离、Swift 5 语言模式；测试类标 `@MainActor`。
- 解码统一 `convertFromSnakeCase`：`member_status` → `memberStatus`，`rank_infos` → `rankInfos`，`max_win_streak` → `maxWinStreak`。**注意**这个策略对带数字的键的转换结果要以实际运行为准（例如 `player1_name` → `player1Name`），拿不准就写测试确认。
- `APIClient` 只检查响应**顶层**的 `success`；聚合接口里各分区自己的 `success` 需要业务代码判断。
- 需要「当前时间」的规则一律把 `now: Date` 作为参数传入，测试里传固定时间；不要在规则内部直接 `Date()`。
- 服务端的时间文本有两种：`yyyy-MM-dd HH:mm:ss`（按北京时间 +08:00 解释）和 ISO 8601；解析规则照搬 JS 里 `parseMemberExpiresAt` / `parseRewardTime` 的做法。
- bash 工具跑 xcodebuild 时超时设为 600000 毫秒，输出必须用 grep 过滤。

## 任务

### 1. `ResourceLoader` 增加新鲜度（修改 `iOS/iOS/Shared/ResourceLoader.swift`）

Tab 根页面常驻不销毁，需要「隔一段时间再出现就静默刷新」，对应小程序 `USER_OVERVIEW_CACHE_TTL = 60 秒`。

- `init` 增加参数 `now: @escaping () -> Date = Date.init`（放在 `load` 之前，有默认值，现有调用不用改）。
- 记录最近一次成功采纳结果的时间。
- `loadIfNeeded()` 改为 `loadIfNeeded(maxAge: TimeInterval? = nil)`：其余规则不变；当 `phase` 是 `.loaded`、未被 `invalidate()`、且传了 `maxAge`、且距上次成功已超过 `maxAge` 时，以刷新方式重读（保留旧数据）。不传 `maxAge` 时行为与现在完全一致。
- 身份变化时清掉记录的时间。
- 新增测试：未超过 `maxAge` 不请求；超过后刷新且期间旧数据可见；不传 `maxAge` 永不因时间重读。

### 2. 球种目录与常用球种偏好（新建 `iOS/iOS/Shared/GameType.swift`、`iOS/iOS/Shared/GameTypePreference.swift`）

```swift
/// 球种目录，顺序与小程序 utils/game-types.js 一致：中式八球(3)、九球追分(2)、斯诺克(1)、美式九球(4)。
struct GameTypeOption: Equatable, Identifiable {
    let value: Int
    let key: String        // chinese_eight / nine_ball / snooker / american_nine
    let label: String
    var id: Int { value }
}

enum GameTypeCatalog {
    static let ordered: [GameTypeOption]
    static func label(for gameType: Int, fallback: String = "台球") -> String
    static func key(for gameType: Int) -> String?
    static func value(forKey key: String) -> Int?
    static func isSupported(_ gameType: Int) -> Bool
}

/// 用户的常用球种，按账号保存在本机；0 表示「每次询问」。对应 utils/game-type-preference.js。
struct GameTypePreference {
    init(defaults: UserDefaults = .standard)
    static func storageKey(userId: Int64) -> String?      // "default-game-type:<userId>"，userId <= 0 返回 nil
    static func normalize(_ value: Int) -> Int             // 不支持的值 → 0
    func read(userId: Int64) -> Int
    @discardableResult func save(userId: Int64, gameType: Int) -> Int   // 0 或不支持的值 → 删除并返回 0
}
```

- 不要改 `MatchHistoryModels.swift` 里现有的球种文案逻辑。
- 测试：把 `game-types.test.mjs`、`game-type-preference.test.mjs` 里与上述接口对应的用例搬过来（偏好测试用 `UserDefaults(suiteName:)` 的独立 suite，测完 `removePersistentDomain`）。

### 3. 数据模型与接口（新建 `iOS/iOS/Features/Profile/ProfileModels.swift`、`ProfileAPI.swift`）

模型字段与 `.api` 对齐，全部 `Decodable, Equatable`：

- `UserStats`：`success`、`totalMatches`、`wins`、`losses`、`winRate: Double`、`maxWinStreak`。
- `ReputationStatus`：`success`、`score`、`status`、`statusText`、`banUntil: String?`。
- `MemberStatus`：`success`、`isActive`、`currentTime`、以及全部 optional 字段（`planCode`、`planName`、`memberExpiresAt`、`growthLevel`、`growthPoints`、`todayGrowthCount`、`growthDailyCap`、`growthFrozen`、`nextGrowthLevel`、`nextGrowthLevelPoints`、`remainingGrowthPoints`，都用可选类型）。
- `FavoriteVenueRewardStatus`：`success`、`enabled`、`popupEnabled`、`rewardDays`、`newUserWindowDays`、`status`、`memberExpiresAt?`、`submittedVenueId?`、`submittedVenueName?`、`rejectReason?`。
- `UserOverview`：`stats: UserStats?`、`reputation: ReputationStatus?`、`memberStatus: MemberStatus?`、`favoriteVenueReward: FavoriteVenueRewardStatus?`。另提供四个只读属性 `availableStats` / `availableReputation` / `availableMemberStatus` / `availableFavoriteVenueReward`：分区存在且其 `success == true` 才返回，否则 nil（对应页面里的 `overview.stats?.success ? overview.stats : null`）。
- `RankInfo`：`level`、`name`、`icon`、`rankScore`、`totalWins`、`totalLosses`、`maxStreak`、`nextLevel`、`nextName`、`nextScore`、`progress`、`placementRemaining`（`placementRemaining` 用 `Int?`，旧服务端可能不返回）。
- `RankInfoMap`：`typealias RankInfoMap = [Int: RankInfo]`，由 `rank_infos` 归一化得到——只保留球种 1–4 且 `rank_info` 非空的项（照搬 `normalizeUserRankInfos`）。
- `MatchQRCode`：`qrcodeData: String`、`expiresInSeconds: Int64?`。**不要**把 `invite_token` 解码进模型（凭证不进模型、不进日志）。

```swift
/// 「我的」页的读取入口，全部经由 SessionStore.authorizedRequest。
@MainActor
struct ProfileAPI {
    let session: SessionStore
    func overview() async throws -> UserOverview          // GET /api/user/overview
    func rankInfos() async throws -> RankInfoMap          // GET /api/rank/infos
    func matchQRCode() async throws -> MatchQRCode        // GET /api/match/qrcode；qrcode_data 为空时抛 APIError(message: "二维码加载失败", kind: .invalidResponse)
}
```

测试夹具新建 `iOS/iOSAuthTests/ProfileFixtures.swift`（写法参照 `MatchHistoryFixtures.swift`，字段名必须是服务端的 snake_case）。

### 4. 会员成长卡（新建 `iOS/iOS/Features/Profile/MemberGrowth.swift`）

迁移 `member-center.js` 里的 `resolveMemberGrowthCard` 及它依赖的私有函数（会员是否有效、成长值、每日上限、当前等级、下一等级、进度百分比等）。只迁这一条调用链，`member-center.js` 里其他导出（套餐、支付、入口卡、摘要等）本轮不迁。

```swift
struct MemberGrowthCard: Equatable {
    let statusText: String
    let level: Int
    let levelLabel: String          // "Lv3"
    let growthPoints: Int
    let growthPointsText: String
    let todayGrowthCount: Int
    let dailyCap: Int
    let todayProgressText: String
    let frozen: Bool
    let description: String
    let nextLevelText: String
    let nextLevel: Int?             // 已是最高级为 nil
    let progressPercent: Int

    static func resolve(_ status: MemberStatus?, now: Date) -> MemberGrowthCard
}
```

`status` 为 nil 等同 JS 传入 `{}`。等级阈值、默认每日上限等常量照搬 JS。测试：把 `member-growth-center.test.mjs`（以及 `member-center.test.mjs` 里涉及 `resolveMemberGrowthCard` 的用例）逐条搬过来。

### 5. 常玩球馆奖励（新建 `iOS/iOS/Features/Profile/FavoriteVenueReward.swift`）

迁移 `favorite-venue-reward.js` 的：`formatRewardDuration`、`isFavoriteVenueRewardClaimable`、`resolveFavoriteVenueRewardTaskCard`、悬浮入口的稍后提醒（`getFavoriteVenueRewardFloatSnoozeKey`、`calcFavoriteVenueRewardFloatSnoozeUntil`、`isFavoriteVenueRewardFloatSnoozed`，时长 3 天）。**不迁**旧弹窗键的迁移逻辑（`resolveFavoriteVenueRewardFloatSnoozeMigration`、popup 相关——原生端没有历史数据）和 `resolveFavoriteVenueMemberCard`（会员中心那一轮再做）。

```swift
struct FavoriteVenueRewardTaskCard: Equatable {
    let visible: Bool
    let title: String
    let description: String
    let actionText: String
    let statusText: String
}

enum FavoriteVenueReward {
    static func formatDuration(rewardDays: Int) -> String
    static func isClaimable(_ status: FavoriteVenueRewardStatus?) -> Bool
    static func taskCard(_ status: FavoriteVenueRewardStatus?) -> FavoriteVenueRewardTaskCard
    static func floatSnoozeKey(userId: Int64, status: String) -> String
    static func snoozeUntil(from now: Date) -> Date
    static func isSnoozed(until: Date?, now: Date) -> Bool
}
```

`status` 为 nil 等同 JS 传入 `{}`（`reward_days` 缺省按 30）。测试：把 `favorite-venue-reward.test.mjs` 里对应这些函数的用例逐条搬过来。

### 6. 「我的」页规则（新建 `iOS/iOS/Features/Profile/UserHomepageRules.swift`）

迁移 `user-homepage.js` 里页面用到的规则。函数逐条对应、文案逐字一致：

```swift
struct CoreMetric: Equatable, Identifiable {
    let key: String      // total / wins / win-rate / max-streak
    let label: String
    let value: String    // 加载中为 "—"
    var id: String { key }
}

struct RankCardView: Equatable {
    let ranked: Bool
    let name: String
    let iconURL: URL?     // 经 CDNAsset.url(for:)；默认图标 /static/images/ranks/rank_bronze.png
    let score: Int
    let progress: Int     // 0...100
    let hint: String
}

struct RankGameTypeView: Equatable {
    let primaryGameType: Int
    let primaryLabel: String
    let tabs: [GameTypeOption]   // 不足两个时为空
    let needsPick: Bool
}

struct ReputationChip: Equatable {
    let text: String
    let restricted: Bool
}

struct MemberEntry: Equatable {
    let state: String    // none / active / expired
    let text: String
}

struct CompactRewardEntry: Equatable {
    let visible: Bool
    let floatable: Bool
    let inline: Bool
    let state: String
    let title: String
    let description: String
    let statusText: String
    let action: String       // "favorite-venue" 或 ""
    let actionText: String
}

enum UserHomepageRules {
    static func coreMetrics(stats: UserStats?, isLoading: Bool) -> [CoreMetric]
    static func rankCard(_ rankInfo: RankInfo?) -> RankCardView
    static func rankGameTypeView(preferredGameType: Int, rankInfoMap: RankInfoMap) -> RankGameTypeView
    static func reputationChip(_ reputation: ReputationStatus?, isLoading: Bool) -> ReputationChip
    static func selfMemberIdentity(isLoggedIn: Bool, memberStatus: MemberStatus?, rankInfo: RankInfo?, rankLoading: Bool, now: Date) -> MemberIdentity
    static func memberEntry(_ memberStatus: MemberStatus?, now: Date) -> MemberEntry
    static func compactRewardEntry(_ rewardStatus: FavoriteVenueRewardStatus?) -> CompactRewardEntry
    /// 资料头部等级标签：对应页面里的 profileLevelText。
    static func profileLevelText(memberStatus: MemberStatus?, identity: MemberIdentity, isLoading: Bool, now: Date) -> String
}
```

- `coreMetrics`：胜率的显示与 JS 的 `` `${winRate}%` `` 一致——JS 数字转字符串时整数不带小数（`50` → `50%`，`66.7` → `66.7%`），Swift 的 `Double` 要自己处理，不能输出 `50.0%`。
- `profileLevelText`：`memberStatus == nil && isLoading` → `等级同步中`；`identity.showBadge` → `"\(badgeText) · \(tierLabel)"`；否则 `MemberGrowthCard.resolve(memberStatus, now:).levelLabel`。
- `resolveStatusCardContent`、`resolveHighestRankDisplay`、`formatDuration` 本轮不迁（「我的」页没用到）。
- 测试：把 `user-homepage.test.mjs` 里对应上述函数的用例逐条搬过来，并补：`coreMetrics` 的整数/小数胜率显示、`profileLevelText` 的三种分支、`selfMemberIdentity` 在会员过期 / 段位加载中 / 未登录时的结果。

### 7. 二维码生成（新建 `iOS/iOS/Shared/QRCodeImage.swift`）

```swift
enum QRCodeImage {
    /// 用系统 CoreImage 生成二维码位图（纠错级别 M、黑白、不带额外留白之外的装饰）。内容为空或生成失败返回 nil。
    static func make(content: String, size: CGFloat) -> UIImage?
}
```

- 用 `CIFilter.qrCodeGenerator()`，按整数倍放大到不小于 `size * UIScreen.main.scale` 的像素尺寸，最近邻插值（不能糊）。
- 测试：生成的图片用 `CIDetector(ofType: CIDetectorTypeQRCode, ...)` 能识别回原内容（内容用一段含中文和 URL 参数的字符串）；空内容返回 nil。如果模拟器上 `CIDetector` 不可用，改为断言图片非空、边长不小于请求尺寸，并在交付说明里写明。

## 接口读取的测试（`ProfileAPITests.swift`）

用 `MockHTTPServer`：

1. `overview()`：四个分区都成功时全部可用；某个分区 `success: false` 或为 `null` 时对应的 `availableXxx` 为 nil、其他分区不受影响；顶层 `success: false` 时抛错。
2. `rankInfos()`：只保留球种 1–4 且 `rank_info` 非空的项；`rank_infos` 为空数组时返回空字典。
3. `matchQRCode()`：成功返回正文；`qrcode_data` 为空时抛错；请求带 Bearer。
4. 三个接口未登录时都不发请求、直接抛错。

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

要求：Debug 与 Release 构建成功；全部测试通过。测试失败就修到通过，不允许靠删除、跳过或弱化测试来通过。如果发现小程序的 JS 测试期望与你的实现不一致，以 JS 源码的实际行为为准，不要改期望值去迁就实现。

## 交付

完成后用中文给出：新增/修改的文件清单；每个被迁移的 JS 测试文件搬了多少条用例、有没有没搬的及原因；测试总数与结果（贴出 `Executed N tests` 那一行）；与本规格不一致的地方及原因。没做到的写明没做到。
