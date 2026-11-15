import XCTest
@testable import iOS

/// 等待对手页面模型：详情读取、状态文案、接受/进入/退出等写操作。
@MainActor
final class ChallengeWaitingModelTests: XCTestCase {
    private func bootstrap() -> [String: Any] {
        [
            "success": true,
            "current_match": NSNull(),
            "current_challenge": NSNull(),
            "received_pending_challenge_count": 0,
            "server_time": "2026-08-25 12:00:00",
            "unread_count": 0,
            "pending_friend_request_count": 0,
            "latest_season_rollover": NSNull(),
            "availability": ActivityFixtures.allAvailable,
        ]
    }

    private func makeModel(
        _ server: MockHTTPServer,
        challengeId: Int64 = 7,
        sleep: @escaping (TimeInterval) async throws -> Void = { _ in
            try await Task.sleep(for: .seconds(3600))
        }
    ) async -> (ChallengeWaitingModel, ActivityStore, SessionStore) {
        server.stub("GET", "/api/user/bootstrap", .json(bootstrap()))
        let session = await makeRestoredAccountSession(server)
        let activity = ActivityStore(session: session)
        let model = ChallengeWaitingModel(
            session: session,
            activity: activity,
            challengeId: challengeId,
            sleep: sleep
        )
        return (model, activity, session)
    }

    func testLoadReadsDetailAndServerTime() async {
        let server = MockHTTPServer()
        server.stub("GET", "/api/challenge/detail", .json(ChallengeFixtures.detail(ChallengeFixtures.challenge(
            id: 7, status: 1, fromUserId: 1, fromNickname: "测试用户", toUserId: 29, toNickname: "球友A"
        ))))
        server.stub("GET", "/api/challenge/summary", .json(ChallengeFixtures.summary(serverTime: "2026-08-25 12:00:00")))
        let (model, _, _) = await makeModel(server)

        await model.load()

        XCTAssertEqual(model.phase, .loaded)
        XCTAssertEqual(model.challenge?.id, 7)
        XCTAssertEqual(model.serverTime, "2026-08-25 12:00:00")
        XCTAssertEqual(model.opponentUserId, 29)
        XCTAssertEqual(model.scheduleLabel, "今天16点—17点")
    }

    func testFirstLoadFailureShowsFailureInsteadOfEmpty() async {
        let server = MockHTTPServer()
        server.stub("GET", "/api/challenge/detail", .raw("down", status: 503))
        let (model, _, _) = await makeModel(server)

        await model.load()

        XCTAssertEqual(model.phase, .failed)
        XCTAssertNil(model.challenge)
    }

    func testMetaHintFollowsStatusAndExpiry() async {
        let server = MockHTTPServer()
        server.stub("GET", "/api/challenge/summary", .json(ChallengeFixtures.summary(serverTime: "2026-08-25 21:00:00")))
        server.stub("GET", "/api/challenge/detail", .json(ChallengeFixtures.detail(ChallengeFixtures.challenge(
            id: 7, status: 1, expiresAt: "2026-08-26 07:00:00"
        ))))
        let (waiting, _, _) = await makeModel(server)
        await waiting.load()

        XCTAssertFalse(waiting.expiredNow)
        XCTAssertEqual(waiting.metaHint, "返回或锁屏后仍会等待；暂时不打，可以退出等待")

        let expiredServer = MockHTTPServer()
        expiredServer.stub("GET", "/api/challenge/summary", .json(ChallengeFixtures.summary(serverTime: "2026-08-26 08:00:00")))
        expiredServer.stub("GET", "/api/challenge/detail", .json(ChallengeFixtures.detail(ChallengeFixtures.challenge(
            id: 8, status: 1, expiresAt: "2026-08-26 07:00:00"
        ))))
        let (expired, _, _) = await makeModel(expiredServer)
        await expired.load()

        XCTAssertTrue(expired.expiredNow)
        XCTAssertEqual(expired.metaHint, "已失效")

        let pendingServer = MockHTTPServer()
        pendingServer.stub("GET", "/api/challenge/summary", .json(ChallengeFixtures.summary(serverTime: "2026-08-25 21:00:00")))
        pendingServer.stub("GET", "/api/challenge/detail", .json(ChallengeFixtures.detail(ChallengeFixtures.challenge(
            id: 9, status: 0, expiresAt: "2026-08-26 07:00:00"
        ))))
        let (pending, _, _) = await makeModel(pendingServer)
        await pending.load()

        XCTAssertEqual(pending.metaHint, "明天早晨7点失效")
    }

    func testTerminalStatusCopies() async {
        let server = MockHTTPServer()
        server.stub("GET", "/api/challenge/summary", .json(ChallengeFixtures.summary()))
        server.stub("GET", "/api/challenge/detail", .json(ChallengeFixtures.detail(ChallengeFixtures.challenge(
            id: 7, status: 5
        ))))
        let (model, _, _) = await makeModel(server)
        await model.load()

        XCTAssertEqual(model.statusLabel, "对方已放弃，本次约球已结束")
    }

    func testAcceptNeedsCloseOwnPrompt() async {
        let server = MockHTTPServer()
        server.stub("POST", "/api/challenge/accept", .json(ChallengeFixtures.businessFailure(
            "接受后会自动关闭已发出的邀请",
            needConfirmCloseOwn: true
        )))
        let (model, _, _) = await makeModel(server)

        await model.accept()

        XCTAssertEqual(model.prompt, .closeOwn)
        XCTAssertFalse(model.isActing)
        XCTAssertNil(model.navigation)
    }

    func testAcceptTypeConflictPrompt() async {
        let server = MockHTTPServer()
        server.stub("POST", "/api/challenge/accept", .json(ChallengeFixtures.businessFailure(
            "你们发起的约球类型不同，不能直接匹配",
            typeConflict: true
        )))
        let (model, _, _) = await makeModel(server)

        await model.accept()

        XCTAssertEqual(model.prompt, .typeConflict(message: "你们发起的约球类型不同，不能直接匹配"))
    }

    func testAcceptSuccessMarksActivityDirtyAndRefreshes() async {
        let server = MockHTTPServer()
        server.stub("GET", "/api/challenge/summary", .json(ChallengeFixtures.summary(serverTime: "2026-08-25 12:00:00")))
        server.stub("GET", "/api/challenge/detail", .json(ChallengeFixtures.detail(ChallengeFixtures.challenge(id: 7, status: 1))))
        server.stub("POST", "/api/challenge/accept", .json(ChallengeFixtures.acceptResponse(challengeId: 7)))
        let (model, activity, _) = await makeModel(server)
        let dirtyBefore = activity.dirtySeq

        await model.accept()

        XCTAssertEqual(model.feedback, "已接受，双方进入后正式开局")
        XCTAssertEqual(model.challenge?.status, 1)
        XCTAssertEqual(model.phase, .loaded)
        XCTAssertGreaterThan(activity.dirtySeq, dirtyBefore)
        XCTAssertEqual(server.requestCount("/api/challenge/accept"), 1)
    }

    func testAcceptMatchCreatedNavigatesToMatch() async {
        let server = MockHTTPServer()
        server.stub("POST", "/api/challenge/accept", .json(ChallengeFixtures.acceptResponse(challengeId: 7, matchId: 88)))
        let (model, _, _) = await makeModel(server)

        await model.accept()

        XCTAssertEqual(model.navigation, .match(88))
    }

    func testEnterConfirmEarlyUsesScheduleCopy() async {
        let server = MockHTTPServer()
        server.stub("GET", "/api/challenge/summary", .json(ChallengeFixtures.summary(serverTime: "2026-08-25 12:00:00")))
        server.stub("GET", "/api/challenge/detail", .json(ChallengeFixtures.detail(ChallengeFixtures.challenge(id: 7, status: 1))))
        server.stub("POST", "/api/challenge/enter", .json(ChallengeFixtures.businessFailure(
            "尚未到预约开始时间，确认提前进入？",
            action: "confirm_early"
        )))
        let (model, _, _) = await makeModel(server)
        await model.load()

        await model.enter()

        XCTAssertEqual(model.prompt, .confirmEarly(schedule: "今天16点—17点"))
        XCTAssertNil(model.navigation)
    }

    func testEnterWaitingFeedsBackAndMarksDirty() async {
        let server = MockHTTPServer()
        server.stub("GET", "/api/challenge/summary", .json(ChallengeFixtures.summary(serverTime: "2026-08-25 12:00:00")))
        server.stub("GET", "/api/challenge/detail", .json(ChallengeFixtures.detail(ChallengeFixtures.challenge(
            id: 7, status: 1, waitingUserId: 1
        ))))
        server.stub("POST", "/api/challenge/enter", .json(ChallengeFixtures.enterResponse(action: "waiting")))
        let (model, activity, _) = await makeModel(server)
        await model.load()
        let dirtyBefore = activity.dirtySeq

        await model.enter()

        XCTAssertEqual(model.feedback, "已进入等待")
        XCTAssertGreaterThan(activity.dirtySeq, dirtyBefore)
        XCTAssertNil(model.navigation)
    }

    func testEnterAfterOpponentLeftWaitingUsesOpponentLeftCopy() async {
        let server = MockHTTPServer()
        server.stub("GET", "/api/challenge/summary", .json(ChallengeFixtures.summary(serverTime: "2026-08-25 12:00:00")))
        server.stub("GET", "/api/challenge/detail", .json(ChallengeFixtures.detail(ChallengeFixtures.challenge(
            id: 7, status: 1, waitingUserId: 18
        ))))
        server.stub("POST", "/api/challenge/enter", .json(ChallengeFixtures.enterResponse(action: "waiting")))
        let (model, _, _) = await makeModel(server)
        await model.load()

        await model.enter(confirmEarly: true)

        XCTAssertEqual(model.feedback, "对方刚退出等待，你已成为等待方")
    }

    func testEnterMatchCreatedNavigatesToMatch() async {
        let server = MockHTTPServer()
        server.stub("POST", "/api/challenge/enter", .json(ChallengeFixtures.enterResponse(action: "match_created", matchId: 66)))
        let (model, _, _) = await makeModel(server)

        await model.enter(confirmEarly: true)

        XCTAssertEqual(model.navigation, .match(66))
    }

    func testEnterSelfOngoingOffersCurrentMatch() async {
        let server = MockHTTPServer()
        server.stub("POST", "/api/challenge/enter", .json(ChallengeFixtures.enterResponse(
            action: "self_ongoing", matchId: 55, message: "你还有未结束的对局"
        )))
        let (model, _, _) = await makeModel(server)

        await model.enter()

        XCTAssertEqual(model.prompt, .ongoingMatch(message: "你还有未结束的对局", matchId: 55))
        XCTAssertNil(model.navigation)
    }

    func testLeaveAndCancelFeedBackThenReload() async {
        let server = MockHTTPServer()
        server.stub("GET", "/api/challenge/summary", .json(ChallengeFixtures.summary()))
        server.stub("GET", "/api/challenge/detail", .json(ChallengeFixtures.detail(ChallengeFixtures.challenge(id: 7, status: 1))))
        server.stub("POST", "/api/challenge/leave", .json(ChallengeFixtures.actionResponse()))
        server.stub("POST", "/api/challenge/cancel", .json(ChallengeFixtures.actionResponse()))
        let (model, _, _) = await makeModel(server)

        await model.leave()
        XCTAssertEqual(model.feedback, "已退出等待，约球保留")

        await model.cancel()
        XCTAssertEqual(model.feedback, "约球已取消")
    }

    func testPollingLoadsOnceThenStopsWhenSleepCancels() async {
        let server = MockHTTPServer()
        server.stub("GET", "/api/challenge/detail", .json(ChallengeFixtures.detail(ChallengeFixtures.challenge(id: 7, status: 1))))
        let (model, _, _) = await makeModel(server, sleep: { _ in throw CancellationError() })

        await model.start()

        XCTAssertEqual(model.phase, .loaded)
        XCTAssertEqual(server.requestCount("/api/challenge/detail"), 1)
    }

    func testStoppedPageDiscardsActionResults() async {
        let server = MockHTTPServer()
        server.stub("POST", "/api/challenge/accept", .json(ChallengeFixtures.acceptResponse(challengeId: 7, matchId: 88)))
        let (model, activity, _) = await makeModel(server)
        let dirtyBefore = activity.dirtySeq
        model.stop()

        await model.accept()

        XCTAssertNil(model.navigation)
        XCTAssertNil(model.feedback)
        XCTAssertEqual(activity.dirtySeq, dirtyBefore)
    }

    func testSignedOutLoadDoesNotApplyDetail() async {
        let server = MockHTTPServer()
        server.stub("GET", "/api/challenge/detail", .json(ChallengeFixtures.detail(ChallengeFixtures.challenge(id: 7, status: 1))))
        let (model, _, session) = await makeModel(server)
        session.logout()

        await model.load()

        XCTAssertEqual(model.phase, .failed)
        XCTAssertNil(model.challenge)
    }
}
