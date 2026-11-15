## Context

原生端现状：`SessionStore`（Keychain 凭据、401 刷新重放、认证代次）、`APIClient`、`AppTheme`、登录、昵称、设置、对局记录与战报，共约 2,500 行，98 个测试。小程序约 60 个页面、3 万多行 Vue，业务规则大量沉淀在 `uniapp/utils/*.js` 纯函数和对应的 Node 测试里。

约束：最低 iOS 17.0；不引入第三方依赖；认证请求只能走 `SessionStore`；测试宿主不得创建真实会话；工程使用文件系统同步组，新增 Swift 文件不需要改 pbxproj。

## Goals / Non-Goals

**Goals:**

- 小程序已注册、且在合规模式下对用户可见的页面，原生端都有对应页面，数据、状态与文案一致。
- 每个阶段结束时应用都能构建、测试通过，未迁移的入口显示占位而不是崩溃或死链。
- 业务规则迁移时连同小程序的测试用例一起迁移。

**Non-Goals:**

- 不改后端接口、不改小程序。
- 不接微信开放平台 SDK（微信登录、微信分享）、不接 App 内购买、不接 APNs 证书配置——这些需要账号侧准备，另行立项。
- 不做 iPad 专门布局；不做离线缓存。

## Decisions

1. **目录**：现有文件原地不动。新增 `iOS/iOS/App/`（路由、导航、主 Tab）、`iOS/iOS/Shared/`（基础设施与 `UI/` 组件）、`iOS/iOS/Features/<模块>/`（每个业务模块自成目录）。理由：模块互不重叠，迁移可以逐模块推进和复核；不移动旧文件避免无意义 diff。

2. **路由**：单一 `AppRoute` 枚举对应 `pages.json`，`AppRouteDestination` 把路由映射到页面，未迁移的映射到占位页。每个 Tab 一个 `NavigationStack`，路径存在 `AppNavigator`。理由：消息通知、首页卡片、跨 Tab 跳转都需要「按路由打开页面」；迁移一个页面只需改一行映射。备选是各页面直接 `NavigationLink { View }`，但无法从通知或其他 Tab 打开。

3. **扫码 Tab**：用标准 `TabView` 的第三个 Tab 项，自定义 `Binding` 拦截选中，只触发扫码、不切换 Tab。不用在 TabBar 上叠加悬浮按钮。理由：悬浮按钮在 iPad、键盘弹出和不同安全区下位置不可靠。

4. **读取模型**：`ResourceLoader<Value>` 统一「单资源读取」的状态机——首次加载、刷新保留旧数据、失败重试、写后失效、只采纳最新请求、结果绑定发起时的身份。分页列表沿用并泛化现有 `PagedHistoryList`。不做跨页面缓存与 TTL。理由：小程序的 SWR/TTL 缓存是为了小程序频繁 `onShow`；原生页面生命周期由导航栈持有，先保证正确，性能问题出现再加。

5. **访客可读接口**：`SessionStore.optionalAuthRequest`，有凭据带 token（含刷新规则），没有凭据裸请求；请求期间登录完成则丢弃访客结果。

6. **规则迁移**：`uniapp/utils` 的纯函数迁成 Swift 的纯函数/值类型，对应的 `uniapp/tests/*.test.mjs` 用例迁成 XCTest。实时计分的规则判定以后端返回的状态为准，客户端只做展示与输入约束，与小程序一致。

7. **界面验收手段**：测试里用 `UIWindow` + `drawHierarchy` 把页面按夹具数据渲染成 PNG（亮/暗），供人工查看；它不是像素对比测试，只断言能渲染。真实后端联调与真机验收单独记录。

8. **合规开关**：`AppFlags` 与 `uniapp/utils/compliance-mode.js` 同值同名；受它控制的入口和页面与小程序同步显示/隐藏。

9. **平台能力替代**：扫码用 AVFoundation；定位用 CoreLocation；地图用 MapKit（后端坐标为 GCJ-02，中国大陆的 MapKit 同为 GCJ-02）；图片上传用 PhotosPicker + 七牛上传凭证；海报用 SwiftUI 渲染成图后走系统分享；实时通道用 `URLSessionWebSocketTask`。

## Risks / Trade-offs

- [规模大，由较弱的模型编写] → 每轮规格只覆盖一个模块，写明硬性约束、接口契约出处和必须新增的测试；每轮都人工读 diff、复跑构建和测试，问题写成补丁规格让它修。
- [实时计分页逻辑复杂（斯诺克、中八、九球、裁判模式）] → 放在后期，先迁规则与测试，再迁界面；对照小程序 Node 测试逐条验证。
- [没有真实账号联调] → 夹具必须对照 `backend/chasing_points.api` 与后端实现编写；真实联调需要用户授权后单独执行并记录。
- [合规模式下部分页面不可见] → 这些页面迁移优先级最低，路由保留占位。
- [App Store 审核对登录与支付的要求] → 只提供手机号登录时不需要 Sign in with Apple；会员购买在接入 App 内购买前不展示入口。

## Migration Plan

在分支 `feature/native-ios-parity` 上按阶段提交，每阶段可单独合入 `devlop`。原生 iOS 尚未上架，不涉及线上回滚。

## Open Questions

- 微信登录、微信分享、APNs、App 内购买是否要做，取决于账号侧准备，需用户决定后另行安排。
- 生产发布所需的签名、隐私清单与上架材料不在本变更范围。
