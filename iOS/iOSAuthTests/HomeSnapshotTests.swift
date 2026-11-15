import SwiftUI
import XCTest
@testable import iOS

/// 首页快照：亮暗各一张，覆盖访客、登录、进行中、局部错误与整页错误。
@MainActor
final class HomeSnapshotTests: XCTestCase {
    private let size = CGSize(width: 393, height: 852)
    private let coordinate = Coordinate(latitude: 22.5431, longitude: 114.0579)

    private func eventNewsResponse() -> MockHTTPServer.Stub {
        .json([
            "success": true,
            "total": 1,
            "list": [
                EventNewsFixtures.eventNewsJSON(
                    title: "世界斯诺克公开赛",
                    sourceType: "",
                    tournamentName: "世界斯诺克公开赛",
                    status: 0,
                    currentRoundText: "1/4 决赛"
                ),
            ],
        ])
    }

    private func stubPublicSections(_ server: MockHTTPServer, live: Bool = true) {
        server.stub("GET", HomeFixtures.leaderboardPath, HomeFixtures.leaderboardSummary(
            topThree: [
                RankingFixtures.itemJSON(rank: 1, userId: 11, nickname: "球手甲"),
                RankingFixtures.itemJSON(rank: 2, userId: 12, nickname: "球手乙"),
                RankingFixtures.itemJSON(rank: 3, userId: 13, nickname: "球手丙"),
            ],
            myRanking: RankingFixtures.itemJSON(rank: 88, userId: 1, nickname: "测试用户")
        ))
        server.stub("GET", HomeFixtures.eventNewsPath, eventNewsResponse())
        server.stub("GET", HomeFixtures.matchesPath, SpectatorLobbyFixtures.listResponse(
            live ? [
                SpectatorLobbyFixtures.itemJSON(
                    id: 1, player1Name: "甲", player2Name: "乙"
                ),
                SpectatorLobbyFixtures.itemJSON(
                    id: 2, player1Name: "小林", player1Score: 3, player2Name: "阿杰", player2Score: 2
                ),
            ] : []
        ))
        server.stub("GET", HomeFixtures.nearbyPath, .json(VenueFixtures.listResponse([
            VenueFixtures.venueJSON(id: 1, name: "星轨台球俱乐部", address: "科苑路 8 号", distance: 860),
            VenueFixtures.venueJSON(id: 2, name: "南山桌球馆", address: "海德三道 2 号", distance: 1260),
        ])))
    }

    private func makeModel(
        _ server: MockHTTPServer,
        session: SessionStore
    ) -> HomeModel {
        HomeModel(
            session: session,
            location: StubLocationProvider(result: .success(coordinate)),
            now: { HomeFixtures.now }
        )
    }

    private func snapshotContent(
        session: SessionStore,
        model: HomeModel,
        activity: ActivityStore
    ) -> some View {
        let navigator = AppNavigator()
        let toast = ToastCenter()
        return NavigationStack {
            HomeView(session: session, model: model)
                .navigationDestination(for: AppRoute.self) { AppRouteDestination(route: $0) }
        }
        .environmentObject(session)
        .environmentObject(navigator)
        .environmentObject(activity)
        .environmentObject(toast)
    }

    private func render(
        _ name: String,
        session: SessionStore,
        model: HomeModel,
        activity: ActivityStore
    ) throws {
        for scheme in [ColorScheme.light, .dark] {
            let image = try renderSnapshot(name, colorScheme: scheme, size: size) {
                snapshotContent(session: session, model: model, activity: activity)
            }
            XCTAssertEqual(image.size, size)
        }
    }

    // MARK: - 访客

    func testGuestSnapshot() async throws {
        let server = MockHTTPServer()
        stubPublicSections(server)
        let session = makeSessionStore(server, credentialStore: InMemoryCredentialStore(credentials: nil))
        let activity = ActivityStore(session: session)
        let model = makeModel(server, session: session)
        await model.onAppear()
        XCTAssertTrue(model.hasPublicContent)

        try render("home-guest", session: session, model: model, activity: activity)
    }

    // MARK: - 登录

    func testSignedInSnapshot() async throws {
        let server = MockHTTPServer()
        stubPublicSections(server)
        let session = await makeRestoredAccountSession(server)
        server.stub("GET", HomeFixtures.bootstrapPath, .json(HomeFixtures.bootstrap(challenge: HomeFixtures.challenge(
            id: 5,
            fromUserId: 7,
            fromNickname: "阿杰",
            toUserId: 1,
            toNickname: "测试用户",
            status: 0
        ))))
        let activity = ActivityStore(session: session)
        await activity.refresh()
        let model = makeModel(server, session: session)
        await model.onAppear()

        try render("home-signed-in", session: session, model: model, activity: activity)
    }

    // MARK: - 进行中

    func testOngoingMatchSnapshot() async throws {
        let server = MockHTTPServer()
        stubPublicSections(server)
        let session = await makeRestoredAccountSession(server)
        server.stub("GET", HomeFixtures.bootstrapPath, .json(HomeFixtures.bootstrap(
            currentMatch: HomeFixtures.currentMatch(
                id: 31,
                viewerRole: "player1",
                durationSeconds: 720,
                myScore: 3,
                opponentScore: 5,
                player1Name: "测试用户",
                player2Name: "球友 B",
                opponentName: "球友 B"
            )
        )))
        let activity = ActivityStore(session: session)
        await activity.refresh()
        let model = makeModel(server, session: session)
        await model.onAppear()
        XCTAssertNotNil(activity.currentMatch)

        try render("home-ongoing-match", session: session, model: model, activity: activity)
    }

    // MARK: - 局部错误

    func testPartialErrorSnapshot() async throws {
        let server = MockHTTPServer()
        stubPublicSections(server)
        server.stub("GET", HomeFixtures.eventNewsPath, .raw("down", status: 503))
        let session = makeSessionStore(server, credentialStore: InMemoryCredentialStore(credentials: nil))
        let activity = ActivityStore(session: session)
        let model = makeModel(server, session: session)
        await model.onAppear()
        XCTAssertTrue(model.hasPublicContent)
        XCTAssertNil(model.pageFailure)

        try render("home-partial-error", session: session, model: model, activity: activity)
    }

    // MARK: - 空态

    func testEmptySnapshot() async throws {
        let server = MockHTTPServer()
        server.stub("GET", HomeFixtures.leaderboardPath, HomeFixtures.leaderboardSummary())
        server.stub("GET", HomeFixtures.eventNewsPath, .json(["success": true, "total": 0, "list": []]))
        server.stub("GET", HomeFixtures.matchesPath, SpectatorLobbyFixtures.listResponse([]))
        server.stub("GET", HomeFixtures.nearbyPath, .json(VenueFixtures.listResponse([])))
        let session = makeSessionStore(server, credentialStore: InMemoryCredentialStore(credentials: nil))
        let activity = ActivityStore(session: session)
        let model = makeModel(server, session: session)
        await model.onAppear()
        XCTAssertFalse(model.hasPublicContent)
        XCTAssertNil(model.pageFailure)

        try render("home-empty", session: session, model: model, activity: activity)
    }

    // MARK: - 整页错误

    func testAllErrorSnapshot() async throws {
        let server = MockHTTPServer()
        server.stub("GET", HomeFixtures.leaderboardPath, .raw("down", status: 503))
        server.stub("GET", HomeFixtures.eventNewsPath, .raw("down", status: 503))
        server.stub("GET", HomeFixtures.matchesPath, SpectatorLobbyFixtures.listResponse([]))
        server.stub("GET", HomeFixtures.nearbyPath, .json(VenueFixtures.listResponse([])))
        let session = makeSessionStore(server, credentialStore: InMemoryCredentialStore(credentials: nil))
        let activity = ActivityStore(session: session)
        let model = makeModel(server, session: session)
        await model.onAppear()
        XCTAssertNotNil(model.pageFailure)
        XCTAssertFalse(model.hasPublicContent)

        try render("home-all-error", session: session, model: model, activity: activity)
    }
}
