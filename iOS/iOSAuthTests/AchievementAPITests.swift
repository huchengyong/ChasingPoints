import XCTest
@testable import iOS

@MainActor
final class AchievementAPITests: XCTestCase {
    func testHonorWallBuildsBoundedQueryWithUserAndSeasonContext() async throws {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        server.stub("GET", "/api/achievement/honor-wall", .json(AchievementFixtures.honorWall()))
        let api = AchievementAPI(session: session)

        let response = try await api.honorWall(
            userId: 18,
            gameType: 4,
            historySeasonId: 3,
            historyPage: 2,
            historyPageSize: 10
        )

        XCTAssertTrue(response.success)
        let request = server.requests("/api/achievement/honor-wall").first
        XCTAssertEqual(request?.query["user_id"], "18")
        XCTAssertEqual(request?.query["game_type"], "4")
        XCTAssertEqual(request?.query["history_season_id"], "3")
        XCTAssertEqual(request?.query["history_page"], "2")
        XCTAssertEqual(request?.query["history_page_size"], "10")
        XCTAssertEqual(request?.bearerToken, AuthFixtures.oldAccess)
    }

    func testHonorWallOmitsOptionalContextWhenAbsent() async throws {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        server.stub("GET", "/api/achievement/honor-wall", .json(AchievementFixtures.emptyHonorWall()))
        let api = AchievementAPI(session: session)

        _ = try await api.honorWall(gameType: 3, historySeasonId: 0, historyPage: 1)

        let request = server.requests("/api/achievement/honor-wall").first
        XCTAssertNil(request?.query["user_id"])
        XCTAssertNil(request?.query["history_season_id"])
        XCTAssertEqual(request?.query["game_type"], "3")
    }

    func testDetailTitlesEquipAndNotificationRequests() async throws {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        server.stub("GET", "/api/achievement/detail", .json(AchievementFixtures.unlockedDetail()))
        server.stub("GET", "/api/achievement/titles", .json(AchievementFixtures.titles(AchievementFixtures.defaultTitles)))
        server.stub("POST", "/api/achievement/title/equip", .json(AchievementFixtures.common()))
        server.stub("GET", "/api/notification/list", .json(AchievementFixtures.notificationList([
            AchievementFixtures.notification(id: 2),
        ])))
        server.stub("POST", "/api/notification/read", .json(AchievementFixtures.common()))
        let api = AchievementAPI(session: session)

        let detail = try await api.detail(achievementId: 7)
        XCTAssertEqual(detail.achievement?.id, 7)
        XCTAssertEqual(server.requests("/api/achievement/detail").first?.query["achievement_id"], "7")

        let titles = try await api.titles()
        XCTAssertEqual(titles.list.map(\.id), [2, 3])

        try await api.equipTitle(titleId: 3, equip: true)
        XCTAssertEqual(server.requests("/api/achievement/title/equip").first?.bodyJSON["title_id"] as? Int64, 3)
        XCTAssertEqual(server.requests("/api/achievement/title/equip").first?.bodyJSON["equip"] as? Bool, true)

        let notifications = try await api.latestSeasonRolloverNotifications()
        XCTAssertEqual(notifications.map(\.id), [2])
        let listRequest = server.requests("/api/notification/list").first
        XCTAssertEqual(listRequest?.query["page"], "1")
        XCTAssertEqual(listRequest?.query["page_size"], "10")
        XCTAssertEqual(listRequest?.query["type"], "season_rollover")

        try await api.markNotificationRead(notificationId: 2)
        XCTAssertEqual(server.requests("/api/notification/read").first?.bodyJSON["notification_id"] as? Int64, 2)
    }

    func testEquipFailureSurfacesServerMessage() async {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        server.stub("POST", "/api/achievement/title/equip", .json(["success": false, "message": "称号不存在"]))
        let api = AchievementAPI(session: session)

        do {
            try await api.equipTitle(titleId: 3, equip: false)
            XCTFail("业务失败应抛出错误")
        } catch {
            XCTAssertEqual(error.localizedDescription, "称号不存在")
        }
    }
}
