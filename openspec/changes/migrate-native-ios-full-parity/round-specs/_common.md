
---

# 通用约束与工程须知（每一轮都适用）

你在一个 git worktree 的根目录里工作（当前目录就是仓库根）。总目标是把微信小程序（`uniapp/`）几乎 1:1 迁移到原生 SwiftUI（`iOS/`）。动手前先读：`AGENTS.md`、`iOS/AGENTS.md`、`DESIGN.md`，以及 `iOS/iOS/` 下的全部 Swift 文件——已经迁好的模块就是范例：

- 访客可读页面：`Features/Rules/`（接口门面、页面模型、页面、失败分类提示）
- 需要登录的聚合页面：`Features/Profile/`（`ProfileAPI`、`MyPageModel`、`UserHomepageRules`）
- 分页列表：`PagedHistoryList.swift` + `MatchHistoryViews.swift`
- 测试与快照：`iOSAuthTests/TestSupport.swift`、`MockHTTPServer.swift`、`SnapshotSupport.swift`、`RulesSnapshotTests.swift`、`MyPageModelTests.swift`

## 硬性约束（违反任何一条都算失败）

1. 只允许新增/修改 `iOS/iOS/**` 与 `iOS/iOSAuthTests/**`。不得修改 `iOS/iOS.xcodeproj/**`、`uniapp/**`、`backend/**`、`openspec/**` 及其他目录。
2. 不执行任何 git 写操作（commit / add / checkout / reset / stash 等），只允许 `git status`、`git diff` 等只读命令。
3. 不联网、不 ssh、不 curl；不读取任何 `.env`、密钥、Keychain、真实会话；不访问当前目录之外的路径。所有验证只用 `MockHTTPServer`，绝不对真实后端发请求。
4. 不引入第三方依赖，只用系统框架。最低部署目标 iOS 17.0。
5. 现有全部测试必须继续通过。除本轮任务点名的文件外不改现有代码，不顺手重构。`AppRoute.swift` 只做任务点名的改动（新增 case、把点名的路由指向新页面），不动其他行。
6. 颜色一律走 `AppTheme` / `AppPalette`（任务明确允许的例外除外）。用户可见文案与小程序源码**逐字一致**（包括标点、空格）。注释用简体中文，只写「为什么」。
7. 不创建 Markdown / 文档文件。

## 工程须知

- App target 默认 MainActor 隔离、Swift 5 语言模式；测试类标 `@MainActor`。
- JSON 解码统一 `convertFromSnakeCase`；模型字段以 `backend/chasing_points.api` 为准：标了 `optional` 的字段用可选类型，数组按非可选解码。拿不准某个字段在失败 / 空数据时返回什么，就去读 `backend/internal/logic/` 里对应的实现，不要猜。
- **类型与路由不得重名**：有多条开发线在并行。新建任何顶层类型（struct / enum / class）或 `AppRoute` case 之前，先 `grep -rnw "<名字>" iOS/iOS iOS/iOSAuthTests` 确认不存在；已存在就直接复用，不要再定义一份。与 `backend/chasing_points.api` 同名的模型（如 `CurrentMatchInfo`、`ChallengeInfo`、`NotificationInfo`、`NotificationListResponse`）全仓只能有一份，统一放在 `iOS/iOS/App/ActivityModels.swift` 或 `iOS/iOS/Shared/` 下；只取其中几个字段的窄解码类型必须起带用途的名字（例如 `OngoingMatchRef`），不能沿用 `.api` 里的名字。
- 需要登录的读取用 `session.authorizedRequest`，写请求用 `session.authorizedPost`（请求体 snake_case，`nil` 字段不会被编码）；访客可读接口用 `session.optionalAuthRequest`；带参数的 GET 用 `APIQuery.path`（nil 值跳过）。页面不自建 URLSession。
- 单资源读取用 `ResourceLoader`（按需传 `maxAge`），分页用 `PagedHistoryList`；失败提示用 `LoadFailure.feedback(resource:)`，界面状态用 `LoadableContent` / `StateMessageView`。
- 所有异步结果都要绑定发起时的 `session.identity`，并且只采纳最新一次请求：退出、切号或被新请求取代后到达的结果不得改动状态、不得弹提示。
- 业务规则写成不依赖界面的纯函数 / 值类型；需要「当前时间」的把 `now: Date` 作为参数传入。小程序 `uniapp/utils/*.js` 里有对应规则的，逐条迁移并把 `uniapp/tests/` 里对应的用例搬成 XCTest；JS 测试期望与你的实现不一致时，以 JS 源码的实际行为为准。
- 页面跳转只用 `NavigationLink(value: AppRoute.xxx)` 或 `navigator.open(.xxx)`；轻提示用 `ToastCenter`。
- 受合规开关控制的入口与小程序同步（`AppFlags`）。小程序里依赖微信专属能力、原生端没有等价能力的操作不显示。
- **数字显示不要带千分位**：SwiftUI 的 `Text("排位分 \(score)")` 会按系统区域把整数格式化成 `1,320`，小程序显示的是 `1320`。凡是把数字拼进文案，一律先转成字符串（`String(score)`）再拼，或用 `Text(verbatim:)`；规则函数返回的也应是已经拼好的 `String`。为含数字的文案写断言时用四位数以上的值。
- rpx 换算大致 `pt ≈ rpx / 2`，字号用 `.font(.system(size:weight:))`；布局以小程序的模板和 scss 为依据，控件用原生形态。
- 测试里用 `DispatchSemaphore` 做闸门时一律带超时（`gate.wait(timeout: .now() + 5)`），并确认每个会被闸门挡住的请求都会被放行；不要让请求卡到 `APIClient` 的 30 秒超时。自检时看一眼新增测试类的耗时，单个用例超过 2 秒要查明原因。
- 每个页面都要有快照测试（`renderSnapshot`，亮暗各一张），至少覆盖：有数据、空、失败三种状态。**快照里不要指望页面内的 `.task` 把网络请求跑完**（渲染窗口很快就销毁，请求会被取消）。固定做法：页面的 `init` 接受一个可选的页面模型（`init(session:, model: XxxModel? = nil)`，内部 `_model = StateObject(wrappedValue: model ?? XxxModel(session: session))`）；测试里先创建模型、`await` 它读完数据，再把模型传给页面渲染。不要为此去改 `SnapshotSupport.swift` 或 `MockHTTPServer.swift`，也不要写探针测试排查。
- 遇到与任务无关的疑难（工具行为、测试框架怪癖），最多尝试两种办法；不行就换一种不依赖它的实现继续完成任务，并在交付说明里写明，不要在同一个问题上反复排查。

## 自检（在 `iOS/` 目录执行；bash 工具的超时设为 600000 毫秒；输出必须用 grep 过滤，不要把完整日志读进上下文）

模拟器用 **__SIM_NAME__**（UDID `__SIM_UDID__`），其他模拟器被别的任务占用。xcodebuild 一律加 `-derivedDataPath __DD__`（相对 `iOS/` 目录，`build/` 已被 git 忽略）。

```bash
cd iOS
xcodebuild build -project iOS.xcodeproj -scheme iOS -configuration Debug -derivedDataPath __DD__ \
  -destination 'generic/platform=iOS Simulator' CODE_SIGNING_ALLOWED=NO 2>&1 | grep -E "error:|BUILD (SUCCEEDED|FAILED)" | head -60

xcodebuild test -project iOS.xcodeproj -scheme iOS -configuration Debug -derivedDataPath __DD__ \
  -destination "platform=iOS Simulator,id=__SIM_UDID__" \
  -parallel-testing-enabled NO 2>&1 | grep -E "error:|: error|failed|Executed [0-9]+ tests|TEST (SUCCEEDED|FAILED)" | tail -60

xcodebuild build -project iOS.xcodeproj -scheme iOS -configuration Release -derivedDataPath __DD__ \
  -destination 'generic/platform=iOS Simulator' CODE_SIGNING_ALLOWED=NO 2>&1 | grep -E "error:|BUILD (SUCCEEDED|FAILED)" | head -20
```

要求：Debug 与 Release 构建成功；全部测试通过。测试失败就修到通过，不允许靠删除、跳过或弱化测试来通过。

## 交付

完成后用中文给出：新增 / 修改的文件清单；JS 用例搬了多少、没搬的及原因；测试总数与结果（贴出 `Executed N tests` 那一行）；与任务或小程序不一致的地方及原因；你认为小程序或后端本身有问题的地方（只报告，不要改）。没做到的写明没做到，不要说成做到了。
