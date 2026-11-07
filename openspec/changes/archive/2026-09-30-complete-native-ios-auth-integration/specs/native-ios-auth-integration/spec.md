## ADDED Requirements

### Requirement: 原生短信发送与验证码登录遵守现有契约

原生 iOS SHALL 通过统一请求层调用短信与登录接口，提交 `phone`、`scene: login` 或 `sms_code`，解析顶层成功响应。客户端 MUST 校验手机号和六位数字验证码，并在提交登录前确认用户已同意协议；进行中的发送/登录操作 MUST 防止重复提交。

#### Scenario: 发送成功后进入倒计时
- **WHEN** 有效手机号的发送请求返回业务成功
- **THEN** 页面 SHALL 提示验证码已发送并启动六十秒重发倒计时
- **AND** 倒计时期间 MUST 禁止重复发送，倒计时结束后允许再次发送

#### Scenario: 发送失败或触发频控
- **WHEN** 短信接口返回 `success: false`，包括 HTTP 200 的失败响应
- **THEN** 页面 MUST 显示实际失败类别对应的提示，不能显示发送成功
- **AND** SHALL 结束忙碌状态，允许用户按提示处理或稍后重试

#### Scenario: 本地输入或协议条件不满足
- **WHEN** 手机号无效、验证码不是六位数字，或用户未同意协议
- **THEN** 客户端 MUST 阻止相应的发送或登录请求；协议确认只约束登录提交
- **AND** 页面 SHALL 提供可见的输入校验或协议提示

#### Scenario: 验证码登录成功
- **WHEN** 登录接口返回完整非空令牌对和有效普通用户资料，且 Keychain 保存成功
- **THEN** 客户端 SHALL 更新当前会话、关闭登录页并在“我的”展示返回的用户资料
- **AND** 后续认证请求 MUST 使用该会话的 access token

#### Scenario: 登录响应不完整或凭据保存失败
- **WHEN** 登录响应缺少令牌/有效用户资料，或 Keychain 写入失败
- **THEN** 客户端 MUST 保持此前的会话状态并显示失败，不能发布新的已登录状态
- **AND** MUST NOT 留下部分更新的凭据

### Requirement: 原生错误反馈保留真实失败类别

请求层 SHALL 保留 HTTP 状态和 `reason`，支持现有 JSON 业务错误及后端的用户可读纯文本错误。客户端 MUST 区分验证码/频控、网络连接、服务故障与会话失效，不得将所有错误展示为“网络异常”或登录成功。

#### Scenario: 错误或过期验证码返回纯文本
- **WHEN** 登录接口以 HTTP 400 纯文本返回已核对的验证码错误或过期提示
- **THEN** 登录页 SHALL 展示对应的可理解原因并保持可重试
- **AND** MUST NOT 更新用户或凭据

#### Scenario: 未知服务错误不泄露响应体
- **WHEN** 响应是 HTML、堆栈、空体或不可展示的内部错误
- **THEN** 客户端 SHALL 使用与失败类别匹配的通用中文提示
- **AND** MUST NOT 将原始响应体直接显示给用户

### Requirement: 启动恢复必须验证 Keychain 会话

原生 App SHALL 从当前环境 Keychain 读取凭据，并通过 `GET /api/user/info` 验证后恢复用户；同一会话的并发恢复触发 MUST 避免重复网络验证。本地仅有 token 不能作为已登录资料的来源。

#### Scenario: 无凭据启动
- **WHEN** 当前环境 Keychain 没有凭据
- **THEN** App SHALL 保持访客状态，提供登录入口，并且不发起用户信息或刷新请求

#### Scenario: 登录后杀进程再启动
- **WHEN** 本机保存有效凭据后 App 被终止并重新启动
- **THEN** App SHALL 展示恢复状态，用 access token 请求用户信息，并在成功后显示该用户
- **AND** MUST NOT 要求用户再次输入短信验证码

#### Scenario: 恢复遇到暂时性故障
- **WHEN** 用户信息验证因断网、超时或 5xx 失败
- **THEN** App MUST 保留可能有效的凭据，结束加载并显示准确的故障提示及重试入口
- **AND** 重试成功后 SHALL 恢复用户，而不要求重新登录

### Requirement: 普通认证失败共享刷新且最多重放一次

对不含 `SESSION_INVALID` 的 HTTP 401，原生会话层 SHALL 共享当前会话的刷新操作，在新令牌对保存成功后重放原请求一次；MUST 避免同批旧凭据请求因响应先后差异重复刷新或无限重试。

#### Scenario: Access token 过期但 refresh token 有效
- **WHEN** 认证请求返回普通 401 且刷新接口返回有效新令牌对
- **THEN** 客户端 SHALL 先持久化新凭据，再使用新 access token 重放一次原请求
- **AND** 用户 SHALL 保持登录，下次启动 SHALL 使用更新后的凭据恢复

#### Scenario: 多个旧凭据请求交错返回 401
- **WHEN** 多个请求使用相同旧凭据发出，其 401 在刷新进行中及刷新完成后先后返回
- **THEN** 客户端 SHALL 共用一次刷新结果，每个原请求最多重放一次
- **AND** MUST NOT 因较晚返回的旧 401 再发起一次刷新

#### Scenario: 刷新或重放发生暂时性故障
- **WHEN** 刷新或重放因断网、超时、5xx 失败，且没有明确认证失效证据
- **THEN** 客户端 MUST 保留当前仍可能有效的凭据，结束当前请求并允许后续重试
- **AND** MUST NOT 把该失败宣称为账号失效

#### Scenario: 重放仍然返回 401
- **WHEN** 原请求使用刷新后的 access token 重放后仍收到 401
- **THEN** 客户端 SHALL 进入当前会话失效流程
- **AND** MUST NOT 再次刷新或重放该请求

### Requirement: 明确失效必须幂等清理并引导重新登录

原生会话层 MUST 区分明确失效和普通 401。当前会话明确失效时 SHALL 清理内存及本环境 Keychain，统一提示重新登录并引导登录页面；同一认证代次的清理、提示与导航最多一次。

#### Scenario: 用户信息返回失效原因
- **WHEN** 当前会话请求收到 HTTP 401 且 `reason: SESSION_INVALID`
- **THEN** 客户端 MUST 直接清理会话并引导重新登录，不能发起刷新

#### Scenario: 刷新返回业务层失效
- **WHEN** 刷新接口以 HTTP 200 + `success: false` + `reason: SESSION_INVALID`，或明确认证失败的 401 返回
- **THEN** 客户端 MUST 将其视为无法恢复的会话并清理凭据
- **AND** MUST NOT 重放原请求或保留已登录展示

#### Scenario: 并发请求同时确认失效
- **WHEN** 同一认证代次的多个请求确认失效或共享刷新失效
- **THEN** 客户端 SHALL 只清理、提示并导航一次
- **AND** 页面级错误处理 MUST NOT 重复提示或打开多个登录页

### Requirement: 异步结果必须受当前认证代次约束

登录、恢复、刷新、原请求及其重放的结果 MUST 仅能作用于发起时所属会话。退出或新登录替换会话后，旧操作 MUST NOT 改写当前用户、Keychain 或当前会话的错误与导航状态。

#### Scenario: 刷新或登录返回前用户退出
- **WHEN** 请求尚未完成时执行退出，随后旧登录/刷新响应返回成功
- **THEN** 客户端 MUST 丢弃旧结果，保持未登录且 Keychain 无旧凭据

#### Scenario: 旧用户响应在新用户登录后到达
- **WHEN** 用户 A 的恢复、业务请求或刷新响应在用户 B 登录后返回成功或失效
- **THEN** 用户 B 的资料及凭据 MUST 保持不变
- **AND** MUST NOT 因用户 A 的失败清理 B 或提示 B 重新登录

### Requirement: 本机退出清除可恢复身份且保持环境隔离

主动退出 SHALL 清除当前环境内存身份与 Keychain 凭据并淘汰进行中的会话操作。Debug/Release MUST 使用各自后端地址和 Keychain service；退出语义限定为本机，不承诺服务端全设备撤销。

#### Scenario: 退出后重新启动
- **WHEN** 用户主动退出后终止并重新启动 App
- **THEN** App MUST 保持未登录，不能恢复旧用户或使用旧凭据发起请求
- **AND** 主动退出 MUST NOT 显示被动会话失效提示

#### Scenario: 开发与生产凭据隔离
- **WHEN** 两个构建环境分别存在凭据，任一环境读取、刷新或清理自己的会话
- **THEN** 操作 MUST 仅访问自身后端及 Keychain service，不能复用或删除另一环境的凭据

### Requirement: 联调完成必须具有分层且可复现的证据

本变更 SHALL 保留 iOS 17.0 最低部署目标，并提供构建、可控原生测试、真实后端人工联调三类独立结果。所有真实必验项通过前 MUST NOT 将整条登录链路标为完成；缺少手机号、可用短信或隔离刷新环境时 SHALL 记录具体阻塞。

#### Scenario: 真实链路验收
- **WHEN** 执行开发环境的人工验收
- **THEN** 记录 SHALL 包含实际收短信、登录展示、杀进程重启恢复、真实过期刷新、刷新后重启恢复、退出后重启这组必验步骤的预期和实际结果
- **AND** SHALL 记录源码/后端版本、构建配置、设备/runtime、操作步骤和通过/失败/阻塞结论

#### Scenario: 可控测试验证异常与竞争
- **WHEN** 执行原生自动化
- **THEN** 测试 SHALL 覆盖真实响应格式、业务失败/纯文本错误、暂时性故障、Keychain 保存失败、并发刷新、失效清理及旧会话迟到响应
- **AND** SHALL 使用隔离存储和受控响应，不能自动发送真实短信或使用生产凭据

#### Scenario: 人工输入与自动化结果独立确认
- **WHEN** 选择模拟器或真机进行人工联调
- **THEN** 测试人员 SHALL 先确认四个 Tab 可点击切换、登录页可打开并能输入
- **AND** MUST NOT 仅凭编译或 XCTest 通过推断人工输入可用

#### Scenario: 测试数据与密钥边界
- **WHEN** 准备真实短信和过期刷新测试，或保存联调记录
- **THEN** 测试人员 SHALL 使用已确认可测试的手机号与开发环境，短信验证码在 App 内输入，签名及供应商密钥由后端管理
- **AND** 源码与记录 MUST NOT 包含真实 token、验证码、私钥或完整测试手机号
- **AND** 真实过期刷新 SHALL 使用隔离开发实例实际签发并过期的 access token，临时有效期配置在测试后恢复
