import Foundation
import XCTest
@testable import iOS

/// 赛讯 Tab 页面模型：首次加载、筛选重读、分页、失败保留数据与 60 秒新鲜度。
@MainActor
final class SaiXunListModelTests: XCTestCase {
    private let listPath = "/api/event-news/list"

    func testFirstLoadRequestsCurrentYearAndBuildsSections() async {
        let server = MockHTTPServer()
        server.stub("GET", listPath, .json(EventNewsFixtures.listResponse(
            total: 3,
            list: [
                EventNewsFixtures.eventNewsJSON(id: 1, tournamentName: "进行中的赛事", status: 1, startDate: "2026-04-01"),
                EventNewsFixtures.eventNewsJSON(id: 2, tournamentName: "即将开始的赛事", status: 0, startDate: "2026-04-20"),
                EventNewsFixtures.eventNewsJSON(id: 3, tournamentName: "已结束的赛事", status: 2, startDate: "2026-02-10"),
            ]
        )))
        let model = makeModel(server)

        await model.loadIfNeeded()

        XCTAssertEqual(model.phase, .loaded)
        XCTAssertEqual(model.items.map(\.id), [1, 2, 3])
        XCTAssertEqual(model.total, 3)
        XCTAssertFalse(model.hasMore)
        XCTAssertEqual(model.sections.map(\.title), ["进行中", "即将开始", "2月"])
        XCTAssertEqual(server.requests(listPath).first?.query, [
            "page": "1",
            "page_size": "10",
            "from": "2026-01-01",
            "to": "2026-12-31",
        ])
    }

    func testYearSwitchReloadsFirstPageAndDiscardsStaleResponse() async {
        let server = MockHTTPServer()
        let gate = DispatchSemaphore(value: 0)
        server.stub("GET", listPath) { request in
            if request.query["from"] == "2026-01-01" {
                _ = gate.wait(timeout: .now() + 5)
                return .json(EventNewsFixtures.listResponse(
                    total: 8,
                    list: [EventNewsFixtures.eventNewsJSON(id: 1, status: 2, startDate: "2026-02-01")]
                ))
            }
            return .json(EventNewsFixtures.listResponse(
                total: 1,
                list: [EventNewsFixtures.eventNewsJSON(id: 9, status: 2, startDate: "2025-03-01")]
            ))
        }
        let model = makeModel(server)

        let first = Task { await model.loadIfNeeded() }
        await waitUntil { server.requestCount(self.listPath) == 1 }
        await model.selectYear(2025)
        XCTAssertEqual(model.items.map(\.id), [9])
        XCTAssertEqual(model.total, 1)

        gate.signal()
        await first.value

        XCTAssertEqual(model.items.map(\.id), [9], "旧年份的迟到结果必须丢弃")
        XCTAssertEqual(model.total, 1)
        XCTAssertEqual(server.requests(listPath).last?.query, [
            "page": "1",
            "page_size": "10",
            "from": "2025-01-01",
            "to": "2025-12-31",
        ])
    }

    func testLoadMoreAppendsNextPage() async {
        let server = MockHTTPServer()
        server.stub("GET", listPath) { request in
            let page = request.query["page"] ?? "1"
            let ids = page == "1" ? Array(1...10) : [11, 12]
            return .json(EventNewsFixtures.listResponse(
                total: 12,
                list: ids.map {
                    EventNewsFixtures.eventNewsJSON(id: Int64($0), status: 2, startDate: "2026-02-01")
                }
            ))
        }
        let model = makeModel(server)

        await model.loadIfNeeded()
        XCTAssertEqual(model.items.count, 10)
        XCTAssertTrue(model.hasMore)

        await model.loadMore()

        XCTAssertEqual(model.items.map(\.id), Array(1...10).map(Int64.init) + [11, 12])
        XCTAssertFalse(model.hasMore)
        XCTAssertEqual(server.requests(listPath).map { $0.query["page"] }, ["1", "2"])
    }

    func testLoadMoreFailureKeepsPageAndItems() async {
        let server = MockHTTPServer()
        server.stub("GET", listPath) { request in
            if request.query["page"] == "2" {
                return .raw("boom", status: 503)
            }
            return .json(EventNewsFixtures.listResponse(
                total: 12,
                list: (1...10).map {
                    EventNewsFixtures.eventNewsJSON(id: Int64($0), status: 2, startDate: "2026-02-01")
                }
            ))
        }
        let model = makeModel(server)

        await model.loadIfNeeded()
        await model.loadMore()

        XCTAssertEqual(model.items.count, 10)
        XCTAssertTrue(model.hasMore)
        XCTAssertEqual(model.phase, .loaded)
    }

    func testRefreshFailureKeepsExistingItems() async {
        let server = MockHTTPServer()
        server.stub("GET", listPath, .json(EventNewsFixtures.listResponse(
            total: 1,
            list: [EventNewsFixtures.eventNewsJSON(id: 5, status: 1, startDate: "2026-04-01")]
        )))
        let model = makeModel(server)
        await model.loadIfNeeded()

        server.stub("GET", listPath, .raw("boom", status: 503))
        await model.refresh()

        XCTAssertEqual(model.items.map(\.id), [5])
        XCTAssertEqual(model.phase, .loaded)
        XCTAssertNotNil(model.refreshError)
    }

    func testEmptyResponseShowsStatusSpecificEmptyCopy() async {
        let server = MockHTTPServer()
        server.stub("GET", listPath, .json(EventNewsFixtures.listResponse(total: 0, list: [])))
        let model = makeModel(server)

        await model.loadIfNeeded()

        XCTAssertEqual(model.phase, .empty)
        XCTAssertEqual(model.emptyTitle, "这一年还没有赛讯")
        XCTAssertEqual(model.emptyDescription, "换个年份看看，或稍后下拉刷新。")

        await model.selectStatus(1)

        XCTAssertEqual(model.emptyTitle, "当前没有进行中的赛事")
        XCTAssertEqual(model.emptyDescription, "可以先看看即将开始的赛事。")
        XCTAssertTrue(model.hasNarrowFilter)
    }

    func testInitialFailureShowsFeedbackAndRetryReloads() async {
        let server = MockHTTPServer()
        server.stub("GET", listPath, .raw("down", status: 503))
        let model = makeModel(server)

        await model.loadIfNeeded()

        guard case .failed(let failure) = model.phase else {
            return XCTFail("首次加载失败应进入失败态")
        }
        XCTAssertEqual(failure.category, .server)
        XCTAssertEqual(failure.feedback(resource: "赛讯").title, "服务暂不可用")

        server.stub("GET", listPath, .json(EventNewsFixtures.listResponse(total: 1)))
        await model.retry()

        XCTAssertEqual(model.phase, .loaded)
        XCTAssertEqual(model.items.count, 1)
    }

    func testAppearReloadsOnlyAfterSixtySecondFreshness() async {
        let server = MockHTTPServer()
        server.stub("GET", listPath, .json(EventNewsFixtures.listResponse(total: 1)))
        var now = EventNewsFixtures.time("2026-04-01T12:00:00+08:00")
        let model = SaiXunListModel(session: makeGuestSession(server), now: { now })

        await model.loadIfNeeded()
        XCTAssertEqual(server.requestCount(listPath), 1)

        now = now.addingTimeInterval(30)
        await model.loadIfNeeded()
        XCTAssertEqual(server.requestCount(listPath), 1, "未超过 60 秒时不得重复读取")

        now = now.addingTimeInterval(31)
        await model.loadIfNeeded()
        XCTAssertEqual(server.requestCount(listPath), 2)
        XCTAssertEqual(model.phase, .loaded)
    }

    func testFilterChangeResetsPaginationToFirstPage() async {
        let server = MockHTTPServer()
        server.stub("GET", listPath, .json(EventNewsFixtures.listResponse(
            total: 12,
            list: [EventNewsFixtures.eventNewsJSON(id: 1, status: 1, startDate: "2026-04-01")]
        )))
        let model = makeModel(server)
        await model.loadIfNeeded()
        await model.loadMore()

        await model.selectStatus(1)

        XCTAssertEqual(model.items.map(\.id), [1])
        XCTAssertEqual(model.total, 12)
        XCTAssertEqual(server.requests(listPath).last?.query, [
            "page": "1",
            "page_size": "10",
            "from": "2026-01-01",
            "to": "2026-12-31",
            "status": "1",
        ])
    }

    func testLogoutClearsListState() async {
        let server = MockHTTPServer()
        let session = await makeSignedInSession(server)
        server.stub("GET", listPath, .json(EventNewsFixtures.listResponse(total: 1)))
        let model = SaiXunListModel(session: session)

        await model.loadIfNeeded()
        XCTAssertEqual(model.items.count, 1)

        session.logout()
        model.syncIdentity()

        XCTAssertTrue(model.items.isEmpty)
        XCTAssertEqual(model.total, 0)
        XCTAssertEqual(model.phase, .idle)
    }

    private func makeModel(_ server: MockHTTPServer) -> SaiXunListModel {
        SaiXunListModel(
            session: makeGuestSession(server),
            now: { EventNewsFixtures.time("2026-04-01T12:00:00+08:00") }
        )
    }

    private func makeGuestSession(_ server: MockHTTPServer) -> SessionStore {
        makeSessionStore(server, credentialStore: InMemoryCredentialStore(credentials: nil))
    }

    private func makeSignedInSession(_ server: MockHTTPServer) async -> SessionStore {
        server.stub("GET", "/api/user/info", .json(AuthFixtures.userInfoResponse()))
        let session = makeSessionStore(
            server,
            credentialStore: InMemoryCredentialStore(credentials: AuthFixtures.credentials())
        )
        await session.restore()
        return session
    }
}
