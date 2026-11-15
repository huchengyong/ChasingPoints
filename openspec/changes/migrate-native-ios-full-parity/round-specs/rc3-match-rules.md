# RC3：对局核心规则迁移（纯逻辑，不含界面）

为后面的扫码开局、实时计分、赛果、复盘做准备：把小程序 `uniapp/utils/` 里与对局相关、不依赖页面的规则逐个迁成 Swift，并搬运对应的 Node 测试。**只新增文件**（`iOS/iOS/Features/Match/Rules/` 与 `iOS/iOSAuthTests/`），不修改任何现有文件，不写 SwiftUI 界面，不写接口门面。

## 要迁移的模块（每个 JS 文件对应一个 Swift 文件，放在 `iOS/iOS/Features/Match/Rules/`）

| JS（`uniapp/utils/`） | 测试（`uniapp/tests/`） | Swift 文件 |
| --- | --- | --- |
| `match-role-view.js` | `match-role-view.test.mjs` | `MatchRoleView.swift` |
| `match-scoring-target.js` | `match-scoring-target.test.mjs` | `MatchScoringTarget.swift` |
| `pool-match-format.js` | `pool-match-format.test.mjs` | `PoolMatchFormat.swift` |
| `snooker-frame.js` | `snooker-frame.test.mjs` | `SnookerFrame.swift` |
| `snooker-match-format.js` | `snooker-match-format.test.mjs` | `SnookerMatchFormat.swift` |
| `snooker-v2.js` | `snooker-v2.test.mjs` | `SnookerV2.swift` |
| `chasing-score.js` | `chasing-score.test.mjs` | `ChasingScore.swift` |
| `finish-countdown.js` | （没有独立测试文件；在 `uniapp/tests/` 里搜 `finish-countdown` 找到引用它的测试并搬运相关用例） | `FinishCountdown.swift` |
| `start-match.js` | `start-match.test.mjs` | `StartMatch.swift` |
| `match-navigation.js` | `match-navigation.test.mjs` | `MatchNavigation.swift` |
| `match-action.js` | `match-action.test.mjs` | `MatchAction.swift` |
| `entry-link.js` | `entry-link.test.mjs` | `EntryLink.swift` |
| `match-scan.js` | `match-scan.test.mjs` | `MatchScan.swift` |
| `match-replay-entry.js` | `match-replay-entry.test.mjs` | `MatchReplayEntry.swift` |
| `member-replay-view.js` | `member-replay-view.test.mjs` | `MemberReplayView.swift` |

## 迁移要求

1. 每个 JS 导出对应一个 Swift 静态函数 / 常量（用 `enum <模块名>` 承载），语义、边界、文案逐条一致。JS 返回对象的，对应一个 `Equatable` 值类型，字段 camelCase。
2. JS 函数的入参是「接口返回的对象」时，Swift 侧定义**只含该规则用到的字段**的窄输入类型（`Decodable, Equatable`，字段名、可选性以 `backend/chasing_points.api` 为准——先按 JS 里读取的字段名去 `.api` 里找到对应类型）。不同模块读取同一个接口对象时可以共用一个输入类型，放在 `MatchRuleInputs.swift`。不要一次性把 `.api` 里几十个字段的大类型全搬过来。
3. `match-navigation.js`、`entry-link.js` 里生成的是小程序页面路径（`/subPages/...?match_id=...`）。Swift 侧返回一个表达「去哪」的枚举（例如 `enum MatchDestination { case playing(matchId: Int64), result(matchId: Int64, ...), spectate(matchId: Int64), ... }`，case 与参数以 JS 实际生成的路径和查询参数为准），**不要**引用 `AppRoute`（它正被其他开发线修改）。外部链接 / 小程序码 / 二维码正文的解析规则（哪些文本算对局邀请、裁判码、观赛链接，token 怎么取）逐条照搬，这是扫码开局的安全边界：不认识的内容一律按「无法识别」处理，不能放宽。
4. `match-scan.js` 里调用相机、相册、权限的部分（依赖 `uni` 和 `permission-helper.js`）不迁；只迁「拿到扫码文本之后」的解析与分流规则，以及预览接口返回值的归一化。需要调用接口的地方（预览邀请、解析邀请码）定义成注入的 `async` 闭包参数，本轮不实现真实请求。
5. `match-action.js` 里如果只是对 WebSocket / HTTP 的转发封装，迁移其中的纯规则部分（动作名、负载的构造与校验）；与 `uni` 或 store 绑定的部分不迁，在交付说明里列出。
6. 依赖 `./game-types.js` 的地方用现有 `GameTypeCatalog`；依赖 `./cdn-asset.js` 的地方用 `CDNAsset`。遇到依赖本表之外模块（如 `format.js`、`honor-wall.js`、`user-homepage.js`）的导出，本轮跳过该导出并在交付说明里列出。
7. 需要「当前时间」的规则把 `now: Date` 作为参数传入（`finish-countdown.js` 尤其注意：倒计时的截止时间解析、剩余时间的取整方式、到期判定与 JS 完全一致）。
8. 凭证（invite token、裁判码、ws ticket）不写日志、不放进任何会被持久化或打印的结构的 `description`。

## 测试

每个 JS 测试文件对应一个 XCTest 文件，用例逐条搬运（一条 JS 用例对应一个 `func test...`）。测试里读 `.vue` 源码做断言的用例不搬，在交付说明里列出。

## 并行开发提醒

其他开发线正在修改 `iOS/iOS/Features/**`（除 `Features/Match/Rules/`）、`iOS/iOS/App/**`、`iOS/iOS/Shared/**`、`iOS/iOS/SessionStore.swift` 及其测试。本轮只新增 `iOS/iOS/Features/Match/Rules/**` 和对应测试文件，其他一概不碰。本轮没有界面，不需要快照测试。
