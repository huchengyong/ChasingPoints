# RB2：排行榜 + 段位说明（对应小程序 `pages/ranking/index`、`subPages/user/rankExplain`）

你在一个 git worktree 的根目录里工作（当前目录就是仓库根）。总目标是把微信小程序（`uniapp/`）几乎 1:1 迁移到原生 SwiftUI（`iOS/`）。骨架、共享组件、「我的」页、规则百科已经迁完。本轮迁移两个页面：排行榜、段位说明。

先读：`AGENTS.md`、`iOS/AGENTS.md`、`DESIGN.md`；`iOS/iOS/` 下全部 Swift 文件（重点 `App/`、`Shared/`、`Shared/UI/`、`Features/Rules/`——它是「访客可读页面」的现成范例，`Features/Profile/ProfileModels.swift` 里的 `RankInfo`、`PagedHistoryList.swift`）；`iOS/iOSAuthTests/` 的 `TestSupport.swift`、`MockHTTPServer.swift`、`SnapshotSupport.swift`、`RulesSnapshotTests.swift`、`PagedHistoryListTests.swift`。
小程序侧：`uniapp/pages/ranking/index.vue` + `index.scss`、`uniapp/utils/ranking-podium.js`、`uniapp/tests/ranking-podium.test.mjs`；`uniapp/subPages/user/rankExplain.vue` + `rankExplain.scss`、`uniapp/utils/rank-explain.js`、`uniapp/tests/rank-explain.test.mjs`；`uniapp/api/rank.js`。
接口契约（`backend/chasing_points.api`）：`GetLeaderboardReq/Resp`、`LeaderboardItem`（`GET /api/public/rank/leaderboard`，访客可读，登录后带 `my_ranking`）；`GetRankListReq/Resp`、`RankItem`（`GET /api/rank/list`，需登录）；`GetUserRankInfosResp`（已有 `ProfileAPI.rankInfos()`）；`GetRankConfigsResp`、`RankConfigItem`（`GET /api/public/rank/configs`，访客可读）。

## 硬性约束（违反任何一条都算失败）

1. 只允许新增/修改 `iOS/iOS/**` 与 `iOS/iOSAuthTests/**`。不得修改 `iOS/iOS.xcodeproj/**`、`uniapp/**`、`backend/**`、`openspec/**` 及其他目录。
2. 不执行任何 git 写操作，只允许 `git status`、`git diff` 等只读命令。
3. 不联网、不 ssh、不 curl；不读取任何 `.env`、密钥、Keychain、真实会话；不访问当前目录之外的路径。
4. 不引入第三方依赖。最低部署目标 iOS 17.0。
5. 现有全部测试必须继续通过。除本规格点名的文件外不改现有代码。**另一条开发线正在并行修改 `iOS/iOS/SettingsView.swift`、`iOS/iOS/SessionStore.swift`、`iOS/iOS/Features/Profile/**`、`iOS/iOS/Features/Settings/**` 及其测试——这些文件一个字都不要碰。** `AppRoute.swift` 只改 `AppRouteDestination` 里 `.ranking` 和 `.rankExplain` 两个分支的目的地，不动其他行。
6. 颜色走 `AppTheme` / `AppPalette`（领奖台金 / 银 / 铜三个名次色、段位英雄卡的渐变可以在各自文件内定义私有常量，取值照搬 scss）。文案与小程序逐字一致。
7. 不创建 Markdown / 文档文件。

## 工程须知

- App target 默认 MainActor 隔离、Swift 5 语言模式；测试类标 `@MainActor`。
- 解码统一 `convertFromSnakeCase`。访客可读接口用 `session.optionalAuthRequest`，带参数的 GET 用 `APIQuery.path`。
- 失败提示统一用 `LoadFailure` 的 `feedback(resource:)`。
- 需要「当前时间」的规则把 `now: Date` 作为参数传入。
- rpx 换算大致 `pt ≈ rpx / 2`。
- 模拟器请用 **iPhone 15 Pro Max**（UDID `BE3E9E49-0427-45A2-A5DD-905C033EC260`）。xcodebuild 一律加 `-derivedDataPath build/dd-b`（相对 `iOS/` 目录，已被 git 忽略）。
- bash 工具跑 xcodebuild 时超时设为 600000 毫秒，输出必须用 grep 过滤。

## 任务

### 1. 排行榜（新建 `iOS/iOS/Features/Ranking/`）

`.ranking` → `RankingView`，导航标题「排行榜」。访客可看。

**模型与接口**（`RankingModels.swift`、`RankingAPI.swift`）：

```swift
struct LeaderboardItem: Decodable, Equatable, Identifiable {
    let rank: Int
    let userId: Int64
    let nickname: String
    let avatar: String
    let rankLevel: Int
    let rankName: String
    let rankScore: Int
    let winRate: Int
    let vipLevel: Int?        // 旧服务端可能不返回
    var id: Int64 { userId }
}

struct LeaderboardPage: Decodable, Equatable {
    let total: Int
    let topThree: [LeaderboardItem]
    let myRanking: LeaderboardItem?
    let list: [LeaderboardItem]
}

struct PodiumSlot: Equatable, Identifiable {
    let placement: String     // second / first / third（显示顺序）
    let rank: Int
    let user: LeaderboardItem?
    var id: String { placement }
    /// 对应 buildLeaderboardPodiumSlots：显示顺序是 第二名、第一名、第三名。
    static func build(topThree: [LeaderboardItem]) -> [PodiumSlot]
}

@MainActor
struct RankingAPI {
    let session: SessionStore
    static let pageSize = 20
    func leaderboard(gameType: Int, page: Int) async throws -> LeaderboardPage   // GET /api/public/rank/leaderboard?game_type=&page=&page_size=20
}
```

**页面模型** `@MainActor final class RankingModel: ObservableObject`，行为逐条对应 `index.vue` 的 `fetchLeaderboard`：

- 状态：首次加载 / 已加载 / 空 / 失败(`LoadFailure`)；`isRefreshing`、`isLoadingMore`、`refreshFailure: LoadFailure?`；`topThree`、`list`、`myRanking`、`total`、`currentGameType`（默认 3 中式八球）。
- `hasMore = topThree.count + list.count < total`。
- 前三名和「我的排名」只在第一页或刷新时更新；加载更多只追加 `list`。
- 加载更多失败：页码回退、已有数据不变。刷新失败且已有数据：数据不变，`refreshFailure` 有值。
- 切换球种：回到第一页并读取；旧球种的迟到结果丢弃。
- 只采纳最新请求；身份变化（登录 / 退出 / 切号）后清空并重读（登录后才有 `my_ranking`）。
- 空态判定：第一页读取成功且前三名和列表都为空。

**界面**（对照模板与 `index.scss`）：

- 顶部横向球种 Tab（四个球种，取自 `GameTypeCatalog.ordered`）。
- 领奖台：第二名 / 第一名 / 第三名三列，第一名最高并带奖牌图标；头像用 `MemberAvatarView`（`fitFrame: true`），名次角标（金 / 银 / 铜），昵称（空则「球手」）、「{分}分」，台座上显示段位名和「胜率 {n}%」。缺位的名次留空。
- 第 4 名起的列表：名次、`MemberAvatarView`（`reserveFrame: true`）、昵称、「{段位名} | {分}分」、右侧「胜率: {n}%」。滚动到底自动加载更多。
- 下拉刷新。
- 底部固定「我的排名」卡（有 `myRanking` 时显示，用 `.safeAreaInset(edge: .bottom)`）：名次（未上榜显示「-」）、头像、「我的排名」/「暂未上榜」、段位名 /「完成首场该模式对局后入榜」、右侧「{分}分」/「未记录榜单分数」。
- 状态：加载中「加载中...」；失败（`feedback(resource: "排行榜")`，重试 / 返回）；空「暂无排行数据」+「快去对战提升排名吧！」+ 按钮「发起PK」→ `navigator.requestScan(isSignedIn:)`；刷新失败时顶部一行提示 +「重试」。
- 点击榜单上的用户：Toast「对方隐藏战绩，无法查看」（与小程序一致）。

### 2. 段位说明（新建 `iOS/iOS/Features/Rank/`）

`.rankExplain(gameType:)` → `RankExplainView`，导航标题「段位说明」。需要登录（未登录显示登录引导）。

**规则迁移**（`RankExplainRules.swift`）：把 `uniapp/utils/rank-explain.js` 的导出逐条迁成 Swift——`MAX_RANK_LEVEL`、`buildRankGrowthPath`、`buildRankHeroModel`、`LEGACY_RANK_RULES`、`RATING_V2_RANK_RULES`、`resolveRankRulesState`、`resolveRankRules`，以及页面要用到的其他导出。两套计分规则的全部文案逐字照搬。`resolveRankExplainLoadingMode`、`shouldApplyRankExplainResponse`、`RANK_CONFIGS_STATIC_VERSION` 这类只服务于小程序缓存 / 请求去重的导出不用迁（原生端由页面模型处理）。把 `uniapp/tests/rank-explain.test.mjs` 里对应的用例全部搬成 XCTest。

**数据**（`RankExplainAPI.swift`）：

- `GET /api/rank/list?game_type=`（`authorizedRequest`）→ `[RankItem]`（`level`、`name`、`icon`、`minScore`、`isCurrent`）。
- `GET /api/public/rank/configs`（`optionalAuthRequest`）→ `rulesVersion`、`rulesEffectiveAt`（本页只用这两个字段决定展示哪套计分规则；读取失败时按旧规则展示，不阻塞页面）。
- 当前球种的段位信息用现有 `ProfileAPI.rankInfos()`。

**页面模型** `@MainActor final class RankExplainModel: ObservableObject`：对应 `rankExplain.vue` 的读取逻辑——进入时按传入球种（不合法时按页面里的回退规则）读取；切换球种时保留旧内容并标记刷新中（Tab 显示 pending 态），新数据到了再替换；首次失败显示错误态；只采纳最新请求；结果绑定身份。

**界面**（对照模板与 `rankExplain.scss`，自上而下）：吸顶球种 Tab；当前段位英雄卡（球种 ·「当前段位」、段位名、排位分、定级赛剩余场次胶囊、段位徽章图、晋级进度条或满级态）；「段位之路」（六个段位节点，已达成 / 当前 / 未达成三种状态）；「段位分怎么算？」及其下面的全部规则区块（按 `resolveRankRules` 的结果渲染，区块结构、标题、条目以模板为准，折叠 / 展开交互保持一致）。段位图标用 `RemoteImage(url: CDNAsset.url(for:))`。

错误态文案：「段位数据加载失败」「请检查网络连接后重新加载」「重新加载」。

### 3. 测试

- `RankingTests.swift`：`PodiumSlot.build`（搬 `ranking-podium.test.mjs`）；`RankingAPI` 的路径与参数、访客不带 Authorization、登录带 Bearer；`RankingModel`——首次加载、空榜、`hasMore` 计算、加载更多追加且不改前三名、加载更多失败回退页码、刷新失败保留数据、切换球种丢弃旧结果、登录后重读拿到 `myRanking`、退出后清掉 `myRanking`。
- `RankExplainRulesTests.swift`：搬运 `rank-explain.test.mjs` 全部适用用例。
- `RankExplainModelTests.swift`：首次读取三个接口；configs 失败时按旧规则且页面可用；切换球种期间旧内容可见；首次失败 → 错误态，重试成功；未登录不请求。
- `RankingSnapshotTests.swift`（亮暗各一张，等数据读完再渲染）：`ranking-full`（登录、前三名齐全、列表 6 条、有我的排名）；`ranking-guest-partial`（访客、只有 2 人上榜）；`ranking-empty`；`rank-explain-gold`（黄金段位、进度 60%、V2 规则）；`rank-explain-king`（王者满级）；`rank-explain-error`。

## 自检（在 `iOS/` 目录执行，超时 600000 毫秒，输出必须过滤）

```bash
cd iOS
xcodebuild build -project iOS.xcodeproj -scheme iOS -configuration Debug -derivedDataPath build/dd-b \
  -destination 'generic/platform=iOS Simulator' CODE_SIGNING_ALLOWED=NO 2>&1 | grep -E "error:|BUILD (SUCCEEDED|FAILED)" | head -60

xcodebuild test -project iOS.xcodeproj -scheme iOS -configuration Debug -derivedDataPath build/dd-b \
  -destination "platform=iOS Simulator,id=BE3E9E49-0427-45A2-A5DD-905C033EC260" \
  -parallel-testing-enabled NO 2>&1 | grep -E "error:|: error|failed|Executed [0-9]+ tests|TEST (SUCCEEDED|FAILED)" | tail -60

xcodebuild build -project iOS.xcodeproj -scheme iOS -configuration Release -derivedDataPath build/dd-b \
  -destination 'generic/platform=iOS Simulator' CODE_SIGNING_ALLOWED=NO 2>&1 | grep -E "error:|BUILD (SUCCEEDED|FAILED)" | head -20
```

要求：Debug 与 Release 构建成功；全部测试通过。测试失败就修到通过，不允许靠删除、跳过或弱化测试来通过。

## 交付

完成后用中文给出：新增/修改的文件清单；JS 用例搬了多少、没搬的及原因；测试总数与结果（贴出 `Executed N tests` 那一行）；与本规格或小程序不一致的地方及原因。没做到的写明没做到。
