import Foundation
import XCTest
@testable import iOS

/// 对局详情三种入口、赛道视角与实时推送的行为；规则本身由 `MatchDetailRulesTests` 覆盖。
@MainActor
final class MatchDetailModelTests: XCTestCase {
    /// 替身实时通道：记录连接与订阅，并允许测试直接推消息。
    private final class FakeLiveChannel: LiveMatchChannel {
        var isConnected = false
        var openCalls: [(matchId: Int64, allowAnonymous: Bool)] = []
        var syncRequests = 0
        var closeCount = 0
        private var handlers: [UUID: (RealtimeMessage) -> Void] = [:]

        func openLive(matchId: Int64, allowAnonymous: Bool) async throws {
            openCalls.append((matchId, allowAnonymous))
            isConnected = true
        }

        func requestLiveSync() async {
            syncRequests += 1
        }

        func setLiveActive(_ active: Bool) {}

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

    private final class DelayRecorder: @unchecked Sendable {
        private let lock = NSLock()
        private var values: [TimeInterval] = []
        func record(_ value: TimeInterval) {
            lock.lock()
            values.append(value)
            lock.unlock()
        }
        var delays: [TimeInterval] {
            lock.lock()
            defer { lock.unlock() }
            return values
        }
    }

    // MARK: - 入口

    func testSpectateEntryReadsPublicDetailAndConnectsWithAnonymousTicket() async throws {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        server.stub("GET", "/api/public/match/detail", .json(publicResponse(status: 1)))
        let channel = FakeLiveChannel()
        let model = MatchDetailModel(session: session, matchId: 88, mode: "spectate", channel: channel)

        await model.start()

        XCTAssertEqual(server.requestCount("/api/public/match/detail"), 1)
        XCTAssertEqual(server.requestCount("/api/match/detail"), 0)
        XCTAssertEqual(channel.openCalls.map(\.matchId), [88])
        XCTAssertEqual(channel.openCalls.map(\.allowAnonymous), [true])
        guard case .loaded(let data) = model.phase else { return XCTFail("应加载成功：\(model.phase)") }
        XCTAssertEqual(data.matchData.status, 1)
        model.stop()
    }

    func testParticipantEntryReadsPrivateDetailAndClosesOnLeave() async throws {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        server.stub("GET", "/api/match/detail", .json(MatchHistoryFixtures.playerDetail(status: 1)))
        let channel = FakeLiveChannel()
        let model = MatchDetailModel(session: session, matchId: 101, channel: channel)

        await model.start()

        XCTAssertEqual(server.requestCount("/api/match/detail"), 1)
        XCTAssertEqual(server.requestCount("/api/public/match/detail"), 0)
        XCTAssertEqual(channel.openCalls.map(\.allowAnonymous), [false])
        XCTAssertEqual(channel.observerCount, 1)

        model.stop()

        XCTAssertEqual(channel.closeCount, 1)
        XCTAssertEqual(channel.observerCount, 0)
    }

    func testHistoryEntryAlwaysReadsPublicDetail() async throws {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        server.stub("GET", "/api/public/match/detail", .json(publicResponse(status: 2)))
        let channel = FakeLiveChannel()
        let model = MatchDetailModel(
            session: session,
            matchId: 88,
            source: "history",
            perspectiveUserId: 12,
            channel: channel
        )

        await model.start()

        XCTAssertEqual(server.requestCount("/api/public/match/detail"), 1)
        XCTAssertEqual(server.requestCount("/api/match/detail"), 0)
        // 已结束的对局不连实时通道。
        XCTAssertTrue(channel.openCalls.isEmpty)
        guard case .loaded(let data) = model.phase else { return XCTFail("应加载成功：\(model.phase)") }
        XCTAssertEqual(data.player1UserId, 12)
        XCTAssertEqual(data.player1Info.name, "选手乙")
        XCTAssertEqual(data.matchData.player1Score, 1)
        XCTAssertEqual(data.matchData.player2Score, 3)
    }

    // MARK: - 推送

    func testScoreUpdateAppliesWithoutReloadingFinishedRounds() async throws {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        server.stub("GET", "/api/public/match/detail", .json(publicResponse(status: 1, gameType: 3)))
        let channel = FakeLiveChannel()
        let model = MatchDetailModel(session: session, matchId: 88, mode: "spectate", channel: channel)
        await model.start()

        channel.send(WSMessageType.scoreUpdate, [
            "player1_score": 5,
            "player2_score": 2,
            "current_round": 5,
            "rounds": [["round_number": 9, "winner": 1]],
        ])

        guard case .loaded(let data) = model.phase else { return XCTFail("推送后应保持已加载") }
        XCTAssertEqual(data.matchData.player1Score, 5)
        XCTAssertEqual(data.matchData.player2Score, 2)
        XCTAssertEqual(data.matchData.currentRound, 5)
        // 普通比分推送不带局记录，不能凭空追加。
        XCTAssertEqual(data.roundRecords.count, 1)
        XCTAssertEqual(data.roundRecords.first?.roundNumber, 1)
        XCTAssertEqual(channel.syncRequests, 0)
        model.stop()
    }

    func testRoundEndRequestsSyncForTheMissingRoundRecord() async throws {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        server.stub("GET", "/api/public/match/detail", .json(publicResponse(status: 1)))
        let channel = FakeLiveChannel()
        let model = MatchDetailModel(session: session, matchId: 88, mode: "spectate", channel: channel)
        await model.start()

        channel.send(WSMessageType.roundEnd, [
            "player1_score": 4,
            "player2_score": 1,
            "current_round": 2,
            "round_number": 1,
        ])
        await waitUntil { channel.syncRequests == 1 }

        guard case .loaded(let data) = model.phase else { return XCTFail("应保持已加载") }
        XCTAssertEqual(data.matchData.player1Score, 4)
        model.stop()
    }

    func testSyncAppliesRoundsAndRefereeFromTheSnapshot() async throws {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        server.stub("GET", "/api/public/match/detail", .json(publicResponse(status: 1)))
        let channel = FakeLiveChannel()
        let model = MatchDetailModel(session: session, matchId: 88, mode: "spectate", channel: channel)
        await model.start()

        channel.send(WSMessageType.sync, [
            "player1_score": 4,
            "player2_score": 2,
            "total_rounds": 9,
            "referee_bound": true,
            "referee_name": "裁判小李",
            "rounds": [
                ["round_number": 1, "winner": 1],
                ["round_number": 2, "winner": 2],
            ],
        ])

        guard case .loaded(let data) = model.phase else { return XCTFail("应保持已加载") }
        XCTAssertEqual(data.matchData.totalRounds, 9)
        XCTAssertTrue(data.matchData.refereeBound)
        XCTAssertEqual(data.matchData.refereeName, "裁判小李")
        XCTAssertEqual(data.roundRecords.map(\.roundNumber), [1, 2])
        model.stop()
    }

    func testMatchEndReloadsTheDetailInsteadOfTrustingThePush() async throws {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        server.stub("GET", "/api/public/match/detail", .json(publicResponse(status: 1, player1Score: 3, player2Score: 1)))
        let channel = FakeLiveChannel()
        let model = MatchDetailModel(session: session, matchId: 88, mode: "spectate", channel: channel)
        await model.start()

        server.stub("GET", "/api/public/match/detail", .json(publicResponse(status: 2, player1Score: 9, player2Score: 4)))
        channel.send(WSMessageType.matchEnd, ["status": 2, "player1_score": 9, "player2_score": 4])

        await waitUntil { server.requestCount("/api/public/match/detail") == 2 }
        await waitUntil { model.viewData?.matchData.status == 2 }
        guard case .loaded(let data) = model.phase else { return XCTFail("应保持已加载") }
        XCTAssertEqual(data.matchData.player1Score, 9)
        XCTAssertEqual(data.matchData.player2Score, 4)
        model.stop()
    }

    // MARK: - 九球追分静默补读

    func testNineBallPushRefreshesAfterARandomDelayWithoutATicket() async throws {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        server.stub("GET", "/api/public/match/detail", .json(publicResponse(status: 1, gameType: 2)))
        let channel = FakeLiveChannel()
        let recorder = DelayRecorder()
        // 只替换 sleep：延迟仍用默认的 400–1000ms 随机值。
        let model = MatchDetailModel(
            session: session,
            matchId: 88,
            mode: "spectate",
            channel: channel,
            sleep: { recorder.record($0) }
        )
        await model.start()

        channel.send(WSMessageType.scoreUpdate, ["player1_score": 6, "player2_score": 3])

        await waitUntil { server.requestCount("/api/public/match/detail") == 2 }
        let delay = try XCTUnwrap(recorder.delays.first)
        XCTAssertGreaterThanOrEqual(delay, 0.4)
        XCTAssertLessThan(delay, 1.0)
        model.stop()
    }

    func testSilentRefreshFailureKeepsThePreviousScores() async throws {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        server.stub("GET", "/api/public/match/detail", .json(publicResponse(status: 1, gameType: 2)))
        let channel = FakeLiveChannel()
        let model = MatchDetailModel(
            session: session,
            matchId: 88,
            mode: "spectate",
            channel: channel,
            sleep: { _ in }
        )
        await model.start()

        server.stub("GET", "/api/public/match/detail", .json(MatchHistoryFixtures.businessFailure))
        channel.send(WSMessageType.scoreUpdate, ["player1_score": 6, "player2_score": 3])

        await waitUntil { server.requestCount("/api/public/match/detail") == 2 }
        await waitUntil { model.viewData?.matchData.player1Score == 6 }
        guard case .loaded(let data) = model.phase else { return XCTFail("刷新失败不得切回错误页") }
        XCTAssertEqual(data.matchData.player1Score, 6)
        model.stop()
    }

    // MARK: - 失败与身份

    func testBusinessFailureShowsTheSharedFeedback() async throws {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        server.stub("GET", "/api/match/detail", .json(MatchHistoryFixtures.businessFailure))
        let model = MatchDetailModel(session: session, matchId: 101, channel: FakeLiveChannel())

        await model.start()

        guard case .failed(let failure) = model.phase else { return XCTFail("应进入失败态：\(model.phase)") }
        XCTAssertEqual(failure.category, .business)
        XCTAssertEqual(failure.feedback(resource: "对局").title, "对局加载失败")
        XCTAssertEqual(failure.message, "暂时无法查看这场对局")
    }

    func testNotFoundFailureKeepsTheReturnAction() async throws {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        server.stub("GET", "/api/match/detail", .json(["success": false], status: 404))
        let model = MatchDetailModel(session: session, matchId: 101, channel: FakeLiveChannel())

        await model.start()

        guard case .failed(let failure) = model.phase else { return XCTFail("应进入失败态：\(model.phase)") }
        XCTAssertEqual(failure.category, .notFound)
        XCTAssertFalse(failure.feedback(resource: "对局").retryable)
    }

    func testLateDetailAfterLogoutIsDiscarded() async throws {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        let gate = DispatchSemaphore(value: 0)
        server.stub("GET", "/api/match/detail") { _ in
            _ = gate.wait(timeout: .now() + 5)
            return .json(MatchHistoryFixtures.playerDetail(status: 1))
        }
        let channel = FakeLiveChannel()
        let model = MatchDetailModel(session: session, matchId: 101, channel: channel)

        let load = Task { await model.load() }
        await waitUntil { server.requestCount("/api/match/detail") == 1 }
        session.logout()
        gate.signal()
        await load.value

        guard case .loading = model.phase else {
            return XCTFail("退出后旧账号的详情不得写入：\(model.phase)")
        }
        XCTAssertTrue(channel.openCalls.isEmpty)
    }

    // MARK: - 夹具

    /// `GET /api/public/match/detail`：公开详情是选手1 / 选手2 固定视角。
    private func publicResponse(
        status: Int,
        gameType: Int = 3,
        player1Score: Int = 3,
        player2Score: Int = 1
    ) -> [String: Any] {
        [
            "success": true,
            "match": [
                "id": 88,
                "game_type": gameType,
                "status": status,
                "server_revision": 7,
                "player1_id": 11,
                "player1_name": "选手甲",
                "player2_id": 12,
                "player2_name": "选手乙",
                "player1_score": player1Score,
                "player2_score": player2Score,
                "current_round": 2,
                "total_rounds": 7,
                "duration_seconds": 1280,
                "rounds": [["round_number": 1, "player1_score": 1, "player2_score": 0, "winner": 1]],
            ],
        ]
    }
}
