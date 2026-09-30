# NATIVE IOS GUIDE

适用于 `iOS/` 原生 SwiftUI 用户端，同时遵循[仓库指南](../AGENTS.md)。最低部署目标保持 **iOS 17.0**；不要把 UniApp 的 JS、rpx、WXSS 或页面生命周期规则套到原生端。

## 关键入口
| 任务 | 位置 |
| --- | --- |
| 应用入口与测试宿主隔离 | [iOS/iOSApp.swift](iOS/iOSApp.swift) |
| Tab 与“我的”导航 | [iOS/ContentView.swift](iOS/ContentView.swift) |
| 会话、Keychain 与认证请求 | [iOS/SessionStore.swift](iOS/SessionStore.swift) |
| HTTP 解析、错误反馈与构建环境 | [iOS/APIClient.swift](iOS/APIClient.swift) 中的 `APIClient` / `AppConfiguration` |
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

## 主题、环境与测试隔离
- 品牌与状态语义遵循 [DESIGN.md](../DESIGN.md)；颜色走 `AppTheme`，外观偏好沿用 `theme_mode` / `ThemeMode`，布局、导航与可访问性使用原生能力。
- Debug/Release 的 API 地址与 Keychain service 由 `AppConfiguration` 区分，不共用真实会话存储，不为联调把生产配置改成测试地址。
- XCTest 使用共享 scheme 的 `CHASING_POINTS_AUTH_TEST_HOST=1`：宿主不能创建日常 `SessionStore` 或自动读取真实凭据；该标记只用于 TestAction，不能带到正常 Run。
- 网络测试注入 `MockHTTPServer` / `MockURLProtocol`；Keychain 测试使用独立 service 并清理自身数据，不访问真实短信/账号。当前 MockURLProtocol 持有共享替身，测试保持串行执行。
- 不通过读取/复制真实 Keychain、伪造 JWT 或修改正常启动路径来补测试条件。真实账号联调须另行授权并与模拟响应结果分开记录。

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

认证与资料改动运行现有 XCTest 并核对上述构建/隔离配置；UI、恢复与真实后端行为另做人工验收。验收记录模板与既有证据见 [docs/testing/](../docs/testing/)，历史通过不代表本次已执行。
