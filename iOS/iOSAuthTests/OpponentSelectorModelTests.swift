import XCTest
@testable import iOS

/// 选择球友：服务端已去重排序，页面只负责加载与展示。
@MainActor
final class OpponentSelectorModelTests: XCTestCase {
    private func candidatesJSON() -> [String: Any] { MatchStartFixtures.opponentCandidatesJSON() }

    func testLoadsServerDeduplicatedCandidatesWithLimit30() async throws {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        server.stub("GET", "/api/opponent/candidates", .json(candidatesJSON()))
        let model = OpponentSelectorModel(session: session, api: MatchStartAPI(session: session))

        await model.loadIfNeeded()

        XCTAssertEqual(model.phase, .loaded)
        XCTAssertEqual(model.opponents.map(\.userId), [20, 30, 40])
        XCTAssertEqual(model.opponents.map(\.sourceText), ["好友", "最近对手", "最近对手"])
        let request = try XCTUnwrap(server.requests("/api/opponent/candidates").first)
        XCTAssertEqual(request.query, ["limit": "30"])
    }

    func testEmptyListShowsTheEmptyState() async {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        server.stub("GET", "/api/opponent/candidates", .json(["success": true, "list": []]))
        let model = OpponentSelectorModel(session: session, api: MatchStartAPI(session: session))

        await model.loadIfNeeded()

        XCTAssertEqual(model.phase, .empty)
        XCTAssertTrue(model.opponents.isEmpty)
    }

    func testRetryLoadsAgainAfterFailure() async {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        let attempts = LockedCounter()
        server.stub("GET", "/api/opponent/candidates") { _ in
            attempts.next() == 1
                ? .failure(.notConnectedToInternet)
                : .json(self.candidatesJSON())
        }
        let model = OpponentSelectorModel(session: session, api: MatchStartAPI(session: session))

        await model.loadIfNeeded()
        guard case .failed = model.phase else {
            return XCTFail("首次应失败：\(model.phase)")
        }

        await model.retry()
        XCTAssertEqual(model.phase, .loaded)
        XCTAssertEqual(model.opponents.count, 3)
    }

    func testResultArrivingAfterAccountChangeIsDropped() async {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        let gate = DispatchSemaphore(value: 0)
        let json = candidatesJSON()
        server.stub("GET", "/api/opponent/candidates") { _ in
            gate.wait()
            return .json(json)
        }
        let model = OpponentSelectorModel(session: session, api: MatchStartAPI(session: session))

        let run = Task { await model.loadIfNeeded() }
        while server.requestCount("/api/opponent/candidates") == 0 { await Task.yield() }
        session.logout()
        gate.signal()
        await run.value

        XCTAssertTrue(model.opponents.isEmpty, "退出后的迟到响应不能写入列表")
    }
}
