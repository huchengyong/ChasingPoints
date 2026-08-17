# Auth-Safe Client Read Coordination

## Purpose

定义认证隔离的客户端资源复用、并发去重、TTL/SWR、页面生命周期和 WebSocket/前台恢复失效行为。

## Requirements

### Requirement: 用户级读缓存必须绑定认证身份
移动端 SHALL 以 `ownerUserId` 和当前 `authGeneration` 标识所有用户级资源状态，并 MUST 在 logout、账号合并、登录替换或认证代次变化时清除值、时间戳、dirty 状态和进行中的请求。

#### Scenario: 切换登录账号
- **WHEN** 用户级资源由账号 A 加载后，当前认证身份切换为账号 B
- **THEN** 系统 MUST 不得向账号 B 展示或返回账号 A 的缓存数据
- **AND** 账号 A 尚未完成的请求结果 MUST 不得写入账号 B 的 Store

#### Scenario: 同一账号继续会话
- **WHEN** 当前用户 ID 和 `authGeneration` 均未变化
- **THEN** 页面 SHALL 复用仍有效的用户级资源
- **AND** 不得仅因普通 Tab 切换清空缓存

### Requirement: 同一资源的并发读取必须复用请求
每个领域资源 Store SHALL 维护单个 `inFlight` 请求，同一身份、同一参数和同一资源的并发调用 MUST 共享该请求结果。

#### Scenario: App 与首页同时读取当前对局
- **WHEN** App 前台恢复和首页显示在首个当前对局请求完成前同时请求同一资源
- **THEN** 客户端 SHALL 只发起一个 HTTP 请求
- **AND** 两个调用方 SHALL 收到同一结果

#### Scenario: 参数不同的资源
- **WHEN** 两个请求的球种、分页、目标用户或位置桶不同
- **THEN** 系统 SHALL 使用不同资源键
- **AND** 不得错误复用不匹配的响应

### Requirement: 页面读取必须采用 loaded、dirty、TTL 和 SWR 规则
页面 SHALL 仅在资源未加载、被标记 dirty、超过资源 TTL 或用户强制刷新时请求；存在旧值且只是 TTL 到期时 SHALL 先展示旧值并后台刷新。

#### Scenario: 普通返回已有页面
- **WHEN** 页面从子页面返回且其资源仍在 TTL 内并且 `dirty = false`
- **THEN** 页面 MUST 直接复用现有数据
- **AND** 不得因 `onShow` 无条件重新请求

#### Scenario: 后台刷新失败
- **WHEN** 已有数据的 SWR 请求失败
- **THEN** 页面 SHALL 保留旧数据和可重试状态
- **AND** 不得把成功加载过的页面清空为默认值

#### Scenario: 用户手动下拉刷新
- **WHEN** 用户执行下拉刷新
- **THEN** 页面 SHALL 忽略普通 TTL 并发起强制刷新

### Requirement: App 前台恢复必须通过一次 Bootstrap 校准共享资源
登录后的冷启动、登录完成和 App 从后台恢复 SHALL 至多发起一次用户 Bootstrap，并以其结果更新用户资料、当前对局、通知角标、好友申请角标、换季提醒和竞技 revision。

#### Scenario: 从后台恢复后进入多个 Tab
- **WHEN** Bootstrap 已成功完成，随后用户依次进入首页、观赛和“我的”
- **THEN** 三个页面 SHALL 复用 Bootstrap 中的当前对局和角标
- **AND** 不得各自立即重复请求相同资源

#### Scenario: 后台期间错过 WebSocket
- **WHEN** 客户端断线期间服务端竞技 revision 已变化
- **THEN** 前台 Bootstrap SHALL 返回新 revision
- **AND** 对应竞技资源 Store SHALL 被标记 dirty 或重新加载

#### Scenario: Bootstrap 预约遇到 logout 或身份切换
- **WHEN** Bootstrap 预约的 in-flight Promise 尚未被响应，认证代次已变化或用户退出登录
- **THEN** Store SHALL 释放并 resolve 原预约等待者
- **AND** 新身份不得永久等待旧预约或接收旧 Bootstrap 回写

### Requirement: 实时失效不得依赖通知偏好
比赛结算、好友关系、通知计数和用户资料等资源变化 SHALL 通过专用用户数据事件或精确业务事件更新/失效；竞技数据失效 MUST 不依赖用户是否启用某类通知。

#### Scenario: 用户关闭比赛结果通知
- **WHEN** 排位比赛完成且用户关闭了比赛结果通知
- **THEN** 服务端仍 SHALL 发送与通知偏好无关的竞技数据更新事件
- **AND** 客户端 SHALL 失效段位、统计、H2H、对手、荣誉和赛季相关资源

#### Scenario: 收到计数事件
- **WHEN** 用户 WebSocket 消息携带最新未读通知数或好友申请数
- **THEN** 客户端 SHALL 直接更新对应 Store
- **AND** 更新 MUST 携带当前 `ownerUserId + authGeneration`
- **AND** 不得为每个 category 同时重新请求两个计数接口

#### Scenario: 保留在导航栈的私有页面重新显示
- **WHEN** stats、H2H、历史、对手、荣誉、通知或好友申请页面在账号切换后重新显示，或收到其精确 scope 的失效事件
- **THEN** 页面 SHALL 清除旧身份的局部数据，并只为当前身份重新读取
- **AND** 旧请求返回时 MUST 不得覆盖当前身份或更新后的 scope 版本

### Requirement: 核心页面不得因生命周期重复请求
排行榜首次进入、挑战状态切换、用户设置与资料编辑以及保留在导航栈中的详情页 SHALL 避免可确定的重复读取。

#### Scenario: 排行榜首次进入
- **WHEN** `onMounted` 和 `onShow` 在首次渲染周期内均触发
- **THEN** 排行榜 SHALL 只请求一次当前球种数据

#### Scenario: 切换挑战状态 Tab
- **WHEN** 待处理、已接受和已拒绝使用同一份挑战集合
- **THEN** 客户端 SHALL 在本地筛选已加载集合
- **AND** 不得因每次 Tab 切换重新下载同一集合

#### Scenario: 设置和编辑资料页打开
- **WHEN** 当前 userStore 已有匹配身份的最新用户资料
- **THEN** 页面 SHALL 使用该资料初始化
- **AND** 仅在资料缺失、dirty 或用户强制刷新时调用用户信息接口

### Requirement: 公共持久缓存与用户内存缓存必须分离
规则、术语、地区、段位配置和会员套餐等公共数据 MAY 按 app/schema version 持久化；统计、会员状态、信誉、当前对局和角标等用户数据在第一阶段 MUST 仅保存在认证隔离的内存 Store。

#### Scenario: App 升级或缓存 schema 变化
- **WHEN** 公共缓存记录的 app/schema version 与当前版本不一致
- **THEN** 客户端 SHALL 丢弃旧公共缓存并重新获取

#### Scenario: 用户退出登录
- **WHEN** 用户退出登录
- **THEN** 公共静态缓存 MAY 保留
- **AND** 所有用户级内存缓存 MUST 清空
