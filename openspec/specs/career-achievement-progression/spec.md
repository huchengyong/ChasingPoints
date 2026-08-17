# Career Achievement Progression

## Purpose

定义通用与球种专属生涯成就目录、永久进度、称号发放及历史重建规则。

## Requirements

### Requirement: 通用生涯里程碑目录
系统 SHALL 启用以下 15 项 `game_type = 0` 的通用生涯成就，并按照既有指标事件聚合进度：

| Key | 名称 | 指标 | 模式 | 阈值 |
|---|---|---|---|---:|
| `match_1` | 初入战局 | `matches_total` | sum | 1 |
| `match_10` | 渐入佳境 | `matches_total` | sum | 10 |
| `match_50` | 五十征程 | `matches_total` | sum | 50 |
| `match_100` | 百战磨砺 | `matches_total` | sum | 100 |
| `wins_1` | 首战告捷 | `wins_total` | sum | 1 |
| `wins_10` | 十胜起步 | `wins_total` | sum | 10 |
| `wins_50` | 五十胜将 | `wins_total` | sum | 50 |
| `wins_100` | 百胜丰碑 | `wins_total` | sum | 100 |
| `streak_3` | 状态正佳 | `max_win_streak` | max | 3 |
| `streak_5` | 势如破竹 | `max_win_streak` | max | 5 |
| `streak_10` | 十连制霸 | `max_win_streak` | max | 10 |
| `tournament_join_1` | 赛事启程 | `tournament_join_total` | sum | 1 |
| `tournament_finish_1` | 初登赛场 | `tournament_finish_total` | sum | 1 |
| `tournament_finish_10` | 久经赛场 | `tournament_finish_total` | sum | 10 |
| `tournament_champion_1` | 初次登顶 | `tournament_champion_total` | sum | 1 |

#### Scenario: 第一场有效比赛提供即时反馈
- **WHEN** 用户第一次完成一场能够推进成就的有效比赛
- **THEN** 系统将 `match_1` 进度更新为 1 并永久解锁“初入战局”
- **AND** 同一场比赛继续计入 `match_10`、`match_50` 和 `match_100` 的共享进度

#### Scenario: 通用进度跨球种累计
- **WHEN** 用户分别在多个支持球种中完成有效比赛或获得胜利
- **THEN** `game_type = 0` 的对局、胜场和连胜成就 SHALL 聚合所有支持球种的相应事件

#### Scenario: 赛事报名与完赛分别记录
- **WHEN** 用户首次成功报名正式赛事但尚未形成最终名次
- **THEN** 系统解锁“赛事启程”但不得解锁“初登赛场”
- **AND WHEN** 用户在正式赛事中形成有效最终名次
- **THEN** 系统推进 `tournament_finish_total` 并按阈值解锁完赛成就

### Requirement: 球种专属绝技目录
系统 SHALL 启用以下 10 项球种专属成就，并且仅聚合与定义 `game_type` 相同的进度事件：

| Key | 名称 | 球种 | 指标 | 模式 | 阈值 |
|---|---|---:|---|---|---:|
| `snooker_break_50_1` | 半百一杆 | 1 | `break_50_total` | sum | 1 |
| `snooker_break_100_1` | 破百时刻 | 1 | `break_100_total` | sum | 1 |
| `break_147_1` | 满分时刻 | 1 | `break_147_total` | sum | 1 |
| `chasing_golden_break_1` | 小金初现 | 2 | `golden_break_total` | sum | 1 |
| `chasing_nine_on_break_1` | 大金降临 | 2 | `nine_on_break_total` | sum | 1 |
| `continue_clear_1` | 初次接清 | 3 | `continue_clear_total` | sum | 1 |
| `break_clear_1` | 初次炸清 | 3 | `break_clear_total` | sum | 1 |
| `break_clear_20` | 全台掌控 | 3 | `break_clear_total` | sum | 20 |
| `american_golden_break_1` | 小金初现 | 4 | `golden_break_total` | sum | 1 |
| `american_nine_on_break_1` | 大金降临 | 4 | `nine_on_break_total` | sum | 1 |

#### Scenario: 相同指标按九球球种隔离
- **WHEN** 用户在九球追分中完成一次小金
- **THEN** 系统推进并解锁 `chasing_golden_break_1`
- **AND** 不得推进 `american_golden_break_1`

#### Scenario: 斯诺克绝技不计入其他球种
- **WHEN** 用户产生 `break_100_total` 事件且事件球种为斯诺克
- **THEN** 系统可以推进“破百时刻”
- **AND WHEN** 同名指标事件的球种不是斯诺克
- **THEN** 系统不得将其计入该成就

#### Scenario: 炸清与接清使用不同成就
- **WHEN** 中式八球用户首次完成接清
- **THEN** 系统解锁“初次接清”而不得推进炸清指标
- **AND WHEN** 用户首次完成炸清
- **THEN** 系统解锁“初次炸清”并推进“全台掌控”

### Requirement: 生涯成就跨赛季永久累计
系统 MUST 将所有生涯成就和进度作为账号级永久资产，赛季开始、结束、结算或切换不得重置、降低、重新锁定或回收这些资产。

#### Scenario: 进度跨多个赛季累加
- **WHEN** 用户在多个赛季中累计完成的有效比赛总数达到某个生涯阈值
- **THEN** 系统 SHALL 使用全部赛季的有效事件解锁对应生涯成就

#### Scenario: 赛季切换不影响生涯资产
- **WHEN** 当前赛季完成结算并启用下一赛季
- **THEN** 用户的所有 `user_achievements` 进度、解锁状态和称号 SHALL 保持不变
- **AND** 新赛季挑战仍按其现有规则从零开始

#### Scenario: 赛季报告只做时间归因
- **WHEN** 生涯成就在某赛季日期范围内首次解锁
- **THEN** 赛季报告可以将其展示为“本赛季解锁的生涯成就”
- **AND** 不得为下一赛季复制或重新锁定该成就

### Requirement: 关键里程碑发放稳定称号
系统 SHALL 仅为下表列出的生涯成就自动发放对应称号，称号来源 MUST 指向该成就并保持幂等：

| 成就 Key | 称号 |
|---|---|
| `match_100` | 资深球手 |
| `wins_10` | 胜场新星 |
| `wins_100` | 百胜名将 |
| `streak_5` | 连胜猎手 |
| `streak_10` | 连胜主宰 |
| `break_clear_1` | 清台猎手 |
| `break_clear_20` | 炸清大师 |
| `break_147_1` | 满分王 |
| `tournament_finish_10` | 赛事常客 |
| `tournament_champion_1` | 冠军球手 |

#### Scenario: 新成就首次解锁时发放称号
- **WHEN** 用户首次达到带奖励称号的成就阈值
- **THEN** 系统 SHALL 解锁成就并只创建一条对应来源的用户称号

#### Scenario: 重放不重复发放称号
- **WHEN** 相同比赛、赛事或历史重建再次刷新已经解锁的成就
- **THEN** 系统 MUST 保持原称号记录且不得创建重复称号

#### Scenario: 展示名调整不破坏佩戴状态
- **WHEN** 既有称号“百胜球手”或“不败王者”分别更新为“百胜名将”或“连胜主宰”
- **THEN** 系统 SHALL 保留原 title key、记录 ID、来源引用和 equipped 状态
- **AND** 更新该成就来源称号的展示名与来源名称快照

### Requirement: 历史战绩幂等补算
系统 SHALL 提供可 dry-run 和可执行的历史成就重建能力，从权威比赛、特殊战绩及赛事数据补齐新增定义所需进度，而不得依赖用户下一次比赛才刷新历史资产。

#### Scenario: 新增累计成就使用历史比赛补算
- **WHEN** 老用户在变更上线前已经完成 50 场有效比赛
- **THEN** 正式重建 SHALL 将 `match_50` 进度补到至少 50 并解锁“五十征程”

#### Scenario: 使用阈值跨越时间作为解锁时间
- **WHEN** 历史事件能够确定用户首次达到某项 sum 或 max 阈值的业务时间
- **THEN** 系统 SHALL 使用该时间作为新增成就的 `unlocked_at`
- **AND** 历史称号 SHALL 使用同一业务时间作为 `granted_at`

#### Scenario: 重建重复执行保持幂等
- **WHEN** 操作人员重复执行同一批历史重建
- **THEN** 进度事件、用户成就和用户称号数量不得因重复执行而增加
- **AND** 已有进度不得降低

#### Scenario: dry-run 不修改数据
- **WHEN** 操作人员以 dry-run 模式运行历史重建
- **THEN** 系统 SHALL 输出预计处理用户、事件、新解锁、称号及跳过记录数量
- **AND** 不得创建或修改任何业务数据

### Requirement: 不可靠历史特殊战绩不得猜测归属
系统 MUST 仅在历史特殊战绩能够可靠归属具体用户时补算球种绝技；无法从可信 actor、回合或动作事实确认归属时必须跳过并报告。

#### Scenario: 可推导 actor 的旧记录被补算
- **WHEN** 旧特殊战绩没有可信 actor 但能够从完整回合或动作记录唯一推导触发用户
- **THEN** 重建服务 SHALL 为该用户补齐对应进度事件

#### Scenario: 无法确认归属的旧记录被跳过
- **WHEN** 旧特殊战绩无法唯一确认触发用户
- **THEN** 重建服务 MUST 不为任一用户补算该记录
- **AND** SHALL 将该记录计入无法归属的汇总结果

### Requirement: 既有用户资产在目录升级中受保护
目录迁移和历史重建 MUST 保留所有已有成就解锁、称号、佩戴状态、较早解锁时间和可追溯来源，不得因定义重命名或球种范围收窄而撤销用户资产。

#### Scenario: 已解锁成就在定义更新后仍然有效
- **WHEN** `break_147_1` 从通用范围更新为斯诺克专属或现有成就发生展示名调整
- **THEN** 已解锁用户 SHALL 继续保持解锁状态和原始解锁时间

#### Scenario: 重建结果低于已有快照
- **WHEN** 可重建的历史数据不完整且计算进度低于用户现有进度
- **THEN** 系统 MUST 保留较高的现有进度、解锁状态和既有来源

#### Scenario: 历史补算不产生通知洪峰
- **WHEN** 正式重建为用户补齐多项历史成就和称号
- **THEN** 系统不得为每一项补算发送实时解锁通知或连续奖励弹窗
