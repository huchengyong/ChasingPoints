import XCTest
@testable import iOS

/// 逐条对应小程序 `uniapp/tests/link-entry-flow.test.mjs`，另补入口链接路由与凭证交接。
@MainActor
final class LinkEntryFlowTests: XCTestCase {
    private let code = "Ab3dEf6hIj9k"

    func testGuestSeesInviterFirstAndExchangesOnlyAfterLoggingIn() async throws {
        let session = LinkSession(loggedIn: false)
        let (flow, calls) = makeFlow(session: session)

        let first = await flow.begin(code)
        guard case .awaitLogin(let inviter, let expiresAt) = first else {
            return XCTFail("访客应先看到邀请人，实际 \(first)")
        }
        XCTAssertEqual(inviter, LinkEntryInviter(name: "邀请人", avatar: "a.png"))
        XCTAssertEqual(expiresAt, "2030-01-01T00:00:00Z")
        XCTAssertEqual(calls.exchange, 0)
        XCTAssertEqual(calls.preview, 1)

        // 放弃登录直接返回：仍然留在预览态，不兑换。
        let ignoredWhileGuest = await flow.resume()
        XCTAssertEqual(ignoredWhileGuest, .ignore)
        XCTAssertEqual(calls.exchange, 0)

        session.loggedIn = true
        let afterLogin = await flow.resume()
        XCTAssertEqual(afterLogin, .confirm)
        XCTAssertEqual(calls.exchange, 1)
        XCTAssertEqual(calls.handoffs.first?.token, "signed.token")
        XCTAssertEqual(calls.handoffs.first?.opponent, MatchScanOpponent(id: 9, nickname: "邀请人", avatar: "a.png"))
    }

    func testAlreadyLoggedInVisitorExchangesImmediatelyWithoutPreviewing() async {
        let (flow, calls) = makeFlow(session: LinkSession(loggedIn: true))

        let action = await flow.begin(code)
        XCTAssertEqual(action, .confirm)
        XCTAssertEqual(calls.preview, 0)
        XCTAssertEqual(calls.exchange, 1)
    }

    func testCodeExpiresWhileGuestIsLoggingIn() async {
        let session = LinkSession(loggedIn: false)
        let (flow, calls) = makeFlow(
            session: session,
            exchange: { _ in
                MatchInviteResolveResponse(
                    success: false,
                    message: "匹配二维码已失效，请让对方刷新二维码",
                    inviteToken: nil,
                    preview: nil
                )
            }
        )

        _ = await flow.begin(code)
        session.loggedIn = true
        let action = await flow.resume()

        XCTAssertEqual(action, .expired(message: "匹配二维码已失效，请让对方刷新二维码"))
        XCTAssertTrue(calls.handoffs.isEmpty)
    }

    func testRepeatedResumeCallsDoNotExchangeTwice() async {
        let (flow, calls) = makeFlow(session: LinkSession(loggedIn: true))

        _ = await flow.begin(code)
        let repeated = await flow.resume()
        XCTAssertEqual(repeated, .ignore)
        XCTAssertEqual(calls.exchange, 1)
        XCTAssertEqual(calls.handoffs.count, 1)
    }

    func testLateExchangeResponseIsDroppedAfterAccountChanged() async {
        let session = LinkSession(loggedIn: true)
        let gate = AsyncGate()
        let started = LockedCounter()
        let (flow, calls) = makeFlow(
            session: session,
            exchange: { _ in
                _ = started.next()
                await gate.wait()
                return MatchInviteResolveResponse(
                    success: true,
                    message: nil,
                    inviteToken: "signed.token",
                    preview: MatchInvitePreviewInfo(opponentId: 9, opponentName: "邀请人", opponentAvatar: "a.png")
                )
            }
        )

        let running = Task { await flow.begin(code) }
        while started.current == 0 { await Task.yield() }
        session.userId = 8
        await gate.open()

        let result = await running.value
        XCTAssertEqual(result, .ignore)
        XCTAssertTrue(calls.handoffs.isEmpty)
    }

    func testFailedPreviewReportsExpiredInvitation() async {
        let (flow, calls) = makeFlow(
            session: LinkSession(loggedIn: false),
            preview: { _ in
                LinkEntryPreviewResponse(
                    success: false,
                    message: "匹配二维码已失效，请让对方刷新二维码",
                    preview: nil
                )
            }
        )

        let expired = await flow.begin(code)
        XCTAssertEqual(expired, .expired(message: "匹配二维码已失效，请让对方刷新二维码"))
        XCTAssertEqual(calls.exchange, 0)
    }

    func testNetworkFailureNeverHandsOverCredential() async {
        let (flow, calls) = makeFlow(
            session: LinkSession(loggedIn: true),
            exchange: { _ in throw URLError(.notConnectedToInternet) }
        )

        let failed = await flow.begin(code)
        XCTAssertEqual(failed, .error(message: LinkEntryFlow.unavailableMessage))
        XCTAssertTrue(calls.handoffs.isEmpty)
    }

    func testNetworkFailureCanBeRetriedAndThenHandsOverOnce() async {
        var failing = true
        let (flow, calls) = makeFlow(
            session: LinkSession(loggedIn: true),
            exchange: { _ in
                if failing { throw URLError(.notConnectedToInternet) }
                return MatchInviteResolveResponse(
                    success: true,
                    message: nil,
                    inviteToken: "signed.token",
                    preview: MatchInvitePreviewInfo(opponentId: 9, opponentName: "邀请人", opponentAvatar: "a.png")
                )
            }
        )

        let firstAttempt = await flow.begin(code)
        XCTAssertEqual(firstAttempt, .error(message: LinkEntryFlow.unavailableMessage))
        failing = false
        let retried = await flow.retry()
        XCTAssertEqual(retried, .confirm)
        XCTAssertEqual(calls.handoffs.count, 1)
        let repeatedRetry = await flow.retry()
        XCTAssertEqual(repeatedRetry, .ignore)
    }

    func testGuestPreviewNetworkFailureIsErrorNotExpiredCode() async {
        var failing = true
        let (flow, _) = makeFlow(
            session: LinkSession(loggedIn: false),
            preview: { _ in
                if failing { throw URLError(.notConnectedToInternet) }
                return LinkEntryPreviewResponse(
                    success: true,
                    message: nil,
                    preview: LinkEntryPreviewInfo(
                        inviterName: "邀请人",
                        inviterAvatar: "a.png",
                        expiresAt: "2030-01-01T00:00:00Z"
                    )
                )
            }
        )

        let guestFailure = await flow.begin(code)
        XCTAssertEqual(guestFailure, .error(message: LinkEntryFlow.unavailableMessage))
        failing = false
        let retried = await flow.retry()
        guard case .awaitLogin = retried else {
            return XCTFail("重试成功后应回到预览态")
        }
    }

    func testExpiredCodeCannotBeRetried() async {
        let (flow, calls) = makeFlow(
            session: LinkSession(loggedIn: false),
            preview: { _ in
                LinkEntryPreviewResponse(
                    success: false,
                    message: "匹配二维码已失效，请让对方刷新二维码",
                    preview: nil
                )
            }
        )

        let expiredAgain = await flow.begin(code)
        XCTAssertEqual(expiredAgain, .expired(message: "匹配二维码已失效，请让对方刷新二维码"))
        let retried = await flow.retry()
        XCTAssertEqual(retried, .ignore)
        XCTAssertEqual(calls.preview, 1)
    }

    // MARK: - 原生交接与路由

    func testResolvedInviteIsHandedToQuickScanForTheCurrentAccount() async {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        server.stub("POST", "/api/match/invite/resolve", .json([
            "success": true,
            "invite_token": "signed.token",
            "preview": ["opponent_id": 9, "opponent_name": "邀请人", "opponent_avatar": "a.png"],
        ]))
        let quickScan = QuickScanFlow(session: session)
        let flow = LinkEntryFlow(session: session, quickScan: quickScan)

        let confirmed = await flow.begin(code)
        XCTAssertEqual(confirmed, .confirm)

        let handoff = quickScan.takeHandoff(identity: session.identity)
        XCTAssertEqual(handoff?.inviteToken, "signed.token")
        XCTAssertEqual(handoff?.opponent, MatchScanOpponent(id: 9, nickname: "邀请人", avatar: "a.png"))
        XCTAssertFalse(String(describing: handoff).contains("signed.token"), "凭证不能进打印")
        XCTAssertNil(quickScan.takeHandoff(identity: session.identity), "交接只能取一次")
    }

    func testHandoffRejectedForAnotherAccountIsIgnored() async {
        let session = LinkSession(loggedIn: true)
        let (flow, calls) = makeFlow(session: session, handed: false)

        let rejected = await flow.begin(code)
        XCTAssertEqual(rejected, .ignore)
        XCTAssertTrue(calls.handoffs.isEmpty)
    }

    func testRoutingOnlyAcceptsEntryLinks() {
        let invite = "https://www.zhuifen.cn/go/?j=Ab3dEf6hIj9k"
        let match = "https://www.zhuifen.cn/go/?m=42"

        XCTAssertEqual(LinkEntryRouting.route(for: invite), .linkEntry(url: invite))
        XCTAssertEqual(LinkEntryRouting.route(for: match), .linkEntry(url: match))
        XCTAssertEqual(LinkEntryRouting.route(for: EntryLink.base), .linkEntry(url: EntryLink.base))
        XCTAssertNil(LinkEntryRouting.route(for: "https://example.com/go/?j=Ab3dEf6hIj9k"))
        XCTAssertNil(LinkEntryRouting.route(for: "not a link"))
    }

    /// 小程序对局链接走 buildSpectateRoute → matchDetail?mode=spectate，原生落点是观战详情。
    func testMatchLinkOpensSpectateDetailWhileInviteStillExchanges() {
        XCTAssertEqual(
            LinkEntryRouting.landing(for: EntryLink.parse("https://www.zhuifen.cn/go/?m=42")),
            .publicMatch(matchId: 42)
        )
        XCTAssertEqual(
            LinkEntryRouting.landing(for: EntryLink.parse("https://www.zhuifen.cn/go/?j=Ab3dEf6hIj9k")),
            .invite(code: "Ab3dEf6hIj9k")
        )
        XCTAssertEqual(LinkEntryRouting.landing(for: EntryLink.parse(EntryLink.base)), .home)
        XCTAssertEqual(LinkEntryRouting.landing(for: .other), .home)
    }

    func testRemainingTextRoundsUpAndHidesExpiredOrInvalidValues() {
        let now = Date(timeIntervalSince1970: 1_893_455_730) // 2029-12-31T23:55:30Z

        XCTAssertEqual(LinkEntryRules.remainingText(expiresAt: "2030-01-01T00:00:00Z", now: now), "5 分钟内有效")
        XCTAssertEqual(LinkEntryRules.remainingText(expiresAt: "2020-01-01T00:00:00Z", now: now), "")
        XCTAssertEqual(LinkEntryRules.remainingText(expiresAt: "", now: now), "")
        XCTAssertEqual(LinkEntryRules.remainingText(expiresAt: "not-a-date", now: now), "")
    }

    // MARK: - 支撑

    private func makeFlow(
        session: LinkSession,
        preview: @escaping (LinkCalls) async throws -> LinkEntryPreviewResponse = { _ in
            LinkEntryPreviewResponse(
                success: true,
                message: nil,
                preview: LinkEntryPreviewInfo(
                    inviterName: "邀请人",
                    inviterAvatar: "a.png",
                    expiresAt: "2030-01-01T00:00:00Z"
                )
            )
        },
        exchange: @escaping (LinkCalls) async throws -> MatchInviteResolveResponse = { _ in
            MatchInviteResolveResponse(
                success: true,
                message: nil,
                inviteToken: "signed.token",
                preview: MatchInvitePreviewInfo(opponentId: 9, opponentName: "邀请人", opponentAvatar: "a.png")
            )
        },
        handed: Bool = true
    ) -> (LinkEntryFlow, LinkCalls) {
        let calls = LinkCalls()
        let flow = LinkEntryFlow(
            currentIdentity: { session.identity },
            previewInvite: { _ in
                calls.preview += 1
                return try await preview(calls)
            },
            resolveInvite: { _ in
                calls.exchange += 1
                return try await exchange(calls)
            },
            handoffResolved: { _, token, opponent in
                if handed { calls.handoffs.append((token, opponent)) }
                return handed
            }
        )
        return (flow, calls)
    }
}

@MainActor
private final class LinkSession {
    var loggedIn: Bool
    var userId: Int64 = 7
    var generation = 1

    init(loggedIn: Bool) {
        self.loggedIn = loggedIn
    }

    var identity: SessionIdentity? {
        loggedIn ? SessionIdentity(userId: userId, generation: generation) : nil
    }
}

@MainActor
private final class LinkCalls {
    var preview = 0
    var exchange = 0
    var handoffs: [(token: String, opponent: MatchScanOpponent)] = []
}
