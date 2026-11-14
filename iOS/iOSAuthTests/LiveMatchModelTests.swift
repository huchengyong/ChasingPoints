import XCTest
@testable import iOS

/// 逐条对应 `uniapp/subPages/match/playing.vue` 的页面模型行为；规则本身已在 Rules 测试中覆盖。
@MainActor
final class LiveMatchModelTests: XCTestCase {
    /// 替身实时通道：记录连接与可见性，并允许测试直接推消息。
    private final class FakeLiveChannel: LiveMatchChannel {
        var isConnected = true
        var openCalls: [(matchId: Int64, allowAnonymous: Bool)] = []
        var syncRequests = 0
        var activeValues: [Bool] = []
        var closeCount = 0
        private var handlers: [UUID: (RealtimeMessage) -> Void] = [:]

        func openLive(matchId: Int64, allowAnonymous: Bool) async throws {
            openCalls.append((matchId, allowAnonymous))
        }

        func requestLiveSync() async {
            syncRequests += 1
        }

        func setLiveActive(_ active: Bool) {
            activeValues.append(active)
        }

        func closeLive() {
            closeCount += 1
            isConnected = false
        }

        func observeLive(_ handler: @escaping (RealtimeMessage) -> Void) -> UUID {
            let token = UUID()
            handlers[token] = handler
            return token
        }

        func stopObservingLive(_ token: UUID) {
            handlers[token] = nil
        }

        func send(_ type: String, _ object: [String: Any]) {
            let data = (try? JSONSerialization.data(withJSONObject: object)) ?? Data()
            let message = RealtimeMessage(type: type, payload: data)
            handlers.values.forEach { $0(message) }
        }

        var observerCount: Int { handlers.count }
    }

    private final class CallCounter: @unchecked Sendable {
        private let lock = NSLock()
        private var value = 0
        func increment() {
            lock.lock()
            value += 1
            lock.unlock()
        }
        var count: Int {
            lock.lock()
            defer { lock.unlock() }
            return value
        }
    }

    private final class DelayRecorder: @unchecked Sendable {
        private let lock = NSLock()
        private(set) var delays: [TimeInterval] = []
        func record(_ value: TimeInterval) {
            lock.lock()
            delays.append(value)
            lock.unlock()
        }
    }

    // MARK: - 夹具

    private func detailJSON(
        id: Int64 = 88,
        viewerRole: String = "player1",
        isPlayer1: Bool = true,
        gameType: Int = 3,
        status: Int = 1,
        revision: Int64 = 5,
        canScore: Bool = true,
        canUndo: Bool = true,
        canFinish: Bool = true,
        canRequestFinish: Bool = false,
        canConfirmFinish: Bool = false,
        canDisputeFinish: Bool = false,
        canWithdrawFinish: Bool = false,
        finishState: String = "none",
        finishRequestedBy: Int64 = 0,
        remainingSeconds: Int? = nil,
        myScore: Int = 0,
        opponentScore: Int = 0,
        frameStarted: Bool = true,
        frameMy: Int = 0,
        frameOpponent: Int = 0,
        currentRound: Int = 1,
        challengeId: Int64 = 0,
        matchMode: String = "ranked",
        matchFormat: String = "legacy",
        targetWins: Int = 0,
        canChangeMatchFormat: Bool = false,
        refereeBound: Bool = false
    ) -> [String: Any] {
        var json: [String: Any] = [
            "id": id,
            "challenge_id": challengeId,
            "game_type": gameType,
            "match_mode": matchMode,
            "status": status,
            "server_revision": revision,
            "is_player1": isPlayer1,
            "viewer_role": viewerRole,
            "referee_bound": refereeBound,
            "can_score": canScore,
            "can_undo": canUndo,
            "can_finish": canFinish,
            "can_request_finish": canRequestFinish,
            "can_confirm_finish": canConfirmFinish,
            "can_dispute_finish": canDisputeFinish,
            "can_withdraw_finish": canWithdrawFinish,
            "finish_state": finishState,
            "finish_requested_by": finishRequestedBy,
            "my_score": myScore,
            "opponent_score": opponentScore,
            "current_frame_started": frameStarted,
            "current_frame_my_score": frameMy,
            "current_frame_opponent_score": frameOpponent,
            "current_round": currentRound,
            "match_format": matchFormat,
            "target_wins": targetWins,
            "can_change_match_format": canChangeMatchFormat,
            "my_name": "我",
            "opponent_name": "对手",
            "my_win_rate": 55,
            "opponent_win_rate": 45,
            "duration_seconds": 65,
        ]
        if let remainingSeconds {
            json["finish_request_remaining_seconds"] = remainingSeconds
        }
        return json
    }

    private func publicDetailJSON(
        id: Int64 = 88,
        gameType: Int = 3,
        status: Int = 1,
        revision: Int64 = 5,
        player1Score: Int = 2,
        player2Score: Int = 1
    ) -> [String: Any] {
        [
            "id": id,
            "game_type": gameType,
            "status": status,
            "server_revision": revision,
            "player1_id": 11,
            "player1_name": "选手甲",
            "player1_avatar": "",
            "player2_id": 12,
            "player2_name": "选手乙",
            "player2_avatar": "",
            "player1_score": player1Score,
            "player2_score": player2Score,
            "current_frame_started": true,
            "current_round": 2,
            "total_rounds": 3,
            "duration_seconds": 120,
        ]
    }

    private func stubDetail(_ server: MockHTTPServer, _ json: [String: Any]) {
        server.stub("GET", "/api/match/detail", .json(["success": true, "match": json]))
    }

    private func stubPublic(_ server: MockHTTPServer, _ json: [String: Any]) {
        server.stub("GET", "/api/public/match/detail", .json(["success": true, "match": json]))
    }

    private func makeModel(
        server: MockHTTPServer,
        entry: LiveMatchEntry = .automatic,
        channel: FakeLiveChannel? = nil,
        now: @escaping () -> Date = Date.init,
        sleep: @escaping (TimeInterval) async throws -> Void = { _ in },
        onSettled: @escaping () -> Void = {},
        onRankInvalidated: @escaping () -> Void = {}
    ) async -> (LiveMatchModel, SessionStore) {
        let session = await makeRestoredAccountSession(server)
        let model = LiveMatchModel(
            session: session,
            matchId: 88,
            entry: entry,
            channel: channel ?? FakeLiveChannel(),
            now: now,
            sleep: sleep,
            onMatchSettled: onSettled,
            onRankInvalidated: onRankInvalidated
        )
        return (model, session)
    }

    // MARK: - 入口

    func testParticipantLoadsDetailAndConnectsChannel() async {
        let server = MockHTTPServer()
        stubDetail(server, detailJSON(myScore: 2, opponentScore: 1, frameMy: 1, frameOpponent: 0))
        let channel = FakeLiveChannel()
        let (model, session) = await makeModel(server: server, channel: channel)

        await model.start()

        XCTAssertEqual(model.phase, .ready)
        XCTAssertEqual(model.gameType, 3)
        XCTAssertEqual(model.gameTypeName, "中式八球")
        XCTAssertEqual(model.myScore, 2)
        XCTAssertEqual(model.opponentScore, 1)
        XCTAssertEqual(model.currentFrameMyScore, 1)
        XCTAssertEqual(model.serverRevision, 5)
        XCTAssertEqual(model.durationText, "1分钟")
        XCTAssertEqual(channel.openCalls.count, 1)
        XCTAssertEqual(channel.openCalls.first?.allowAnonymous, false)
        XCTAssertTrue(model.isConnected)
        XCTAssertNotNil(session.identity)
    }

    func testSpectatorEntryReadsPublicDetailAndIsReadOnly() async {
        let server = MockHTTPServer()
        stubPublic(server, publicDetailJSON())
        let channel = FakeLiveChannel()
        let (model, _) = await makeModel(server: server, entry: .spectator, channel: channel)

        await model.start()

        XCTAssertEqual(model.phase, .ready)
        XCTAssertEqual(model.viewerRole, "spectator")
        XCTAssertTrue(model.isSpectator)
        XCTAssertEqual(model.myScore, 2)
        XCTAssertEqual(model.opponentScore, 1)
        XCTAssertEqual(model.myInfo.nickname, "选手甲")
        XCTAssertEqual(model.opponentInfo.nickname, "选手乙")
        XCTAssertFalse(model.viewerUi.showActionPanel)
        XCTAssertEqual(model.readonlyTitle, "当前为只读观赛")
        XCTAssertEqual(channel.openCalls.first?.allowAnonymous, true)
        XCTAssertEqual(server.requestCount("/api/match/current"), 0, "观赛不查当前对局")
    }

    func testSignedInNonParticipantFallsBackToPublicDetail() async {
        let server = MockHTTPServer()
        server.stub("GET", "/api/match/detail", .json(["success": false, "match": NSNull()]))
        stubPublic(server, publicDetailJSON())
        let (model, _) = await makeModel(server: server)

        await model.start()

        XCTAssertEqual(model.phase, .ready)
        XCTAssertEqual(model.viewerRole, "spectator")
        XCTAssertEqual(model.myScore, 2)
        XCTAssertEqual(server.requestCount("/api/public/match/detail"), 1)
    }

    func testCompletedMatchLeavesForHistoryWithoutConnecting() async {
        let server = MockHTTPServer()
        stubDetail(server, detailJSON(status: 2))
        let channel = FakeLiveChannel()
        let (model, _) = await makeModel(server: server, channel: channel)

        await model.start()

        XCTAssertEqual(model.destination, .matchHistory)
        XCTAssertEqual(channel.openCalls.count, 0)
    }

    func testLoadFailureShowsMessageAndRetryRecovers() async {
        let server = MockHTTPServer()
        server.stub("GET", "/api/match/detail", .json(["success": false, "message": "对局不存在"], status: 404))
        let (model, _) = await makeModel(server: server)

        await model.start()
        guard case .failed(let message) = model.phase else {
            return XCTFail("期望加载失败")
        }
        XCTAssertFalse(message.isEmpty)

        stubDetail(server, detailJSON())
        await model.retry()
        XCTAssertEqual(model.phase, .ready)
    }

    // MARK: - 修订号

    func testStalePushIsDroppedAndNewerPushApplies() async {
        let server = MockHTTPServer()
        stubDetail(server, detailJSON(revision: 5, myScore: 1, opponentScore: 0))
        let channel = FakeLiveChannel()
        let (model, _) = await makeModel(server: server, channel: channel)
        await model.start()

        channel.send(WSMessageType.scoreUpdate, [
            "server_revision": 3,
            "player1_score": 9,
            "player2_score": 9,
        ])
        XCTAssertEqual(model.myScore, 1, "旧修订不能覆盖新状态")
        XCTAssertEqual(model.serverRevision, 5)

        channel.send(WSMessageType.scoreUpdate, [
            "server_revision": 6,
            "player1_score": 2,
            "player2_score": 1,
        ])
        XCTAssertEqual(model.myScore, 2)
        XCTAssertEqual(model.serverRevision, 6)
    }

    func testWriteResponseAheadOfPushWinsAndPushAheadOfResponseWins() async {
        let server = MockHTTPServer()
        stubDetail(server, detailJSON(revision: 5))
        server.stub("POST", "/api/match/round/end", .json([
            "success": true,
            "server_revision": 6,
            "my_score": 1,
            "opponent_score": 0,
        ]))
        let channel = FakeLiveChannel()
        let (model, _) = await makeModel(server: server, channel: channel)
        await model.start()

        await model.addRoundWin(winType: "normal")
        XCTAssertEqual(model.myScore, 1)
        XCTAssertEqual(model.serverRevision, 6)

        // 先到的写响应不能被随后到达的旧推送改回去。
        channel.send(WSMessageType.scoreUpdate, [
            "server_revision": 5,
            "player1_score": 0,
            "player2_score": 0,
        ])
        XCTAssertEqual(model.myScore, 1)

        // 推送先到新修订时，旧写响应同样被丢弃。
        channel.send(WSMessageType.scoreUpdate, [
            "server_revision": 8,
            "player1_score": 3,
            "player2_score": 0,
            "current_frame_player1_score": 0,
            "current_frame_player2_score": 0,
        ])
        XCTAssertEqual(model.myScore, 3)
        XCTAssertEqual(model.serverRevision, 8)
    }

    func testScoreUpdateSwapsSidesForPlayerTwo() async {
        let server = MockHTTPServer()
        stubDetail(server, detailJSON(viewerRole: "player2", isPlayer1: false))
        let channel = FakeLiveChannel()
        let (model, _) = await makeModel(server: server, channel: channel)
        await model.start()

        channel.send(WSMessageType.scoreUpdate, [
            "server_revision": 6,
            "player1_score": 3,
            "player2_score": 1,
            // 后端推送里的 my_score / opponent_score 恒为选手1视角。
            "my_score": 3,
            "opponent_score": 1,
            "current_frame_player1_score": 2,
            "current_frame_player2_score": 0,
        ])

        XCTAssertEqual(model.myScore, 1)
        XCTAssertEqual(model.opponentScore, 3)
        XCTAssertEqual(model.currentFrameMyScore, 0)
        XCTAssertEqual(model.currentFrameOpponentScore, 2)
    }

    func testRoundEndToastUsesViewerPerspective() async {
        let server = MockHTTPServer()
        stubDetail(server, detailJSON(viewerRole: "player2", isPlayer1: false))
        let channel = FakeLiveChannel()
        let (model, _) = await makeModel(server: server, channel: channel)
        await model.start()

        // 后端 actor 相对选手1：选手2 看到的是对手赢得本局。
        channel.send(WSMessageType.roundEnd, [
            "server_revision": 6,
            "player1_score": 3,
            "player2_score": 1,
            "winner": 1,
            "actor": "me",
        ])

        XCTAssertEqual(model.toastMessage, "对手赢得本局")
    }

    // MARK: - 写锁与失败提示

    func testDuplicateTapWhileWriteInFlightIsIgnored() async {
        let server = MockHTTPServer()
        stubDetail(server, detailJSON())
        server.stub("POST", "/api/match/round/end", .json(["success": true, "server_revision": 6]))
        let channel = FakeLiveChannel()
        let (model, _) = await makeModel(server: server, channel: channel)
        await model.start()

        let first = Task { await model.addRoundWin(winType: "normal") }
        await Task.yield()
        await model.addRoundWin(winType: "normal")
        await first.value

        XCTAssertEqual(server.requestCount("/api/match/round/end"), 1)
    }

    func testWriteWithoutScorePermissionOnlyShowsHint() async {
        let server = MockHTTPServer()
        stubDetail(server, detailJSON(canScore: false))
        let (model, _) = await makeModel(server: server)
        await model.start()

        await model.addRoundWin(winType: "normal")

        XCTAssertEqual(model.toastMessage, "当前只有裁判可以记分")
        XCTAssertEqual(server.requestCount("/api/match/round/end"), 0)
    }

    func testFailedWriteShowsServerMessage() async {
        let server = MockHTTPServer()
        stubDetail(server, detailJSON())
        server.stub("POST", "/api/match/round/end", .json([
            "success": false,
            "message": "比分已更新，请重试",
        ]))
        let channel = FakeLiveChannel()
        let (model, _) = await makeModel(server: server, channel: channel)
        await model.start()

        await model.addRoundWin(winType: "normal")

        XCTAssertEqual(model.toastMessage, "比分已更新，请重试")
    }

    func testWritePayloadCarriesIdempotencyAndRevision() async {
        let server = MockHTTPServer()
        stubDetail(server, detailJSON(revision: 12))
        server.stub("POST", "/api/match/round/end", .json(["success": true, "server_revision": 13]))
        let channel = FakeLiveChannel()
        let (model, _) = await makeModel(server: server, channel: channel)
        await model.start()

        await model.addRoundWin(winType: "break_clear", score: 1, winnerActor: 2)

        let body = try? XCTUnwrap(server.requests("/api/match/round/end").first).bodyJSON
        XCTAssertEqual((body?["match_id"] as? NSNumber)?.intValue, 88)
        XCTAssertEqual((body?["base_revision"] as? NSNumber)?.intValue, 12)
        XCTAssertEqual(body?["win_type"] as? String, "break_clear")
        XCTAssertEqual((body?["winner"] as? NSNumber)?.intValue, 2)
        XCTAssertFalse((body?["client_action_id"] as? String ?? "").isEmpty)
    }

    func testUndoRequestsSyncAndShowsResultMessage() async {
        let server = MockHTTPServer()
        stubDetail(server, detailJSON())
        server.stub("POST", "/api/match/undo", .json([
            "success": true,
            "server_revision": 6,
            "message": "已撤销上一步",
        ]))
        let channel = FakeLiveChannel()
        let (model, _) = await makeModel(server: server, channel: channel)
        await model.start()

        await model.undo()

        await waitUntil { channel.syncRequests == 1 }
        XCTAssertEqual(model.toastMessage, "已撤销上一步")
    }

    // MARK: - 赛制

    func testSavingMatchFormatSendsOnlyRevisionAndUpdatesState() async throws {
        let server = MockHTTPServer()
        stubDetail(server, detailJSON(canChangeMatchFormat: true))
        server.stub("POST", "/api/match/format", .json([
            "success": true,
            "snapshot": [
                "game_type": 3,
                "server_revision": 6,
                "match_format": "race_to",
                "target_wins": 7,
                "can_change_match_format": true,
            ],
        ]))
        let (model, _) = await makeModel(server: server)
        await model.start()

        model.toggleFormatEditor()
        model.selectFormatMode(PoolMatchFormat.raceTo)
        model.setDraftTargetWins(7)
        await model.saveMatchFormat()

        XCTAssertEqual(model.matchFormat, PoolMatchFormat.raceTo)
        XCTAssertEqual(model.targetWins, 7)
        XCTAssertEqual(model.matchFormatLabel, "抢7局制")
        XCTAssertFalse(model.isFormatEditorShown)
        XCTAssertEqual(model.serverRevision, 6)

        let request = try XCTUnwrap(server.requests("/api/match/format").first)
        XCTAssertNil(request.bodyJSON["client_action_id"], "赛制接口不带幂等键")
        XCTAssertEqual((request.bodyJSON["base_revision"] as? NSNumber)?.intValue, 5)
        XCTAssertEqual((request.bodyJSON["target_wins"] as? NSNumber)?.intValue, 7)
        XCTAssertEqual(request.bodyJSON["match_format"] as? String, "race_to")
    }

    func testFormatSaveFailureKeepsEditorAndShowsInlineError() async {
        let server = MockHTTPServer()
        stubDetail(server, detailJSON(canChangeMatchFormat: true))
        server.stub("POST", "/api/match/format", .json([
            "success": false,
            "message": "比分已变化，请重新核对",
        ]))
        let (model, _) = await makeModel(server: server)
        await model.start()

        model.toggleFormatEditor()
        await model.saveMatchFormat()

        XCTAssertTrue(model.isFormatEditorShown)
        XCTAssertEqual(model.formatError, "比分已变化，请重新核对")
    }

    // MARK: - 结束流程

    func testRequestFinishShowsExactCopyThenSubmits() async {
        let server = MockHTTPServer()
        stubDetail(server, detailJSON(canRequestFinish: true))
        server.stub("POST", "/api/match/finish/request", .json(["success": true, "server_revision": 6]))
        let channel = FakeLiveChannel()
        let (model, _) = await makeModel(server: server, channel: channel)
        await model.start()

        model.requestFinish()
        let confirm = model.confirm
        XCTAssertEqual(confirm?.title, "发起结束确认")
        XCTAssertEqual(confirm?.confirmTitle, "发起结束")
        XCTAssertEqual(confirm?.message, "比分将冻结，对手需在30秒内确认；期间你可撤销，无人处理则按当前比分自动结束，之后不能再提出异议。")

        model.acceptConfirm()
        await waitUntil { server.requestCount("/api/match/finish/request") == 1 }
        XCTAssertEqual(model.toastMessage, "已发起结束，等待对方确认")
    }

    func testResponderSeesDialogAndSwipeConfirmSubmitsConfirm() async {
        let server = MockHTTPServer()
        stubDetail(server, detailJSON(
            canConfirmFinish: true,
            canDisputeFinish: true,
            finishState: "pending_confirmation",
            finishRequestedBy: 99,
            remainingSeconds: 30
        ))
        server.stub("POST", "/api/match/finish/confirm", .json([
            "success": true,
            "server_revision": 6,
            "snapshot": ["status": 2, "server_revision": 6],
        ]))
        let channel = FakeLiveChannel()
        let (model, _) = await makeModel(server: server, channel: channel)
        await model.start()

        XCTAssertTrue(model.showFinishConfirmDialog)
        XCTAssertEqual(model.finishConfirmView.title, "对手申请结束对局")
        XCTAssertEqual(model.finishSecondsLeft, 30)

        await model.commitSlide()

        XCTAssertEqual(server.requestCount("/api/match/finish/confirm"), 1)
        XCTAssertEqual(model.destination, .result(88))
        XCTAssertEqual(model.toastMessage, "已确认结束并完成结算")
    }

    /// 结束状态推送把确认能力与倒计时放在 `snapshot` 里，只读顶层会让对手端看不见确认弹层。
    func testFinishPushReadsNestedSnapshot() async {
        let server = MockHTTPServer()
        stubDetail(server, detailJSON())
        let channel = FakeLiveChannel()
        let (model, _) = await makeModel(server: server, channel: channel)
        await model.start()

        channel.send(WSMessageType.matchFinishRequest, [
            "server_revision": 6,
            "finish_state": "pending_confirmation",
            "finish_requested_by": 99,
            "snapshot": [
                "server_revision": 6,
                "status": 1,
                "finish_state": "pending_confirmation",
                "finish_requested_by": 99,
                "finish_request_remaining_seconds": 30,
                "can_confirm_finish": true,
                "can_dispute_finish": true,
                "player1_score": 3,
                "player2_score": 1,
            ],
        ])

        XCTAssertTrue(model.showFinishConfirmDialog)
        XCTAssertEqual(model.finishSecondsLeft, 30)
        XCTAssertEqual(model.myScore, 3)
    }

    func testDisputeFinishOnlySubmitsDispute() async {
        let server = MockHTTPServer()
        stubDetail(server, detailJSON(
            canConfirmFinish: true,
            canDisputeFinish: true,
            finishState: "pending_confirmation",
            finishRequestedBy: 99,
            remainingSeconds: 30
        ))
        server.stub("POST", "/api/match/finish/dispute", .json(["success": true, "server_revision": 6]))
        let channel = FakeLiveChannel()
        let (model, _) = await makeModel(server: server, channel: channel)
        await model.start()

        await model.disputeFinish()

        XCTAssertEqual(server.requestCount("/api/match/finish/dispute"), 1)
        XCTAssertEqual(model.toastMessage, "已提出异议，对局继续")
    }

    func testRequesterSeesPanelAndCanWithdraw() async {
        let server = MockHTTPServer()
        stubDetail(server, detailJSON(
            canRequestFinish: true,
            canWithdrawFinish: true,
            finishState: "pending_confirmation",
            finishRequestedBy: 1,
            remainingSeconds: 20
        ))
        server.stub("POST", "/api/match/finish/withdraw", .json(["success": true, "server_revision": 6]))
        let channel = FakeLiveChannel()
        let (model, _) = await makeModel(server: server, channel: channel)
        await model.start()

        XCTAssertFalse(model.showFinishConfirmDialog)
        XCTAssertTrue(model.showFinishConfirmPanel)
        XCTAssertTrue(model.finishConfirmView.showWithdraw)

        await model.withdrawFinish()
        XCTAssertEqual(server.requestCount("/api/match/finish/withdraw"), 1)
        XCTAssertEqual(model.toastMessage, "已撤销结束，对局继续")
    }

    func testRefereeFinishShowsFinalScoreAndSettlementCopy() async {
        let server = MockHTTPServer()
        stubDetail(server, detailJSON(
            viewerRole: "referee",
            isPlayer1: true,
            canFinish: true,
            myScore: 2,
            opponentScore: 1
        ))
        server.stub("POST", "/api/match/finish", .json(["success": true, "server_revision": 6]))
        let (model, _) = await makeModel(server: server)
        await model.start()

        model.finishMatch()
        XCTAssertEqual(model.confirm?.title, "确认最终比分并完成结算")
        XCTAssertEqual(model.confirm?.message, "双方最终比分：2 : 1\n\n完成后立即结算且无需选手确认")
        XCTAssertEqual(model.confirm?.confirmTitle, "确认完成")
        XCTAssertEqual(model.confirm?.cancelTitle, "继续记分")

        model.acceptConfirm()
        await waitUntil { model.destination == .result(88) }
        XCTAssertEqual(server.requestCount("/api/match/finish"), 1)
    }

    func testPlayerFinishAsksForConfirmationBeforeEnding() async {
        let server = MockHTTPServer()
        stubDetail(server, detailJSON(canFinish: true))
        server.stub("POST", "/api/match/finish", .json(["success": true, "server_revision": 6]))
        let (model, _) = await makeModel(server: server)
        await model.start()

        model.finishMatch()
        XCTAssertEqual(model.confirm?.title, "确认结束")
        XCTAssertEqual(model.confirm?.message, "确定要结束本次对局吗？")
        XCTAssertNil(server.requests("/api/match/finish").first)

        model.cancelConfirm()
        XCTAssertNil(model.confirm)
        XCTAssertEqual(server.requestCount("/api/match/finish"), 0)
    }

    func testFreePoolMatchCannotFinishBeforeFirstRound() async {
        let server = MockHTTPServer()
        stubDetail(server, detailJSON(canFinish: true, matchFormat: PoolMatchFormat.free))
        let (model, _) = await makeModel(server: server)
        await model.start()

        model.finishMatch()

        XCTAssertNil(model.confirm)
    }

    func testCountdownSettlesWithFreshReadAfterGrace() async {
        var current = Date(timeIntervalSince1970: 1_000)
        let recorder = DelayRecorder()
        let server = MockHTTPServer()
        stubDetail(server, detailJSON(
            finishState: "pending_confirmation",
            finishRequestedBy: 99,
            remainingSeconds: 1
        ))
        let channel = FakeLiveChannel()
        let (model, _) = await makeModel(
            server: server,
            channel: channel,
            now: { current },
            sleep: { recorder.record($0) }
        )
        await model.start()

        XCTAssertEqual(model.finishSecondsLeft, 1)
        // 宽限期满回源，服务端确认倒计时已归零。
        stubDetail(server, detailJSON(
            finishState: "pending_confirmation",
            finishRequestedBy: 99,
            remainingSeconds: 0
        ))
        current = current.addingTimeInterval(2)
        model.tick()
        await waitUntil { server.requestCount("/api/match/detail") == 2 }

        XCTAssertEqual(recorder.delays.first ?? 0, 3, accuracy: 0.001)
        XCTAssertEqual(model.finishSecondsLeft, 0)
    }

    // MARK: - 下一局

    func testNextRoundSettlesCurrentFrameThenStartsNext() async {
        let server = MockHTTPServer()
        stubDetail(server, detailJSON(frameStarted: true, frameMy: 3, frameOpponent: 1))
        server.stub("POST", "/api/match/round/end", .json(["success": true, "server_revision": 6]))
        server.stub("POST", "/api/match/round/start", .json([
            "success": true,
            "server_revision": 7,
            "round_no": 2,
        ]))
        let channel = FakeLiveChannel()
        let (model, _) = await makeModel(server: server, channel: channel)
        await model.start()

        model.requestNextRound()
        XCTAssertEqual(model.confirm?.title, "开始下一局")
        XCTAssertEqual(model.confirm?.message, "将按当前局比分 3:1 自动判定我方赢下本局，并开始下一局。")

        model.acceptConfirm()
        await waitUntil { model.toastMessage == "第2局开始" }

        XCTAssertEqual(server.requestCount("/api/match/round/end"), 1)
        XCTAssertEqual(server.requestCount("/api/match/round/start"), 1)
    }

    // MARK: - 消息与结算

    func testMatchEndNavigatesToResultOnlyOnce() async {
        let settled = CallCounter()
        let server = MockHTTPServer()
        stubDetail(server, detailJSON(status: 1, revision: 5))
        let channel = FakeLiveChannel()
        let (model, _) = await makeModel(server: server, channel: channel, onSettled: { settled.increment() })
        await model.start()

        channel.send(WSMessageType.matchEnd, [
            "server_revision": 6,
            "status": 2,
            "match_mode": "ranked",
            "player1_score": 2,
            "player2_score": 1,
        ])
        XCTAssertEqual(model.destination, .result(88))
        XCTAssertEqual(settled.count, 1)
        model.consumeDestination()

        channel.send(WSMessageType.matchEnd, [
            "server_revision": 7,
            "status": 2,
            "player1_score": 2,
            "player2_score": 1,
        ])
        XCTAssertNil(model.destination, "赛果只跳一次")
        XCTAssertEqual(settled.count, 2, "重复结束消息仍刷新活动数据")
    }

    func testRankInvalidationOnlyForDecisiveRankedParticipant() async {
        let server = MockHTTPServer()
        stubDetail(server, detailJSON())
        let invalidated = CallCounter()
        let channel = FakeLiveChannel()
        let (model, _) = await makeModel(
            server: server,
            channel: channel,
            onRankInvalidated: { invalidated.increment() }
        )
        await model.start()

        channel.send(WSMessageType.matchEnd, [
            "server_revision": 6,
            "status": 2,
            "match_mode": "ranked",
            "player1_score": 2,
            "player2_score": 1,
        ])
        XCTAssertEqual(invalidated.count, 1)

        channel.send(WSMessageType.matchEnd, [
            "server_revision": 7,
            "status": 2,
            "match_mode": "ranked",
            "player1_score": 2,
            "player2_score": 2,
        ])
        XCTAssertEqual(invalidated.count, 1, "平局不失效段位")
    }

    func testSyncWithCompletedStatusNavigatesToResult() async {
        let server = MockHTTPServer()
        stubDetail(server, detailJSON(revision: 5))
        let channel = FakeLiveChannel()
        let (model, _) = await makeModel(server: server, channel: channel)
        await model.start()

        channel.send(WSMessageType.sync, [
            "server_revision": 6,
            "status": 2,
            "match_mode": "practice",
            "player1_score": 1,
            "player2_score": 0,
        ])

        XCTAssertEqual(model.destination, .result(88))
    }

    func testRoleChangedReloadsCapabilitiesAndRequestsSync() async {
        let server = MockHTTPServer()
        stubDetail(server, detailJSON(viewerRole: "player1", canScore: false))
        let channel = FakeLiveChannel()
        let (model, _) = await makeModel(server: server, channel: channel)
        await model.start()

        XCTAssertFalse(model.viewerUi.showActionPanel)

        stubDetail(server, detailJSON(viewerRole: "referee", canScore: true, canUndo: true))
        channel.send(WSMessageType.matchRoleChanged, [:])

        await waitUntil { model.viewerRole == "referee" }
        XCTAssertTrue(model.viewerUi.showActionPanel)
        XCTAssertEqual(model.viewerUi.leftIdentity, "选手1")
        XCTAssertEqual(channel.syncRequests, 1)
        XCTAssertEqual(model.toastMessage, "你已接管本场记分")
    }

    // MARK: - 生命周期

    func testVisibilityTogglesChannelAndReloadsOnReturn() async {
        let server = MockHTTPServer()
        stubDetail(server, detailJSON())
        let channel = FakeLiveChannel()
        let (model, _) = await makeModel(server: server, channel: channel)
        await model.start()

        model.setVisible(false)
        XCTAssertEqual(channel.activeValues, [false])

        model.setVisible(true)
        await waitUntil { server.requestCount("/api/match/detail") == 2 }
        XCTAssertEqual(channel.activeValues, [false, true])
        XCTAssertEqual(channel.syncRequests, 1)
    }

    func testIdentityChangeStopsChannelAndDropsLateMessages() async {
        let server = MockHTTPServer()
        stubDetail(server, detailJSON(myScore: 2))
        let channel = FakeLiveChannel()
        let (model, session) = await makeModel(server: server, channel: channel)
        await model.start()
        XCTAssertEqual(channel.observerCount, 1)

        session.logout()
        model.handleIdentityChange()

        XCTAssertEqual(model.phase, .failed("登录状态已变更，请返回重试"))
        XCTAssertEqual(channel.observerCount, 0)
        XCTAssertGreaterThanOrEqual(channel.closeCount, 1)

        channel.send(WSMessageType.scoreUpdate, [
            "server_revision": 6,
            "player1_score": 9,
            "player2_score": 9,
        ])
        XCTAssertEqual(model.myScore, 2, "身份变化后的推送不得回写")
    }

    func testStopClosesChannelAndStopsObserving() async {
        let server = MockHTTPServer()
        stubDetail(server, detailJSON())
        let channel = FakeLiveChannel()
        let (model, _) = await makeModel(server: server, channel: channel)
        await model.start()

        model.stop()

        XCTAssertEqual(channel.closeCount, 1)
        XCTAssertEqual(channel.observerCount, 0)
    }

    // MARK: - 裁判二维码

    func testRefereeQRCodeLoadsImageAndClosesCleanly() async {
        let server = MockHTTPServer()
        stubDetail(server, detailJSON())
        server.stub("GET", "/api/match/referee/qrcode", .json([
            "success": true,
            "qrcode_data": "https://example.test/referee?token=secret",
            "expires_in_seconds": 120,
        ]))
        let (model, _) = await makeModel(server: server)
        await model.start()

        await model.openRefereeQRCode()

        XCTAssertTrue(model.isRefereeQRCodePresented)
        XCTAssertFalse(model.refereeQRCodeContent.isEmpty)
        XCTAssertNotNil(model.refereeQRCodeImage)

        model.closeRefereeQRCode()
        XCTAssertFalse(model.isRefereeQRCodePresented)
        XCTAssertTrue(model.refereeQRCodeContent.isEmpty)
        XCTAssertNil(model.refereeQRCodeImage)
    }

    func testRefereeQRCodeFailureShowsRefreshHint() async {
        let server = MockHTTPServer()
        stubDetail(server, detailJSON())
        server.stub("GET", "/api/match/referee/qrcode", .json([
            "success": false,
            "message": "生成裁判码失败，请点击刷新",
        ]))
        let (model, _) = await makeModel(server: server)
        await model.start()

        await model.openRefereeQRCode()

        XCTAssertEqual(model.refereeQRCodeError, "生成裁判码失败，请点击刷新")
        XCTAssertNil(model.refereeQRCodeImage)
    }

    func testLateRefereeQRCodeResultIsDroppedAfterClose() async {
        let server = MockHTTPServer()
        stubDetail(server, detailJSON())
        server.stub("GET", "/api/match/referee/qrcode") { _ in
            Thread.sleep(forTimeInterval: 0.2)
            return .json(["success": true, "qrcode_data": "late-token"])
        }
        let (model, _) = await makeModel(server: server)
        await model.start()

        let task = Task { await model.openRefereeQRCode() }
        try? await Task.sleep(for: .milliseconds(50))
        model.closeRefereeQRCode()
        await task.value

        XCTAssertFalse(model.isRefereeQRCodePresented)
        XCTAssertTrue(model.refereeQRCodeContent.isEmpty)
        XCTAssertNil(model.refereeQRCodeImage)
    }

    // MARK: - 返回与菜单

    func testBackAsksForConfirmationWithExactCopy() async {
        let server = MockHTTPServer()
        stubDetail(server, detailJSON())
        let (model, _) = await makeModel(server: server)
        await model.start()

        model.requestBack()
        XCTAssertEqual(model.confirm?.title, "提示")
        XCTAssertEqual(model.confirm?.message, "对局进行中，确定要退出吗？")
        XCTAssertEqual(model.confirm?.confirmTitle, "退出")
        XCTAssertEqual(model.confirm?.cancelTitle, "继续")

        model.acceptConfirm()
        XCTAssertTrue(model.shouldDismiss)
    }

    func testMenuShowsSettingsAndForfeitUnlessChallengeSlider() async {
        let server = MockHTTPServer()
        stubDetail(server, detailJSON())
        let (model, _) = await makeModel(server: server)
        await model.start()
        XCTAssertEqual(model.menuOptions.map(\.title), ["对局设置", "放弃对局"])

        server.stub("GET", "/api/match/detail", .json(["success": true, "match": detailJSON(challengeId: 7)]))
        await model.retry()
        XCTAssertEqual(model.menuOptions.map(\.title), ["对局设置"])
        XCTAssertTrue(model.isChallengeMatchFinishSlider)
    }
}
