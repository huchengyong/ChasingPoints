import SwiftUI
import XCTest
@testable import iOS

@MainActor
final class SubPageSnapshotTests: XCTestCase {
    private let pageSize = CGSize(width: 393, height: 852)

    func testReputationSnapshot() async throws {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        server.stub("GET", "/api/user/overview", .json(ProfileFixtures.overview(
            reputation: ProfileFixtures.reputation(score: 186, status: "good", statusText: "良好")
        )))
        server.stub("GET", "/api/user/reputation/logs", .json(SubPageFixtures.reputationLogs(
            items: [
                SubPageFixtures.reputationLog(id: 9001, changeScore: -3),
                SubPageFixtures.reputationLog(id: 9002, changeScore: 1, reasonText: "正常完成对局", matchId: nil),
            ],
            total: 2
        )))
        let model = ReputationModel(session: session)
        await model.onAppear()
        XCTAssertEqual(model.logs.items.count, 2)

        for scheme in [ColorScheme.light, .dark] {
            let image = try renderSnapshot("subpage-reputation", colorScheme: scheme, size: pageSize) {
                page(session) { ReputationView(session: session, model: model) }
            }
            XCTAssertEqual(image.size, pageSize)
        }
    }

    func testOpponentRecordSnapshot() async throws {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        server.stub("GET", "/api/opponent/list", .json(SubPageFixtures.opponentList(
            items: [
                SubPageFixtures.opponentItem(id: 18, name: "阿杰"),
                SubPageFixtures.opponentItem(id: 20, name: "阿峰", totalMatches: 6, wins: 4, losses: 0, winRate: 100),
            ],
            totalOpponents: 2,
            totalWins: 11
        )))
        let model = OpponentRecordModel(session: session)
        await model.onAppear()
        XCTAssertEqual(model.list.items.count, 2)

        for scheme in [ColorScheme.light, .dark] {
            let image = try renderSnapshot("subpage-opponents", colorScheme: scheme, size: pageSize) {
                page(session) { OpponentRecordView(session: session, model: model) }
            }
            XCTAssertEqual(image.size, pageSize)
        }
    }

    func testH2HRecordSnapshot() async throws {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        server.stub("GET", "/api/match/h2h/overview", .json(SubPageFixtures.h2hOverview(
            items: [
                MatchHistoryFixtures.participantItem(id: 1, result: 1, myScore: 7, opponentScore: 5),
                MatchHistoryFixtures.participantItem(id: 2, result: 2, myScore: 3, opponentScore: 6),
            ],
            total: 8
        )))
        let model = H2HRecordModel(
            session: session,
            opponentId: 18,
            opponentName: "阿杰",
            monthKey: "2026-10"
        )
        await model.reload()
        XCTAssertEqual(model.history.count, 2)

        for scheme in [ColorScheme.light, .dark] {
            let image = try renderSnapshot("subpage-h2h", colorScheme: scheme, size: pageSize) {
                page(session) { H2HRecordView(session: session, model: model) }
            }
            XCTAssertEqual(image.size, pageSize)
        }

        model.historyViewMode = .list
        for scheme in [ColorScheme.light, .dark] {
            let image = try renderSnapshot("subpage-h2h-list", colorScheme: scheme, size: pageSize) {
                page(session) { H2HRecordView(session: session, model: model) }
            }
            XCTAssertEqual(image.size, pageSize)
        }
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
    }
}
