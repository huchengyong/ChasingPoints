import XCTest
@testable import iOS

@MainActor
final class AchievementDetailModelTests: XCTestCase {
    private let path = "/api/achievement/detail"

    func testLoadRequestsAchievementAndExposesDisplayValues() async {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        server.stub("GET", path, .json(AchievementFixtures.unlockedDetail()))
        let model = AchievementDetailModel(session: session, achievementId: 7)

        await model.loadIfNeeded()

        XCTAssertTrue(model.hasLoaded)
        XCTAssertFalse(model.isInitialLoading)
        XCTAssertNil(model.failure)
        XCTAssertEqual(model.achievement?.name, "十胜起步")
        XCTAssertEqual(model.progressPercent, 100)
        XCTAssertEqual(model.progressText, "10 / 10")
        XCTAssertEqual(model.categoryLabel, "胜场")
        XCTAssertEqual(model.gameTypeLabel, "通用")
        XCTAssertEqual(model.fallbackEmoji, "🏅")
        XCTAssertFalse(model.dismissesOnError)
        XCTAssertEqual(server.requests(path).first?.query["achievement_id"], "7")
        XCTAssertEqual(server.requests(path).first?.bearerToken, AuthFixtures.oldAccess)
    }

    func testLockedAchievementShowsPartialProgressAndFallbackEmoji() async {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        server.stub("GET", path, .json(AchievementFixtures.lockedDetail()))
        let model = AchievementDetailModel(session: session, achievementId: 8)

        await model.loadIfNeeded()

        XCTAssertEqual(model.achievement?.unlocked, false)
        XCTAssertEqual(model.progressPercent, 33)
        XCTAssertEqual(model.progressText, "1 / 3")
        XCTAssertEqual(model.categoryLabel, "连胜")
        XCTAssertEqual(model.fallbackEmoji, "🔥")
    }

    func testMissingAchievementFailsThenRetryLoadsContent() async {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        server.stub("GET", path, .json(AchievementFixtures.missingDetail()))
        let model = AchievementDetailModel(session: session, achievementId: 99)

        await model.loadIfNeeded()

        XCTAssertFalse(model.hasLoaded)
        XCTAssertTrue(model.isInitialError)
        XCTAssertNotNil(model.failure)
        XCTAssertFalse(model.dismissesOnError, "业务失败可以重试")

        server.stub("GET", path, .json(AchievementFixtures.lockedDetail()))
        await model.retry()

        XCTAssertTrue(model.hasLoaded)
        XCTAssertNil(model.failure)
        XCTAssertFalse(model.isInitialError)
    }

    func testPermissionFailureDismissesInsteadOfRetrying() async {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        server.stub("GET", path, .json(["success": false, "message": "暂无权限"], status: 403))
        let model = AchievementDetailModel(session: session, achievementId: 7)

        await model.loadIfNeeded()

        XCTAssertTrue(model.dismissesOnError)
        XCTAssertEqual(model.failureFeedback?.actionText, "返回")
        XCTAssertEqual(model.failure?.category, .permission)
    }

    func testGuestDoesNotRequest() async {
        let server = MockHTTPServer()
        let session = RankingFixtures.makeGuestSession(server)
        let model = AchievementDetailModel(session: session, achievementId: 7)

        await model.loadIfNeeded()

        XCTAssertTrue(server.requests.isEmpty)
        XCTAssertFalse(model.hasLoaded)
        XCTAssertFalse(model.isInitialLoading)
        XCTAssertFalse(model.isInitialError)
    }

    func testIdentityChangeClearsDetail() async {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        server.stub("GET", path, .json(AchievementFixtures.unlockedDetail()))
        let model = AchievementDetailModel(session: session, achievementId: 7)
        await model.loadIfNeeded()
        XCTAssertTrue(model.hasLoaded)

        session.logout()
        model.syncIdentity()

        XCTAssertFalse(model.hasLoaded)
        XCTAssertNil(model.failure)
    }
}
