## MODIFIED Requirements

### Requirement: 用户 Bootstrap 必须提供共享首屏状态
系统 SHALL 提供认证用户 Bootstrap 接口，一次返回已校验用户资料、当前对局、当前有效约球摘要、收到待回应邀请数、服务端当前时刻、未读通知数、待处理好友申请数、最近未读换季通知和竞技数据 revision。约球摘要与数量 MUST 使用有界、纯读查询，区分「没有约球」与「约球数据不可用」，供两个主 Tab 复用；已开始的比赛仍以当前对局为优先展示。

#### Scenario: Bootstrap 成功
- **WHEN** 有效登录用户请求 Bootstrap
- **THEN** 响应 SHALL 包含所有可用共享资源及其 availability
- **AND** 用户资料 SHALL 复用认证中间件已校验的用户而不是再次查询同一主键

#### Scenario: 当前没有进行中对局
- **WHEN** 用户没有进行中的参赛或裁判对局
- **THEN** Bootstrap SHALL 成功返回空 current match
- **AND** 其他角标和用户资料仍 SHALL 可用

#### Scenario: 双入口恢复有效约球
- **WHEN** 登录用户有一场已接受或等待中的约球或自己发出的有效待回应邀请，且另有多条收到未回应邀请
- **THEN** Bootstrap SHALL 返回权威当前约球摘要和收到邀请数量，两个主 Tab SHALL 用同一份数据渲染，收到邀请不得覆盖当前活动

#### Scenario: 主页面依据服务端时间显示预约日期
- **WHEN** 用户设备时区/时钟与北京时间不一致而打开对局或我的页
- **THEN** Bootstrap SHALL 返回服务端当前时刻供客户端计算预约相对日期与失效文案；是否可进入仍由服务端判定

#### Scenario: 约球区块读取失败
- **WHEN** 当前约球查询失败而当前对局及其他区块仍可用
- **THEN** Bootstrap SHALL 标记约球区块不可用并返回其他成功区块
- **AND** 客户端 MUST 不得把约球未知状态渲染为「无约球」
