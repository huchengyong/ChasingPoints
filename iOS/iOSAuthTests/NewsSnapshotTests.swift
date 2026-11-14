import SwiftUI
import XCTest
@testable import iOS

/// 赛讯 Tab 与赛事详情的快照：有数据、空、失败；详情覆盖进行中与已结束。
@MainActor
final class NewsSnapshotTests: XCTestCase {
    private let size = CGSize(width: 393, height: 852)
    private let now = EventNewsFixtures.time("2026-04-01T21:00:00+08:00")
    private let listPath = "/api/event-news/list"
    private let viewPath = "/api/event-news/view"

    // MARK: - 赛讯 Tab

    func testNewsHomeSnapshot() async throws {
        let (session, model) = await makeHomeModel { server in
            server.stub("GET", self.listPath, .json(EventNewsFixtures.listResponse(
                total: 4,
                list: [
                    EventNewsFixtures.eventNewsJSON(
                        id: 1,
                        tournamentName: "Sportsbet.io Tour Championship 2026",
                        status: 1,
                        startDate: "2026-03-30",
                        endDate: "2026-04-05",
                        city: "曼彻斯特",
                        venue: "Manchester Central",
                        currentRoundText: "Semi Finals",
                        latestResultText: "赵心童 4 - 2 特鲁姆普",
                        matchCount: 11
                    ),
                    EventNewsFixtures.eventNewsJSON(
                        id: 2,
                        tournamentName: "中国公开赛",
                        status: 0,
                        startDate: "2026-04-18",
                        endDate: "2026-04-25",
                        city: "深圳",
                        matchCount: 0
                    ),
                    EventNewsFixtures.eventNewsJSON(
                        id: 3,
                        tournamentName: "球员锦标赛",
                        status: 2,
                        startDate: "2026-02-10",
                        endDate: "2026-02-16",
                        city: "伦敦",
                        currentRoundText: "Final",
                        latestResultText: "奥沙利文 10 - 7 塞尔比",
                        matchCount: 15
                    ),
                    EventNewsFixtures.eventNewsJSON(
                        id: 4,
                        tournamentName: "冠军联赛",
                        status: 2,
                        startDate: "2026-02-02",
                        endDate: "2026-02-04",
                        currentRoundText: "Final",
                        latestResultText: "希金斯 3 - 1 罗伯逊",
                        matchCount: 8
                    ),
                ]
            )))
        }
        XCTAssertEqual(model.phase, .loaded)

        try render(session: session, name: "news-home") {
            NewsHomeView(session: session, model: model)
        }
    }

    func testNewsHomeEmptySnapshot() async throws {
        let (session, model) = await makeHomeModel { server in
            server.stub("GET", self.listPath, .json(EventNewsFixtures.listResponse(total: 0, list: [])))
        }
        XCTAssertEqual(model.phase, .empty)

        try render(session: session, name: "news-home-empty") {
            NewsHomeView(session: session, model: model)
        }
    }

    func testNewsHomeErrorSnapshot() async throws {
        let (session, model) = await makeHomeModel { server in
            server.stub("GET", self.listPath, .raw("service down", status: 503))
        }
        guard case .failed = model.phase else {
            return XCTFail("应进入失败态")
        }

        try render(session: session, name: "news-home-error") {
            NewsHomeView(session: session, model: model)
        }
    }

    // MARK: - 赛事详情

    func testNewsDetailLiveSnapshot() async throws {
        let (session, model) = await makeDetailModel { server in
            server.stub("GET", self.viewPath, .json(EventNewsFixtures.viewResponse(
                eventNews: EventNewsFixtures.eventNewsJSON(
                    tournamentName: "Sportsbet.io Tour Championship 2026",
                    status: 1,
                    startDate: "2026-03-30",
                    endDate: "2026-04-05",
                    city: "曼彻斯特",
                    currentRoundText: "Semi Finals",
                    matchCount: 3
                ),
                tournament: EventNewsFixtures.tournamentJSON(id: 3, name: "斯诺克巡回锦标赛 2026"),
                matches: [
                    EventNewsFixtures.matchJSON(
                        id: 1,
                        roundName: "Semi Finals",
                        roundOrder: 60,
                        status: 1,
                        startTime: "2026-04-01 14:00:00",
                        bestOf: 11,
                        homePlayerName: "Zhao Xintong",
                        awayPlayerName: "Judd Trump",
                        homeScore: 4,
                        awayScore: 3
                    ),
                    EventNewsFixtures.matchJSON(
                        id: 2,
                        roundName: "Semi Finals",
                        roundOrder: 60,
                        status: 2,
                        startTime: "2026-04-01 19:30:00",
                        bestOf: 11,
                        homePlayerName: "Ronnie O'Sullivan",
                        awayPlayerName: "Mark Selby",
                        homeScore: 6,
                        awayScore: 4,
                        winnerSide: 1
                    ),
                    EventNewsFixtures.matchJSON(
                        id: 3,
                        roundName: "Final",
                        roundOrder: 70,
                        status: 0,
                        startTime: "2026-04-02 20:00:00",
                        bestOf: 19,
                        homePlayerName: "Zhao Xintong",
                        awayPlayerName: "Ronnie O'Sullivan"
                    ),
                ]
            )))
        }
        XCTAssertTrue(model.isLive)

        try render(session: session, name: "news-detail-live") {
            NewsDetailView(session: session, eventId: 18, model: model)
        }
    }

    func testNewsDetailFinishedSnapshot() async throws {
        let (session, model) = await makeDetailModel { server in
            server.stub("GET", self.viewPath, .json(EventNewsFixtures.viewResponse(
                eventNews: EventNewsFixtures.eventNewsJSON(
                    tournamentName: "球员锦标赛",
                    status: 2,
                    startDate: "2026-02-10",
                    endDate: "2026-02-16",
                    city: "伦敦",
                    currentRoundText: "Final",
                    matchCount: 2
                ),
                matches: [
                    EventNewsFixtures.matchJSON(
                        id: 1,
                        roundName: "Semi Finals",
                        roundOrder: 60,
                        status: 2,
                        startTime: "2026-02-15 14:00:00",
                        bestOf: 11,
                        homePlayerName: "Ronnie O'Sullivan",
                        awayPlayerName: "Judd Trump",
                        homeScore: 6,
                        awayScore: 3,
                        winnerSide: 1
                    ),
                    EventNewsFixtures.matchJSON(
                        id: 2,
                        roundName: "Final",
                        roundOrder: 70,
                        status: 2,
                        startTime: "2026-02-16 20:00:00",
                        bestOf: 19,
                        homePlayerName: "Ronnie O'Sullivan",
                        awayPlayerName: "Mark Selby",
                        homeScore: 10,
                        awayScore: 7,
                        winnerSide: 1
                    ),
                ]
            )))
        }
        XCTAssertNotNil(model.detail?.champion)

        try render(session: session, name: "news-detail-finished") {
            NewsDetailView(session: session, eventId: 18, model: model)
        }
    }

    func testNewsDetailErrorSnapshot() async throws {
        let (session, model) = await makeDetailModel { server in
            server.stub("GET", self.viewPath, .raw("service down", status: 503))
        }
        guard case .failed = model.phase else {
            return XCTFail("应进入失败态")
        }

        try render(session: session, name: "news-detail-error") {
            NewsDetailView(session: session, eventId: 18, model: model)
        }
    }

    // MARK: - 夹具

    private func makeHomeModel(
        stub: (MockHTTPServer) -> Void
    ) async -> (SessionStore, SaiXunListModel) {
        let server = MockHTTPServer()
        let session = makeSessionStore(server, credentialStore: InMemoryCredentialStore(credentials: nil))
        stub(server)
        let model = SaiXunListModel(session: session, now: { self.now })
        await model.loadIfNeeded()
        return (session, model)
    }

    private func makeDetailModel(
        stub: (MockHTTPServer) -> Void
    ) async -> (SessionStore, SaiXunDetailModel) {
        let server = MockHTTPServer()
        let session = makeSessionStore(server, credentialStore: InMemoryCredentialStore(credentials: nil))
        stub(server)
        // 定时重算的睡眠必须真的挂起：不挂起的空实现会让页面里的 runAutoRefresh 在主线程上空转。
        let model = SaiXunDetailModel(session: session, eventId: 18, now: { self.now }, sleep: { _ in
            try await Task.sleep(for: .seconds(3600))
        })
        await model.loadIfNeeded()
        return (session, model)
    }

    private func render<Content: View>(
        session: SessionStore,
        name: String,
        @ViewBuilder content: () -> Content
    ) throws {
        let navigator = AppNavigator()
        let toast = ToastCenter()

        for scheme in [ColorScheme.light, .dark] {
            let image = try renderSnapshot(name, colorScheme: scheme, size: size) {
                NavigationStack {
                    content()
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
