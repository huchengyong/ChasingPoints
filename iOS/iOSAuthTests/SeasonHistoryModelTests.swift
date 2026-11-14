import XCTest
@testable import iOS

@MainActor
final class SeasonHistoryModelTests: XCTestCase {
    private let path = "/api/season/history"

    func testFirstLoadReadsFirstPageForTheGameType() async {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        server.stub("GET", path, .json(SeasonFixtures.history(
            total: 12,
            list: (1...10).map { SeasonFixtures.historyItem(season: SeasonFixtures.season(id: Int64($0), status: 2)) }
        )))
        let model = SeasonHistoryModel(session: session, gameType: 2)

        await model.loadIfNeeded()

        XCTAssertEqual(model.phase, .ready)
        XCTAssertEqual(model.items.count, 10)
        XCTAssertTrue(model.hasMore)

        let request = server.requests(path).first
        XCTAssertEqual(request?.query["game_type"], "2")
        XCTAssertEqual(request?.query["page"], "1")
        XCTAssertEqual(request?.query["page_size"], "10")
        XCTAssertEqual(request?.bearerToken, AuthFixtures.oldAccess)
    }

    func testSecondAppearanceDoesNotRequestAgain() async {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        server.stub("GET", path, .json(SeasonFixtures.history()))
        let model = SeasonHistoryModel(session: session, gameType: 3)

        await model.loadIfNeeded()
        await model.loadIfNeeded()

        XCTAssertEqual(server.requestCount(path), 1)
    }

    func testEmptyHistoryShowsEmptyPhase() async {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        server.stub("GET", path, .json(SeasonFixtures.history(total: 0, list: [])))
        let model = SeasonHistoryModel(session: session, gameType: 3)

        await model.loadIfNeeded()

        XCTAssertEqual(model.phase, .empty)
        XCTAssertFalse(model.hasMore)
        XCTAssertTrue(model.items.isEmpty)
    }

    func testSwitchingGameTypeResetsPaginationAndReadsNewType() async {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        server.stub("GET", path) { request in
            let gameType = Int(request.query["game_type"] ?? "3") ?? 3
            return .json(SeasonFixtures.history(
                total: 1,
                list: [SeasonFixtures.historyItem(season: SeasonFixtures.season(id: Int64(gameType), status: 2))]
            ))
        }
        let model = SeasonHistoryModel(session: session, gameType: 3)
        await model.loadIfNeeded()

        await model.selectGameType(2)

        XCTAssertEqual(model.currentGameType, 2)
        XCTAssertEqual(model.items.map(\.id), [2])
        XCTAssertFalse(model.isSwitchingGameType)
        XCTAssertEqual(server.requests(path).last?.query["game_type"], "2")
        XCTAssertEqual(server.requests(path).last?.query["page"], "1")
    }

    func testLoadMoreAppendsAndStopsAtTotal() async {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        server.stub("GET", path) { request in
            if request.query["page"] == "2" {
                return .json(SeasonFixtures.history(
                    total: 12,
                    list: [
                        SeasonFixtures.historyItem(season: SeasonFixtures.season(id: 11, status: 2)),
                        SeasonFixtures.historyItem(season: SeasonFixtures.season(id: 12, status: 2)),
                    ]
                ))
            }
            return .json(SeasonFixtures.history(
                total: 12,
                list: (1...10).map { SeasonFixtures.historyItem(season: SeasonFixtures.season(id: Int64($0), status: 2)) }
            ))
        }
        let model = SeasonHistoryModel(session: session, gameType: 3)
        await model.loadIfNeeded()

        await model.loadMore()

        XCTAssertEqual(model.items.count, 12)
        XCTAssertFalse(model.hasMore)
        XCTAssertFalse(model.isLoadingMore)
        XCTAssertEqual(server.requests(path).last?.query["page"], "2")
    }

    func testEmptyPageStopsPaginationWithoutLooping() async {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        server.stub("GET", path) { request in
            let page = Int(request.query["page"] ?? "1") ?? 1
            if page == 2 {
                return .json(SeasonFixtures.history(total: 25, list: []))
            }
            return .json(SeasonFixtures.history(
                total: 25,
                list: (1...10).map { SeasonFixtures.historyItem(season: SeasonFixtures.season(id: Int64($0), status: 2)) }
            ))
        }
        let model = SeasonHistoryModel(session: session, gameType: 3)
        await model.loadIfNeeded()

        await model.loadMore()

        XCTAssertEqual(model.items.count, 10)
        XCTAssertFalse(model.hasMore, "空页必须立即停止分页")
    }

    func testLoadMoreFailureKeepsItemsAndRetriesTheSamePage() async {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        server.stub("GET", path) { request in
            let page = Int(request.query["page"] ?? "1") ?? 1
            if page == 2 {
                return .raw("service down", status: 503)
            }
            return .json(SeasonFixtures.history(
                total: 15,
                list: (1...10).map { SeasonFixtures.historyItem(season: SeasonFixtures.season(id: Int64($0), status: 2)) }
            ))
        }
        let model = SeasonHistoryModel(session: session, gameType: 3)
        await model.loadIfNeeded()

        await model.loadMore()

        XCTAssertEqual(model.items.count, 10, "加载更多失败必须保留已读条目")
        XCTAssertNotNil(model.refreshFailure)
        XCTAssertFalse(model.isLoadingMore)

        server.stub("GET", path) { request in
            let page = Int(request.query["page"] ?? "1") ?? 1
            if page == 2 {
                return .json(SeasonFixtures.history(
                    total: 15,
                    list: (11...15).map { SeasonFixtures.historyItem(season: SeasonFixtures.season(id: Int64($0), status: 2)) }
                ))
            }
            return .json(SeasonFixtures.history(total: 15, list: []))
        }
        await model.loadMore()

        XCTAssertEqual(model.items.count, 15)
        XCTAssertEqual(server.requests(path).last?.query["page"], "2", "重试必须仍请求失败的那一页")
        XCTAssertNil(model.refreshFailure)
    }

    func testInitialFailureThenRetry() async {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        server.stub("GET", path, .raw("service down", status: 503))
        let model = SeasonHistoryModel(session: session, gameType: 3)

        await model.loadIfNeeded()
        guard case .failed(let failure) = model.phase else {
            XCTFail("首次加载失败应进入失败态，实际为 \(model.phase)")
            return
        }
        XCTAssertEqual(failure.category, .server)

        server.stub("GET", path, .json(SeasonFixtures.history()))
        await model.retry()

        XCTAssertEqual(model.phase, .ready)
        XCTAssertEqual(model.items.count, 1)
    }

    func testGuestDoesNotRequest() async {
        let server = MockHTTPServer()
        let session = RankingFixtures.makeGuestSession(server)
        let model = SeasonHistoryModel(session: session, gameType: 3)

        await model.loadIfNeeded()

        XCTAssertEqual(model.phase, .idle)
        XCTAssertTrue(server.requests.isEmpty)
    }

    func testLateResultAfterLogoutIsDiscarded() async {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        let gate = DispatchSemaphore(value: 0)
        server.stub("GET", path) { _ in
            _ = gate.wait(timeout: .now() + 5)
            return .json(SeasonFixtures.history())
        }
        let model = SeasonHistoryModel(session: session, gameType: 3)

        let load = Task { await model.loadIfNeeded() }
        await waitUntil { server.requestCount(self.path) == 1 }
        session.logout()
        model.syncIdentity()
        gate.signal()
        await load.value

        XCTAssertEqual(model.phase, .idle)
        XCTAssertTrue(model.items.isEmpty)
    }
}
