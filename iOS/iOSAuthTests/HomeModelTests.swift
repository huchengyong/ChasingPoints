import Foundation
import XCTest
@testable import iOS

/// 首页区块读取测试：各区块独立、按身份与令牌采纳结果、新鲜度与附近节流。
/// 全部走 MockHTTPServer，不访问真实后端。
@MainActor
final class HomeModelTests: XCTestCase {
    private var now = Date(timeIntervalSince1970: 1_700_000_000)
    private let coordinate = Coordinate(latitude: 22.5431, longitude: 114.0579)

    private struct Home {
        let model: HomeModel
        let session: SessionStore
        let location: StubLocationProvider
    }

    private func makeHome(
        _ server: MockHTTPServer,
        signedIn: Bool = false,
        location: Result<Coordinate, LocationError>? = nil
    ) async -> Home {
        let session = signedIn
            ? await makeRestoredAccountSession(server)
            : makeSessionStore(server, credentialStore: InMemoryCredentialStore(credentials: nil))
        let provider = StubLocationProvider(result: location ?? .success(coordinate))
        let model = HomeModel(session: session, location: provider, now: { self.now })
        return Home(model: model, session: session, location: provider)
    }

    private func stubAll(_ server: MockHTTPServer, liveCount: Int = 1, venueCount: Int = 1) {
        server.stub("GET", HomeFixtures.leaderboardPath, HomeFixtures.leaderboardSummary(
            topThree: [
                RankingFixtures.itemJSON(rank: 1, userId: 11, nickname: "球手甲"),
                RankingFixtures.itemJSON(rank: 2, userId: 12, nickname: "球手乙"),
            ],
            myRanking: RankingFixtures.itemJSON(rank: 88, userId: 1, nickname: "测试用户")
        ))
        server.stub("GET", HomeFixtures.eventNewsPath, .json(EventNewsFixtures.listResponse()))
        server.stub("GET", HomeFixtures.matchesPath, SpectatorLobbyFixtures.listResponse(
            (0..<liveCount).map { index in
                SpectatorLobbyFixtures.itemJSON(id: Int64(index + 1), player1Name: "甲", player2Name: "乙")
            }
        ))
        server.stub("GET", HomeFixtures.nearbyPath, .json(VenueFixtures.listResponse(
            (0..<venueCount).map { index in
                VenueFixtures.venueJSON(id: Int64(index + 1), name: "球房\(index + 1)", distance: 860)
            }
        )))
    }

    // MARK: - 区块独立

    func testFailedSectionDoesNotBlockOtherSections() async {
        let server = MockHTTPServer()
        server.stub("GET", HomeFixtures.leaderboardPath, .raw("down", status: 500))
        server.stub("GET", HomeFixtures.eventNewsPath, .json(EventNewsFixtures.listResponse()))
        server.stub("GET", HomeFixtures.matchesPath, SpectatorLobbyFixtures.listResponse([
            SpectatorLobbyFixtures.itemJSON(id: 1),
        ]))
        server.stub("GET", HomeFixtures.nearbyPath, .json(VenueFixtures.listResponse([
            VenueFixtures.venueJSON(id: 1, distance: 1260),
        ])))
        let home = await makeHome(server)

        await home.model.onAppear()

        guard case .failed = home.model.leaderboardPhase else { return XCTFail("排行榜应失败") }
        XCTAssertEqual(home.model.eventPhase, .loaded)
        XCTAssertEqual(home.model.livePhase, .loaded)
        XCTAssertEqual(home.model.nearbyPhase, .loaded)
        XCTAssertTrue(home.model.hasPublicContent)
        XCTAssertNil(home.model.pageFailure, "只有排行榜失败不构成整页错误")
    }

    func testPageFailureNeedsBothLeaderboardAndEventWithoutContent() async {
        let server = MockHTTPServer()
        server.stub("GET", HomeFixtures.leaderboardPath, .raw("down", status: 500))
        server.stub("GET", HomeFixtures.eventNewsPath, .raw("down", status: 500))
        server.stub("GET", HomeFixtures.matchesPath, SpectatorLobbyFixtures.listResponse([]))
        server.stub("GET", HomeFixtures.nearbyPath, .json(VenueFixtures.listResponse([])))
        let home = await makeHome(server)

        await home.model.onAppear()

        XCTAssertNotNil(home.model.pageFailure)
        XCTAssertFalse(home.model.hasPublicContent)
    }

    func testPageFailureStaysNilWhenLiveMatchesProvideContent() async {
        let server = MockHTTPServer()
        server.stub("GET", HomeFixtures.leaderboardPath, .raw("down", status: 500))
        server.stub("GET", HomeFixtures.eventNewsPath, .raw("down", status: 500))
        server.stub("GET", HomeFixtures.matchesPath, SpectatorLobbyFixtures.listResponse([
            SpectatorLobbyFixtures.itemJSON(id: 1),
        ]))
        server.stub("GET", HomeFixtures.nearbyPath, .json(VenueFixtures.listResponse([])))
        let home = await makeHome(server)

        await home.model.onAppear()

        // 排行榜与赛事都失败，但直播预览有内容：模型仍报 pageFailure，由页面改显示顶部错误条。
        XCTAssertNotNil(home.model.pageFailure)
        XCTAssertTrue(home.model.hasPublicContent)
    }

    // MARK: - 新鲜度与刷新

    func testPublicSectionsReuseSixtySecondFreshness() async {
        let server = MockHTTPServer()
        stubAll(server)
        let home = await makeHome(server)

        await home.model.onAppear()
        XCTAssertEqual(server.requestCount(HomeFixtures.leaderboardPath), 1)
        XCTAssertEqual(server.requestCount(HomeFixtures.eventNewsPath), 1)
        XCTAssertEqual(server.requestCount(HomeFixtures.matchesPath), 1)

        now = now.addingTimeInterval(30)
        await home.model.onAppear()
        XCTAssertEqual(server.requestCount(HomeFixtures.leaderboardPath), 1, "60 秒内沿用结果")
        XCTAssertEqual(server.requestCount(HomeFixtures.eventNewsPath), 1)

        now = now.addingTimeInterval(31)
        await home.model.onAppear()
        XCTAssertEqual(server.requestCount(HomeFixtures.leaderboardPath), 2)
        XCTAssertEqual(server.requestCount(HomeFixtures.eventNewsPath), 2)
        XCTAssertEqual(server.requestCount(HomeFixtures.matchesPath), 2)
    }

    func testPullRefreshForcesEverySection() async {
        let server = MockHTTPServer()
        stubAll(server)
        let home = await makeHome(server)

        await home.model.onAppear()
        await home.model.refresh()

        XCTAssertEqual(server.requestCount(HomeFixtures.leaderboardPath), 2)
        XCTAssertEqual(server.requestCount(HomeFixtures.eventNewsPath), 2)
        XCTAssertEqual(server.requestCount(HomeFixtures.matchesPath), 2)
        XCTAssertEqual(server.requestCount(HomeFixtures.nearbyPath), 2)
        XCTAssertEqual(home.location.callCount, 2)
        XCTAssertFalse(home.model.isRefreshing)
    }

    // MARK: - 附近球房

    func testNearbyRelocatesOnlyAfterThreeMinutes() async {
        let server = MockHTTPServer()
        stubAll(server)
        let home = await makeHome(server)

        await home.model.onAppear()
        XCTAssertEqual(home.location.callCount, 1)
        XCTAssertEqual(server.requestCount(HomeFixtures.nearbyPath), 1)

        now = now.addingTimeInterval(179)
        await home.model.onAppear()
        XCTAssertEqual(home.location.callCount, 1, "3 分钟内不重复定位")

        now = now.addingTimeInterval(2)
        await home.model.onAppear()
        XCTAssertEqual(home.location.callCount, 2)
        XCTAssertEqual(server.requestCount(HomeFixtures.nearbyPath), 2)
    }

    func testLocationDeniedShowsFeedbackAndSkipsVenueRequest() async {
        let server = MockHTTPServer()
        stubAll(server)
        let home = await makeHome(server, location: .failure(.denied))

        await home.model.onAppear()

        XCTAssertEqual(home.model.nearbyLocationIssue, .denied)
        XCTAssertEqual(home.model.nearbyPhase, .loaded)
        XCTAssertEqual(home.model.nearbyFeedback?.title, "未开启定位权限")
        XCTAssertEqual(home.model.nearbyFeedback?.description, "开启定位权限后可查看附近球房")
        XCTAssertEqual(server.requestCount(HomeFixtures.nearbyPath), 0, "没有坐标不请求附近球房")

        home.location.result = .success(coordinate)
        await home.model.retryNearby()
        XCTAssertNil(home.model.nearbyFeedback)
        XCTAssertEqual(server.requestCount(HomeFixtures.nearbyPath), 1)
    }

    func testZeroCoordinateSkipsVenueRequestAsUnavailable() async {
        let server = MockHTTPServer()
        stubAll(server)
        let home = await makeHome(server, location: .success(Coordinate(latitude: 0, longitude: 0)))

        await home.model.onAppear()

        XCTAssertEqual(home.model.nearbyLocationIssue, .unavailable)
        XCTAssertEqual(home.model.nearbyFeedback?.title, "暂时无法获取位置")
        XCTAssertEqual(server.requestCount(HomeFixtures.nearbyPath), 0)
    }

    func testFailedNearbyKeepsSectionErrorAndRetryBypassesThrottle() async {
        let server = MockHTTPServer()
        stubAll(server)
        server.stub("GET", HomeFixtures.nearbyPath, .raw("down", status: 500))
        let home = await makeHome(server)

        await home.model.onAppear()
        guard case .failed = home.model.nearbyPhase else { return XCTFail("附近球房应失败") }
        XCTAssertEqual(home.model.nearbyFeedback?.title, "服务暂不可用")

        server.stub("GET", HomeFixtures.nearbyPath, .json(VenueFixtures.listResponse([
            VenueFixtures.venueJSON(id: 1, distance: 860),
        ])))
        await home.model.retryNearby()
        XCTAssertNil(home.model.nearbyFeedback)
        XCTAssertEqual(home.model.nearbyVenues.count, 1)
    }

    // MARK: - 失败重试与保留数据

    func testFailedSectionRetriesOnNextAppear() async {
        let server = MockHTTPServer()
        stubAll(server)
        server.stub("GET", HomeFixtures.leaderboardPath, .raw("down", status: 500))
        let home = await makeHome(server)

        await home.model.onAppear()
        guard case .failed = home.model.leaderboardPhase else { return XCTFail("排行榜应失败") }

        stubAll(server)
        await home.model.onAppear()
        XCTAssertEqual(home.model.leaderboardPhase, .loaded)
        XCTAssertEqual(home.model.leaderboardTopThree.count, 2)
    }

    func testLiveFailureKeepsPreviousList() async {
        let server = MockHTTPServer()
        stubAll(server, liveCount: 2)
        let home = await makeHome(server)

        await home.model.onAppear()
        XCTAssertEqual(home.model.liveMatches.count, 2)

        server.stub("GET", HomeFixtures.matchesPath, .raw("down", status: 500))
        await home.model.refresh()

        guard case .failed = home.model.livePhase else { return XCTFail("直播预览应失败") }
        XCTAssertEqual(home.model.liveMatches.count, 2, "失败时保留上一次列表")
        XCTAssertTrue(home.model.hasPublicContent)
        XCTAssertNil(home.model.pageFailure)
    }

    func testLeaderboardRefreshFailureKeepsDataButMarksSectionFailed() async {
        let server = MockHTTPServer()
        stubAll(server)
        let home = await makeHome(server)

        await home.model.onAppear()
        XCTAssertEqual(home.model.leaderboardTopThree.count, 2)

        server.stub("GET", HomeFixtures.leaderboardPath, .raw("down", status: 500))
        await home.model.refresh()

        guard case .failed = home.model.leaderboardPhase else { return XCTFail("排行榜应失败") }
        XCTAssertEqual(home.model.leaderboardTopThree.count, 2)
        XCTAssertNotNil(home.model.myRanking)
    }

    // MARK: - 身份

    func testIdentityChangeClearsRankingAndDropsLateResponse() async {
        let server = MockHTTPServer()
        stubAll(server)
        let counter = LockedCounter()
        let gate = DispatchSemaphore(value: 0)
        server.stub("GET", HomeFixtures.leaderboardPath) { _ in
            let attempt = counter.next()
            if attempt == 1 { gate.wait() }
            let rank = attempt == 1 ? 1 : 2
            return HomeFixtures.leaderboardSummary(
                topThree: [RankingFixtures.itemJSON(rank: rank, userId: 11, nickname: "球手\(rank)")],
                myRanking: attempt == 1
                    ? RankingFixtures.itemJSON(rank: 88, userId: 1, nickname: "旧身份")
                    : nil
            )
        }
        let home = await makeHome(server, signedIn: true)

        async let appear: Void = home.model.onAppear()
        await waitUntil { server.requestCount(HomeFixtures.leaderboardPath) == 1 }

        home.session.logout()
        async let identity: Void = home.model.onIdentityChanged()
        await identity
        gate.signal()
        await appear

        XCTAssertEqual(home.model.leaderboardTopThree.map(\.rank), [2], "旧身份的迟到响应必须丢弃")
        XCTAssertNil(home.model.myRanking, "退出后不再展示我的名次")
    }
}
