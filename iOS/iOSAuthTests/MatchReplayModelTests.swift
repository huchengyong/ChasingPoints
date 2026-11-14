import Foundation
import XCTest
@testable import iOS

/// 单场复盘页的读取状态：身份、失败分类与迟到响应。
@MainActor
final class MatchReplayModelTests: XCTestCase {
    func testReplayLoadsWithTheMatchIdAndResolvesTheMemberView() async throws {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        server.stub("GET", "/api/member/match-replay", .json(MatchReplayFixtures.snooker))
        let model = MatchReplayModel(session: session, matchId: 501)

        await model.load()

        guard case .loaded(let view) = model.phase else { return XCTFail("复盘应加载成功：\(model.phase)") }
        XCTAssertEqual(view.title, "中式八球复盘")
        XCTAssertEqual(view.scoreText, "1 : 1")
        XCTAssertEqual(view.frames?.count, 2)
        XCTAssertEqual(server.requests("/api/member/match-replay").first?.query["match_id"], "501")
    }

    func testServerRefusalIsReportedAsNotFound() async throws {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        server.stub("GET", "/api/member/match-replay", .json(["success": false]))
        let model = MatchReplayModel(session: session, matchId: 501)

        await model.load()

        guard case .failed(let failure) = model.phase else { return XCTFail("拒绝应进入失败态：\(model.phase)") }
        XCTAssertEqual(failure.feedback(resource: "单场复盘").title, "资源不存在或已下线")
    }

    func testMissingMatchIdFailsWithoutRequesting() async throws {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        let model = MatchReplayModel(session: session, matchId: 0)

        await model.load()

        guard case .failed(let failure) = model.phase else { return XCTFail("缺少对局信息应进入失败态") }
        XCTAssertEqual(failure.category, .notFound)
        XCTAssertEqual(server.requestCount("/api/member/match-replay"), 0)
    }

    func testReplayLoadedAfterLogoutIsDiscarded() async throws {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        let gate = DispatchSemaphore(value: 0)
        server.stub("GET", "/api/member/match-replay") { _ in
            _ = gate.wait(timeout: .now() + 5)
            return .json(MatchReplayFixtures.snooker)
        }
        let model = MatchReplayModel(session: session, matchId: 501)

        let load = Task { await model.load() }
        await waitUntil { server.requestCount("/api/member/match-replay") == 1 }
        session.logout()
        gate.signal()
        await load.value

        guard case .loading = model.phase else {
            return XCTFail("退出后旧账号的复盘不得写入：\(model.phase)")
        }
    }

    func testLoadWithoutSessionDoesNotRequest() async throws {
        let server = MockHTTPServer()
        let session = makeSessionStore(server, credentialStore: InMemoryCredentialStore(credentials: nil))
        let model = MatchReplayModel(session: session, matchId: 501)

        await model.load()

        XCTAssertEqual(server.requestCount("/api/member/match-replay"), 0)
    }
}

enum MatchReplayFixtures {
    static let snooker: [String: Any] = [
        "success": true,
        "game_type": 3,
        "locked": false,
        "frame_count": 2,
        "summary": [
            "my_frame_wins": 1,
            "opponent_frame_wins": 1,
            "my_max_run": 45,
            "opponent_max_run": 30,
            "win_type_stats": [
                ["win_type": "normal", "label": "普胜", "my_count": 1, "opponent_count": 1],
            ],
        ],
        "frames": [
            [
                "frame_no": 1,
                "winner": 1,
                "my_score": 60,
                "opponent_score": 45,
                "win_type": "normal",
                "duration_seconds": 600,
                "max_deficit": 12,
                "key_tags": ["comeback"],
                "my_max_run": 45,
                "opponent_max_run": 30,
                "trend": [
                    ["step": 1, "diff": 5],
                    ["step": 2, "diff": -3],
                    ["step": 3, "diff": 12],
                ],
            ],
            [
                "frame_no": 2,
                "winner": 2,
                "my_score": 30,
                "opponent_score": 70,
                "max_deficit": 0,
                "duration_seconds": 540,
                "key_tags": ["decider"],
                "trend": [
                    ["step": 1, "diff": -4],
                    ["step": 2, "diff": -10],
                ],
            ],
        ],
    ]
}
