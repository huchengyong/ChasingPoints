# RC4：扫码开局（扫码、确认对局、选择球友）

本轮迁移开局流程：点「扫码」→ 相机扫码 → 确认页（确认球种与对局类型后开局，或预览后确认执裁）。上一轮已经把扫码文本解析（`MatchScan`）、入口链接（`EntryLink`）、开局参数（`StartMatch`）等规则迁到 `iOS/iOS/Features/Match/Rules/`。

## 本轮要读的小程序源码

- `uniapp/utils/quick-scan.js`（统一扫码流程，全部要迁）+ `quick-scan.test.mjs`；`quick-scan-entry.js`（接线）
- `uniapp/subPages/match/scanConfirm.vue` + 样式；`uniapp/subPages/match/opponentSelector.vue`
- `uniapp/utils/ongoing-match-guard.js` + `ongoing-match-guard.test.mjs`（已有进行中对局时的拦截与去向）
- `uniapp/api/match.js`：`previewMatchInvite`、`resolveMatchInviteCode`、`startMatch`、`previewRefereeJoin`、`joinMatchReferee`、`getCurrentMatch`；`uniapp/api/opponent.js`
- 契约（`backend/chasing_points.api`）：`MatchInvitePreviewReq/Resp`、`MatchInviteResolveReq/Resp`、`StartMatchReq/Resp`、`RefereePreviewReq/Resp`、`JoinMatchRefereeReq/Resp`、`GetCurrentMatchResp`、`/api/opponent/*`。各种失败（凭证过期、对手有进行中对局、不能和自己开局、信誉受限等）的返回形态与文案读 `backend/internal/logic/match/` 确认。
- 规格 `openspec/specs/quick-match-invites/spec.md`（开局与邀请的权威业务规则）

## 任务

1. **接口门面**（`iOS/iOS/Features/Match/MatchStartAPI.swift`）：上面列出的接口。写请求用 `authorizedPost`。凭证（invite token、join token）只作为参数传递，不进日志。
2. **统一扫码流程**（`iOS/iOS/Features/Match/QuickScanFlow.swift`）：把 `createQuickScan` 迁成 `@MainActor final class QuickScanFlow: ObservableObject`——
   - `start(rematch:replace:)`：未登录 → 去登录；正在扫码中 → 忽略；扫码（注入的 `scan: () async -> ScanOutcome` 闭包，outcome 为 扫到文本 / 取消 / 失败(原因)）→ `MatchScan.resolvePayload` → 按结果分流（观赛、裁判确认、开局确认、再来一局时对手不符的提示等），每个分支的提示文案与 JS 一致。
   - 凭证交接：扫码结果只在内存里交给确认页一次（`takeHandoff(identity:)`），取走或身份不符即清除；`handoffResolved`（入口链接兑换后的交接）同样迁移。不进路由参数、不进 `UserDefaults`、不进日志。
   - 「去哪」用回调 / 发布的导航意图表达，由视图层映射成 `AppRoute`；搬运 `quick-scan.test.mjs` 全部用例。
3. **相机扫码**（`iOS/iOS/Features/Match/QRScannerView.swift`）：`AVCaptureSession` + `AVCaptureMetadataOutput`（只识别 QR），用 `UIViewControllerRepresentable` 包装；全屏展示，带取景框、关闭按钮、手电筒开关。相机权限：未决定时请求；被拒绝时显示「需要相机权限才能扫码」和「去设置」（`UIApplication.openSettingsURLString`）；设备不支持（模拟器）时显示「当前端暂不支持扫码」，并在 **Debug 构建**下额外提供一个「粘贴二维码内容」的输入框，方便在模拟器里走通流程（`#if DEBUG`，Release 不包含）。扫到第一个结果后立即停止采集并回调一次。权限说明文案已在工程里配置（`NSCameraUsageDescription`），本轮不要改工程文件。
4. **接到「扫码」入口**：修改 `iOS/iOS/App/MainTabView.swift` 里扫码占位 sheet → 全屏展示 `QRScannerView` 并驱动 `QuickScanFlow`（只改扫码相关的行）。`AppNavigator.requestScan` 的语义不变。
5. **确认对局**（`iOS/iOS/Features/Match/ScanConfirmView.swift` + 模型）：新增路由 `case scanConfirm`（标题「确认对局」）。进入时从 `QuickScanFlow` 取交接数据（取不到 → 提示并返回，文案照搬）。两种模式，以 `scanConfirm.vue` 为准：
   - 开局：对手信息、球种选择（默认值取常用球种 `GameTypePreference`、再来一局的预设）、对局类型与可见性选项、开局按钮（`StartMatch` 规则生成请求）；成功 → 新增占位路由去实时计分页（用已有或新增的 `case liveMatch(matchId: Int64)`，标题「对局」）；各种失败的提示与处理（含「已有进行中对局」时按 `ongoing-match-guard.js` 的规则引导继续当前对局）照搬；「重新扫码」→ `QuickScanFlow.start(replace: true)`。
   - 执裁：预览对局双方与球种，确认执裁（`joinMatchReferee`）→ 去实时计分页；失败提示照搬。
6. **选择球友**（`OpponentSelectorView.swift`）：新增路由 `case opponentSelector`（标题「选择球友」），以 `opponentSelector.vue` 为准（它在当前流程里怎么被进入、选中后做什么，按页面实际代码）。
7. `ongoing-match-guard.js` 的规则迁移并搬运测试（放 `Features/Match/Rules/`）。

## 测试

搬运 JS 用例；接口门面的路径 / 请求体（snake_case、凭证字段名）；`QuickScanFlow`：各分流、取消、失败原因文案、扫码中重复触发忽略、身份变化后结果作废、交接只能取一次且身份不符取不到；确认页模型：两种模式的成功路径、各失败分支文案、开局中不可重复提交、已有进行中对局的引导；断言任何模型的 `description` / `debugDescription` 不含 token 原文。快照：`scan-confirm-start`、`scan-confirm-referee`、`scan-confirm-missing-handoff`、`scanner-permission-denied`、`scanner-unsupported`（相机画面用占位，不启动真实采集）。

## 并行开发提醒

其他开发线正在修改 `iOS/iOS/Features/**`（除 `Features/Match/`）、`iOS/iOS/SessionStore.swift`、`iOS/iOS/SettingsView.swift`、`iOS/iOS/iOSApp.swift` 及其测试，本轮不要碰这些文件。`MainTabView.swift` 只改扫码 sheet 相关的行；`AppRoute.swift` 只新增本轮点名的 case 与目的地。
