# R7.1：欢迎页、登录页对齐小程序

原生 iOS 只提供手机号验证码登录（平台差异：不接微信、华为登录）。本轮把首次启动的欢迎页迁过来，并让现有登录页 `iOS/iOS/LoginView.swift` 在界面、协议勾选、提示文案上与小程序的手机号登录模式对齐。认证链路（`SessionStore.sendSMS` / `login`、Keychain、刷新）**不改**——它有既有规格 `openspec/specs/native-ios-auth-integration/spec.md` 和一整套测试保护。

## 先补一处测试缺口（改完再做本轮任务）

`iOS/iOSAuthTests/MatchHistorySnapshotTests.swift` 的 `match-history-with-referee-banner` 快照里，执裁列表还在加载（只拍到顶部「继续执裁」条和一个转圈）——测试没有先把列表读完再渲染。按通用约束的做法：让 `MatchHistoryView` 能注入已读好数据的两个分栏列表模型（`init` 增加可选参数，默认仍自建），测试里先 `await` 两个分栏读完再渲染。改完导出快照确认列表行可见。

## 本轮要读的小程序源码

- `uniapp/pages/welcome/index.vue`（+ `index.scss`）：首次启动欢迎页
- `uniapp/pages/login/login.vue`（1291 行，+ 样式）：**只看手机号登录模式**（`resolveLoginMode` 在非微信端恒为 `phone`）的模板与逻辑；微信 / 华为 / 绑定手机号相关的分支不迁
- `uniapp/utils/entry-funnel.js` + `uniapp/tests/entry-funnel.test.mjs`：欢迎页按钮、协议状态、认证中反馈（「追分正在为您完成登录」/ 3 秒后「网络有些慢…」）、验证码 / 手机号校验等规则
- `uniapp/components/agreementConsentSheet.vue`：未勾选协议时点主按钮弹出的协议确认底部弹层
- 原生现有：`iOS/iOS/LoginView.swift`、`iOS/iOSAuthTests/` 里所有 `SessionStore*` 与登录相关测试、`openspec/specs/native-ios-auth-integration/spec.md`（登录页的既有需求，不能违反）

## 任务

1. **规则迁移**（`iOS/iOS/Features/Entry/EntryFunnelRules.swift`）：`entry-funnel.js` 中与手机号登录、欢迎页、协议、认证反馈相关的导出逐条迁移并搬运测试；只服务于微信 / 华为 / 绑定手机号的导出不迁，在交付说明里列出。
2. **欢迎页**（`iOS/iOS/Features/Entry/WelcomeView.swift`）：
   - 首次启动（`UserDefaults` 里没有「已看过欢迎页」标记，键名照搬 `welcome_page_viewed`）且当前没有登录会话时，以全屏覆盖层显示在主 Tab 之上；已登录或看过则不显示。实现放在 `ContentView` 里（`fullScreenCover` 或条件覆盖），测试宿主不受影响。
   - 内容照搬模板：品牌行（`BrandLogo` 图片资源已存在）、标语、三条示例数据卡（带「示例数据」角标）、主按钮（原生端为「手机号登录」——按 `resolveWelcomeActions` 在非微信端的结果取文案）、「先逛逛」类的次要按钮（文案取规则结果）、底部协议勾选与两个协议链接（跳应用内协议页：如果路由里已有 `.userAgreement` / `.privacyPolicy` 就用；没有就继续用现有外链，并在交付说明里注明）。
   - 主按钮：未勾选协议 → 弹出协议确认底部弹层（对应 `agreementConsentSheet.vue`，「同意并继续」后勾选并继续）；已勾选 → 关闭欢迎页并打开登录页（`navigator.openLogin()`）。次要按钮 → 关闭欢迎页留在首页。两者都写入「已看过」标记。协议勾选状态的保存与恢复按 `resolveEntryFunnelAgreementState` / `shouldClearEntryFunnelAgreementSession` 的规则（键名照搬）。
3. **登录页对齐**（修改 `iOS/iOS/LoginView.swift` 的界面层，不改它调用 `SessionStore` 的方式）：逐项对照 `login.vue` 的手机号模式——标题与副标题、输入框占位与格式校验提示、获取验证码按钮的倒计时文案、协议勾选（未勾选时点登录弹出同一个协议确认弹层，而不是现在的做法——以小程序为准）、认证中反馈条（规则来自任务 1）、错误提示。现有 `LoginView` 的行为测试必须继续通过；如果某个既有断言与小程序文案冲突，以小程序为准修改断言，并在交付说明里逐条列出改了哪些断言、为什么。

## 测试

搬运 JS 用例；欢迎页的显示条件（首次 / 已看过 / 已登录）；主按钮在协议未勾选与已勾选时的行为；次要按钮；协议状态持久化规则；登录页的协议弹层、认证中反馈的 3 秒切换（可注入时钟）。快照：`welcome`、`welcome-agreement-sheet`、`login`、`login-authenticating`、`login-error`。

## 并行开发提醒

其他开发线正在修改 `iOS/iOS/Features/**`（除 `Features/Entry/`）、`iOS/iOS/SessionStore.swift` 及其测试，本轮不要碰这些文件。`ContentView.swift` 只加欢迎页的显示逻辑。
