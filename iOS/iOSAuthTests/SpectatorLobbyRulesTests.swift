import XCTest
@testable import iOS

@MainActor
final class SpectatorLobbyRulesTests: XCTestCase {
    func testListParamsKeepHallStatusAndGameTypeFiltersExplicit() {
        XCTAssertEqual(
            SpectatorLobbyRules.listParams(scope: "hall", status: 2, gameType: 2, page: 3, pageSize: 20),
            [
                "scope": "hall",
                "status": "2",
                "game_type": "2",
                "page": "3",
                "page_size": "20",
            ]
        )
    }

    func testListParamsOmitStatusForFriendScopeAndKeepGameTypeFilter() {
        XCTAssertEqual(
            SpectatorLobbyRules.listParams(scope: "friends", status: 1, gameType: 3, page: 1, pageSize: 20),
            [
                "scope": "friends",
                "game_type": "3",
                "page": "1",
                "page_size": "20",
            ]
        )
    }

    func testEmptyStateSeparatesHallFinishedFriendAndGameTypeCopy() {
        XCTAssertEqual(
            SpectatorLobbyRules.emptyState(scope: "hall", status: 1, gameType: 0),
            SpectatorLobbyEmptyState(title: "暂无正在进行的对局", subtitle: "公开对局开打后会出现在这里")
        )
        XCTAssertEqual(
            SpectatorLobbyRules.emptyState(scope: "hall", status: 2, gameType: 0),
            SpectatorLobbyEmptyState(title: "暂无已结束对局", subtitle: "有新的完赛记录后会出现在这里")
        )
        XCTAssertEqual(
            SpectatorLobbyRules.emptyState(scope: "friends", status: 1, gameType: 2),
            SpectatorLobbyEmptyState(title: "暂无九球追分好友对局", subtitle: "好友参与该球种的对局会集中展示在这里")
        )
        XCTAssertEqual(
            SpectatorLobbyRules.emptyState(scope: "hall", status: 1, gameType: 2),
            SpectatorLobbyEmptyState(title: "暂无九球追分对局", subtitle: "该球种有新的公开对局后会出现在这里")
        )
        XCTAssertEqual(
            SpectatorLobbyRules.emptyState(scope: "friends", status: 1, gameType: 0),
            SpectatorLobbyEmptyState(title: "暂无好友对局", subtitle: "好友参与的进行中和已结束对局会展示在这里")
        )
    }

    func testStatusTextFormatsOngoingDurationAndFinishedCopy() {
        XCTAssertEqual(SpectatorLobbyRules.statusText(status: 1, durationSeconds: 65), "进行中 1分")
        XCTAssertEqual(SpectatorLobbyRules.statusText(status: 2, durationSeconds: 3600), "已结束")
        XCTAssertEqual(SpectatorLobbyRules.formatDuration(0), "0分")
        XCTAssertEqual(SpectatorLobbyRules.formatDuration(45), "45秒")
        XCTAssertEqual(SpectatorLobbyRules.formatDuration(3660), "1小时1分")
        XCTAssertEqual(SpectatorLobbyRules.formatDuration(90_000), "1天1小时")
    }

    func testFinishedMatchUsesRelativeEndTimeBeforeStatusCopy() {
        let now = Date(timeIntervalSince1970: 1_800_000_000)
        let endedAt = ISO8601DateFormatter().string(from: now.addingTimeInterval(-7200))

        let card = SpectatorLobbyRules.normalize(
            SpectatorLobbyFixtures.match(id: 9, status: 2, endTime: endedAt),
            now: now
        )

        XCTAssertEqual(card.statusText, "2小时前结束")
    }

    func testShouldOpenPlayingOnlyTreatsMyOngoingMatchAsEditable() {
        let live = SpectatorLobbyRules.normalize(
            SpectatorLobbyFixtures.match(id: 1, status: 1, player1Id: 18, player2Id: 20),
            userId: 18
        )
        XCTAssertTrue(SpectatorLobbyRules.shouldOpenPlaying(live, userId: 18, isLoggedIn: true))
        XCTAssertFalse(SpectatorLobbyRules.shouldOpenPlaying(live, userId: 18, isLoggedIn: false))
        XCTAssertFalse(SpectatorLobbyRules.shouldOpenPlaying(live, userId: 0, isLoggedIn: true))

        let finished = SpectatorLobbyRules.normalize(
            SpectatorLobbyFixtures.match(id: 2, status: 2, player1Id: 18, player2Id: 20),
            userId: 18
        )
        XCTAssertFalse(SpectatorLobbyRules.shouldOpenPlaying(finished, userId: 18, isLoggedIn: true))
    }

    func testFinishedMatchRoutesParticipantsToReportAndOthersToPublicDetail() {
        let mine = SpectatorLobbyRules.normalize(
            SpectatorLobbyFixtures.match(id: 88, status: 2, player1Id: 18, player2Id: 52),
            userId: 18
        )
        XCTAssertEqual(SpectatorLobbyRules.destination(for: mine, userId: 18, isLoggedIn: true), .report)

        let mineAsSecond = SpectatorLobbyRules.normalize(
            SpectatorLobbyFixtures.match(id: 88, status: 2, player1Id: 52, player2Id: 18),
            userId: 18
        )
        XCTAssertEqual(SpectatorLobbyRules.destination(for: mineAsSecond, userId: 18, isLoggedIn: true), .report)

        let others = SpectatorLobbyRules.normalize(
            SpectatorLobbyFixtures.match(id: 88, status: 2, player1Id: 52, player2Id: 77),
            userId: 18
        )
        XCTAssertEqual(SpectatorLobbyRules.destination(for: others, userId: 18, isLoggedIn: true), .publicDetail)

        let guest = SpectatorLobbyRules.normalize(
            SpectatorLobbyFixtures.match(id: 88, status: 2, player1Id: 18, player2Id: 52),
            userId: 0
        )
        XCTAssertEqual(SpectatorLobbyRules.destination(for: guest, userId: 0, isLoggedIn: false), .publicDetail)

        let live = SpectatorLobbyRules.normalize(
            SpectatorLobbyFixtures.match(id: 89, status: 1, player1Id: 18, player2Id: 52),
            userId: 18
        )
        XCTAssertEqual(SpectatorLobbyRules.destination(for: live, userId: 18, isLoggedIn: true), .live)
    }

    func testLobbyTabsHideFriendsForGuestsAndMapToScopeAndStatusQueries() {
        XCTAssertEqual(SpectatorLobbyRules.tabs(isLoggedIn: false).map(\.label), ["进行中", "已结束"])
        XCTAssertEqual(SpectatorLobbyRules.tabs(isLoggedIn: true).map(\.label), ["好友", "进行中", "已结束"])
        XCTAssertEqual(SpectatorLobbyRules.query(for: "friends"), SpectatorLobbyQuery(scope: "friends", status: 0))
        XCTAssertEqual(SpectatorLobbyRules.query(for: "live"), SpectatorLobbyQuery(scope: "hall", status: 1))
        XCTAssertEqual(SpectatorLobbyRules.query(for: "finished"), SpectatorLobbyQuery(scope: "hall", status: 2))
        XCTAssertEqual(SpectatorLobbyRules.query(for: "unknown"), SpectatorLobbyQuery(scope: "hall", status: 1))
    }

    func testNormalizeMarksTheViewerAndOnlyFinishedWinners() {
        let live = SpectatorLobbyRules.normalize(
            SpectatorLobbyFixtures.match(
                id: 1,
                status: 1,
                gameType: 3,
                gameTypeName: "中式八球",
                matchMode: "ranked",
                player1Id: 18,
                player1Name: "我",
                player1Score: 4,
                player2Id: 20,
                player2Name: "阿杰",
                player2Score: 2,
                durationSeconds: 65
            ),
            userId: 18
        )

        XCTAssertTrue(live.isLive)
        XCTAssertTrue(live.isMine)
        XCTAssertEqual(live.metaText, "中式八球 · 排位")
        XCTAssertEqual(live.statusText, "进行中 1分")
        XCTAssertEqual(live.players.map { [$0.name, String($0.score), String($0.isMe), $0.outcome?.rawValue ?? ""] }, [
            ["我", "4", "true", ""],
            ["阿杰", "2", "false", ""],
        ])

        let finished = SpectatorLobbyRules.normalize(
            SpectatorLobbyFixtures.match(
                id: 2,
                status: 2,
                gameType: 1,
                gameTypeName: "",
                matchMode: "practice",
                player1Id: 30,
                player1Name: "",
                player1Score: 3,
                player2Id: 31,
                player2Name: "球友 B",
                player2Score: 5
            ),
            userId: 18
        )

        XCTAssertFalse(finished.isLive)
        XCTAssertFalse(finished.isMine)
        XCTAssertEqual(finished.metaText, "斯诺克 · 练习")
        XCTAssertEqual(finished.statusText, "已结束")
        XCTAssertEqual(finished.players.map { [$0.name, $0.outcome?.rawValue ?? ""] }, [
            ["选手1", "loser"],
            ["球友 B", "winner"],
        ])

        let draw = SpectatorLobbyRules.normalize(
            SpectatorLobbyFixtures.match(id: 3, status: 2, player1Score: 2, player2Score: 2)
        )
        XCTAssertEqual(draw.players.map(\.outcome), [nil, nil])
    }

    func testSectionsGroupOnlyTheMixedFriendsTab() {
        let live = SpectatorLobbyRules.normalize(SpectatorLobbyFixtures.match(id: 1, status: 1))
        let ended = SpectatorLobbyRules.normalize(SpectatorLobbyFixtures.match(id: 2, status: 2))

        let friendSections = SpectatorLobbyRules.sections([ended, live], tabValue: "friends")
        XCTAssertEqual(friendSections.map(\.title), ["进行中", "已结束"])
        XCTAssertEqual(friendSections.map { $0.items.map(\.id) }, [[1], [2]])
        XCTAssertEqual(
            SpectatorLobbyRules.sections([ended], tabValue: "friends").map(\.key),
            ["finished"]
        )
        XCTAssertEqual(
            SpectatorLobbyRules.sections([live], tabValue: "live").map(\.key),
            ["live"]
        )
        XCTAssertTrue(SpectatorLobbyRules.sections([live], tabValue: "live").first?.title.isEmpty ?? false)
        XCTAssertEqual(SpectatorLobbyRules.sections([], tabValue: "live"), [])
    }

    func testGameTypeOptionsIncludeAllOptionFirst() {
        XCTAssertEqual(SpectatorLobbyRules.gameTypeOptions.map(\.label), [
            "全部球种",
            "中式八球",
            "九球追分",
            "斯诺克",
            "美式九球",
        ])
        XCTAssertEqual(SpectatorLobbyRules.gameTypeOptions.map(\.value), [0, 3, 2, 1, 4])
    }

    func testHomeLivePreviewShowsAtMostThreeMatchesAndSkipsTheViewerCurrentMatch() {
        XCTAssertEqual(SpectatorLobbyRules.homeLiveMatchLimit, 3)

        let four = (1...4).map { SpectatorLobbyFixtures.match(id: Int64($0)) }
        XCTAssertEqual(SpectatorLobbyRules.buildHomeLiveMatches(four).map(\.id), [1, 2, 3])
        XCTAssertEqual(
            SpectatorLobbyRules.buildHomeLiveMatches(four, currentMatchId: 2).map(\.id),
            [1, 3, 4]
        )
        XCTAssertEqual(SpectatorLobbyRules.buildHomeLiveMatches([SpectatorLobbyFixtures.match(id: 2)], currentMatchId: 2), [])
    }

    func testHomeLivePreviewReusesTheLobbyCardShapeAndMarksTheViewer() throws {
        let card = try XCTUnwrap(
            SpectatorLobbyRules.buildHomeLiveMatches(
                [SpectatorLobbyFixtures.match(id: 1, player1Id: 7, player1Name: "甲1")],
                userId: 7
            ).first
        )

        XCTAssertTrue(card.isLive)
        XCTAssertTrue(card.isMine)
        XCTAssertEqual(card.metaText, "中式八球 · 练习")
        XCTAssertTrue(card.statusText.hasPrefix("进行中"))
        XCTAssertEqual(card.players.map { [$0.name, String($0.score), String($0.isMe)] }, [
            ["甲1", "1", "true"],
            ["乙", "0", "false"],
        ])
    }

    func testHomeLivePreviewToleratesAMissingList() {
        XCTAssertEqual(SpectatorLobbyRules.buildHomeLiveMatches(nil), [])
    }

    func testNewRoutesCarryPageTitles() {
        XCTAssertEqual(AppRoute.spectatorLobby.title, "观赛大厅")
        XCTAssertEqual(AppRoute.liveMatch(matchId: 1).title, "对局")
        XCTAssertEqual(AppRoute.publicMatchDetail(matchId: 1).title, "对局详情")
        XCTAssertNotEqual(AppRoute.liveMatch(matchId: 1), AppRoute.liveMatch(matchId: 2))
        XCTAssertNotEqual(AppRoute.publicMatchDetail(matchId: 1), AppRoute.publicMatchDetail(matchId: 2))
    }
}
