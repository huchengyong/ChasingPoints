## ADDED Requirements

### Requirement: 已完成比赛必须生成双方参与者读投影
每场有效完成比赛 SHALL 为每个注册参赛用户生成一条面向该用户视角的参与者投影，并以 `match_id + user_id` 保证唯一和幂等。

#### Scenario: 双方注册用户完成比赛
- **WHEN** 一场比赛成功完成
- **THEN** 系统 SHALL 在同一业务事务中写入两条参与者投影
- **AND** 每条 SHALL 包含该用户视角的结果、比分、对手、球种、模式、完成时间和比赛级统计

#### Scenario: 重复提交 Finish
- **WHEN** 同一比赛因客户端重试或多实例竞争重复执行完成逻辑
- **THEN** 每个 `match_id + user_id` MUST 仍只有一条投影
- **AND** 统计增量 MUST 不得重复应用

#### Scenario: 访客对手比赛
- **WHEN** 对手没有注册用户 ID
- **THEN** 发起方投影 SHALL 保存规范化访客身份键
- **AND** 系统不得为不存在的注册用户创建用户统计快照

### Requirement: 用户竞技统计必须在比赛结算时增量维护
系统 SHALL 在比赛成功结算时维护用户总体和分球种统计快照，包括总场次、胜负、当前/最大连胜、比赛最高分、单杆最高分、时长聚合、最后比赛和单调 revision；在线用户统计接口 MUST 不得扫描全部比赛历史。

#### Scenario: 用户赢得一场排位
- **WHEN** 有效排位比赛首次应用到用户快照
- **THEN** 对应总体行和球种行 SHALL 原子增加总场次与胜场
- **AND** 当前连胜、最大连胜、最高分、时长和 revision SHALL 按该场结果更新

#### Scenario: 用户输掉一场排位
- **WHEN** 有效排位比赛首次应用到用户快照
- **THEN** 负场 SHALL 增加且当前连胜 SHALL 重置
- **AND** 其他累计字段 SHALL 保持正确

#### Scenario: 读取用户统计
- **WHEN** 用户统计、分球种统计、最高分或时长接口被调用
- **THEN** 系统 SHALL 从固定数量的快照行读取
- **AND** SQL 数量不得随用户历史场次增长

### Requirement: H2H 与对手统计必须使用增量快照
系统 SHALL 按用户、对手身份和球种维护对手统计快照，包含交手场次、胜负和最后交手时间，并以双方各自视角独立更新。

#### Scenario: 注册双方完成比赛
- **WHEN** 用户 A 与用户 B 完成有效比赛
- **THEN** A 对 B 和 B 对 A 的对手统计 SHALL 各更新一次
- **AND** 两个视角的胜负 MUST 互为对应

#### Scenario: 读取 H2H 聚合
- **WHEN** H2H overview 请求统计区块
- **THEN** 系统 SHALL 读取对应对手快照
- **AND** 不得聚合双方全部历史比赛

#### Scenario: 读取近期对手
- **WHEN** 请求对手列表或对手候选
- **THEN** 系统 SHALL 按 `last_match_at` 从对手快照有界分页
- **AND** 不得先加载全部比赛再去重

### Requirement: 历史列表与趋势必须使用参与者索引有界分页
对局历史、H2H 历史、近期胜率趋势和近期对手 SHALL 基于参与者投影按用户、球种、对手和完成时间索引查询，并 MUST 使用明确 LIMIT/游标或数据库分页。

#### Scenario: 用户拥有大量历史比赛
- **WHEN** 请求历史第一页或最近 30 场趋势
- **THEN** 数据库 SHALL 只读取请求范围内的参与者行
- **AND** 应用层不得加载其全部历史后切片

#### Scenario: 双方创建者方向不同
- **WHEN** 用户在不同比赛中有时为发起方、有时为对手方
- **THEN** 参与者投影 SHALL 已统一用户视角
- **AND** 读取逻辑不得再逐行翻转原始 `result` 和比分

### Requirement: 对手强度必须使用结算时段位快照
强弱对手分析 SHALL 以比赛结算时记录的对手段位分档为历史口径，并在用户快照中增量维护各档位场次和胜场。

#### Scenario: 对手赛后升段
- **WHEN** 某对手在历史比赛之后升到更高段位
- **THEN** 已完成比赛的强度分档 MUST 保持为比赛结算时分档
- **AND** 用户分析不得因对手当前段位变化重算全部历史

### Requirement: 当前赛季记录必须随比赛结算更新
当有效排位完成时间落入活动赛季窗口时，系统 SHALL 增量维护该赛季、用户和球种的 `season_records`，包括场次、胜场、起始分、当前结束分和峰值分。

#### Scenario: 活动赛季内完成排位
- **WHEN** 排位比赛完成时间属于当前赛季
- **THEN** 双方赛季记录 SHALL 在结算后立即可读
- **AND** 当前赛季榜单和本人记录不得等待换季才出现

#### Scenario: 完成时间处于赛季边界
- **WHEN** 比赛完成时间恰好等于下一赛季开始时间
- **THEN** 记录 SHALL 只更新下一赛季

### Requirement: 对局结果必须组合不可变核心与当前最新用户指标
完成对局的比分、局、动作摘要、段位变化、成就和完成归因 SHALL 作为核心摘要复用；结果页显示的双方胜率和最高分 MUST 从请求时最新用户统计快照读取。

#### Scenario: 查看刚完成的比赛
- **WHEN** 用户打开结果页
- **THEN** 核心摘要 SHALL 对应该场比赛
- **AND** 胜率与最高分 SHALL 对应双方最新统计 revision

#### Scenario: 之后又完成新比赛
- **WHEN** 用户在比赛 X 之后又完成比赛 Y，再重新打开比赛 X 结果页
- **THEN** 比赛 X 的核心比分和过程 MUST 保持不变
- **AND** 用户胜率和最高分 SHALL 展示包含比赛 Y 的当前最新值

#### Scenario: 打开分享页
- **WHEN** 分享页需要同一比赛结果
- **THEN** 分享链路 SHALL 复用结果核心摘要服务
- **AND** 不得重新扫描双方历史统计

### Requirement: 读模型必须具有可审计 revision 和失效事件
用户竞技快照每次成功变化 SHALL 增加单调 revision，并在数据库提交后发送与通知偏好无关的失效事件。

#### Scenario: 比赛结算提交成功
- **WHEN** 用户快照 revision 从 N 更新到 N+1
- **THEN** WebSocket 事件 SHALL 携带新 revision 和受影响 scopes
- **AND** 客户端缓存 SHALL 能据此失效

#### Scenario: WebSocket 发送失败
- **WHEN** 数据库已提交但实时事件发送失败
- **THEN** 数据库快照 MUST 保持有效
- **AND** 客户端 SHALL 在下次 Bootstrap 通过 revision 差异收敛

### Requirement: 历史读模型重建必须有界、幂等且可恢复
系统 SHALL 提供 dry-run 和正式重建能力，按稳定主键或时间游标分批读取已完成比赛，记录 checkpoint，并使用与在线结算相同的投影规则幂等 upsert。

#### Scenario: 重建中途停止
- **WHEN** 重建进程在某批次后退出
- **THEN** 下次执行 SHALL 从已保存 checkpoint 继续
- **AND** 已处理比赛不得重复计数

#### Scenario: 重复完整重建
- **WHEN** 对相同比赛范围再次执行正式重建
- **THEN** 参与者、用户统计、对手统计和赛季记录 MUST 与首次结果一致

#### Scenario: 大量历史数据
- **WHEN** 需要回建全部历史
- **THEN** 每批查询 SHALL 包含索引游标和固定上限
- **AND** 不得一次性加载整张比赛表或全部用户历史到内存

#### Scenario: 回建顺序与结算顺序不同
- **WHEN** 比赛 ID 顺序与 `completed_at` 顺序不同
- **THEN** 系统 SHALL 先以有界 ID 游标回填缺失 `completed_at`
- **AND** 投影阶段 SHALL 以 `(completed_at, id)` 稳定游标回放
- **AND** 连胜、赛季起止分和最后比赛快照 MUST 与按完成时间在线结算的结果一致

#### Scenario: 单场投影失败
- **WHEN** 回建某场比赛因事实关联缺失或事务失败而中断
- **THEN** checkpoint MUST 保留最后一场成功投影的 `(completed_at, id)`
- **AND** 修复事实后重试 MUST 再次处理失败比赛，不得跳过它

### Requirement: 竞技读模型切读必须受显式就绪门禁保护
新读模型 SHALL 在兼容双写、历史回建和审计完成前保持禁用；线上读接口不得将空或缺失快照作为成功的零值响应。

#### Scenario: 新表刚迁移完成
- **WHEN** 新读模型表为空或 `ReadMode` 未显式设为 `enabled`
- **THEN** 用户统计、历史、H2H、对手、趋势和赛季读取 SHALL 使用兼容旧读路径或明确 partial 语义
- **AND** 不得从空快照返回成功零值

#### Scenario: 审计完成后切读
- **WHEN** 双写已部署、catch-up rebuild 完成且 audit `differences=0`
- **THEN** 运维 MAY 将 `ReadMode` 切为 `enabled`
- **AND** 任意异常 SHALL 能通过恢复为 `disabled` 回退到兼容读路径，不回滚完成比赛事实或双写投影
