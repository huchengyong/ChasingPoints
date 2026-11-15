import XCTest
@testable import iOS

/// PK 报表读取模型：请求参数、状态、失败横幅与登录隔离。
@MainActor
final class PKReportModelTests: XCTestCase {
    private let target = FriendHomepageTarget(
        userId: 18,
        name: "阿杰",
        avatar: "",
        rankName: "",
        vipLevel: 0,
        rankLevel: 0
    )

    func testOverviewLoadsStatsHistoryAndPosterPayload() async {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        server.stub("GET", "/api/match/h2h/overview", .json(SubPageFixtures.h2hOverview(
            items: [
                MatchHistoryFixtures.participantItem(id: 1, result: 1, myScore: 7, opponentScore: 5),
                MatchHistoryFixtures.participantItem(id: 2, result: 2, myScore: 3, opponentScore: 6),
            ],
            total: 12
        )))
        let model = PKReportModel(session: session, target: target)

        await model.onAppear()

        XCTAssertEqual(model.phase, .ready)
        XCTAssertEqual(model.stats?.totalMatches, 8)
        XCTAssertEqual(model.hero.title, "你对阿杰略占上风")
        XCTAssertEqual(model.hero.scoreText, "5 : 3")
        XCTAssertEqual(model.evidenceList, ["近 2 场 1 胜 1 负", "当前最长连胜 3 场"])
        XCTAssertEqual(model.posterPayload.winRateLabel, "63%")
        XCTAssertEqual(model.posterPayload.avgScoreDiffLabel, "+1.8")
        XCTAssertEqual(server.requests("/api/match/h2h/overview").first?.query, [
            "opponent_id": "18",
            "page_size": "5",
        ])
    }

    func testHiddenOverviewBecomesEmptyWithoutStats() async {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        server.stub("GET", "/api/match/h2h/overview", .json(SubPageFixtures.h2hOverview(hidden: true)))
        let model = PKReportModel(session: session, target: target)

        await model.onAppear()

        XCTAssertEqual(model.phase, .empty)
        XCTAssertNil(model.stats)
        XCTAssertTrue(model.history.isEmpty)
        XCTAssertNil(model.refreshError)
    }

    func testPartialAvailabilityFailsWithBusinessMessage() async {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        server.stub("GET", "/api/match/h2h/overview", .json(SubPageFixtures.h2hOverview(
            availability: ["opponent": true, "stats": false, "history": true]
        )))
        let model = PKReportModel(session: session, target: target)

        await model.onAppear()

        XCTAssertEqual(model.phase, .failed(LoadFailure(category: .business, message: PKReportRules.partialFailureMessage)))
        XCTAssertEqual(model.failureFeedback?.title, "PK 报表加载失败")
        XCTAssertEqual(model.failureFeedback?.description, PKReportRules.partialFailureMessage)
    }

    func testRefreshFailureKeepsLoadedData() async {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        server.stub("GET", "/api/match/h2h/overview", .json(SubPageFixtures.h2hOverview(
            items: [MatchHistoryFixtures.participantItem(id: 1)],
            total: 8
        )))
        let model = PKReportModel(session: session, target: target)
        await model.onAppear()
        XCTAssertEqual(model.phase, .ready)

        server.stub("GET", "/api/match/h2h/overview", .raw("down", status: 503))
        await model.retry()

        XCTAssertEqual(model.phase, .ready)
        XCTAssertEqual(model.history.map(\.id), [1])
        XCTAssertNotNil(model.refreshError)
    }

    func testGuestSeesSessionFailure() async {
        let server = MockHTTPServer()
        let session = makeSessionStore(server, credentialStore: InMemoryCredentialStore())
        let model = PKReportModel(session: session, target: target)

        await model.onAppear()

        XCTAssertEqual(model.phase, .failed(LoadFailure(category: .session, message: "登录后查看 PK 报表")))
        XCTAssertEqual(server.requestCount("/api/match/h2h/overview"), 0)
    }

    func testIdentityChangeClearsDataAndRefetches() async {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        server.stub("GET", "/api/match/h2h/overview", .json(SubPageFixtures.h2hOverview(
            items: [MatchHistoryFixtures.participantItem(id: 1)],
            total: 8
        )))
        let model = PKReportModel(session: session, target: target)
        await model.onAppear()
        XCTAssertEqual(model.phase, .ready)

        session.logout()
        await model.onAppear()

        XCTAssertEqual(model.phase, .failed(LoadFailure(category: .session, message: "登录后查看 PK 报表")))
        XCTAssertNil(model.stats)
        XCTAssertTrue(model.history.isEmpty)
    }
}
