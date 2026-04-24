# 成就、称号与特殊战绩闭环设计

## 目标

把当前仓库里混用的 `achievement` 概念拆清，形成一套能真正跑通的闭环：

1. 比赛完成后能稳定记录“特殊战绩”
2. 特殊战绩、胜场、连胜、对局、赛事参与能推进账号级“成就”进度
3. 成就解锁后能自动发放“成就称号”
4. 赛季结算后能自动发放“赛季称号”
5. 赛事结算后能自动发放“赛事称号”
6. 用户能在前端看到成就进度、已拥有称号，并装备一个当前称号

---

## 范围结论

首版只做最小闭环，优先覆盖当前仓库里已经有数据基础的链路。

### 首版纳入

1. 特殊战绩
2. 对局成就
3. 胜场成就
4. 连胜成就
5. 特殊战绩成就
6. 赛事参与/夺冠成就
7. 成就称号
8. 赛季称号
9. 赛事称号

### 首版不做

1. 社交成就
2. 后台可视化成就配置页
3. 称号时效与过期回收
4. 历史全量自动补算 UI
5. 称号展示位扩展到排行榜、好友主页、动态卡片

`社交` 相关成就在数据库结构里虽然预留了分类，但当前仓库没有稳定的行为闭环，首版不纳入。

---

## 统一定义

### 1. 特殊战绩

特殊战绩是“单场比赛里的高光事实”，例如：

1. 炸清
2. 接清
3. 小金
4. 大金
5. 单杆 50+
6. 单杆 100+
7. 单杆 147

它的职责是记录“这场比赛发生了什么”，不是长期资产，也不能装备。

### 2. 成就

成就是“账号级长期里程碑”，例如：

1. 累计 10 胜
2. 最高 5 连胜
3. 累计 20 次炸清
4. 累计参加 10 场赛事

它的职责是记录“用户长期做到过什么”，有阈值、进度、解锁状态和解锁时间。

### 3. 称号

称号是“可展示的身份标签”，例如：

1. 百胜球手
2. 炸清大师
3. S1 赛季冠军
4. 上海公开赛冠军

它的职责是记录“用户当前拥有什么荣誉标签”，没有进度，只有拥有与装备状态。

---

## 关系模型

统一按下面这条链路理解：

`用户行为/比赛结算 -> 特殊战绩或累计事件 -> 成就进度更新 -> 成就解锁 -> 发放称号 -> 用户装备称号`

其中：

1. 特殊战绩是事件
2. 成就是长期结果
3. 称号是奖励与展示物

三者不是同义词，也不应该共用一个业务名。

---

## 当前缺口

### 1. `match_achievements` 缺失归属人

当前表只有：

1. `match_id`
2. `achievement_type`
3. `count`

没有“是谁打出来的”字段。这样像炸清、147 这类高光只能知道“这场出现过”，却不能稳定归到具体用户名下，后续成就进度无法准确累计。

### 2. 缺少进度事件流水

当前 `user_achievements` 可以存最终进度，但没有独立的幂等事件表。这样一旦比赛回放、赛果修正、重建排位，就无法稳定回滚或重算成就。

### 3. 称号缺少来源引用

当前 `user_titles` 只有 `source`，没有具体引用对象。系统只能知道“来自赛季”或“来自赛事”，却不知道是哪个赛季或哪场赛事。

### 4. 前端命名混用

当前：

1. 成就页展示的是账号成就
2. 对局结果页把“特殊战绩”也叫成就
3. 称号页文案已隐含“来自成就/赛季/赛事”

业务边界不清晰，容易让用户误解。

### 5. 成就分类码和前端显示值存在不一致风险

数据库迁移注释使用的是：

1. `wins`
2. `streak`
3. `special`
4. `match`
5. `social`
6. `tournament`

而前端页面当前直接拿中文值做过滤：

1. 胜场
2. 连胜
3. 特殊
4. 对局
5. 社交
6. 赛事

首版需要统一为“后端存码值，前端做映射”。

---

## 方案结论

采用“事件驱动进度 + 解锁发奖 + 赛季/赛事结算发奖”的闭环模型。

核心原则：

1. 特殊战绩只记录事实
2. 成就只管累计和解锁
3. 称号只管拥有和装备
4. 所有成就推进必须有幂等事件
5. 所有称号发放必须可追溯来源
6. 所有赛果驱动的进度都必须支持重算

---

## 数据模型

### 1. `achievements`

成就定义表，一条记录代表一个可解锁里程碑。

建议字段：

1. `id`
2. `key`
3. `name`
4. `description`
5. `icon`
6. `category`
7. `game_type`
8. `metric_key`
9. `progress_mode`
10. `threshold`
11. `reward_title_key`
12. `reward_title_name`
13. `sort`
14. `status`
15. `created_at`
16. `updated_at`

字段语义：

1. `category` 使用码值：`wins/streak/special/match/tournament`
2. `game_type` 使用 `0` 表示不区分球种
3. `metric_key` 指向统一进度指标
4. `progress_mode` 仅允许 `sum/max`
5. `reward_title_*` 允许为空，表示只解锁成就不发称号

### 2. `user_achievements`

用户成就快照表。

建议字段：

1. `id`
2. `user_id`
3. `achievement_id`
4. `progress`
5. `unlocked`
6. `unlocked_at`
7. `reward_granted`
8. `reward_granted_at`
9. `created_at`
10. `updated_at`

唯一键：

1. `uk_user_achievement (user_id, achievement_id)`

字段语义：

1. `progress` 是当前累计值
2. `reward_granted` 保证成就称号只发一次

### 3. `achievement_progress_events`

进度事件流水表，用于幂等、重算和审计。

建议字段：

1. `id`
2. `user_id`
3. `source_type`
4. `source_id`
5. `game_type`
6. `metric_key`
7. `metric_value`
8. `created_at`

唯一键：

1. `uk_user_source_metric (user_id, source_type, source_id, metric_key)`

`source_type` 首版只需支持：

1. `match`
2. `tournament_join`
3. `tournament_finish`
4. `season_rebuild`

### 4. `match_achievements`

现有特殊战绩表需要补字段。

新增字段：

1. `actor`，值域只允许 `1/2`

唯一键改为：

1. `uk_match_actor_achievement (match_id, actor, achievement_type)`

这样一场比赛里双方都能分别拥有自己的炸清、147 等记录。

### 5. `user_titles`

用户称号表需要升级为可追溯来源的快照表。

建议字段：

1. `id`
2. `user_id`
3. `title_key`
4. `title_name`
5. `source_type`
6. `source_ref_id`
7. `source_ref_name`
8. `granted_by_achievement_id`
9. `equipped`
10. `equipped_at`
11. `granted_at`
12. `created_at`

唯一键：

1. `uk_user_title_source (user_id, title_key, source_type, source_ref_id)`

字段语义：

1. `source_type` 只允许 `achievement/season/tournament`
2. `source_ref_id` 记录赛季 ID、赛事 ID 或成就 ID
3. `source_ref_name` 冗余快照，避免后续展示强依赖 join
4. `title_key` 必须稳定，避免同一个来源重复发放

---

### 称号 `title_key` 约定

首版落地使用以下规则：

1. 成就称号：使用 `achievements.reward_title_key`
2. 赛季称号：`season_{season_id}_game_{game_type}_rank_{final_rank}`
3. 赛事称号：`tournament_{tournament_id}_rank_{final_rank}`

其中成就称号的 `source_ref_id` 指向 `achievement_id`，赛季称号指向 `season_id`，赛事称号指向 `tournament_id`。

---

## 有效比赛定义

首版只有“有效比赛”才能推进特殊战绩和成就。

有效比赛必须同时满足：

1. `match.status = 2`
2. 有明确胜者
3. 双方都是注册用户
4. 双方用户 ID 不相同
5. 至少完成 1 局
6. 不是取消局或中断局

不满足以上任一条件时：

1. 不推进对局成就
2. 不推进胜场成就
3. 不推进连胜成就
4. 不推进特殊战绩成就

---

## 指标与进度计算规则

### 统一指标

首版统一使用以下 `metric_key`：

1. `matches_total`
2. `wins_total`
3. `max_win_streak`
4. `break_clear_total`
5. `continue_clear_total`
6. `golden_break_total`
7. `nine_on_break_total`
8. `break_50_total`
9. `break_100_total`
10. `break_147_total`
11. `tournament_join_total`
12. `tournament_finish_total`
13. `tournament_champion_total`

### 聚合模式

1. `sum`：对次数类指标累计求和
2. `max`：对峰值类指标取最大值

### 来源规则

#### 比赛完成时

1. 双方各产生一条 `matches_total = 1`
2. 胜者产生一条 `wins_total = 1`
3. 双方各产生一条 `max_win_streak = 当前 ranking.max_streak`
4. 从带 `actor` 的 `match_achievements` 里为对应用户写入特殊战绩事件

#### 报名赛事成功时

1. 参赛用户产生一条 `tournament_join_total = 1`

#### 赛事结算时

1. 所有有最终名次的用户产生一条 `tournament_finish_total = 1`
2. `final_rank = 1` 的用户再产生一条 `tournament_champion_total = 1`

#### 赛季重建时

首版不通过赛季事件推进账号成就，赛季只负责发赛季称号。

---

## 最小成就定义

首版建议落 12 条成就定义，先把闭环跑通。

| key | 类别 | metric_key | 模式 | 阈值 | 奖励称号 |
|---|---|---|---|---|---|
| `match_10` | 对局 | `matches_total` | `sum` | 10 | 无 |
| `match_100` | 对局 | `matches_total` | `sum` | 100 | `资深球手` |
| `wins_10` | 胜场 | `wins_total` | `sum` | 10 | `胜场新星` |
| `wins_100` | 胜场 | `wins_total` | `sum` | 100 | `百胜球手` |
| `streak_5` | 连胜 | `max_win_streak` | `max` | 5 | `连胜猎手` |
| `streak_10` | 连胜 | `max_win_streak` | `max` | 10 | `不败王者` |
| `break_clear_1` | 特殊 | `break_clear_total` | `sum` | 1 | `清台猎手` |
| `break_clear_20` | 特殊 | `break_clear_total` | `sum` | 20 | `炸清大师` |
| `break_147_1` | 特殊 | `break_147_total` | `sum` | 1 | `满分王` |
| `tournament_join_1` | 赛事 | `tournament_join_total` | `sum` | 1 | 无 |
| `tournament_finish_10` | 赛事 | `tournament_finish_total` | `sum` | 10 | `赛事常客` |
| `tournament_champion_1` | 赛事 | `tournament_champion_total` | `sum` | 1 | `冠军球手` |

---

## 最小称号发放规则

### 1. 成就称号

在成就由“未解锁”变成“已解锁”时发放。

| 条件 | 发放称号 |
|---|---|
| 解锁 `wins_10` | `胜场新星` |
| 解锁 `wins_100` | `百胜球手` |
| 解锁 `streak_5` | `连胜猎手` |
| 解锁 `streak_10` | `不败王者` |
| 解锁 `break_clear_1` | `清台猎手` |
| 解锁 `break_clear_20` | `炸清大师` |
| 解锁 `break_147_1` | `满分王` |
| 解锁 `tournament_finish_10` | `赛事常客` |
| 解锁 `tournament_champion_1` | `冠军球手` |

### 2. 赛季称号

在赛季结算完成并生成最终 `season_records.final_rank` 后发放。

| 条件 | 发放称号 |
|---|---|
| `final_rank = 1` | `{season_name}{game_type_name}赛季冠军` |
| `final_rank = 2` | `{season_name}{game_type_name}赛季亚军` |
| `final_rank = 3` | `{season_name}{game_type_name}赛季季军` |
| `4 <= final_rank <= 10` | `{season_name}{game_type_name}赛季前十` |

赛季称号建议永久保留，不做赛季结束后自动回收。

### 3. 赛事称号

在赛事结束并写入 `tournament_participants.final_rank` 后发放。

| 条件 | 发放称号 |
|---|---|
| `final_rank = 1` | `{tournament_name}冠军` |
| `final_rank = 2` | `{tournament_name}亚军` |
| `final_rank = 3` | `{tournament_name}季军` |
| `final_rank <= 4` 且参赛人数 >= 4 | `{tournament_name}四强` |

如果一场赛事已结算过，再次回放只允许幂等补齐，不允许重复生成重复称号。

---

## 发放时机

### 1. 成就称号

触发点：

1. 进度事件写入后
2. 成就聚合完成后
3. 某条成就首次满足阈值时

处理：

1. 更新 `user_achievements.unlocked`
2. 写 `reward_granted = 1`
3. `upsert user_titles`

### 2. 赛季称号

触发点：

1. 赛季记录重建完成
2. `final_rank` 已生成

处理：

1. 扫描目标赛季全部 `season_records`
2. 按名次规则 `upsert user_titles`
3. 赛季记录重建、最终名次生成和称号发放必须基于同一轮重建结果

### 3. 赛事称号

触发点：

1. `FinishTournament` 成功写入 `final_rank`

处理：

1. 扫描当前赛事全部参赛者最终名次
2. 按名次规则 `upsert user_titles`

---

## 前端命名与页面归属

### 命名统一

对用户展示的名称统一为：

1. `特殊战绩`
2. `成就`
3. `称号`

### 页面归属

1. 对局结果页：只展示“特殊战绩”或“本场亮点”
2. 成就列表页：只展示账号成就
3. 成就详情页：只展示单条成就
4. 称号管理页：只展示已拥有称号与装备状态
5. 赛季报告页：只展示本赛季解锁成就，不直接展示称号

### 分类映射

后端统一输出分类码值：

1. `wins`
2. `streak`
3. `special`
4. `match`
5. `tournament`

前端自行映射为中文显示，避免用中文值做数据过滤。

---

## 幂等与重算

### 1. 事件幂等

任何推进进度的来源都必须先写 `achievement_progress_events`，依赖唯一键避免重复累计。

### 2. 比赛回放

如果比赛结果被修正：

1. 删除该 `match_id` 关联的成就事件
2. 重新回放比赛特殊战绩与胜负指标
3. 重刷受影响用户的成就快照

### 3. 赛事回放

如果赛事最终名次被修正：

1. 删除该 `tournament_id` 的赛事相关事件
2. 重新生成 `tournament_finish_total/tournament_champion_total`
3. 重刷赛事称号

### 4. 赛季重建

赛季称号不依赖用户手工操作，统一挂到现有赛季重建链路里重放，保证回放后名次称号和赛季记录一致。

---

## 闭环流程

### 1. 比赛闭环

1. 用户完成一场有效比赛
2. 系统按 `actor` 写入特殊战绩
3. 系统写入比赛进度事件
4. 系统聚合刷新成就进度
5. 系统解锁新成就
6. 系统发放成就称号
7. 前端成就页展示最新进度
8. 前端称号页可装备新称号

### 2. 赛事闭环

1. 用户报名赛事
2. 系统写入 `tournament_join_total`
3. 用户完赛并写入最终名次
4. 系统写入赛事完成/夺冠事件
5. 系统刷新赛事成就
6. 系统按名次发放赛事称号

### 3. 赛季闭环

1. 赛季记录重建
2. 生成 `season_records.final_rank`
3. 系统按赛季名次发放赛季称号
4. 用户在称号页看到并可装备

---

## 风险与约束

### 1. 现有比赛特殊战绩数据可能无法直接补齐归属人

如果历史 `match_achievements` 没有 `actor`，老数据只能：

1. 放弃精确回补
2. 或按回合记录重新推导

首版建议只保证新比赛闭环，历史数据单独评估补算。

### 2. 赛季称号依赖赛季记录重建

当前赛季名次生成依赖 `rank_rebuild_service`。如果后续有新的赛季结算入口，称号发放必须跟着同一个入口走，避免赛季记录与赛季称号不一致。

赛季重建期间，排行榜变更日志读取需要使用同一个事务上下文，避免刚重放出的 rank change 还没提交时，赛季记录计算读不到本轮数据。

### 3. 社交成就暂不开放

前端先隐藏或不展示 `social` 分类，等有真实行为闭环后再打开。

---

## 最终结论

首版最小闭环应该是：

1. 给 `match_achievements` 补 `actor`
2. 新增 `achievement_progress_events`
3. 升级 `achievements / user_achievements / user_titles`
4. 让比赛、赛事、赛季三个结算点都能写进度或发称号
5. 统一前端命名为“特殊战绩 / 成就 / 称号”

这套方案能把当前仓库里分散的结构收成一条完整链路，并且后续支持回放、幂等和阶段性扩展。
