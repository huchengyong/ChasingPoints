## Why

裁判接管对局后虽然拥有记分权限，但当前操作面板仍沿用选手视角的“给对手记分”模型，导致多数操作只能记到选手2，无法完整记录双方得分、胜局和犯规。该缺口会直接造成裁判模式下的比赛数据错误，需要让裁判以固定的选手1/选手2语义操作双方。

## What Changes

- 为裁判记分模式提供面向选手1和选手2的双向操作入口。
- 中式八球、美式九球等局胜模式允许裁判为任意一方选择获胜方式并结束当前局。
- 九球追分允许裁判为任意一方记录普胜、小金、大金，并明确选择犯规方。
- 斯诺克允许裁判明确选择得分方和犯规方，同时保留红球、彩球及清彩顺序校验。
- 选手视角继续保持现有相对的“我方/对手”行为，选手只读和裁判权限规则不变。
- 补充裁判双向记分的纯逻辑与页面结构测试，覆盖双方 actor/winner 映射。

## Capabilities

### New Capabilities
- `referee-bidirectional-scoring`: 定义裁判在各球种中以固定选手1/选手2身份为双方记分、判定胜局和记录犯规的行为。

### Modified Capabilities

无。

## Impact

- 主要影响 `app/subPages/match/playing.vue`、`app/subPages/match/playing.scss` 及对局角色/操作相关纯函数和测试。
- 继续使用现有 `MatchScoreReq.Actor`、`MatchFoulReq.Actor`、`EndRoundReq.Winner` 协议，不新增后端接口或数据库字段。
- 需要核对后端 actor/winner 的固定 player1/player2 语义，并按需修正 API 注释或增加回归测试，避免将裁判操作解释为“我/对手”。
- 不改变选手端对局流程、WebSocket 同步协议、revision 并发控制和结束比赛权限。
