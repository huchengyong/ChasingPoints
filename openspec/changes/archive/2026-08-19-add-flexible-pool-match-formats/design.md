## Context

中式八球（`game_type=3`）和美式九球（`game_type=4`）当前都通过 `EndRound` 记录每局胜方，但服务端仍信任请求中的 `score`，通用 `MatchScore` / `MatchFoul` 也能直接改总比分，结束则主要依赖用户手动操作。因此服务端无法保证“每局固定 1 胜”、抢 N 到点结束或自由局数 129 局上限。

斯诺克已经有 `snooker_format` / `snooker_target_wins`、首局前编辑、快照同步和正常结束约束，但其字段同时承载 `best_of_frames` 兼容、WPBSA 规则版本以及 49 局 / 抢 25 的专属语义，不能直接用于中八和美九。`game_mode` 也已有既定业务含义，不适合作为赛制枚举。

本变更横跨 `matches` 数据模型、go-zero API、对局写事务、结束确认、WebSocket 快照和 UniApp 对局页。约束是：仅影响中八和美九的新灵活赛制对局；九球追分（`game_type=2`）、斯诺克和存量对局行为保持不变；API 契约修改后立即通过 goctl 重新生成代码。

## Goals / Non-Goals

**Goals:**

- 为中八和美九提供 `free` 与 `race_to` 两种服务端可验证的赛制。
- 自由局数至少完成一局后可正常结束并允许平局，完成第 129 局后自动进入正常结束流程。
- 抢 N 支持 1～65 胜，任一方到点即进入正常结束流程，最多实际完成 129 局。
- 赛制仅允许创建者在首条对局动作和首局产生前修改，并通过 revision 与实时快照同步给双方。
- 灵活赛制总比分只能由 `EndRound` 每局固定加 1 产生，不能通过通用加分或犯规接口绕过。
- 练习赛直接结算，排位赛沿用现有双方结束确认、裁判权限和结算副作用。
- 存量记录以及未显式声明新赛制的旧客户端对局继续使用 `legacy` 行为。

**Non-Goals:**

- 不改变斯诺克的 `snooker_format`、`snooker_target_wins`、`best_of_frames` 或其 49 局 / 抢 25 规则。
- 不支持九球追分，也不改变其 4/7/10 分计分方式。
- 不扩展球种选择弹层，不新增赛事编排、让局、局中改赛制或自定义 129 以上局数。
- 不新增中八或美九的整场认输协议；抢 N 未达目标时不得按正常完赛提前结束，既有取消和裁判处置路径保持原语义。
- 不迁移存量中八、美九对局为新赛制，也不推断存量比分对应的目标胜局。

## Decisions

### 1. 新增独立的通用赛制字段，但当前只赋予中八和美九语义

在 `matches` 增加 `match_format` 和 `target_wins`：

- `legacy`：旧行为，`target_wins=0`。
- `free`：自由局数，`target_wins=0`。
- `race_to`：抢 N，`target_wins` 必须为 1～65。

模型层提供面向逐局球种的规范化与判断 helper，并且只有 `game_type IN (3, 4)` 能创建或更新为 `free` / `race_to`；其他球种规范化为空值或 `legacy`，不会进入新分支。字段名保持通用，是为了 API、快照和客户端组件表达“整场赛制”，但本次不把它推广到其他球种。

不复用 `snooker_*`，因为斯诺克字段带有规则版本、旧 `best_of_frames` 映射和不同上限，复用会形成互相影响的兼容分支。不复用 `game_mode`，因为它不是目标胜局模型，覆盖它会改变现有接口和历史数据语义。也不新增一张赛制表；每场只有一个小型枚举和整数目标，直接存于 `matches` 最简单且读取无需额外查询。

### 2. 以显式声明区分新客户端赛制与 legacy 兼容

数据库列默认 `match_format='legacy'`、`target_wins=0`，迁移不回填存量记录。更新后的 App 在创建中八或美九时显式提交 `match_format='free'`、`target_wins=0`，因此用户看到的新对局默认是自由局数；显式提交 `race_to` 时服务端校验 1～65。

旧客户端不会发送新字段，服务端将缺省值规范化为 `legacy`，继续允许既有手动结束和旧计分入口。这比把所有缺省请求直接解释为 `free` 更安全，否则旧客户端在没有赛制 UI 和到点提示的情况下会被新结束约束锁住。也不增加单独的协议版本字段，因为是否显式提交 `match_format` 已足以完成兼容分流。

### 3. 使用通用赛制更新接口，并沿用 revision/snapshot 写协议

新增通用 `UpdateMatchFormat` 契约，输入 `match_id`、`match_format`、`target_wins`、`base_revision`，输出 `accepted`、`success`、`message`、`server_revision` 和完整 `MatchSyncSnapshot`。接口仅接受已由新版客户端创建的 `free` / `race_to` 对局，不允许客户端把新对局改回 `legacy`，也不允许把存量或旧客户端创建的 `legacy` 对局转换为新赛制。

事务内对 `matches` 行加锁并重新检查：当前用户是创建者、球种为 3/4、当前赛制为 `free` 或 `race_to`、对局仍在进行、结束状态为空、没有已绑定裁判、`base_revision` 一致、动作数与局数均为 0。成功后更新两个字段并 bump revision；失败返回当前权威快照。提交后向双方发送按用户视角构建的 `sync` 快照。

只检查页面本地的 `current_round` 会受陈旧状态和并发影响；只检查 `match_rounds` 又可能漏掉首局内的通用写分动作。因此同时以服务端动作数和局数作为锁定条件，与现有斯诺克首局前编辑模型保持一致。

### 4. 灵活赛制的唯一计分入口是事务化的 `EndRound`

`EndRound` 为 `free` / `race_to` 的中八、美九增加独立分支。该分支在一个数据库事务中完成行锁、幂等动作检查、revision 校验、局数读取、局记录写入、胜方总比分固定 `+1`、动作记录和结束状态转换。请求中的 `score` 必须为 1，服务端仍以固定 `+1` 计算而不信任请求数值；`win_type` 只用于记录该局胜法。

事务使用加锁后的数据判断结束条件：

- `free`：完成局数达到 129。
- `race_to`：任一方比分达到 `target_wins`；由于每局只有一个胜方且目标最大为 65，最长比分为 65:64，即 129 局。

判断与比分落库必须处于同一事务，避免双方并发结束一局时写出第 130 局、超过目标胜局或重复触发结算。保留现有 `client_action_id` 幂等键和单调 `server_revision`；事务提交后再执行通知与 WebSocket 等副作用。

不在客户端根据比分主动调用第二个“结束对局”请求，因为网络重试或另一端同时操作会产生中间状态；也不接受 `EndRound.score != 1`，否则请求语义仍可能被误解为一局增加多胜。`legacy`、斯诺克和九球追分继续走原分支，不改变现有请求语义。

### 5. 在所有通用改分入口阻止绕过

`MatchScore` 和 `MatchFoul` 在权限检查后、任何比分修改前识别 `game_type IN (3,4)` 且 `match_format IN ('free','race_to')`，返回拒绝响应和当前快照，提示使用“结束一局”。该限制只针对新灵活赛制；`legacy` 对局仍保留旧接口行为。

仅在到点后检查比分不足以防止绕过，因为一次通用加分可能直接越过目标且不会生成对应局记录。将保护放在写入口边界，可以维持“总比分之和等于已完成局数”的新赛制不变量，并让 129 局判断保持可靠。

### 6. 正常结束资格与练习/排位流程复用现有结算能力

为灵活中八、美九集中定义正常结束资格：`free` 在至少完成 1 局后可由用户结束，允许相同比分；`race_to` 仅在一方达到目标后可结束。`FinishMatch` 和排位 `request finish` 都复用该判断，未满足条件时返回当前快照而不改变状态。`legacy` 不受此限制。

当 `EndRound` 达到强制结束条件时：

- 练习赛在同一事务内复用现有 `settleMatchWithTx` 完成状态、结果和结算数据写入；平分时沿用 `result=3`。
- 无裁判的排位赛在同一事务内创建现有 `pending_confirmation` 结束请求及幂等 `finish_request` 动作，由对手通过既有确认/争议流程处理。
- 已绑定裁判的排位赛沿用现有裁判控制与直接结算语义，不额外叠加双方确认。

事务返回明确的“仅结束一局 / 已发起确认 / 已完成结算”结果，提交后复用现有结算后处理，确保段位、信誉、成就、通知与 `match_end` 只执行一次。达到条件后 `status` 或 `finish_state` 会阻止后续写分。

不新建一套 pool 专用结算服务，因为胜负、平局、排位确认和结算副作用已经由通用结束链路负责；本变更只增加何时允许或自动进入该链路的规则。

### 7. API、快照和 WebSocket 使用同一组赛制字段

在 `StartMatchReq`、`CurrentMatchInfo`、登录态/公开详情、`MatchSyncSnapshot` 以及相关响应中增加 `match_format`、`target_wins`、`can_change_match_format`。对于灵活中八、美九始终返回规范化值；`legacy` 返回 `legacy/0`；不支持的球种返回空值/0/false，避免客户端误开编辑器。

WebSocket 的 `sync`、`round_end`、`score_update`（兼容结构）和 `match_end` 携带同一组字段及最新 `status`、`finish_state`、`server_revision`。赛制更新继续发送双方各自视角的完整 `sync`；单局结束后先以事务最终状态构建快照，再广播，因此客户端无需自行推导是否已到点。

不让 HTTP 详情、写响应和 WebSocket 各自计算规则；它们统一调用同一规范化和 `canChange` helper，减少 `legacy` 判定或权限结果漂移。`.api` 是契约单一真源，修改后立即运行 goctl，不手改生成的 types 或 routes。

### 8. App 复用赛制交互形态，但使用 pool 专属纯规则 helper

App 创建中八、美九 payload 时显式带上默认 `free/0`。对局页在服务端 `can_change_match_format=true` 时展示首局前编辑入口，可切换自由局数或抢 1～65；不修改球种选择 Modal。格式化、范围校验、结束提示和可编辑判断沉到通用逐局赛制纯函数并补 `node:test`，页面只消费快照和调用 `app/api/match.js` 门面。

客户端不以本地比分作为最终结束依据；所有写响应和 WebSocket 都用 `server_revision` 与权威 snapshot 收敛。这样旧客户端仍可处理其 `legacy` 对局，新客户端也不会因断线重连丢失赛制或结束确认状态。

## Risks / Trade-offs

- [旧客户端创建的新对局没有赛制 UI] → 缺省请求保持 `legacy`；仅更新后的 App 显式创建 `free`，避免旧客户端被不可见的新规则约束。
- [双方并发结束一局导致超过 129 局或重复结算] → 在同一行锁事务内重新校验 revision、幂等动作、局数、比分与结束状态，并只基于事务最终值触发一次状态转换。
- [通用加分/犯规破坏“每局 1 胜”不变量] → 对 `free/race_to` 的 game type 3/4 在 `MatchScore`、`MatchFoul` 写入前硬拒绝，仅允许 `EndRound` 改总比分。
- [排位到点同时产生 round、finish request 和多次 revision，客户端收到乱序消息] → 每次持久化保持单调 revision，广播事务后的完整快照；客户端忽略旧 revision，而不是按消息到达顺序累加本地状态。
- [立即结算后外部通知或段位副作用重复] → 数据库内复用幂等结算动作，事务外复用现有 post-commit 流程及其去重语义，不从客户端补发第二次结束请求。
- [通用字段未来被其他球种误用] → 所有 normalization、创建校验、更新权限和 UI 展示都显式限定 game type 3/4；扩展其他球种需另行定义规则。
- [回滚服务后仍存在进行中的新赛制对局] → 常规回滚保留数据列并先停止新客户端创建；确认无进行中的 `free/race_to` 对局后再回退执行逻辑，避免中途丢失到点约束。

## Migration Plan

1. 新增 goose 迁移，通过 `information_schema.COLUMNS` 判断后为 `matches` 添加 `match_format VARCHAR(20) NOT NULL DEFAULT 'legacy'` 和 `target_wins SMALLINT UNSIGNED NOT NULL DEFAULT 0`；同步 Gorm `Match` 字段。Up 不回填为 `free`，因此所有存量行自然保持 `legacy`。
2. 修改 `backend/chasing_points.api` 后立即运行 goctl，再实现模型 helper、更新接口、详情/快照映射、WebSocket 字段、结束资格和写入口保护。
3. 先部署后端。此时旧客户端缺省创建仍为 `legacy`，不会发生行为变化；后端也能安全返回新增可选字段。
4. 部署 App。新版本开始对 game type 3/4 显式提交 `free`，并展示赛制编辑和到点/确认状态。
5. 通过后端测试覆盖 legacy、free 平局与 129 局、race_to 1/10/65、并发/幂等、练习直结、排位确认和 score/foul 拒绝；通过 App 测试覆盖创建 payload、范围校验、快照恢复与旧字段缺失降级。

回滚时先下线或回退新 App，停止创建新的灵活赛制对局。应用层回滚默认保留两列，额外列不会影响旧服务；等待或人工处理所有进行中的 `free/race_to` 对局后再回退后端。只有确认不再需要恢复赛制元数据时才执行 Down：同样通过 `information_schema` 条件执行删除列，避免旧 MySQL 不支持 `DROP COLUMN IF EXISTS`，并接受该步骤会丢失历史赛制字段。

## Open Questions

无。适用球种、平局语义、129 局上限、抢 1～65、编辑窗口以及练习/排位结束方式均已明确。
