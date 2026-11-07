# Performance Observability

## Purpose

定义运行配置单一真源、HTTP/SQL/缓存性能观测、阶段基线和量化回归门槛。

## Requirements

### Requirement: 后端运行配置必须具有单一真源
主服务、开发命令、运维文档和辅助命令 SHALL 使用 `backend/etc/chasing_points-api.yaml` 作为权威运行配置；未被引用的同名变体 MUST 被移除或明确标记为不可用于运行。

#### Scenario: 启动后端服务
- **WHEN** 使用仓库标准命令启动服务
- **THEN** 加载的日志 Encoding、Path、KeepDays、Level 和 Stat SHALL 来自权威配置
- **AND** 不得因两个配置文件参数不同产生环境漂移

### Requirement: 每个 HTTP 请求必须产生标准性能观测
服务 SHALL 使用 go-zero 标准能力或等价低开销中间件记录规范化 route、method、status、duration、response bytes 和 request id，并 SHALL 支持按 route 聚合 QPS、错误率和 P50/P95/P99。

#### Scenario: 请求成功或失败
- **WHEN** 任意 HTTP 请求完成
- **THEN** 系统 SHALL 产生一条结构化请求完成观测
- **AND** 动态 ID、Token 或完整查询敏感值不得成为 route 标签

#### Scenario: 使用 JSON 文件日志
- **WHEN** 日志 Mode 为 file 且 Encoding 为 json
- **THEN** 请求性能字段 SHALL 写入配置 Path 下的实际日志文件
- **AND** 运维 SHALL 能验证文件非空及轮转生效

### Requirement: 数据库访问必须可统计和定位慢查询
系统 SHALL 记录每请求 SQL 次数与总耗时，并对超过阈值的 SQL 记录规范化模板、耗时和关联 request id；测试环境 SHALL 能断言指定接口查询次数上限。

#### Scenario: 接口发生 N+1
- **WHEN** 列表条数增加导致 SQL 次数增加
- **THEN** query-count 测试 SHALL 失败或观测 SHALL 明确暴露该线性增长

#### Scenario: 慢查询日志
- **WHEN** SQL 耗时超过配置阈值
- **THEN** 系统 SHALL 记录其规范化查询与耗时
- **AND** 不得记录 Token、短信验证码、原始数据库错误值或其他敏感数据

#### Scenario: 认证和业务 SQL 属于同一 HTTP 请求
- **WHEN** 有效认证请求先校验活跃用户、再执行业务 Model 查询
- **THEN** 认证查询和业务查询 SHALL 都继承请求 Context 并计入同一 `sql_count`
- **AND** 该行为 MUST 由真实 HTTP 中间件链测试覆盖，不得仅靠手工向 Gorm DB 注入 Context

#### Scenario: Handler panic
- **WHEN** Handler 或其下游中间件发生 panic
- **THEN** 请求观测 SHALL 在 panic 向外传播前记录一次 status 500 的完成事件
- **AND** 外层恢复处理不得导致该请求缺失完成观测

### Requirement: 缓存必须暴露命中与回源指标
每个服务端缓存领域 SHALL 记录 hit、miss、decode error、Redis error、write error 和 database fallback；客户端 SHALL 能统计页面复用命中和实际网络请求数。

#### Scenario: Redis 故障回源
- **WHEN** Redis 读取失败但数据库读取成功
- **THEN** 响应 SHALL 成功
- **AND** 指标 SHALL 同时增加 Redis error 与 fallback

#### Scenario: 客户端复用赛讯
- **WHEN** 页面命中客户端赛讯缓存而不发网络请求
- **THEN** 客户端观测 SHALL 记录一次本地命中和零次对应 HTTP 请求

### Requirement: 每个实施阶段必须以同一基线验收
阶段 0 SHALL 记录优化前核心路径数据；后续每阶段 SHALL 使用相同数据规模和场景比较页面请求数、SQL 次数、P95/P99、响应大小和缓存命中率。

#### Scenario: 客户端第一阶段完成
- **WHEN** loaded/dirty/TTL 和 single-flight 上线
- **THEN** 验收 SHALL 比较 App 冷启动、前台恢复、首页、观赛、赛讯和“我的”的实际请求数

#### Scenario: 读模型阶段完成
- **WHEN** 用户统计、H2H 和历史接口切换到新模型
- **THEN** 验收 SHALL 证明 SQL 次数不随用户历史场次增长
- **AND** 新旧结果抽样一致

#### Scenario: 缓存阶段完成
- **WHEN** 排行榜、静态内容或对局核心缓存上线
- **THEN** 验收 SHALL 同时报告命中率和缓存 miss 的冷请求性能

### Requirement: 指标阈值必须在基线后固化为回归门槛
由于当前缺少有效 access 样本，具体 P95/P99 数值 MAY 在阶段 0 后确定；一旦确定，核心接口和页面请求数上限 MUST 写入测试或发布检查清单。

#### Scenario: 阈值确认
- **WHEN** 阶段 0 获得代表性流量或压测数据
- **THEN** 团队 SHALL 为 Bootstrap、用户概览、统计概览、H2H、排行榜和历史列表设置目标
- **AND** 后续任务不得仅以“已加缓存”作为完成标准

### Requirement: 可观测性开销必须受控
性能观测 SHALL 使用有界字段、慢查询阈值和必要采样，且 MUST 不得显著改变被测接口延迟或生成无界日志。

#### Scenario: 高 QPS 路由
- **WHEN** 路由达到高请求量
- **THEN** 标签基数 SHALL 受规范化 route 控制
- **AND** 日志保留期与轮转 SHALL 按权威配置执行
