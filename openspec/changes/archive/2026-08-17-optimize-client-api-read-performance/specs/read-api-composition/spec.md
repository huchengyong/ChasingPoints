## ADDED Requirements

### Requirement: 用户 Bootstrap 必须提供共享首屏状态
系统 SHALL 提供认证用户 Bootstrap 接口，一次返回已校验用户资料、当前对局、未读通知数、待处理好友申请数、最近未读换季通知和竞技数据 revision。

#### Scenario: Bootstrap 成功
- **WHEN** 有效登录用户请求 Bootstrap
- **THEN** 响应 SHALL 包含所有可用共享资源及其 availability
- **AND** 用户资料 SHALL 复用认证中间件已校验的用户而不是再次查询同一主键

#### Scenario: 当前没有进行中对局
- **WHEN** 用户没有进行中的参赛或裁判对局
- **THEN** Bootstrap SHALL 成功返回空 current match
- **AND** 其他角标和用户资料仍 SHALL 可用

### Requirement: 用户概览必须合并同页稳定数据
系统 SHALL 提供用户概览接口，返回用户总体统计、信誉、会员状态和常玩球馆奖励状态；段位、当前对局和角标 MUST 保持由各自共享资源负责。

#### Scenario: 打开“我的”页
- **WHEN** 客户端需要渲染用户概览
- **THEN** 客户端 SHALL 至多调用一个用户概览接口取得这四个区块
- **AND** 不得再分别调用四个接口完成同一首屏

#### Scenario: 可选区块失败
- **WHEN** 会员配置暂时不可用但统计与信誉读取成功
- **THEN** 响应 SHALL 标记会员区块不可用
- **AND** 统计与信誉数据 MUST 仍可渲染

### Requirement: 竞技统计概览必须复用快照和有界趋势查询
系统 SHALL 提供按球种的统计概览接口，返回分球种摘要、近期趋势、段位趋势、最高分、时长和对手强度，并 MUST 从统一快照或有界索引查询构建。

#### Scenario: 首次打开竞技分析页
- **WHEN** 用户选择一个球种打开统计详情
- **THEN** 客户端 SHALL 只调用一个统计概览接口
- **AND** 服务端不得顺序执行六次历史全量统计

#### Scenario: 切换球种
- **WHEN** 用户从一个球种切换到另一个球种
- **THEN** 只 SHALL 加载新球种相关区块
- **AND** 与球种无关且仍有效的分球种总览不得重复读取

### Requirement: H2H 概览必须由单个接口返回首屏完整数据
系统 SHALL 提供 H2H overview 接口，一次返回目标对手资料、指定球种/月份的聚合统计、首屏历史、total 和分页游标或页码。

#### Scenario: 打开 H2H 页面
- **WHEN** 用户首次打开某对手的 H2H 页面
- **THEN** 客户端 SHALL 发起一个 overview 请求
- **AND** 不得再对相同月份分别请求 page size 10 和 page size 100 的历史数据

#### Scenario: 加载更多 H2H 历史
- **WHEN** 首屏历史不足且用户加载下一页
- **THEN** 客户端 SHALL 使用独立历史分页接口或 overview 返回的下一游标
- **AND** 已加载统计与对手资料不得重复请求

#### Scenario: 对手是访客身份
- **WHEN** H2H 目标没有注册用户 ID但具有规范化对手名称
- **THEN** overview SHALL 按稳定访客身份键返回一致的统计和历史

### Requirement: 对手候选必须直接返回去重后的业务数据
系统 SHALL 提供对手候选接口，以批量查询返回好友候选和有限数量的近期对手，不得要求客户端下载 100 个好友与 100 条比赛后自行去重。

#### Scenario: 打开对手选择页
- **WHEN** 用户创建新比赛并选择对手
- **THEN** 客户端 SHALL 从一个候选接口取得好友与近期对手
- **AND** 响应中的同一注册用户 MUST 只出现一次

### Requirement: 赛季首屏必须提供概览接口
系统 SHALL 提供赛季 overview，一次返回当前赛季状态、权威时间边界、指定球种本人记录和排行榜第一页；后续排行榜分页 SHALL 独立加载。

#### Scenario: 存在活动赛季
- **WHEN** 用户打开赛季页且存在有效窗口
- **THEN** overview SHALL 返回 `season_state = active`、当前赛季、本人实时记录和首屏榜单

#### Scenario: 赛季不可用
- **WHEN** 生命周期返回 `not_started` 或 `unavailable`
- **THEN** overview SHALL 返回准确状态和空赛季数据
- **AND** 不得继续查询本人记录或排行榜

### Requirement: 首页排行榜必须提供轻量摘要
系统 SHALL 提供排行榜摘要接口，只返回首页使用的前三名和当前用户排名；完整排行榜接口继续负责分页列表和 total。

#### Scenario: 首页加载榜单
- **WHEN** 首页只需要前三名和我的排名
- **THEN** 服务端 MUST 不得额外查询或返回第 4 至第 6 名及无用分页数据

### Requirement: 通知列表必须携带一致的未读数
通知列表首屏响应 SHALL 同时返回该用户的最新未读数，或者客户端 SHALL 使用 Bootstrap/WS 中具有相同 revision 的未读数，不得为同一首屏无条件再发 count 请求。

#### Scenario: 打开通知页
- **WHEN** 客户端加载通知列表第一页
- **THEN** 页面 SHALL 同时获得可用的未读数
- **AND** 不得立即重复调用未读 count 接口

### Requirement: 聚合接口必须保留部分可用和有界并行语义
聚合接口 SHALL 区分主资源与可选区块，独立查询 MAY 有界并行执行，但 MUST 不得共享可变事务或为列表元素创建无界并发。

#### Scenario: 一个可选区块超时
- **WHEN** 聚合接口的一个可选区块失败或超时
- **THEN** 响应 SHALL 返回其他成功区块和明确 availability/partial error
- **AND** 不得把整个页面降级为空

#### Scenario: 主资源失败
- **WHEN** 认证用户、H2H 目标或活动赛季等主资源无法确定
- **THEN** 接口 SHALL 返回整体失败或对应业务状态
- **AND** 不得返回可能属于错误身份或窗口的组合数据

### Requirement: 段位静态配置必须与用户当前段位解耦
系统 SHALL 提供可长期缓存的段位配置读取，并由客户端结合 rankStore 中的当前等级计算展示状态；静态配置读取不得为了标记 `is_current` 隐式初始化或查询用户段位。

#### Scenario: 打开段位说明页
- **WHEN** 客户端已有四球种段位 Store 和有效段位配置缓存
- **THEN** 页面 SHALL 直接组合两者渲染
- **AND** 不得同时调用单球种段位和个性化段位配置列表接口

### Requirement: 实体详情必须优先复用已加载实体并提供直接回源
从列表或荣誉墙进入的详情页 SHALL 优先复用已加载实体；深链或缓存缺失时 SHALL 使用按 ID 的直接详情读取，不得下载完整列表后在客户端查找单项。

#### Scenario: 从荣誉墙打开成就详情
- **WHEN** 荣誉墙缓存已包含目标成就
- **THEN** 详情页 SHALL 使用该实体立即渲染
- **AND** 不得重新请求完整成就列表

#### Scenario: 深链打开成就详情
- **WHEN** 客户端没有目标成就缓存
- **THEN** 客户端 SHALL 调用按 ID 的成就详情接口
- **AND** 服务端 SHALL 只查询目标定义和当前用户进度

### Requirement: 旧细粒度接口必须复用同一读模型并保持迁移兼容
聚合接口上线期间，现有细粒度接口 SHALL 保持可用并从相同快照/投影读取，避免新旧接口统计口径分叉。

#### Scenario: 旧客户端请求用户统计
- **WHEN** 尚未升级的客户端调用现有用户统计接口
- **THEN** 接口 SHALL 返回与用户概览相同快照 revision 下的统计结果

### Requirement: 可公开访问且可个性化的读取必须安全解析可选身份
排行榜、排行榜摘要和公开对局 SHALL 允许匿名访问；携带有效 access token 时 MUST 通过统一可选 JWT 中间件将已验证用户身份传入业务逻辑，且不得由 handler 再次以宽松规则解析 token。

#### Scenario: 匿名读取公开榜单
- **WHEN** 请求未携带 Authorization
- **THEN** 公开榜单和摘要 SHALL 正常返回公共部分
- **AND** `my_ranking` SHALL 为空

#### Scenario: 登录用户读取个性化公开数据
- **WHEN** 请求携带有效 access token
- **THEN** 排行榜和摘要 SHALL 按已验证 viewer 组装其 `my_ranking`
- **AND** `scope=friends` 的公开对局 SHALL 按该 viewer 的好友关系过滤

#### Scenario: 无效的公开请求令牌
- **WHEN** 请求携带无效、过期、错误签名或非 access 类型 token
- **THEN** 中间件 MUST 不得向业务逻辑注入任何用户身份
- **AND** 请求 SHALL 被安全拒绝或按明确匿名策略处理，不得把未验证 claim 用于个性化响应
