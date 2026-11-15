# RC4b：扫码开局的三处修正（小补丁轮，只做下面三件事）

先读 `iOS/iOS/Features/Match/QRScannerView.swift`、`QuickScanFlow.swift`、`ScanConfirmView.swift`、`ScanConfirmModel.swift`、`iOS/iOS/App/MainTabView.swift`，以及它们的测试。

## 1. 扫码页状态布局是坏的

`scanner-unsupported` 快照里，相机图标、「粘贴二维码内容」输入框、标题「当前端暂不支持扫码」、说明文字、「确认」「关闭」两个按钮全部叠在一起。`scanner-permission-denied` 也检查一遍。

要求：状态内容用一个普通的纵向 `VStack(spacing:)` 自上而下排列——图标 → 标题 → 说明 →（仅 Debug）粘贴输入框 + 「确认」→ 主操作按钮（「去设置」或「关闭」）。不要用 `ZStack` / `overlay` / 负的 `padding` / `offset` 来摆这些元素。输入框要有可见的背景、边框和内边距，占位文字不与任何东西重叠；两个按钮上下间距不小于 12。改完后**自己用 Read 工具打开导出的快照 PNG 看一眼**确认不再重叠（导出方法：运行测试时在 xcodebuild 命令前加 `TEST_RUNNER_SNAPSHOT_DIR=$PWD/build/snap-c`，PNG 会写到那个目录；`build/` 已被 git 忽略）。

## 2. 去掉 `QuickScanFlow.shared` 单例

全局单例绑定会话，测试之间会互相污染，凭证交接也可能串到别的会话。改为：

- 删除 `static let shared`。`QuickScanFlow` 改成 `ObservableObject`（不需要 `@Published` 属性），`init(session:)` 直接绑定会话；保留给测试用的 `init(currentIdentity:api:)`。
- `MainTabView` 用 `@StateObject` 持有一个实例。因为 `@StateObject` 初始化时拿不到环境里的 `session`，做法是让 `MainTabView` 增加 `init(session: SessionStore)`，在里面创建 `QuickScanFlow(session: session)`，并把它用 `.environmentObject` 注入给各 Tab 的导航栈；`ContentView` 与测试里创建 `MainTabView` 的地方相应传入 `session`（`ContentView` 可以从 `@EnvironmentObject` 取到后传入）。只改这些调用点。
- `ScanConfirmView` / `ScanConfirmModel` 及其他用到 `QuickScanFlow.shared` 的地方改为使用注入的实例。
- 补一个测试：两个各自绑定不同会话的 `QuickScanFlow` 实例互不影响（A 的交接 B 取不到）。

## 3. 扫到对局链接要能进入对局

现在 `StartResult.spectate(matchId:)` 只是关闭扫码页。改为 `navigator.open(.liveMatch(matchId: matchId))`（目的地目前是占位页，下一轮换成实时计分页）。相应更新测试。

## 约束

- 除上面点名的文件及其测试外不改任何代码。`AppRoute.swift` 不用改。
- 其他开发线正在修改 `iOS/iOS/Features/**`（除 `Features/Match/`）、`iOS/iOS/iOSApp.swift`、`iOS/iOS/App/ActivityStore.swift` 及其测试，不要碰。`MainTabView.swift` 只改扫码与 `init` 相关的行。
