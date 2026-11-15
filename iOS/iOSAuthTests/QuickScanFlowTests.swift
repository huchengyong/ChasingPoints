import XCTest
@testable import iOS

/// 逐条对应小程序 `uniapp/tests/quick-scan.test.mjs`。
/// 路由跳转（`navigateTo` / `redirectTo`）与「打开确认页失败」由 `MainTabView` / `ScanConfirmView`
/// 的导航承担，页面无法在单元测试里触发导航失败，这两类未搬。
@MainActor
final class QuickScanFlowTests: XCTestCase {
    private func makeFlow(
        _ session: SessionStore,
        identity: IdentityBox? = nil
    ) -> QuickScanFlow {
        let box = identity ?? IdentityBox(session.identity)
        return QuickScanFlow(
            currentIdentity: { box.value },
            api: MatchStartAPI(session: session)
        )
    }

    func testGuestIsSentToLoginAndCameraIsNeverOpened() async throws {
        let session = makeSessionStore(MockHTTPServer(), credentialStore: InMemoryCredentialStore())
        let flow = QuickScanFlow()
        let counter = LockedCounter()

        let result = try await flow.start(scan: {
            _ = counter.next()
            return .scanned(MatchStartFixtures.inviteToken)
        })

        XCTAssertEqual(result, .login)
        XCTAssertEqual(counter.current, 0, "访客不应打开相机")
        XCTAssertNil(flow.takeHandoff(identity: session.identity))
    }

    func testOpponentCodeOpensConfirmWithoutCredentialInRoute() async throws {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        server.stub("POST", "/api/match/invite/preview", .json([
            "success": true,
            "preview": ["opponent_id": 9, "opponent_name": "对手乙", "opponent_avatar": "a.png"],
        ]))
        let flow = makeFlow(session)

        let result = try await flow.start(scan: { .scanned(MatchStartFixtures.inviteToken) })

        XCTAssertEqual(result, .confirm(mode: .start))
        let handoff = try XCTUnwrap(flow.takeHandoff(identity: session.identity))
        XCTAssertEqual(handoff.mode, .start)
        XCTAssertEqual(handoff.inviteToken, MatchStartFixtures.inviteToken)
        XCTAssertEqual(handoff.opponent, MatchScanOpponent(id: 9, nickname: "对手乙", avatar: "a.png"))
        XCTAssertNil(handoff.preset)
        XCTAssertNil(handoff.rematch)
        XCTAssertFalse(String(describing: handoff).contains(MatchStartFixtures.inviteToken), "凭证不能进打印")
        XCTAssertNil(flow.takeHandoff(identity: session.identity), "交接只能取一次")
    }

    func testRefereeCodeOpensConfirmInRefereeMode() async throws {
        let session = await makeRestoredAccountSession(MockHTTPServer())
        let flow = makeFlow(session)
        let raw = #"{"type":"match_referee","match_id":77,"join_token":"referee-secret"}"#

        let result = try await flow.start(scan: { .scanned(raw) })
        XCTAssertEqual(result, .confirm(mode: .referee))
        let handoff = try XCTUnwrap(flow.takeHandoff(identity: session.identity))
        XCTAssertEqual(handoff.mode, .referee)
        XCTAssertEqual(handoff.refereeJoin, RefereeJoin(matchId: 77, joinToken: "referee-secret"))
        XCTAssertFalse(String(describing: handoff).contains("referee-secret"))
    }

    func testCancellingStaysSilentAndInvalidCodeExplainsWhy() async throws {
        let session = await makeRestoredAccountSession(MockHTTPServer())
        let flow = makeFlow(session)

        let cancelled = try await flow.start(scan: { .cancelled })
        XCTAssertEqual(cancelled, .cancelled)
        XCTAssertNil(flow.takeHandoff(identity: session.identity))

        let invalid = try await flow.start(scan: { .scanned("not a code") })
        XCTAssertEqual(invalid, .error(message: "无效的二维码"))
        XCTAssertNil(flow.takeHandoff(identity: session.identity))

        let noMessage = try await flow.start(scan: { .failure(.failed(nil)) })
        XCTAssertEqual(noMessage, .error(message: nil))
    }

    func testCameraFailureCopyMatchesTheMiniProgram() async throws {
        let session = await makeRestoredAccountSession(MockHTTPServer())
        let flow = makeFlow(session)

        XCTAssertEqual(ScanFailure.permissionDenied.message, "需要相机权限才能扫码")
        XCTAssertEqual(ScanFailure.unsupported.message, "当前端暂不支持扫码")
        let denied = try await flow.start(scan: { .failure(.permissionDenied) })
        XCTAssertEqual(denied, .error(message: "需要相机权限才能扫码"))
    }

    func testOnlyOneScanRunsAtATimeAndFlagIsReleasedAfterwards() async throws {
        let session = await makeRestoredAccountSession(MockHTTPServer())
        let flow = makeFlow(session)
        let gate = AsyncGate()
        let counter = LockedCounter()

        let first = Task {
            try await flow.start(scan: {
                _ = counter.next()
                await gate.wait()
                return .cancelled
            })
        }
        while counter.current == 0 { await Task.yield() }

        let second = try await flow.start(scan: {
            _ = counter.next()
            return .cancelled
        })
        XCTAssertEqual(second, .busy)
        XCTAssertEqual(counter.current, 1)

        await gate.open()
        let firstResult = try await first.value
        XCTAssertEqual(firstResult, .cancelled)

        let third = try await flow.start(scan: {
            _ = counter.next()
            return .cancelled
        })
        XCTAssertEqual(third, .cancelled)
        XCTAssertEqual(counter.current, 2)
    }

    func testBusyFlagIsReleasedWhenScanThrows() async throws {
        let session = await makeRestoredAccountSession(MockHTTPServer())
        let flow = makeFlow(session)
        let counter = LockedCounter()

        do {
            _ = try await flow.start(scan: {
                _ = counter.next()
                throw URLError(.unknown)
            })
            XCTFail("扫码抛出的错误应向上传递")
        } catch {}

        let recovered = try await flow.start(scan: {
            _ = counter.next()
            return .cancelled
        })
        XCTAssertEqual(recovered, .cancelled)
        XCTAssertEqual(counter.current, 2)
    }

    func testResultArrivingAfterAccountChangeIsDropped() async throws {
        let session = await makeRestoredAccountSession(MockHTTPServer())
        let identity = IdentityBox(session.identity)
        let flow = makeFlow(session, identity: identity)
        let gate = AsyncGate()
        let counter = LockedCounter()

        let run = Task {
            try await flow.start(scan: {
                _ = counter.next()
                await gate.wait()
                return .scanned(MatchStartFixtures.inviteToken)
            })
        }
        while counter.current == 0 { await Task.yield() }

        identity.value = SessionIdentity(userId: 1, generation: 2)
        await gate.open()

        let stale = try await run.value
        XCTAssertEqual(stale, .stale)
        XCTAssertNil(flow.takeHandoff(identity: identity.value))
        XCTAssertNil(flow.takeHandoff(identity: session.identity))
    }

    func testLoggingOutDuringScanAlsoDropsTheResult() async throws {
        let session = await makeRestoredAccountSession(MockHTTPServer())
        let flow = QuickScanFlow(currentIdentity: { session.identity }, api: MatchStartAPI(session: session))
        let gate = AsyncGate()
        let counter = LockedCounter()

        let run = Task {
            try await flow.start(scan: {
                _ = counter.next()
                await gate.wait()
                return .scanned(#"{"type":"match_referee","match_id":77,"join_token":"referee-secret"}"#)
            })
        }
        while counter.current == 0 { await Task.yield() }

        session.logout()
        await gate.open()

        let stale = try await run.value
        XCTAssertEqual(stale, .stale)
        XCTAssertNil(flow.takeHandoff(identity: session.identity))
    }

    func testHandoffIsOnlyReleasedToTheScanningIdentity() async throws {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        server.stub("POST", "/api/match/invite/preview", .json([
            "success": true,
            "preview": ["opponent_id": 9, "opponent_name": "对手乙", "opponent_avatar": ""],
        ]))
        let flow = makeFlow(session)
        _ = try await flow.start(scan: { .scanned(MatchStartFixtures.inviteToken) })

        XCTAssertNil(flow.takeHandoff(identity: SessionIdentity(userId: 8, generation: 1)))
        XCTAssertNil(flow.takeHandoff(identity: nil))
        XCTAssertNil(flow.takeHandoff(identity: session.identity), "身份不符的取用必须清掉凭证")
    }

    func testFlowsBoundToDifferentSessionsDoNotShareHandoffs() async {
        let sessionA = await makeRestoredAccountSession(MockHTTPServer(), user: AuthFixtures.userInfo(id: 1))
        let sessionB = await makeRestoredAccountSession(MockHTTPServer(), user: AuthFixtures.userInfo(id: 2))
        let flowA = QuickScanFlow(session: sessionA)
        let flowB = QuickScanFlow(session: sessionB)
        let opponent = MatchScanOpponent(id: 9, nickname: "对手乙", avatar: "")

        XCTAssertTrue(flowA.handoffResolved(
            identity: sessionA.identity,
            inviteToken: "token.a",
            opponent: opponent
        ))
        XCTAssertNil(flowB.takeHandoff(identity: sessionB.identity), "A 的交接不能从绑定 B 的实例取到")
        XCTAssertNil(flowB.takeHandoff(identity: sessionA.identity))
        XCTAssertEqual(flowA.takeHandoff(identity: sessionA.identity)?.inviteToken, "token.a")
    }

    func testNewScanReplacesUntakenHandoff() async throws {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        var round = 0
        server.stub("POST", "/api/match/invite/preview") { _ in
            round += 1
            return .json([
                "success": true,
                "preview": ["opponent_id": round, "opponent_name": "对手\(round)", "opponent_avatar": ""],
            ])
        }
        let flow = makeFlow(session)

        _ = try await flow.start(scan: { .scanned("token.1") })
        _ = try await flow.start(scan: { .scanned("token.2") })

        XCTAssertEqual(flow.takeHandoff(identity: session.identity)?.inviteToken, "token.2")
    }

    func testRematchOnlyAcceptsOriginalOpponentAndCarriesPreset() async throws {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        let flow = makeFlow(session)
        let rematch = ScanRematch(opponentId: 9, gameType: 1, matchMode: "ranked", visibility: "private")
        server.stub("POST", "/api/match/invite/preview", .json([
            "success": true,
            "preview": ["opponent_id": 9, "opponent_name": "对手乙", "opponent_avatar": ""],
        ]))

        let confirmed = try await flow.start(rematch: rematch, scan: { .scanned("signed.token") })
        XCTAssertEqual(confirmed, .confirm(mode: .start))
        let handoff = try XCTUnwrap(flow.takeHandoff(identity: session.identity))
        XCTAssertEqual(handoff.preset, ScanRematchPreset(gameType: 1, matchMode: "ranked", visibility: "public"))
        XCTAssertEqual(handoff.rematch, rematch, "确认页重新扫码需要原始请求")

        server.stub("POST", "/api/match/invite/preview", .json([
            "success": true,
            "preview": ["opponent_id": 3, "opponent_name": "路人", "opponent_avatar": ""],
        ]))
        let mismatch = try await flow.start(rematch: rematch, scan: { .scanned("signed.token") })
        XCTAssertEqual(mismatch, .mismatch)
        XCTAssertNil(flow.takeHandoff(identity: session.identity))
    }

    func testRematchPresetsDefaultVisibilityFromMatchMode() async throws {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        let flow = makeFlow(session)
        server.stub("POST", "/api/match/invite/preview", .json([
            "success": true,
            "preview": ["opponent_id": 9, "opponent_name": "对手乙", "opponent_avatar": ""],
        ]))

        let practice = ScanRematch(opponentId: 9, gameType: 3, matchMode: "practice", visibility: nil)
        _ = try await flow.start(rematch: practice, scan: { .scanned("signed.token") })
        XCTAssertEqual(
            flow.takeHandoff(identity: session.identity)?.preset,
            ScanRematchPreset(gameType: 3, matchMode: "practice", visibility: "private")
        )

        let publicPractice = ScanRematch(opponentId: 9, gameType: 4, matchMode: "practice", visibility: "public")
        _ = try await flow.start(rematch: publicPractice, scan: { .scanned("signed.token") })
        XCTAssertEqual(
            flow.takeHandoff(identity: session.identity)?.preset,
            ScanRematchPreset(gameType: 4, matchMode: "practice", visibility: "public")
        )
    }

    func testRefereeCodeDuringRematchIsStillRefereeCode() async throws {
        let session = await makeRestoredAccountSession(MockHTTPServer())
        let flow = makeFlow(session)
        let rematch = ScanRematch(opponentId: 9, gameType: 3, matchMode: "practice", visibility: nil)
        let raw = #"{"type":"match_referee","match_id":77,"join_token":"referee-secret"}"#

        let result = try await flow.start(rematch: rematch, scan: { .scanned(raw) })
        XCTAssertEqual(result, .confirm(mode: .referee))
        XCTAssertNil(flow.takeHandoff(identity: session.identity)?.preset)
    }

    func testMatchLinkOpensSpectateDetailInsteadOfConfirm() async throws {
        let session = await makeRestoredAccountSession(MockHTTPServer())
        let flow = makeFlow(session)

        let result = try await flow.start(scan: { .scanned("https://www.zhuifen.cn/go/?m=123") })
        XCTAssertEqual(result, .spectate(matchId: 123))
        XCTAssertNil(flow.takeHandoff(identity: session.identity))
    }

    func testResolvedInviteLinkHandsCredentialOverOnce() async {
        let session = await makeRestoredAccountSession(MockHTTPServer())
        let flow = makeFlow(session)
        let opponent = MatchScanOpponent(id: 9, nickname: "对手乙", avatar: "a.png")

        XCTAssertTrue(flow.handoffResolved(
            identity: session.identity,
            inviteToken: "link.token",
            opponent: opponent
        ))
        let handoff = flow.takeHandoff(identity: session.identity)
        XCTAssertEqual(handoff?.mode, .start)
        XCTAssertEqual(handoff?.inviteToken, "link.token")
        XCTAssertEqual(handoff?.opponent, opponent)
        XCTAssertNil(flow.takeHandoff(identity: session.identity))
    }

    func testResolvedInviteLinkDroppedWhenIdentityChangedOrTokenMissing() async {
        let session = await makeRestoredAccountSession(MockHTTPServer())
        let flow = makeFlow(session)
        let opponent = MatchScanOpponent(id: 9, nickname: "对手乙", avatar: "a.png")

        XCTAssertFalse(flow.handoffResolved(
            identity: SessionIdentity(userId: 8, generation: 1),
            inviteToken: "link.token",
            opponent: opponent
        ))
        XCTAssertFalse(flow.handoffResolved(
            identity: session.identity,
            inviteToken: "",
            opponent: opponent
        ))
        XCTAssertNil(flow.takeHandoff(identity: session.identity))
    }
}
