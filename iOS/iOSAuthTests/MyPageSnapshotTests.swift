import SwiftUI
import XCTest
@testable import iOS

@MainActor
final class MyPageSnapshotTests: XCTestCase {
    private let pageSize = CGSize(width: 393, height: 852)

    func testGuestSnapshot() async throws {
        let server = MockHTTPServer()
        let session = makeSessionStore(server, credentialStore: InMemoryCredentialStore(credentials: nil))
        let navigator = AppNavigator()
        let toast = ToastCenter()

        for scheme in [ColorScheme.light, .dark] {
            let image = try renderSnapshot("my-guest", colorScheme: scheme, size: pageSize) {
                page(session, navigator, toast)
            }
            XCTAssertEqual(image.size, pageSize)
        }
        XCTAssertEqual(server.requests.count, 0, "访客不发起任何读取")
    }

    func testMemberSnapshot() async throws {
        let userId: Int64 = 8801
        clearStandardDefaults(userId: userId)

        let server = MockHTTPServer()
        server.stub(
            "GET",
            "/api/user/info",
            .json(AuthFixtures.userInfoResponse(AuthFixtures.userInfo(id: userId, nickname: "阿追")))
        )
        server.stub("GET", "/api/user/overview", .json(ProfileFixtures.overview(
            stats: ProfileFixtures.userStats(),
            reputation: ProfileFixtures.reputation(),
            memberStatus: ProfileFixtures.memberStatusJSON(
                isActive: true,
                memberExpiresAt: "2099-08-01 12:00:00",
                growthLevel: 3
            ),
            favoriteVenueReward: ProfileFixtures.favoriteVenueRewardJSON(
                status: "pending_review",
                submittedVenueName: "南山台球会"
            )
        )))
        server.stub("GET", "/api/rank/infos", .json(ProfileFixtures.rankInfos([
            ProfileFixtures.rankInfoItem(
                gameType: 3,
                rankInfo: ProfileFixtures.rankInfoJSON(
                    level: 3,
                    name: "黄金",
                    rankScore: 1320,
                    totalWins: 30,
                    totalLosses: 10,
                    progress: 64
                )
            ),
            ProfileFixtures.rankInfoItem(
                gameType: 2,
                rankInfo: ProfileFixtures.rankInfoJSON(
                    level: 2,
                    name: "白银",
                    rankScore: 900,
                    totalWins: 8,
                    totalLosses: 6
                )
            ),
        ])))
        server.stub("GET", "/api/notification/list", .json(AchievementFixtures.notificationList([])))

        let session = makeSessionStore(
            server,
            credentialStore: InMemoryCredentialStore(credentials: AuthFixtures.credentials())
        )
        await session.restore()
        XCTAssertNotNil(session.identity)

        let navigator = AppNavigator()
        let toast = ToastCenter()
        // 页面内的 `.task` 在快照渲染的同步等待里跑不完网络读取，这里先把数据读进模型再渲染。
        let model = MyPageModel(session: session)
        await model.onAppear()
        XCTAssertEqual(server.requestCount("/api/user/overview"), 1)
        XCTAssertEqual(server.requestCount("/api/rank/infos"), 1)

        for scheme in [ColorScheme.light, .dark] {
            let image = try renderSnapshot("my-member", colorScheme: scheme, size: pageSize) {
                page(session, navigator, toast, model: model)
            }
            XCTAssertEqual(image.size, pageSize)
        }
        XCTAssertEqual(server.requestCount("/api/user/overview"), 1, "渲染时数据已就绪，不应重复读取")
        XCTAssertEqual(server.requestCount("/api/rank/infos"), 1)
    }

    func testNewcomerSnapshot() async throws {
        let userId: Int64 = 8802
        clearStandardDefaults(userId: userId)

        let server = MockHTTPServer()
        server.stub(
            "GET",
            "/api/user/info",
            .json(AuthFixtures.userInfoResponse(AuthFixtures.userInfo(id: userId, nickname: "新手")))
        )
        server.stub("GET", "/api/user/overview", .json(ProfileFixtures.overview(
            stats: ProfileFixtures.userStats(totalMatches: 0, wins: 0, losses: 0, winRate: 0, maxWinStreak: 0),
            reputation: ProfileFixtures.reputation(),
            favoriteVenueReward: ProfileFixtures.favoriteVenueRewardJSON(status: "not_started")
        )))
        server.stub("GET", "/api/rank/infos", .json(ProfileFixtures.rankInfos(
            GameTypeCatalog.ordered.map {
                ProfileFixtures.rankInfoItem(
                    gameType: $0.value,
                    rankInfo: ProfileFixtures.rankInfoJSON(level: 1, name: "青铜")
                )
            }
        )))
        server.stub("GET", "/api/notification/list", .json(AchievementFixtures.notificationList([])))

        let session = makeSessionStore(
            server,
            credentialStore: InMemoryCredentialStore(credentials: AuthFixtures.credentials())
        )
        await session.restore()
        XCTAssertNotNil(session.identity)

        let navigator = AppNavigator()
        let toast = ToastCenter()
        let model = MyPageModel(session: session)
        await model.onAppear()

        for scheme in [ColorScheme.light, .dark] {
            let image = try renderSnapshot("my-newcomer", colorScheme: scheme, size: pageSize) {
                page(session, navigator, toast, model: model)
            }
            XCTAssertEqual(image.size, pageSize)
        }
    }

    func testMatchQRCodeSnapshot() async throws {
        let server = MockHTTPServer()
        let session = makeSessionStore(
            server,
            credentialStore: InMemoryCredentialStore(credentials: AuthFixtures.credentials())
        )
        server.stub("GET", "/api/match/qrcode", .json(ProfileFixtures.matchQRCode()))
        let model = MatchQRCodeModel(session: session)
        await model.load()
        if case .ready = model.phase {} else {
            XCTFail("二维码应生成成功，实际为 \(model.phase)")
        }

        let size = CGSize(width: 393, height: 420)
        for scheme in [ColorScheme.light, .dark] {
            let image = try renderSnapshot("my-qrcode", colorScheme: scheme, size: size) {
                MatchQRCodeContent(model: model, onClose: {})
            }
            XCTAssertEqual(image.size, size)
        }
    }

    private func page(
        _ session: SessionStore,
        _ navigator: AppNavigator,
        _ toast: ToastCenter,
        model: MyPageModel? = nil
    ) -> some View {
        NavigationStack {
            MyPage(session: session, model: model)
                .navigationDestination(for: AppRoute.self) { AppRouteDestination(route: $0) }
        }
        .environmentObject(session)
        .environmentObject(navigator)
        .environmentObject(toast)
        .environmentObject(ActivityStore(session: session))
    }

    /// 页面自己创建模型时用 `.standard`，避免本机残留的偏好与稍后提醒影响快照。
    private func clearStandardDefaults(userId: Int64) {
        if let key = GameTypePreference.storageKey(userId: userId) {
            UserDefaults.standard.removeObject(forKey: key)
        }
        for status in ["not_started", "rejected"] {
            UserDefaults.standard.removeObject(
                forKey: FavoriteVenueReward.floatSnoozeKey(userId: userId, status: status)
            )
        }
    }
}
