import SwiftUI
import XCTest
@testable import iOS

/// 实时计分页关键状态的亮 / 暗快照；只断言能渲染，不做像素比对。
@MainActor
final class LiveMatchSnapshotTests: XCTestCase {
    private let size = CGSize(width: 393, height: 852)

    func testEightBallPlayerSnapshot() async throws {
        let server = MockHTTPServer()
        stubDetail(server, detailJSON(viewerRole: "player1", canScore: true, canUndo: true))
        try await render("live-eight-ball-player", server: server)
    }

    func testEightBallRefereeSnapshot() async throws {
        let server = MockHTTPServer()
        stubDetail(server, detailJSON(
            viewerRole: "referee",
            canScore: true,
            canUndo: true,
            canFinish: true,
            refereeBound: true
        ))
        try await render("live-eight-ball-referee", server: server)
    }

    func testAmericanNinePlayerSnapshot() async throws {
        let server = MockHTTPServer()
        stubDetail(server, detailJSON(gameType: 4, viewerRole: "player1"))
        try await render("live-american-nine-player", server: server)
    }

    func testSpectatorSnapshot() async throws {
        let server = MockHTTPServer()
        server.stub("GET", "/api/public/match/detail", .json(["success": true, "match": publicDetailJSON()]))
        try await render("live-spectator", server: server, entry: .spectator)
    }

    func testFinishCountdownSnapshot() async throws {
        let server = MockHTTPServer()
        stubDetail(server, detailJSON(
            canFinish: false,
            canRequestFinish: true,
            canWithdrawFinish: true,
            finishState: "pending_confirmation",
            finishRequestedBy: 1,
            remainingSeconds: 20
        ))
        try await render("live-finish-countdown", server: server)
    }

    func testFinishConfirmDialogSnapshot() async throws {
        let server = MockHTTPServer()
        stubDetail(server, detailJSON(
            canFinish: false,
            canConfirmFinish: true,
            canDisputeFinish: true,
            finishState: "pending_confirmation",
            finishRequestedBy: 2,
            remainingSeconds: 20
        ))
        try await render("live-finish-confirm-dialog", server: server)
    }

    func testRefereeQRCodeSnapshot() async throws {
        let server = MockHTTPServer()
        stubDetail(server, detailJSON(viewerRole: "player1"))
        server.stub("GET", "/api/match/referee/qrcode", .json([
            "success": true,
            "qrcode_data": "https://chasingpoints.example.com/referee/scan?token=fixture",
            "expires_in_seconds": 600,
        ]))
        let session = await makeRestoredAccountSession(server)
        let model = LiveMatchModel(session: session, matchId: 88)
        await model.start()
        await model.openRefereeQRCode()

        try await render(name: "live-referee-qrcode", session: session, model: model)
    }

    func testLoadErrorSnapshot() async throws {
        let server = MockHTTPServer()
        server.stub("GET", "/api/match/detail", .json(["success": false, "message": "对局不存在"], status: 404))
        try await render("live-load-error", server: server)
    }

    // MARK: - 渲染

    private func render(
        _ name: String,
        server: MockHTTPServer,
        entry: LiveMatchEntry = .automatic
    ) async throws {
        let session = await makeRestoredAccountSession(server)
        let model = LiveMatchModel(session: session, matchId: 88, entry: entry)
        await model.start()
        try await render(name: name, session: session, model: model)
    }

    private func render(name: String, session: SessionStore, model: LiveMatchModel) async throws {
        let navigator = AppNavigator()
        let toast = ToastCenter()
        let activity = ActivityStore(session: session)
        for scheme in [ColorScheme.light, .dark] {
            let image = try renderSnapshot(name, colorScheme: scheme, size: size) {
                NavigationStack {
                    LiveMatchView(session: session, matchId: 88, model: model)
                }
                .environmentObject(session)
                .environmentObject(navigator)
                .environmentObject(activity)
                .environmentObject(toast)
            }
            XCTAssertEqual(image.size, size)
        }
    }

    // MARK: - 夹具

    private func stubDetail(_ server: MockHTTPServer, _ json: [String: Any]) {
        server.stub("GET", "/api/match/detail", .json(["success": true, "match": json]))
    }

    private func detailJSON(
        gameType: Int = 3,
        viewerRole: String = "player1",
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
        refereeBound: Bool = false
    ) -> [String: Any] {
        var json: [String: Any] = [
            "id": 88,
            "challenge_id": 0,
            "game_type": gameType,
            "match_mode": "ranked",
            "status": 1,
            "server_revision": 7,
            "is_player1": true,
            "viewer_role": viewerRole,
            "referee_bound": refereeBound,
            "referee_name": refereeBound ? "裁判小李" : "",
            "can_score": canScore,
            "can_undo": canUndo,
            "can_finish": canFinish,
            "can_request_finish": canRequestFinish,
            "can_confirm_finish": canConfirmFinish,
            "can_dispute_finish": canDisputeFinish,
            "can_withdraw_finish": canWithdrawFinish,
            "finish_state": finishState,
            "finish_requested_by": finishRequestedBy,
            "my_score": 3,
            "opponent_score": 1,
            "current_frame_started": true,
            "current_frame_my_score": 1,
            "current_frame_opponent_score": 0,
            "current_round": 4,
            "match_format": "race_to",
            "target_wins": 7,
            "can_change_match_format": true,
            "my_name": "我",
            "opponent_name": "对手",
            "my_win_rate": 55,
            "opponent_win_rate": 45,
            "duration_seconds": 1280,
            "last_action": ["description": "我方赢下本局"],
        ]
        if let remainingSeconds {
            json["finish_request_remaining_seconds"] = remainingSeconds
        }
        return json
    }

    private func publicDetailJSON() -> [String: Any] {
        [
            "id": 88,
            "game_type": 3,
            "status": 1,
            "server_revision": 7,
            "player1_id": 11,
            "player1_name": "选手甲",
            "player2_id": 12,
            "player2_name": "选手乙",
            "player1_score": 3,
            "player2_score": 1,
            "current_frame_started": true,
            "current_round": 4,
            "match_format": "race_to",
            "target_wins": 7,
            "duration_seconds": 1280,
        ]
    }
}
