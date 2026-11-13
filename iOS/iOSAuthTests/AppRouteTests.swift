import XCTest
@testable import iOS

@MainActor
final class AppRouteTests: XCTestCase {
    func testTitlesMatchPagesJSON() {
        let expected: [(AppRoute, String)] = [
            (.login, "登录"),
            (.settings, "设置"),
            (.editProfile, "编辑资料"),
            (.editNickname, "编辑昵称"),
            (.matchHistory, "对局记录"),
            (.matchReport(matchId: 1), "对局详情"),
            (.ranking, "排行榜"),
            (.spectatorLobby, "观赛大厅"),
            (.rankExplain(gameType: 3), "段位说明"),
            (.memberCenter, "VIP中心"),
            (.statsDetail(gameType: 2), "竞技分析"),
            (.achievementWall(gameType: 3), "荣誉墙"),
            (.friendHomepage(userId: 1), "好友主页"),
            (.venueDetail(id: 1), "球馆详情"),
            (.seasonHistory, "历届赛季"),
            (.gearIndex, "我的装备"),
            (.scanConfirm, "确认对局"),
            (.opponentSelector, "选择球友"),
            (.liveMatch(matchId: 1), "对局"),
        ]

        for (route, title) in expected {
            XCTAssertEqual(route.title, title, "路由 \(route) 的标题应与 pages.json 一致")
        }
    }

    func testParameterizedRoutesCompareByIdentifier() {
        XCTAssertNotEqual(AppRoute.matchReport(matchId: 1), AppRoute.matchReport(matchId: 2))
        XCTAssertNotEqual(AppRoute.h2hRecord(opponentId: 1), AppRoute.h2hRecord(opponentId: 2))
        XCTAssertNotEqual(AppRoute.achievementDetail(achievementId: 1), AppRoute.achievementDetail(achievementId: 2))
        XCTAssertNotEqual(AppRoute.friendHomepage(userId: 1), AppRoute.friendHomepage(userId: 2))
        XCTAssertNotEqual(AppRoute.tournamentDetail(id: 1), AppRoute.tournamentDetail(id: 2))
        XCTAssertNotEqual(AppRoute.venueDetail(id: 1), AppRoute.venueDetail(id: 2))
        XCTAssertNotEqual(AppRoute.matchReplay(matchId: 1), AppRoute.matchReplay(matchId: 2))
        XCTAssertNotEqual(AppRoute.rankExplain(gameType: 1), AppRoute.rankExplain(gameType: 2))
        XCTAssertNotEqual(AppRoute.statsDetail(gameType: 1), AppRoute.statsDetail(gameType: 2))
        XCTAssertNotEqual(AppRoute.achievementWall(gameType: 1), AppRoute.achievementWall(gameType: 2))
        XCTAssertNotEqual(AppRoute.challenges(showHistory: true), AppRoute.challenges(showHistory: false))
        XCTAssertEqual(AppRoute.challenges(showHistory: true), AppRoute.challenges(showHistory: true))
        XCTAssertEqual(AppRoute.matchReport(matchId: 3), AppRoute.matchReport(matchId: 3))
        XCTAssertNotEqual(AppRoute.liveMatch(matchId: 1), AppRoute.liveMatch(matchId: 2))
        XCTAssertEqual(AppRoute.liveMatch(matchId: 3), AppRoute.liveMatch(matchId: 3))
    }
}
