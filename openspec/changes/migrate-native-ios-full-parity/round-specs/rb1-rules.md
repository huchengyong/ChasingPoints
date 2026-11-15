# RB1：错误反馈分类 + 规则百科（对应小程序 `subPages/rules/*`）

你在一个 git worktree 的根目录里工作（当前目录就是仓库根）。总目标是把微信小程序（`uniapp/`）几乎 1:1 迁移到原生 SwiftUI（`iOS/`）。骨架已就绪：`iOS/iOS/App/`（`AppRoute`、`AppNavigator`、`MainTabView`）、`iOS/iOS/Shared/`（`ResourceLoader`、`APIQuery`、`GameTypeCatalog` 等）、`iOS/iOS/Shared/UI/`（`LoadableContent`、`StateMessageView`、按钮样式、`appCard` 等）。本轮做两件事：给读取失败加上与小程序一致的**分类提示**，然后迁移**规则百科**三个页面。

先读：`AGENTS.md`、`iOS/AGENTS.md`、`DESIGN.md`；`iOS/iOS/` 下全部 Swift 文件；`iOS/iOSAuthTests/` 的 `TestSupport.swift`、`MockHTTPServer.swift`、`ResourceLoaderTests.swift`、`SnapshotSupport.swift`、`SharedUISnapshotTests.swift`。
小程序侧：`uniapp/subPages/rules/index.vue`、`detail.vue`、`glossary.vue`（含各自的 `<style>`），`uniapp/api/rules.js`，`uniapp/utils/async-page-state.js`（`resolveAsyncPageErrorFeedback`）、`uniapp/utils/request-errors.js`（`classifyRequestError`），`uniapp/tests/async-page-state.test.mjs`。
接口契约：`backend/chasing_points.api` 的 `/api/rules/*`（`GetRuleContentReq/Resp`、`RuleContentItem`、`GetGlossaryReq/Resp`、`GlossaryItemInfo`、`SearchRulesReq/Resp`）。

## 硬性约束（违反任何一条都算失败）

1. 只允许新增/修改 `iOS/iOS/**` 与 `iOS/iOSAuthTests/**`。不得修改 `iOS/iOS.xcodeproj/**`、`uniapp/**`、`backend/**` 及其他目录。
2. 不执行任何 git 写操作，只允许 `git status`、`git diff` 等只读命令。
3. 不联网、不 ssh、不 curl；不读取任何 `.env`、密钥、Keychain、真实会话；不访问当前目录之外的路径（Xcode 默认 DerivedData 除外）。
4. 不引入第三方依赖。最低部署目标 iOS 17.0。
5. 现有全部测试必须继续通过。除本规格点名的文件外不改现有代码。**有另一条开发线正在并行修改 `iOS/iOS/Features/Profile/**`、`MyPage` 相关测试——这些文件一个字都不要碰。** 对 `ResourceLoader.swift`、`LoadableContent.swift`、`AppRoute.swift` 的修改必须是「只增不改」：不改现有属性、方法、枚举 case 的名字和含义。
6. 颜色一律走 `AppTheme` / `AppPalette`。文案与小程序逐字一致。
7. 不创建 Markdown / 文档文件。

## 工程须知

- App target 默认 MainActor 隔离、Swift 5 语言模式；测试类标 `@MainActor`。
- 解码统一 `convertFromSnakeCase`。访客可读的接口用 `session.optionalAuthRequest`。带参数的 GET 用 `APIQuery.path` 拼接。
- rpx 换算：大致 `pt ≈ rpx / 2`。字号用 `.font(.system(size:weight:))`。
- 模拟器请用 **iPhone 15 Pro Max**（UDID `BE3E9E49-0427-45A2-A5DD-905C033EC260`），其他模拟器被别的任务占用。xcodebuild 一律加 `-derivedDataPath build/dd-b`（相对 `iOS/` 目录；`build/` 已被 git 忽略），不要用默认 DerivedData。
- bash 工具跑 xcodebuild 时超时设为 600000 毫秒，输出必须用 grep 过滤。

## 任务

### 1. 读取失败的分类提示

小程序把请求错误分成 network / session / permission / not-found / rate-limit / server / business 七类，并据此给出不同的标题、说明和操作（`resolveAsyncPageErrorFeedback`）。原生端目前只显示「加载失败 + 原始文案」，需要对齐。

新建 `iOS/iOS/Shared/LoadFailure.swift`：

```swift
/// 读取失败的分类与界面提示，规则对应小程序 utils/async-page-state.js 的 resolveAsyncPageErrorFeedback。
struct LoadFailure: Equatable {
    enum Category: String {
        case network, session, permission, notFound, rateLimit, server, business
    }

    let category: Category
    let message: String          // 原始错误文案

    /// 由任意错误得到分类：APIError 按 kind / statusCode 判断，其余错误归为 business。
    init(_ error: Error)
    init(category: Category, message: String)

    /// 界面提示。resource 是资源名（如「规则内容」），用于「{resource}加载失败」。
    func feedback(resource: String = "内容") -> Feedback

    struct Feedback: Equatable {
        let title: String
        let description: String
        let actionText: String      // 「重试」或「返回」
        let retryable: Bool
    }
}
```

- 分类规则对应 `classifyRequestError`：`APIError.kind == .network` → network；`.sessionInvalid` / `.unauthorized` 或 statusCode 401 → session；statusCode 403 → permission；404 → notFound；429 → rateLimit；`.server` 或 statusCode ≥ 500 → server；其余（含 `.business`、`.invalidResponse`、非 `APIError`）→ business。
- `feedback` 的七种标题 / 说明 / 操作文案逐字照搬 `resolveAsyncPageErrorFeedback`（business 的说明用原始 `message`，为空时用「请稍后重试」）。

`ResourceLoader`（只增不改）：新增 `@Published private(set) var failure: LoadFailure?`——首次加载失败时与 `phase = .failed(message)` 同时设置；开始新的读取、读取成功、身份变化时清空。`phase`、`failureMessage`、`refreshError` 现有行为完全不变。

`LoadableContent`（只增不改）：`init` 增加参数 `resource: String = "内容"`（放在 `loginPrompt` 之后，有默认值）。失败态改为用 `loader.failure?.feedback(resource:)` 渲染：标题、说明、操作按钮文案取自 feedback；`retryable` 为 true 时点按钮调 `loader.retry()`，为 false 时调 `@Environment(\.dismiss)` 返回上一页。`loader.failure` 为 nil 时退回现在的写法。

测试：

- `LoadFailureTests.swift`：七种分类各自的判定条件；`feedback` 的七组文案（把 `async-page-state.test.mjs` 里关于 `resolveAsyncPageErrorFeedback` 的用例搬过来）；business 且 message 为空时的说明。
- `ResourceLoaderTests.swift` 追加：首次加载遇到 503 → `failure?.category == .server`；重试成功后 `failure == nil`；刷新失败不设置 `failure`（只有 `refreshError`）。

### 2. 规则百科

新建目录 `iOS/iOS/Features/Rules/`。

**路由**（`AppRoute.swift`，只增）：新增 `case ruleDetail(category: String)`，标题「规则详情」。`AppRouteDestination` 把 `.rulesIndex` → `RulesIndexView`、`.ruleDetail` → `RuleDetailView`、`.glossary` → `GlossaryView`。

**模型与接口**（`RulesModels.swift`、`RulesAPI.swift`）：

```swift
struct RuleContentItem: Decodable, Equatable, Identifiable {
    let id: Int64
    let title: String
    let content: String
    let sortOrder: Int
}

struct GlossaryItem: Decodable, Equatable, Identifiable {
    let id: Int64
    let title: String
    let content: String
    let category: String
}

enum RuleContentType: String, CaseIterable {   // 规则 / 犯规 / 术语
    case rule, foul, glossary
    var label: String { get }
}

/// 规则百科的球种分类。
enum RuleCategory {
    static let ordered: [(key: String, name: String, desc: String, emoji: String)]
    static func label(for category: String, fallback: String = "") -> String   // 对应 getRuleCategoryLabel：查不到时返回 fallback，fallback 为空则返回 category 本身
    static func emoji(for category: String) -> String                            // 查不到返回 🎱
    /// 美式九球没有单独的规则内容，读取时按九球追分取。
    static func contentCategory(for category: String) -> String
}

@MainActor
struct RulesAPI {
    let session: SessionStore
    func content(category: String, type: RuleContentType) async throws -> [RuleContentItem]   // GET /api/rules/content?category=&content_type=
    func glossary() async throws -> [GlossaryItem]                                             // GET /api/rules/glossary
    func search(keyword: String) async throws -> [RuleContentItem]                             // GET /api/rules/search?keyword=
}
```

- 四个球种的名称、说明、emoji 照搬 `index.vue` 模板：中式八球「中国流行的花色球比赛」⚫、九球追分「9球制，必须按顺序击球」🟡、斯诺克「22球制台球，得分制比赛」🔴、美式九球「赛局制九球，支持普胜、小金和大金」🟠；key 依次为 `chinese_eight`、`nine_ball`、`snooker`、`american_nine`。球种名称复用 `GameTypeCatalog`（如果它提供了按 key 取 label 的方法），不要再抄一份名称表。
- 三个接口都是访客可读，走 `session.optionalAuthRequest`。响应里的 `list` 按非可选数组解码。
- `search`：关键词先去掉首尾空白；为空时不请求，直接返回空数组。

**规则百科首页 `RulesIndexView`**（对应 `index.vue`，导航标题「规则百科」）：

- 顶部搜索框（占位「搜索规则、术语...」，键盘提交触发搜索）。
- 没有在搜索时：标题「选择球种」+ 四张球种卡（emoji、名称、说明、右箭头）→ `.ruleDetail(category:)`；标题「更多」+ 术语词典入口（📚「术语词典」「台球专业术语大全」）→ `.glossary`。
- 搜索时：标题行「搜索结果」+ 右侧「清除」；加载中、失败（用 `LoadFailure.feedback(resource: "规则搜索")`，按钮重试/返回）、结果列表、空结果「未找到相关内容」。
- **与小程序的一处有意差异**：后端搜索接口只返回 `id / title / content / sort_order`，不返回球种和内容类型，小程序的结果行因此显示不出分类、点击也无法跳到正确的球种页。原生端的结果行改为：显示标题，点击后在行内展开 / 收起正文（同一时间只展开一条），不做跳转。
- 搜索状态放进 `@MainActor final class RulesSearchModel: ObservableObject`（`keyword`、`phase: idle / loading / loaded([RuleContentItem]) / empty / failed(LoadFailure)`、`search()`、`clear()`），只采纳最新一次搜索的结果，`clear()` 后到达的结果丢弃。

**规则详情 `RuleDetailView`**（对应 `detail.vue`）：

- 导航标题为「{球种名} - 规则」（球种名查不到时用「规则」）。
- 顶部球种标题（emoji + 名称），下面三个分段 Tab：规则 / 犯规 / 术语。
- 切换 Tab：收起已展开的条目并读取对应内容（请求里的 category 用 `RuleCategory.contentCategory`）。每个 Tab 的内容读过一次后在本页面内保留，切回来不重复请求。
- 内容列表：每条一张卡——序号、标题、展开箭头；点击展开 / 收起正文（同一时间只展开一条）。正文保留换行。
- 状态：加载中、失败（`LoadFailure.feedback(resource: "规则内容")`，重试 / 返回）、空「暂无内容」。
- 实现建议：每个 Tab 一个 `ResourceLoader<[RuleContentItem]>`（`access: .anyone`），放进一个 `RuleDetailModel`。

**术语词典 `GlossaryView`**（对应 `glossary.vue`，导航标题「术语词典」）：

- 顶部搜索框（占位「搜索术语...」），本地过滤：标题或内容包含关键词（不区分大小写、去首尾空白）。过滤规则抽成纯函数，便于测试。
- 列表：术语名、解释、球种标签（`category` 非空时显示 `RuleCategory.label`）。
- 状态：加载中、失败（`resource: "术语词典"`）、空「暂无术语数据」、过滤后无结果「未找到匹配术语」。

三个页面访客都能用（`LoadableContent` 的 `requiresLogin: false`），背景 `palette.page`，卡片用 `.appCard()`。

### 3. 测试

- `RulesAPITests.swift`：三个接口的路径与查询参数（`content` 的 `category` / `content_type`、`search` 的 `keyword` 含中文时正确编码）；访客请求不带 Authorization；`success: false` 抛错；空关键词不发请求。
- `RulesModelTests.swift`：`RuleCategory` 的 label / emoji / `contentCategory`（`american_nine` → `nine_ball`，其余不变）；术语过滤规则；`RulesSearchModel`（成功、空结果、失败带分类、只采纳最新搜索、`clear()` 后迟到结果丢弃）；`RuleDetailModel`（切 Tab 读取对应类型、读过的 Tab 不重复请求、美式九球请求的 category 是 `nine_ball`）。
- `RulesSnapshotTests.swift`（`renderSnapshot`，亮暗各一张，页面放在带 `SessionStore`（访客）、`AppNavigator`、`ToastCenter` 环境的 `NavigationStack` 里，用 `MockHTTPServer` 提供数据并等数据读完再渲染）：
  1. `rules-index`：首页默认态；
  2. `rules-search`：有三条搜索结果、第一条展开；
  3. `rule-detail`：斯诺克「规则」Tab，五条内容、第二条展开；
  4. `rule-detail-error`：内容接口返回 503；
  5. `glossary`：六条术语。

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

要求：Debug 与 Release 构建成功；全部测试通过。测试失败就修到通过，不允许靠删除、跳过或弱化测试来通过。完成后确认 `git status` 里没有 `iOS/build/` 之外的意外文件。

## 交付

完成后用中文给出：新增/修改的文件清单、测试总数与结果（贴出 `Executed N tests` 那一行）、与本规格或小程序界面不一致的地方及原因。没做到的写明没做到。
