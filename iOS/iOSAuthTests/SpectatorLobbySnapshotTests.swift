import SwiftUI
import XCTest
@testable import iOS

@MainActor
final class SpectatorLobbySnapshotTests: XCTestCase {
    private let size = CGSize(width: 393, height: 852)

    func testLobbyWithLiveAndFinishedMatchesSnapshot() async throws {
        let server = MockHTTPServer()
        let session = await SpectatorLobbyFixtures.makeSignedInSession(server)
        server.stub("GET", "/api/public/matches", SpectatorLobbyFixtures.listResponse([
            SpectatorLobbyFixtures.itemJSON(
                id: 1, status: 1, gameType: 3, gameTypeName: "中式八球", matchMode: "ranked",
                player1Id: 1, player1Name: "测试用户", player1Score: 4,
                player2Id: 52, player2Name: "球友 B", player2Score: 2, durationSeconds: 600
            ),
            SpectatorLobbyFixtures.itemJSON(
                id: 2, status: 1, gameType: 2, gameTypeName: "九球追分", matchMode: "practice",
                player1Id: 60, player1Name: "阿杰", player1Score: 30,
                player2Id: 61, player2Name: "小林", player2Score: 24, durationSeconds: 1800
            ),
            SpectatorLobbyFixtures.itemJSON(
                id: 3, status: 1, gameType: 1, gameTypeName: "斯诺克", matchMode: "practice",
                player1Id: 62, player1Name: "老陈", player1Score: 1,
                player2Id: 1, player2Name: "测试用户", player2Score: 3, durationSeconds: 900
            ),
            SpectatorLobbyFixtures.itemJSON(
                id: 4, status: 2, gameType: 3, gameTypeName: "中式八球", matchMode: "ranked",
                player1Id: 63, player1Name: "大刘", player1Score: 5,
                player2Id: 64, player2Name: "小赵", player2Score: 3
            ),
            SpectatorLobbyFixtures.itemJSON(
                id: 5, status: 2, gameType: 4, gameTypeName: "美式九球", matchMode: "practice",
                player1Id: 1, player1Name: "测试用户", player1Score: 2,
                player2Id: 65, player2Name: "阿凯", player2Score: 2
            ),
        ]))
        let model = SpectatorLobbyModel(session: session)
        await model.loadIfNeeded()
        XCTAssertEqual(model.phase, .loaded)
        XCTAssertEqual(model.sections.map(\.title), ["进行中", "已结束"])

        for scheme in [ColorScheme.light, .dark] {
            let image = try renderSnapshot("spectator-lobby", colorScheme: scheme, size: size) {
                snapshotContent(session: session, model: model)
            }
            XCTAssertEqual(image.size, size)
        }
    }

    func testLobbyEmptySnapshot() async throws {
        let server = MockHTTPServer()
        let session = SpectatorLobbyFixtures.makeGuestSession(server)
        server.stub("GET", "/api/public/matches", SpectatorLobbyFixtures.listResponse([]))
        let model = SpectatorLobbyModel(session: session)
        await model.loadIfNeeded()
        XCTAssertEqual(model.phase, .empty)

        for scheme in [ColorScheme.light, .dark] {
            let image = try renderSnapshot("spectator-lobby-empty", colorScheme: scheme, size: size) {
                snapshotContent(session: session, model: model)
            }
            XCTAssertEqual(image.size, size)
        }
    }

    func testLobbyErrorSnapshot() async throws {
        let server = MockHTTPServer()
        let session = SpectatorLobbyFixtures.makeGuestSession(server)
        server.stub("GET", "/api/public/matches", .raw("service down", status: 503))
        let model = SpectatorLobbyModel(session: session)
        await model.loadIfNeeded()
        guard case .failed = model.phase else { return XCTFail("应进入失败态") }

        for scheme in [ColorScheme.light, .dark] {
            let image = try renderSnapshot("spectator-lobby-error", colorScheme: scheme, size: size) {
                snapshotContent(session: session, model: model)
            }
            XCTAssertEqual(image.size, size)
        }
    }

    private func snapshotContent(session: SessionStore, model: SpectatorLobbyModel) -> some View {
        let navigator = AppNavigator()
        let toast = ToastCenter()
        return NavigationStack {
            SpectatorLobbyView(session: session, model: model)
                .navigationDestination(for: AppRoute.self) { AppRouteDestination(route: $0) }
        }
        .environmentObject(session)
        .environmentObject(navigator)
        .environmentObject(toast)
    }
}
