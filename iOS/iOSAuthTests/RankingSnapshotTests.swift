import SwiftUI
import XCTest
@testable import iOS

@MainActor
final class RankingSnapshotTests: XCTestCase {
    private let size = CGSize(width: 393, height: 852)
    private let leaderboardPath = "/api/public/rank/leaderboard"

    func testFullRankingSnapshot() async throws {
        let server = MockHTTPServer()
        let session = await RankingFixtures.makeSignedInSession(server)
        server.stub("GET", leaderboardPath, RankingFixtures.leaderboard(
            topThree: RankingFixtures.listItems(1...3),
            list: RankingFixtures.listItems(4...12),
            total: 12,
            myRanking: RankingFixtures.itemJSON(rank: 8, userId: 1, nickname: "测试用户")
        ))
        let model = RankingModel(session: session)
        await model.loadIfNeeded()
        XCTAssertEqual(model.phase, .loaded)

        for scheme in [ColorScheme.light, .dark] {
            let image = try renderSnapshot("ranking-full", colorScheme: scheme, size: size) {
                rankingContent(session: session, model: model)
            }
            XCTAssertEqual(image.size, size)
        }
    }

    func testPartialGuestRankingSnapshot() async throws {
        let server = MockHTTPServer()
        let session = RankingFixtures.makeGuestSession(server)
        server.stub("GET", leaderboardPath, RankingFixtures.leaderboard(
            topThree: [RankingFixtures.itemJSON(rank: 1, userId: 201, nickname: "第一名")],
            total: 1
        ))
        let model = RankingModel(session: session)
        await model.loadIfNeeded()
        XCTAssertEqual(model.phase, .loaded)

        for scheme in [ColorScheme.light, .dark] {
            let image = try renderSnapshot("ranking-guest-partial", colorScheme: scheme, size: size) {
                rankingContent(session: session, model: model)
            }
            XCTAssertEqual(image.size, size)
        }
    }

    func testEmptyRankingSnapshot() async throws {
        let server = MockHTTPServer()
        let session = RankingFixtures.makeGuestSession(server)
        server.stub("GET", leaderboardPath, RankingFixtures.leaderboard(total: 0))
        let model = RankingModel(session: session)
        await model.loadIfNeeded()
        XCTAssertEqual(model.phase, .empty)

        for scheme in [ColorScheme.light, .dark] {
            let image = try renderSnapshot("ranking-empty", colorScheme: scheme, size: size) {
                rankingContent(session: session, model: model)
            }
            XCTAssertEqual(image.size, size)
        }
    }

    func testGoldRankExplainSnapshot() async throws {
        let server = MockHTTPServer()
        let session = await RankingFixtures.makeSignedInSession(server)
        stubRankExplain(server)
        let model = RankExplainModel(session: session, gameType: 3)
        await model.loadIfNeeded()
        XCTAssertEqual(model.heroModel?.name, "黄金球手")

        for scheme in [ColorScheme.light, .dark] {
            let image = try renderSnapshot("rank-explain-gold", colorScheme: scheme, size: size) {
                rankExplainContent(session: session, model: model)
            }
            XCTAssertEqual(image.size, size)
        }
    }

    func testKingRankExplainSnapshot() async throws {
        let server = MockHTTPServer()
        let session = await RankingFixtures.makeSignedInSession(server)
        stubRankExplain(server, rankInfo: RankingFixtures.kingRankInfo())
        let model = RankExplainModel(session: session, gameType: 3)
        await model.loadIfNeeded()
        XCTAssertEqual(model.heroModel?.isMax, true)

        for scheme in [ColorScheme.light, .dark] {
            let image = try renderSnapshot("rank-explain-king", colorScheme: scheme, size: size) {
                rankExplainContent(session: session, model: model)
            }
            XCTAssertEqual(image.size, size)
        }
    }

    func testRankExplainErrorSnapshot() async throws {
        let server = MockHTTPServer()
        let session = await RankingFixtures.makeSignedInSession(server)
        server.stub("GET", "/api/rank/infos", .raw("service down", status: 500))
        server.stub("GET", "/api/public/rank/configs", RankingFixtures.configs(rulesVersion: 1))
        let model = RankExplainModel(session: session, gameType: 3)
        await model.loadIfNeeded()
        XCTAssertTrue(model.isInitialError)

        for scheme in [ColorScheme.light, .dark] {
            let image = try renderSnapshot("rank-explain-error", colorScheme: scheme, size: size) {
                rankExplainContent(session: session, model: model)
            }
            XCTAssertEqual(image.size, size)
        }
    }

    // MARK: - 辅助

    private func stubRankExplain(
        _ server: MockHTTPServer,
        rankInfo: RankInfo = RankingFixtures.goldRankInfo(placementRemaining: 4)
    ) {
        server.stub("GET", "/api/rank/infos", RankingFixtures.rankInfos([
            (gameType: 3, info: RankingFixtures.rankInfoJSON(
                level: rankInfo.level,
                name: rankInfo.name,
                icon: rankInfo.icon,
                rankScore: rankInfo.rankScore,
                nextLevel: rankInfo.nextLevel,
                nextName: rankInfo.nextName,
                nextScore: rankInfo.nextScore,
                progress: rankInfo.progress,
                placementRemaining: rankInfo.placementRemaining
            )),
        ]))
        server.stub("GET", "/api/public/rank/configs", RankingFixtures.configs(
            rulesVersion: 2,
            rulesEffectiveAt: "2026-11-01T00:00:00+08:00"
        ))
    }

    private func rankingContent(session: SessionStore, model: RankingModel) -> some View {
        let navigator = AppNavigator()
        let toast = ToastCenter()
        return NavigationStack {
            RankingView(session: session, model: model)
                .navigationDestination(for: AppRoute.self) { AppRouteDestination(route: $0) }
        }
        .environmentObject(session)
        .environmentObject(navigator)
        .environmentObject(toast)
    }

    private func rankExplainContent(session: SessionStore, model: RankExplainModel) -> some View {
        let navigator = AppNavigator()
        let toast = ToastCenter()
        return NavigationStack {
            RankExplainView(session: session, gameType: 3, model: model)
                .navigationDestination(for: AppRoute.self) { AppRouteDestination(route: $0) }
        }
        .environmentObject(session)
        .environmentObject(navigator)
        .environmentObject(toast)
    }
}
