import XCTest
@testable import iOS

/// 迁移自 `uniapp/tests/challenge-time.test.mjs`、`challenge-view.test.mjs`、`challenge-entry.test.mjs`。
@MainActor
final class ChallengeRulesTests: XCTestCase {
    // MARK: - 时间

    func testParseServerTimeReadsBeijingTimeWithoutDeviceTimezone() {
        XCTAssertEqual(
            ChallengeRules.parseServerTime("2026-08-25 15:20:30"),
            ChallengeRules.DayParts(year: 2026, month: 8, day: 25, hour: 15, minute: 20)
        )
        XCTAssertNil(ChallengeRules.parseServerTime(""))
        XCTAssertNil(ChallengeRules.parseServerTime("garbage"))
    }

    func testComputeScheduledDateShiftsByDayOffset() {
        XCTAssertEqual(ChallengeRules.computeScheduledDate(serverTime: "2026-08-25 15:20:00", dayOffset: 0), "2026-08-25")
        XCTAssertEqual(ChallengeRules.computeScheduledDate(serverTime: "2026-08-25 15:20:00", dayOffset: 1), "2026-08-26")
        XCTAssertEqual(ChallengeRules.computeScheduledDate(serverTime: "2026-08-31 23:00:00", dayOffset: 2), "2026-09-02")
        XCTAssertEqual(ChallengeRules.computeScheduledDate(serverTime: "", dayOffset: 0), "")
    }

    func testAvailableStartHoursLimitsTodayRemainingHours() {
        XCTAssertEqual(ChallengeRules.availableStartHours(serverTime: "2026-08-25 15:20:00", dayOffset: 0).first, 15)
        XCTAssertEqual(ChallengeRules.availableStartHours(serverTime: "2026-08-25 15:20:00", dayOffset: 1).count, 24)
        XCTAssertEqual(ChallengeRules.availableStartHours(serverTime: "", dayOffset: 0).count, 24)
    }

    func testDefaultSlotPicksNextFullHourAndRollsToTomorrow() {
        XCTAssertEqual(
            ChallengeRules.defaultSlot(serverTime: "2026-08-25 15:20:00"),
            ChallengeRules.Slot(dayOffset: 0, startHour: 16, endHour: 17)
        )
        XCTAssertEqual(
            ChallengeRules.defaultSlot(serverTime: "2026-08-25 16:00:00"),
            ChallengeRules.Slot(dayOffset: 0, startHour: 16, endHour: 17)
        )
        XCTAssertEqual(
            ChallengeRules.defaultSlot(serverTime: "2026-08-25 23:20:00"),
            ChallengeRules.Slot(dayOffset: 1, startHour: 0, endHour: 1)
        )
    }

    func testEndHourOptionsMustBeLaterThanStartAndCapAt24() {
        XCTAssertEqual(ChallengeRules.endHourOptions(startHour: 22), [23, 24])
        XCTAssertEqual(ChallengeRules.endHourOptions(startHour: 0).first, 1)
        XCTAssertEqual(ChallengeRules.endHourOptions(startHour: 0).count, 24)
        XCTAssertEqual(ChallengeRules.endHourOptions(startHour: 24), [])
    }

    // MARK: - 展示

    func testDayLabelFollowsServerDateAcrossMidnightAndYears() {
        XCTAssertEqual(ChallengeRules.dayLabel("2026-08-25", serverTime: "2026-08-25 21:00:00"), "今天")
        XCTAssertEqual(ChallengeRules.dayLabel("2026-08-25", serverTime: "2026-08-26 00:30:00"), "昨天")
        XCTAssertEqual(ChallengeRules.dayLabel("2026-08-26", serverTime: "2026-08-25 12:00:00"), "明天")
        XCTAssertEqual(ChallengeRules.dayLabel("2026-08-27", serverTime: "2026-08-25 12:00:00"), "8月27日")
        XCTAssertEqual(ChallengeRules.dayLabel("2026-08-25", serverTime: "2026-08-27 12:00:00"), "8月25日")
        XCTAssertEqual(ChallengeRules.dayLabel("2025-12-31", serverTime: "2027-01-02 12:00:00"), "2025年12月31日")
    }

    func testScheduleLineComposesDayAndHourRange() throws {
        let today = try XCTUnwrap(ChallengeInfo.fixture(
            scheduledDate: "2026-08-25", startHour: 16, endHour: 17
        ))
        XCTAssertEqual(ChallengeRules.schedule(today, serverTime: "2026-08-25 12:00:00"), "今天16点—17点")

        let yesterday = try XCTUnwrap(ChallengeInfo.fixture(
            scheduledDate: "2026-08-25", startHour: 23, endHour: 24
        ))
        XCTAssertEqual(ChallengeRules.schedule(yesterday, serverTime: "2026-08-26 00:30:00"), "昨天23点—24点")
    }

    func testExpiryCopyUsesNextMorningSevenAndExpiredAfter() throws {
        let tomorrow = try XCTUnwrap(ChallengeInfo.fixture(expiresAt: "2026-08-26 07:00:00"))
        XCTAssertEqual(ChallengeRules.expiry(tomorrow, serverTime: "2026-08-25 21:00:00"), "明天早晨7点失效")
        XCTAssertEqual(ChallengeRules.expiry(tomorrow, serverTime: "2026-08-26 00:30:00"), "今天早晨7点失效")
        XCTAssertEqual(ChallengeRules.expiry(tomorrow, serverTime: "2026-08-26 07:00:00"), "已失效")
    }

    func testQuickCardFlagsChallengesWaitingOnMe() throws {
        let serverTime = "2026-08-25 12:00:00"
        let opponentEntered = try XCTUnwrap(ChallengeInfo.fixture(
            id: 5,
            status: 1,
            fromUserId: 7,
            fromNickname: "阿杰",
            toUserId: 18,
            waitingUserId: 7,
            gameType: 3,
            matchMode: "ranked",
            scheduledDate: "2026-08-25",
            startHour: 16,
            endHour: 17
        ))
        let opponentCard = try XCTUnwrap(ChallengeRules.quickCard(opponentEntered, userId: 18, serverTime: serverTime))
        XCTAssertEqual(opponentCard.title, "对方已进入，等你开始")
        XCTAssertTrue(opponentCard.needsAction)
        XCTAssertEqual(opponentCard.subText, "阿杰 · 中式八球 · 排位 · 今天16点—17点")
        XCTAssertEqual(opponentCard.actionLabel, "进入")

        let sent = try XCTUnwrap(ChallengeInfo.fixture(
            id: 6, status: 0, fromUserId: 18, toNickname: "球友 B", gameType: 2, matchMode: "practice", scheduledDate: nil
        ))
        let sentCard = try XCTUnwrap(ChallengeRules.quickCard(sent, userId: 18, serverTime: serverTime))
        XCTAssertEqual(sentCard.title, "等待对方回应")
        XCTAssertFalse(sentCard.needsAction)
        XCTAssertEqual(sentCard.subText, "球友 B · 九球追分 · 练习")

        let received = try XCTUnwrap(ChallengeInfo.fixture(
            id: 7, status: 0, fromUserId: 9, fromNickname: "老王", scheduledDate: nil
        ))
        let receivedCard = try XCTUnwrap(ChallengeRules.quickCard(received, userId: 18, serverTime: serverTime))
        XCTAssertEqual(receivedCard.title, "收到约球邀请")
        XCTAssertTrue(receivedCard.needsAction)

        let started = try XCTUnwrap(ChallengeInfo.fixture(
            id: 8, status: 6, matchId: 42, fromUserId: 18, scheduledDate: nil
        ))
        let startedCard = try XCTUnwrap(ChallengeRules.quickCard(started, userId: 18, serverTime: serverTime))
        XCTAssertEqual(startedCard.matchId, 42)
        XCTAssertEqual(startedCard.challengeId, 8)
        XCTAssertEqual(startedCard.actionLabel, "继续比赛")

        XCTAssertNil(ChallengeRules.quickCard(nil, userId: 18, serverTime: serverTime))
        let cancelled = try XCTUnwrap(ChallengeInfo.fixture(id: 9, status: 4))
        XCTAssertNil(ChallengeRules.quickCard(cancelled, userId: 18, serverTime: serverTime))
    }

    // MARK: - 入口

    func testBuildPayloadUsesCanonicalTargetUserId() {
        XCTAssertEqual(
            ChallengeRules.buildPayload(targetUserId: 88, gameType: 2, message: "今晚来一局"),
            ChallengeEntryPayload(toUserId: 88, gameType: 2, message: "今晚来一局")
        )
        XCTAssertEqual(
            ChallengeRules.buildPayload(targetUserId: 0, gameType: 1, message: ""),
            ChallengeEntryPayload(toUserId: 0, gameType: 1, message: "")
        )
    }

    func testOpponentMapsSentAndReceivedDirections() throws {
        let sent = try XCTUnwrap(ChallengeInfo.fixture(
            id: 1,
            fromUserId: 18,
            fromNickname: "我",
            toUserId: 29,
            toNickname: "球友A",
            toAvatar: "a.png"
        ))
        XCTAssertEqual(
            ChallengeRules.opponent(sent, userId: 18),
            ChallengeListItemView(direction: .sent, opponentId: 29, nickname: "球友A", avatar: "a.png")
        )

        let received = try XCTUnwrap(ChallengeInfo.fixture(
            id: 2,
            fromUserId: 41,
            fromNickname: "球友B",
            fromAvatar: "b.png",
            toUserId: 18,
            toNickname: "我"
        ))
        XCTAssertEqual(
            ChallengeRules.opponent(received, userId: 18),
            ChallengeListItemView(direction: .received, opponentId: 41, nickname: "球友B", avatar: "b.png")
        )
    }

    func testEntryActionsMatchUniAppRules() throws {
        let linked = try XCTUnwrap(ChallengeInfo.fixture(
            id: 77, status: 1, matchId: 9001, toUserId: 29, gameType: 2
        ))
        XCTAssertEqual(ChallengeRules.resolveAction(linked), ChallengeEntryAction(kind: .viewMatch, label: "查看对局", matchId: 9001))
        let offline = try XCTUnwrap(ChallengeInfo.fixture(id: 77, status: 1, matchId: 0, toUserId: 29, gameType: 2))
        let opponent = ChallengeListItemView(direction: .sent, opponentId: 29, nickname: "球友A", avatar: "a.png")
        XCTAssertNil(ChallengeRules.buildStartContext(linked, opponent: opponent))
        XCTAssertEqual(ChallengeRules.resolveAction(offline), ChallengeEntryAction(kind: .offlineStart, label: "线下扫码开局", matchId: 0))
        XCTAssertEqual(
            ChallengeRules.buildStartContext(offline, opponent: opponent),
            ChallengeStartContext(challengeId: 77, opponentId: 29, opponentName: "球友A", opponentAvatar: "a.png", gameType: 2)
        )
    }

    func testStatusLabelsAndTerminalStates() throws {
        let pending = try XCTUnwrap(ChallengeInfo.fixture(id: 1, status: 0, fromUserId: 29, toUserId: 18))
        XCTAssertEqual(ChallengeRules.statusLabel(pending, userId: 18), "待你回应")
        XCTAssertEqual(ChallengeRules.statusLabel(pending, userId: 29), "等待回应")

        let waiting = try XCTUnwrap(ChallengeInfo.fixture(id: 2, status: 1, waitingUserId: 18))
        XCTAssertEqual(ChallengeRules.statusLabel(waiting, userId: 18), "等待对方进入")
        let opponentWaiting = try XCTUnwrap(ChallengeInfo.fixture(id: 3, status: 1, waitingUserId: 29))
        XCTAssertEqual(ChallengeRules.statusLabel(opponentWaiting, userId: 18), "对方已进入")

        let terminal = try XCTUnwrap(ChallengeInfo.fixture(id: 4, status: 7))
        XCTAssertTrue(ChallengeRules.isTerminal(terminal))
        XCTAssertFalse(ChallengeRules.isTerminal(pending))
    }
}

private extension ChallengeInfo {
    /// 用夹具响应解码出模型，避免手写长初始化列表。
    static func fixture(
        id: Int64 = 1,
        status: Int = 0,
        matchId: Int64 = 0,
        fromUserId: Int64 = 18,
        fromNickname: String = "我",
        fromAvatar: String = "",
        toUserId: Int64 = 29,
        toNickname: String = "球友A",
        toAvatar: String = "",
        waitingUserId: Int64? = nil,
        gameType: Int = 3,
        matchMode: String = "ranked",
        message: String = "",
        scheduledDate: String? = "2026-08-25",
        startHour: Int? = 16,
        endHour: Int? = 17,
        expiresAt: String = "2026-08-26 07:00:00"
    ) -> ChallengeInfo? {
        let object = ChallengeFixtures.challenge(
            id: id,
            status: status,
            matchId: matchId,
            fromUserId: fromUserId,
            fromNickname: fromNickname,
            fromAvatar: fromAvatar,
            toUserId: toUserId,
            toNickname: toNickname,
            toAvatar: toAvatar,
            gameType: gameType,
            matchMode: matchMode,
            message: message,
            waitingUserId: waitingUserId,
            scheduledDate: scheduledDate,
            startHour: startHour,
            endHour: endHour,
            expiresAt: expiresAt
        )
        guard let data = try? JSONSerialization.data(withJSONObject: object) else { return nil }
        let decoder = JSONDecoder()
        decoder.keyDecodingStrategy = .convertFromSnakeCase
        return try? decoder.decode(ChallengeInfo.self, from: data)
    }
}
