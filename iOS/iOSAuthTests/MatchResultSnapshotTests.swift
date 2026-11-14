import SwiftUI
import XCTest
@testable import iOS

/// 赛果页与单场复盘页的亮 / 暗快照；只断言能渲染，不做像素比对。
@MainActor
final class MatchResultSnapshotTests: XCTestCase {
    private let size = CGSize(width: 393, height: 852)

    func testPlayerResultSnapshot() async throws {
        let server = MockHTTPServer()
        stubReport(server)
        try await renderReport("match-result-player", server: server)
    }

    func testHistoryResultSnapshot() async throws {
        let server = MockHTTPServer()
        stubReport(server)
        try await renderReport("match-result-history", server: server, fromHistory: true)
    }

    func testRefereeResultSnapshot() async throws {
        let server = MockHTTPServer()
        server.stub("GET", "/api/match/detail", .json(MatchHistoryFixtures.refereeDetail()))
        try await renderReport("match-result-referee", server: server)
    }

    func testReplayMemberSnapshot() async throws {
        let server = MockHTTPServer()
        server.stub("GET", "/api/member/match-replay", .json(MatchReplayFixtures.snooker))
        try await renderReplay("match-replay-member", server: server, matchId: 501)
    }

    func testReplayLockedSnapshot() async throws {
        let server = MockHTTPServer()
        server.stub("GET", "/api/member/match-replay", .json([
            "success": true,
            "game_type": 3,
            "locked": true,
            "frame_count": 5,
        ]))
        try await renderReplay("match-replay-locked", server: server, matchId: 501)
    }

    // MARK: - 渲染

    private func renderReport(
        _ name: String,
        server: MockHTTPServer,
        fromHistory: Bool = false
    ) async throws {
        let session = await makeRestoredAccountSession(server)
        let model = MatchReportModel(session: session, matchId: 101, retryDelay: {})
        await model.load()
        guard case .loaded = model.phase else {
            return XCTFail("赛果页应加载成功：\(model.phase)")
        }

        let navigator = AppNavigator()
        let toast = ToastCenter()
        for scheme in [ColorScheme.light, .dark] {
            let image = try renderSnapshot(name, colorScheme: scheme, size: size) {
                NavigationStack {
                    MatchReportView(
                        session: session,
                        matchId: 101,
                        fromHistory: fromHistory,
                        model: model
                    )
                }
                .environmentObject(session)
                .environmentObject(navigator)
                .environmentObject(toast)
            }
            XCTAssertEqual(image.size, size)
        }
    }

    private func renderReplay(_ name: String, server: MockHTTPServer, matchId: Int64) async throws {
        let session = await makeRestoredAccountSession(server)
        let model = MatchReplayModel(session: session, matchId: matchId)
        await model.load()
        guard case .loaded = model.phase else {
            return XCTFail("复盘页应加载成功：\(model.phase)")
        }

        let navigator = AppNavigator()
        for scheme in [ColorScheme.light, .dark] {
            let image = try renderSnapshot(name, colorScheme: scheme, size: size) {
                NavigationStack {
                    MatchReplayView(session: session, matchId: matchId, model: model)
                }
                .environmentObject(session)
                .environmentObject(navigator)
            }
            XCTAssertEqual(image.size, size)
        }
    }

    // MARK: - 夹具

    private func stubReport(_ server: MockHTTPServer) {
        server.stub("GET", "/api/match/detail", .json(MatchHistoryFixtures.playerDetail(
            highlights: [
                ["label": "最长连赢", "value": "4 局", "desc": "本场最多连续拿下四局", "tone": "gold"],
                ["label": "最高单杆", "value": "68 分", "desc": "关键局一杆制胜", "tone": "green"],
            ],
            stats: [
                ["label": "总得分", "value": "312", "tone": "blue"],
                ["label": "犯规", "value": "3 次", "tone": "red"],
            ]
        )))
        server.stub("GET", "/api/match/reward-summary", .json(MatchHistoryFixtures.rewardSummary(
            list: [MatchHistoryFixtures.rewardItem()]
        )))
        server.stub("GET", "/api/gear/match", .json([
            "success": true,
            "match_id": 101,
            "game_type": 3,
            "supports_break": true,
            "me": [
                "user_id": 1,
                "recorded": true,
                "cue": ["brand_name": "美洲豹", "model_name": "P3", "category": "cue"],
                "tip_product": ["brand_name": "卡姆", "name": "黑冰", "category": "tip"],
                "chalk_product": ["brand_name": "三角", "name": "绿巧", "category": "chalk"],
                "break_cue": ["brand_name": "毒药", "model_name": "B1", "category": "cue"],
            ],
            "has_opponent": true,
            "opponent": [
                "user_id": 2,
                "recorded": true,
                "cue": ["brand_name": "美洲豹", "model_name": "P3", "category": "cue"],
            ],
            "editable": true,
            "edit_deadline": "2026-10-10T20:30:00+08:00",
        ]))
    }
}
