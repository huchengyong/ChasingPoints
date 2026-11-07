## ADDED Requirements

### Requirement: 页面必须区分加载、成功空数据和加载失败
所有 App / 小程序数据页面 SHALL 使用可判定的 `loading`、`ready`、`empty` 和 `error` 状态。`empty` 只允许在目标请求成功且权威结果确实为空时出现；网络、权限、会话、业务或服务错误 MUST NOT 被渲染成“暂无数据”。

#### Scenario: 首次加载返回空列表
- **WHEN** 请求成功且服务端明确返回空列表或总数为零
- **THEN** 页面 SHALL 展示与业务匹配的空状态
- **AND** MAY 提供创建、筛选或返回等下一步操作

#### Scenario: 首次加载发生网络错误
- **WHEN** 页面没有旧数据且请求因网络连接失败
- **THEN** 页面 SHALL 展示网络失败状态和重试操作
- **AND** MUST NOT 展示“暂无记录”

#### Scenario: 服务端返回业务失败或 5xx
- **WHEN** 请求到达服务端但返回普通业务失败或服务错误
- **THEN** 页面 SHALL 展示暂时无法加载及服务端允许展示的错误信息
- **AND** MUST NOT 将默认空数组解释为成功空数据

### Requirement: 错误类别必须保留到页面反馈层
统一请求层 SHALL 为网络、会话、权限、未找到、限流、服务和普通业务失败提供稳定类别；页面 SHALL 按类别选择反馈，已由全局会话流程处理的错误 MUST NOT 再显示重复空状态或 toast。

#### Scenario: 权限拒绝
- **WHEN** 服务端返回 403 或稳定的权限拒绝类别
- **THEN** 页面 SHALL 告知用户无权查看或执行该操作
- **AND** MUST NOT 提示“暂无数据”或“请检查网络”

#### Scenario: 会话失效由全局接管
- **WHEN** 请求错误标记为 `SESSION_INVALID` 且全局流程已开始清理会话
- **THEN** 页面 MUST NOT 再覆盖为本地空状态
- **AND** MUST NOT 重复弹出登录或网络提示

#### Scenario: 资源确实不存在
- **WHEN** 服务端返回稳定的 404 / not-found 类别
- **THEN** 详情页 SHALL 展示资源不存在或已下线
- **AND** 列表页不得把该错误误当成整个列表为空

### Requirement: 刷新失败必须保留已成功加载的数据
页面已有成功数据时，后台刷新、重新进入或下拉刷新失败 SHALL 保留旧数据并展示非阻塞的失败与重试反馈；不得清空为默认值后展示空状态。

#### Scenario: 已有列表的刷新失败
- **WHEN** 用户已看到成功加载的列表且刷新请求失败
- **THEN** 页面 SHALL 保留原列表
- **AND** SHALL 标记数据可能不是最新并允许再次刷新

#### Scenario: 身份已经切换
- **WHEN** 旧身份请求失败或返回时当前 auth generation 已变化
- **THEN** 页面 MUST 丢弃旧身份结果
- **AND** MUST NOT 用旧数据或旧错误覆盖新身份状态

### Requirement: 错误页面必须提供可执行恢复路径
可恢复的初始加载失败 SHALL 提供重试；权限、会话、资源不存在和平台能力不足等不可由原请求直接恢复的状态 SHALL 提供与原因匹配的返回、登录或设置指引。

#### Scenario: 用户点击重试
- **WHEN** 初始加载失败页面提供重试且用户点击
- **THEN** 页面 SHALL 重新进入 loading 状态并发起一次新请求
- **AND** 成功后 SHALL 切换到 ready 或 empty

#### Scenario: 重试再次失败
- **WHEN** 新请求仍失败
- **THEN** 页面 SHALL 保持 error 状态并更新可展示错误
- **AND** MUST NOT 因重试次数而切换为 empty

### Requirement: 核心客户端页面必须纳入状态契约门禁
首页与 Bootstrap、对局大厅与历史、排行榜、通知、好友与 PK 邀约、赛事与赛季、球房、规则、统计与荣誉等发布范围内的数据页面 MUST 通过“成功有数据、成功空数据、网络失败、服务失败、权限/会话失败”契约检查。

#### Scenario: 页面 catch 后只记录日志
- **WHEN** 自动化检查发现页面在首次加载 catch 后仅记录日志并继续渲染空数组对应的空状态
- **THEN** 状态契约检查 MUST 失败

#### Scenario: 页面完成状态矩阵测试
- **WHEN** 页面对适用的成功、空数据和错误类别均有组件或纯逻辑测试
- **THEN** 该页面 MAY 通过错误态上线门禁
