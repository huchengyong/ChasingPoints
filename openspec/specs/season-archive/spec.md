# Season Archive Specification

## Purpose

定义赛季档案的历届回顾与进行中实时排名：历届列表有界且只含已结算赛季，赛季页在各状态下提供可达的历届入口，换季通知引导到本人赛季报告。

## Requirements

### Requirement: 历届赛季列表必须有界且只含已结算赛季
系统 SHALL 提供认证用户的历届赛季列表接口，按 `end_date` 降序、`id` 降序分页返回已发布结算（`status = 2`）的赛季，并为每届附带当前用户在指定球种的赛季记录。分页大小 MUST 有上限，SQL 次数 MUST 固定且不随返回条数或历届数量线性增长。

#### Scenario: 存在多届已结算赛季
- **WHEN** 认证用户请求某球种的历届赛季第一页
- **THEN** 系统 SHALL 只返回已结算赛季，按结束时间从近到远排列
- **AND** 每项 SHALL 包含赛季权威时间边界与该用户在该球种的记录（含 `final_rank`、场次、胜场、起止分与峰值分）
- **AND** 响应 SHALL 返回已结算赛季总数供客户端判断是否还有下一页

#### Scenario: 用户未参加某届该球种排位
- **WHEN** 某届已结算赛季中该用户在指定球种没有赛季记录
- **THEN** 该项 SHALL 返回 `record = null`
- **AND** 不得返回零值记录或为其创建默认记录

#### Scenario: 进行中或未开始的赛季
- **WHEN** 赛季仍处于进行中、未开始，或结算尚未发布
- **THEN** 该赛季 MUST 不出现在历届列表中

#### Scenario: 分页参数越界
- **WHEN** 客户端传入非正页码或超过上限的分页大小
- **THEN** 系统 SHALL 归一到第一页或分页上限
- **AND** 不得执行无界查询

#### Scenario: 未认证访问
- **WHEN** 请求未携带有效登录凭证
- **THEN** 系统 MUST 拒绝请求且不得返回任何用户赛季记录

### Requirement: 赛季档案必须提供历届回顾入口
「赛季档案」页 SHALL 在进行中、未开始与不可用等所有赛季状态下提供进入「历届赛季」的入口；历届赛季页 SHALL 支持切换球种，并让有记录的赛季进入该届赛季报告。页面 MUST 不展示永远无法满足条件的入口。

#### Scenario: 进行中赛季打开赛季档案
- **WHEN** 用户在进行中赛季打开「赛季档案」
- **THEN** 页面 SHALL 展示「历届赛季」入口
- **AND** 不得再渲染依赖当前赛季 `status = 2` 的赛季报告按钮

#### Scenario: 赛季未开始或不可用
- **WHEN** 当前赛季状态为 `not_started` 或 `unavailable`
- **THEN** 页面 SHALL 仍提供「历届赛季」入口

#### Scenario: 打开有记录的历届赛季
- **WHEN** 用户在历届赛季页点击一届自己在当前球种有记录的赛季
- **THEN** 客户端 SHALL 打开该届赛季报告，并携带赛季 ID 与当前球种

#### Scenario: 打开无记录的历届赛季
- **WHEN** 某届赛季在当前球种没有本人记录
- **THEN** 该项 SHALL 标注未参加该球种排位且不可进入报告

#### Scenario: 没有已结算赛季
- **WHEN** 历届列表为空
- **THEN** 历届赛季页 SHALL 展示明确空状态，而不是加载失败

#### Scenario: 切号或迟到响应
- **WHEN** 历届赛季请求返回前用户退出、切换账号或认证代次变化，或已切换到其他球种
- **THEN** 旧请求结果 MUST 不得写入当前页面

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

### Requirement: 换季通知必须直达该届回顾
新生成的换季通知 SHALL 对在被结算赛季拥有赛季记录的用户，将跳转目标设为该届赛季报告，并携带该用户在该届场次最多的球种（场次相同取较小球种值）；无赛季记录的用户 SHALL 保持跳转荣誉墙赛季 Tab。「我的」页换季弹窗在目标为赛季报告时 SHALL 提供进入回顾的操作。

#### Scenario: 参加过排位的用户收到换季通知
- **WHEN** 换季结算为在该届有赛季记录的用户生成通知
- **THEN** 通知载荷 SHALL 指向 `/subPages/season/report`，并携带该届赛季 ID 与该用户的主要参赛球种
- **AND** 通知的幂等键与发送时机 MUST 保持不变

#### Scenario: 只有挑战进度的用户收到换季通知
- **WHEN** 用户在该届没有赛季记录但有挑战快照
- **THEN** 通知 SHALL 继续跳转荣誉墙赛季 Tab

#### Scenario: 历史通知
- **WHEN** 本变更上线前已持久化的换季通知被点击
- **THEN** 客户端 SHALL 按其原有目标跳转
- **AND** 系统不得批量回写历史通知

#### Scenario: 在「我的」页查看换季弹窗
- **WHEN** 最新未读换季通知的目标为赛季报告
- **THEN** 弹窗 SHALL 提供「查看回顾」并在确认后标记已读、打开该报告
- **AND** 选择关闭时 SHALL 仅标记已读

### Requirement: 历届赛季必须标注未达标
历届赛季页 SHALL 对有记录但未达标（所在赛季门槛大于 0 且 `qualified = false`）的赛季标注「未达标」而不是名次；仍 SHALL 允许进入该届赛季报告。

#### Scenario: V2 赛季未达标
- **WHEN** 历届列表中某届 V2 赛季本人有记录但未达标
- **THEN** 该项 SHALL 显示「未达标」与场次、胜场
- **AND** 点击 SHALL 打开该届赛季报告
