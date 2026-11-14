import Foundation
import XCTest
@testable import iOS

/// 赛事详情页面模型：赛事数据合并、赛程分栏、分批显示、失败保留数据与进行中定时刷新。
@MainActor
final class SaiXunDetailModelTests: XCTestCase {
    private let viewPath = "/api/event-news/view"
    private let now = EventNewsFixtures.time("2026-04-01T21:00:00+08:00")

    func testLoadMergesTournamentFallbacksAndBuildsSchedule() async {
        let server = MockHTTPServer()
        let session = makeGuestSession(server)
        server.stub("GET", viewPath, .json(EventNewsFixtures.viewResponse(
            eventNews: EventNewsFixtures.eventNewsJSON(
                sourceType: "official",
                tournamentName: "",
                startDate: "",
                endDate: ""
            ),
            tournament: EventNewsFixtures.tournamentJSON(id: 3, name: "赛事实体"),
            matches: [EventNewsFixtures.matchJSON(id: 88, roundName: "Final", roundOrder: 70)]
        )))
        let model = makeModel(session)

        await model.loadIfNeeded()

        XCTAssertEqual(model.phase, .loaded)
        let detail = try? XCTUnwrap(model.detail)
        XCTAssertEqual(detail?.card.title, "赛事实体")
        XCTAssertEqual(detail?.card.coverImage, .remote(SaiXunRules.officialDefaultEventCover))
        XCTAssertEqual(detail?.card.matchCountText, "1 场比赛")
        XCTAssertEqual(detail?.dateText, "3月30日 - 4月5日")
        XCTAssertEqual(detail?.regionText, "中国 · Shenzhen")
        XCTAssertEqual(detail?.showTime, false)
        XCTAssertEqual(detail?.timeText, "")
        XCTAssertEqual(detail?.scheduleTabs.map(\.key), ["ended", "upcoming"])
        XCTAssertEqual(model.activeScheduleTab.key, "upcoming")
    }

    func testChampionComesFromFinishedFinal() async {
        let server = MockHTTPServer()
        let session = makeGuestSession(server)
        server.stub("GET", viewPath, .json(EventNewsFixtures.viewResponse(
            eventNews: EventNewsFixtures.eventNewsJSON(tournamentName: "已结束的赛事", status: 2),
            matches: [
                EventNewsFixtures.matchJSON(
                    id: 1,
                    roundName: "Semi Finals",
                    roundOrder: 60,
                    status: 2,
                    startTime: "2026-03-31 14:00:00",
                    homeScore: 6,
                    awayScore: 4,
                    winnerSide: 1
                ),
                EventNewsFixtures.matchJSON(
                    id: 2,
                    roundName: "Final",
                    roundOrder: 70,
                    status: 2,
                    startTime: "2026-04-01 20:00:00",
                    homeScore: 18,
                    awayScore: 14,
                    winnerSide: 1
                ),
            ]
        )))
        let model = makeModel(session)

        await model.loadIfNeeded()

        let champion = model.detail?.champion
        XCTAssertEqual(champion?.name, "Yuan Sijun")
        XCTAssertEqual(champion?.opponentName, "Wu Yize")
        XCTAssertEqual(champion?.scoreText, "18-14")
    }

    func testScheduleTabSwitchKeepsManualChoiceAndResetsRenderLimit() async {
        let server = MockHTTPServer()
        let session = makeGuestSession(server)
        server.stub("GET", viewPath, .json(EventNewsFixtures.viewResponse(
            eventNews: EventNewsFixtures.eventNewsJSON(tournamentName: "进行中的赛事", status: 1),
            matches: [
                EventNewsFixtures.matchJSON(id: 1, roundName: "Semi Finals", roundOrder: 60, status: 2, startTime: "2026-03-31 14:00:00", homeScore: 6, awayScore: 4, winnerSide: 1),
                EventNewsFixtures.matchJSON(id: 2, roundName: "Final", roundOrder: 70, status: 0, startTime: "2026-04-02 20:00:00"),
            ]
        )))
        let model = makeModel(session, sleep: { _ in })
        await model.loadIfNeeded()

        XCTAssertEqual(model.activeScheduleTab.key, "upcoming")
        model.showMoreScheduleMatches()

        model.selectScheduleTab("ended")

        XCTAssertEqual(model.selectedScheduleTabKey, "ended")
        XCTAssertEqual(model.activeScheduleTab.key, "ended")
        XCTAssertEqual(model.activeScheduleTab.matches.map(\.id), [1])
        XCTAssertEqual(model.scheduleRenderLimit, SaiXunRules.schedulePageSize)
    }

    func testScheduleRendersThirtyMatchesThenLoadsMore() async {
        let server = MockHTTPServer()
        let session = makeGuestSession(server)
        let matches = (100..<131).map {
            EventNewsFixtures.matchJSON(
                id: Int64($0),
                roundName: "Round 1",
                roundOrder: 10,
                startTime: "2026-04-10 10:00:00"
            )
        }
        server.stub("GET", viewPath, .json(EventNewsFixtures.viewResponse(
            eventNews: EventNewsFixtures.eventNewsJSON(tournamentName: "即将开始的赛事", status: 0),
            matches: matches
        )))
        let model = makeModel(session)
        await model.loadIfNeeded()

        XCTAssertEqual(model.visibleDays.flatMap(\.matches).count, 30)
        XCTAssertTrue(model.hasMoreScheduleMatches)

        model.showMoreScheduleMatches()

        XCTAssertEqual(model.visibleDays.flatMap(\.matches).count, 31)
        XCTAssertFalse(model.hasMoreScheduleMatches)
    }

    func testAutoRefreshTicksOnlyForLiveEventAndStopsWhenCancelled() async throws {
        let server = MockHTTPServer()
        let session = makeGuestSession(server)
        stubLiveView(server)
        var clock = now
        let ticks = LockedCounter()
        let model = SaiXunDetailModel(
            session: session,
            eventId: 18,
            now: { clock },
            interval: 30,
            sleep: { _ in
                _ = ticks.next()
                try await Task.sleep(nanoseconds: 1_000_000)
            }
        )
        await model.loadIfNeeded()
        XCTAssertTrue(model.isLive)
        XCTAssertEqual(model.lastRenderedAt, clock)

        let task = Task { await model.runAutoRefresh() }
        let advanced = clock.addingTimeInterval(60)
        clock = advanced
        await waitUntil { model.lastRenderedAt == advanced }

        task.cancel()
        await task.value
        let rendered = model.lastRenderedAt
        clock = advanced.addingTimeInterval(60)
        try await Task.sleep(nanoseconds: 20_000_000)
        XCTAssertEqual(model.lastRenderedAt, rendered, "页面离开后不得继续刷新")
    }

    func testAutoRefreshDoesNotStartForFinishedEvent() async {
        let server = MockHTTPServer()
        let session = makeGuestSession(server)
        server.stub("GET", viewPath, .json(EventNewsFixtures.viewResponse(
            eventNews: EventNewsFixtures.eventNewsJSON(tournamentName: "已结束的赛事", status: 2),
            matches: [EventNewsFixtures.matchJSON()]
        )))
        let ticks = LockedCounter()
        let model = SaiXunDetailModel(
            session: session,
            eventId: 18,
            now: { self.now },
            interval: 30,
            sleep: { _ in
                _ = ticks.next()
                throw CancellationError()
            }
        )
        await model.loadIfNeeded()
        let rendered = model.lastRenderedAt

        await model.runAutoRefresh()

        XCTAssertEqual(ticks.current, 0)
        XCTAssertEqual(model.lastRenderedAt, rendered)
    }

    func testRefreshFailureKeepsExistingDetail() async {
        let server = MockHTTPServer()
        let session = makeGuestSession(server)
        stubLiveView(server)
        let model = makeModel(session, sleep: { _ in })
        await model.loadIfNeeded()

        server.stub("GET", viewPath, .raw("down", status: 503))
        await model.refresh()

        XCTAssertNotNil(model.detail)
        XCTAssertEqual(model.phase, .loaded)
        XCTAssertNotNil(model.refreshError)
    }

    func testFailureShowsFeedbackAndRetryReloads() async {
        let server = MockHTTPServer()
        let session = makeGuestSession(server)
        server.stub("GET", viewPath, .raw("down", status: 503))
        let model = makeModel(session, sleep: { _ in })

        await model.loadIfNeeded()
        guard case .failed(let failure) = model.phase else {
            return XCTFail("首次加载失败应进入失败态")
        }
        XCTAssertEqual(failure.category, .server)
        XCTAssertEqual(failure.feedback(resource: "赛事详情").title, "服务暂不可用")

        stubLiveView(server)
        await model.retry()

        XCTAssertEqual(model.phase, .loaded)
        XCTAssertNotNil(model.detail)
    }

    func testMissingEventNewsPayloadIsNotFound() async {
        let server = MockHTTPServer()
        let session = makeGuestSession(server)
        server.stub("GET", viewPath, .json([
            "success": true,
            "event_news": NSNull(),
            "matches": [],
        ]))
        let model = makeModel(session, sleep: { _ in })

        await model.loadIfNeeded()

        guard case .failed(let failure) = model.phase else {
            return XCTFail("缺少 event_news 应进入失败态")
        }
        XCTAssertEqual(failure.category, .notFound)
        XCTAssertEqual(failure.message, "赛事数据不存在或已下线")
    }

    func testLogoutClearsDetailState() async {
        let server = MockHTTPServer()
        let session = await makeSignedInSession(server)
        stubLiveView(server)
        let model = SaiXunDetailModel(session: session, eventId: 18, now: { self.now }, sleep: { _ in })

        await model.loadIfNeeded()
        XCTAssertNotNil(model.detail)

        session.logout()
        model.syncIdentity()

        XCTAssertNil(model.detail)
        XCTAssertEqual(model.phase, .idle)
    }

    private func stubLiveView(_ server: MockHTTPServer) {
        server.stub("GET", viewPath, .json(EventNewsFixtures.viewResponse(
            eventNews: EventNewsFixtures.eventNewsJSON(
                tournamentName: "进行中的赛事",
                status: 1,
                startDate: "2026-03-30",
                endDate: "2026-04-05"
            ),
            matches: [
                EventNewsFixtures.matchJSON(
                    id: 1,
                    roundName: "Semi Finals",
                    roundOrder: 60,
                    status: 1,
                    startTime: "2026-04-01 14:00:00",
                    homeScore: 4,
                    awayScore: 3
                ),
                EventNewsFixtures.matchJSON(
                    id: 2,
                    roundName: "Semi Finals",
                    roundOrder: 60,
                    status: 2,
                    startTime: "2026-04-01 19:30:00",
                    homeScore: 6,
                    awayScore: 4,
                    winnerSide: 1
                ),
                EventNewsFixtures.matchJSON(
                    id: 3,
                    roundName: "Final",
                    roundOrder: 70,
                    status: 0,
                    startTime: "2026-04-02 20:00:00"
                ),
            ]
        )))
    }

    private func makeModel(
        _ session: SessionStore,
        sleep: @escaping (TimeInterval) async throws -> Void = { try await Task.sleep(for: .seconds($0)) }
    ) -> SaiXunDetailModel {
        SaiXunDetailModel(session: session, eventId: 18, now: { self.now }, sleep: sleep)
    }

    private func makeGuestSession(_ server: MockHTTPServer) -> SessionStore {
        makeSessionStore(server, credentialStore: InMemoryCredentialStore(credentials: nil))
    }

    private func makeSignedInSession(_ server: MockHTTPServer) async -> SessionStore {
        server.stub("GET", "/api/user/info", .json(AuthFixtures.userInfoResponse()))
        let session = makeSessionStore(
            server,
            credentialStore: InMemoryCredentialStore(credentials: AuthFixtures.credentials())
        )
        await session.restore()
        return session
    }
}
