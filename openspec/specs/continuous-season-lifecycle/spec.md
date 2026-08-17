# Continuous Season Lifecycle Specification

## Purpose

定义赛季功能启用后的连续生命周期：由系统依据一次性策略确定性生成并持续确保赛季排期，统一按业务时区半开窗口归属数据，原子换季且可幂等追赶，并为空窗数据提供可审计修复；客户端通过 `season_state` 明确区分尚未启用、进行中和数据不可用。

## Requirements

### Requirement: 赛季启用后连续覆盖业务时间
赛季功能 SHALL 从配置的锚点时间开始以首尾相接且互不重叠的窗口持续覆盖业务时间，使每个有效排位或正式赛事完成事件恰好归属一个赛季。

#### Scenario: 相邻赛季无缝衔接
- **WHEN** 当前赛季的排他结束时间到达
- **THEN** 下一赛季 SHALL 在同一时间点开始
- **AND** 两个窗口之间不得存在空窗或重叠

#### Scenario: 事件处于赛季边界
- **WHEN** 一个有效事件的业务完成时间恰好等于下一赛季开始时间
- **THEN** 该事件 SHALL 只归属下一赛季
- **AND** 不得同时计入上一赛季

#### Scenario: 锚点之前的历史数据
- **WHEN** 事件发生在配置的赛季锚点之前
- **THEN** 该事件 SHALL 继续参与生涯成就和永久统计
- **AND** 系统不得为其虚构赛季归属或赛季奖励

### Requirement: 一次性策略确定性生成赛季排期
系统 SHALL 使用是否启用、月初锚点、初始赛季编号、正整数自然月周期和业务时区组成的一次性策略确定性生成赛季名称与日期，并至少确保当前窗口和一个未来窗口存在。

#### Scenario: 空赛季表首次启用
- **WHEN** 赛季表为空、策略已启用且当前业务时间不早于锚点
- **THEN** 系统 SHALL 创建从锚点到当前窗口的连续赛季
- **AND** SHALL 创建至少一个紧随当前窗口的待开始赛季
- **AND** 当前窗口名称 SHALL 按 `S{编号}` 规则确定性生成

#### Scenario: 当前时间早于锚点
- **WHEN** 策略已配置但当前业务时间早于锚点
- **THEN** 系统 SHALL 返回 `not_started` 状态
- **AND** 不得提前创建或激活当前赛季

#### Scenario: 重复或并发确保排期
- **WHEN** 同一实例重复执行或多个服务实例同时确保相同时间范围的排期
- **THEN** 每个开始日期 MUST 至多存在一条赛季记录
- **AND** 重复执行不得改变已经正确生成的名称、日期或已完成结算

#### Scenario: 既有排期与策略冲突
- **WHEN** 既有赛季存在重复起点、重叠窗口或与确定性排期不一致的日期
- **THEN** 系统 SHALL 返回 `unavailable` 并报告冲突
- **AND** 不得自动覆盖、删除或猜测修改既有赛季

### Requirement: 当前赛季按有效时间解析
所有读取当前赛季的接口和内部服务 SHALL 使用业务时区下包含当前时间的权威赛季窗口，而不得只依赖持久化 `status = 1`；在线解析 MUST 根据策略计算期望窗口并使用唯一开始日期或窗口索引读取有限候选，不得加载全部历史赛季。

#### Scenario: Worker 状态更新短暂延迟
- **WHEN** 当前时间已经进入下一赛季窗口但持久化 status 尚未完成切换
- **THEN** 当前赛季读取 SHALL 解析为包含当前时间的下一赛季
- **AND** 客户端不得因此进入无赛季或间歇状态

#### Scenario: 启用后缺少覆盖窗口
- **WHEN** 当前时间已经达到锚点但没有任何有效窗口覆盖当前时间
- **THEN** 系统 SHALL 返回 `unavailable`
- **AND** 不得把该状态解释为正常的赛季间歇

#### Scenario: 在线解析当前窗口
- **WHEN** 当前赛季接口、本人赛季记录默认查询、默认赛季榜单或本人荣誉墙解析当前赛季
- **THEN** 系统 SHALL 先由 policy 计算当前期望开始和排他结束时间
- **AND** SHALL 通过索引读取当前窗口及至多有限相邻/重叠候选进行校验
- **AND** MUST 不得调用无范围 `ListAll`

#### Scenario: 完成时间跨 UTC 日期边界
- **WHEN** 比赛 `completed_at` 在 UTC 日期仍属于上一日、但在 `SeasonLifecycle.Timezone` 已进入下一赛季首日
- **THEN** 投影 SHALL 先转换到该业务时区再确定赛季日期
- **AND** 比赛 MUST 只归属业务时区对应的新赛季，不受数据库主机 `loc=Local` 影响

#### Scenario: 需要完整排期冲突审计
- **WHEN** 运维需要检查所有历史窗口是否冲突
- **THEN** 系统 SHALL 使用专用 repair/dry-run 命令按窗口或游标分批审计
- **AND** 不得把完整历史审计放入在线请求

### Requirement: 换季结算与启用可幂等追赶
系统 SHALL 在确保连续排期后，按开始日期顺序结算所有已经结束但尚未完成结算的赛季。当前赛季记录 SHALL 在比赛结算时持续维护；换季 SHALL 基于有界赛季快照完成最终排名、挑战归档、荣誉、通知、关闭和下一届启用，并通过可恢复阶段及最终发布事务保证幂等一致性，而不得在换季热路径扫描整季原始比赛或执行按用户球种 N+1。

#### Scenario: 正常边界换季
- **WHEN** 当前赛季到达排他结束时间且下一赛季已准备
- **THEN** 系统 SHALL 基于已维护的赛季记录最终化排名
- **AND** SHALL 以批量聚合归档当前赛季挑战
- **AND** SHALL 幂等发放当前赛季荣誉
- **AND** SHALL 在所有必要阶段完成后关闭当前赛季并启用下一赛季

#### Scenario: 换季阶段失败
- **WHEN** 某届换季在最终排名、挑战、荣誉、通知或发布任一步骤失败
- **THEN** 已完成的中间阶段 SHALL 保持幂等且不得对外标记整届 completed
- **AND** 最终发布事务 MUST 回滚其状态切换
- **AND** 下一次执行 MUST 从未完成阶段继续且不得重复快照、记录、称号或通知

#### Scenario: 服务跨越多个边界后恢复
- **WHEN** 服务停机或 Worker 失败期间已经跨越多个赛季边界
- **THEN** 系统 SHALL 通过到期未结算索引按开始日期分页处理全部未结算赛季
- **AND** 最终 SHALL 收敛到当前有效赛季和至少一个未来赛季

#### Scenario: 新赛季激活晚于其开始时间
- **WHEN** 下一赛季窗口开始后已经产生有效事件，系统才完成状态激活
- **THEN** 新赛季挑战和记录 SHALL 包含从既定开始时间起发生的事件
- **AND** 不得从实际激活时刻重新从零计算

#### Scenario: 换季数据量较大
- **WHEN** 一届赛季包含大量用户和比赛
- **THEN** 换季 SHALL 读取该赛季的增量记录和有索引的赛季范围聚合
- **AND** MUST 不得加载全部 matches 后在 Go 中汇总
- **AND** MUST 不得为每个用户球种分别查询完整段位日志

### Requirement: 所有赛季数据使用统一半开时间窗口
比赛、赛事、段位日志、赛季挑战、挑战快照、赛季记录、赛季报告和历史重建 SHALL 统一使用业务时区下的 `[start_date 00:00:00, end_date + 1 day 00:00:00)` 窗口。

#### Scenario: 结束日期当天产生数据
- **WHEN** 有效事件发生在赛季 `end_date` 当天任意时刻且早于次日零点
- **THEN** 该事件 SHALL 计入该赛季

#### Scenario: 跨边界完成排位
- **WHEN** 一场排位在上一赛季开始但在下一赛季窗口内完成
- **THEN** 系统 SHALL 使用 `COALESCE(end_time, match_time)` 将其归入下一赛季

#### Scenario: 正式赛事完成事件
- **WHEN** 正式赛事完成指标的 `occurred_at` 落入某一赛季窗口
- **THEN** 对应赛季挑战 SHALL 只在该窗口内累计该指标

#### Scenario: 报告与重建复用同一窗口
- **WHEN** 系统生成赛季报告、执行历史重建或执行赛季结算
- **THEN** 场次、胜场、段位趋势、解锁成就和挑战进度 SHALL 与实时赛季窗口采用相同边界

### Requirement: 空窗数据修复可审计且幂等
系统 SHALL 提供 dry-run 与正式执行模式，用于补齐锚点之后缺失的赛季窗口并静默结算已经结束的窗口，而不得复制或改写既有比赛、段位日志和成就进度事件。

#### Scenario: 修复 dry-run
- **WHEN** 运维以 dry-run 模式执行赛季生命周期修复
- **THEN** 系统 SHALL 输出待创建窗口、冲突窗口、待结算赛季、覆盖事件与用户及预计资产变化
- **AND** 数据库 MUST 不发生写入

#### Scenario: 正式修复当前空窗
- **WHEN** 锚点之后已经存在有效事件但赛季表缺少覆盖窗口
- **THEN** 正式修复 SHALL 创建确定性连续窗口
- **AND** 当前赛季挑战 SHALL 通过既有事件时间立即反映对应进度

#### Scenario: 修复已结束窗口
- **WHEN** 修复生成了已经结束且尚未结算的赛季
- **THEN** 系统 SHALL 幂等生成对应赛季记录、挑战快照和荣誉
- **AND** 不得创建历史未读换季通知或发送 Push/WebSocket 提醒

#### Scenario: 重复正式修复
- **WHEN** 对同一配置和数据重复执行正式修复
- **THEN** 第二次执行 SHALL 不新增重复赛季、挑战快照、赛季记录、称号或通知

#### Scenario: 修复发现排期冲突
- **WHEN** dry-run 或正式修复发现无法安全自动处理的重叠或日期冲突
- **THEN** 正式修复 MUST 在写入冲突范围前停止
- **AND** SHALL 输出需要人工处理的具体赛季和日期

### Requirement: API 和客户端明确展示赛季生命周期状态
当前赛季接口和本人荣誉墙 SHALL 以加法式字段返回 `season_state`，取值为 `not_started`、`active` 或 `unavailable`；新版客户端 SHALL 根据该状态展示准确文案。

#### Scenario: 正常进行中的赛季
- **WHEN** 当前业务时间由有效赛季窗口覆盖
- **THEN** API SHALL 返回 `season_state = active`
- **AND** SHALL 返回当前赛季名称、日期、挑战数据及 RFC3339 的 `start_at`、`end_at_exclusive` 权威边界

#### Scenario: 赛季尚未启用
- **WHEN** 策略未启用或当前时间早于锚点
- **THEN** API SHALL 返回 `season_state = not_started` 和空当前赛季
- **AND** 新版客户端 SHALL 显示“赛季尚未开启”而不是“赛季间歇中”

#### Scenario: 赛季数据不可用
- **WHEN** 已到启用时间但排期冲突、缺失或赛季查询失败
- **THEN** API SHALL 返回 `season_state = unavailable`
- **AND** 本人荣誉墙 SHALL 继续返回可用的生涯成就与历届荣誉
- **AND** 新版客户端 SHALL 显示“赛季数据更新中”而不是正常间歇状态

#### Scenario: 旧客户端保持兼容
- **WHEN** 旧客户端忽略新增的 `season_state` 字段
- **THEN** 原有 `success`、`season`、`current_season` 和荣誉墙字段的数据类型与语义 MUST 保持兼容

### Requirement: 连续换季保护永久资产并保持段位连续
连续排期、正常换季和历史修复 SHALL 只刷新赛季范围数据，不得清空生涯成就、撤销既有荣誉、改变称号佩戴状态或应用段位软重置。

#### Scenario: 新赛季开始
- **WHEN** 系统从当前赛季切换到下一赛季
- **THEN** 三项赛季挑战 SHALL 按新窗口独立累计
- **AND** 生涯成就进度 SHALL 保持不变并继续永久累计

#### Scenario: 换季前后段位分
- **WHEN** 用户在换季边界前后完成排位
- **THEN** 用户当前段位分 SHALL 按既有排位结算规则连续变化
- **AND** 系统不得因换季应用 `rank_reset_ratio` 或其他未声明重置

#### Scenario: 修复既有荣誉用户
- **WHEN** 生命周期修复涉及已经拥有成就、赛季称号、赛事称号或已佩戴称号的用户
- **THEN** 修复 MUST 保留这些记录、来源引用和佩戴状态

### Requirement: 当前赛季记录必须在赛季进行中实时可读
系统 SHALL 在每场有效排位结算时按完成时间确定唯一赛季，并增量维护该用户、球种和赛季的场次、胜场、起始分、当前分和峰值分。

#### Scenario: 查询当前赛季本人记录
- **WHEN** 用户在活动赛季完成至少一场有效排位后请求赛季 overview
- **THEN** 响应 SHALL 立即包含该场比赛后的本人记录
- **AND** 不得等待赛季结束或 rank rebuild

#### Scenario: 查询当前赛季榜单
- **WHEN** 活动赛季内用户段位分发生变化
- **THEN** 当前赛季榜单 SHALL 从增量 season records 返回最新排序数据
- **AND** 排名共享缓存 SHALL 按赛季和球种版本失效

### Requirement: 生命周期 Worker 必须使用稳定状态快路径
每分钟生命周期 Worker SHALL 在正常稳定状态只读取期望当前/下一窗口和到期未结算记录，并 MUST 不得在一次 tick 中重复扫描全部 seasons。

#### Scenario: 当前与下一赛季均已准备
- **WHEN** Worker tick 发生且没有赛季到期
- **THEN** Worker SHALL 使用固定数量索引查询确认状态
- **AND** 不得执行多个无范围 `ListAll`

#### Scenario: 存在多个待追赶赛季
- **WHEN** Worker 发现多个已结束未结算窗口
- **THEN** 它 SHALL 按开始日期使用固定 page/batch 顺序处理
- **AND** 单次内存加载的赛季数量 SHALL 有上限

### Requirement: 赛季挑战归档必须使用批量聚合
赛季挑战归档 SHALL 使用一次或有界批次的按用户、球种和指标分组查询，并使用支持赛季范围读取的组合索引；不得先列出用户球种后逐对执行 SUM。

#### Scenario: 归档赛季挑战
- **WHEN** 某赛季进入挑战归档阶段
- **THEN** 系统 SHALL 批量生成该赛季的挑战快照
- **AND** SQL 次数不得随用户球种 pair 数量线性增加

### Requirement: 赛季修复与审计必须按范围分批
生命周期 dry-run、repair 和历史重建 MAY 遍历历史窗口，但 SHALL 使用窗口、主键或时间游标和固定 batch size，且 MUST 支持重复执行和中断恢复。

#### Scenario: 多年历史赛季审计
- **WHEN** repair 命令检查多年历史
- **THEN** 每批 SHALL 只读取有限窗口及其索引范围数据
- **AND** 不得一次加载全部赛季、比赛、成就事件和段位日志到内存
