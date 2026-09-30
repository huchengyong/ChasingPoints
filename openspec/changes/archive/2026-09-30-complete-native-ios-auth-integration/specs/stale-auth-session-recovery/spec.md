## MODIFIED Requirements

### Requirement: 客户端幂等处理失效会话

移动端 SHALL 对 `SESSION_INVALID` 直接执行全局会话失效流程，不得使用同属失效用户的 refresh token 重试；并发会话失效 MUST 只触发一次清理、提示和登录导航。UniApp 的普通 401 刷新失败 SHALL 按下述既有规则清理；原生 iOS SHALL 区分明确认证失效与暂时性刷新故障，只有前者进入全局会话失效流程。

#### Scenario: 失效主体跳过刷新

- **WHEN** 任一请求收到 HTTP 401 且响应原因为 `SESSION_INVALID`
- **THEN** 客户端 MUST NOT 调用 refresh-token 接口
- **AND** SHALL 立即进入全局会话失效流程

#### Scenario: 普通 401 刷新成功

- **WHEN** 请求收到不含 `SESSION_INVALID` 的 HTTP 401 且 refresh token 有效
- **THEN** 客户端 SHALL 共享一次 refresh 请求
- **AND** 刷新成功后 SHALL 重放原请求
- **AND** 用户登录状态 SHALL 保持有效

#### Scenario: 普通 401 刷新失败

- **WHEN** UniApp 的 access token 认证失败且 refresh 请求无法恢复会话
- **THEN** UniApp SHALL 进入与 `SESSION_INVALID` 相同的全局清理流程

#### Scenario: 原生 iOS 刷新明确认证失败

- **WHEN** 原生 iOS 的普通 401 触发刷新，刷新返回明确认证失败的 HTTP 401 或 `success: false` 且 `reason: SESSION_INVALID`
- **THEN** 原生 iOS SHALL 进入与 `SESSION_INVALID` 相同的全局清理流程
- **AND** MUST NOT 重放原请求

#### Scenario: 原生 iOS 刷新发生暂时性故障

- **WHEN** 原生 iOS 的普通 401 触发刷新，刷新因断网、超时、5xx 或其他不能证明认证失效的服务/解析错误失败
- **THEN** 原生 iOS MUST 保留当前仍可能有效的凭据，结束当前请求并允许后续重试
- **AND** MUST NOT 执行会话清理、显示重新登录提示或导航登录页
- **AND** SHALL 按真实故障类别向调用方返回错误

#### Scenario: 原生 iOS 并发刷新暂时失败后可重试

- **WHEN** 原生 iOS 的多个认证请求返回普通 401，共享的 refresh 返回 HTTP 503，且没有明确认证失效证据
- **THEN** 原生 iOS MUST 保留凭据，清理、重新登录提示和登录导航的次数均为零
- **AND** SHALL 结束并释放失败的共享刷新任务
- **AND** 服务恢复后的新请求 SHALL 能重新发起刷新，并在刷新成功后重放一次原请求

#### Scenario: 多个请求同时发现失效会话

- **WHEN** 多个并发请求同时收到 `SESSION_INVALID`，或共享刷新按所属客户端规则需要清理会话（UniApp 刷新失败，或原生 iOS 刷新明确认证失败）
- **THEN** 客户端 MUST 只执行一次状态清理
- **AND** MUST 只显示一次重新登录提示
- **AND** MUST 只安排一次登录页导航
- **AND** 页面级调用方不得重复展示认证错误

#### Scenario: 静默请求与普通请求同时发现失效

- **WHEN** 静默请求与普通请求同时确认同一个会话已失效
- **THEN** 静默请求不得抑制全局重新登录提示与导航
- **AND** 全局失效副作用仍 MUST 只执行一次

#### Scenario: 旧请求在新登录后才返回

- **WHEN** 旧 access token 发出的请求在用户已登录新会话后才返回 `SESSION_INVALID`
- **THEN** 客户端 MUST 仅拒绝旧请求
- **AND** MUST NOT 清除、覆盖或导航离开新会话

#### Scenario: 被新会话淘汰的旧请求不展示重新登录

- **WHEN** 旧请求返回会话错误时 access token 或 auth generation 已不再属于当前会话
- **THEN** 客户端 MUST 将该错误标记为 superseded
- **AND** 页面 MUST NOT 显示当前会话正在退出或即将跳转登录页

#### Scenario: 旧 refresh 响应在会话切换后返回

- **WHEN** refresh 请求发出后用户退出或切换了账号
- **THEN** 旧 refresh 响应 MUST NOT 写回 token 或恢复旧登录状态
