import Foundation
import XCTest
@testable import iOS

@MainActor
final class RankingTests: XCTestCase {
    private let path = "/api/public/rank/leaderboard"

    // MARK: - 领奖台

    func testPodiumKeepsThreeFixedSlotsWhenFewerThanThreePlayersExist() {
        let slots = PodiumSlot.build(topThree: [
            RankingFixtures.item(rank: 1, userId: 101, nickname: "冠军"),
            RankingFixtures.item(rank: 2, userId: 102, nickname: "亚军"),
        ])

        XCTAssertEqual(slots.map(\.placement), ["second", "first", "third"])
        XCTAssertEqual(slots.map(\.rank), [2, 1, 3])
        XCTAssertEqual(slots[0].user?.nickname, "亚军")
        XCTAssertEqual(slots[1].user?.nickname, "冠军")
        XCTAssertNil(slots[2].user)
    }

    func testPodiumPadsMissingEntriesWhenOnlyChampionExists() {
        let slots = PodiumSlot.build(topThree: [
            RankingFixtures.item(rank: 1, userId: 201, nickname: "第一名"),
        ])

        XCTAssertEqual(slots.count, 3)
        XCTAssertNil(slots[0].user)
        XCTAssertEqual(slots[1].user?.nickname, "第一名")
        XCTAssertNil(slots[2].user)
    }

    // MARK: - 读取

    func testGuestFirstLoadRequestsFirstPageWithoutAuthorization() async {
        let server = MockHTTPServer()
        let session = RankingFixtures.makeGuestSession(server)
        server.stub("GET", path, RankingFixtures.leaderboard(
            topThree: RankingFixtures.listItems(1...3),
            list: RankingFixtures.listItems(4...10),
            total: 10
        ))
        let model = RankingModel(session: session)

        await model.loadIfNeeded()

        XCTAssertEqual(model.phase, .loaded)
        XCTAssertEqual(model.topThree.map(\.rank), [1, 2, 3])
        XCTAssertEqual(model.list.map(\.rank), Array(4...10))
        XCTAssertNil(model.myRanking)
        let request = server.requests(path).first
        XCTAssertEqual(request?.query["game_type"], "3")
        XCTAssertEqual(request?.query["page"], "1")
        XCTAssertEqual(request?.query["page_size"], "20")
        XCTAssertNil(request?.bearerToken)
    }

    func testSignedInLoadSendsBearerToken() async {
        let server = MockHTTPServer()
        let session = await RankingFixtures.makeSignedInSession(server)
        server.stub("GET", path, RankingFixtures.leaderboard(
            topThree: RankingFixtures.listItems(1...3),
            total: 3
        ))
        let model = RankingModel(session: session)

        await model.loadIfNeeded()

        XCTAssertEqual(server.requests(path).first?.bearerToken, AuthFixtures.oldAccess)
    }

    func testReturningToLoadedRankingDoesNotRequestAgain() async {
        let server = MockHTTPServer()
        let session = RankingFixtures.makeGuestSession(server)
        server.stub("GET", path, RankingFixtures.leaderboard(
            topThree: RankingFixtures.listItems(1...3),
            total: 3
        ))
        let model = RankingModel(session: session)

        await model.loadIfNeeded()
        await model.loadIfNeeded()
        await model.loadIfNeeded()

        XCTAssertEqual(server.requestCount(path), 1)
    }

    func testEmptyLeaderboardIsEmptyPhaseNotFailure() async {
        let server = MockHTTPServer()
        let session = RankingFixtures.makeGuestSession(server)
        server.stub("GET", path, RankingFixtures.leaderboard(total: 0))
        let model = RankingModel(session: session)

        await model.loadIfNeeded()

        XCTAssertEqual(model.phase, .empty)
        XCTAssertTrue(model.topThree.isEmpty)
        XCTAssertTrue(model.list.isEmpty)
        XCTAssertFalse(model.hasMore)
    }

    func testFirstLoadFailureCanRetryIntoLoadedState() async {
        let server = MockHTTPServer()
        let session = RankingFixtures.makeGuestSession(server)
        server.stub("GET", path, .raw("service down", status: 503))
        let model = RankingModel(session: session)

        await model.loadIfNeeded()
        guard case .failed = model.phase else {
            return XCTFail("首页失败应进入错误态")
        }

        server.stub("GET", path, RankingFixtures.leaderboard(
            topThree: RankingFixtures.listItems(1...3),
            total: 3
        ))
        await model.retry()

        XCTAssertEqual(model.phase, .loaded)
        XCTAssertEqual(model.topThree.count, 3)
    }

    // MARK: - 分页

    func testLoadMoreAppendsNextPageWithoutTouchingPodium() async {
        let server = MockHTTPServer()
        let session = RankingFixtures.makeGuestSession(server)
        server.stub("GET", path) { request in
            let page = Int(request.query["page"] ?? "") ?? 1
            if page == 1 {
                return RankingFixtures.leaderboard(
                    topThree: RankingFixtures.listItems(1...3),
                    list: RankingFixtures.listItems(4...23),
                    total: 25
                )
            }
            return RankingFixtures.leaderboard(
                topThree: RankingFixtures.listItems(1...3),
                list: RankingFixtures.listItems(24...25),
                total: 25
            )
        }
        let model = RankingModel(session: session)
        await model.loadIfNeeded()
        XCTAssertEqual(model.list.count, 20)
        XCTAssertTrue(model.hasMore)

        await model.loadMore()

        XCTAssertEqual(model.topThree.map(\.rank), [1, 2, 3])
        XCTAssertEqual(model.list.map(\.rank), Array(4...25))
        XCTAssertEqual(model.page, 2)
        XCTAssertFalse(model.hasMore)
        XCTAssertEqual(server.requests(path).last?.query["page"], "2")

        await model.loadMore()
        XCTAssertEqual(server.requestCount(path), 2, "没有更多时不应再请求")
    }

    func testLoadMoreFailureKeepsPageAndAllowsRetry() async {
        let server = MockHTTPServer()
        let session = RankingFixtures.makeGuestSession(server)
        server.stub("GET", path, RankingFixtures.leaderboard(
            topThree: RankingFixtures.listItems(1...3),
            list: RankingFixtures.listItems(4...23),
            total: 25
        ))
        let model = RankingModel(session: session)
        await model.loadIfNeeded()

        server.stub("GET", path, .raw("service down", status: 503))
        await model.loadMore()

        XCTAssertEqual(model.page, 1, "失败不得推进页码")
        XCTAssertEqual(model.list.count, 20)
        XCTAssertEqual(model.phase, .loaded)
        XCTAssertNotNil(model.refreshFailure)

        server.stub("GET", path) { request in
            RankingFixtures.leaderboard(
                list: RankingFixtures.listItems(24...25),
                total: 25
            )
        }
        await model.loadMore()

        XCTAssertEqual(model.list.count, 22)
        XCTAssertNil(model.refreshFailure)
        XCTAssertEqual(model.page, 2)
    }

    func testRefreshFailureKeepsExistingRankingWithBanner() async {
        let server = MockHTTPServer()
        let session = RankingFixtures.makeGuestSession(server)
        server.stub("GET", path, RankingFixtures.leaderboard(
            topThree: RankingFixtures.listItems(1...3),
            total: 3
        ))
        let model = RankingModel(session: session)
        await model.loadIfNeeded()

        server.stub("GET", path, .raw("service down", status: 503))
        await model.refresh()

        XCTAssertEqual(model.phase, .loaded)
        XCTAssertEqual(model.topThree.map(\.rank), [1, 2, 3])
        XCTAssertFalse(model.isRefreshing)
        XCTAssertNotNil(model.refreshFailure)
    }

    // MARK: - 球种与身份

    func testSwitchingGameTypeDiscardsStaleResponse() async {
        let server = MockHTTPServer()
        let session = RankingFixtures.makeGuestSession(server)
        let gate = DispatchSemaphore(value: 0)
        server.stub("GET", path) { request in
            if request.query["game_type"] == "2" {
                _ = gate.wait(timeout: .now() + 5)
                return RankingFixtures.leaderboard(
                    topThree: [RankingFixtures.itemJSON(rank: 1, userId: 21, nickname: "九球榜")],
                    total: 1
                )
            }
            return RankingFixtures.leaderboard(
                topThree: [RankingFixtures.itemJSON(rank: 1, userId: 41, nickname: "美式榜")],
                total: 1
            )
        }
        let model = RankingModel(session: session)

        let stale = Task { await model.selectGameType(2) }
        await waitUntil { server.requestCount(path) == 1 }
        await model.selectGameType(4)
        gate.signal()
        await stale.value

        XCTAssertEqual(model.currentGameType, 4)
        XCTAssertEqual(model.topThree.first?.nickname, "美式榜", "旧球种的迟到结果不得覆盖新球种")
        XCTAssertEqual(model.phase, .loaded)
    }

    func testLoginAfterGuestLoadRereadsWithMyRanking() async throws {
        let server = MockHTTPServer()
        let session = RankingFixtures.makeGuestSession(server)
        server.stub("GET", path, RankingFixtures.leaderboard(
            topThree: RankingFixtures.listItems(1...3),
            total: 10
        ))
        let model = RankingModel(session: session)
        await model.loadIfNeeded()
        XCTAssertNil(model.myRanking)

        server.stub("GET", path) { request in
            RankingFixtures.leaderboard(
                topThree: RankingFixtures.listItems(1...3),
                total: 10,
                myRanking: request.bearerToken == nil
                    ? nil
                    : RankingFixtures.itemJSON(rank: 8, userId: 1, nickname: "测试用户")
            )
        }
        server.stub("POST", "/api/auth/login", .json(AuthFixtures.loginResponse()))
        try await session.login(phone: AuthFixtures.phone, code: "123456")

        model.syncIdentity()
        XCTAssertTrue(model.topThree.isEmpty, "换身份后不得沿用访客结果")
        await model.loadIfNeeded()

        XCTAssertEqual(model.myRanking?.rank, 8)
        XCTAssertEqual(server.requests(path).last?.bearerToken, "login-access-token")
    }

    func testLogoutClearsMyRankingAndRereadsAsGuest() async {
        let server = MockHTTPServer()
        let session = await RankingFixtures.makeSignedInSession(server)
        server.stub("GET", path) { request in
            RankingFixtures.leaderboard(
                topThree: RankingFixtures.listItems(1...3),
                total: 10,
                myRanking: request.bearerToken == nil
                    ? nil
                    : RankingFixtures.itemJSON(rank: 8, userId: 1, nickname: "测试用户")
            )
        }
        let model = RankingModel(session: session)
        await model.loadIfNeeded()
        XCTAssertEqual(model.myRanking?.rank, 8)

        session.logout()
        model.syncIdentity()

        XCTAssertNil(model.myRanking)
        XCTAssertEqual(model.phase, .idle)

        await model.loadIfNeeded()

        XCTAssertNil(model.myRanking)
        XCTAssertNil(server.requests(path).last?.bearerToken)
    }
}
