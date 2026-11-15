import XCTest
@testable import iOS

@MainActor
final class StatsDetailModelTests: XCTestCase {
    private let path = "/api/stats/overview"
    private let april = AppFormat.parseDateTime("2026-04-15 12:00:00")!

    func testFirstLoadRequestsOneOverviewWithGameTypeAndTrendLimit() async {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        server.stub("GET", path, .json(StatsFixtures.overview()))
        let model = model(session)

        await model.loadIfNeeded()

        XCTAssertTrue(model.hasLoadedOnce)
        XCTAssertFalse(model.isInitialLoading)
        XCTAssertNil(model.failure)
        XCTAssertEqual(model.currentStats?.wins, 26)
        XCTAssertEqual(model.currentStats?.winRate, 61.9)
        XCTAssertEqual(model.duration, StatsDurationValues(average: 125, fastest: 40, longest: 3720, total: 42))
        XCTAssertEqual(model.highScore, StatsHighScoreSummary(personalBest: 87, monthBest: 87))
        XCTAssertEqual(model.opponentRows.count, 3)
        XCTAssertEqual(model.trendCalendar.summary.matchCount, 4)
        XCTAssertEqual(model.trendCalendar.summary.winRate, 50)
        XCTAssertEqual(model.rankTrend.currentScore, 1040)
        XCTAssertEqual(model.rankTrend.peakScore, 1055)
        XCTAssertEqual(model.rankDeltaChart.summary.upDays, 2)
        XCTAssertEqual(server.requestCount(path), 1)
        XCTAssertEqual(server.requests(path).first?.query, [
            "game_type": "3",
            "trend_limit": "100",
        ])
        XCTAssertEqual(server.requests(path).first?.bearerToken, AuthFixtures.oldAccess)
    }

    func testSecondAppearanceUsesPageCacheWithoutRequesting() async {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        server.stub("GET", path, .json(StatsFixtures.overview()))
        let model = model(session)

        await model.loadIfNeeded()
        await model.loadIfNeeded()

        XCTAssertEqual(server.requestCount(path), 1)
    }

    func testSwitchingToUnreadGameKeepsOldContentAndShowsPending() async {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        stubByGame(server, gatedCall: 2)
        let model = model(session)
        await model.loadIfNeeded()
        XCTAssertEqual(model.highScore.personalBest, 87)

        let switching = Task { await model.selectGameType(2) }
        await waitUntil { server.requestCount(path) == 2 }

        XCTAssertEqual(model.currentGameType, 2)
        XCTAssertTrue(model.isRefreshing)
        XCTAssertFalse(model.isInitialLoading)
        XCTAssertEqual(model.highScore.personalBest, 87, "新数据到达前保留旧内容")

        gate.signal()
        await switching.value

        XCTAssertEqual(model.currentGameType, 2)
        XCTAssertEqual(model.highScore.personalBest, 55)
        XCTAssertFalse(model.isRefreshing)
    }

    func testSwitchingBackToReadGameUsesCacheWithoutRequesting() async {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        stubByGame(server, gatedCall: nil)
        let model = model(session)
        await model.loadIfNeeded()

        await model.selectGameType(2)
        XCTAssertEqual(model.highScore.personalBest, 55)
        await model.selectGameType(3)

        XCTAssertEqual(model.highScore.personalBest, 87)
        XCTAssertEqual(server.requestCount(path), 2)
    }

    func testStaleGameResponseIsDiscarded() async {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        stubByGame(server, gatedCall: 2)
        let model = model(session)
        await model.loadIfNeeded()

        let stale = Task { await model.selectGameType(2) }
        await waitUntil { server.requestCount(path) == 2 }
        await model.selectGameType(4)
        XCTAssertEqual(model.highScore.personalBest, 444)

        gate.signal()
        await stale.value

        XCTAssertEqual(model.currentGameType, 4)
        XCTAssertEqual(model.highScore.personalBest, 444)
        XCTAssertFalse(model.isRefreshing)
    }

    func testEmptyOverviewStillRendersEmptySections() async {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        server.stub("GET", path, .json(StatsFixtures.overview(
            byGameType: [],
            recentTrend: [],
            rankScoreTrend: [],
            singleHighScores: [],
            duration: [
                "average_seconds": 0,
                "fastest_seconds": 0,
                "longest_seconds": 0,
                "total_matches": 0,
            ],
            opponentStrength: []
        )))
        let model = model(session)

        await model.loadIfNeeded()

        XCTAssertTrue(model.hasLoadedOnce)
        XCTAssertNil(model.failure)
        XCTAssertNil(model.currentStats)
        XCTAssertEqual(model.opponentRows, [])
        XCTAssertEqual(model.highScore, StatsHighScoreSummary(personalBest: 0, monthBest: 0))
        XCTAssertEqual(model.rankDeltaChart.summary.upDays + model.rankDeltaChart.summary.downDays, 0)
        XCTAssertEqual(model.trendCalendar.summary.matchCount, 0)
    }

    func testInitialFailureShowsErrorAndRetryLoadsContent() async {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        server.stub("GET", path, .raw("service down", status: 503))
        let model = model(session)

        await model.loadIfNeeded()

        XCTAssertTrue(model.isInitialError)
        XCTAssertEqual(model.failure?.category, .server)
        XCTAssertFalse(model.hasLoadedOnce)
        XCTAssertNil(model.overview)

        server.stub("GET", path, .json(StatsFixtures.overview()))
        await model.retry()

        XCTAssertFalse(model.isInitialError)
        XCTAssertTrue(model.hasLoadedOnce)
        XCTAssertEqual(model.currentStats?.wins, 26)
    }

    func testRefreshFailureKeepsContentAndShowsBanner() async {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        server.stub("GET", path, .json(StatsFixtures.overview()))
        let model = model(session)
        await model.loadIfNeeded()

        server.stub("GET", path, .raw("service down", status: 503))
        await model.selectGameType(2)

        XCTAssertNotNil(model.overview)
        XCTAssertEqual(model.refreshFailure?.category, .server)
        XCTAssertEqual(model.refreshMessage, "请稍后重试")
        XCTAssertNil(model.failure)
        XCTAssertFalse(model.isInitialError)
        XCTAssertFalse(model.isRefreshing)
    }

    func testMonthSwitchFiltersLocallyWithoutRequesting() async {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        server.stub("GET", path, .json(StatsFixtures.overview()))
        let model = model(session)
        await model.loadIfNeeded()

        model.shiftMonth(by: 1)

        XCTAssertEqual(model.selectedMonthKey, "2026-05")
        XCTAssertEqual(model.trendCalendar.title, "2026年5月")
        XCTAssertEqual(model.rankDeltaChart.title, "2026年5月")
        XCTAssertEqual(server.requestCount(path), 1)

        model.shiftMonth(by: -1)
        XCTAssertEqual(model.selectedMonthKey, "2026-04")
    }

    func testUnsupportedGameTypeFallsBackToDefault() async {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        server.stub("GET", path, .json(StatsFixtures.overview()))
        let model = model(session, gameType: 9)

        await model.loadIfNeeded()

        XCTAssertEqual(model.currentGameType, 3)
        XCTAssertEqual(server.requests(path).first?.query["game_type"], "3")
    }

    func testGuestDoesNotRequest() async {
        let server = MockHTTPServer()
        let session = RankingFixtures.makeGuestSession(server)
        let model = model(session)

        await model.loadIfNeeded()

        XCTAssertTrue(server.requests.isEmpty)
        XCTAssertFalse(model.hasLoadedOnce)
        XCTAssertFalse(model.isInitialError)
        XCTAssertNil(model.overview)
    }

    func testLogoutClearsContentWithoutRequesting() async {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        server.stub("GET", path, .json(StatsFixtures.overview()))
        let model = model(session)
        await model.loadIfNeeded()
        let totalRequests = server.requests.count

        session.logout()
        await model.reloadForIdentityChange()

        XCTAssertNil(model.overview)
        XCTAssertFalse(model.hasLoadedOnce)
        XCTAssertNil(model.failure)
        XCTAssertEqual(server.requests.count, totalRequests, "退出后不得继续请求")
    }

    // MARK: - 辅助

    private var gate: DispatchSemaphore { Self.gate }

    private static let gate = DispatchSemaphore(value: 0)

    private func model(_ session: SessionStore, gameType: Int = 3) -> StatsDetailModel {
        StatsDetailModel(session: session, gameType: gameType, now: { [april] in april })
    }

    private func stubByGame(_ server: MockHTTPServer, gatedCall: Int?) {
        let calls = LockedCounter()
        server.stub("GET", path) { request in
            let call = calls.next()
            if call == gatedCall {
                _ = Self.gate.wait(timeout: .now() + 5)
            }
            let gameType = Int(request.query["game_type"] ?? "3") ?? 3
            let score = gameType == 2 ? 55 : gameType == 4 ? 444 : 87
            return .json(StatsFixtures.overview(
                singleHighScores: [
                    StatsFixtures.highScore(matchId: 1, score: score, date: "2026-04-08"),
                ]
            ))
        }
    }
}
