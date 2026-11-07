# Shared Read Caching

## Purpose

定义公共静态数据、排行榜、球馆、赛讯和已完成对局核心摘要的客户端/Redis 缓存、版本失效、回源及禁缓存范围。

## Requirements

### Requirement: 缓存策略必须按领域数据语义定义
系统 SHALL 为每类可缓存数据明确客户端层、服务端层、缓存键维度、TTL、版本和失效事件，并 MUST 不得在统一请求层自动缓存所有 GET。

#### Scenario: 新增可缓存接口
- **WHEN** 一个接口需要接入缓存
- **THEN** 实现 SHALL 明确数据是否共享、允许陈旧时间和写入失效源
- **AND** 缓存键 MUST 包含所有影响响应的身份、球种、分页、过滤或位置桶维度

#### Scenario: 用户级响应
- **WHEN** 缓存数据属于认证用户
- **THEN** 客户端键 MUST 绑定用户身份和认证代次
- **AND** 服务端不得把一个用户的响应作为公共共享缓存

### Requirement: 赛讯必须同时使用现有服务端缓存和客户端 SWR
赛讯列表与详情 SHALL 保留现有 Redis 版本缓存，并在客户端按对应允许陈旧时间复用和后台刷新。

#### Scenario: 在首页与赛讯页之间切换
- **WHEN** 同一赛讯列表客户端缓存仍有效
- **THEN** 第二个页面 SHALL 复用缓存
- **AND** 不得仅为了命中后端 Redis 再产生网络请求

#### Scenario: 管理端或 WST 更新赛讯
- **WHEN** 赛讯内容发生写入
- **THEN** 服务端 SHALL bump 对应版本
- **AND** 下一次回源 MUST 取得新内容

### Requirement: 公共排行榜必须拆分共享缓存和个人排名
排行榜前三名、分页列表与 total SHALL 按球种、分页和领域版本共享缓存；当前用户排名 SHALL 单独查询或使用用户级短缓存后与共享部分组合。

#### Scenario: 多个用户请求同一榜单页
- **WHEN** 共享榜单版本和参数相同
- **THEN** 用户 SHALL 命中同一个共享缓存条目
- **AND** 每个用户的 `my_ranking` MUST 保持独立

#### Scenario: 排位比赛结算
- **WHEN** 某球种排名发生变化
- **THEN** 系统 SHALL 使该球种共享榜单版本失效
- **AND** 其他球种缓存不得被无条件清空

#### Scenario: 用户更新昵称或头像
- **WHEN** 排行榜展示资料发生变化
- **THEN** 相关共享榜单 SHALL 在版本失效或短 TTL 后反映新资料

### Requirement: 公共静态数据必须使用长时版本缓存
规则、术语、地区、段位配置和会员套餐 SHALL 使用 app/schema/config version 控制的客户端持久缓存及服务端进程内或 Redis 缓存。

#### Scenario: 数据版本未变化
- **WHEN** 客户端已有匹配版本的静态数据
- **THEN** 页面 SHALL 直接使用本地缓存

#### Scenario: 配置版本变化
- **WHEN** 服务端配置或 App schema version 变化
- **THEN** 旧缓存 MUST 被判定失效并重新获取

### Requirement: 球馆缓存必须限制地理键基数
球馆列表、详情和附近球馆 MAY 使用短 TTL 缓存；附近球馆键 MUST 使用半径、limit 和离散位置桶，不得使用无限精度经纬度创建高基数键。

#### Scenario: 相邻位置请求
- **WHEN** 两个坐标位于同一配置位置桶且查询半径和 limit 相同
- **THEN** 它们 MAY 复用同一附近球馆候选缓存
- **AND** 响应距离 MAY 在有限候选上重新精确计算

#### Scenario: 球馆审核或资料变化
- **WHEN** 球馆可见性或资料被修改
- **THEN** 对应球馆领域版本 SHALL 失效或由短 TTL 保证收敛

### Requirement: 已完成对局只能长期缓存不可变核心
完成对局核心摘要在奖励/成就核心状态 ready 后 SHALL 可按 match id 和核心版本长期缓存；双方当前胜率、最高分、会员或其他可变用户指标 MUST 不得固化在该长期缓存中。

#### Scenario: 核心摘要命中缓存
- **WHEN** 用户重新打开已完成比赛
- **THEN** 比分、局、动作摘要、段位变化和完成归因 SHALL 从核心缓存复用
- **AND** 当前用户指标 SHALL 从最新快照重新组装

#### Scenario: 奖励仍 pending
- **WHEN** 对局奖励或成就同步尚未 ready
- **THEN** pending 结果 MUST 不得写入长期核心缓存

### Requirement: 实时、安全敏感和写接口不得使用普通响应缓存
认证、短信、Token 刷新、支付订单状态、上传凭证、裁判二维码或加入、进行中比赛详情、当前比分及所有写操作 MUST 不得使用普通响应缓存。

#### Scenario: 进行中比赛
- **WHEN** 客户端需要进行中比赛状态
- **THEN** 系统 SHALL 使用 HTTP 首次快照和 WebSocket 更新
- **AND** 不得返回普通 TTL 缓存中的旧比分

#### Scenario: 支付状态轮询
- **WHEN** 客户端查询订单支付状态
- **THEN** 请求 SHALL 读取权威状态
- **AND** 不得由公共或客户端响应缓存回答

### Requirement: Redis 缓存必须故障回源并防止无界击穿
Redis 读取、解析或写入失败时系统 SHALL 回源权威数据；热点共享键 MAY 使用进程内 single-flight 合并同一回源，但不得建立无界锁或缓存所有参数组合。

#### Scenario: Redis 不可用
- **WHEN** Redis 连接失败
- **THEN** 可缓存读接口 SHALL 从数据库或静态真源返回正确响应
- **AND** 核心业务不得因缓存故障失败

#### Scenario: 热点键同时过期
- **WHEN** 多个请求同时请求同一已过期共享键
- **THEN** 单实例 SHALL 至多执行一个相同回源计算
- **AND** 其他请求 SHALL 复用结果或正常等待有界时间

### Requirement: 用户派生数据首期必须优先读取数据库快照
用户统计、H2H、对手、信誉、会员和荣誉墙首期 SHALL 由客户端 Store 复用并读取有界 MySQL 快照/查询；只有性能指标证明需要时才 MAY 增加用户级 Redis。

#### Scenario: 用户统计快照查询稳定
- **WHEN** 用户历史比赛数增长但快照查询仍为固定行数
- **THEN** 系统 SHALL 不得仅为追求缓存覆盖率增加高基数用户 Redis 键
