## ADDED Requirements

### Requirement: 我的页面提供对局记录入口

原生“我的” SHALL 仅在会话已具有有效用户资料时显示“对局记录”入口，进入包含“参赛”“执裁”两个分栏的对局记录页。访客、恢复中及恢复失败状态 MUST NOT 显示该入口或发起对局记录读取。

#### Scenario: 已登录用户进入对局记录

- **WHEN** 已登录用户点击“对局记录”
- **THEN** 客户端 SHALL 打开对局记录页并默认显示“参赛”分栏

#### Scenario: 访客不显示入口

- **WHEN** 当前没有已验证用户资料
- **THEN** “我的” MUST NOT 显示对局记录入口，也不调用对局记录接口

### Requirement: 对局记录分栏按需分页加载

参赛分栏 SHALL 通过 `GET /api/match/list`、执裁分栏 SHALL 通过 `GET /api/match/referee/history` 读取，每页 20 条，两个分栏状态互相独立。客户端 MUST 只在分栏首次显示、用户下拉刷新或滚动到底部且仍有更多数据时请求，普通返回与分栏切换不得重复请求已加载的分栏。

#### Scenario: 首次进入只加载当前分栏

- **WHEN** 用户首次进入对局记录页
- **THEN** 客户端 SHALL 只请求参赛第一页，直到用户切换到执裁分栏才请求执裁第一页

#### Scenario: 返回与切换不重复请求

- **WHEN** 用户从战报返回列表，或在两个已加载的分栏之间切换
- **THEN** 客户端 MUST NOT 重新请求已加载的分栏，并保留原有列表与位置

#### Scenario: 加载更多

- **WHEN** 用户滚动到底部且已加载条数小于服务端 `total`，并且当前没有加载、刷新或加载更多在进行
- **THEN** 客户端 SHALL 请求下一页并追加结果
- **AND** 已加载条数达到 `total` 时 SHALL 显示没有更多，不再请求

#### Scenario: 下拉刷新

- **WHEN** 用户下拉刷新某个分栏
- **THEN** 客户端 SHALL 重新请求第一页，成功后用新结果替换该分栏列表

### Requirement: 对局记录区分空态、错误与刷新失败

客户端 MUST 区分无数据、首次加载失败、刷新失败和加载更多失败。HTTP 失败、网络失败及 `success: false` MUST NOT 显示为“暂无对局记录”。

#### Scenario: 无对局记录

- **WHEN** 分栏第一页成功返回空列表
- **THEN** 客户端 SHALL 显示该分栏的空态说明

#### Scenario: 首次加载失败

- **WHEN** 分栏第一页请求失败或返回 `success: false`
- **THEN** 客户端 SHALL 显示整页错误与重试操作，重试 SHALL 重新请求第一页

#### Scenario: 刷新失败保留已有数据

- **WHEN** 分栏已有数据且下拉刷新失败
- **THEN** 客户端 SHALL 保留已有列表并显示可重试的刷新失败提示

#### Scenario: 加载更多失败

- **WHEN** 加载更多失败
- **THEN** 客户端 SHALL 保留已加载数据、不推进页码，并允许再次加载该页

### Requirement: 对局记录行展示服务端结果

参赛行 SHALL 显示对手头像与昵称、球种、相对时间、服务端 `result` 对应的胜利/失败/平局及“我方比分 - 对手比分”。执裁行 SHALL 显示双方选手、比分、球种和状态。结果 MUST 同时用文字表达，不只依赖颜色。

#### Scenario: 参赛平局

- **WHEN** 参赛记录的 `result` 为 3
- **THEN** 该行 SHALL 显示“平局”

#### Scenario: 头像缺失或加载失败

- **WHEN** 头像地址为空、无效或加载失败
- **THEN** 该行 SHALL 显示稳定占位图，其他内容正常显示

#### Scenario: 已取消的执裁记录

- **WHEN** 执裁记录状态为已取消
- **THEN** 该行 SHALL 显示“已取消”且 MUST NOT 可进入战报

### Requirement: 战报以只读方式展示已完成对局

参赛记录与已完成的执裁记录 SHALL 进入战报，战报通过 `GET /api/match/detail` 读取一次。加载失败 MUST 显示错误与重试，不得以默认比分或默认对手代替。战报 MUST NOT 显示本期未迁移功能的入口（再来一局、战绩海报、交锋记录、荣誉墙）。

#### Scenario: 选手视角战报

- **WHEN** 详情的 `viewer_role` 为选手
- **THEN** 战报 SHALL 显示双方头像昵称、最终比分、胜负平结论、球种、分差、结束时间、我方与对手排位分变化及我方排位明细、亮点与统计，以及存在时的裁判卡与完成方式

#### Scenario: 平局战报

- **WHEN** 已完成对局的我方比分与对手比分相等
- **THEN** 战报 SHALL 显示平局结论，不显示为失利

#### Scenario: 裁判视角战报

- **WHEN** 详情的 `viewer_role` 为 `referee`
- **THEN** 战报 SHALL 显示双方选手比分、裁判卡与完成方式
- **AND** MUST NOT 显示排位分变化、荣誉奖励或胜负评价文案，也不请求奖励摘要

#### Scenario: 战报加载失败

- **WHEN** 详情请求失败、被拒绝或返回 `success: false`
- **THEN** 战报 SHALL 显示错误与重试，MUST NOT 展示 `0 - 0` 等默认数据

#### Scenario: 时间无法解析

- **WHEN** 结束时间为空或无法解析
- **THEN** 战报 SHALL 显示“时间待同步”

### Requirement: 战报荣誉奖励独立读取且不阻塞战报

选手视角的已完成对局 SHALL 通过 `GET /api/match/reward-summary` 读取一次奖励摘要。奖励结果 MUST NOT 阻塞或替换战报其他内容。

#### Scenario: 本场解锁奖励

- **WHEN** 奖励摘要返回 `ready` 且列表非空
- **THEN** 战报 SHALL 展示解锁的成就名称、描述、图标与可选称号

#### Scenario: 无奖励

- **WHEN** 奖励摘要返回 `ready` 且列表为空
- **THEN** 战报 MUST NOT 显示奖励区

#### Scenario: 奖励处理中

- **WHEN** 奖励摘要返回 `pending`
- **THEN** 战报 SHALL 显示奖励仍在处理中的说明，本期不自动轮询

#### Scenario: 奖励读取失败

- **WHEN** 奖励摘要请求失败或返回 `success: false`
- **THEN** 战报 SHALL 隐藏奖励区，其余内容保持正常

### Requirement: 对局记录与战报读取绑定当前身份

列表与战报的每个读取结果 MUST 绑定发起时的用户与认证代次。退出、切号或会话失效后，旧身份的成功、失败及加载状态 MUST NOT 显示给新身份；认证失败 SHALL 复用既有刷新、一次重放与失效清理规则，页面不重复弹出全局重新登录提示。

#### Scenario: 读取中退出或切号

- **WHEN** 用户 A 的列表或战报请求尚未返回时退出或登录 B，随后 A 的结果到达
- **THEN** 客户端 MUST 丢弃该结果，不向 B 展示 A 的记录、错误或加载状态

#### Scenario: 已加载列表遇到身份变化

- **WHEN** 对局记录页已显示 A 的数据，随后会话失效、退出或切换为 B
- **THEN** 客户端 MUST 清空两个分栏的数据与错误；无用户时显示登录引导，为 B 重新加载时只展示 B 的结果

#### Scenario: 同一分栏的迟到响应

- **WHEN** 同一分栏存在较早的请求晚于较新的请求返回
- **THEN** 客户端 MUST 只采用最新请求的结果
