## MODIFIED Requirements

### Requirement: 进行中赛季必须展示本人实时排名
「赛季档案」页 SHALL 在活动赛季使用 overview 返回的 `my_rank` 展示本人当前名次，而不得依赖只在结算时写入的 `final_rank`；本人尚无记录时 SHALL 给出上榜指引；赛季有达标门槛且本人未达标时 SHALL 展示达标进度；本人有记录时 SHALL 展示本季排位分净变化。赛季榜 SHALL 使用服务端返回的名次，未达标记录不显示名次。

#### Scenario: 本人已上榜
- **WHEN** overview 返回本人记录且 `my_rank > 0`
- **THEN** 页面 SHALL 展示「当前排名：第 N 名」

#### Scenario: 本人未达标
- **WHEN** overview 返回本人记录、赛季门槛大于 0 且记录未达标
- **THEN** 页面 SHALL 展示已完成场次与不同对手数相对门槛的进度
- **AND** 不得展示「第 0 名」

#### Scenario: 本季净变化
- **WHEN** overview 返回本人记录
- **THEN** 页面 SHALL 展示 `end_rank_score − start_rank_score` 的带符号数值

#### Scenario: 本人尚无赛季记录
- **WHEN** overview 返回 `record = null` 且记录区块可用
- **THEN** 页面 SHALL 提示完成一场该球种排位即可上榜，赛季有门槛时同时说明门槛
- **AND** 不得展示零值战绩卡

#### Scenario: 名次暂不可用
- **WHEN** overview 返回 `availability.my_rank = false`
- **THEN** 页面 SHALL 继续展示本人战绩与榜单
- **AND** 不得展示「第 0 名」或把名次缺失表达为未上榜

#### Scenario: 榜单包含未达标记录
- **WHEN** 榜单项返回 `qualified = false`
- **THEN** 页面 SHALL 以「未达标」代替名次
- **AND** 加载更多时 MUST 按用户去重，不得出现重复行

## ADDED Requirements

### Requirement: 历届赛季必须标注未达标
历届赛季页 SHALL 对有记录但未达标（所在赛季门槛大于 0 且 `qualified = false`）的赛季标注「未达标」而不是名次；仍 SHALL 允许进入该届赛季报告。

#### Scenario: V2 赛季未达标
- **WHEN** 历届列表中某届 V2 赛季本人有记录但未达标
- **THEN** 该项 SHALL 显示「未达标」与场次、胜场
- **AND** 点击 SHALL 打开该届赛季报告
