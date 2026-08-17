## Context

现有成就系统已经完成事件驱动闭环：有效比赛和赛事写入 `achievement_progress_events`，`AchievementProgressService` 按 `metric_key` 与 `sum/max` 聚合，首次达标后写入 `user_achievements` 并通过 `TitleGrantService` 发放可追溯称号。荣誉墙已经区分生涯成就、当前赛季和历届荣誉。

第一版只初始化了 12 项最小定义，且全部以 `game_type = 0` 展示。结果是首场和首胜没有即时反馈，10 到 100 等梯度过大，炸清与 147 等球种绝技被计入所有用户的同一个完成率，九球已有进度指标却没有正式生涯成就。前端 `AchievementDef` 也没有返回球种和奖励称号，生涯页只能按分类分组，详情页统一显示“累计达成 N 次”。

本变更横跨数据库迁移、后端 API/逻辑、历史数据重建、UniApp 荣誉墙和静态图标。实现必须遵守：API 契约先改 `backend/chasing_points.api` 并立即运行 goctl；线上 schema 只由 migrations 管理；页面只通过 `app/api` 调用后端；不撤销用户已获得的成就、称号或当前佩戴状态。

## Goals / Non-Goals

**Goals:**

- 在现有进度引擎上提供 25 项可运营的生涯成就，不新建第二套累计体系。
- 将通用里程碑与四球种绝技分开统计和展示。
- 让第一场、首胜和中间阶段提供连续反馈，同时保留百场、百胜、十连胜等长期目标。
- 保证生涯进度跨赛季永久累计，赛季切换只影响赛季挑战。
- 以幂等方式补算可验证的历史进度和真实达成时间，并保护既有荣誉。
- 用加法式 API 扩展支持独立完成度、即将达成、精确详情和正式图标。

**Non-Goals:**

- 不修改当前三项赛季挑战及其归档规则。
- 不增加成就积分、经验等级、隐藏成就、社交成就或每日/每周任务。
- 不建设管理后台成就配置页。
- 不增加动态稀有度、裁判认证徽标或会员专属竞技成就。
- 不重构赛季称号、赛事名次称号或称号装备模型。

## Decisions

### 1. 复用现有事件与定义模型

继续使用：

```text
比赛/赛事事实
  -> achievement_progress_events
  -> achievements(metric_key, progress_mode, threshold, game_type)
  -> user_achievements
  -> user_titles
```

新增阶段只需增加共享同一 `metric_key` 的定义。例如 `matches_total` 同时驱动 1、10、50、100 场四个里程碑。`game_type = 0` 表示跨球种生涯累计；`game_type = 1..4` 只聚合对应球种事件。

**替代方案：** 为每个成就单独写业务判断。该方案会复制比赛、赛事和回放逻辑，破坏现有幂等与重算能力，因此不采用。

### 2. 固定 25 项首期目录

所有定义保持稳定 Key；现有 Key 语义不变时只更新名称、描述、图标、球种或奖励称号。目录如下：

| Key | 名称 | 分类 | 球种 | 指标 | 模式 | 阈值 | 奖励称号 |
|---|---|---|---:|---|---|---:|---|
| `match_1` | 初入战局 | match | 0 | matches_total | sum | 1 | - |
| `match_10` | 渐入佳境 | match | 0 | matches_total | sum | 10 | - |
| `match_50` | 五十征程 | match | 0 | matches_total | sum | 50 | - |
| `match_100` | 百战磨砺 | match | 0 | matches_total | sum | 100 | 资深球手 |
| `wins_1` | 首战告捷 | wins | 0 | wins_total | sum | 1 | - |
| `wins_10` | 十胜起步 | wins | 0 | wins_total | sum | 10 | 胜场新星 |
| `wins_50` | 五十胜将 | wins | 0 | wins_total | sum | 50 | - |
| `wins_100` | 百胜丰碑 | wins | 0 | wins_total | sum | 100 | 百胜名将 |
| `streak_3` | 状态正佳 | streak | 0 | max_win_streak | max | 3 | - |
| `streak_5` | 势如破竹 | streak | 0 | max_win_streak | max | 5 | 连胜猎手 |
| `streak_10` | 十连制霸 | streak | 0 | max_win_streak | max | 10 | 连胜主宰 |
| `tournament_join_1` | 赛事启程 | tournament | 0 | tournament_join_total | sum | 1 | - |
| `tournament_finish_1` | 初登赛场 | tournament | 0 | tournament_finish_total | sum | 1 | - |
| `tournament_finish_10` | 久经赛场 | tournament | 0 | tournament_finish_total | sum | 10 | 赛事常客 |
| `tournament_champion_1` | 初次登顶 | tournament | 0 | tournament_champion_total | sum | 1 | 冠军球手 |
| `snooker_break_50_1` | 半百一杆 | special | 1 | break_50_total | sum | 1 | - |
| `snooker_break_100_1` | 破百时刻 | special | 1 | break_100_total | sum | 1 | - |
| `break_147_1` | 满分时刻 | special | 1 | break_147_total | sum | 1 | 满分王 |
| `continue_clear_1` | 初次接清 | special | 3 | continue_clear_total | sum | 1 | - |
| `break_clear_1` | 初次炸清 | special | 3 | break_clear_total | sum | 1 | 清台猎手 |
| `break_clear_20` | 全台掌控 | special | 3 | break_clear_total | sum | 20 | 炸清大师 |
| `chasing_golden_break_1` | 小金初现 | special | 2 | golden_break_total | sum | 1 | - |
| `chasing_nine_on_break_1` | 大金降临 | special | 2 | nine_on_break_total | sum | 1 | - |
| `american_golden_break_1` | 小金初现 | special | 4 | golden_break_total | sum | 1 | - |
| `american_nine_on_break_1` | 大金降临 | special | 4 | nine_on_break_total | sum | 1 | - |

奖励称号 Key 继续沿用已有稳定 Key；仅更新展示名时同步更新对应 `user_titles.title_name` 与成就来源快照，不更换 title ID、source 或 equipped 状态。

**替代方案：** 删除赛事报名成就。该方案会使已解锁记录失去定义或要求引入 legacy 展示规则。保留并降级为无称号入门徽章更简单，也能形成报名、完赛、常客、夺冠路线。

### 3. 生涯与赛季采用双轨聚合

生涯成就聚合用户全部可验证历史事件，不按赛季日期过滤，也不在赛季切换时重置。当前赛季挑战继续按 `occurred_at`、赛季日期和球种过滤同一批事件。

当生涯成就在某赛季期间首次达标时，赛季报告可以按 `unlocked_at` 展示“本赛季解锁的生涯成就”，但成就本身不带 `season_id`，新赛季不得重新锁定或清零。

**替代方案：** 每个赛季复制一套成就。该方案会重复稀有绝技、膨胀称号和列表，并与现有赛季挑战职责重叠，因此不采用。

### 4. API 保持兼容并补充展示元数据

在 `AchievementDef` 增加：

- `game_type`
- `reward_title_name`

在 `HonorWallSummary` 保留现有 `career_unlocked/career_total`，并增加：

- `universal_unlocked`
- `universal_total`
- `specialty_game_type`
- `specialty_unlocked`
- `specialty_total`

`career_unlocked/career_total` 继续表示全部启用定义的总数，供旧客户端兼容；新客户端不把它作为主要完成率。`GetHonorWallReq.GameType` 同时用于当前赛季挑战和球种专精摘要。

荣誉墙仍返回完整的 `career_achievements`：本人看到全部启用定义，好友只看到已解锁定义。列表规模仅 25 项，由前端按 `game_type` 分组，无需新增分页或详情接口。

**替代方案：** 后端只返回当前球种定义。该方案会使前端切换球种反复请求，并改变好友荣誉墙的完整已解锁展示，因此不采用。

### 5. 生涯页在现有三轨内增加两段式信息架构

一级 Tab 保持：生涯成就、当前赛季、历届荣誉。

生涯成就内容顺序为：

1. 当前称号和四项摘要：通用进度、当前球种专精、赛季荣誉、赛事荣誉。
2. “即将达成”：从未解锁且适用于通用或当前球种的定义中，按完成百分比降序、定义排序升序取前三项；好友视图不展示未解锁进度。
3. “通用里程碑”：按对局、胜场、连胜、赛事历程顺序分组。
4. “球种绝技”：使用斯诺克、九球追分、中式八球、美式九球切换，仅显示对应 `game_type`。

成就详情直接使用定义描述作为精确解锁条件，并展示球种、奖励称号、进度和解锁时间；移除通用的“累计达成 N 次”文案。

### 6. 图标采用系列化静态资源，不新增稀有度模型

新增 `/static/images/achievements/` 资源目录。对局、胜场、连胜、赛事、斯诺克、中式八球和九球使用七个视觉母题，同系列里程碑通过铜、银、金或高亮构图形成阶段差异。每条启用定义必须配置非空 icon。

首期不新增 `rarity` 字段。真实稀有度需要基于上线后的解锁率统计，避免以主观标签固化错误认知。

### 7. 历史补算使用专用幂等重建服务

新增可测试的 `CareerAchievementRebuildService` 及命令入口，支持 dry-run 与执行模式。它从权威记录补齐缺失进度事件并重建用户快照，而不是在查询荣誉墙时懒计算，也不在 SQL migration 中实现复杂业务回放。

重建来源：

- 对局、胜场：有效完成比赛。
- 连胜：按用户、球种、有效时间顺序回放胜负，计算首次达到 3/5/10 连胜的事件。
- 特殊战绩：优先使用带可靠 actor 的 `match_achievements`，必要时从回合或动作记录推导。
- 赛事：报名记录、有效最终名次和冠军名次。

对于 sum 指标，首次累计达到阈值的事件时间作为 `unlocked_at`；对于 max 指标，首次 `metric_value >= threshold` 的事件时间作为 `unlocked_at`。新发放的历史称号使用同一业务时间作为 `granted_at`。

重建必须：

- 通过现有唯一键幂等写事件。
- 不降低已有 progress，不将 unlocked 从 1 改回 0。
- 不覆盖更早的既有 `unlocked_at` 或解锁来源。
- 不撤销已有称号，不改变 equipped 状态。
- 不为历史补算逐项推送或弹出实时奖励。
- 无法可靠确认旧特殊战绩 actor 时跳过该记录并计入报告，不猜测归属。

**替代方案：** 仅依赖当前 `achievement_progress_events`。第一版未保证 actor 上线前的全部历史都存在于事件表，会让老用户缺失首场、首胜和中间里程碑，因此不采用。

## Risks / Trade-offs

- **[旧特殊战绩的 actor 可能因历史默认值而不可信]** → 重建不得盲信迁移前记录；优先使用动作、回合或可识别的闭环时间边界，无法确认时跳过并报告。
- **[全量历史回放可能耗时]** → 使用批次处理、dry-run、幂等断点和汇总日志；不把重建放在用户请求链路。
- **[新增成就在发布时集中解锁]** → 使用真实阈值跨越时间，静默补算，避免迁移时间污染最近荣誉和通知洪峰。
- **[API 新旧客户端摘要含义不同]** → 保留旧字段并增加拆分字段，旧客户端继续工作，新客户端只使用通用与球种专精摘要。
- **[25 个图标增加设计工作]** → 使用七个系列母题和阶段变体，避免 25 套无关视觉语言。
- **[50、100、20 等门槛未经过真实分布校准]** → 作为首期固定配置上线，后续只通过新的幂等 migration 调整；本次不引入后台动态配置。

## Migration Plan

1. 修改 `backend/chasing_points.api`，运行 goctl，补齐新增响应字段和生成代码。
2. 新增 goose migration：幂等插入 13 项新定义，更新 12 项现有定义的名称、描述、图标、球种、排序和奖励称号；同步更新受影响的成就来源称号快照。
3. 部署支持新定义、拆分摘要和重建服务的后端；旧客户端因字段为加法式扩展继续可用。
4. 先运行重建 dry-run，输出用户数、事件数、新解锁数、称号数和无法归属的特殊战绩数；核对后分批执行正式重建。
5. 发布新版移动端，启用通用/球种分区、即将达成、精确详情和图标。
6. 验证赛季切换、赛事结算、比赛完成仍只通过现有闭环推进，不修改赛季挑战。

回滚时可将新增定义 `status` 设为 0 并恢复旧展示元数据；API 新字段可保留。已经补算出的用户成就和称号不自动删除，以避免回滚进一步损坏用户资产。若重建 dry-run 发现历史来源不可靠，则暂停正式执行，不影响新比赛继续按新目录累计。

## Open Questions

无阻塞实现的问题。图标最终视觉稿和历史 actor 可推导比例在实施阶段验证；无法推导的旧特殊战绩按既定规则跳过，不改变本变更范围。
