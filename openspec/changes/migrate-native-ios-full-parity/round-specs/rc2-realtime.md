# RC2：实时通道（WebSocket 客户端，纯基础设施，不含界面）

本轮迁移小程序的实时通道：对局通道（`/api/match/ws`）和用户通道（`/api/user/ws`）。**只新增文件**（`iOS/iOS/Shared/Realtime/` 与 `iOS/iOSAuthTests/`），不修改任何现有文件，不写 SwiftUI 界面。

## 本轮要读的小程序源码

- `uniapp/utils/websocket.js`（656 行，全部要迁）、`uniapp/utils/websocket-reconnect.js`
- 测试：`uniapp/tests/websocket-reconnect.test.mjs`、`websocket-lifecycle-contract.test.mjs`、`websocket-ticket-contract.test.mjs`、`user-websocket.test.mjs`
- `uniapp/App.vue` 里对 `userWS` / `matchWS` 的使用（前后台切换、网络状态、登录 / 退出时怎么连、怎么断）——只为了解调用方式
- 票据接口：`POST /api/user/ws-ticket`、`POST /api/match/ws-ticket`（`MatchWSTicketReq`、`WSTicketResp`，见 `backend/chasing_points.api`）；服务端握手、心跳、消息信封的格式读 `backend/internal/` 里 WebSocket 相关实现确认（搜 `ws-ticket`、`/ws`），消息类型以 `WS_MESSAGE_TYPES` 为准。

## 任务

1. **重连规则**（`RealtimeReconnect.swift`）：`websocket-reconnect.js` 的全部导出逐条迁成纯函数（`classifyWebSocketFailure`、`calculateReconnectDelay`（随机数可注入）、`shouldReconnectWebSocket`、连接代次相关函数），搬运 `websocket-reconnect.test.mjs` 全部用例。
2. **传输层抽象**（`RealtimeTransport.swift`）：

```swift
/// WebSocket 连接的最小抽象，便于在测试里注入替身。
protocol RealtimeSocket: AnyObject {
    func send(_ text: String) async throws
    func receive() async throws -> String      // 连接关闭或出错时抛错
    func close()
}

protocol RealtimeTransport {
    func connect(url: URL) async throws -> RealtimeSocket
}

/// 系统实现：URLSessionWebSocketTask。
struct URLSessionRealtimeTransport: RealtimeTransport { ... }
```

3. **连接管理**（`RealtimeConnection.swift`）：把 `websocket.js` 里基类的行为迁成 `@MainActor` 的类，对外能力与 JS 一一对应：
   - 按目标连接（同一目标重复连接复用现有连接并触发 `onReuse` 对应的行为）、连接代次（旧连接的回调一律忽略）、连接前向票据签发方要票据（签发方是注入的 `async` 闭包）、拼接地址（`AppConfiguration.wsBaseURL` + 路径 + 查询参数，票据做 URL 编码）。
   - 心跳（间隔与内容照搬）、消息分发（`on(type:)` / `off`，按消息 `type` 分发；解析失败的消息忽略）、发送（未连接时的处理照搬）。
   - 失败与重连：按 `shouldReconnectWebSocket` 决策；`session-invalid` 时调用注入的「会话失效」回调并停止重连；`permission-denied` 停止重连；可恢复错误按 `calculateReconnectDelay` 退避重连。
   - 前后台 / 网络 / 目标是否活跃：`setForeground`、`setNetworkOnline`、`setTargetActive`、`pauseConnection`、`resumeConnection`、`disconnect` 的语义逐条对应。
   - 计时（心跳、重连延时）用可注入的睡眠函数（默认 `Task.sleep`），测试里用可控时钟，不真的等待。
   - 票据、带票据的地址不得写日志、不得落盘。
4. **两条通道**（`MatchRealtime.swift`、`UserRealtime.swift`）：对应 `MatchWebSocket`（按 `matchId` 连接，查询参数、连接成功 / 复用后的同步请求照搬）与 `UserWebSocket`。票据签发分别走 `session.authorizedPost("/api/match/ws-ticket", ...)` 与 `session.authorizedPost("/api/user/ws-ticket", ...)`；观赛（访客）场景下对局通道是否需要票据、怎么连，以 JS 与后端实现为准。
5. **消息类型**（`RealtimeMessage.swift`）：`WS_MESSAGE_TYPES` 的全部类型常量；消息信封的解码结构（`type` + 原始 payload，payload 保留为 `Data` 供后续各业务自行解码）。
6. 本轮**不做**：把通道接到应用生命周期（`scenePhase`、网络监听）和任何页面——后面的轮次再接。但要在 `UserRealtime` / `MatchRealtime` 上把这些接入点（前后台、网络、身份变化时断开）以方法形式备好，并在交付说明里列出接入时需要调用的方法清单。

## 测试

搬运四个 JS 测试文件里适用的用例；另外用替身传输层覆盖：票据签发失败 / 401 / 403 的不同结局、同目标复用不重复连接、切换目标时旧连接的迟到消息被忽略、心跳按间隔发送、断线后按退避重连且成功后退避重置、后台 / 离线 / 目标不活跃时不重连并在恢复后重连、`disconnect()` 后不再重连也不再分发消息、会话失效回调只触发一次、地址中的票据被正确编码。

## 并行开发提醒

其他开发线正在修改 `iOS/iOS/Features/**`、`iOS/iOS/App/**`、`iOS/iOS/SessionStore.swift`、`iOS/iOS/SettingsView.swift` 及其测试。本轮只新增 `iOS/iOS/Shared/Realtime/**` 和对应测试文件，其他一概不碰。本轮没有界面，不需要快照测试。
