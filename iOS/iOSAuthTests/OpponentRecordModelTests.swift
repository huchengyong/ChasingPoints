import XCTest
@testable import iOS

@MainActor
final class OpponentRecordModelTests: XCTestCase {
    func testLoadsFirstPageAndStats() async {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        server.stub("GET", "/api/opponent/list", .json(SubPageFixtures.opponentList(
            items: [SubPageFixtures.opponentItem(id: 18)],
            totalOpponents: 4,
            totalWins: 14
        )))
        let model = OpponentRecordModel(session: session)

        await model.onAppear()

        XCTAssertEqual(model.list.items.map(\.id), [18])
        XCTAssertEqual(model.totalOpponents, 4)
        XCTAssertEqual(model.totalWins, 14)
        XCTAssertEqual(model.cards.first?.winRateText, "58%")
        XCTAssertEqual(server.requests("/api/opponent/list").first?.query, [
            "page": "1",
            "page_size": "20",
            "keyword": "",
        ])
    }

    func testSearchKeepsKeywordAndUpdatesStats() async {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        server.stub("GET", "/api/opponent/list") { request in
            request.query["keyword"] == "阿杰"
                ? .json(SubPageFixtures.opponentList(
                    items: [SubPageFixtures.opponentItem(id: 18)],
                    totalOpponents: 1,
                    totalWins: 7
                ))
                : .json(SubPageFixtures.opponentList(items: [], totalOpponents: 4, totalWins: 14))
        }
        let model = OpponentRecordModel(session: session)
        await model.onAppear()

        model.keyword = "阿杰"
        await model.search()

        XCTAssertEqual(model.keyword, "阿杰")
        XCTAssertEqual(model.list.items.map(\.id), [18])
        XCTAssertEqual(model.totalOpponents, 1)
        XCTAssertEqual(model.totalWins, 7)
        XCTAssertEqual(server.requests("/api/opponent/list").last?.query["keyword"], "阿杰")
    }

    func testLoadMoreAppendsNextPage() async {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        server.stub("GET", "/api/opponent/list") { request in
            let page = Int(request.query["page"] ?? "") ?? 1
            let ids = page == 1 ? Array(1...20) : Array(21...25)
            return .json(SubPageFixtures.opponentList(
                items: ids.map { SubPageFixtures.opponentItem(id: Int64($0)) },
                total: 25
            ))
        }
        let model = OpponentRecordModel(session: session)

        await model.onAppear()
        XCTAssertEqual(model.list.items.count, 20)
        XCTAssertTrue(model.list.hasMore)

        await model.list.loadMore()
        XCTAssertEqual(model.list.items.count, 25)
        XCTAssertFalse(model.list.hasMore)
        XCTAssertEqual(server.requests("/api/opponent/list").map { $0.query["page"] }, ["1", "2"])
    }

    func testEmptyListShowsEmptyPhase() async {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        server.stub("GET", "/api/opponent/list", .json(SubPageFixtures.opponentList(
            items: [],
            totalOpponents: 0,
            totalWins: 0
        )))
        let model = OpponentRecordModel(session: session)

        await model.onAppear()

        XCTAssertEqual(model.list.phase, .empty)
        XCTAssertEqual(model.statsSummary.subtitle, "先找到最值得点进复盘的那个对手")
    }

    func testFailureCanBeRetried() async {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        let attempts = LockedCounter()
        server.stub("GET", "/api/opponent/list") { _ in
            attempts.next() == 1
                ? .raw("down", status: 503)
                : .json(SubPageFixtures.opponentList(items: [SubPageFixtures.opponentItem(id: 3)]))
        }
        let model = OpponentRecordModel(session: session)

        await model.onAppear()
        guard case .failed = model.list.phase else {
            return XCTFail("首次应失败：\(model.list.phase)")
        }

        await model.retry()
        XCTAssertEqual(model.list.items.map(\.id), [3])
        XCTAssertEqual(model.list.phase, .loaded)
    }

    func testIdentityChangeDropsLateListAndResetsStats() async {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        let gate = DispatchSemaphore(value: 0)
        server.stub("GET", "/api/opponent/list") { _ in
            _ = gate.wait(timeout: .now() + 5)
            return .json(SubPageFixtures.opponentList(
                items: [SubPageFixtures.opponentItem(id: 7)],
                totalOpponents: 1,
                totalWins: 7
            ))
        }
        let model = OpponentRecordModel(session: session)

        let load = Task { await model.onAppear() }
        await waitUntil { server.requestCount("/api/opponent/list") == 1 }
        session.logout()
        await model.onIdentityChanged()
        gate.signal()
        await load.value

        XCTAssertTrue(model.list.items.isEmpty, "退出后迟到的列表不得写入")
        XCTAssertEqual(model.totalOpponents, 0)
        XCTAssertEqual(model.totalWins, 0)
        XCTAssertNotEqual(model.list.phase, .loaded)
    }
}
