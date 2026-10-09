import Foundation
import XCTest
@testable import iOS

@MainActor
final class PagedHistoryListTests: XCTestCase {
    func testFirstLoadRequestsOnlyFirstPageWithPageSize() async throws {
        let server = MockHTTPServer()
        let session = await makeRestoredSession(server)
        server.stub("GET", "/api/match/list", .json(MatchHistoryFixtures.participantPage(
            items: [MatchHistoryFixtures.participantItem(id: 1)], total: 1
        )))
        let list = makeParticipantList(session)

        await list.loadIfNeeded()

        XCTAssertEqual(list.items.map(\.id), [1])
        XCTAssertEqual(list.phase, .loaded)
        let requests = server.requests("/api/match/list")
        XCTAssertEqual(requests.count, 1)
        XCTAssertEqual(requests.first?.query["page"], "1")
        XCTAssertEqual(requests.first?.query["page_size"], "20")
        XCTAssertEqual(requests.first?.bearerToken, AuthFixtures.oldAccess)
    }

    func testReturningToLoadedListDoesNotRequestAgain() async throws {
        let server = MockHTTPServer()
        let session = await makeRestoredSession(server)
        server.stub("GET", "/api/match/list", .json(MatchHistoryFixtures.participantPage(
            items: [MatchHistoryFixtures.participantItem(id: 1)], total: 1
        )))
        let list = makeParticipantList(session)

        await list.loadIfNeeded()
        await list.loadIfNeeded()
        await list.loadIfNeeded()

        XCTAssertEqual(server.requestCount("/api/match/list"), 1)
    }

    func testLoadMoreAppendsNextPageUntilTotalIsReached() async throws {
        let server = MockHTTPServer()
        let session = await makeRestoredSession(server)
        stubParticipantPages(server, total: 25)
        let list = makeParticipantList(session)

        await list.loadIfNeeded()
        XCTAssertEqual(list.items.count, 20)
        XCTAssertTrue(list.hasMore)

        await list.loadMore()
        XCTAssertEqual(list.items.count, 25)
        XCTAssertFalse(list.hasMore)
        XCTAssertEqual(server.requests("/api/match/list").last?.query["page"], "2")

        await list.loadMore()
        XCTAssertEqual(server.requestCount("/api/match/list"), 2, "没有更多时不应再请求")
    }

    func testEmptyListIsEmptyPhaseNotFailure() async throws {
        let server = MockHTTPServer()
        let session = await makeRestoredSession(server)
        server.stub("GET", "/api/match/list", .json(MatchHistoryFixtures.participantPage(items: [], total: 0)))
        let list = makeParticipantList(session)

        await list.loadIfNeeded()

        XCTAssertEqual(list.phase, .empty)
        XCTAssertTrue(list.items.isEmpty)
    }

    func testBusinessFailureIsFailureNotEmpty() async throws {
        let server = MockHTTPServer()
        let session = await makeRestoredSession(server)
        server.stub("GET", "/api/match/list", .json(MatchHistoryFixtures.businessFailure))
        let list = makeParticipantList(session)

        await list.loadIfNeeded()

        guard case .failed = list.phase else {
            return XCTFail("业务失败不应显示为空态：\(list.phase)")
        }
        XCTAssertTrue(list.items.isEmpty)
    }

    func testFirstPageFailureCanRetryIntoLoadedState() async throws {
        let server = MockHTTPServer()
        let session = await makeRestoredSession(server)
        server.stub("GET", "/api/match/list", .raw("service down", status: 503))
        let list = makeParticipantList(session)

        await list.loadIfNeeded()
        guard case .failed = list.phase else { return XCTFail("首页失败应进入错误态") }

        stubParticipantPages(server, total: 1)
        await list.retry()

        XCTAssertEqual(list.phase, .loaded)
        XCTAssertEqual(list.items.map(\.id), [1])
    }

    func testRefreshFailureKeepsExistingItemsWithBanner() async throws {
        let server = MockHTTPServer()
        let session = await makeRestoredSession(server)
        stubParticipantPages(server, total: 1)
        let list = makeParticipantList(session)
        await list.loadIfNeeded()

        server.stub("GET", "/api/match/list", .raw("service down", status: 503))
        await list.refresh()

        XCTAssertEqual(list.items.map(\.id), [1])
        XCTAssertEqual(list.phase, .loaded)
        XCTAssertNotNil(list.refreshError)
    }

    func testLoadMoreFailureKeepsPageAndAllowsRetry() async throws {
        let server = MockHTTPServer()
        let session = await makeRestoredSession(server)
        stubParticipantPages(server, total: 25)
        let list = makeParticipantList(session)
        await list.loadIfNeeded()

        server.stub("GET", "/api/match/list", .json(MatchHistoryFixtures.businessFailure))
        await list.loadMore()

        XCTAssertEqual(list.loadedPage, 1, "失败不得推进页码")
        XCTAssertEqual(list.items.count, 20)
        XCTAssertNotNil(list.loadMoreError)

        stubParticipantPages(server, total: 25)
        await list.loadMore()

        XCTAssertEqual(list.items.count, 25)
        XCTAssertNil(list.loadMoreError)
        XCTAssertEqual(server.requests("/api/match/list").last?.query["page"], "2")
    }

    func testLoadMoreDoesNothingWhileRefreshing() async throws {
        let server = MockHTTPServer()
        let session = await makeRestoredSession(server)
        stubParticipantPages(server, total: 25)
        let list = makeParticipantList(session)
        await list.loadIfNeeded()

        let gate = DispatchSemaphore(value: 0)
        server.stub("GET", "/api/match/list") { _ in
            _ = gate.wait(timeout: .now() + 5)
            return participantPageResponse(page: 1, total: 25)
        }
        let refresh = Task { await list.refresh() }
        await waitUntil { server.requestCount("/api/match/list") == 2 }

        await list.loadMore()
        XCTAssertEqual(server.requestCount("/api/match/list"), 2, "刷新中不得发起加载更多")

        gate.signal()
        await refresh.value
    }

    func testCancelledFirstLoadReturnsToIdleAndLoadsLater() async throws {
        let server = MockHTTPServer()
        let session = await makeRestoredSession(server)
        server.stub("GET", "/api/match/list", .failure(.cancelled))
        let list = makeParticipantList(session)

        await list.loadIfNeeded()
        XCTAssertEqual(list.phase, .idle, "页面离开导致的取消不是失败，不应显示错误")

        stubParticipantPages(server, total: 1)
        await list.loadIfNeeded()
        XCTAssertEqual(list.items.map(\.id), [1])
        XCTAssertEqual(list.phase, .loaded)
    }

    func testLateResultAfterLogoutIsDiscarded() async throws {
        let server = MockHTTPServer()
        let session = await makeRestoredSession(server)
        let gate = DispatchSemaphore(value: 0)
        server.stub("GET", "/api/match/list") { _ in
            _ = gate.wait(timeout: .now() + 5)
            return .json(MatchHistoryFixtures.participantPage(items: [MatchHistoryFixtures.participantItem(id: 1)], total: 1))
        }
        let list = makeParticipantList(session)

        let load = Task { await list.loadIfNeeded() }
        await waitUntil { server.requestCount("/api/match/list") == 1 }
        session.logout()
        gate.signal()
        await load.value

        XCTAssertTrue(list.items.isEmpty, "退出后旧账号的结果不得写入列表")
        XCTAssertNotEqual(list.phase, .loaded)

        list.syncIdentity()
        XCTAssertEqual(list.phase, .idle)
        await list.loadIfNeeded()
        XCTAssertEqual(server.requestCount("/api/match/list"), 1, "退出后不得使用旧账号继续请求")
    }

    func testSwitchingAccountClearsListAndLoadsOnlyNewUser() async throws {
        let server = MockHTTPServer()
        let session = await makeRestoredSession(server)
        server.stub("GET", "/api/match/list") { request in
            let id: Int64 = request.bearerToken == AuthFixtures.oldAccess ? 100 : 200
            return .json(MatchHistoryFixtures.participantPage(items: [MatchHistoryFixtures.participantItem(id: id)], total: 1))
        }
        let list = makeParticipantList(session)
        await list.loadIfNeeded()
        XCTAssertEqual(list.items.map(\.id), [100])

        server.stub("POST", "/api/auth/login", .json(AuthFixtures.loginResponse(
            access: "b-access-token",
            refresh: "b-refresh-token",
            user: AuthFixtures.userInfo(id: 2, nickname: "用户B")
        )))
        try await session.login(phone: AuthFixtures.phone, code: "123456")

        list.syncIdentity()
        XCTAssertTrue(list.items.isEmpty)
        XCTAssertEqual(list.phase, .idle)

        await list.loadIfNeeded()
        XCTAssertEqual(list.items.map(\.id), [200])
        XCTAssertEqual(server.requests("/api/match/list").last?.bearerToken, "b-access-token")
    }

    private func stubParticipantPages(_ server: MockHTTPServer, total: Int) {
        server.stub("GET", "/api/match/list") { request in
            participantPageResponse(page: Int(request.query["page"] ?? "") ?? 1, total: total)
        }
    }

    private func makeParticipantList(_ session: SessionStore) -> PagedHistoryList<MatchListItem> {
        let api = MatchHistoryAPI(session: session)
        return PagedHistoryList(session: session) { page in
            try await api.participantPage(page: page)
        }
    }
}

/// 按服务端规则返回某一页：每页 `MatchHistoryAPI.pageSize` 条，最后一页按 `total` 截断。
private func participantPageResponse(page: Int, total: Int) -> MockHTTPServer.Stub {
    let pageSize = MatchHistoryAPI.pageSize
    let first = (page - 1) * pageSize + 1
    let last = min(page * pageSize, total)
    let ids: [Int64] = first <= last ? (first...last).map { Int64($0) } : []
    return .json(MatchHistoryFixtures.participantPage(
        items: ids.map { MatchHistoryFixtures.participantItem(id: $0) },
        total: total
    ))
}

@MainActor
private func makeRestoredSession(_ server: MockHTTPServer) async -> SessionStore {
    server.stub("GET", "/api/user/info", .json(AuthFixtures.userInfoResponse()))
    let store = InMemoryCredentialStore(credentials: AuthFixtures.credentials())
    let session = makeSessionStore(server, credentialStore: store)
    await session.restore()
    return session
}
