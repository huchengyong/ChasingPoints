import SwiftUI
import XCTest
@testable import iOS

/// 扫码确认页三态的亮 / 暗快照；只断言能渲染，不做像素比对。
@MainActor
final class ScanConfirmSnapshotTests: XCTestCase {
    private let size = CGSize(width: 393, height: 852)
    private var suiteName = ""

    override func setUp() {
        super.setUp()
        suiteName = "ScanConfirmSnapshotTests.\(UUID().uuidString)"
        UserDefaults(suiteName: suiteName)?.removePersistentDomain(forName: suiteName)
    }

    override func tearDown() {
        UserDefaults(suiteName: suiteName)?.removePersistentDomain(forName: suiteName)
        super.tearDown()
    }

    private func makePreferences() -> GameTypePreference {
        GameTypePreference(defaults: UserDefaults(suiteName: suiteName)!)
    }

    func testStartFormSnapshot() async throws {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        let model = ScanConfirmModel(
            session: session,
            handoff: MatchStartFixtures.startHandoff(identity: session.identity!),
            preferences: makePreferences()
        )
        model.selectGameType(3)

        try await render("scan-confirm-start", session: session, view: ScanConfirmView(session: session, model: model))
    }

    func testRefereeFormSnapshot() async throws {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        server.stub("POST", "/api/match/referee/preview", .json([
            "success": true,
            "preview": MatchStartFixtures.refereePreviewJSON,
        ]))
        let model = ScanConfirmModel(
            session: session,
            handoff: MatchStartFixtures.refereeHandoff(identity: session.identity!),
            preferences: makePreferences()
        )
        await model.loadIfNeeded()

        try await render("scan-confirm-referee", session: session, view: ScanConfirmView(session: session, model: model))
    }

    func testMissingHandoffSnapshot() async throws {
        let session = await makeRestoredAccountSession(MockHTTPServer())
        let model = ScanConfirmModel(
            session: session,
            handoff: nil,
            preferences: makePreferences()
        )

        try await render(
            "scan-confirm-missing-handoff",
            session: session,
            view: ScanConfirmView(session: session, model: model)
        )
    }

    private func render<Content: View>(
        _ name: String,
        session: SessionStore,
        view: Content
    ) async throws {
        let navigator = AppNavigator()
        let toast = ToastCenter()
        for scheme in [ColorScheme.light, .dark] {
            let image = try renderSnapshot(name, colorScheme: scheme, size: size) {
                NavigationStack {
                    view
                        .navigationDestination(for: AppRoute.self) { AppRouteDestination(route: $0) }
                }
                .environmentObject(session)
                .environmentObject(navigator)
                .environmentObject(toast)
            }
            XCTAssertEqual(image.size, size)
        }
    }
}
