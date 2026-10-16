# ranked-finish-countdown Specification

## Purpose
TBD - created by archiving change replace-ranked-dispute-with-finish-countdown. Update Purpose after archive.
## Requirements
### Requirement: 无裁判排位对局结束进入30秒确认
无裁判的排位对局（约球与扫码开局一致）在参赛方通过正式结束接口结束，或因赛制达标自动触发结束时，系统 SHALL 冻结比分并进入 `pending_confirmation`，记录发起方与请求时刻，确认期为服务端时间30秒。确认期间系统 MUST 拒绝记分、结束一局、犯规与再次发起结束。练习对局与已绑定裁判的对局 MUST NOT 进入确认期。

#### Scenario: 约球排位手动结束
- **WHEN** A 在约球排位对局中满足正常结束条件并滑动结束
- **THEN** 比赛保持进行中并进入待确认，比分冻结，关联约球尚未完成，B 收到确认请求

#### Scenario: 赛制达标自动触发
- **WHEN** 抢 N 局排位对局因 A 记下的一局达到目标胜局
- **THEN** 系统 SHALL 进入同一30秒确认期，发起方记为 A

#### Scenario: 确认期间拒绝记分
- **WHEN** 确认期间任一方提交加分、结束一局、犯规或再次发起结束
- **THEN** 系统 MUST 拒绝请求并保持冻结比分不变

#### Scenario: 练习与裁判局不受影响
- **WHEN** 练习对局的参赛方结束比赛，或裁判结束已绑定裁判的对局
- **THEN** 系统 SHALL 直接完成结算，不产生确认期

### Requirement: 确认期内可确认、撤回或提出异议
确认期内，另一参赛方 SHALL 可以确认结束，系统立即以冻结比分完成结算（完成来源 `player_confirmed`，完成人记为确认方，沿用现有语义），并同步完成关联约球。另一参赛方提出异议或发起方撤回时，系统 SHALL 清除请求并恢复进行中，比分不变，之后任一参赛方可再次发起结束。发起方 MUST NOT 确认或异议自己的请求；非参赛者 MUST NOT 执行任何确认动作。所有动作 SHALL 校验 revision，并按 `client_action_id` 幂等。

#### Scenario: 对方确认立即结束
- **WHEN** A 发起结束后，B 在截止前确认
- **THEN** 比赛立即完成，不等待剩余时间；关联约球完成；`completion_source=player_confirmed`，`completed_by_user_id` 为确认方 B

#### Scenario: 发起方撤回
- **WHEN** A 在截止前撤回结束请求
- **THEN** 比赛恢复进行中，比分不变，可继续记分，A 或 B 可再次发起结束

#### Scenario: 对方提出异议
- **WHEN** B 在截止前提出异议
- **THEN** 比赛恢复进行中，比分不变，A 收到对局继续的状态，任一方可再次发起结束

#### Scenario: 越权动作被拒绝
- **WHEN** A 试图确认或异议自己的请求，或非参赛者调用任一确认动作
- **THEN** 系统 MUST 拒绝并返回权威比赛状态

### Requirement: 30秒无人处理按冻结比分完成且不可再申诉
自请求时刻起满30秒仍未被确认、撤回或异议的请求，系统 SHALL 不依赖任何客户端在线，由服务端按冻结比分完成结算：完成来源记为 `finish_timeout`，结束方为发起方，并写系统动作 `finish_timeout`；关联约球 SHALL 在同一终态事务中完成。完成时延 SHOULD 不超过截止后数秒。截止后到达的确认、撤回或异议 MUST NOT 改变结果，系统 SHALL 返回已结束的权威状态。比赛完成后 MUST NOT 提供任何异议、申诉、作废或恢复入口。

#### Scenario: 对方离线
- **WHEN** A 发起结束后 B 离线，双方都不再操作
- **THEN** 截止后服务端完成结算并同步完成约球，双方恢复后看到赛果，可以重新约球

#### Scenario: 截止后到达的动作
- **WHEN** B 在截止后提交确认、异议，或 A 在截止后撤回
- **THEN** 结果不变；若请求尚未处理，系统 SHALL 先按超时完成，再向请求方返回已结束状态

#### Scenario: 恰好一次
- **WHEN** 多个 worker 实例、任务重试与 B 的确认同时处理同一到期请求
- **THEN** 系统 SHALL 只完成一次结算，排位变化、约球状态与通知都不重复

#### Scenario: 无事后申诉
- **WHEN** 比赛已因确认或超时完成
- **THEN** 读取与接口 MUST NOT 提供异议、作废或恢复操作

### Requirement: 读取返回倒计时权威字段
当前对局、对局详情、同步快照与 WebSocket 权威消息在待确认时 SHALL 返回 `finish_requested_by`、`finish_request_expires_at`，以及按服务端时间计算的 `finish_request_remaining_seconds`（最小为0），并按查看者身份返回可撤回、可确认、可异议权限。到期但尚未完成的请求 SHALL 仍返回待确认、剩余0秒，且各操作权限为 false；读取 MUST NOT 把它改写为已取消或进行中。读取 MUST 保持纯读，不在读取中完成结算。

#### Scenario: 双方读取一致
- **WHEN** A 与 B 在确认期内分别读取当前对局
- **THEN** 两者返回相同的截止时刻；A 可撤回，B 可确认与异议

#### Scenario: 到期尚未处理
- **WHEN** 截止已过而服务端尚未完成结算
- **THEN** 读取返回待确认、剩余0秒且无操作权限

#### Scenario: 读取不结算
- **WHEN** 读取一场到期尚未处理的比赛
- **THEN** 读取 MUST NOT 写库，结算仍由 worker 或写动作完成

### Requirement: 客户端倒计时与确认交互
UniApp（App 与微信小程序一致）的进行中页在待确认时 SHALL 向发起方显示倒计时与“撤销结束”，向另一参赛方显示确认弹窗：发起人、比分、倒计时、与结束滑块同样防误触的“向右滑动确认结束”，以及“有异议，继续比赛”。倒计时 SHALL 以服务端剩余秒数为起点，按本地经过时间递减，不以手机时钟的绝对值判断；归零后显示“正在结束…”，未收到结束消息时回源。操作失败 SHALL 保留入口并以权威状态刷新；异步结果 MUST 按账号与认证代次隔离。排位赛果页 SHALL 显示结束方式“X发起 · Y确认”或“X发起 · 30秒未处理，自动结束”；练习赛果页不显示结束方式。

#### Scenario: 发起方视图
- **WHEN** A 发起结束后停留在进行中页
- **THEN** 页面显示等待 B 确认、剩余秒数与“撤销结束”，不再显示结束滑块

#### Scenario: 对方确认弹窗防误触
- **WHEN** B 在进行中页看到确认弹窗，并点击轨道、中途松手或触摸被取消
- **THEN** App MUST NOT 发送确认；只有从起点滑到终点并松手才确认一次

#### Scenario: 倒计时归零
- **WHEN** 倒计时归零而尚未收到结束消息
- **THEN** 页面显示“正在结束…”，收到结束消息或回源后显示赛果与结束方式

#### Scenario: 切换账号
- **WHEN** 确认、撤回或异议请求返回前用户切换了账号
- **THEN** 旧响应 MUST NOT 回写新账号的页面

### Requirement: 取消10分钟排位异议期
系统 MUST NOT 再把排位结算延后生效，MUST NOT 提供 `POST /api/match/result-dispute`，也 MUST NOT 返回 `rank_settlement_status`、`rank_settlement_due_at` 或 `can_dispute_result`。排位比赛完成时 SHALL 立即结算，竞技读模型 SHALL 在完成时一次性计入排位口径统计。

#### Scenario: 异议接口已移除
- **WHEN** 客户端调用 `POST /api/match/result-dispute`
- **THEN** 服务端返回未找到，不改变任何比赛

#### Scenario: 完成即结算
- **WHEN** 排位比赛因确认或超时完成
- **THEN** 段位、排位变动日志与排位口径统计在同一完成流程中生效
