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

    func testReputationEmptySnapshot() async throws {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        server.stub("GET", "/api/user/overview", .json(ProfileFixtures.overview()))
        server.stub("GET", "/api/user/reputation/logs", .json(SubPageFixtures.reputationLogs(items: [], total: 0)))
        let model = ReputationModel(session: session)
        await model.onAppear()
        XCTAssertTrue(model.logs.items.isEmpty)

        for scheme in [ColorScheme.light, .dark] {
            let image = try renderSnapshot("subpage-reputation-empty", colorScheme: scheme, size: pageSize) {
                page(session) { ReputationView(session: session, model: model) }
            }
            XCTAssertEqual(image.size, pageSize)
        }
    }

    func testOpponentRecordEmptySnapshot() async throws {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        server.stub("GET", "/api/opponent/list", .json(SubPageFixtures.opponentList(
            items: [],
            totalOpponents: 0,
            totalWins: 0
        )))
        let model = OpponentRecordModel(session: session)
        await model.onAppear()
        XCTAssertTrue(model.list.items.isEmpty)

        for scheme in [ColorScheme.light, .dark] {
            let image = try renderSnapshot("subpage-opponents-empty", colorScheme: scheme, size: pageSize) {
                page(session) { OpponentRecordView(session: session, model: model) }
            }
            XCTAssertEqual(image.size, pageSize)
        }
    }

    func testH2HRecordErrorSnapshot() async throws {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        server.stub("GET", "/api/match/h2h/overview", .raw("service down", status: 503))
        let model = H2HRecordModel(
            session: session,
            opponentId: 18,
            opponentName: "阿杰",
            monthKey: "2026-10"
        )
        await model.reload()
        XCTAssertNotNil(model.failure)

        for scheme in [ColorScheme.light, .dark] {
            let image = try renderSnapshot("subpage-h2h-error", colorScheme: scheme, size: pageSize) {
                page(session) { H2HRecordView(session: session, model: model) }
            }
            XCTAssertEqual(image.size, pageSize)
        }
    }

    func testStatsDetailSnapshot() async throws {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        server.stub("GET", "/api/stats/overview", .json(StatsFixtures.overview()))
        let model = StatsDetailModel(session: session, gameType: 3, now: { self.april })
        await model.loadIfNeeded()
        XCTAssertTrue(model.hasLoadedOnce)

        for scheme in [ColorScheme.light, .dark] {
            let image = try renderSnapshot("stats-detail", colorScheme: scheme, size: pageSize) {
                page(session) { StatsDetailView(session: session, gameType: 3, model: model) }
            }
            XCTAssertEqual(image.size, pageSize)
        }
    }

    func testStatsDetailEmptySnapshot() async throws {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        server.stub("GET", "/api/stats/overview", .json(StatsFixtures.overview(
            byGameType: [],
            recentTrend: [],
            rankScoreTrend: [],
            singleHighScores: [],
            duration: nil,
            opponentStrength: []
        )))
        let model = StatsDetailModel(session: session, gameType: 3, now: { self.april })
        await model.loadIfNeeded()
        XCTAssertTrue(model.hasLoadedOnce)

        for scheme in [ColorScheme.light, .dark] {
            let image = try renderSnapshot("stats-detail-empty", colorScheme: scheme, size: pageSize) {
                page(session) { StatsDetailView(session: session, gameType: 3, model: model) }
            }
            XCTAssertEqual(image.size, pageSize)
        }
    }

    func testStatsDetailErrorSnapshot() async throws {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        server.stub("GET", "/api/stats/overview", .raw("service down", status: 503))
        let model = StatsDetailModel(session: session, gameType: 3, now: { self.april })
        await model.loadIfNeeded()
        XCTAssertTrue(model.isInitialError)

        for scheme in [ColorScheme.light, .dark] {
            let image = try renderSnapshot("stats-detail-error", colorScheme: scheme, size: pageSize) {
                page(session) { StatsDetailView(session: session, gameType: 3, model: model) }
            }
            XCTAssertEqual(image.size, pageSize)
        }
    }

    func testPerformanceMemberSnapshot() async throws {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        server.stub("GET", "/api/member/performance", .json(StatsFixtures.performance(
            summary: StatsFixtures.summary(),
            bestWin: StatsFixtures.matchItem(matchId: 9),
            worstLoss: StatsFixtures.matchItem(matchId: 10, opponentName: "老李", beforeScore: 1300, winRate: 80),
            months: StatsFixtures.defaultMonths()
        )))
        let model = PerformanceModel(session: session, gameType: 3)
        await model.loadIfNeeded()
        XCTAssertNotNil(model.data)

        for scheme in [ColorScheme.light, .dark] {
            let image = try renderSnapshot("performance-member", colorScheme: scheme, size: pageSize) {
                page(session) { PerformanceView(session: session, gameType: 3, model: model) }
            }
            XCTAssertEqual(image.size, pageSize)
        }
    }

    func testPerformanceLockedSnapshot() async throws {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        server.stub("GET", "/api/member/performance", .json(StatsFixtures.performance(
            locked: true,
            sampleCount: 12,
            monthPreview: StatsFixtures.monthPreview(month: "2026-04", sampleCount: 6)
        )))
        let model = PerformanceModel(session: session, gameType: 3)
        await model.loadIfNeeded()
        XCTAssertEqual(model.lockedView?.showPurchase, false)

        for scheme in [ColorScheme.light, .dark] {
            let image = try renderSnapshot("performance-locked", colorScheme: scheme, size: pageSize) {
                page(session) { PerformanceView(session: session, gameType: 3, model: model) }
            }
            XCTAssertEqual(image.size, pageSize)
        }
    }

    private var april: Date {
        AppFormat.parseDateTime("2026-04-15 12:00:00")!
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
