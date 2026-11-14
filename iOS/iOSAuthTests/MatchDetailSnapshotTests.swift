import SwiftUI
import XCTest
@testable import iOS

/// 对局详情页的亮 / 暗快照；只断言能渲染，不做像素比对。
@MainActor
final class MatchDetailSnapshotTests: XCTestCase {
    private let size = CGSize(width: 393, height: 852)

    /// 快照不推进实时消息，只需要一个不连网的通道替身。
    private final class SnapshotChannel: LiveMatchChannel {
        var isConnected = false
        func openLive(matchId: Int64, allowAnonymous: Bool) async throws {}
        func requestLiveSync() async {}
        func setLiveActive(_ active: Bool) {}
        func closeLive() {}
        func observeLive(_ handler: @escaping (RealtimeMessage) -> Void) -> UUID { UUID() }
        func stopObservingLive(_ token: UUID) {}
    }

    func testPublicLiveSnapshot() async throws {
        let server = MockHTTPServer()
        server.stub("GET", "/api/public/match/detail", .json(publicResponse(status: 1)))
        try await render("match-detail-public", server: server, mode: "spectate")
    }

    func testPublicFinishedSnapshot() async throws {
        let server = MockHTTPServer()
        server.stub("GET", "/api/public/match/detail", .json(publicResponse(status: 2)))
        try await render("match-detail-public-finished", server: server, mode: "spectate")
    }

    func testHistoryPerspectiveSnapshot() async throws {
        let server = MockHTTPServer()
        server.stub("GET", "/api/public/match/detail", .json(publicResponse(status: 2)))
        try await render(
            "match-detail-history-perspective",
            server: server,
            source: "history",
            perspectiveUserId: 12
        )
    }

    func testLoadErrorSnapshot() async throws {
        let server = MockHTTPServer()
        server.stub("GET", "/api/match/detail", .json(MatchHistoryFixtures.businessFailure))
        try await render("match-detail-error", server: server)
    }

    /// 已结束的参与者视角：复盘 / 交锋记录 / 再约一场三个按钮，加本场用杆卡。
    func testParticipantFinishedSnapshot() async throws {
        let server = MockHTTPServer()
        server.stub("GET", "/api/match/detail", .json(MatchHistoryFixtures.playerDetail()))
        server.stub("GET", "/api/gear/match", .json(gearMatchResponse()))
        try await render("match-detail-participant", server: server, matchId: 101, gearEnabled: true)
    }

    // MARK: - 渲染

    private func render(
        _ name: String,
        server: MockHTTPServer,
        matchId: Int64 = 88,
        mode: String = "",
        source: String? = nil,
        perspectiveUserId: Int64? = nil,
        gearEnabled: Bool = false
    ) async throws {
        let session = await makeRestoredAccountSession(server)
        let model = MatchDetailModel(
            session: session,
            matchId: matchId,
            mode: mode,
            source: source ?? "",
            perspectiveUserId: perspectiveUserId ?? 0,
            channel: SnapshotChannel()
        )
        await model.start()

        let navigator = AppNavigator()
        let toast = ToastCenter()
        let gearStore = GearStore(session: session)
        let gearModel: MatchGearCardModel?
        if gearEnabled {
            let preloaded = MatchGearCardModel(session: session, matchId: matchId, enabled: true)
            await preloaded.load()
            gearModel = preloaded
        } else {
            gearModel = nil
        }
        for scheme in [ColorScheme.light, .dark] {
            let image = try renderSnapshot(name, colorScheme: scheme, size: size) {
                NavigationStack {
                    MatchDetailView(
                        session: session,
                        matchId: matchId,
                        mode: mode,
                        source: source,
                        perspectiveUserId: perspectiveUserId,
                        model: model,
                        gearModel: gearModel
                    )
                }
                .environmentObject(session)
                .environmentObject(navigator)
                .environmentObject(toast)
                .environmentObject(gearStore)
            }
            XCTAssertEqual(image.size, size)
        }
    }

    private func gearMatchResponse() -> [String: Any] {
        [
            "success": true,
            "message": "",
            "match_id": 101,
            "game_type": 3,
            "supports_break": true,
            "me": [
                "user_id": 1,
                "recorded": true,
                "cue": GearFixtures.itemJSON(7, brandName: "美洲豹", modelName: "P3"),
                "tip_product": GearFixtures.productJSON(31, category: "tip", brandName: "卡姆", name: "黑冰"),
                "chalk_product": GearFixtures.productJSON(41, category: "chalk", brandName: "三角", name: "绿巧"),
                "break_cue": GearFixtures.itemJSON(
                    8, category: "break_jump", brandName: "毒药", modelName: "B1", roles: ["break"]
                ),
            ],
            "has_opponent": true,
            "opponent": [
                "user_id": 2,
                "recorded": true,
                "cue": GearFixtures.itemJSON(9, brandName: "美洲豹", modelName: "P3"),
            ],
        ]
    }

    private func publicResponse(status: Int) -> [String: Any] {
        [
            "success": true,
            "match": [
                "id": 88,
                "game_type": 3,
                "status": status,
                "server_revision": 7,
                "player1_id": 11,
                "player1_name": "选手甲",
                "player2_id": 12,
                "player2_name": "选手乙",
                "player1_score": 3,
                "player2_score": 2,
                "current_round": 4,
                "total_rounds": 7,
                "duration_seconds": 1280,
                "rounds": [
                    ["round_number": 1, "player1_score": 1, "player2_score": 0, "winner": 1],
                    ["round_number": 2, "player1_score": 0, "player2_score": 1, "winner": 2],
                ],
            ],
        ]
    }
}
