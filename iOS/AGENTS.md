# NATIVE IOS GUIDE

适用于 `iOS/` 原生 SwiftUI 用户端，同时遵循[仓库指南](../AGENTS.md)。最低部署目标保持 **iOS 17.0**；不要把 UniApp 的 JS、rpx、WXSS 或页面生命周期规则套到原生端。

## 关键入口
| 任务 | 位置 |
| --- | --- |
| 应用入口与测试宿主隔离 | [iOS/iOSApp.swift](iOS/iOSApp.swift) |
| Tab 与“我的”导航 | [iOS/ContentView.swift](iOS/ContentView.swift) |
| 会话、Keychain 与认证请求 | [iOS/SessionStore.swift](iOS/SessionStore.swift) |
| HTTP 解析、错误反馈与构建环境 | [iOS/APIClient.swift](iOS/APIClient.swift) 中的 `APIClient` / `AppConfiguration` |
| 路由、主 Tab 与导航栈 | [iOS/App/](iOS/App/)：`AppRoute`（与小程序页面对应，每个 case 都在 `AppRouteDestination` 登记）、`AppNavigator`、`MainTabView` |
| 外部链接入口 | [iOS/iOSApp.swift](iOS/iOSApp.swift) 的 `onOpenURL` / Universal Links → [iOS/Features/LinkEntry/](iOS/Features/LinkEntry/)；关联域名 `applinks:www.zhuifen.cn` 在 [iOS/iOS.entitlements](iOS/iOS.entitlements)，对应的官网声明文件见 [website/AGENTS.md](../website/AGENTS.md) |
| 隐私清单 | [iOS/PrivacyInfo.xcprivacy](iOS/PrivacyInfo.xcprivacy)：新增需声明原因的系统 API 或新的数据收集时同步更新 |
| 读取模型与失败提示 | [iOS/Shared/ResourceLoader.swift](iOS/Shared/ResourceLoader.swift)、[iOS/PagedHistoryList.swift](iOS/PagedHistoryList.swift)、[iOS/Shared/LoadFailure.swift](iOS/Shared/LoadFailure.swift) |
| 共享 UI 组件 | [iOS/Shared/UI/](iOS/Shared/UI/)（状态页、头像、按钮样式、卡片、Toast） |
| 业务模块 | [iOS/Features/](iOS/Features/)，每个模块一个目录：接口门面、规则、页面模型、页面 |
| 主题与外观偏好 | [iOS/AppTheme.swift](iOS/AppTheme.swift)、[iOS/SettingsView.swift](iOS/SettingsView.swift) |
| 构建与测试配置 | [iOS.xcodeproj/project.pbxproj](iOS.xcodeproj/project.pbxproj)、[共享 scheme](iOS.xcodeproj/xcshareddata/xcschemes/iOS.xcscheme) |
| 可控测试与夹具 | [iOSAuthTests/](iOSAuthTests/) |

工程使用文件系统同步组；在现有源码/测试目录新增 Swift 文件会自动纳入对应 target，不为普通新增文件手工补 pbxproj 条目。

## 会话与资料边界
- 认证请求复用 `SessionStore.authorizedRequest`；页面不另建 URLSession 请求路径、手动拼 token 或重复实现刷新/全局失效导航。
- 保留普通 401 的共享刷新与一次重放、明确 `SESSION_INVALID` 的清理规则，以及暂时性网络/服务失败不误清会话的行为。
- `SessionStore.user` 是共享资料来源；写入响应经会话层校验后发布，不公开任意 setter。草稿与异步结果绑定身份/认证代次，退出或切号后旧结果不能更改新用户资料、提示和导航。
- 普通 Tab 切换、设置/编辑页进出复用会话资料，不在 `onAppear` 新增无条件资料 GET。需要新增读取入口时补去重、迟到响应及保存后不被旧读取回滚的测试。
- 具体业务行为见 [认证规格](../openspec/specs/native-ios-auth-integration/spec.md) 与 [资料规格](../openspec/specs/native-ios-profile-basics/spec.md)，接口字段仍以 [backend/chasing_points.api](../backend/chasing_points.api) 为准。

## 页面迁移约定
- 页面与小程序一一对应，业务规则以小程序对应页面和 `uniapp/utils` 为准；迁移规则时把 `uniapp/tests` 里的用例一并迁成 XCTest。合规开关 [iOS/Shared/AppFlags.swift](iOS/Shared/AppFlags.swift) 与小程序 `utils/compliance-mode.js` 同值。
- 页面跳转走 `AppRoute`（`NavigationLink(value:)` 或 `AppNavigator.open`）；新页面在 `AppRouteDestination` 里登记，不在页面里直接构造其他模块的视图。
- 需要登录的读取用 `authorizedRequest` / `authorizedPost`，访客可读接口用 `optionalAuthRequest`，查询串用 `APIQuery.path`。读取结果绑定发起时的 `session.identity`，只采纳最新一次请求。
- 页面的 `init` 接受可选的页面模型（默认自建），便于测试注入已读好数据的模型。
- 把整数拼进文案时先转成字符串：`Text("排位分 \(score)")` 会按系统区域加千分位。
- 分享海报用 SwiftUI 视图 + `ImageRenderer` 生成（参考 `Features/PKReport/`、`Features/Match/Share/`）；远程图片要在渲染前下载好，失败退回默认图，不阻塞海报。
- 新增顶层类型或 `AppRoute` case 前先搜索同名定义；与 `.api` 同名的模型全仓只保留一份，只取部分字段的窄解码类型起带用途的名字。

## 主题、环境与测试隔离
- 品牌与状态语义遵循 [DESIGN.md](../DESIGN.md)；颜色走 `AppTheme`，外观偏好沿用 `theme_mode` / `ThemeMode`，布局、导航与可访问性使用原生能力。
- Debug/Release 的 API 地址与 Keychain service 由 `AppConfiguration` 区分，不共用真实会话存储，不为联调把生产配置改成测试地址。
- XCTest 使用共享 scheme 的 `CHASING_POINTS_AUTH_TEST_HOST=1`：宿主不能创建日常 `SessionStore` 或自动读取真实凭据；该标记只用于 TestAction，不能带到正常 Run。
- 网络测试注入 `MockHTTPServer` / `MockURLProtocol`；Keychain 测试使用独立 service 并清理自身数据，不访问真实短信/账号。当前 MockURLProtocol 持有共享替身，测试保持串行执行。
- 快照测试先让页面模型读完数据再传给页面渲染；给页面定时循环注入的 `sleep` 必须真正挂起，空实现会让主线程空转、卡住后续测试。
- 不通过读取/复制真实 Keychain、伪造 JWT 或修改正常启动路径来补测试条件。真实账号联调须另行授权并与模拟响应结果分开记录。
- 真实后端只读验收用 `iOSAuthTests/LiveBackendAcceptanceTests`：只有通过 `TEST_RUNNER_LIVE_*` 提供测试环境地址与专用账号 token 时才运行，否则整类跳过；该文件只允许 GET，token 不得写入源码、日志或文档。运行方式见 [docs/testing/native-ios-full-parity.md](../docs/testing/native-ios-full-parity.md)。

## 验证命令
在 `iOS/` 执行，需要 Xcode 与可用 iOS Simulator runtime；模拟器实际运行版本与最低部署目标分别记录。

```bash
# 选择本机可用模拟器，将其 UDID 设置到 SIMULATOR_ID 环境变量
xcrun simctl list devices available
xcodebuild test -project iOS.xcodeproj -scheme iOS -configuration Debug \
  -destination "platform=iOS Simulator,id=${SIMULATOR_ID:?请先设置本机可用模拟器UDID}" \
  -parallel-testing-enabled NO

# 不依赖指定模拟器的 Debug/Release 编译
xcodebuild build -project iOS.xcodeproj -scheme iOS -configuration Debug \
  -destination 'generic/platform=iOS Simulator' CODE_SIGNING_ALLOWED=NO
xcodebuild build -project iOS.xcodeproj -scheme iOS -configuration Release \
  -destination 'generic/platform=iOS Simulator' CODE_SIGNING_ALLOWED=NO
```

查看界面：测试里的 `renderSnapshot`（[iOSAuthTests/SnapshotSupport.swift](iOSAuthTests/SnapshotSupport.swift)）会把页面按夹具数据渲染成 PNG；运行测试时在命令前加 `TEST_RUNNER_SNAPSHOT_DIR=<输出目录>` 才会落盘。快照只断言能渲染，不做像素比对，也不替代真机验收。

认证与资料改动运行现有 XCTest 并核对上述构建/隔离配置；UI、恢复与真实后端行为另做人工验收。验收记录模板与既有证据见 [docs/testing/](../docs/testing/)，历史通过不代表本次已执行。
