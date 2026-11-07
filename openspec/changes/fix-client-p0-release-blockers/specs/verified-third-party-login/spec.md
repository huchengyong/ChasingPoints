## ADDED Requirements

### Requirement: 第三方身份必须由服务端验证得到
通用第三方登录接口 MUST 只接受受支持 Provider 的可验证短期授权凭据，并 SHALL 由服务端通过 Provider 官方验证链路解析稳定用户标识。客户端提交的 `open_id`、`union_id`、昵称或头像 MUST NOT 作为身份真实性依据。

#### Scenario: Huawei 授权凭据有效
- **WHEN** HarmonyOS 客户端提交由 Huawei 登录 SDK 刚获取且属于当前应用的有效授权凭据
- **THEN** 服务端 SHALL 通过 Huawei 官方验证链路解析用户标识
- **AND** SHALL 以服务端解析出的 Provider 与用户标识查找或创建 OAuth 关联

#### Scenario: 客户端伪造 OpenID
- **WHEN** 请求只提交 `provider/open_id` 或提交的 `open_id` 与 Provider 验证结果不一致
- **THEN** 服务端 MUST 拒绝登录
- **AND** MUST NOT 创建用户、OAuth 关联或签发任何 token

#### Scenario: Provider 不受支持
- **WHEN** 请求指定未进入服务端允许列表的 Provider
- **THEN** 服务端 MUST 返回稳定的“不支持的登录方式”错误
- **AND** MUST NOT 将任意 Provider 名称直接写入 OAuth 关联

### Requirement: Provider 验证必须失败关闭
服务端 MUST 校验授权凭据的签发方、受众应用、有效期和 Provider 返回状态；Provider 配置缺失、响应异常、超时或凭据失效时 MUST 拒绝登录，不得降级为信任客户端身份。

#### Scenario: 生产环境缺少 Provider 密钥配置
- **WHEN** 生产环境启用了 Huawei 登录但服务端缺少验证所需配置
- **THEN** 登录请求 SHALL 返回服务不可用类别错误
- **AND** MUST NOT 回退到旧版 `open_id` 登录

#### Scenario: Provider 暂时不可用
- **WHEN** 官方验证接口超时、返回 5xx 或返回无法解析的响应
- **THEN** 服务端 SHALL 返回可重试的上游服务错误
- **AND** MUST NOT 把故障解释为新用户并创建账号

#### Scenario: 授权凭据过期或被拒绝
- **WHEN** Provider 明确拒绝凭据或凭据已过期
- **THEN** 服务端 SHALL 返回稳定的授权失效错误
- **AND** 客户端 SHALL 引导用户重新发起平台授权

### Requirement: 同一第三方身份必须幂等绑定
同一 Provider 的同一服务端验证身份 SHALL 始终解析到同一有效用户；并发或重复登录 MUST NOT 创建重复用户或重复 OAuth 关联。

#### Scenario: 同一凭据被并发提交
- **WHEN** 两个请求并发提交可解析到同一 Provider 身份的有效凭据
- **THEN** 系统 SHALL 最多创建一个 OAuth 关联和一个新用户
- **AND** 两个成功请求 SHALL 返回同一用户身份

#### Scenario: 身份已绑定停用用户
- **WHEN** 验证后的 Provider 身份关联到不存在、停用或已注销用户
- **THEN** 服务端 MUST 拒绝登录
- **AND** MUST NOT 自动创建替代账号或重新激活原账号

### Requirement: 客户端只传递授权凭据并保护敏感信息
App 客户端 SHALL 在用户主动触发并同意协议后获取新的 Provider 授权凭据，只向认证 API 传递完成服务端验证所需的最小字段；客户端与服务端日志 MUST NOT 记录完整授权凭据、access token 或 Provider 响应中的敏感字段。

#### Scenario: 用户发起 Huawei 登录
- **WHEN** 用户在 HarmonyOS 客户端点击 Huawei 登录且同意协议
- **THEN** 客户端 SHALL 获取新的授权凭据并提交到认证 API
- **AND** MUST NOT 再把 SDK 返回的 `open_id` 作为登录凭据

#### Scenario: 登录失败被记录
- **WHEN** 第三方验证或登录失败需要记录诊断日志
- **THEN** 日志 SHALL 只包含 Provider、错误类别、请求标识和脱敏后的关联信息
- **AND** MUST NOT 包含完整授权码、access token、OpenID 或 UnionID
