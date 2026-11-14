import SwiftUI
import XCTest
@testable import iOS

/// 赛果页与单场复盘页的亮 / 暗快照；只断言能渲染，不做像素比对。
@MainActor
final class MatchResultSnapshotTests: XCTestCase {
    private let size = CGSize(width: 393, height: 852)

    func testWinResultSnapshot() async throws {
        let server = MockHTTPServer()
        stubReport(server)
        try await renderReport("match-result-win", server: server)
    }

    func testLoseRankedPendingSnapshot() async throws {
        let server = MockHTTPServer()
        stubReport(server, detail: MatchHistoryFixtures.playerDetail(
            myScore: 1,
            opponentScore: 3,
            rankSettlementStatus: "pending",
            rankSettlementDueAt: "2026-10-11T20:30:00+08:00"
        ))
        try await renderReport("match-result-lose-ranked-pending", server: server)
    }

    func testNineBallResultSnapshot() async throws {
        let server = MockHTTPServer()
        stubReport(server, detail: nineBallDetail())
        try await renderReport("match-result-nine-ball-detail", server: server)
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

    func testLoadErrorSnapshot() async throws {
        let server = MockHTTPServer()
        server.stub("GET", "/api/match/detail", .json(MatchHistoryFixtures.businessFailure))
        try await renderReport("match-result-error", server: server, expectLoaded: false)
    }

    /// 本场用杆编辑器：展开态且皮头已按所选球杆预填。
    func testGearEditSnapshot() async throws {
        let server = MockHTTPServer()
        server.stub("GET", "/api/gear/match", .json(gearMatchResponse()))
        server.stub("GET", "/api/gear/mine", .json(GearFixtures.mineJSON(items: [
            GearFixtures.itemJSON(
                7,
                brandName: "美洲豹",
                modelName: "P3",
                tip: GearFixtures.productJSON(31, category: "tip", brandName: "卡姆", name: "黑冰"),
                chalk: GearFixtures.productJSON(41, category: "chalk", brandName: "三角", name: "绿巧")
            ),
            GearFixtures.itemJSON(8, category: "break_jump", brandName: "毒药", modelName: "B1", roles: ["break"]),
        ])))
        let session = await makeRestoredAccountSession(server)
        let model = MatchGearCardModel(session: session, matchId: 101, enabled: true)
        await model.load()
        guard case .ready = model.phase else {
            return XCTFail("本场用杆应读取成功：\(model.phase)")
        }
        model.openEditor()
        await model.loadItems()
        guard model.editor.itemsError.isEmpty else {
            return XCTFail("装备列表应读取成功：\(model.editor.itemsError)")
        }
        model.chooseCue(model.cueOptions.first)

        let gearStore = GearStore(session: session)
        let navigator = AppNavigator()
        let toast = ToastCenter()
        for scheme in [ColorScheme.light, .dark] {
            let image = try renderSnapshot("match-result-gear-edit", colorScheme: scheme, size: size) {
                NavigationStack {
                    MatchGearEditorSheet(model: model)
                }
                .environmentObject(session)
                .environmentObject(gearStore)
                .environmentObject(navigator)
                .environmentObject(toast)
            }
            XCTAssertEqual(image.size, size)
        }
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

    func testReplayErrorSnapshot() async throws {
        let server = MockHTTPServer()
        server.stub("GET", "/api/member/match-replay", .json(["success": false], status: 404))
        try await renderReplay("match-replay-error", server: server, matchId: 501, expectLoaded: false)
    }

    // MARK: - 渲染

    private func renderReport(
        _ name: String,
        server: MockHTTPServer,
        fromHistory: Bool = false,
        expectLoaded: Bool = true
    ) async throws {
        let session = await makeRestoredAccountSession(server)
        let model = MatchReportModel(session: session, matchId: 101, retryDelay: {})
        await model.load()
        if expectLoaded {
            guard case .loaded = model.phase else {
                return XCTFail("赛果页应加载成功：\(model.phase)")
            }
        } else {
            guard case .failed = model.phase else {
                return XCTFail("赛果页应进入失败态：\(model.phase)")
            }
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

    private func renderReplay(
        _ name: String,
        server: MockHTTPServer,
        matchId: Int64,
        expectLoaded: Bool = true
    ) async throws {
        let session = await makeRestoredAccountSession(server)
        let model = MatchReplayModel(session: session, matchId: matchId)
        await model.load()
        if expectLoaded {
            guard case .loaded = model.phase else {
                return XCTFail("复盘页应加载成功：\(model.phase)")
            }
        } else {
            guard case .failed = model.phase else {
                return XCTFail("复盘页应进入失败态：\(model.phase)")
            }
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

    private func stubReport(_ server: MockHTTPServer, detail: [String: Any]? = nil) {
        server.stub("GET", "/api/match/detail", .json(detail ?? MatchHistoryFixtures.playerDetail(
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
        server.stub("GET", "/api/gear/match", .json(gearMatchResponse()))
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
            "editable": true,
            "edit_deadline": "2026-10-10T20:30:00+08:00",
        ]
    }

    /// 九球追分：按总分与得分明细展示，不走局记录。
    private func nineBallDetail() -> [String: Any] {
        var response = MatchHistoryFixtures.playerDetail(myScore: 16, opponentScore: 13)
        guard var match = response["match"] as? [String: Any] else { return response }
        match["game_type"] = 2
        match["score_breakdown"] = [
            "available": true,
            "player1": [
                "score": 16, "normal_wins": 2, "small_golds": 1, "big_golds": 0,
                "foul_count": 3, "foul_points_given": 3, "other_points": 0,
            ],
            "player2": [
                "score": 13, "normal_wins": 0, "small_golds": 0, "big_golds": 1,
                "foul_count": 1, "foul_points_given": 1, "other_points": 0,
            ],
            "events": [
                [
                    "seq": 1, "kind": "big_gold", "scorer": 1, "points": 10,
                    "player1_score": 5, "player2_score": 10, "created_at": "2026-11-10 12:00:03",
                ],
                [
                    "seq": 2, "kind": "foul", "scorer": 1, "points": 1,
                    "player1_score": 11, "player2_score": 10, "created_at": "2026-11-10 12:00:04",
                ],
            ],
            "events_truncated": false,
        ]
        response["match"] = match
        return response
    }
}
