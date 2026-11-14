import XCTest
@testable import iOS

/// 首页纯规则测试，用例搬自 `uniapp/tests/home-index.test.mjs`、
/// `match-core-flow.test.mjs` 与 `challenge-view.test.mjs` 中首页用到的部分。
final class HomeRulesTests: XCTestCase {
    // MARK: - 问候语

    func testGreetingFollowsLocalTimeOfDay() {
        XCTAssertEqual(HomeRules.greeting(hour: 7), "早上好")
        XCTAssertEqual(HomeRules.greeting(hour: 12), "中午好")
        XCTAssertEqual(HomeRules.greeting(hour: 15), "下午好")
        XCTAssertEqual(HomeRules.greeting(hour: 21), "晚上好")
        XCTAssertEqual(HomeRules.greeting(hour: 2), "晚上好")
    }

    // MARK: - 时长

    func testFormatDurationCoversSecondsMinutesHoursAndDays() {
        XCTAssertEqual(HomeRules.formatDuration(0), "0分")
        XCTAssertEqual(HomeRules.formatDuration(45), "45秒")
        XCTAssertEqual(HomeRules.formatDuration(720), "12分")
        XCTAssertEqual(HomeRules.formatDuration(3660), "1小时1分")
        XCTAssertEqual(HomeRules.formatDuration(7200), "2小时")
        XCTAssertEqual(HomeRules.formatDuration(90_000), "1天1小时")
        XCTAssertEqual(HomeRules.formatDuration(172_800), "2天")
    }

    // MARK: - 我的对局

    func testOngoingMatchCardKeepsFixedPlayerOrderAndMarksViewer() {
        let card = HomeRules.ongoingMatchCard(
            HomeFixtures.currentMatchModel(HomeFixtures.currentMatch(
                id: 31,
                viewerRole: "player2",
                durationSeconds: 720,
                myScore: 3,
                opponentScore: 5,
                player1Name: "选手甲",
                player2Name: "选手乙"
            )),
            currentUser: HomeFixtures.decode(
                AuthFixtures.userInfo(id: 22, nickname: "选手乙"),
                as: UserInfo.self
            )
        )

        XCTAssertEqual(card?.id, 31)
        XCTAssertEqual(card?.statusText, "进行中")
        XCTAssertEqual(card?.metaText, "中式八球 · 排位")
        XCTAssertEqual(card?.durationText, "12分")
        XCTAssertEqual(card?.actionLabel, "继续对局")
        XCTAssertEqual(card?.players.map { [$0.name, String($0.score), String($0.isMe)] }, [
            ["选手甲", "5", "false"],
            ["选手乙", "3", "true"],
        ])
    }

    func testOngoingMatchCardCoversRefereeAndPendingConfirmation() {
        let referee = HomeRules.ongoingMatchCard(
            HomeFixtures.currentMatchModel(HomeFixtures.currentMatch(
                id: 7,
                viewerRole: "referee",
                myScore: 2,
                opponentScore: 1
            )),
            currentUser: HomeFixtures.decode(AuthFixtures.userInfo(id: 99, nickname: "裁判"), as: UserInfo.self)
        )
        XCTAssertEqual(referee?.statusText, "执裁中")
        XCTAssertEqual(referee?.actionLabel, "进入裁判记分")
        XCTAssertEqual(referee?.players.map(\.isMe), [false, false])

        let pending = HomeRules.ongoingMatchCard(
            HomeFixtures.currentMatchModel(HomeFixtures.currentMatch(
                id: 8,
                matchMode: "practice",
                finishState: "pending_confirmation",
                player1Name: "",
                player2Name: "",
                opponentName: "",
                gameTypeName: ""
            )),
            currentUser: nil
        )
        XCTAssertEqual(pending?.isPending, true)
        XCTAssertEqual(pending?.statusText, "待确认结束")
        XCTAssertEqual(pending?.actionLabel, "处理结束确认")
        XCTAssertEqual(pending?.metaText, "台球 · 练习")

        XCTAssertNil(HomeRules.ongoingMatchCard(nil, currentUser: nil))
    }

    func testOngoingMatchCardFillsMissingNamesByViewerRole() {
        let player2 = HomeRules.ongoingMatchCard(
            HomeFixtures.currentMatchModel(HomeFixtures.currentMatch(
                viewerRole: "player2",
                player1Name: "",
                player2Name: "",
                opponentName: ""
            )),
            currentUser: HomeFixtures.decode(AuthFixtures.userInfo(id: 22, nickname: "我"), as: UserInfo.self)
        )
        XCTAssertEqual(player2?.players.map(\.name), ["玩家1", "我"])

        let player1 = HomeRules.ongoingMatchCard(
            HomeFixtures.currentMatchModel(HomeFixtures.currentMatch(
                viewerRole: "player1",
                player1Name: "",
                player2Name: "",
                opponentName: "老王"
            )),
            currentUser: HomeFixtures.decode(AuthFixtures.userInfo(id: 1, nickname: "甲"), as: UserInfo.self)
        )
        XCTAssertEqual(player1?.players.map(\.name), ["甲", "老王"])
    }

    // MARK: - 约球

    func testChallengeQuickCardFlagsChallengesThatWaitOnMe() {
        let card = HomeRules.challengeQuickCard(
            HomeFixtures.challengeModel(HomeFixtures.challenge(
                id: 5,
                fromUserId: 7,
                fromNickname: "阿杰",
                toUserId: 18,
                status: 1,
                waitingUserId: 7,
                scheduledDate: "2026-08-25"
            )),
            userId: 18,
            serverTime: "2026-08-25 12:00:00"
        )

        XCTAssertEqual(card?.title, "对方已进入，等你开始")
        XCTAssertEqual(card?.needsAction, true)
        XCTAssertEqual(card?.subText, "阿杰 · 中式八球 · 排位 · 今天16点—17点")
        XCTAssertEqual(card?.actionLabel, "进入")
        XCTAssertEqual(card?.matchId, 0)
        XCTAssertEqual(card?.challengeId, 5)
    }

    func testChallengeQuickCardCoversSentReceivedAndStarted() {
        let sent = HomeRules.challengeQuickCard(
            HomeFixtures.challengeModel(HomeFixtures.challenge(
                id: 6,
                fromUserId: 18,
                toNickname: "球友 B",
                status: 0,
                scheduledDate: nil,
                startHour: nil,
                endHour: nil,
                matchMode: "practice",
                gameType: 2
            )),
            userId: 18,
            serverTime: "2026-08-25 12:00:00"
        )
        XCTAssertEqual(sent?.title, "等待对方回应")
        XCTAssertEqual(sent?.needsAction, false)
        XCTAssertEqual(sent?.subText, "球友 B · 九球追分 · 练习")

        let received = HomeRules.challengeQuickCard(
            HomeFixtures.challengeModel(HomeFixtures.challenge(
                id: 7,
                fromUserId: 9,
                fromNickname: "老王",
                status: 0,
                scheduledDate: nil,
                startHour: nil,
                endHour: nil
            )),
            userId: 18,
            serverTime: "2026-08-25 12:00:00"
        )
        XCTAssertEqual(received?.title, "收到约球邀请")
        XCTAssertEqual(received?.needsAction, true)

        let started = HomeRules.challengeQuickCard(
            HomeFixtures.challengeModel(HomeFixtures.challenge(
                id: 8,
                fromUserId: 18,
                status: 6,
                matchId: 42,
                scheduledDate: nil,
                startHour: nil,
                endHour: nil
            )),
            userId: 18,
            serverTime: "2026-08-25 12:00:00"
        )
        XCTAssertEqual(started?.matchId, 42)
        XCTAssertEqual(started?.challengeId, 8)
        XCTAssertEqual(started?.actionLabel, "继续比赛")

        XCTAssertNil(HomeRules.challengeQuickCard(nil, userId: 18, serverTime: ""))
        XCTAssertNil(HomeRules.challengeQuickCard(
            HomeFixtures.challengeModel(HomeFixtures.challenge(id: 9, status: 4)),
            userId: 18,
            serverTime: ""
        ))
    }

    func testChallengeDayLabelFollowsServerDateAcrossMidnightAndYears() {
        XCTAssertEqual(HomeRules.challengeDayLabel("2026-08-25", serverTime: "2026-08-25 21:00:00"), "今天")
        XCTAssertEqual(HomeRules.challengeDayLabel("2026-08-25", serverTime: "2026-08-26 00:30:00"), "昨天")
        XCTAssertEqual(HomeRules.challengeDayLabel("2026-08-26", serverTime: "2026-08-25 12:00:00"), "明天")
        XCTAssertEqual(HomeRules.challengeDayLabel("2026-08-27", serverTime: "2026-08-25 12:00:00"), "8月27日")
        XCTAssertEqual(HomeRules.challengeDayLabel("2026-08-25", serverTime: "2026-08-27 12:00:00"), "8月25日")
        XCTAssertEqual(HomeRules.challengeDayLabel("2025-12-31", serverTime: "2027-01-02 12:00:00"), "2025年12月31日")
    }

    func testChallengeScheduleComposesDayAndHourRange() {
        let today = HomeFixtures.challengeModel(HomeFixtures.challenge(
            scheduledDate: "2026-08-25",
            startHour: 16,
            endHour: 17
        ))
        XCTAssertEqual(HomeRules.challengeSchedule(today, serverTime: "2026-08-25 12:00:00"), "今天16点—17点")

        let yesterday = HomeFixtures.challengeModel(HomeFixtures.challenge(
            scheduledDate: "2026-08-25",
            startHour: 23,
            endHour: 24
        ))
        XCTAssertEqual(HomeRules.challengeSchedule(yesterday, serverTime: "2026-08-26 00:30:00"), "昨天23点—24点")
    }

    // MARK: - 排行榜

    func testMyRankingLineCoversRankedAndUnrankedViewers() {
        let ranked = LeaderboardItem(
            rank: 128, userId: 1, nickname: "球手", avatar: "",
            rankLevel: 3, rankName: "黄金", rankScore: 1320, winRate: 50, vipLevel: 0
        )
        XCTAssertEqual(HomeRules.myRankingText(ranked), "第 128 名 · 黄金 · 1320 分")
        XCTAssertEqual(
            HomeRules.myRankingText(RankingFixtures.item(rank: 0, userId: 1)),
            "暂未上榜，打一场排位赛开始积分"
        )
        XCTAssertEqual(HomeRules.myRankingText(nil), "暂未上榜，打一场排位赛开始积分")
    }

    // MARK: - 附近球房

    func testNearbyVenueParamsUseFiveKmAndThreeCardLimit() {
        XCTAssertEqual(HomeRules.nearbyRadiusMeters, 5000)
        XCTAssertEqual(HomeRules.nearbyLimit, 3)
        XCTAssertEqual(
            HomeRules.nearbyVenueParams(Coordinate(latitude: 22.5431, longitude: 114.0579)),
            HomeNearbyVenueParams(latitude: 22.5431, longitude: 114.0579, radius: 5000, limit: 3)
        )
    }

    func testNearbyVenueParamsRejectMissingCoordinates() {
        XCTAssertNil(HomeRules.nearbyVenueParams(Coordinate(latitude: 0, longitude: 114.0579)))
        XCTAssertNil(HomeRules.nearbyVenueParams(Coordinate(latitude: 22.5431, longitude: 0)))
        XCTAssertNil(HomeRules.nearbyVenueParams(nil))
    }

    func testVenueDistanceKeepsHomeCardsCompact() {
        XCTAssertEqual(HomeRules.venueDistance(860), "860m")
        XCTAssertEqual(HomeRules.venueDistance(1260), "1.3km")
        XCTAssertEqual(HomeRules.venueDistance(0), "")
        XCTAssertEqual(HomeRules.venueDistance(nil), "")
    }

    func testVenueEmptyActionMentionsMemberRewardOnlyWhenClaimable() {
        XCTAssertEqual(
            HomeRules.venueEmptyAction(rewardClaimable: true),
            HomeVenueEmptyAction(
                text: "添加球馆领取VIP",
                desc: "你可以提交常玩球馆，审核通过后VIP会自动到账。"
            )
        )
        XCTAssertEqual(
            HomeRules.venueEmptyAction(rewardClaimable: false),
            HomeVenueEmptyAction(
                text: "添加附近球馆",
                desc: "你可以提交常去球馆，审核通过后会展示给附近球友。"
            )
        )
    }

    func testNearbyRelocatesOnlyFirstTimeAfterIntervalOrForced() {
        let now = Date(timeIntervalSince1970: 1_000_000)
        XCTAssertTrue(HomeRules.shouldReloadNearby(force: false, lastAttemptAt: nil, now: now))
        XCTAssertFalse(HomeRules.shouldReloadNearby(
            force: false,
            lastAttemptAt: now.addingTimeInterval(-1),
            now: now
        ))
        XCTAssertTrue(HomeRules.shouldReloadNearby(
            force: false,
            lastAttemptAt: now.addingTimeInterval(-HomeRules.nearbyRefreshInterval),
            now: now
        ))
        XCTAssertTrue(HomeRules.shouldReloadNearby(force: true, lastAttemptAt: now, now: now))
    }
}
