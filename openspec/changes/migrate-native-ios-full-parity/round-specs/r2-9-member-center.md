# R2.9：VIP 中心 + 对局记录对齐小程序

## 本轮要读的小程序源码

- `uniapp/subPages/user/memberCenter.vue`（+ scss）、`uniapp/utils/member-center.js`（除已迁的 `resolveMemberGrowthCard` 外的全部导出）、`favorite-venue-reward.js` 的 `resolveFavoriteVenueMemberCard`、`member-ranking-rights.js`；测试 `member-center.test.mjs`、`member-growth-center.test.mjs`、`member-ranking-rights.test.mjs`、`favorite-venue-reward.test.mjs`（上次没搬的会员卡相关用例）
- `uniapp/api/member.js`；契约 `/api/member/plans`、`/api/member/status`（`GetMemberPlansResp`、`GetMemberStatusResp`）
- `uniapp/subPages/user/matchHistory.vue`（+ scss）：与原生现有 `iOS/iOS/MatchHistoryViews.swift` 对照
- 规格 `openspec/specs/` 里与会员权益相关的条目（如有）

## 任务

1. **VIP 中心**（`iOS/iOS/Features/Member/`）：`.memberCenter` → `MemberCenterView`，标题「VIP中心」。
   - `member-center.js` 其余导出、`resolveFavoriteVenueMemberCard`、`member-ranking-rights.js` 逐条迁移并搬运测试。**支付相关**（`getMemberPayChannelOptions`、`createMemberPaymentRequest`、`shouldTreatMemberOrderAsPaid`、下单与查单接口）**不迁**：原生端在接入 App 内购买之前不提供任何购买入口。
   - 页面在合规模式（`AppFlags.complianceMode == true`，当前即如此）下的呈现以 `memberCenter.vue` 里 `APP_COMPLIANCE_MODE` 为真的分支为准：身份预览、成长卡、权益说明、常玩球馆奖励卡、去「对局记录」/「表现分析」的入口等。非合规分支里的套餐与支付区块不实现；用 `AppFlags.canShowPaidMembershipEntry` 包住，为 false 时整块不出现。
   - 身份预览头像用 `MemberAvatarView`，段位取常用球种（`GameTypePreference`）对应的段位，规则以页面为准。
2. **对局记录对齐**：逐项比对 `matchHistory.vue` 与原生 `MatchHistoryView`，把小程序有而原生没有的展示与入口补上（例如：正在执裁的对局提示条、列表行的字段、空态文案与引导、时长显示），但**不改变**现有分页 / 身份隔离逻辑和既有测试的断言。跳转到尚未迁移页面（实时计分）的入口用占位路由 `.liveMatch(matchId:)`（如路由表里还没有，就新增这个 case，标题「对局」，目的地占位页）。逐条列出你比对出的差异以及每条的处理结果。

## 测试

搬运 JS 用例；VIP 中心页面模型（会员有效 / 已过期冻结 / 从未开通、合规模式下不含购买动作、失败重试、身份变化）；对局记录新增展示对应的模型 / 规则测试；快照：`member-center-active`、`member-center-expired`、`member-center-none`、`match-history-with-referee-banner`。

## 并行开发提醒

其他开发线正在修改 `iOS/iOS/Features/News/**`、`Features/Match/**`、`Features/Home/**`、`Features/Messages/**`、`Features/Friends/**`、`iOS/iOS/App/MainTabView.swift` 及其测试，本轮不要碰这些文件。
