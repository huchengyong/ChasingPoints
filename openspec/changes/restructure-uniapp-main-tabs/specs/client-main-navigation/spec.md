## ADDED Requirements

### Requirement: 主导航必须由四个 Tab 与中间扫码按钮组成
UniApp 在微信小程序与鸿蒙端 SHALL 按「首页、赛讯、扫码、消息、我的」的顺序展示底部主导航。四个 Tab 依次对应 `pages/index/index`、`pages/tournament/index`、`pages/message/index`、`pages/user/index`；中间扫码按钮位于赛讯与消息之间，不对应任何页面，点击后 SHALL 启动 `universal-scan-entry` 规定的扫码流程，且不改变当前选中的 Tab。两端 SHALL 使用相同的 Tab 文字、图标、选中色和角标位置，并跟随用户的主题偏好（跟随系统 / 浅色 / 深色）。中间扫码按钮 SHALL 不超出 TabBar 高度，在两种主题下都可辨识；在支持为它设置可访问名称的端（微信小程序自定义 TabBar），SHALL 提供「扫码」的可访问名称。

#### Scenario: 小程序切换到消息 Tab
- **WHEN** 用户在微信小程序点击底部「消息」
- **THEN** 应用 SHALL 切换到 `pages/message/index`，自定义 TabBar 显示「消息」为选中态
- **AND** 角标与主题 SHALL 与当前账号、当前主题一致

#### Scenario: 鸿蒙点击中间扫码按钮
- **WHEN** 用户在鸿蒙端点击 TabBar 中间按钮
- **THEN** 应用 SHALL 启动一次扫码流程，当前选中的 Tab 保持不变

#### Scenario: 用户切换为深色主题
- **WHEN** 用户在设置中改为深色后回到任一 Tab 页
- **THEN** 两端的 TabBar 背景、文字和图标 SHALL 使用深色主题资源，中间扫码按钮仍清晰可见

### Requirement: TabBar 不得遮挡页面内容与弹层操作
微信小程序的自定义 TabBar 覆盖在页面底部，四个 Tab 页 SHALL 为它预留 TabBar 高度和底部安全区；页面内固定在底部的元素 SHALL 显示在 TabBar 上方。Tab 页打开会覆盖底部区域的弹层时，SHALL 通过统一的显隐方法隐藏 TabBar，在弹层关闭或页面卸载时恢复。鸿蒙原生 TabBar 不占用页面区域，统一显隐方法在鸿蒙上 SHALL 调用原生 TabBar 的显隐。

#### Scenario: 首页滚动到底部
- **WHEN** 用户在小程序首页滚动到最后一个区块
- **THEN** 该区块 SHALL 完整显示在 TabBar 上方，不被遮挡

#### Scenario: 出示二维码弹层
- **WHEN** 用户在「我的」或首页打开「出示我的二维码」弹层
- **THEN** 两端 SHALL 隐藏 TabBar，弹层按钮不与 TabBar 重叠
- **AND** 弹层关闭或页面卸载后 TabBar SHALL 恢复显示

### Requirement: 观赛必须并入首页
首页 SHALL 在开局卡和约球卡之后展示「正在进行」区块，最多列出 3 场公开进行中对局，且不包含本人的当前对局；区块标题旁 SHALL 始终提供「观赛大厅」入口，没有对局或读取失败时也不例外。点击卡片 SHALL 沿用观赛列表的进入规则：本人参与的进行中对局进入计分页，其他对局进入观战页。预览数据与访问者无关，SHALL 走短 TTL 的公共缓存，在 TTL 内切回首页不得重复请求；下拉刷新 SHALL 强制刷新。预览读取失败 SHALL 只影响本区块，并提供重试。

#### Scenario: 有公开进行中对局
- **WHEN** 平台有多场公开进行中对局，其中一场是本人的当前对局
- **THEN** 首页 SHALL 展示最多 3 场其他对局，本人对局只出现在开局卡

#### Scenario: 没有进行中对局
- **WHEN** 当前没有公开进行中对局
- **THEN** 区块 SHALL 显示空态说明，并保留「观赛大厅」入口

#### Scenario: TTL 内切回首页
- **WHEN** 用户在预览缓存有效期内从其他 Tab 切回首页
- **THEN** 首页 SHALL 复用缓存，不得再次请求公开对局列表

#### Scenario: 预览读取失败
- **WHEN** 公开对局列表请求失败，而赛事、排行榜等区块读取成功
- **THEN** 只有「正在进行」区块 SHALL 显示失败与重试，其他区块正常展示

### Requirement: 观赛大厅只承担观赛列表
`pages/match/index` SHALL 改为普通页面「观赛大厅」，从首页入口进入。它保留好友 / 进行中 / 已结束标签（未登录时不显示好友标签）、球种筛选、分页加载、下拉刷新和既有的进入规则；SHALL NOT 再展示开局卡、约球卡、约球入口、扫码入口、出示二维码和裁判确认弹层。本人参与的对局 SHALL 留在列表中并标注「我」。任何代码 SHALL NOT 再用 `switchTab` 打开该路径。

#### Scenario: 从首页进入观赛大厅
- **WHEN** 用户点击首页「观赛大厅」
- **THEN** 应用 SHALL 以普通页面打开 `pages/match/index`，返回时回到首页

#### Scenario: 列表包含本人对局
- **WHEN** 好友标签中有一场本人参与的进行中对局
- **THEN** 该对局 SHALL 以「我」标识显示，点击后进入计分页

### Requirement: 原观赛页入口必须迁移到新位置
「收到的邀请」SHALL 由消息 Tab 提供；「约球记录」SHALL 由「我的」提供，并直接打开约球页的记录标签；扫码开局和扫码执裁 SHALL 由中间扫码按钮提供。首页「发起对局」SHALL 继续提供「扫描对手二维码」「出示我的二维码」「选择好友或最近对手」三项。「我的」头部 SHALL 保留「出示我的二维码」和设置，移除扫码按钮与通知铃铛；首页头部 SHALL NOT 再展示消息图标或未读提示。约球发送成功后 SHALL 回到首页。

#### Scenario: 从「我的」查看约球记录
- **WHEN** 用户在「我的」点击「约球记录」
- **THEN** 应用 SHALL 打开约球页并选中记录标签

#### Scenario: 发送约球后返回
- **WHEN** 用户在发起约球页发送成功
- **THEN** 应用 SHALL 切换到首页，首页约球卡按标脏后的活动状态刷新

#### Scenario: 首页头部
- **WHEN** 已登录用户在任一端打开首页
- **THEN** 首页头部 SHALL 只展示标题与问候语，没有消息图标或未读文字

### Requirement: 消息 Tab 必须聚合通知、好友申请与约球邀请
`pages/message/index` SHALL 是主包中的 Tab 页。已登录时，页面顶部展示「好友申请」和「约球邀请」两个入口及各自的待处理数量，分别打开好友申请页和约球页的「收到」标签；下方展示通知列表，保留单条已读、全部已读、删除、分页、按通知数据跳转和下拉刷新。两个数量 SHALL 取自共享的活动快照（Bootstrap、活动 Store 及 WebSocket 更新），页面不得为它们单独发计数请求；通知列表首屏 SHALL 使用自身响应里的未读数。未登录时 SHALL 展示登录引导，不发起任何用户级请求。原 `subPages/notification/index` SHALL 删除，应用内不得再有指向它的跳转。

#### Scenario: 登录用户首次进入消息 Tab
- **WHEN** Bootstrap 已完成，用户第一次切到消息 Tab
- **THEN** 页面 SHALL 只请求一次通知列表首屏，两个入口的数量取自活动快照

#### Scenario: 未登录进入消息 Tab
- **WHEN** 未登录用户切到消息 Tab
- **THEN** 页面 SHALL 展示登录引导，且不请求通知、好友申请或约球数据

#### Scenario: 切换账号后回到消息 Tab
- **WHEN** 用户退出并登录另一个账号后回到消息 Tab
- **THEN** 旧账号的列表和数量 SHALL 被清空，旧账号的迟到响应不得写入页面

### Requirement: 消息 Tab 常驻后必须按失效刷新
消息 Tab 再次显示时，SHALL 只在身份变化或通知 scope 失效后刷新列表首屏，否则复用已加载的列表，不得在每次显示时无条件回源。页面可见期间收到通知失效时 SHALL 刷新首屏，短时间内的多次失效合并为一次请求。

#### Scenario: 在其他 Tab 时收到新通知
- **WHEN** 用户停留在首页时收到新通知的 WebSocket 事件，之后切到消息 Tab
- **THEN** 消息 Tab SHALL 刷新通知首屏并显示新通知

#### Scenario: 停留在消息 Tab 时连续收到通知
- **WHEN** 用户停留在消息 Tab 时，短时间内连续收到多条通知失效事件
- **THEN** 页面 SHALL 只刷新一次首屏，并显示这些新通知

#### Scenario: 没有变化时切回
- **WHEN** 没有身份变化或通知失效，用户从其他 Tab 切回消息 Tab
- **THEN** 页面 SHALL 不发起通知列表请求

### Requirement: 消息角标只表达一种计数
消息 Tab 的角标 SHALL 按以下规则显示：已登录且未读通知数大于 0 时显示该数，超过 99 显示「99+」；未读通知为 0，但待处理好友申请数或收到的待回应约球邀请数大于 0 时显示红点；其他情况及未登录时不显示。角标数据 SHALL 绑定当前 `userId + authGeneration`，退出或切换账号后旧计数不得显示。计数变化时，若当前页面是 Tab 页 SHALL 立即更新角标，否则在下一次显示 Tab 页时更新。

#### Scenario: 有未读通知
- **WHEN** 当前账号有 3 条未读通知和 1 条待处理好友申请
- **THEN** 消息角标 SHALL 显示「3」

#### Scenario: 只有待处理事项
- **WHEN** 未读通知为 0，但有收到的待回应约球邀请
- **THEN** 消息角标 SHALL 显示红点

#### Scenario: 未读超过 99
- **WHEN** 未读通知数为 120
- **THEN** 消息角标 SHALL 显示「99+」

#### Scenario: 在计分页时计数变化
- **WHEN** 用户在计分页时收到新通知，之后返回 Tab 页
- **THEN** 返回后显示的角标 SHALL 等于最新计数

#### Scenario: 退出登录
- **WHEN** 用户退出登录
- **THEN** 消息角标 SHALL 立即不再显示旧账号计数
