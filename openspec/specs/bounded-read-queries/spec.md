# Bounded Read Queries

## Purpose

定义在线读取必须遵守的有界索引分页、批量关联、纯读边界、完成时间索引和查询验收约束。

## Requirements

### Requirement: 在线历史查询必须使用有界索引访问
所有在线列表、趋势、统计和候选查询 SHALL 使用可验证索引、数据库 LIMIT/游标或固定数量快照行，并 MUST 不得先加载全部历史再在 Go 中过滤、翻转或分页。

#### Scenario: 请求对局历史第一页
- **WHEN** 用户拥有任意数量历史比赛
- **THEN** 数据库 SHALL 只返回请求页及必要 total
- **AND** 应用内存占用不得与全部历史场次线性增长

#### Scenario: 请求最近 N 场趋势
- **WHEN** 客户端请求最近 N 场
- **THEN** SQL SHALL 包含稳定排序和上限 N
- **AND** 不得读取 N 之外的历史用于应用层切片

### Requirement: 列表关联数据必须批量加载并消除 N+1
好友、好友申请、挑战、球馆、赛季榜单、裁判历史和公开比赛列表 SHALL 通过 JOIN、聚合子查询或有限 `IN` 批量读取关联数据；SQL 次数 MUST 不随返回条数线性增加。

#### Scenario: 获取 100 个好友
- **WHEN** 好友列表返回 100 条
- **THEN** 用户资料和段位 SHALL 通过固定数量查询取得
- **AND** 不得为每个好友分别查询 user 和 ranking

#### Scenario: 获取球馆列表
- **WHEN** 球馆列表返回 N 家球馆
- **THEN** 签到数 SHALL 在列表查询或单个批量聚合中取得
- **AND** 不得执行 N 次 count

#### Scenario: 获取公开比赛列表
- **WHEN** 公开比赛列表返回 N 场
- **THEN** 玩家资料和已完成局数 SHALL 通过 JOIN/聚合一次取得
- **AND** 不得逐场查询 round count

### Requirement: 计数接口不得借用完整列表
未读通知、待处理好友申请和其他角标 SHALL 使用专用 COUNT、已维护计数或聚合响应，不得通过请求完整列表并丢弃内容来取得 total。

#### Scenario: 获取好友申请角标
- **WHEN** 页面只需要待处理申请数量
- **THEN** 系统 SHALL 执行 count 或读取已维护计数
- **AND** 不得加载全部申请和用户资料

### Requirement: 附近球馆必须先执行可索引候选预筛选
附近球馆查询 SHALL 先使用经纬度边界框、位置桶或标准空间索引缩小候选，再对有限候选计算精确距离和排序。

#### Scenario: 城市中存在大量球馆
- **WHEN** 用户请求 5 公里内前 20 家球馆
- **THEN** 数据库 SHALL 使用可索引条件排除边界框外记录
- **AND** Haversine 计算不得作用于整张球馆表

### Requirement: 热路径 GET 必须是业务纯读
规则、挑战、段位、赛事对阵、当前对局和奖励摘要等 GET SHALL 不再承担全局播种、全局过期、缺行初始化、对阵生成或成就补偿写入；状态维护 SHALL 移到 migration、用户初始化、显式命令或有界异步 Worker。

#### Scenario: 读取规则
- **WHEN** 客户端请求规则内容或术语
- **THEN** 请求 SHALL 只读取已初始化内容
- **AND** 不得每次执行 SeedData

#### Scenario: 读取待处理挑战
- **WHEN** 客户端请求挑战列表
- **THEN** 查询 SHALL 按 `expires_at` 返回有效业务状态
- **AND** 不得先执行无范围全局 UPDATE

#### Scenario: 读取段位信息
- **WHEN** 用户请求段位
- **THEN** GET SHALL 读取已初始化段位行
- **AND** 缺失修复不得隐式发生在热读路径

#### Scenario: 读取奖励摘要
- **WHEN** 成就同步尚未完成
- **THEN** GET SHALL 返回 pending/availability
- **AND** 补偿同步 SHALL 由独立可重试任务处理

### Requirement: 统一完成时间必须使用可索引字段
需要按比赛完成时间过滤或排序的热路径 SHALL 使用持久化且有索引的完成时间字段，不得长期依赖 `COALESCE(end_time, match_time)` 或其他包裹索引列的表达式。

#### Scenario: 查询赛季范围比赛
- **WHEN** 系统按半开时间窗口读取完成比赛
- **THEN** SQL SHALL 使用可索引 completed time 范围
- **AND** 查询计划不得退化为全表扫描

### Requirement: 认证中间件读取结果必须在请求内复用
活动用户中间件成功校验用户后 SHALL 将必要用户对象或快照放入 Context，同一请求内的 Logic SHALL 优先复用。

#### Scenario: 请求用户 Bootstrap
- **WHEN** 中间件已经读取并确认用户有效
- **THEN** Bootstrap SHALL 不得再次按同一用户主键读取用户表

#### Scenario: 管理端停用用户
- **WHEN** 新请求到达且用户已被停用
- **THEN** 中间件 SHALL 仍按既定安全语义拒绝请求
- **AND** 请求内复用不得跨请求缓存活动状态

### Requirement: 查询优化必须有 EXPLAIN 和查询次数验收
每个已识别热点 SHALL 具备代表性查询计划验证和 query-count 测试，确保结果条数或历史数据增长不会造成 SQL 次数线性增长。

#### Scenario: N+1 回归测试
- **WHEN** 测试数据从 1 条扩展到 100 条列表记录
- **THEN** 接口 SQL 次数 SHALL 保持在定义上限内

#### Scenario: 索引未被使用
- **WHEN** 代表性 MySQL EXPLAIN 显示热路径使用全表扫描或 filesort 超出预期
- **THEN** 该优化任务 MUST 不得标记完成

### Requirement: 在线逻辑不得使用非标准或无界并行规避查询问题
实现 SHALL 使用项目已有 MySQL、Redis、Gorm 和 go-zero 标准能力；并行只允许用于固定数量、彼此独立的查询，且 MUST 有并发上限。

#### Scenario: 聚合列表元素
- **WHEN** 聚合接口返回 N 条列表
- **THEN** 服务端 MUST 不得为每条记录启动独立 goroutine 查询关联数据
- **AND** 应先使用 JOIN 或批量查询

#### Scenario: 历史修复任务
- **WHEN** 离线任务必须遍历历史数据
- **THEN** 它 SHALL 使用索引游标和固定 batch size
- **AND** 不得一次性 `Find` 全表到内存
