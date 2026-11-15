import XCTest
@testable import iOS

/// 发起约球页面模型：时间基准、默认时段、校验与提交结果。
@MainActor
final class ChallengeComposeModelTests: XCTestCase {
    private func bootstrap(currentChallenge: [String: Any]? = nil) -> [String: Any] {
        [
            "success": true,
            "current_match": NSNull(),
            "current_challenge": currentChallenge ?? NSNull(),
            "received_pending_challenge_count": 0,
            "server_time": "2026-08-25 15:20:00",
            "unread_count": 0,
            "pending_friend_request_count": 0,
            "latest_season_rollover": NSNull(),
            "availability": ActivityFixtures.allAvailable,
        ]
    }

    private func makeModel(
        _ server: MockHTTPServer,
        target: ChallengeComposeTarget? = ChallengeComposeTarget(userId: 29, nickname: "球友A", avatar: ""),
        now: Date? = nil,
        bootstrapJSON: [String: Any]? = nil
    ) async -> (ChallengeComposeModel, ActivityStore, SessionStore) {
        server.stub("GET", "/api/user/bootstrap", .json(bootstrapJSON ?? bootstrap()))
        let session = await makeRestoredAccountSession(server)
        let activity = ActivityStore(session: session)
        // 应用内 ActivityStore 在启动时已同步身份；测试补齐同样前提。
        activity.syncIdentity()
        let model = target == nil
            ? ChallengeComposeModel(session: session, activity: activity, now: { now ?? Date() })
            : ChallengeComposeModel(session: session, activity: activity, target: target, now: { now ?? Date() })
        return (model, activity, session)
    }

    func testTargetPrefillsGameTypeAndMatchMode() async {
        let server = MockHTTPServer()
        var target = ChallengeComposeTarget(userId: 29, nickname: "球友A", avatar: "")
        target.gameType = 2
        target.matchMode = "practice"
        let (model, _, _) = await makeModel(server, target: target)

        XCTAssertEqual(model.gameType, 2)
        XCTAssertEqual(model.matchMode, "practice")
        XCTAssertEqual(model.opponentName, "球友A")
        XCTAssertTrue(model.canSubmit)
    }

    func testAppearLoadsServerTimeAndDefaultSlot() async {
        let server = MockHTTPServer()
        server.stub("GET", "/api/challenge/summary", .json(ChallengeFixtures.summary(serverTime: "2026-08-25 15:20:00")))
        let (model, _, _) = await makeModel(server)

        await model.onAppear()

        XCTAssertEqual(model.serverTime, "2026-08-25 15:20:00")
        XCTAssertEqual(model.scheduledDate, "2026-08-25")
        XCTAssertEqual(model.startHour, 16)
        XCTAssertEqual(model.endHour, 17)
        XCTAssertTrue(model.dayOptions.contains(ChallengeRules.DayOption(value: 0, label: "今天")))
    }

    func testAppearFallsBackToBeijingTimeWhenSummaryFails() async throws {
        let server = MockHTTPServer()
        server.stub("GET", "/api/challenge/summary", .raw("down", status: 503))
        let now = try XCTUnwrap(ISO8601DateFormatter().date(from: "2026-08-25T07:20:00Z"))
        let (model, _, _) = await makeModel(server, now: now)

        await model.onAppear()

        XCTAssertEqual(model.serverTime, "2026-08-25 15:20:00")
        XCTAssertEqual(model.scheduledDate, "2026-08-25")
        XCTAssertEqual(model.startHour, 16)
    }

    func testChooseDayKeepsDateAndHoursConsistent() async {
        let server = MockHTTPServer()
        server.stub("GET", "/api/challenge/summary", .json(ChallengeFixtures.summary(serverTime: "2026-08-25 15:20:00")))
        let (model, _, _) = await makeModel(server)
        await model.onAppear()

        model.chooseDay(1)

        XCTAssertEqual(model.scheduledDate, "2026-08-26")
        XCTAssertEqual(model.activeDayOffset, 1)
        XCTAssertEqual(model.startHour, 0)
        XCTAssertEqual(model.endHour, 1)
    }

    func testMessageTruncatesAtTwoHundredCharacters() async {
        let server = MockHTTPServer()
        let (model, _, _) = await makeModel(server)

        model.message = String(repeating: "球", count: 205)

        XCTAssertEqual(model.message.count, ChallengeComposeModel.maxMessageLength)
    }

    func testReplaceOpponentKeepsFormFields() async {
        let server = MockHTTPServer()
        let (model, _, _) = await makeModel(server, target: nil)
        model.gameType = 2
        model.matchMode = "practice"
        model.message = "保留附言"

        model.replaceOpponent(ChallengeComposeTarget(userId: 31, nickname: "球友B", avatar: ""))

        XCTAssertEqual(model.opponentName, "球友B")
        XCTAssertEqual(model.gameType, 2)
        XCTAssertEqual(model.matchMode, "practice")
        XCTAssertEqual(model.message, "保留附言")
    }

    func testSubmitWithoutOpponentFailsValidation() async {
        let server = MockHTTPServer()
        server.stub("GET", "/api/challenge/summary", .json(ChallengeFixtures.summary(serverTime: "2026-08-25 15:20:00")))
        let (model, _, _) = await makeModel(server, target: nil)
        await model.onAppear()

        let outcome = await model.submit()

        XCTAssertEqual(outcome, .failure("请先选择球友"))
        XCTAssertEqual(model.validationMessage, "请先选择球友")
        XCTAssertTrue(server.requests("/api/challenge/send").isEmpty)
    }

    func testSubmitSuccessSendsSelectionAndMarksActivityDirty() async {
        let server = MockHTTPServer()
        server.stub("GET", "/api/challenge/summary", .json(ChallengeFixtures.summary(serverTime: "2026-08-25 15:20:00")))
        server.stub("POST", "/api/challenge/send", .json(ChallengeFixtures.sendResponse(challengeId: 77)))
        let (model, activity, _) = await makeModel(server)
        await model.onAppear()
        let dirtyBefore = activity.dirtySeq
        model.message = "今晚来一局"

        let outcome = await model.submit()

        XCTAssertEqual(outcome, .success(challengeId: 77))
        let body = server.requests("/api/challenge/send").first?.bodyJSON
        XCTAssertEqual(body?["to_user_id"] as? Int, 29)
        XCTAssertEqual(body?["game_type"] as? Int, 3)
        XCTAssertEqual(body?["match_mode"] as? String, "ranked")
        XCTAssertEqual(body?["message"] as? String, "今晚来一局")
        XCTAssertEqual(body?["scheduled_date"] as? String, "2026-08-25")
        XCTAssertEqual(body?["start_hour"] as? Int, 16)
        XCTAssertEqual(body?["end_hour"] as? Int, 17)
        XCTAssertGreaterThan(activity.dirtySeq, dirtyBefore)
    }

    func testSubmitPendingConflictReturnsOwnPendingChallengeId() async {
        let server = MockHTTPServer()
        server.stub("GET", "/api/challenge/summary", .json(ChallengeFixtures.summary(serverTime: "2026-08-25 15:20:00")))
        server.stub("POST", "/api/challenge/send", .json(ChallengeFixtures.businessFailure(
            "你发出的约球还在等待回应，请先处理",
            pendingChallengeId: 55
        )))
        let (model, _, _) = await makeModel(server)
        await model.onAppear()

        let outcome = await model.submit()

        XCTAssertEqual(
            outcome,
            .pendingConflict(message: "你发出的约球还在等待回应，请先处理", challengeId: 55)
        )
    }

    func testSubmitFailureReturnsServerMessage() async {
        let server = MockHTTPServer()
        server.stub("GET", "/api/challenge/summary", .json(ChallengeFixtures.summary(serverTime: "2026-08-25 15:20:00")))
        server.stub("POST", "/api/challenge/send", .json(ChallengeFixtures.businessFailure("对方已有未结束的约球，请稍后再试")))
        let (model, _, _) = await makeModel(server)
        await model.onAppear()

        let outcome = await model.submit()

        XCTAssertEqual(outcome, .failure("对方已有未结束的约球，请稍后再试"))
    }
}
