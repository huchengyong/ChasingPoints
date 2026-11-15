监工复核后的补充任务（仍遵守本会话开头的全部硬性约束与工程须知；不要做 git 写操作；只改 `iOS/iOS/**`、`iOS/iOSAuthTests/**`）。

`uniapp/subPages/match/matchDetail.vue` 在参与者视角下还有两样东西，原生 `MatchDetailView` 缺了：

1. **本场用杆卡**（模板约 104–105 行）：`<matchGearCard :match-id="matchId" :enabled="showMatchGear" />`，放在操作按钮行之后、九球追分卡之前。`showMatchGear` 用 `GearMatchRules.shouldShowMatchGear(status:viewerRole:isPlayer1:isSpectate:)`（参数取详情数据与观战模式，见约 280 行）。直接复用赛果页已经在用的 `MatchGearCardView(session:matchId:enabled:)`，不要另写一份。
2. **「再约一场」按钮**（约 98–101 行与 287–292、631–642 行）：显示条件 `viewer_role != "referee"` 且本人是参与者（`is_player1 == true` 或 `viewer_role` 是 `player1` / `player2`）且 `opponent_id > 0`；点了打开 `.challengeCompose(target: ChallengeComposeTarget(userId: opponent_id, nickname: opponent_name, avatar: opponent_avatar, gameType: game_type, matchMode: match_mode == "practice" ? "practice" : "ranked"))`。按钮放在「单场复盘」「查看双方交锋记录」同一行，样式与它们一致（小程序里多一个 `rematch-button` 类，按 scss 处理强调色）。如果详情数据里缺 `opponent_id` / `opponent_name` / `opponent_avatar` / `match_mode` 字段，按 `backend/chasing_points.api` 补到 `LiveMatchSnapshot` 或 `MatchDetailViewData` 里（可选字段）。

测试：显示条件（裁判、观战、对手 id 为 0 时都不显示）；路由值正确；本场用杆卡只在已结束的参与者视角显示。补一张快照 `match-detail-participant`（已结束、参与者视角，能看到三个按钮和用杆卡），亮暗各一张。

完成后照常跑 Debug / Release 构建与全量测试，交付说明只写本次补充的部分。仓库根目录不要留下 `snap-c/` 之类的临时目录，快照导出放到 `iOS/build/` 下。

3. **修快照 `match-result-gear-edit`**：现在导出的图只有「加载我的装备...」，没有拍到编辑器内容。按 `_common.md` 的做法先让编辑器模型把「我的装备」和本场用杆读完、并把皮头设成一个已选产品，再渲染；导出后自己看一眼图片，确认能看到球杆选择、皮头（已选）、巧克三行和「保存」按钮。
