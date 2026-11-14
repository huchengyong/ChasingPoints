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
            (.friendHomepage(target: Self.target(1)), "好友主页"),
            (.friendHonorWall(userId: 1), "荣誉墙"),
            (.friendPkReport(target: Self.target(1)), "PK报表"),
            (.gearFriend(userId: 1), "TA 的装备"),
            (.opponentRecord(target: nil), "过往对手"),
            (.venueDetail(id: 1), "球馆详情"),
            (.seasonIndex, "赛季"),
            (.seasonHistory(gameType: 3), "历届赛季"),
            (.seasonReport(seasonId: 1, gameType: 3), "赛季报告"),
            (.gearIndex, "我的装备"),
            (.scanConfirm, "确认对局"),
            (.opponentSelector, "选择球友"),
            (.feedback, "帮助、投诉与举报"),
            (.userAgreement, "用户协议"),
            (.privacyPolicy, "隐私政策"),
            (.liveMatch(matchId: 1), "对局"),
        ]

        for (route, title) in expected {
            XCTAssertEqual(route.title, title, "路由 \(route) 的标题应与 pages.json 一致")
        }
    }

    func testParameterizedRoutesCompareByIdentifier() {
        XCTAssertNotEqual(AppRoute.matchReport(matchId: 1), AppRoute.matchReport(matchId: 2))
        XCTAssertNotEqual(
            AppRoute.h2hRecord(opponentId: 1, targetUserId: 0, targetName: "", targetAvatar: "", opponentName: "甲"),
            AppRoute.h2hRecord(opponentId: 2, targetUserId: 0, targetName: "", targetAvatar: "", opponentName: "甲")
        )
        XCTAssertNotEqual(
            AppRoute.h2hRecord(opponentId: 1, targetUserId: 0, targetName: "", targetAvatar: "", opponentName: "甲"),
            AppRoute.h2hRecord(opponentId: 1, targetUserId: 52, targetName: "球友 B", targetAvatar: "", opponentName: "甲")
        )
        XCTAssertNotEqual(AppRoute.achievementDetail(achievementId: 1), AppRoute.achievementDetail(achievementId: 2))
        XCTAssertNotEqual(AppRoute.friendHomepage(target: Self.target(1)), AppRoute.friendHomepage(target: Self.target(2)))
        XCTAssertNotEqual(AppRoute.tournamentDetail(id: 1), AppRoute.tournamentDetail(id: 2))
        XCTAssertNotEqual(AppRoute.venueDetail(id: 1), AppRoute.venueDetail(id: 2))
        XCTAssertNotEqual(AppRoute.matchReplay(matchId: 1), AppRoute.matchReplay(matchId: 2))
        XCTAssertNotEqual(AppRoute.rankExplain(gameType: 1), AppRoute.rankExplain(gameType: 2))
        XCTAssertNotEqual(AppRoute.statsDetail(gameType: 1), AppRoute.statsDetail(gameType: 2))
        XCTAssertNotEqual(AppRoute.achievementWall(gameType: 1), AppRoute.achievementWall(gameType: 2))
        XCTAssertNotEqual(AppRoute.seasonHistory(gameType: 1), AppRoute.seasonHistory(gameType: 2))
        XCTAssertNotEqual(AppRoute.seasonReport(seasonId: 1, gameType: 3), AppRoute.seasonReport(seasonId: 2, gameType: 3))
        XCTAssertNotEqual(AppRoute.seasonReport(seasonId: 1, gameType: 3), AppRoute.seasonReport(seasonId: 1, gameType: 2))
        XCTAssertNotEqual(AppRoute.challenges(tab: .history), AppRoute.challenges(tab: .received))
        XCTAssertEqual(AppRoute.challenges(tab: .history), AppRoute.challenges(tab: .history))
        XCTAssertEqual(AppRoute.matchReport(matchId: 3), AppRoute.matchReport(matchId: 3))
        XCTAssertNotEqual(AppRoute.liveMatch(matchId: 1), AppRoute.liveMatch(matchId: 2))
        XCTAssertEqual(AppRoute.liveMatch(matchId: 3), AppRoute.liveMatch(matchId: 3))
        XCTAssertNotEqual(AppRoute.friendHonorWall(userId: 1), AppRoute.friendHonorWall(userId: 2))
        XCTAssertNotEqual(AppRoute.friendPkReport(target: Self.target(1)), AppRoute.friendPkReport(target: Self.target(2)))
        XCTAssertNotEqual(AppRoute.gearFriend(userId: 1), AppRoute.gearFriend(userId: 2))
        XCTAssertNotEqual(
            AppRoute.opponentRecord(target: nil),
            AppRoute.opponentRecord(target: OpponentRecordTarget(userId: 1, name: "球友", avatar: ""))
        )
        XCTAssertEqual(
            AppRoute.opponentRecord(target: OpponentRecordTarget(userId: 1, name: "球友", avatar: "")),
            AppRoute.opponentRecord(target: OpponentRecordTarget(userId: 1, name: "球友", avatar: ""))
        )
    }

    private static func target(_ userId: Int64) -> FriendHomepageTarget {
        FriendHomepageTarget(
            userId: userId,
            name: "球友",
            avatar: "",
            rankName: "",
            vipLevel: 0,
            rankLevel: 0
        )
    }
}
