# RB3a：赛讯的数据模型、接口与规则（纯逻辑，不含界面）

你在一个 git worktree 的根目录里工作（当前目录就是仓库根）。总目标是把微信小程序（`uniapp/`）几乎 1:1 迁移到原生 SwiftUI（`iOS/`）。本轮为「赛讯」Tab（`uniapp/pages/tournament/index.vue`）和赛事详情（`uniapp/subPages/tournament/detail.vue`）迁移**数据模型、接口门面和业务规则**，并把小程序的 Node 测试用例迁成 XCTest。**本轮不写任何 SwiftUI 界面。**

先读：`AGENTS.md`、`iOS/AGENTS.md`；`iOS/iOS/` 下全部 Swift 文件（重点 `Shared/`、`Features/Rules/RulesAPI.swift`、`Features/Profile/UserHomepageRules.swift`——规则迁移的现成范例）；`iOS/iOSAuthTests/` 的 `TestSupport.swift`、`MockHTTPServer.swift`、`RulesAPITests.swift`、`UserHomepageRulesTests.swift`。
小程序侧：`uniapp/utils/saixun.js`（807 行，全部要迁）、`uniapp/utils/saixun-filter.js`、`uniapp/utils/event-news-response.js`、`uniapp/api/event-news.js`；测试 `uniapp/tests/saixun.test.mjs`、`saixun-filter.test.mjs`、`event-news-response.test.mjs`。页面 `uniapp/pages/tournament/index.vue`、`uniapp/subPages/tournament/detail.vue` 只用来了解这些规则怎么被调用。
接口契约（`backend/chasing_points.api`）：`EventNewsInfo`、`EventNewsMatchInfo`、`TournamentInfo`、`GetEventNewsListReq/Resp`、`GetEventNewsViewReq/Resp`（`GET /api/event-news/list`、`GET /api/event-news/view`，访客可读）。

## 硬性约束（违反任何一条都算失败）

1. 只允许新增/修改 `iOS/iOS/**` 与 `iOS/iOSAuthTests/**`。不得修改 `iOS/iOS.xcodeproj/**`、`uniapp/**`、`backend/**`、`openspec/**` 及其他目录。
2. 不执行任何 git 写操作，只允许 `git status`、`git diff` 等只读命令。
3. 不联网、不 ssh、不 curl；不读取任何 `.env`、密钥、Keychain、真实会话；不访问当前目录之外的路径。
4. 不引入第三方依赖。最低部署目标 iOS 17.0。
5. 现有全部测试必须继续通过。本轮**只新增文件**（放在 `iOS/iOS/Features/News/` 与 `iOS/iOSAuthTests/`），不修改任何现有文件。
6. 文案、翻译表与小程序源码**逐字一致**。`saixun.js` 里的各种对照表（赛事名、轮次、球员名、国家名、国旗等）必须**完整**迁移，一条都不能少、不能改、不能凭印象补。迁完后自己数一遍：每张表在 JS 里的条目数与 Swift 里的条目数必须相等，并把每张表的条目数写进交付说明。
7. 不创建 Markdown / 文档文件。不写 SwiftUI 视图。

## 工程须知

- App target 默认 MainActor 隔离、Swift 5 语言模式；测试类标 `@MainActor`。
- 解码统一 `convertFromSnakeCase`。访客可读接口用 `session.optionalAuthRequest`，带参数的 GET 用 `APIQuery.path`（nil 值会被跳过）。
- 需要「当前时间」的规则一律把 `now: Date` 作为参数传入。JS 里按本地时区取年月日的地方，Swift 里用可注入的 `Calendar`（默认 `Asia/Shanghai` 时区的公历），测试传固定日历，保证结果不随运行机器的时区变化。
- JS 里的正则、`trim`、大小写折叠、全角 / 半角处理要逐条对应；拿不准 JS 行为时，以 `uniapp/tests/` 里的期望值为准并写测试确认。
- 模拟器请用 **iPhone 15 Pro Max**（UDID `BE3E9E49-0427-45A2-A5DD-905C033EC260`）。xcodebuild 一律加 `-derivedDataPath build/dd-b`（相对 `iOS/` 目录，已被 git 忽略）。
- bash 工具跑 xcodebuild 时超时设为 600000 毫秒，输出必须用 grep 过滤。

## 任务

### 1. 模型与接口（`iOS/iOS/Features/News/EventNewsModels.swift`、`EventNewsAPI.swift`）

- `EventNewsInfo`、`EventNewsMatchInfo`、`TournamentInfo`：字段与 `.api` 完全对齐（`Decodable, Equatable`，有 `id` 的加 `Identifiable`）。`.api` 里标了 `optional` 的字段用可选类型；数组字段按非可选解码。
- `EventNewsView`：`eventNews: EventNewsInfo?`、`tournament: TournamentInfo?`、`matches: [EventNewsMatchInfo]`；`pickEventNewsViewPayload` 的兼容逻辑（响应可能把内容包在 `data` 里）——注意 `APIClient` 已经会自动解开顶层 `data`，确认后在交付说明里写明这条规则在原生端是否还需要单独实现。
- `EventNewsFilter`（年份 / 起止日期 / 球种 / 状态 / 城市）与 `EventNewsAPI`：

```swift
@MainActor
struct EventNewsAPI {
    let session: SessionStore
    func list(filter: EventNewsFilter, page: Int, pageSize: Int) async throws -> (items: [EventNewsInfo], total: Int)
    func view(eventId: Int64) async throws -> EventNewsView
}
```

  查询参数的生成规则逐条对应 `buildEventNewsListParams`（哪些字段为空时不带、默认页大小等）。

### 2. 筛选规则（`iOS/iOS/Features/News/SaiXunFilter.swift`）

迁移 `saixun-filter.js` 的全部导出：`buildYearDateRange`、`buildCurrentYearFilter`、`buildEventNewsListParams`、`buildYearOptions`、`buildYearBoundaryDate`。

### 3. 赛讯规则（`iOS/iOS/Features/News/SaiXunRules.swift`，对照表多可以拆成 `SaiXunLocalization.swift`）

迁移 `saixun.js` 的**全部导出**以及它们依赖的私有函数和对照表：

`DEFAULT_EVENT_COVER`、`OFFICIAL_DEFAULT_EVENT_COVER`、`DEFAULT_PLAYER_AVATAR`、`OFFICIAL_PLAYER_AVATAR_PLACEHOLDER`、`localizeTournamentTitle`、`standardizeRoundText`、`localizeMatchPlayerName`、`formatEventDateRange`、`formatEventTimeRange`、`buildEventLocationText`、`buildEventRegionText`、`formatEventCountdownText`、`normalizeSaiXunCard`、`buildSaiXunListSections`、`buildSaiXunHeroStats`、`formatBestOfText`、`buildSaiXunDetailRounds`、`SAIXUN_SCHEDULE_PAGE_SIZE`、`buildSaiXunScheduleTabs`、`resolveSaiXunScheduleTabKey`、`buildSaiXunScheduleDays`、`resolveSaiXunChampion`、`localizeCountryName`、`sanitizeFlagEmoji`、`resolvePlayerFlag`。

要求：

- 每个 JS 返回的对象对应一个 Swift 值类型（`Equatable`），字段名用 camelCase，含义与 JS 一一对应；用一个 `enum SaiXunRules` 承载静态函数。
- 封面 / 头像路径：CDN 路径经 `CDNAsset.url(for:)` 转成 `URL?`；小程序本地路径（如 `/static/images/wst-player-placeholder.png`、默认用户头像）在模型里保留为「使用本地占位图」的标记（例如一个枚举 `ImageSource { case remote(URL), officialPlayerPlaceholder, defaultAvatar }`），不要拼出一个不存在的 URL。
- 时间：服务端时间文本的解析方式逐条对应 JS（哪种格式按北京时间、哪种按 ISO 8601）；可复用现有的 `ServerTime`。

### 4. 测试

- `SaiXunRulesTests.swift`：把 `saixun.test.mjs` 的 38 条用例逐条搬过来（一条 JS 用例对应一个 `func test...`，名字能看出对应关系）。
- `SaiXunFilterTests.swift`：搬 `saixun-filter.test.mjs` 的 6 条。
- `EventNewsAPITests.swift`：`event-news-response.test.mjs` 的 2 条（按任务 1 的结论适配）；列表接口的查询参数（各筛选项为空 / 非空）、访客不带 Authorization、`success: false` 抛错；详情接口的 `event_id` 参数、`matches` 为空数组时正常。
- 对照表完整性：为每张对照表加一条断言条目数的测试（数字取自你数出来的 JS 条目数），防止以后被改漏。
- 夹具放 `iOS/iOSAuthTests/EventNewsFixtures.swift`，字段名用服务端的 snake_case。

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

要求：Debug 与 Release 构建成功；全部测试通过。测试失败就修到通过，不允许靠删除、跳过或弱化测试来通过。如果发现 JS 测试的期望与你的实现不一致，以 JS 源码的实际行为为准，不要改期望值去迁就实现。

## 交付

完成后用中文给出：新增的文件清单；每个 JS 测试文件搬了多少条、没搬的及原因；每张对照表在 JS 与 Swift 里的条目数；测试总数与结果（贴出 `Executed N tests` 那一行）；与本规格不一致的地方及原因。没做到的写明没做到。
