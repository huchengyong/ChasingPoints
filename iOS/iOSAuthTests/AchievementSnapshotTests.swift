import SwiftUI
import XCTest
@testable import iOS

@MainActor
final class AchievementSnapshotTests: XCTestCase {
    private let pageSize = CGSize(width: 393, height: 852)

    func testHonorWallSnapshot() async throws {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        stubNotifications(server)
        server.stub("GET", "/api/achievement/honor-wall", .json(AchievementFixtures.honorWall()))
        let model = HonorWallModel(session: session, gameType: 3)
        await model.loadIfNeeded()
        XCTAssertTrue(model.hasLoaded)

        for scheme in [ColorScheme.light, .dark] {
            let image = try renderSnapshot("honor-wall", colorScheme: scheme, size: pageSize) {
                page(session) { HonorWallView(session: session, gameType: 3, model: model) }
            }
            XCTAssertEqual(image.size, pageSize)
        }
    }

    func testHonorWallEmptySnapshot() async throws {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        stubNotifications(server)
        server.stub("GET", "/api/achievement/honor-wall", .json(AchievementFixtures.emptyHonorWall()))
        let model = HonorWallModel(session: session, gameType: 3)
        await model.loadIfNeeded()
        XCTAssertTrue(model.recentHonors.isEmpty)

        for scheme in [ColorScheme.light, .dark] {
            let image = try renderSnapshot("honor-wall-empty", colorScheme: scheme, size: pageSize) {
                page(session) { HonorWallView(session: session, gameType: 3, model: model) }
            }
            XCTAssertEqual(image.size, pageSize)
        }
    }

    func testHonorWallErrorSnapshot() async throws {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        server.stub("GET", "/api/achievement/honor-wall", .raw("service down", status: 503))
        let model = HonorWallModel(session: session, gameType: 3)
        await model.loadIfNeeded()
        XCTAssertTrue(model.isInitialFailure)

        for scheme in [ColorScheme.light, .dark] {
            let image = try renderSnapshot("honor-wall-error", colorScheme: scheme, size: pageSize) {
                page(session) { HonorWallView(session: session, gameType: 3, model: model) }
            }
            XCTAssertEqual(image.size, pageSize)
        }
    }

    func testAchievementDetailUnlockedSnapshot() async throws {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        server.stub("GET", "/api/achievement/detail", .json(AchievementFixtures.unlockedDetail()))
        let model = AchievementDetailModel(session: session, achievementId: 7)
        await model.loadIfNeeded()
        XCTAssertEqual(model.progressPercent, 100)

        for scheme in [ColorScheme.light, .dark] {
            let image = try renderSnapshot("achievement-detail-unlocked", colorScheme: scheme, size: pageSize) {
                page(session) { AchievementDetailView(session: session, achievementId: 7, model: model) }
            }
            XCTAssertEqual(image.size, pageSize)
        }
    }

    func testAchievementDetailLockedSnapshot() async throws {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        server.stub("GET", "/api/achievement/detail", .json(AchievementFixtures.lockedDetail()))
        let model = AchievementDetailModel(session: session, achievementId: 8)
        await model.loadIfNeeded()
        XCTAssertEqual(model.progressPercent, 33)

        for scheme in [ColorScheme.light, .dark] {
            let image = try renderSnapshot("achievement-detail-locked", colorScheme: scheme, size: pageSize) {
                page(session) { AchievementDetailView(session: session, achievementId: 8, model: model) }
            }
            XCTAssertEqual(image.size, pageSize)
        }
    }

    private func stubNotifications(_ server: MockHTTPServer) {
        server.stub("GET", "/api/notification/list", .json(AchievementFixtures.notificationList([])))
        server.stub("POST", "/api/notification/read", .json(AchievementFixtures.common()))
    }

    private func page<Content: View>(
        _ session: SessionStore,
        @ViewBuilder content: () -> Content
    ) -> some View {
        NavigationStack {
            content()
                .navigationDestination(for: AppRoute.self) { AppRouteDestination(route: $0) }
        }
        .environmentObject(session)
        .environmentObject(AppNavigator())
        .environmentObject(ToastCenter())
        .environmentObject(ActivityStore(session: session))
    }
}
