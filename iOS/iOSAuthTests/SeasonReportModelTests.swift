import XCTest
@testable import iOS

@MainActor
final class SeasonReportModelTests: XCTestCase {
    private let path = "/api/season/report"

    func testFirstLoadReadsReportAndBuildsRows() async {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        server.stub("GET", path, .json(SeasonFixtures.report(
            topAchievements: [SeasonFixtures.achievement]
        )))
        let model = SeasonReportModel(session: session, seasonId: 2, gameType: 3)

        await model.loadIfNeeded()

        XCTAssertEqual(model.phase, .ready)
        XCTAssertEqual(model.winRate, 75)
        XCTAssertEqual(model.typeWinRows.map(\.type), [1, 3])
        XCTAssertEqual(model.typeWinRows.map(\.label), ["斯诺克", "中式八球"])
        XCTAssertEqual(model.typeWinRows.map(\.wins), [2, 6])
        XCTAssertEqual(model.trendText, "趋势记录：1250 / 1280 / 1310")
        XCTAssertEqual(model.report?.topAchievements?.first?.name, "赛季十胜")

        let request = server.requests(path).first
        XCTAssertEqual(request?.query["season_id"], "2")
        XCTAssertEqual(request?.query["game_type"], "3")
        XCTAssertEqual(request?.bearerToken, AuthFixtures.oldAccess)
    }

    func testMissingSeasonShowsNotFoundWithoutRequest() async {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        let model = SeasonReportModel(session: session, seasonId: 0, gameType: 3)

        await model.loadIfNeeded()

        guard case .failed(let failure) = model.phase else {
            XCTFail("非法赛季应直接给出失败提示，实际为 \(model.phase)")
            return
        }
        XCTAssertEqual(failure.category, .notFound)
        XCTAssertEqual(server.requestCount(path), 0)
    }

    func testEmptyReportShowsEmptyPhase() async {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        server.stub("GET", path, .json(SeasonFixtures.emptyReport))
        let model = SeasonReportModel(session: session, seasonId: 2, gameType: 3)

        await model.loadIfNeeded()

        XCTAssertEqual(model.phase, .empty)
        XCTAssertNil(model.report)
    }

    func testSwitchingGameTypeReloadsWithNewQuery() async {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        server.stub("GET", path) { request in
            let gameType = Int(request.query["game_type"] ?? "3") ?? 3
            return .json(SeasonFixtures.report(
                winRateByType: [String(gameType): gameType]
            ))
        }
        let model = SeasonReportModel(session: session, seasonId: 2, gameType: 3)
        await model.loadIfNeeded()

        await model.selectGameType(2)

        XCTAssertEqual(model.currentGameType, 2)
        XCTAssertEqual(model.typeWinRows.map(\.type), [2])
        XCTAssertEqual(model.typeWinRows.map(\.wins), [2])
        XCTAssertEqual(server.requests(path).last?.query["game_type"], "2")
    }

    func testFailureThenRetry() async {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        server.stub("GET", path, .raw("service down", status: 503))
        let model = SeasonReportModel(session: session, seasonId: 2, gameType: 3)

        await model.loadIfNeeded()
        guard case .failed(let failure) = model.phase else {
            XCTFail("加载失败应进入失败态，实际为 \(model.phase)")
            return
        }
        XCTAssertEqual(failure.category, .server)

        server.stub("GET", path, .json(SeasonFixtures.report()))
        await model.retry()

        XCTAssertEqual(model.phase, .ready)
        XCTAssertEqual(model.winRate, 75)
    }

    func testGuestDoesNotRequest() async {
        let server = MockHTTPServer()
        let session = RankingFixtures.makeGuestSession(server)
        let model = SeasonReportModel(session: session, seasonId: 2, gameType: 3)

        await model.loadIfNeeded()

        XCTAssertEqual(model.phase, .idle)
        XCTAssertTrue(server.requests.isEmpty)
    }

    func testLogoutClearsReportAndLateResultIsDiscarded() async {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        let gate = DispatchSemaphore(value: 0)
        server.stub("GET", path) { _ in
            _ = gate.wait(timeout: .now() + 5)
            return .json(SeasonFixtures.report())
        }
        let model = SeasonReportModel(session: session, seasonId: 2, gameType: 3)

        let load = Task { await model.loadIfNeeded() }
        await waitUntil { server.requestCount(self.path) == 1 }
        session.logout()
        model.syncIdentity()
        gate.signal()
        await load.value

        XCTAssertEqual(model.phase, .idle)
        XCTAssertNil(model.report)
    }
}
