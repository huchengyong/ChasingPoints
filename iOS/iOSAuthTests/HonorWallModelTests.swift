import XCTest
@testable import iOS

@MainActor
final class HonorWallModelTests: XCTestCase {
    private let path = "/api/achievement/honor-wall"
    private let titlesPath = "/api/achievement/titles"
    private let equipPath = "/api/achievement/title/equip"

    func testFirstLoadRequestsHonorWallWithGameTypeAndBearer() async {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        server.stub("GET", path, .json(AchievementFixtures.honorWall()))
        let model = HonorWallModel(session: session, gameType: 3)

        await loadIfNeeded(model)

        XCTAssertTrue(model.hasLoaded)
        XCTAssertFalse(model.isInitialLoading)
        XCTAssertNil(model.failure)
        XCTAssertTrue(model.isSelf)
        XCTAssertEqual(model.profileName, "测试用户")
        XCTAssertEqual(model.equippedTitleName, "S3 冠军")
        XCTAssertEqual(model.equippedTitleSource, "赛季荣誉 · S3")
        XCTAssertEqual(model.activeTitleActionText, "更换")
        XCTAssertEqual(model.summaryItems.map(\.value), ["3/9", "2/4", "1", "1"])
        XCTAssertEqual(model.recentHonors.map(\.id), [20])
        XCTAssertEqual(model.universalGroups.map(\.key), ["match", "wins", "streak"])
        XCTAssertEqual(model.specialtyAchievements.map(\.id), [4, 5])
        XCTAssertEqual(model.upcomingSection.items.map(\.achievement.id), [1, 3, 4])
        XCTAssertEqual(model.currentSeasonState.mode, .active)
        XCTAssertEqual(model.seasonDateText, "2026.07.01 - 2026.09.30")
        XCTAssertEqual(model.currentChallenges.count, 3)
        XCTAssertEqual(model.historyHonors.count, 2)
        XCTAssertEqual(model.historySeasonOptions.map(\.id), [3])
        XCTAssertFalse(model.hasMoreHistory)

        let request = server.requests(path).first
        XCTAssertEqual(request?.query["game_type"], "3")
        XCTAssertEqual(request?.query["history_page"], "1")
        XCTAssertEqual(request?.query["history_page_size"], "20")
        XCTAssertNil(request?.query["user_id"])
        XCTAssertNil(request?.query["history_season_id"])
        XCTAssertEqual(request?.bearerToken, AuthFixtures.oldAccess)
        XCTAssertEqual(server.requestCount(path), 1)
    }

    func testSecondAppearanceDoesNotRequestAgain() async {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        server.stub("GET", path, .json(AchievementFixtures.honorWall()))
        let model = HonorWallModel(session: session, gameType: 3)

        await loadIfNeeded(model)
        await loadIfNeeded(model)

        XCTAssertEqual(server.requestCount(path), 1)
    }

    func testSwitchingGameTypeReloadsWithNewQueryAndKeepsContent() async {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        server.stub("GET", path) { request in
            let gameType = Int(request.query["game_type"] ?? "3") ?? 3
            return .json(AchievementFixtures.honorWall(nickname: "球种 \(String(gameType))"))
        }
        let model = HonorWallModel(session: session, gameType: 3)
        await loadIfNeeded(model)

        await model.selectGameType(2)

        XCTAssertEqual(model.currentGameType, 2)
        XCTAssertEqual(model.currentGameTypeLabel, "九球追分")
        XCTAssertEqual(model.profileName, "球种 2")
        XCTAssertEqual(server.requestCount(path), 2)
        XCTAssertEqual(server.requests(path).last?.query["game_type"], "2")
    }

    func testSwitchingHistorySeasonAndLoadingMoreAppendHonors() async {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        let firstPage = AchievementFixtures.honorWall(history: [
            "total": 4,
            "honors": [
                AchievementFixtures.honorItem(id: 10, sourceType: "season", sourceRefId: 3, sourceRefName: "S3"),
                AchievementFixtures.honorItem(id: 11, sourceType: "tournament"),
            ],
            "challenge_season": NSNull(),
            "challenge_records": [],
        ])
        let secondPage = AchievementFixtures.honorWall(history: [
            "total": 4,
            "honors": [AchievementFixtures.honorItem(id: 12, sourceType: "tournament")],
            "challenge_season": NSNull(),
            "challenge_records": [],
        ])
        server.stub("GET", path) { request in
            if request.query["history_page"] == "2" { return .json(secondPage) }
            return .json(firstPage)
        }
        let model = HonorWallModel(session: session, gameType: 3)
        await loadIfNeeded(model)

        XCTAssertTrue(model.hasMoreHistory)
        await model.loadMoreHistory()

        XCTAssertEqual(model.historyHonors.map(\.id), [10, 11, 12])
        XCTAssertFalse(model.isHistoryLoading)
        XCTAssertEqual(server.requests(path).last?.query["history_page"], "2")
    }

    func testEmptyHonorWallShowsEmptyStates() async {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        server.stub("GET", path, .json(AchievementFixtures.emptyHonorWall()))
        let model = HonorWallModel(session: session, gameType: 3)

        await loadIfNeeded(model)

        XCTAssertTrue(model.hasLoaded)
        XCTAssertNil(model.wall?.equippedTitle)
        XCTAssertEqual(model.equippedTitleName, "暂未佩戴称号")
        XCTAssertEqual(model.equippedTitleSource, "选择一个已获称号进行展示")
        XCTAssertEqual(model.activeTitleActionText, "选择")
        XCTAssertTrue(model.recentHonors.isEmpty)
        XCTAssertTrue(model.universalGroups.isEmpty)
        XCTAssertTrue(model.specialtyAchievements.isEmpty)
        XCTAssertTrue(model.upcomingSection.items.isEmpty)
        XCTAssertFalse(model.upcomingSection.completed)
        XCTAssertEqual(model.currentSeasonState.mode, .notStarted)
        XCTAssertTrue(model.historyHonors.isEmpty)
        XCTAssertFalse(model.hasMoreHistory)
        XCTAssertEqual(model.historyTotalText, "0")
    }

    func testInitialFailureThenRetryLoadsContent() async {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        server.stub("GET", path, .raw("service down", status: 503))
        let model = HonorWallModel(session: session, gameType: 3)

        await loadIfNeeded(model)

        XCTAssertFalse(model.hasLoaded)
        XCTAssertTrue(model.isInitialFailure)
        XCTAssertEqual(model.loadError?.kind, .server)
        XCTAssertEqual(model.loadError?.showRetry, true)

        server.stub("GET", path, .json(AchievementFixtures.honorWall()))
        await model.retry()

        XCTAssertTrue(model.hasLoaded)
        XCTAssertNil(model.failure)
        XCTAssertFalse(model.isInitialFailure)
    }

    func testRefreshFailureKeepsContentAndShowsNotice() async {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        server.stub("GET", path, .json(AchievementFixtures.honorWall()))
        let model = HonorWallModel(session: session, gameType: 3)
        await loadIfNeeded(model)

        server.stub("GET", path, .raw("service down", status: 503))
        await model.refresh()

        XCTAssertTrue(model.hasLoaded)
        XCTAssertNil(model.failure)
        XCTAssertFalse(model.isInitialFailure)
        XCTAssertEqual(model.notice, "服务暂时不可用，请稍后重试")
    }

    func testGuestDoesNotRequest() async {
        let server = MockHTTPServer()
        let session = RankingFixtures.makeGuestSession(server)
        let model = HonorWallModel(session: session, gameType: 3)

        await loadIfNeeded(model)

        XCTAssertTrue(server.requests.isEmpty)
        XCTAssertFalse(model.hasLoaded)
        XCTAssertFalse(model.isInitialLoading)
        XCTAssertFalse(model.isInitialFailure)
    }

    func testIdentityChangeClearsPageState() async {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        server.stub("GET", path, .json(AchievementFixtures.honorWall()))
        let model = HonorWallModel(session: session, gameType: 3)
        await loadIfNeeded(model)
        XCTAssertTrue(model.hasLoaded)

        session.logout()
        model.syncIdentity()

        XCTAssertFalse(model.hasLoaded)
        XCTAssertNil(model.failure)
        XCTAssertTrue(model.titleList.isEmpty)
        XCTAssertEqual(model.currentGameType, 3)
    }

    func testFriendWallKeepsSeasonTabPrivate() async {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        server.stub("GET", path, .json(AchievementFixtures.friendHonorWall()))
        let model = HonorWallModel(session: session, gameType: 3, userId: 2)

        await loadIfNeeded(model)

        XCTAssertFalse(model.isSelf)
        XCTAssertEqual(model.tabs, [.career, .history])
        XCTAssertEqual(model.profileKicker, "好友公开荣誉")
        XCTAssertEqual(model.equippedTitleSource, "TA 暂未佩戴称号")
        XCTAssertTrue(model.upcomingSection.hidden)
        XCTAssertEqual(model.universalGroups.map(\.key), ["match"])
        XCTAssertTrue(model.specialtyAchievements.isEmpty)
        XCTAssertEqual(server.requests(path).first?.query["user_id"], "2")
    }

    func testTitleListLoadsAndEquippingTitleUpdatesWall() async {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        server.stub("GET", path, .json(AchievementFixtures.honorWall()))
        server.stub("GET", titlesPath, .json(AchievementFixtures.titles(AchievementFixtures.defaultTitles)))
        server.stub("POST", equipPath, .json(AchievementFixtures.common()))
        let model = HonorWallModel(session: session, gameType: 3)
        await loadIfNeeded(model)

        await model.prepareTitleSelector()

        XCTAssertTrue(model.isTitleListLoaded)
        XCTAssertFalse(model.isTitleListFailed)
        XCTAssertEqual(model.titleOptions.map(\.id), [2, 3])
        let option = try? XCTUnwrap(model.titleOptions.first { $0.id == 3 })

        let closed = await model.selectTitle(option)

        XCTAssertTrue(closed)
        XCTAssertEqual(model.notice, "已佩戴「城市杯四强」")
        XCTAssertEqual(model.wall?.equippedTitle?.id, 3)
        XCTAssertEqual(model.titleOptions.filter(\.equipped).map(\.id), [3])
        let body = server.requests(equipPath).first?.bodyJSON
        XCTAssertEqual(body?["title_id"] as? Int64, 3)
        XCTAssertEqual(body?["equip"] as? Bool, true)
    }

    func testEquipFailureKeepsCurrentTitleAndKeepsPanelOpen() async {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        server.stub("GET", path, .json(AchievementFixtures.honorWall()))
        server.stub("GET", titlesPath, .json(AchievementFixtures.titles(AchievementFixtures.defaultTitles)))
        server.stub("POST", equipPath, .json(["success": false, "message": "称号不存在"]))
        let model = HonorWallModel(session: session, gameType: 3)
        await loadIfNeeded(model)
        await model.prepareTitleSelector()
        let option = model.titleOptions.first { $0.id == 3 }

        let closed = await model.selectTitle(option)

        XCTAssertFalse(closed)
        XCTAssertEqual(model.notice, "称号不存在")
        XCTAssertEqual(model.wall?.equippedTitle?.id, 2)
        XCTAssertEqual(model.titleOptions.first { $0.id == 2 }?.equipped, true)
        XCTAssertFalse(model.isTitleSubmitting)
    }

    func testEquipWithoutCurrentTitleShowsEmptyNameAndUnequipClearsWall() async {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        server.stub("GET", path, .json(AchievementFixtures.honorWall(equippedTitle: nil)))
        server.stub("GET", titlesPath, .json(AchievementFixtures.titles([
            AchievementFixtures.titleInfo(id: 3, titleName: "城市杯四强", source: "tournament", sourceType: "tournament"),
        ])))
        server.stub("POST", equipPath, .json(AchievementFixtures.common()))
        let model = HonorWallModel(session: session, gameType: 3)
        await loadIfNeeded(model)
        await model.prepareTitleSelector()

        let equipped = await model.selectTitle(model.titleOptions.first)
        XCTAssertTrue(equipped)
        XCTAssertEqual(model.notice, "已佩戴「城市杯四强」")
        XCTAssertEqual(model.wall?.equippedTitle?.id, 3)

        let unequipped = await model.selectNoTitle()
        XCTAssertTrue(unequipped)
        XCTAssertEqual(model.notice, "已停止展示称号")
        XCTAssertNil(model.wall?.equippedTitle)
        XCTAssertEqual(server.requests(equipPath).last?.bodyJSON["equip"] as? Bool, false)
    }

    func testTitleListFailureShowsRetryState() async {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        server.stub("GET", path, .json(AchievementFixtures.honorWall()))
        server.stub("GET", titlesPath, .raw("service down", status: 503))
        let model = HonorWallModel(session: session, gameType: 3)
        await loadIfNeeded(model)

        await model.prepareTitleSelector()

        XCTAssertTrue(model.isTitleListFailed)
        XCTAssertFalse(model.isTitleListLoaded)
        XCTAssertFalse(model.isTitleListLoading)
    }

    func testLatestUnreadRolloverIsHandledByPresenterNotModel() async {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        server.stub("GET", "/api/notification/list", .json(AchievementFixtures.notificationList([
            AchievementFixtures.notification(id: 2),
        ])))
        server.stub("GET", path, .json(AchievementFixtures.honorWall()))
        let model = HonorWallModel(session: session, gameType: 3)

        await model.loadIfNeeded()

        // 换季提醒改由页面弹窗展示，模型加载不再旁路标已读。
        XCTAssertTrue(model.hasLoaded)
        XCTAssertEqual(server.requestCount("/api/notification/list"), 0)
        XCTAssertEqual(server.requestCount("/api/notification/read"), 0)
    }

    // MARK: - 辅助

    private func loadIfNeeded(_ model: HonorWallModel) async {
        await model.loadIfNeeded()
    }
}
