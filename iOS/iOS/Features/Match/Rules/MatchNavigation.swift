import Foundation

/// 小程序页面路径在原生端换成「去哪」的目的地；case 与参数以 JS 实际生成的路径和查询参数为准。
/// 不引用 AppRoute，避免与并行改动的导航表耦合。
enum MatchDestination: Equatable {
    /// `/subPages/match/matchDetail?match_id=...&mode=spectate`
    case spectate(matchId: Int64)
    /// `/subPages/match/matchReplay?match_id=...`
    case replay(matchId: Int64)
    /// `/subPages/user/matchHistory`
    case matchHistory
    /// `/pages/user/index`（Tab 页）
    case userCenter
}

/// 逐条对应小程序 `utils/match-navigation.js`。
enum MatchNavigation {
    static let historyTab: MatchDestination = .userCenter
    static let historyPage: MatchDestination = .matchHistory

    /// 赛果页只允许跳转一次：第一次消费返回 true 并置位，之后返回 false。
    static func consumeResultNavigationGuard(_ hasNavigatedToResult: inout Bool) -> Bool {
        if hasNavigatedToResult { return false }
        hasNavigatedToResult = true
        return true
    }

    /// 对局已结束，或服务端的当前对局已经不是这一场时，进行中页面必须离开。
    /// `currentMatchId == nil` 表示当前对局暂时未知，此时不离开。
    static func shouldLeavePlayingPage(
        pageMatchId: Int64?,
        detailStatus: Int?,
        currentMatchId: Int64? = nil
    ) -> Bool {
        let normalizedPageMatchId = pageMatchId ?? 0
        if normalizedPageMatchId == 0 { return false }

        let normalizedDetailStatus = detailStatus ?? 0
        if normalizedDetailStatus != 0 && normalizedDetailStatus != 1 {
            return true
        }

        if normalizedDetailStatus == 1 {
            guard let currentMatchId else { return false }
            return currentMatchId != normalizedPageMatchId
        }

        return false
    }
}
