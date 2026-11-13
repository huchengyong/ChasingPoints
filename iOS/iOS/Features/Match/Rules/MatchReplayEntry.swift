import Foundation

/// 逐条对应小程序 `utils/match-replay-entry.js`。
/// 「单场复盘」只对已完成对局的选手展示；资格最终由服务端判断。
enum MatchReplayEntry {
    private static let playerRoles: Set<String> = ["player1", "player2"]

    static func replayDestination(matchId: Int64) -> MatchDestination {
        .replay(matchId: matchId)
    }

    /// 本人对局详情带 `viewer_role`；战绩列表进入的公开详情没有 `viewer_role`，
    /// 只有当视角用户就是当前登录用户时才算本人的对局。
    static func canOpenMatchReplay(
        status: Int = 0,
        viewerRole: String = "",
        source: String = "",
        perspectiveUserId: Int64 = 0,
        currentUserId: Int64 = 0
    ) -> Bool {
        if status != 2 { return false }
        if playerRoles.contains(viewerRole) { return true }
        if viewerRole == "referee" { return false }
        return source == "history" && perspectiveUserId > 0 && perspectiveUserId == currentUserId
    }
}
