## MODIFIED Requirements

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

## ADDED Requirements

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
