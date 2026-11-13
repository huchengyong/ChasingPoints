import Foundation
import XCTest
@testable import iOS

@MainActor
final class SpectatorLobbyModelTests: XCTestCase {
    func testGuestFirstLoadRequestsHallLivePageWithoutAuthorization() async {
        let server = MockHTTPServer()
        let session = SpectatorLobbyFixtures.makeGuestSession(server)
        server.stub("GET", "/api/public/matches", SpectatorLobbyFixtures.listResponse([
            SpectatorLobbyFixtures.itemJSON(id: 1, player1Name: "甲1", player2Name: "乙1"),
            SpectatorLobbyFixtures.itemJSON(id: 2, player1Name: "甲2", player2Name: "乙2"),
        ]))
        let model = SpectatorLobbyModel(session: session)

        await model.loadIfNeeded()

        XCTAssertEqual(model.tab, .live)
        XCTAssertEqual(model.cards.map(\.id), [1, 2])
        XCTAssertEqual(model.phase, .loaded)
        XCTAssertEqual(model.total, 2)
        XCTAssertFalse(model.hasMore)
        let request = server.requests("/api/public/matches").first
        XCTAssertEqual(request?.query, [
            "scope": "hall",
            "status": "1",
            "page": "1",
            "page_size": "20",
        ])
        XCTAssertNil(request?.bearerToken, "访客请求不带 Authorization")
    }

    func testSwitchingGameTypeReloadsAndDiscardsInflightResult() async {
        let server = MockHTTPServer()
        let session = SpectatorLobbyFixtures.makeGuestSession(server)
        let gate = DispatchSemaphore(value: 0)
        server.stub("GET", "/api/public/matches") { request in
            if request.query["game_type"] == "2" {
                return SpectatorLobbyFixtures.listResponse([
                    SpectatorLobbyFixtures.itemJSON(id: 200, gameType: 2, gameTypeName: "九球追分"),
                ])
            }
            _ = gate.wait(timeout: .now() + 5)
            return SpectatorLobbyFixtures.listResponse([
                SpectatorLobbyFixtures.itemJSON(id: 100, gameType: 3, gameTypeName: "中式八球"),
            ])
        }
        let model = SpectatorLobbyModel(session: session)

        let firstLoad = Task { await model.loadIfNeeded() }
        await waitUntil { server.requestCount("/api/public/matches") == 1 }
        await model.selectGameType(2)

        XCTAssertEqual(model.gameType, 2)
        XCTAssertEqual(model.cards.map(\.id), [200])
        XCTAssertEqual(server.requests("/api/public/matches").last?.query["game_type"], "2")

        gate.signal()
        await firstLoad.value
        XCTAssertEqual(model.cards.map(\.id), [200], "旧筛选条件的迟到结果不得覆盖新列表")
        XCTAssertEqual(model.phase, .loaded)
    }

    func testSwitchingTabReloadsAndDiscardsInflightResult() async {
        let server = MockHTTPServer()
        let session = SpectatorLobbyFixtures.makeGuestSession(server)
        let gate = DispatchSemaphore(value: 0)
        server.stub("GET", "/api/public/matches") { request in
            if request.query["status"] == "2" {
                return SpectatorLobbyFixtures.listResponse([
                    SpectatorLobbyFixtures.itemJSON(id: 2, status: 2),
                ])
            }
            _ = gate.wait(timeout: .now() + 5)
            return SpectatorLobbyFixtures.listResponse([
                SpectatorLobbyFixtures.itemJSON(id: 1, status: 1),
            ])
        }
        let model = SpectatorLobbyModel(session: session)

        let firstLoad = Task { await model.loadIfNeeded() }
        await waitUntil { server.requestCount("/api/public/matches") == 1 }
        await model.selectTab(.finished)

        XCTAssertEqual(model.tab, .finished)
        XCTAssertEqual(model.cards.map(\.id), [2])

        gate.signal()
        await firstLoad.value
        XCTAssertEqual(model.cards.map(\.id), [2])
        XCTAssertEqual(server.requests("/api/public/matches").last?.query["status"], "2")
    }

    func testPaginationAppendsNextPageAndDeduplicatesById() async {
        let server = MockHTTPServer()
        let session = SpectatorLobbyFixtures.makeGuestSession(server)
        let firstPage = (1...20).map { SpectatorLobbyFixtures.itemJSON(id: Int64($0)) }
        let secondPage = (20...25).map { SpectatorLobbyFixtures.itemJSON(id: Int64($0)) }
        server.stub("GET", "/api/public/matches") { request in
            request.query["page"] == "2"
                ? SpectatorLobbyFixtures.listResponse(secondPage, total: 25)
                : SpectatorLobbyFixtures.listResponse(firstPage, total: 25)
        }
        let model = SpectatorLobbyModel(session: session)

        await model.loadIfNeeded()
        XCTAssertEqual(model.cards.count, 20)
        XCTAssertTrue(model.hasMore)

        await model.loadMore()
        XCTAssertEqual(model.cards.map(\.id), (1...25).map { Int64($0) })
        XCTAssertFalse(model.hasMore)
        XCTAssertEqual(server.requests("/api/public/matches").last?.query["page"], "2")
    }

    func testRefreshFailureKeepsExistingListWithBanner() async {
        let server = MockHTTPServer()
        let session = SpectatorLobbyFixtures.makeGuestSession(server)
        server.stub("GET", "/api/public/matches", SpectatorLobbyFixtures.listResponse([
            SpectatorLobbyFixtures.itemJSON(id: 1),
        ]))
        let model = SpectatorLobbyModel(session: session)
        await model.loadIfNeeded()

        server.stub("GET", "/api/public/matches", .raw("service down", status: 503))
        await model.refresh()

        XCTAssertEqual(model.cards.map(\.id), [1])
        XCTAssertEqual(model.phase, .loaded)
        XCTAssertEqual(model.refreshFailure?.category, .server)
    }

    func testEmptyListShowsEmptyState() async {
        let server = MockHTTPServer()
        let session = SpectatorLobbyFixtures.makeGuestSession(server)
        server.stub("GET", "/api/public/matches", SpectatorLobbyFixtures.listResponse([]))
        let model = SpectatorLobbyModel(session: session)

        await model.loadIfNeeded()

        XCTAssertEqual(model.phase, .empty)
        XCTAssertTrue(model.cards.isEmpty)
        XCTAssertEqual(model.emptyState.title, "暂无正在进行的对局")
    }

    func testFailureCanRetryIntoLoadedState() async {
        let server = MockHTTPServer()
        let session = SpectatorLobbyFixtures.makeGuestSession(server)
        server.stub("GET", "/api/public/matches", .raw("service down", status: 503))
        let model = SpectatorLobbyModel(session: session)

        await model.loadIfNeeded()
        guard case .failed(let failure) = model.phase else { return XCTFail("应进入失败态") }
        XCTAssertEqual(failure.category, .server)

        server.stub("GET", "/api/public/matches", SpectatorLobbyFixtures.listResponse([
            SpectatorLobbyFixtures.itemJSON(id: 7),
        ]))
        await model.retry()

        XCTAssertEqual(model.cards.map(\.id), [7])
        XCTAssertEqual(model.phase, .loaded)
        XCTAssertNil(model.refreshFailure)
    }

    func testLoginAndLogoutReloadAndUpdateTheMeMark() async throws {
        let server = MockHTTPServer()
        let session = SpectatorLobbyFixtures.makeGuestSession(server)
        server.stub("GET", "/api/public/matches", SpectatorLobbyFixtures.listResponse([
            SpectatorLobbyFixtures.itemJSON(id: 1, player1Id: 1, player2Id: 2),
        ]))
        let model = SpectatorLobbyModel(session: session)

        await model.loadIfNeeded()
        XCTAssertFalse(model.cards[0].players[0].isMe, "访客无法识别「我」")
        XCTAssertEqual(model.tab, .live)

        server.stub("POST", "/api/auth/login", .json(AuthFixtures.loginResponse(
            user: AuthFixtures.userInfo(id: 1, nickname: "用户A")
        )))
        try await session.login(phone: AuthFixtures.phone, code: "123456")

        model.syncIdentity()
        XCTAssertTrue(model.isLoggedIn)
        XCTAssertEqual(model.tabs.map(\.label), ["好友", "进行中", "已结束"])
        await model.loadIfNeeded()
        XCTAssertTrue(model.cards[0].players[0].isMe)
        XCTAssertTrue(model.cards[0].isMine)
        XCTAssertEqual(server.requests("/api/public/matches").last?.bearerToken, "login-access-token")

        await model.selectTab(.friends)
        XCTAssertEqual(model.tab, .friends)
        XCTAssertEqual(server.requests("/api/public/matches").last?.query["scope"], "friends")
        XCTAssertNil(server.requests("/api/public/matches").last?.query["status"])

        session.logout()
        model.syncIdentity()
        XCTAssertFalse(model.isLoggedIn)
        XCTAssertEqual(model.tab, .live, "退出后「好友」标签不再可用")
        XCTAssertEqual(model.tabs.map(\.label), ["进行中", "已结束"])
        await model.loadIfNeeded()
        XCTAssertFalse(model.cards[0].players[0].isMe)
        let guestRequest = server.requests("/api/public/matches").last
        XCTAssertEqual(guestRequest?.query["scope"], "hall")
        XCTAssertNil(guestRequest?.bearerToken)
    }

    func testFriendsTabForSignedInUsersHidesStatusFilter() async {
        let server = MockHTTPServer()
        let session = await SpectatorLobbyFixtures.makeSignedInSession(server)
        server.stub("GET", "/api/public/matches", SpectatorLobbyFixtures.listResponse([
            SpectatorLobbyFixtures.itemJSON(id: 1, status: 1),
            SpectatorLobbyFixtures.itemJSON(id: 2, status: 2),
        ]))
        let model = SpectatorLobbyModel(session: session)

        await model.loadIfNeeded()

        XCTAssertEqual(model.tab, .friends)
        XCTAssertEqual(model.sections.map(\.title), ["进行中", "已结束"])
        let query = server.requests("/api/public/matches").first?.query
        XCTAssertEqual(query?["scope"], "friends")
        XCTAssertNil(query?["status"])
        XCTAssertEqual(query?["page"], "1")
        XCTAssertEqual(query?["page_size"], "20")
    }
}
