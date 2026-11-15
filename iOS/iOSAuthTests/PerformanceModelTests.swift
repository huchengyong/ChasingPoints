import XCTest
@testable import iOS

@MainActor
final class PerformanceModelTests: XCTestCase {
    private let path = "/api/member/performance"

    func testFirstLoadRequestsPerformanceWithGameTypeAndBearer() async {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        server.stub("GET", path, .json(StatsFixtures.performance(
            summary: StatsFixtures.summary(),
            bestWin: StatsFixtures.matchItem(matchId: 9),
            worstLoss: StatsFixtures.matchItem(matchId: 10, opponentName: "老李", beforeScore: 1300, winRate: 80),
            months: StatsFixtures.defaultMonths()
        )))
        let model = PerformanceModel(session: session, gameType: 3)

        await model.loadIfNeeded()

        XCTAssertFalse(model.isInitialLoading)
        XCTAssertNil(model.failure)
        XCTAssertFalse(model.isLocked)
        XCTAssertEqual(model.summaryView.wins, 14)
        XCTAssertEqual(model.notableViews.map(\.matchId), [9, 10])
        XCTAssertTrue(model.chart.hasData)
        XCTAssertEqual(model.chart.bars.count, 6)
        XCTAssertEqual(server.requestCount(path), 1)
        XCTAssertEqual(server.requests(path).first?.query, ["game_type": "3"])
        XCTAssertEqual(server.requests(path).first?.bearerToken, AuthFixtures.oldAccess)
    }

    func testSecondAppearanceUsesPageCacheWithoutRequesting() async {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        server.stub("GET", path, .json(payload(gameType: 3)))
        let model = PerformanceModel(session: session, gameType: 3)

        await model.loadIfNeeded()
        await model.loadIfNeeded()

        XCTAssertEqual(server.requestCount(path), 1)
    }

    func testSwitchingToUnreadGameClearsContentAndShowsLoading() async {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        stubByGame(server, gatedCall: 2)
        let model = PerformanceModel(session: session, gameType: 3)
        await model.loadIfNeeded()
        XCTAssertEqual(model.summaryView.wins, 14)

        let switching = Task { await model.selectGameType(2) }
        await waitUntil { server.requestCount(self.path) == 2 }

        XCTAssertEqual(model.currentGameType, 2)
        XCTAssertTrue(model.isInitialLoading, "未读球种不能把旧球种内容挂在新球种下")
        XCTAssertNil(model.data)

        gate.signal()
        await switching.value

        XCTAssertEqual(model.currentGameType, 2)
        XCTAssertEqual(model.summaryView.wins, 7)
        XCTAssertFalse(model.isInitialLoading)
        XCTAssertEqual(server.requests(path).last?.query, ["game_type": "2"])
    }

    func testSwitchingBackToReadGameUsesCacheWithoutRequesting() async {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        stubByGame(server, gatedCall: nil)
        let model = PerformanceModel(session: session, gameType: 3)
        await model.loadIfNeeded()

        await model.selectGameType(2)
        XCTAssertEqual(model.summaryView.wins, 7)
        await model.selectGameType(3)

        XCTAssertEqual(model.summaryView.wins, 14)
        XCTAssertEqual(server.requestCount(path), 2)
    }

    func testStaleGameResponseIsDiscarded() async {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        stubByGame(server, gatedCall: 2)
        let model = PerformanceModel(session: session, gameType: 3)
        await model.loadIfNeeded()

        let stale = Task { await model.selectGameType(2) }
        await waitUntil { server.requestCount(self.path) == 2 }
        await model.selectGameType(4)
        XCTAssertEqual(model.summaryView.wins, 21)

        gate.signal()
        await stale.value

        XCTAssertEqual(model.currentGameType, 4)
        XCTAssertEqual(model.summaryView.wins, 21)
        XCTAssertFalse(model.isFetching)
    }

    func testEmptyMonthsRendersEmptyCurveButKeepsSummary() async {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        server.stub("GET", path, .json(StatsFixtures.performance(
            summary: StatsFixtures.summary(wins: 0, expectedWins: 0, winsDiff: 0, rating: nil),
            months: []
        )))
        let model = PerformanceModel(session: session, gameType: 3)

        await model.loadIfNeeded()

        XCTAssertNil(model.failure)
        XCTAssertFalse(model.chart.hasData)
        XCTAssertEqual(model.chart.bars, [])
        XCTAssertTrue(model.summaryView.available)
        XCTAssertEqual(model.summaryView.wins, 0)
        XCTAssertFalse(model.summaryView.rating.available)
        XCTAssertTrue(model.notableViews.isEmpty)
    }

    func testInitialFailureShowsErrorAndRetryLoadsContent() async {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        server.stub("GET", path, .raw("service down", status: 503))
        let model = PerformanceModel(session: session, gameType: 3)

        await model.loadIfNeeded()

        XCTAssertTrue(model.isInitialError)
        XCTAssertEqual(model.failure?.category, .server)
        XCTAssertNil(model.data)

        server.stub("GET", path, .json(payload(gameType: 3)))
        await model.retry()

        XCTAssertFalse(model.isInitialError)
        XCTAssertNil(model.failure)
        XCTAssertEqual(model.summaryView.wins, 14)
    }

    func testRefreshFailureKeepsContentAndShowsBanner() async {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        server.stub("GET", path, .json(payload(gameType: 3)))
        let model = PerformanceModel(session: session, gameType: 3)
        await model.loadIfNeeded()

        server.stub("GET", path, .raw("service down", status: 503))
        await model.retry()

        XCTAssertEqual(model.summaryView.wins, 14, "刷新失败保留已有内容")
        XCTAssertEqual(model.refreshFailure?.category, .server)
        XCTAssertEqual(model.refreshMessage, "请稍后重试")
        XCTAssertNil(model.failure)
        XCTAssertFalse(model.isInitialError)
        XCTAssertFalse(model.isFetching)
    }

    func testUnsupportedGameTypeFallsBackToDefault() async {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        server.stub("GET", path, .json(payload(gameType: 3)))
        let model = PerformanceModel(session: session, gameType: 9)

        await model.loadIfNeeded()

        XCTAssertEqual(model.currentGameType, 3)
        XCTAssertEqual(server.requests(path).first?.query["game_type"], "3")
    }

    func testGuestDoesNotRequest() async {
        let server = MockHTTPServer()
        let session = RankingFixtures.makeGuestSession(server)
        let model = PerformanceModel(session: session, gameType: 3)

        await model.loadIfNeeded()

        XCTAssertTrue(server.requests.isEmpty)
        XCTAssertNil(model.data)
        XCTAssertFalse(model.isInitialLoading)
        XCTAssertFalse(model.isInitialError)
    }

    func testLogoutClearsContentWithoutRequesting() async {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        server.stub("GET", path, .json(payload(gameType: 3)))
        let model = PerformanceModel(session: session, gameType: 3)
        await model.loadIfNeeded()
        let totalRequests = server.requests.count

        session.logout()
        await model.reloadForIdentityChange()

        XCTAssertNil(model.data)
        XCTAssertNil(model.failure)
        XCTAssertEqual(server.requests.count, totalRequests, "退出后不得继续请求")
    }

    func testLockedViewHasNoPurchaseActionUnderComplianceMode() async {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        server.stub("GET", path, .json(StatsFixtures.performance(
            locked: true,
            sampleCount: 12,
            monthPreview: StatsFixtures.monthPreview(month: "2026-04", sampleCount: 6)
        )))
        let model = PerformanceModel(session: session, gameType: 3)

        await model.loadIfNeeded()

        XCTAssertTrue(model.isLocked)
        let locked = model.lockedView
        XCTAssertEqual(locked?.sampleText, "12")
        XCTAssertEqual(locked?.preview.ratingText, "1678")
        XCTAssertFalse(locked?.showPurchase ?? true, "合规模式下不出现开通入口")
        XCTAssertFalse(AppFlags.canShowPaidMembershipEntry)
    }

    // MARK: - 辅助

    private let gate = DispatchSemaphore(value: 0)

    private func payload(gameType: Int) -> [String: Any] {
        StatsFixtures.performance(
            gameType: gameType,
            sampleCount: gameType == 2 ? 7 : 20,
            summary: StatsFixtures.summary(wins: gameType == 2 ? 7 : (gameType == 4 ? 21 : 14)),
            bestWin: StatsFixtures.matchItem(matchId: Int64(100 + gameType)),
            worstLoss: StatsFixtures.matchItem(matchId: Int64(200 + gameType)),
            months: StatsFixtures.defaultMonths()
        )
    }

    private func stubByGame(_ server: MockHTTPServer, gatedCall: Int?) {
        let calls = LockedCounter()
        server.stub("GET", path) { request in
            let call = calls.next()
            if call == gatedCall {
                _ = self.gate.wait(timeout: .now() + 5)
            }
            let gameType = Int(request.query["game_type"] ?? "3") ?? 3
            return .json(self.payload(gameType: gameType))
        }
    }
}
