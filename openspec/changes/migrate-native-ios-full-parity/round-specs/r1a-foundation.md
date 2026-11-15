# R1a：原生 iOS 基础设施（主题 token / 配置 / 会话层扩展 / 通用读取模型 / CDN 与头像）

你在一个 git worktree 的根目录里工作（当前目录就是仓库根）。目标：为「把微信小程序（`uniapp/`）几乎 1:1 迁移到原生 SwiftUI（`iOS/`）」打地基。本轮**只做基础设施和测试**，不新增任何业务页面，不改动现有页面的界面。

先读：`AGENTS.md`、`iOS/AGENTS.md`、`DESIGN.md` 第 3 节（颜色 Token），以及 `iOS/iOS/` 下全部现有 Swift 文件和 `iOS/iOSAuthTests/` 下的 `TestSupport.swift`、`MockHTTPServer.swift`、`PagedHistoryListTests.swift`、`MatchReportModelTests.swift`（测试写法以它们为准）。

## 硬性约束（违反任何一条都算失败）

1. 只允许新增/修改 `iOS/iOS/**` 与 `iOS/iOSAuthTests/**` 下的文件。不得修改 `iOS/iOS.xcodeproj/**`（工程用文件系统同步组，新文件会自动进 target）、`uniapp/**`、`backend/**` 及其他任何目录。
2. 不执行任何 git 写操作（commit / add / checkout / reset / stash / branch 等都不行），只允许 `git status`、`git diff` 这类只读命令。
3. 不联网、不 ssh、不 curl 任何地址；不读取任何 `.env`、密钥、Keychain、真实会话；不访问当前目录之外的路径（Xcode 默认 DerivedData 除外）。
4. 不引入第三方依赖（不加 SPM 包、CocoaPods）。只用 SwiftUI / Foundation / 系统框架。最低部署目标 iOS 17.0。
5. 不改动现有公开行为：现有 98 个测试必须继续通过；不重命名、不移动现有文件；不顺手重构现有代码。
6. 代码注释与用户可见文案用简体中文，风格与现有文件一致（注释只写「为什么」，不写废话）。
7. 不创建任何 Markdown / 文档文件。

## 工程须知

- App target 开启了 `SWIFT_DEFAULT_ACTOR_ISOLATION = MainActor`、Swift 5 语言模式。测试类照现有写法标 `@MainActor`。
- JSON 解码统一 `convertFromSnakeCase`（`vip_level` → `vipLevel`，`user_id` → `userId`）。
- 所有需要登录的请求走 `SessionStore.authorizedRequest`；页面不自建 URLSession。
- 测试用 `MockHTTPServer`（`server.stub(...)`、`server.requests(path)`、`server.requestCount(path)`），`makeSessionStore(server, credentialStore:)`、`InMemoryCredentialStore`、`AuthFixtures`。测试串行执行。

## 任务

### 1. 主题 token（修改 `iOS/iOS/AppTheme.swift`）

按 `DESIGN.md` 3.1 的表补齐语义色，现有字段和取值不变：

- 新增 `extension Color { init(hex: UInt32, opacity: Double = 1) }`（`0xE0AE12` 这种写法），放在本文件。现有 `Color(red:green:blue:)` 写法不用改。
- `AppPalette` 新增字段：`textMuted`、`brandStrong`、`brandTint`、`success`、`info`、`gradientStart`、`gradientEnd`、`heroColors: [Color]`。
  - 亮色：textMuted `#9A8C67`，brandStrong `#C69200`，brandTint `rgba(224,174,18,0.14)`，success `#18B05B`，info `#3B82F6`，gradientStart `#D9A617`，gradientEnd `#BB8400`，heroColors `[#3C2B08, #8F6507, #C69200]`。
  - 暗色：textMuted `#9F926E`，brandStrong `#F0C542`，brandTint `rgba(224,174,18,0.20)`，success `#22C55E`，info `#60A5FA`，gradientStart `#E0AE12`，gradientEnd `#A97500`，heroColors `[#1E180D, #241D10]`。
- `AppTheme` 新增 `static let warning = Color(hex: 0xF97316)`。

### 2. 运行配置与合规开关

- 在 `iOS/iOS/APIClient.swift` 的 `AppConfiguration` 里新增（保持 `#if DEBUG` 结构）：
  - `wsBaseURL`：Debug `wss://api-zhuifen.kekemate.cn`，Release `wss://ws.zhuifen.cn`。
  - `cdnBaseURL`：两种配置都是 `https://cdn.zhuifen.cn`（放在 `#if` 之外）。
- 新建 `iOS/iOS/Shared/AppFlags.swift`，对应 `uniapp/utils/compliance-mode.js`：

```swift
/// 合规收口：与小程序 `utils/compliance-mode.js` 保持一致。
enum AppFlags {
    static let complianceMode = true
    static var canShowSocialPublishing: Bool { !complianceMode }
    static var canShowPaidMembershipEntry: Bool { !complianceMode }
    static var canShowUserTournamentActions: Bool { !complianceMode }
}
```

### 3. 会话层扩展（修改 `iOS/iOS/SessionStore.swift`，只新增方法，不改现有方法的行为）

```swift
/// 需要登录的写请求：请求体按 snake_case 编码后走 authorizedRequest。
func authorizedPost<Body: Encodable, Response: Decodable>(_ path: String, body: Body) async throws -> Response

/// 访客也能访问的接口：有凭据时走 authorizedRequest（带 token、同样的刷新与失效规则），
/// 没有凭据时不带 Authorization 直接请求。
func optionalAuthRequest<Response: Decodable>(_ path: String, method: HTTPMethod = .get, body: Data? = nil) async throws -> Response
```

- `authorizedPost` 用 `JSONEncoder` + `.convertToSnakeCase` 编码后调用现有 `authorizedRequest(path, method: .post, body:)`。
- `optionalAuthRequest`：`credentials != nil` → 直接返回 `authorizedRequest` 的结果；否则记下当前 `authGeneration`，用 `api.request(path, method:, body:, bearerToken: nil)` 请求，返回前若 `authGeneration` 已变化（请求期间完成了登录）则抛 `SessionError.changed`（访客结果不带个性化字段，不能给登录后的页面用）。

### 4. 通用读取模型（新建 `iOS/iOS/Shared/ResourceLoader.swift`）

把 `MatchReportModel` 里「按身份隔离的一次性读取」抽成可复用的模型，后面几十个页面都会用它。对外接口必须是：

```swift
/// 单个资源的读取状态，结果绑定发起时的身份。
@MainActor
final class ResourceLoader<Value>: ObservableObject {
    enum Access {
        case signedIn   // 必须登录：没有身份时不请求
        case anyone     // 访客可读：没有身份也请求
    }
    enum Phase {
        case idle
        case loading
        case loaded(Value)
        case failed(String)
    }

    @Published private(set) var phase: Phase = .idle
    @Published private(set) var isRefreshing = false
    /// 已有数据时刷新失败的提示；数据保持不变。
    @Published private(set) var refreshError: String?

    var value: Value? { get }          // 仅 .loaded 时有值
    var isLoading: Bool { get }        // phase 为 .loading
    var failureMessage: String? { get } // phase 为 .failed 时的文案

    init(session: SessionStore, access: Access = .signedIn, load: @escaping () async throws -> Value)

    func syncIdentity()
    func loadIfNeeded() async
    func refresh() async
    func retry() async
    func invalidate()
}
```

行为（每条都要有测试）：

- `syncIdentity()`：`session.identity` 与上次记录的不同时，令在途请求全部作废（请求令牌 +1），`phase = .idle`，`isRefreshing = false`，`refreshError = nil`，清除过期标记。相同则什么都不做。
- `loadIfNeeded()`：先 `syncIdentity()`。`access == .signedIn` 且当前没有身份 → 直接返回、不请求。`phase == .idle` → 首次加载（`phase = .loading`）。`.loaded` 且被 `invalidate()` 标记过期 → 以刷新方式重读（保留旧数据显示）。`.loading` 中、`.failed`、未过期的 `.loaded` → 不请求（失败后由用户点重试）。
- `refresh()`：先 `syncIdentity()`，身份规则同上。已有数据（`.loaded`）→ `isRefreshing = true`、保留数据；成功替换数据并清 `refreshError`；失败 → 数据不变、`refreshError = 文案`。没有数据 → 等同首次加载。正在刷新时再调用不重复请求。
- `retry()`：`.failed` → 重新首次加载；其余情况等同 `refresh()`。
- `invalidate()`：仅做过期标记（写操作成功后由调用方调用），不发请求。
- 只采纳最新一次请求的结果（请求令牌），并且只在 `session.identity` 仍等于发起时的身份时采纳；否则丢弃，不改任何状态。
- 取消不是失败：错误是 `APIError` 且 `kind == .canceled`，或者是 `CancellationError` 时——首次加载回到 `.idle`（下次出现再读），刷新只把 `isRefreshing` 置回 `false`，不写 `refreshError`。
- 其他错误：首次加载 → `.failed(error.localizedDescription)`。
- 不做跨页面缓存，不加 TTL。

### 5. 查询串工具（新建 `iOS/iOS/Shared/APIQuery.swift`）

```swift
enum APIQuery {
    /// 生成带查询串的路径：nil 值跳过，键按字典序排列，值做百分号编码。
    static func path(_ path: String, _ items: [String: CustomStringConvertible?]) -> String
}
```

- 没有任何非 nil 项时原样返回 `path`（不带 `?`）。
- 值里的空格、中文、`&`、`=`、`+`、`#`、`?` 都必须被编码（注意 `URLComponents` 不会编码 `+`，需要自己处理成 `%2B`）。
- 例：`APIQuery.path("/api/match/list", ["page": 1, "page_size": 20])` → `/api/match/list?page=1&page_size=20`。
- 不要去改现有 `MatchHistoryAPI` 的拼接写法。

### 6. CDN 资源与默认头像

- 新建 `iOS/iOS/Shared/CDNAsset.swift`，对应 `uniapp/utils/cdn-asset.js`：

```swift
enum CDNAsset {
    /// 后端返回的 `/static/images/...` 路径是否属于 CDN 资源（规则与小程序一致）。
    static func isCDNPath(_ path: String) -> Bool
    /// CDN 路径 → CDN 地址；http(s) 绝对地址原样返回；其他（空串、小程序本地路径）返回 nil。
    static func url(for path: String) -> URL?
}
```

  CDN 路径规则与 JS 里的正则完全一致：`/static/images/(achievements|ranks|avatar-frames|gear)/<不含斜杠的文件名>`，或 `/static/images/default-event-cover.png`、`/static/images/wst-default-event-cover.png`。

- 默认头像：把 `uniapp/static/images/default-avatars/ball-01.png` … `ball-15.png` 复制进 `iOS/iOS/Assets.xcassets/`，每张一个 imageset，命名 `DefaultAvatar01` … `DefaultAvatar15`（`Contents.json` 参照现有 imageset 的格式，单张 universal 图即可，不标 scale）。只复制、不要改动 `uniapp/` 里的原图。
- 新建 `iOS/iOS/Shared/MemberIdentity.swift`，对应 `uniapp/utils/user-profile.js` 的默认头像规则和 `uniapp/utils/member-identity.js`：

```swift
enum DefaultAvatar {
    /// userId 为正整数时取 ((userId - 1) % 15) + 1 号球，否则 1 号。返回 asset 名，如 "DefaultAvatar07"。
    static func assetName(userId: Int64) -> String
}

struct MemberIdentity: Equatable {
    let showBadge: Bool
    let vipLevel: Int        // 0 表示非会员
    let tier: String         // "standard" / "advanced" / "premium" / ""
    let tierLabel: String    // 标准 / 进阶 / 尊享 / ""
    let badgeText: String    // "VIP3" / ""
    let frameURL: URL?       // 段位头像框（CDN），非会员或没有段位时为 nil

    static func resolve(vipLevel: Int, rankLevel: Int) -> MemberIdentity
}

struct MemberAvatarMetrics: Equatable {
    let avatarSize: CGFloat
    let boxSize: CGFloat
    let frameSize: CGFloat
    let badgeFontSize: CGFloat
    let badgeHeight: CGFloat
    let badgePaddingX: CGFloat

    /// size 为头像直径（point）。比例与小程序一致；标识字号下限 9、上限 13（point）。
    static func resolve(size: CGFloat, reserveFrame: Bool = false, fitFrame: Bool = false, hasFrame: Bool = false) -> MemberAvatarMetrics
}
```

  规则逐条照搬 `member-identity.js`（VIP 等级归一化到 0–5、三档划分、6 个段位头像框路径经 `CDNAsset.url(for:)`、`FRAME_SIZE_RATIO = 156/96`、`FRAME_BOX_RATIO = 124/96`、fitFrame / reserveFrame 三种占位方式）。唯一差别：尺寸单位是 point，标识字号范围改为 9…13，各尺寸用 `rounded()`。

- 本轮**不写**头像的 SwiftUI 组件（下一轮做），只做纯逻辑。

## 必须新增的测试（放在 `iOS/iOSAuthTests/`，每个主题一个文件）

- `ResourceLoaderTests.swift`：覆盖任务 4 的每一条行为，至少包括：
  1. 首次加载成功；重复 `loadIfNeeded()` 只发 1 次请求；
  2. `.signedIn` 没有身份时不请求、保持 `.idle`；
  3. `.anyone` 访客可读，且请求不带 Authorization（配合 `optionalAuthRequest`）；
  4. 读取中退出登录，迟到结果被丢弃、状态回到 `.idle`（迟到响应的写法参照 `PagedHistoryListTests`）；
  5. 首次加载业务失败（HTTP 200 + `success: false`）→ `.failed`，不是 `.loaded`；`retry()` 后成功；
  6. 已有数据时刷新失败：数据不变、`refreshError` 有值；再次刷新成功后 `refreshError` 清空；
  7. `invalidate()` 后 `loadIfNeeded()` 会重新请求，期间旧数据仍可见；未 `invalidate()` 则不请求；
  8. 首次加载被取消（`.failure(.cancelled)`）回到 `.idle`，不显示错误；
  9. 只采纳最新请求：先后两次请求、先发的后返回时，结果以后发的为准。
- `SessionStoreRequestTests.swift`：
  1. `authorizedPost` 请求体是 snake_case（如 `matchId` → `match_id`）、带 Bearer；
  2. `optionalAuthRequest` 访客请求不带 Authorization；
  3. 有凭据时带 Bearer，并且 401 后刷新令牌、只重放一次；
  4. 访客请求进行中完成登录 → 抛错，不返回访客结果。
- `APIQueryTests.swift`：nil 跳过、无参数不带 `?`、字典序、中文/空格/`&`/`=`/`+`/`#` 的编码（断言编码后的字符串，并断言用 `URLComponents` 解析回来的值与原值相等）。
- `CDNAssetTests.swift`：把 `uniapp/tests/cdn-asset.test.mjs` 的用例逐条搬过来，另加 http(s) 原样返回、空串与本地路径返回 nil。
- `MemberIdentityTests.swift`：把 `uniapp/tests/member-identity.test.mjs` 里针对 `resolveMemberIdentity`、`normalizeVipLevel`、`resolveMemberAvatarMetrics` 的用例搬过来（尺寸断言按 point 版本的规则重新计算，不要照抄 rpx 数字的字号上下限）；`DefaultAvatar.assetName` 覆盖 1、15、16、0、负数。
- 另加一个断言：`UIImage(named: "DefaultAvatar01")` 到 `"DefaultAvatar15"` 在测试宿主里都能取到（`Bundle.main`），确保资源真的进了 App。

## 自检（全部在 `iOS/` 目录执行；bash 工具的超时请设为 600000 毫秒，输出一定要过滤，不要把完整构建日志读进上下文）

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

要求：Debug 与 Release 构建成功；全部测试通过（原有 98 个 + 新增）；本次新增代码不产生新的编译警告。测试失败就修到通过为止，不允许删除、跳过或弱化测试来「通过」。

## 交付

完成后用中文给出：新增/修改的文件清单、测试总数与结果（贴出 `Executed N tests` 那一行）、任何与本规格不一致的地方及原因。没做到的写明没做到，不要说成做到了。
