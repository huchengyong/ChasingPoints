import SwiftUI
import XCTest
@testable import iOS

@MainActor
final class SeasonSnapshotTests: XCTestCase {
    private let pageSize = CGSize(width: 393, height: 852)

    func testActiveSeasonSnapshot() async throws {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        server.stub("GET", "/api/season/overview", .json(SeasonFixtures.overview(
            leaderboard: [
                SeasonFixtures.leaderboardItem(rank: 1, userId: 7, nickname: "榜首"),
                SeasonFixtures.leaderboardItem(rank: 0, userId: 8, nickname: "高分未达标", qualified: false),
            ]
        )))
        let model = SeasonModel(session: session)
        await model.loadIfNeeded()
        XCTAssertTrue(model.isActiveSeason)

        for scheme in [ColorScheme.light, .dark] {
            let image = try renderSnapshot("season-active", colorScheme: scheme, size: pageSize) {
                page(session) { SeasonView(session: session, model: model) }
            }
            XCTAssertEqual(image.size, pageSize)
        }
    }

    func testOffseasonSnapshot() async throws {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        server.stub("GET", "/api/season/overview", .json(SeasonFixtures.overview(
            seasonState: "not_started",
            season: nil,
            record: nil,
            myRank: 0,
            leaderboard: []
        )))
        let model = SeasonModel(session: session)
        await model.loadIfNeeded()

        for scheme in [ColorScheme.light, .dark] {
            let image = try renderSnapshot("season-offseason", colorScheme: scheme, size: pageSize) {
                page(session) { SeasonView(session: session, model: model) }
            }
            XCTAssertEqual(image.size, pageSize)
        }
    }

    func testSeasonErrorSnapshot() async throws {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        server.stub("GET", "/api/season/overview", .raw("service down", status: 503))
        let model = SeasonModel(session: session)
        await model.loadIfNeeded()
        XCTAssertFalse(model.isActiveSeason)

        for scheme in [ColorScheme.light, .dark] {
            let image = try renderSnapshot("season-error", colorScheme: scheme, size: pageSize) {
                page(session) { SeasonView(session: session, model: model) }
            }
            XCTAssertEqual(image.size, pageSize)
        }
    }

    func testSeasonReportSnapshot() async throws {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        server.stub("GET", "/api/season/report", .json(SeasonFixtures.report(
            topAchievements: [SeasonFixtures.achievement]
        )))
        let model = SeasonReportModel(session: session, seasonId: 2, gameType: 3)
        await model.loadIfNeeded()
        XCTAssertEqual(model.phase, .ready)

        for scheme in [ColorScheme.light, .dark] {
            let image = try renderSnapshot("season-report", colorScheme: scheme, size: pageSize) {
                page(session) { SeasonReportView(session: session, seasonId: 2, gameType: 3, model: model) }
            }
            XCTAssertEqual(image.size, pageSize)
        }
    }

    func testSeasonHistorySnapshot() async throws {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        server.stub("GET", "/api/season/history", .json(SeasonFixtures.history(
            total: 2,
            list: [
                SeasonFixtures.historyItem(),
                SeasonFixtures.historyItem(
                    season: SeasonFixtures.season(id: 1, name: "S1", status: 2),
                    record: nil
                ),
            ]
        )))
        let model = SeasonHistoryModel(session: session, gameType: 3)
        await model.loadIfNeeded()
        XCTAssertEqual(model.items.count, 2)

        for scheme in [ColorScheme.light, .dark] {
            let image = try renderSnapshot("season-history", colorScheme: scheme, size: pageSize) {
                page(session) { SeasonHistoryView(session: session, gameType: 3, model: model) }
            }
            XCTAssertEqual(image.size, pageSize)
        }
    }

    func testEmptySeasonHistorySnapshot() async throws {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        server.stub("GET", "/api/season/history", .json(SeasonFixtures.history(total: 0, list: [])))
        let model = SeasonHistoryModel(session: session, gameType: 3)
        await model.loadIfNeeded()
        XCTAssertEqual(model.phase, .empty)

        for scheme in [ColorScheme.light, .dark] {
            let image = try renderSnapshot("season-history-empty", colorScheme: scheme, size: pageSize) {
                page(session) { SeasonHistoryView(session: session, gameType: 3, model: model) }
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
        .environmentObject(ActivityStore(session: session))
    }
}
