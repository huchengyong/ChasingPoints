import XCTest
@testable import iOS

/// 逐条对应小程序 `uniapp/tests/scan-confirm-page.test.mjs`。
/// 「拒绝执裁返回上一页」「重新扫码替换当前页」属于页面导航，交由 `ScanConfirmView` 承担，未在模型层重复。
@MainActor
final class ScanConfirmModelTests: XCTestCase {
    private var suiteName = ""
    private var defaults: UserDefaults!
    private var preferences: GameTypePreference!

    override func setUp() {
        super.setUp()
        suiteName = "ScanConfirmModelTests.\(UUID().uuidString)"
        defaults = UserDefaults(suiteName: suiteName)
        defaults.removePersistentDomain(forName: suiteName)
        preferences = GameTypePreference(defaults: defaults)
    }

    override func tearDown() {
        defaults.removePersistentDomain(forName: suiteName)
        defaults = nil
        preferences = nil
        super.tearDown()
    }

    private func makeModel(
        _ session: SessionStore,
        handoff: ScanHandoff?,
        api: MatchStartAPI? = nil
    ) -> ScanConfirmModel {
        ScanConfirmModel(
            session: session,
            handoff: handoff,
            api: api,
            preferences: preferences
        )
    }

    private func stubStart(_ server: MockHTTPServer, _ response: StartMatchResponse) {
        server.stub("POST", "/api/match/start") { _ in
            .json([
                "success": response.success,
                "action": response.action ?? NSNull(),
                "block_reason": response.blockReason ?? NSNull(),
                "message": response.message ?? NSNull(),
                "match_id": response.matchId ?? 0,
                "match": NSNull(),
                "ongoing_match": NSNull(),
            ])
        }
    }

    func testPickingGameTypeAndStartingCreatesMatchAndNavigates() async throws {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        let handoff = MatchStartFixtures.startHandoff(identity: session.identity!)
        stubStart(server, MatchStartFixtures.startResponse(matchId: 55))
        let model = makeModel(session, handoff: handoff)

        XCTAssertEqual(model.stage, .ready)
        XCTAssertEqual(model.navigationTitle, "确认对局")
        XCTAssertEqual(model.gameType, 0, "无默认球种时不能直接开始")
        await model.start()
        XCTAssertEqual(server.requestCount("/api/match/start"), 0, "球种未选不应发请求")

        model.selectGameType(3)
        await model.start()

        let request = try XCTUnwrap(server.requests("/api/match/start").first)
        let body = request.bodyJSON
        XCTAssertEqual((body["game_type"] as? NSNumber)?.intValue, 3)
        XCTAssertEqual(body["match_mode"] as? String, "practice")
        XCTAssertEqual(body["visibility"] as? String, "private")
        XCTAssertEqual(body["invite_token"] as? String, MatchStartFixtures.inviteToken)
        XCTAssertEqual(body["match_format"] as? String, "free")
        XCTAssertEqual((body["target_wins"] as? NSNumber)?.intValue, 0)
        XCTAssertEqual(model.liveMatchId, 55)
    }

    func testSavedDefaultGameTypeIsPreselectedAndMarked() async throws {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        preferences.save(userId: session.identity!.userId, gameType: 2)
        let model = makeModel(session, handoff: MatchStartFixtures.startHandoff(identity: session.identity!))

        XCTAssertEqual(model.gameType, 2)
        XCTAssertEqual(model.defaultGameType, 2)
        XCTAssertFalse(model.canSetDefault, "默认球种本身不需要「设为默认」入口")
        XCTAssertEqual(model.stage, .ready)
    }

    func testDifferentChoiceCanBeSavedAsTheNewDefault() async throws {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        let userId = session.identity!.userId
        preferences.save(userId: userId, gameType: 2)
        stubStart(server, MatchStartFixtures.startResponse())
        let model = makeModel(session, handoff: MatchStartFixtures.startHandoff(identity: session.identity!))

        model.selectGameType(1)
        XCTAssertTrue(model.canSetDefault)
        model.toggleSetAsDefault()
        await model.start()

        XCTAssertEqual(preferences.read(userId: userId), 1)
    }

    func testWithoutTheCheckboxTheDefaultStaysUntouched() async throws {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        let userId = session.identity!.userId
        preferences.save(userId: userId, gameType: 2)
        stubStart(server, MatchStartFixtures.startResponse())
        let model = makeModel(session, handoff: MatchStartFixtures.startHandoff(identity: session.identity!))

        model.selectGameType(1)
        await model.start()

        XCTAssertEqual(preferences.read(userId: userId), 2)
    }

    func testMatchTypeChoiceMapsToTheExistingModeAndVisibilityPairs() async throws {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        preferences.save(userId: session.identity!.userId, gameType: 3)
        stubStart(server, MatchStartFixtures.startResponse())
        let model = makeModel(session, handoff: MatchStartFixtures.startHandoff(identity: session.identity!))

        XCTAssertEqual(StartMatch.modeOptions.count, 3)
        XCTAssertEqual(model.modeIndex, 0, "练习赛 + 私密是默认")

        model.selectMode(2)
        await model.start()

        let body = try XCTUnwrap(server.requests("/api/match/start").first).bodyJSON
        XCTAssertEqual(body["match_mode"] as? String, "ranked")
        XCTAssertEqual(body["visibility"] as? String, "public")
    }

    func testRematchPresetPreselectsGameTypeModeAndVisibility() async throws {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        stubStart(server, MatchStartFixtures.startResponse())
        let rematch = ScanRematch(opponentId: 9, gameType: 1, matchMode: "practice", visibility: "public")
        let handoff = MatchStartFixtures.startHandoff(
            identity: session.identity!,
            preset: ScanRematchPreset(gameType: 1, matchMode: "practice", visibility: "public"),
            rematch: rematch
        )
        let model = makeModel(session, handoff: handoff)

        XCTAssertEqual(model.gameType, 1)
        XCTAssertEqual(model.modeIndex, 1)

        await model.start()

        let body = try XCTUnwrap(server.requests("/api/match/start").first).bodyJSON
        XCTAssertEqual((body["game_type"] as? NSNumber)?.intValue, 1)
        XCTAssertEqual(body["match_mode"] as? String, "practice")
        XCTAssertEqual(body["visibility"] as? String, "public")
        XCTAssertEqual(model.rematch, rematch)
    }

    func testRejectedInviteShowsReasonCreatesNothingAndKeepsRematchRequest() async throws {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        let rematch = ScanRematch(opponentId: 9, gameType: 3, matchMode: "practice", visibility: nil)
        stubStart(server, MatchStartFixtures.startResponse(
            success: false,
            action: nil,
            message: "匹配二维码已失效，请让对方刷新二维码"
        ))
        let model = makeModel(session, handoff: MatchStartFixtures.startHandoff(
            identity: session.identity!,
            preset: ScanRematchPreset(gameType: 3, matchMode: "practice", visibility: "private"),
            rematch: rematch
        ))

        await model.start()

        XCTAssertEqual(model.errorMessage, "匹配二维码已失效，请让对方刷新二维码")
        XCTAssertEqual(model.stage, .ready)
        XCTAssertNil(model.liveMatchId)
        XCTAssertEqual(model.rematch, rematch, "重新扫码需要保留原始请求")
    }

    func testNetworkFailureWhileStartingIsReportedInlineAndTheButtonRecovers() async throws {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        preferences.save(userId: session.identity!.userId, gameType: 3)
        server.stub("POST", "/api/match/start", .failure(.notConnectedToInternet))
        let model = makeModel(session, handoff: MatchStartFixtures.startHandoff(identity: session.identity!))

        await model.start()

        XCTAssertEqual(model.errorMessage, "匹配失败，请重试")
        XCTAssertFalse(model.submitting, "网络失败后按钮应恢复")
    }

    func testWhileStartRequestIsInFlightOnlyOneRequestIsSent() async throws {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        preferences.save(userId: session.identity!.userId, gameType: 3)
        let gate = DispatchSemaphore(value: 0)
        server.stub("POST", "/api/match/start") { _ in
            gate.wait()
            return .json(["success": true, "action": "created", "match_id": 55])
        }
        let model = makeModel(session, handoff: MatchStartFixtures.startHandoff(identity: session.identity!))

        let run = Task { await model.start() }
        while server.requestCount("/api/match/start") == 0 { await Task.yield() }
        XCTAssertTrue(model.submitting)
        XCTAssertEqual(model.navigationTitle, "确认对局")

        await model.start()
        XCTAssertEqual(server.requestCount("/api/match/start"), 1, "提交中应互斥")

        gate.signal()
        await run.value
        XCTAssertEqual(model.liveMatchId, 55)
        XCTAssertFalse(model.submitting)
    }

    func testExistingUnfinishedMatchPromptsAndEntersThatMatchOnConfirmation() async throws {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        preferences.save(userId: session.identity!.userId, gameType: 3)
        server.stub("POST", "/api/match/start", .json([
            "success": true,
            "action": "blocked",
            "block_reason": "self_ongoing",
            "message": "你还有未结束的对局，请先处理",
            "ongoing_match": ["id": 31, "game_type": 3, "opponent_name": "老对手", "opponent_avatar": ""],
        ]))
        let model = makeModel(session, handoff: MatchStartFixtures.startHandoff(identity: session.identity!))

        await model.start()

        let prompt = try XCTUnwrap(model.selfOngoingPrompt)
        XCTAssertEqual(prompt.message, "你还有未结束的对局，请先处理")
        XCTAssertEqual(prompt.match.matchId, 31)
        XCTAssertNil(model.liveMatchId)

        model.confirmSelfOngoing()
        XCTAssertEqual(model.liveMatchId, 31)
        XCTAssertNil(model.selfOngoingPrompt)
    }

    func testBusyOpponentIsAToastAndThePageStays() async throws {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        preferences.save(userId: session.identity!.userId, gameType: 3)
        server.stub("POST", "/api/match/start", .json([
            "success": true,
            "action": "blocked",
            "block_reason": "opponent_ongoing",
            "message": "对手还有未结束的对局，暂时无法开始新的 PK",
        ]))
        let model = makeModel(session, handoff: MatchStartFixtures.startHandoff(identity: session.identity!))

        await model.start()

        XCTAssertEqual(model.toastMessage, "对手还有未结束的对局，暂时无法开始新的 PK")
        XCTAssertNil(model.liveMatchId)
    }

    func testStartResultArrivingAfterAccountChangedNeitherNavigatesNorPrompts() async throws {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        preferences.save(userId: session.identity!.userId, gameType: 3)
        let gate = DispatchSemaphore(value: 0)
        server.stub("POST", "/api/match/start") { _ in
            gate.wait()
            return .json([
                "success": true,
                "action": "blocked",
                "block_reason": "self_ongoing",
                "message": "你还有未结束的对局，请先处理",
                "ongoing_match": ["id": 31, "game_type": 3],
            ])
        }
        let model = makeModel(session, handoff: MatchStartFixtures.startHandoff(identity: session.identity!))

        let run = Task { await model.start() }
        while server.requestCount("/api/match/start") == 0 { await Task.yield() }
        session.logout()
        gate.signal()
        await run.value

        XCTAssertNil(model.liveMatchId)
        XCTAssertNil(model.selfOngoingPrompt)
        XCTAssertTrue(model.errorMessage.isEmpty)
    }

    func testComingBackAfterAccountChangedDiscardsTheScan() async throws {
        let session = await makeRestoredAccountSession(MockHTTPServer())
        let model = makeModel(session, handoff: MatchStartFixtures.startHandoff(identity: session.identity!))

        session.logout()
        model.handleIdentityChange()

        XCTAssertEqual(model.stage, .expired)
        XCTAssertEqual(model.errorMessage, "登录状态已变更，请重新扫码")
    }

    func testStartingWithoutCurrentIdentityExpiresTheScan() async throws {
        let session = await makeRestoredAccountSession(MockHTTPServer())
        let model = makeModel(session, handoff: MatchStartFixtures.startHandoff(identity: session.identity!))
        model.selectGameType(3)

        session.logout()
        await model.start()

        XCTAssertEqual(model.stage, .expired)
        XCTAssertEqual(model.errorMessage, "登录状态已变更，请重新扫码")
    }

    func testWithoutHandoffDataThePageAsksToScanAgain() async {
        let session = await makeRestoredAccountSession(MockHTTPServer())
        let model = makeModel(session, handoff: nil)

        XCTAssertEqual(model.stage, .expired)
        XCTAssertEqual(model.errorMessage, "扫码信息已失效，请重新扫码")
        XCTAssertNil(model.rematch)
    }

    func testRefereeModePreviewsTheMatchFirstAndJoinsOnlyAfterConfirmation() async throws {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        server.stub("POST", "/api/match/referee/preview", .json([
            "success": true,
            "preview": MatchStartFixtures.refereePreviewJSON,
        ]))
        server.stub("POST", "/api/match/referee/join", .json([
            "success": true,
            "match_id": 77,
            "match": ["game_type": 3, "opponent_name": "乙", "opponent_avatar": ""],
        ]))
        let model = makeModel(session, handoff: MatchStartFixtures.refereeHandoff(identity: session.identity!))

        await model.loadIfNeeded()

        XCTAssertEqual(model.stage, .ready)
        XCTAssertEqual(model.navigationTitle, "确认执裁")
        XCTAssertEqual(model.refereePreview, MatchStartFixtures.refereePreview)
        XCTAssertEqual(model.refereeModeText, "排位赛")
        XCTAssertEqual(server.requestCount("/api/match/referee/join"), 0, "预览阶段不能加入裁判")

        await model.join()

        XCTAssertEqual(model.liveMatchId, 77)
        let join = try XCTUnwrap(server.requests("/api/match/referee/join").first)
        XCTAssertEqual((join.bodyJSON["match_id"] as? NSNumber)?.int64Value, 77)
        XCTAssertEqual(join.bodyJSON["join_token"] as? String, MatchStartFixtures.refereeJoinToken)
    }

    func testExpiredRefereeCodeIsExplainedAndCanBeRescanned() async throws {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        server.stub("POST", "/api/match/referee/preview", .json([
            "success": false,
            "message": "裁判码已失效，请重新扫码",
        ]))
        let model = makeModel(session, handoff: MatchStartFixtures.refereeHandoff(identity: session.identity!))

        await model.loadIfNeeded()

        XCTAssertEqual(model.stage, .error)
        XCTAssertEqual(model.errorMessage, "裁判码已失效，请重新扫码")
        XCTAssertNil(model.refereePreview)
    }

    func testPreviewFailureWithoutMessageUsesTheDefaultRefereeCopy() async throws {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        server.stub("POST", "/api/match/referee/preview", .failure(.notConnectedToInternet))
        let model = makeModel(session, handoff: MatchStartFixtures.refereeHandoff(identity: session.identity!))

        await model.loadIfNeeded()

        XCTAssertEqual(model.stage, .error)
        XCTAssertEqual(model.errorMessage, "裁判码已失效，请重新扫码")
    }

    func testRefereeCodeThatFailsAtJoinTimeSwitchesToTheErrorState() async throws {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        server.stub("POST", "/api/match/referee/preview", .json([
            "success": true,
            "preview": MatchStartFixtures.refereePreviewJSON,
        ]))
        server.stub("POST", "/api/match/referee/join", .json([
            "success": false,
            "message": "该对局已有裁判",
        ]))
        let model = makeModel(session, handoff: MatchStartFixtures.refereeHandoff(identity: session.identity!))
        await model.loadIfNeeded()

        await model.join()

        XCTAssertEqual(model.stage, .error)
        XCTAssertEqual(model.errorMessage, "该对局已有裁判")
        XCTAssertNil(model.liveMatchId)
    }

    func testThrownJoinExplainsHowToRecover() async throws {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        server.stub("POST", "/api/match/referee/preview", .json([
            "success": true,
            "preview": MatchStartFixtures.refereePreviewJSON,
        ]))
        server.stub("POST", "/api/match/referee/join", .failure(.notConnectedToInternet))
        let model = makeModel(session, handoff: MatchStartFixtures.refereeHandoff(identity: session.identity!))
        await model.loadIfNeeded()

        await model.join()

        XCTAssertEqual(model.stage, .error)
        XCTAssertEqual(model.errorMessage, "加入裁判失败，请重新扫码")
        XCTAssertNil(model.liveMatchId)
    }
}
