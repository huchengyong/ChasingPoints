## ADDED Requirements

### Requirement: 用户必须能够导出自己的个人数据
已登录用户 SHALL 能从客户端设置入口发起个人数据导出。导出内容 MUST 覆盖账号资料、认证关联摘要、设置与偏好、社交关系和用户产生的业务记录，并 SHALL 使用机器可读的 UTF-8 格式；服务端读取 MUST 分批有界且只返回当前用户有权获得的数据。

#### Scenario: 用户成功导出数据
- **WHEN** 有效用户从设置页发起数据导出
- **THEN** 客户端 SHALL 生成或保存一份可识别日期与账号的导出文件
- **AND** 文件 SHALL 包含数据分类、生成时间和格式版本

#### Scenario: 导出包含关联用户信息
- **WHEN** 对局、好友或互动记录涉及其他用户
- **THEN** 导出 SHALL 只包含理解当前用户记录所必需的对方公开展示信息
- **AND** MUST NOT 导出对方手机号、认证标识、推送 token 或私密设置

#### Scenario: 导出过程中服务失败
- **WHEN** 任一导出分段失败或文件未完整生成
- **THEN** 客户端 MUST 显示导出失败并允许重试
- **AND** MUST NOT 把不完整文件宣称为成功导出

### Requirement: 导出结果不得包含认证秘密
个人数据导出 MUST 排除 access token、refresh token、短信验证码、Provider 授权凭据、签名密钥、完整 OpenID/UnionID 和推送 token；认证关联只 MAY 以 Provider 名称、绑定状态和脱敏标识摘要呈现。

#### Scenario: 检查导出文件
- **WHEN** 用户完成数据导出
- **THEN** 文件 MUST NOT 包含可直接登录、刷新会话或调用第三方账户的秘密

#### Scenario: 客户端本地存在 token
- **WHEN** 客户端组装导出文件时本地存储仍含登录 token
- **THEN** 导出逻辑 MUST NOT 读取或写入这些 token

### Requirement: 账号注销必须要求明确确认并再次验证身份
已登录用户 SHALL 能从设置页发起账号注销；系统 MUST 在展示不可逆影响后要求明确二次确认和近期身份验证，单靠后台持有的旧 access token不得静默完成注销。

#### Scenario: 手机号用户确认注销
- **WHEN** 绑定手机号的用户输入有效的注销场景验证码并确认不可逆操作
- **THEN** 服务端 SHALL 接受注销请求并开始数据收口

#### Scenario: OAuth 用户确认注销
- **WHEN** 未绑定手机号的 OAuth 用户提交新的、经服务端验证的 Provider 授权凭据并确认注销
- **THEN** 服务端 SHALL 接受注销请求

#### Scenario: 身份复核失败
- **WHEN** 验证码、Provider 凭据或确认信息无效
- **THEN** 服务端 MUST 拒绝注销
- **AND** 用户账号、会话和数据 MUST 保持不变

### Requirement: 注销必须删除或匿名化个人数据并保留必要事实完整性
注销流程 MUST 清除直接身份信息、OAuth 关联、手机号、头像、推送 token、私密偏好和可删除的社交数据；因对局完整性、安全审计或法定义务需要保留的记录 MUST 去标识化，并不得继续以原昵称或头像公开展示。

#### Scenario: 注销普通用户
- **WHEN** 注销事务成功完成
- **THEN** 用户记录 SHALL 被标记为已注销并清空直接身份字段
- **AND** OAuth 关联与推送标识 SHALL 被删除
- **AND** 公开页面 SHALL 只显示统一的已注销用户占位身份

#### Scenario: 用户存在历史对局
- **WHEN** 已注销用户参与的历史对局需要继续供另一参赛者查看
- **THEN** 比分、时间和比赛事实 MAY 保留
- **AND** 已注销用户的昵称、头像、手机号和认证标识 MUST NOT 保留在展示数据中

#### Scenario: 用户存在进行中对局
- **WHEN** 用户确认注销时仍有进行中对局
- **THEN** 系统 SHALL 使用明确的账号注销原因结束或取消该对局并通知相关参与者
- **AND** MUST NOT 留下无法恢复的进行中状态

### Requirement: 注销结果必须立即收口全部会话
注销成功 SHALL 使所有旧 access token、refresh token、HTTP 请求、用户 WebSocket 和对局写权限立即失效；客户端 SHALL 清除全部持久化身份状态并返回未登录入口。重复注销请求 MUST 幂等且不得恢复已清除数据。

#### Scenario: 旧 token 在注销后访问
- **WHEN** 注销前签发的 token 再次调用受保护 API 或连接 WebSocket
- **THEN** 服务端 MUST 返回 `SESSION_INVALID` 或等价的稳定失效结果

#### Scenario: 客户端收到注销成功
- **WHEN** 注销 API 返回成功
- **THEN** 客户端 SHALL 断开用户和对局 WebSocket、清空认证隔离缓存与本地 token
- **AND** SHALL 跳转到未登录入口并说明注销已完成

#### Scenario: 使用旧 OAuth 身份重新登录
- **WHEN** 已注销账号的旧 OAuth 关联已被删除且用户未重新完成新的注册授权
- **THEN** 系统 MUST NOT 自动恢复原账号或原个人资料
