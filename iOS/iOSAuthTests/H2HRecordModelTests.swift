import XCTest
@testable import iOS

@MainActor
final class H2HRecordModelTests: XCTestCase {
    func testOverviewLoadsStatsAndMonthHistory() async {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        server.stub("GET", "/api/match/h2h/overview", .json(SubPageFixtures.h2hOverview(
            items: [
                MatchHistoryFixtures.participantItem(id: 1, result: 1, myScore: 7, opponentScore: 5),
                MatchHistoryFixtures.participantItem(id: 2, result: 2, myScore: 3, opponentScore: 6),
            ],
            total: 25,
            hasMore: true
        )))
        let model = makeModel(session, opponentId: 18, opponentName: "阿杰")

        await model.reload()

        XCTAssertEqual(model.pageStatus, .ready)
        XCTAssertTrue(model.showSummaryCard)
        XCTAssertEqual(model.stats?.totalMatches, 8)
        XCTAssertEqual(model.history.map(\.id), [1, 2])
        XCTAssertTrue(model.hasMore)
        XCTAssertEqual(model.hero.title, "你对阿杰略占上风")
        XCTAssertEqual(model.viewModel.navigationTitle, "交锋记录 - 阿杰")
        XCTAssertEqual(server.requests("/api/match/h2h/overview").first?.query, [
            "opponent_id": "18",
            "page": "1",
            "page_size": "20",
            "start_date": "2026-10-01",
            "end_date": "2026-10-31",
        ])
    }

    func testHiddenOverviewShowsHiddenStateWithoutFakeStats() async {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        server.stub("GET", "/api/match/h2h/overview", .json(SubPageFixtures.h2hOverview(hidden: true)))
        let model = makeModel(session, opponentId: 18, opponentName: "阿杰")

        await model.reload()

        XCTAssertTrue(model.isHidden)
        XCTAssertNil(model.stats)
        XCTAssertTrue(model.history.isEmpty)
        XCTAssertFalse(model.hasMore)
        XCTAssertFalse(model.showSummaryCard)
        XCTAssertNil(model.failure)
    }

    func testPartialAvailabilityKeepsHistoryAndHidesSummary() async {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        server.stub("GET", "/api/match/h2h/overview", .json(SubPageFixtures.h2hOverview(
            items: [MatchHistoryFixtures.participantItem(id: 1)],
            availability: ["opponent": true, "stats": false, "history": true]
        )))
        let model = makeModel(session, opponentId: 18, opponentName: "阿杰")

        await model.reload()

        XCTAssertEqual(model.pageStatus, .partial)
        XCTAssertFalse(model.showSummaryCard)
        XCTAssertEqual(model.partialMessage, "部分交锋数据加载失败")
        XCTAssertEqual(model.history.map(\.id), [1])
    }

    func testLoadMoreAppendsNextPageUntilTotal() async {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        server.stub("GET", "/api/match/h2h/overview", .json(SubPageFixtures.h2hOverview(
            items: (1...20).map { MatchHistoryFixtures.participantItem(id: Int64($0)) },
            total: 25,
            hasMore: true
        )))
        server.stub("GET", "/api/match/h2h/history", .json(SubPageFixtures.h2hHistory(
            items: (21...25).map { MatchHistoryFixtures.participantItem(id: Int64($0)) },
            total: 25
        )))
        let model = makeModel(session, opponentId: 18, opponentName: "阿杰")
        await model.reload()

        await model.loadMore()

        XCTAssertEqual(model.history.count, 25)
        XCTAssertFalse(model.hasMore)
        XCTAssertEqual(server.requests("/api/match/h2h/history").first?.query["page"], "2")
        XCTAssertEqual(server.requests("/api/match/h2h/history").first?.query["opponent_id"], "18")

        await model.loadMore()
        XCTAssertEqual(server.requestCount("/api/match/h2h/history"), 1, "没有更多时不应再请求")
    }

    func testShiftMonthRequestsNewRange() async {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        server.stub("GET", "/api/match/h2h/overview") { request in
            request.query["start_date"] == "2026-11-01"
                ? .json(SubPageFixtures.h2hOverview(
                    items: [MatchHistoryFixtures.participantItem(id: 11)],
                    total: 1
                ))
                : .json(SubPageFixtures.h2hOverview(
                    items: [MatchHistoryFixtures.participantItem(id: 1)],
                    total: 1
                ))
        }
        let model = makeModel(session, opponentId: 18, opponentName: "阿杰")
        await model.reload()
        XCTAssertEqual(model.history.map(\.id), [1])

        await model.shiftMonth(by: 1)

        XCTAssertEqual(model.selectedMonthKey, "2026-11")
        XCTAssertEqual(model.calendar.title, "2026年11月")
        XCTAssertEqual(model.history.map(\.id), [11])
        XCTAssertEqual(server.requests("/api/match/h2h/overview").last?.query["start_date"], "2026-11-01")
        XCTAssertEqual(server.requests("/api/match/h2h/overview").last?.query["end_date"], "2026-11-30")
    }

    func testLatestOverviewResponseWinsAcrossMonthShift() async {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        let firstGate = DispatchSemaphore(value: 0)
        let secondGate = DispatchSemaphore(value: 0)
        let calls = LockedCounter()
        server.stub("GET", "/api/match/h2h/overview") { _ in
            let call = calls.next()
            let gate = call == 1 ? firstGate : secondGate
            _ = gate.wait(timeout: .now() + 5)
            return .json(SubPageFixtures.h2hOverview(
                items: [MatchHistoryFixtures.participantItem(id: Int64(call))],
                total: 1
            ))
        }
        let model = makeModel(session, opponentId: 18, opponentName: "阿杰")

        let first = Task { await model.reload() }
        await waitUntil { server.requestCount("/api/match/h2h/overview") == 1 }
        let second = Task { await model.shiftMonth(by: 1) }
        await waitUntil { server.requestCount("/api/match/h2h/overview") == 2 }

        secondGate.signal()
        await second.value
        firstGate.signal()
        await first.value

        XCTAssertEqual(server.requestCount("/api/match/h2h/overview"), 2)
        XCTAssertEqual(model.history.map(\.id), [2], "旧月份迟到的结果不得覆盖新月份")
        XCTAssertEqual(model.selectedMonthKey, "2026-11")
    }

    func testEmptyMonthShowsEmptyStatus() async {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        server.stub("GET", "/api/match/h2h/overview", .json(SubPageFixtures.h2hOverview(
            stats: SubPageFixtures.h2hStats(totalMatches: 0, myWins: 0, opponentWins: 0, avgScoreDiff: 0, maxWinStreak: 0),
            items: [],
            total: 0
        )))
        let model = makeModel(session, opponentId: 18, opponentName: "阿杰")

        await model.reload()

        XCTAssertEqual(model.pageStatus, .empty)
        XCTAssertTrue(model.showSummaryCard)
        XCTAssertTrue(model.matchCards.isEmpty)
    }

    func testFailureCanBeRetriedIntoReadyState() async {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        let attempts = LockedCounter()
        server.stub("GET", "/api/match/h2h/overview") { _ in
            attempts.next() == 1
                ? .json(["success": false, "message": "竞技读模型不可用"])
                : .json(SubPageFixtures.h2hOverview(
                    items: [MatchHistoryFixtures.participantItem(id: 1)],
                    total: 1
                ))
        }
        let model = makeModel(session, opponentId: 18, opponentName: "阿杰")

        await model.reload()
        XCTAssertEqual(model.pageStatus, .error)
        XCTAssertEqual(model.failureFeedback?.title, "交锋数据加载失败")
        XCTAssertEqual(model.failureFeedback?.description, "竞技读模型不可用")

        await model.retry()
        XCTAssertEqual(model.pageStatus, .ready)
        XCTAssertEqual(model.history.map(\.id), [1])
    }

    func testTargetAccessFailureExposesToastAndDismissAction() async {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        server.stub("GET", "/api/match/h2h/overview", .json([
            "success": false,
            "message": "仅可查看好友的对方战绩",
        ]))
        let model = H2HRecordModel(
            session: session,
            targetUserId: 52,
            targetName: "球友 B",
            opponentId: 18,
            opponentName: "阿杰",
            monthKey: "2026-10"
        )

        await model.reload()

        XCTAssertEqual(model.loadFailureAction?.toastMessage, "仅可查看好友的对方战绩")
        XCTAssertEqual(model.failureFeedback?.description, "仅可查看好友的对方战绩")
    }

    func testIdentityChangeDropsLateOverview() async {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        let gate = DispatchSemaphore(value: 0)
        server.stub("GET", "/api/match/h2h/overview") { _ in
            _ = gate.wait(timeout: .now() + 5)
            return .json(SubPageFixtures.h2hOverview(
                items: [MatchHistoryFixtures.participantItem(id: 1)],
                total: 1
            ))
        }
        let model = makeModel(session, opponentId: 18, opponentName: "阿杰")

        let load = Task { await model.reload() }
        await waitUntil { server.requestCount("/api/match/h2h/overview") == 1 }
        session.logout()
        await model.onIdentityChanged()
        gate.signal()
        await load.value

        XCTAssertNil(model.stats, "退出后迟到的统计不得写入")
        XCTAssertTrue(model.history.isEmpty, "退出后迟到的历史不得写入")
        XCTAssertNotEqual(model.pageStatus, .ready)
    }

    private func makeModel(
        _ session: SessionStore,
        opponentId: Int64,
        opponentName: String
    ) -> H2HRecordModel {
        H2HRecordModel(
            session: session,
            opponentId: opponentId,
            opponentName: opponentName,
            monthKey: "2026-10"
        )
    }
}
