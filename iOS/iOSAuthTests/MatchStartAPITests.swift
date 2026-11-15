import XCTest
@testable import iOS

/// 扫码开局 / 执裁 / 选择球友的接口门面：路径、请求体与查询参数。
@MainActor
final class MatchStartAPITests: XCTestCase {
    func testPreviewMatchInviteTrimsTokenAndPostsInviteToken() async throws {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        server.stub("POST", "/api/match/invite/preview", .json([
            "success": true,
            "preview": ["opponent_id": 9, "opponent_name": "对手乙", "opponent_avatar": "a.png"],
        ]))

        let response = try await MatchStartAPI(session: session)
            .previewMatchInvite("  signed.token\n")

        XCTAssertEqual(response.preview?.opponentId, 9)
        let request = try XCTUnwrap(server.requests("/api/match/invite/preview").first)
        XCTAssertEqual(request.method, "POST")
        XCTAssertEqual(request.bodyJSON["invite_token"] as? String, "signed.token")
        XCTAssertNotNil(request.bearerToken)
    }

    func testResolveMatchInviteCodePostsInviteCode() async throws {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        server.stub("POST", "/api/match/invite/resolve", .json([
            "success": true,
            "invite_token": "header.payload",
            "preview": ["opponent_id": 9, "opponent_name": "对手乙", "opponent_avatar": ""],
        ]))

        let response = try await MatchStartAPI(session: session)
            .resolveMatchInviteCode(" Ab3dEf6hIj9k ")

        XCTAssertEqual(response.inviteToken, "header.payload")
        let request = try XCTUnwrap(server.requests("/api/match/invite/resolve").first)
        XCTAssertEqual(request.bodyJSON["invite_code"] as? String, "Ab3dEf6hIj9k")
    }

    func testPreviewAndJoinRefereePostTheOneTimeCredentials() async throws {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        server.stub("POST", "/api/match/referee/preview", .json([
            "success": true,
            "preview": MatchStartFixtures.refereePreviewJSON,
        ]))
        server.stub("POST", "/api/match/referee/join", .json([
            "success": true,
            "match_id": 77,
            "match": ["game_type": 3],
        ]))
        let api = MatchStartAPI(session: session)
        let join = RefereeJoin(matchId: 77, joinToken: MatchStartFixtures.refereeJoinToken)

        let preview = try await api.previewMatchReferee(join)
        XCTAssertEqual(preview.preview?.player1Name, "甲")

        let joined = try await api.joinMatchReferee(join)
        XCTAssertEqual(joined.matchId, 77)

        for path in ["/api/match/referee/preview", "/api/match/referee/join"] {
            let request = try XCTUnwrap(server.requests(path).first)
            XCTAssertEqual((request.bodyJSON["match_id"] as? NSNumber)?.int64Value, 77)
            XCTAssertEqual(request.bodyJSON["join_token"] as? String, MatchStartFixtures.refereeJoinToken)
        }
    }

    func testStartMatchPostsTheBuiltPayload() async throws {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        server.stub("POST", "/api/match/start", .json([
            "success": true,
            "action": "created",
            "match_id": 55,
        ]))
        let payload = StartMatch.buildPayload(
            gameType: 3,
            opponent: StartMatchOpponent(id: 9, nickname: "对手乙", avatar: "a.png"),
            inviteToken: MatchStartFixtures.inviteToken
        )

        let response = try await MatchStartAPI(session: session).startMatch(payload)

        XCTAssertEqual(response.matchId, 55)
        let request = try XCTUnwrap(server.requests("/api/match/start").first)
        XCTAssertEqual((request.bodyJSON["game_type"] as? NSNumber)?.intValue, 3)
        XCTAssertEqual(request.bodyJSON["invite_token"] as? String, MatchStartFixtures.inviteToken)
        XCTAssertEqual(request.bodyJSON["match_mode"] as? String, "ranked")
        XCTAssertEqual(request.bodyJSON["visibility"] as? String, "public")
        XCTAssertNil(request.bodyJSON["opponent_id"], "带 token 时不重复提交对手字段")
    }

    func testCurrentMatchReadsTheOngoingMatch() async throws {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        server.stub("GET", "/api/match/current", .json([
            "success": true,
            "match": ["match_id": 31, "game_type": 3, "opponent_name": "老对手"],
        ]))

        let response = try await MatchStartAPI(session: session).currentMatch()

        XCTAssertEqual(response.match?.matchId, 31)
        XCTAssertEqual(server.requests("/api/match/current").first?.method, "GET")
    }

    func testOpponentCandidatesSendsTheFixedLimit() async throws {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        server.stub("GET", "/api/opponent/candidates", .json(MatchStartFixtures.opponentCandidatesJSON()))

        let candidates = try await MatchStartAPI(session: session).opponentCandidates()

        XCTAssertEqual(candidates.map(\.userId), [20, 30, 40])
        XCTAssertEqual(candidates.map(\.sourceText), ["好友", "最近对手", "最近对手"])
        XCTAssertEqual(server.requests("/api/opponent/candidates").first?.query, ["limit": "30"])
    }
}
