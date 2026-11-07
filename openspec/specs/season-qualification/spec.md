# Season Qualification Specification

## Purpose

定义 V2 赛季的达标门槛（真实参与）、达标优先的赛季榜排序与索引要求，以及赛季最终名次与称号只授予达标者的规则。

## Requirements

### Requirement: V2 赛季名次必须要求真实参与
开始日期不早于 V2 起点的赛季（V2 赛季）SHALL 只为达标记录排名：同一球种本季至少完成 5 场有效排位，且交手过的不同对手不少于 3 人。赛季记录 SHALL 在每场结算时增量维护不同对手数与达标状态；V2 起点之前开始的赛季 MUST 保持原有全员排名规则。

#### Scenario: 刚好达标
- **WHEN** 用户在 V2 赛季中与 3 名不同对手完成第 5 场有效排位
- **THEN** 其赛季记录 SHALL 立即标记为达标并参与名次

#### Scenario: 只和同一个人打
- **WHEN** 用户在 V2 赛季中只与同一名对手完成 20 场排位
- **THEN** 其赛季记录 MUST 保持未达标

#### Scenario: 旧赛季
- **WHEN** V2 起点之前开始的赛季进行排名或结算
- **THEN** 系统 SHALL 按原规则为全部记录排名

#### Scenario: 重复投影
- **WHEN** 同一场比赛的投影被重复执行
- **THEN** 不同对手数与场次 MUST 不重复累加

### Requirement: 赛季榜必须达标优先并保持索引排序
赛季榜、本人实时名次与换季最终名次 SHALL 使用同一排序：达标优先，其次排位分降序、胜场降序、记录 ID 降序，并由 `(season_id, game_type, qualified, end_rank_score, wins, id)` 索引支撑而不发生 filesort。达标记录的名次 SHALL 为达标组内排在其前的记录数加一；未达标记录 SHALL 列在达标记录之后且名次为 0。榜单与赛季信息 SHALL 返回达标标记与本季门槛（旧赛季门槛为 0）。

#### Scenario: 达标与未达标混排
- **WHEN** 一名未达标用户的排位分高于所有达标用户
- **THEN** 榜单 SHALL 先列出全部达标用户并编号
- **AND** 该用户 SHALL 列在其后，名次为 0 且标注未达标

#### Scenario: 分页跨越达标边界
- **WHEN** 客户端按页读取榜单且某页同时包含达标与未达标记录
- **THEN** 达标记录的名次 SHALL 按全局位置连续编号，未达标记录名次为 0

### Requirement: V2 赛季最终名次与称号只授予达标者
V2 赛季结算 SHALL 只为达标记录写入 `final_rank`（从 1 连续编号），未达标记录的 `final_rank` MUST 为 0；赛季称号 SHALL 只发给 `final_rank` 为 1 到 10 的记录。结算阶段的幂等与可恢复语义 MUST 保持不变。

#### Scenario: 某球种只有两人达标
- **WHEN** V2 赛季某球种只有 2 条达标记录
- **THEN** 结算 SHALL 只产生冠军与亚军两个称号
- **AND** 其余参与者 MUST 不获得称号

#### Scenario: 未达标的高分用户
- **WHEN** 一名未达标用户在赛季结束时排位分最高
- **THEN** 其 `final_rank` MUST 为 0 且不得获得称号
