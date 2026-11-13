import XCTest
@testable import iOS

@MainActor
final class AppRouteTests: XCTestCase {
    func testTitlesMatchPagesJSON() {
        let expected: [(AppRoute, String)] = [
            (.login, "登录"),
            (.settings, "设置"),
            (.editNickname, "编辑资料"),
            (.matchHistory, "对局记录"),
            (.matchReport(matchId: 1), "对局详情"),
            (.ranking, "排行榜"),
            (.spectatorLobby, "观赛大厅"),
            (.rankExplain, "段位说明"),
            (.memberCenter, "VIP中心"),
            (.statsDetail, "竞技分析"),
            (.achievementWall, "荣誉墙"),
            (.friendHomepage(userId: 1), "好友主页"),
            (.venueDetail(id: 1), "球馆详情"),
            (.seasonHistory, "历届赛季"),
            (.gearIndex, "我的装备"),
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
        XCTAssertEqual(AppRoute.matchReport(matchId: 3), AppRoute.matchReport(matchId: 3))
    }
}
