import SwiftUI
import XCTest
@testable import iOS

@MainActor
final class MatchHistorySnapshotTests: XCTestCase {
    private let pageSize = CGSize(width: 393, height: 852)

    func testRefereeHistoryWithOngoingMatchSnapshot() async throws {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        server.stub("GET", "/api/match/list", .json(MatchHistoryFixtures.participantPage(
            items: [MatchHistoryFixtures.participantItem(id: 101)],
            total: 1
        )))
        server.stub("GET", "/api/match/referee/history", .json(MatchHistoryFixtures.refereePage(
            items: [MatchHistoryFixtures.refereeItem(id: 201)],
            total: 1
        )))
        let ongoing = OngoingRefereeMatch(ref: RefereeOngoingMatch(
            id: 301,
            viewerRole: "referee",
            gameType: 3,
            player1Id: 11,
            player1Name: "选手甲",
            player1Avatar: "",
            player2Id: 12,
            player2Name: "选手乙",
            player2Avatar: "",
            myScore: 2,
            opponentScore: 1
        ))
        XCTAssertNotNil(ongoing)

        for scheme in [ColorScheme.light, .dark] {
            let image = try renderSnapshot("match-history-with-referee-banner", colorScheme: scheme, size: pageSize) {
                page(session) {
                    MatchHistoryView(session: session, section: .referee, ongoingMatch: ongoing)
                }
            }
            XCTAssertEqual(image.size, pageSize)
        }
    }

    func testRefereeHistoryEmptySnapshot() async throws {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        server.stub("GET", "/api/match/referee/history", .json(MatchHistoryFixtures.refereePage(items: [], total: 0)))

        for scheme in [ColorScheme.light, .dark] {
            let image = try renderSnapshot("match-history-referee-empty", colorScheme: scheme, size: pageSize) {
                page(session) {
                    MatchHistoryView(session: session, section: .referee)
                }
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
        }
        .environmentObject(session)
        .environmentObject(AppNavigator())
    }
}
