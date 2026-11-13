import XCTest
@testable import iOS

@MainActor
final class ReputationModelTests: XCTestCase {
    func testLoadsStatusAndLogsWithPageParams() async {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        server.stub("GET", "/api/user/overview", .json(ProfileFixtures.overview(
            reputation: ProfileFixtures.reputation(score: 186, status: "good", statusText: "良好")
        )))
        server.stub("GET", "/api/user/reputation/logs", .json(SubPageFixtures.reputationLogs(
            items: [SubPageFixtures.reputationLog(id: 9001, changeScore: -3)],
            total: 1
        )))
        let model = ReputationModel(session: session)

        await model.onAppear()

        XCTAssertEqual(model.displayScore, "186")
        XCTAssertEqual(model.displayStatusText, "良好")
        XCTAssertFalse(model.isRestricted)
        XCTAssertEqual(model.logs.items.map(\.id), [9001])
        XCTAssertEqual(model.logs.items.first?.changeScoreText, "-3")
        XCTAssertEqual(model.logs.total, 1)
        XCTAssertEqual(server.requests("/api/user/reputation/logs").first?.query, [
            "page": "1",
            "page_size": "20",
        ])
    }

    func testLogsPaginateUntilTotalIsReached() async {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        server.stub("GET", "/api/user/overview", .json(ProfileFixtures.overview(
            reputation: ProfileFixtures.reputation()
        )))
        server.stub("GET", "/api/user/reputation/logs") { request in
            let page = Int(request.query["page"] ?? "") ?? 1
            let ids = page == 1 ? Array(1...20) : Array(21...25)
            return .json(SubPageFixtures.reputationLogs(
                items: ids.map { SubPageFixtures.reputationLog(id: Int64($0)) },
                total: 25
            ))
        }
        let model = ReputationModel(session: session)

        await model.onAppear()
        XCTAssertEqual(model.logs.items.count, 20)
        XCTAssertTrue(model.logs.hasMore)

        await model.logs.loadMore()
        XCTAssertEqual(model.logs.items.count, 25)
        XCTAssertFalse(model.logs.hasMore)
        XCTAssertEqual(server.requests("/api/user/reputation/logs").map { $0.query["page"] }, ["1", "2"])
    }

    func testMissingReputationShowsStatusFailureInsteadOfZero() async {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        server.stub("GET", "/api/user/overview", .json(ProfileFixtures.overview()))
        server.stub("GET", "/api/user/reputation/logs", .json(SubPageFixtures.reputationLogs(items: [], total: 0)))
        let model = ReputationModel(session: session)

        await model.onAppear()

        XCTAssertNil(model.status.value)
        XCTAssertEqual(model.displayScore, "--")
        XCTAssertEqual(model.statusFeedback?.title, "信誉状态加载失败")
        XCTAssertEqual(model.statusFeedback?.description, "信誉状态暂时不可用")
        XCTAssertEqual(model.statusFeedback?.retryable, true)
        XCTAssertEqual(model.logs.phase, .empty)
    }

    func testLogsFailureCanBeRetried() async {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        server.stub("GET", "/api/user/overview", .json(ProfileFixtures.overview(
            reputation: ProfileFixtures.reputation()
        )))
        let attempts = LockedCounter()
        server.stub("GET", "/api/user/reputation/logs") { _ in
            attempts.next() == 1
                ? .raw("down", status: 503)
                : .json(SubPageFixtures.reputationLogs(
                    items: [SubPageFixtures.reputationLog(id: 42)],
                    total: 1
                ))
        }
        let model = ReputationModel(session: session)

        await model.onAppear()
        guard case .failed(let message) = model.logs.phase else {
            return XCTFail("首次应失败：\(model.logs.phase)")
        }
        XCTAssertEqual(model.logsFeedback(for: message).title, "信誉记录加载失败")

        await model.retryLogs()
        XCTAssertEqual(model.logs.items.map(\.id), [42])
        XCTAssertEqual(model.logs.phase, .loaded)
    }

    func testIdentityChangeDropsLateLogs() async {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        server.stub("GET", "/api/user/overview", .json(ProfileFixtures.overview(
            reputation: ProfileFixtures.reputation()
        )))
        let gate = DispatchSemaphore(value: 0)
        server.stub("GET", "/api/user/reputation/logs") { _ in
            _ = gate.wait(timeout: .now() + 5)
            return .json(SubPageFixtures.reputationLogs(
                items: [SubPageFixtures.reputationLog(id: 777)],
                total: 1
            ))
        }
        let model = ReputationModel(session: session)

        let load = Task { await model.onAppear() }
        await waitUntil { server.requestCount("/api/user/reputation/logs") == 1 }
        session.logout()
        await model.onIdentityChanged()
        gate.signal()
        await load.value

        XCTAssertNil(model.status.value, "退出后迟到的状态不得写入")
        XCTAssertTrue(model.logs.items.isEmpty, "退出后迟到的流水不得写入")
        XCTAssertNotEqual(model.logs.phase, .loaded)
    }
}
