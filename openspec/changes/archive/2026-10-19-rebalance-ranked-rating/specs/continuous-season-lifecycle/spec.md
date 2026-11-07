## MODIFIED Requirements

### Requirement: 连续换季保护永久资产并保持段位连续
连续排期、正常换季和历史修复 SHALL 只刷新赛季范围数据，不得清空生涯成就、撤销既有荣誉、改变称号佩戴状态或应用段位软重置。唯一例外是排位规则 V2 在其配置起点进行的一次性、可审计的规则切换；该切换不属于换季软重置，`rank_reset_ratio` SHALL 取 1 以表达不做软重置。

#### Scenario: 新赛季开始
- **WHEN** 系统从当前赛季切换到下一赛季
- **THEN** 三项赛季挑战 SHALL 按新窗口独立累计
- **AND** 生涯成就进度 SHALL 保持不变并继续永久累计

#### Scenario: 换季前后段位分
- **WHEN** 用户在换季边界前后完成排位，且该边界不是 V2 起点
- **THEN** 用户当前段位分 SHALL 按既有排位结算规则连续变化
- **AND** 系统不得因换季应用 `rank_reset_ratio` 或其他未声明重置

#### Scenario: V2 起点
- **WHEN** 当前时间到达 V2 起点
- **THEN** 段位分 SHALL 只按排位规则 V2 的切换规则变化一次并留下 `rating_reset` 记录
- **AND** 生涯胜负、连胜、成就、称号与历届赛季记录 MUST 保持不变

#### Scenario: 读取赛季重置比例
- **WHEN** 客户端读取任一赛季的 `rank_reset_ratio`
- **THEN** 系统 SHALL 返回 1

#### Scenario: 修复既有荣誉用户
- **WHEN** 生命周期修复涉及已经拥有成就、赛季称号、赛事称号或已佩戴称号的用户
- **THEN** 修复 MUST 保留这些记录、来源引用和佩戴状态
