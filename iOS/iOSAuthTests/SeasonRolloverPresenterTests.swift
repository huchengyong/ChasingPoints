import XCTest
@testable import iOS

@MainActor
final class SeasonRolloverPresenterTests: XCTestCase {
    private let listPath = "/api/notification/list"
    private let readPath = "/api/notification/read"

    func testNoUnreadRolloverShowsNothing() async {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        stubList(server, notifications: [])
        let presenter = SeasonRolloverPresenter(session: session)

        await presenter.present()

        XCTAssertNil(presenter.alert)
        XCTAssertEqual(server.requestCount(readPath), 0)
        let request = server.requests(listPath).first
        XCTAssertEqual(request?.query["type"], "season_rollover")
        XCTAssertEqual(request?.query["page"], "1")
        XCTAssertEqual(request?.query["page_size"], "10")
    }

    func testUnreadRolloverIsShownButNotMarkedReadUntilClosed() async {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        stubList(server, notifications: [AchievementFixtures.notification(id: 2)])
        server.stub("POST", readPath, .json(AchievementFixtures.common()))
        let presenter = SeasonRolloverPresenter(session: session)
        var readCallbacks = 0

        let task = Task {
            await presenter.present(
                onRead: { readCallbacks += 1 }
            )
        }
        await waitUntil { presenter.alert != nil }

        XCTAssertEqual(presenter.alert?.title, "S3 已结算")
        XCTAssertEqual(presenter.alert?.confirmText, "知道了")
        XCTAssertEqual(presenter.alert?.showCancel, false)
        XCTAssertNil(presenter.alert?.reportRoute)
        XCTAssertEqual(server.requestCount(readPath), 0, "弹窗关闭前不得标已读")
        XCTAssertEqual(readCallbacks, 0)

        presenter.respond(confirmed: false)
        await task.value

        XCTAssertNil(presenter.alert)
        XCTAssertEqual(server.requestCount(readPath), 1)
        XCTAssertEqual(server.requests(readPath).first?.bodyJSON["notification_id"] as? Int64, 2)
        XCTAssertEqual(readCallbacks, 1)
    }

    func testRespondOnlyMarksReadOnce() async {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        stubList(server, notifications: [AchievementFixtures.notification(id: 3)])
        server.stub("POST", readPath, .json(AchievementFixtures.common()))
        let presenter = SeasonRolloverPresenter(session: session)
        var readCallbacks = 0

        let task = Task {
            await presenter.present(onRead: { readCallbacks += 1 })
        }
        await waitUntil { presenter.alert != nil }

        presenter.respond(confirmed: false)
        presenter.respond(confirmed: false)
        await task.value

        XCTAssertEqual(server.requestCount(readPath), 1)
        XCTAssertEqual(readCallbacks, 1)
    }

    func testConfirmNavigatesToReportAfterMarkingRead() async {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        let reportData = """
        {"from_season_name":"S3","to_season_id":4,"to_season_name":"S4","url":"/subPages/season/report?id=4&game_type=2"}
        """
        stubList(server, notifications: [AchievementFixtures.notification(id: 5, data: reportData)])
        server.stub("POST", readPath, .json(AchievementFixtures.common()))
        let presenter = SeasonRolloverPresenter(session: session)
        var navigated: AppRoute?
        var events: [String] = []

        let task = Task {
            await presenter.present(
                onRead: { events.append("read") },
                onNavigate: { route in
                    events.append("navigate")
                    navigated = route
                }
            )
        }
        await waitUntil { presenter.alert != nil }

        XCTAssertEqual(presenter.alert?.title, "S4 已开启")
        XCTAssertEqual(presenter.alert?.confirmText, "查看回顾")
        XCTAssertEqual(presenter.alert?.showCancel, true)
        XCTAssertEqual(presenter.alert?.reportRoute, .seasonReport(seasonId: 4, gameType: 2))

        presenter.respond(confirmed: true)
        await task.value

        XCTAssertEqual(navigated, .seasonReport(seasonId: 4, gameType: 2))
        XCTAssertEqual(events, ["read", "navigate"], "必须先标已读再跳转")
    }

    func testReportEntryIsHiddenWithoutNavigationHandler() async {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        let reportData = """
        {"from_season_name":"S3","to_season_id":4,"to_season_name":"S4","url":"/subPages/season/report?id=4&game_type=2"}
        """
        stubList(server, notifications: [AchievementFixtures.notification(id: 5, data: reportData)])
        server.stub("POST", readPath, .json(AchievementFixtures.common()))
        let presenter = SeasonRolloverPresenter(session: session)

        let task = Task { await presenter.present() }
        await waitUntil { presenter.alert != nil }

        XCTAssertEqual(presenter.alert?.confirmText, "知道了")
        XCTAssertEqual(presenter.alert?.showCancel, false)
        XCTAssertNil(presenter.alert?.reportRoute)

        presenter.respond(confirmed: true)
        await task.value
        XCTAssertEqual(server.requestCount(readPath), 1)
    }

    func testInvalidReportURLIsIgnored() async {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        let reportData = """
        {"from_season_name":"S3","to_season_id":4,"to_season_name":"S4","url":"/subPages/season/report?id=0&game_type=2"}
        """
        stubList(server, notifications: [AchievementFixtures.notification(id: 5, data: reportData)])
        server.stub("POST", readPath, .json(AchievementFixtures.common()))
        let presenter = SeasonRolloverPresenter(session: session)
        var navigated: AppRoute?

        let task = Task {
            await presenter.present(onNavigate: { navigated = $0 })
        }
        await waitUntil { presenter.alert != nil }

        XCTAssertNil(presenter.alert?.reportRoute)
        XCTAssertEqual(presenter.alert?.confirmText, "知道了")

        presenter.respond(confirmed: true)
        await task.value
        XCTAssertNil(navigated)
    }

    func testConcurrentPresentationsShowOnlyOnce() async {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        let gate = DispatchSemaphore(value: 0)
        server.stub("GET", listPath) { _ in
            _ = gate.wait(timeout: .now() + 5)
            return .json(AchievementFixtures.notificationList([AchievementFixtures.notification(id: 9)]))
        }
        server.stub("POST", readPath, .json(AchievementFixtures.common()))
        let presenter = SeasonRolloverPresenter(session: session)

        let first = Task { await presenter.present() }
        let second = Task { await presenter.present() }
        await waitUntil { server.requestCount(self.listPath) == 1 }
        gate.signal()
        await waitUntil { presenter.alert != nil }

        XCTAssertEqual(server.requestCount(listPath), 1, "同一时刻只能有一次换季提醒读取")

        presenter.respond(confirmed: false)
        await first.value
        await second.value

        XCTAssertEqual(server.requestCount(readPath), 1)
    }

    func testLateNotificationAfterLogoutIsNotShown() async {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        let gate = DispatchSemaphore(value: 0)
        server.stub("GET", listPath) { _ in
            _ = gate.wait(timeout: .now() + 5)
            return .json(AchievementFixtures.notificationList([AchievementFixtures.notification(id: 9)]))
        }
        let presenter = SeasonRolloverPresenter(session: session)

        let task = Task { await presenter.present() }
        await waitUntil { server.requestCount(self.listPath) == 1 }
        session.logout()
        gate.signal()
        await task.value

        XCTAssertNil(presenter.alert, "旧身份的迟到通知不得弹给新身份")
    }

    func testCachedNotificationSkipsListRequest() async throws {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        server.stub("POST", readPath, .json(AchievementFixtures.common()))
        let presenter = SeasonRolloverPresenter(session: session)
        let decoder = JSONDecoder()
        decoder.keyDecodingStrategy = .convertFromSnakeCase
        let notification = try decoder.decode(
            NotificationInfo.self,
            from: JSONSerialization.data(withJSONObject: AchievementFixtures.notification(id: 7))
        )

        let task = Task { await presenter.present(notification: notification) }
        await waitUntil { presenter.alert != nil }

        XCTAssertEqual(server.requestCount(listPath), 0)
        presenter.respond(confirmed: false)
        await task.value
        XCTAssertEqual(server.requestCount(readPath), 1)
    }

    func testGuestShowsNothing() async {
        let server = MockHTTPServer()
        let session = RankingFixtures.makeGuestSession(server)
        let presenter = SeasonRolloverPresenter(session: session)

        await presenter.present()

        XCTAssertNil(presenter.alert)
        XCTAssertTrue(server.requests.isEmpty)
    }

    // MARK: - 辅助

    private func stubList(_ server: MockHTTPServer, notifications: [[String: Any]]) {
        server.stub("GET", listPath, .json(AchievementFixtures.notificationList(notifications)))
    }
}
