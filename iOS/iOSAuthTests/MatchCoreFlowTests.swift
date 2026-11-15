import Foundation
import XCTest
@testable import iOS

/// 对应小程序 `uniapp/tests/match-core-flow.test.mjs` 中仅剩的「再来一局」上下文用例；
/// 其余用例覆盖首页卡、结束动作、路由拼接与 Vue 源码检查，原生端由各自页面实现承接。
final class MatchCoreFlowTests: XCTestCase {
    func testRematchContextKeepsOpponentGameTypeAndPreviousMode() throws {
        let context = MatchCoreFlow.rematch(try report([
            "opponent_id": 2001,
            "opponent_name": "球友A",
            "opponent_avatar": "a.png",
            "game_type": 3,
            "match_mode": "practice",
            "visibility": "private",
        ]))

        // 昵称与头像不从旧对局猜测，扫码后由对手资料补全。
        XCTAssertEqual(context, ScanRematch(
            opponentId: 2001,
            gameType: 3,
            matchMode: "practice",
            visibility: "private"
        ))
    }

    func testRematchNeedsAnOpponentAndClampsModeAndVisibility() throws {
        XCTAssertNil(MatchCoreFlow.rematch(try report(["player2_id": 0])))

        let context = MatchCoreFlow.rematch(try report([
            "player2_id": 2002,
            "match_mode": "friendly",
            "visibility": "unlisted",
        ]))
        XCTAssertEqual(context?.opponentId, 2002)
        XCTAssertEqual(context?.matchMode, "practice")
        XCTAssertEqual(context?.visibility, "private")

        let noVisibility = MatchCoreFlow.rematch(try report(["player2_id": 2002]))
        XCTAssertNil(noVisibility?.visibility)
    }
}

private func report(_ overrides: [String: Any]) throws -> MatchReport {
    var json: [String: Any] = [
        "id": 101,
        "game_type": 3,
        "status": 2,
        "my_name": "我",
        "my_avatar": "",
        "my_score": 5,
        "my_win_rate": 50,
        "my_rank_change": 10,
        "opponent_name": "对手",
        "opponent_avatar": "",
        "opponent_score": 3,
        "opponent_win_rate": 50,
        "opponent_rank_change": -10,
        "created_at": "2026-10-09T20:30:00+08:00",
    ]
    for (key, value) in overrides {
        json[key] = value
    }
    let data = try JSONSerialization.data(withJSONObject: json)
    let decoder = JSONDecoder()
    decoder.keyDecodingStrategy = .convertFromSnakeCase
    return try decoder.decode(MatchReport.self, from: data)
}
