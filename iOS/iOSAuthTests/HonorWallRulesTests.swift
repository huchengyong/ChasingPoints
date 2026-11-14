import XCTest
@testable import iOS

final class HonorWallRulesTests: XCTestCase {
    func testOptionsAndTabsKeepSelfProgressPrivateFromFriendViews() {
        let options = HonorWallRules.normalizeOptions(userId: 18, gameType: 4, tab: "season")
        XCTAssertEqual(options.userId, 18)
        XCTAssertEqual(options.gameType, 4)
        XCTAssertEqual(options.tab, .season)
        XCTAssertEqual(options.historySeasonId, 0)
        XCTAssertEqual(HonorWallRules.tabs(viewerScope: "self"), [.career, .season, .history])
        XCTAssertEqual(HonorWallRules.tabs(viewerScope: "friend"), [.career, .history])
    }

    func testCareerSummarySeparatesUniversalAndSelectedSpecialtyProgress() {
        let summary = HonorWallSummary(
            careerUnlocked: 10,
            careerTotal: 18,
            universalUnlocked: 8,
            universalTotal: 15,
            specialtyGameType: 3,
            specialtyUnlocked: 2,
            specialtyTotal: 3,
            seasonHonors: 4,
            tournamentHonors: 5
        )

        XCTAssertEqual(
            HonorWallRules.careerSummaryItems(summary, gameType: 3, gameTypeLabel: "中式八球"),
            [
                HonorWallSummaryItem(key: "universal", label: "通用成就", value: "8/15"),
                HonorWallSummaryItem(key: "specialty", label: "中式八球专精", value: "2/3"),
                HonorWallSummaryItem(key: "season", label: "赛季荣誉", value: "4"),
                HonorWallSummaryItem(key: "tournament", label: "赛事荣誉", value: "5"),
            ]
        )
    }

    func testUpcomingAchievementsUseUniversalPlusSelectedGameTypeWithStableRanking() {
        let achievements = [
            def(id: 1, gameType: 0, progress: 8, threshold: 10),
            def(id: 2, gameType: 3, progress: 4, threshold: 5),
            def(id: 3, gameType: 3, progress: 0, threshold: 1),
            def(id: 4, gameType: 1, progress: 99, threshold: 100),
            def(id: 5, gameType: 0, progress: 1, threshold: 1, unlocked: true),
            def(id: 6, gameType: 0, progress: 2, threshold: 10),
        ]

        let section = HonorWallRules.upcomingSection(achievements, gameType: 3, viewerScope: "self")
        XCTAssertFalse(section.hidden)
        XCTAssertFalse(section.completed)
        XCTAssertEqual(section.items.map(\.achievement.id), [1, 2, 6])
        XCTAssertEqual(section.items[0].progressPercent, 80)
        XCTAssertEqual(section.items[2].remainingText, "还差 8 达成")

        let zeroFallback = HonorWallRules.upcomingSection(
            [
                def(id: 10, gameType: 3, progress: 0, threshold: 1),
                def(id: 11, gameType: 0, progress: 0, threshold: 10),
            ],
            gameType: 3,
            viewerScope: "self"
        )
        XCTAssertEqual(zeroFallback.items.map(\.achievement.id), [10, 11])

        let complete = HonorWallRules.upcomingSection(
            [
                def(id: 20, gameType: 0, progress: 1, threshold: 1, unlocked: true),
                def(id: 21, gameType: 3, progress: 1, threshold: 1, unlocked: true),
                def(id: 22, gameType: 4, progress: 0, threshold: 1),
            ],
            gameType: 3,
            viewerScope: "self"
        )
        XCTAssertTrue(complete.completed)
        XCTAssertTrue(complete.items.isEmpty)

        let invalid = HonorWallRules.upcomingSection(
            [
                def(id: 30, gameType: 3, progress: -5, threshold: 1),
                def(id: 31, gameType: 3, progress: 1, threshold: 0),
                def(id: 32, gameType: 4, progress: 1, threshold: 1),
            ],
            gameType: 99,
            viewerScope: "self"
        )
        XCTAssertEqual(invalid.items.map(\.achievement.id), [30])
        XCTAssertEqual(invalid.items[0].progress, 0)

        let friend = HonorWallRules.upcomingSection(achievements, gameType: 3, viewerScope: "friend")
        XCTAssertTrue(friend.hidden)
        XCTAssertTrue(friend.items.isEmpty)
    }

    func testLatestRequestGuardKeepsOnlyTheNewestResponse() {
        var guardValue = LatestRequestGuard()
        let first = guardValue.next()
        let second = guardValue.next()

        XCTAssertFalse(guardValue.isLatest(first))
        XCTAssertTrue(guardValue.isLatest(second))
    }

    func testChallengeViewModelExposesReadableProgressAndCompletionCopy() {
        let item = HonorWallRules.challengeItem(
            SeasonChallengeInfo(
                key: "season_matches_20",
                name: "赛季对局",
                description: "本赛季完成有效对局",
                threshold: 20,
                progress: 12,
                completed: false
            )
        )
        XCTAssertEqual(item, HonorWallChallengeItem(
            key: "season_matches_20",
            name: "赛季对局",
            description: "本赛季完成有效对局",
            progress: 12,
            threshold: 20,
            completed: false,
            iconPath: "/static/images/achievements/season_matches_20.png",
            progressPercent: 60,
            progressText: "12/20",
            remainingText: "还差 8 完成"
        ))

        XCTAssertEqual(
            HonorWallRules.challengeItem(challenge(key: "season_wins_10")).iconPath,
            "/static/images/achievements/season_wins_10.png"
        )
        XCTAssertEqual(
            HonorWallRules.challengeItem(challenge(key: "season_tournament_finish_1")).iconPath,
            "/static/images/achievements/season_tournament_finish_1.png"
        )
        XCTAssertEqual(HonorWallRules.challengeItem(challenge(key: "unknown")).iconPath, "")

        let overflow = HonorWallRules.challengeItem(
            SeasonChallengeInfo(key: "k", name: "n", description: "d", threshold: 20, progress: 23, completed: false)
        )
        XCTAssertEqual(overflow.progressPercent, 100)
        XCTAssertEqual(overflow.progressText, "已完成")

        let exact = HonorWallRules.challengeItem(
            SeasonChallengeInfo(key: "k", name: "n", description: "d", threshold: 20, progress: 20, completed: false)
        )
        XCTAssertEqual(exact.progressText, "已完成")
    }

    func testRecentHonorsNormalizeAchievementTitlePairingWithoutDuplicateRows() {
        let item = HonorWallRules.recentHonor(
            honorItem(
                id: 1,
                name: "十胜起步",
                sourceType: "achievement",
                rewardTitleName: "胜场新星"
            )
        )

        XCTAssertEqual(item.id, 1)
        XCTAssertEqual(item.name, "十胜起步")
        XCTAssertEqual(item.sourceLabel, "生涯成就")
        XCTAssertEqual(item.detailText, "同时获得称号「胜场新星」")

        let fallback = HonorWallRules.recentHonor(honorItem(id: 2, description: "", sourceRefName: ""))
        XCTAssertEqual(fallback.detailText, "已永久记录")
    }

    func testCurrentSeasonStateDistinguishesActiveNotStartedAndUnavailable() {
        XCTAssertEqual(
            HonorWallRules.currentSeasonState(nil, seasonState: ""),
            HonorWallSeasonState(
                mode: .notStarted,
                title: "赛季尚未开启",
                description: "首个赛季开启后，这里会展示赛季挑战与当前进度。"
            )
        )
        let unavailable = HonorWallRules.currentSeasonState(nil, seasonState: "unavailable")
        XCTAssertEqual(unavailable.mode, .unavailable)
        XCTAssertEqual(unavailable.title, "赛季数据更新中")

        let active = HonorWallRules.currentSeasonState(season(seasonId: 3, seasonName: "S3"), seasonState: "active")
        XCTAssertEqual(active.mode, .active)
        XCTAssertEqual(active.title, "S3")
    }

    func testHistorySeasonOptionsDedupePermanentSeasonHonors() {
        let options = HonorWallRules.historySeasonOptions([
            honorItem(id: 1, sourceType: "season", sourceRefId: 3, sourceRefName: "S3"),
            honorItem(id: 2, sourceType: "season", sourceRefId: 3, sourceRefName: "S3"),
            honorItem(id: 3, sourceType: "tournament", sourceRefId: 9, sourceRefName: "城市杯"),
        ])

        XCTAssertEqual(options, [HonorWallSeasonOption(id: 3, label: "S3")])
    }

    func testRolloverModalExplainsAutomaticSeasonTransition() {
        let rollover = notification(
            id: 9,
            data: """
            {"from_season_id":3,"from_season_name":"S3","to_season_id":4,"to_season_name":"S4",\
            "challenge_archived_count":3,"challenge_completed_count":2,"season_title_count":1,"intermission":false}
            """
        )
        let modal = HonorWallRules.seasonRollover(rollover)
        XCTAssertEqual(modal.title, "S4 已开启")
        XCTAssertTrue(modal.content.contains("S3 已归档"))
        XCTAssertTrue(modal.content.contains("S4 已自动开启"))
        XCTAssertTrue(modal.content.contains("生涯成就保持不变"))

        let intermission = HonorWallRules.seasonRollover(
            notification(id: 10, data: "{\"from_season_name\":\"S3\",\"to_season_id\":0,\"intermission\":true}")
        )
        XCTAssertEqual(intermission.title, "S3 已结算")
        XCTAssertTrue(intermission.content.contains("赛季间歇期"))
        XCTAssertFalse(intermission.content.contains("S4"))
    }

    func testRolloverOffersReviewOnlyForReportTargets() {
        let report = notification(
            id: 21,
            data: """
            {"from_season_id":3,"from_season_name":"S3","to_season_id":4,"to_season_name":"S4",\
            "url":"/subPages/season/report?id=3&game_type=1"}
            """
        )
        let review = HonorWallRules.seasonRollover(report)
        XCTAssertEqual(review.confirmText, "查看回顾")
        XCTAssertEqual(review.cancelText, "知道了")
        XCTAssertTrue(review.showCancel)
        XCTAssertEqual(review.reportURL, "/subPages/season/report?id=3&game_type=1")

        let withoutNavigation = HonorWallRules.seasonRollover(report, allowReport: false)
        XCTAssertFalse(withoutNavigation.showCancel)
        XCTAssertEqual(withoutNavigation.confirmText, "知道了")
        XCTAssertEqual(withoutNavigation.reportURL, "")

        let legacy = HonorWallRules.seasonRollover(
            notification(id: 22, data: "{\"from_season_name\":\"S2\",\"to_season_id\":3,\"url\":\"/subPages/achievement/index?tab=season\"}")
        )
        XCTAssertFalse(legacy.showCancel)
        XCTAssertEqual(legacy.reportURL, "")
    }

    func testLatestUnreadRolloverSelectionIgnoresReadAndUnrelatedNotifications() {
        let latest = HonorWallRules.latestUnreadSeasonRollover([
            notification(id: 1, isRead: true, createdAt: "2026-07-01"),
            notification(id: 2, type: "match_result", createdAt: "2026-08-01"),
            notification(id: 3, createdAt: "2026-07-02"),
            notification(id: 4, createdAt: "2026-07-03"),
        ])
        XCTAssertEqual(latest?.id, 4)
    }

    func testRouteQueryPreservesFriendAndGameTypeContext() {
        XCTAssertEqual(
            HonorWallRules.routeQuery(userId: 18, gameType: 4, tab: .history, historySeasonId: 3),
            [
                "game_type": "4",
                "tab": "history",
                "user_id": "18",
                "history_season_id": "3",
            ]
        )
        XCTAssertEqual(HonorWallRules.routeQuery(), ["game_type": "3", "tab": "career"])
    }

    func testLoadErrorsAreClassifiedByRequestCategory() {
        let network = HonorWallRules.loadError(category: .network)
        XCTAssertEqual(network.kind, .network)
        XCTAssertEqual(network.title, "荣誉墙暂时没加载出来")
        XCTAssertTrue(network.description.contains("请检查网络"))
        XCTAssertTrue(network.showRetry)

        let permission = HonorWallRules.loadError(category: .permission)
        XCTAssertEqual(permission.kind, .permission)
        XCTAssertFalse(permission.showRetry)
        XCTAssertFalse(permission.description.contains("网络"))

        let server = HonorWallRules.loadError(category: .server)
        XCTAssertEqual(server.kind, .server)
        XCTAssertFalse(server.description.contains("网络"))
        XCTAssertTrue(server.showRetry)

        let business = HonorWallRules.loadError(category: .business)
        XCTAssertEqual(business.kind, .server)
        XCTAssertFalse(business.description.contains("网络"))

        let session = HonorWallRules.loadError(category: .session)
        XCTAssertEqual(session.kind, .session)
        XCTAssertFalse(session.showRetry)
        XCTAssertFalse(session.description.contains("网络"))
    }

    // MARK: - 辅助

    private func def(
        id: Int64,
        gameType: Int,
        progress: Int,
        threshold: Int,
        unlocked: Bool = false
    ) -> AchievementDef {
        AchievementDef(
            id: id,
            key: "k\(id)",
            name: "成就 \(id)",
            description: "",
            icon: "",
            category: "wins",
            gameType: gameType,
            rewardTitleName: nil,
            threshold: threshold,
            progress: progress,
            unlocked: unlocked,
            unlockedAt: nil
        )
    }

    private func challenge(key: String) -> SeasonChallengeInfo {
        SeasonChallengeInfo(key: key, name: "挑战", description: "描述", threshold: 0, progress: 0, completed: false)
    }

    private func season(seasonId: Int64, seasonName: String) -> HonorWallSeasonInfo {
        HonorWallSeasonInfo(
            seasonId: seasonId,
            seasonName: seasonName,
            startDate: "2026-07-01 00:00:00",
            endDate: "2026-09-30 23:59:59",
            status: 1,
            gameType: 3,
            challenges: []
        )
    }

    private func honorItem(
        id: Int64,
        name: String = "荣誉",
        description: String = "描述",
        sourceType: String = "achievement",
        sourceRefId: Int64? = nil,
        sourceRefName: String? = nil,
        rewardTitleName: String? = nil
    ) -> HonorItem {
        HonorItem(
            id: id,
            type: "achievement",
            name: name,
            description: description,
            icon: "",
            sourceType: sourceType,
            sourceRefId: sourceRefId,
            sourceRefName: sourceRefName,
            rewardTitleName: rewardTitleName,
            earnedAt: "2026-09-01 12:00:00"
        )
    }

    private func notification(
        id: Int64,
        type: String = "season_rollover",
        isRead: Bool = false,
        createdAt: String = "2026-07-03 10:00:00",
        data: String = "{}"
    ) -> NotificationInfo {
        NotificationInfo(
            id: id,
            type: type,
            title: "赛季结算",
            content: "S3 已结算",
            data: data,
            isRead: isRead,
            createdAt: createdAt
        )
    }
}
