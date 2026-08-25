## ADDED Requirements

### Requirement: WebSocket 升级必须执行生产 Origin 策略
两条 WebSocket 路由 MUST 使用统一的、由环境配置驱动的 Origin 校验。生产环境 SHALL 只接受明确允许的浏览器 Origin；不得使用无条件 `true` 或通配符放行。

#### Scenario: 允许列表中的 Origin
- **WHEN** 浏览器或小程序运行时携带的 Origin 与生产允许列表精确匹配
- **THEN** 服务端 SHALL 继续执行鉴权和业务授权检查

#### Scenario: 未允许的浏览器 Origin
- **WHEN** WebSocket 升级请求携带不在允许列表中的 Origin
- **THEN** 服务端 MUST 在升级前拒绝请求
- **AND** MUST NOT 建立连接或泄露鉴权细节

#### Scenario: 原生客户端不发送 Origin
- **WHEN** 已验证的平台原生运行时不发送 Origin 且使用安全传输
- **THEN** 服务端 MAY 按明确配置接受缺失 Origin
- **AND** 仍 MUST 完成 token、用户状态、比赛可见性和参与者权限校验

### Requirement: 长期访问令牌不得出现在 WebSocket URL
认证 WebSocket 连接 MUST 先通过正常 Bearer REST 请求申领短期、单次且绑定用户与通道 scope 的连接 ticket，再使用该 ticket完成升级；生产服务 MUST 拒绝 URL 查询参数中的 access token，日志和指标 MUST NOT 记录 access token 或完整 ticket。

#### Scenario: 使用有效连接 ticket
- **WHEN** App 或微信小程序使用 access token 成功申领用户或对局通道 ticket，并在有效期内发起 WebSocket 升级
- **THEN** 服务端 SHALL 原子消费该 ticket并恢复其用户与通道 scope
- **AND** 请求 URL SHALL 不包含长期 access token

#### Scenario: 旧客户端使用 query token
- **WHEN** 生产请求通过 `?token=<access-token>` 连接任一 WebSocket 路由
- **THEN** 服务端 MUST 拒绝该认证方式
- **AND** MUST NOT 将 query token 复制到错误响应或日志

#### Scenario: 重放已消费 ticket
- **WHEN** 同一连接 ticket被再次用于 WebSocket 升级
- **THEN** 服务端 MUST 拒绝重放
- **AND** MUST NOT 建立第二条认证连接

#### Scenario: Refresh token 申领 ticket
- **WHEN** 调用方使用 refresh token 请求连接 ticket
- **THEN** REST 认证层 MUST 返回认证失败
- **AND** MUST NOT 签发 ticket

### Requirement: 实时通道必须保持既有业务授权边界
用户 WebSocket MUST 只允许有效且启用的登录用户连接；私密对局 WebSocket MUST 只允许参赛者或裁判连接；公开对局 MAY 匿名观赛，但匿名连接 MUST 不具备写操作权限。

#### Scenario: 匿名观赛公开对局
- **WHEN** 未登录用户连接一个公开对局的 WebSocket 且不提交认证凭据
- **THEN** 服务端 MAY 建立只读连接
- **AND** 该连接 MUST NOT 被识别为任何登录用户

#### Scenario: 非参与者连接私密对局
- **WHEN** 有效登录用户不是私密对局的参赛者或裁判
- **THEN** 服务端 MUST 在升级前返回禁止访问

#### Scenario: 已注销用户尝试重连
- **WHEN** access token 指向已注销、已删除或停用用户
- **THEN** 服务端 MUST 拒绝升级
- **AND** 客户端 SHALL 进入全局失效会话流程

### Requirement: 可恢复断线不得因固定次数永久停止
客户端 SHALL 使用有上限延迟的指数退避和抖动恢复用户及对局 WebSocket；只要应用位于前台、网络可用、目标仍有效且会话未被确认失效，就 MUST NOT 因固定重试次数耗尽而永久停止。

#### Scenario: 网络短时中断后恢复
- **WHEN** WebSocket 因网络故障断开且稍后网络恢复
- **THEN** 客户端 SHALL 自动恢复连接
- **AND** 对局通道 SHALL 在连接成功后请求最新快照

#### Scenario: 长时间离线
- **WHEN** 客户端持续无法连接
- **THEN** 重连延迟 SHALL 逐步增加到配置上限并带抖动
- **AND** MUST NOT 以固定 5 次失败作为永久终止条件

#### Scenario: 明确认证失败
- **WHEN** HTTP 握手、关闭码或会话校验明确表明凭据无效
- **THEN** 客户端 MUST 停止该会话的自动重连
- **AND** SHALL 交由统一会话恢复或退出流程处理

### Requirement: WebSocket 恢复必须感知应用生命周期
客户端进入后台、用户退出、账号切换或离开目标对局时 SHALL 暂停或取消对应重连；回到前台且身份与目标仍匹配时 SHALL 恢复连接，旧连接回调 MUST NOT 覆盖新连接状态。

#### Scenario: App 进入后台
- **WHEN** App 在等待重连期间进入后台
- **THEN** 客户端 SHALL 取消待执行的重连定时器并停止心跳
- **AND** MUST NOT 在后台形成持续重连风暴

#### Scenario: 重连期间切换账号
- **WHEN** 用户 A 的连接正在重试时客户端切换到用户 B
- **THEN** 用户 A 的定时器、回调和连接结果 MUST 被淘汰
- **AND** 用户 B 的连接 MUST 使用自己的 auth generation 与凭据

#### Scenario: 用户主动断开
- **WHEN** 页面卸载或用户退出触发主动断开
- **THEN** 客户端 MUST NOT 再自动重连该通道
