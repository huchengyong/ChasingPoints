import SwiftUI
import XCTest
@testable import iOS

/// 选择球友列表的有数据 / 空 / 失败三态快照。
@MainActor
final class OpponentSelectorSnapshotTests: XCTestCase {
    private let size = CGSize(width: 393, height: 852)

    func testLoadedSnapshot() async throws {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        server.stub("GET", "/api/opponent/candidates", .json(MatchStartFixtures.opponentCandidatesJSON()))
        let model = OpponentSelectorModel(session: session, api: MatchStartAPI(session: session))
        await model.loadIfNeeded()

        try await render("opponent-selector", session: session, view: OpponentSelectorView(session: session, model: model))
    }

    func testEmptySnapshot() async throws {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        server.stub("GET", "/api/opponent/candidates", .json(["success": true, "list": []]))
        let model = OpponentSelectorModel(session: session, api: MatchStartAPI(session: session))
        await model.loadIfNeeded()

        try await render("opponent-selector-empty", session: session, view: OpponentSelectorView(session: session, model: model))
    }

    func testFailedSnapshot() async throws {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        server.stub("GET", "/api/opponent/candidates", .failure(.notConnectedToInternet))
        let model = OpponentSelectorModel(session: session, api: MatchStartAPI(session: session))
        await model.loadIfNeeded()

        try await render("opponent-selector-failed", session: session, view: OpponentSelectorView(session: session, model: model))
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
