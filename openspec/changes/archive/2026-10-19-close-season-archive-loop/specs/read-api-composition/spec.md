## MODIFIED Requirements

### Requirement: 赛季首屏必须提供概览接口
系统 SHALL 提供赛季 overview，一次返回当前赛季状态、权威时间边界、指定球种本人记录、本人实时名次和排行榜第一页；后续排行榜分页 SHALL 独立加载。本人实时名次 MUST 与赛季榜单、换季最终排名使用同一排序（`end_rank_score` 降序、`wins` 降序、`id` 降序），且读取失败时以 `availability.my_rank = false` 独立表达，不影响其他区块。

#### Scenario: 存在活动赛季
- **WHEN** 用户打开赛季页且存在有效窗口
- **THEN** overview SHALL 返回 `season_state = active`、当前赛季、本人实时记录和首屏榜单

#### Scenario: 赛季不可用
- **WHEN** 生命周期返回 `not_started` 或 `unavailable`
- **THEN** overview SHALL 返回准确状态和空赛季数据
- **AND** 不得继续查询本人记录或排行榜

#### Scenario: 本人已有当前赛季记录
- **WHEN** 活动赛季内用户在指定球种已有赛季记录
- **THEN** overview SHALL 返回 `my_rank` 为排序在本人之前的记录数加一
- **AND** 在赛季记录未变化时，该名次 MUST 与按相同排序分页读取的榜单位置一致

#### Scenario: 本人尚无当前赛季记录
- **WHEN** 活动赛季内用户在指定球种没有赛季记录
- **THEN** overview SHALL 返回 `record = null`、`my_rank = 0` 且 `availability.my_rank = true`
- **AND** 不得为计算名次创建默认赛季记录

#### Scenario: 名次读取失败
- **WHEN** 本人记录可读但名次计数查询失败
- **THEN** overview SHALL 保留本人记录与榜单，并返回 `availability.my_rank = false` 和对应 partial error
- **AND** 不得把名次表达为成功的 `0`

#### Scenario: 竞技读模型尚未切读
- **WHEN** `CompetitiveReadModel.ReadMode` 不是 `enabled`
- **THEN** overview SHALL 继续从 `season_records` 返回本人记录、实时名次与首屏榜单
- **AND** 只有依赖竞技快照的 `competitive_revision` 以 partial 表达不可用

#### Scenario: 查询数量保持固定
- **WHEN** 当前赛季榜单与参赛人数从 1 增长到 100
- **THEN** overview 的 SQL 次数 MUST 保持不变
- **AND** 名次计数 MUST 使用 `season_id`、`game_type` 与排序列组成的索引范围，不得加载全部赛季记录后在内存排序
