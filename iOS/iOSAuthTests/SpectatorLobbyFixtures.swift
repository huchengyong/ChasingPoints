import Foundation
@testable import iOS

/// 观赛大厅的虚构夹具，字段与 `backend/chasing_points.api` 的 PublicMatchListItem 一致。
enum SpectatorLobbyFixtures {
    static func match(
        id: Int64,
        status: Int = 1,
        gameType: Int = 3,
        gameTypeName: String = "中式八球",
        matchMode: String? = "practice",
        player1Id: Int64 = 11,
        player1Name: String = "甲",
        player1Avatar: String = "",
        player1Score: Int = 1,
        player2Id: Int64 = 22,
        player2Name: String = "乙",
        player2Avatar: String = "",
        player2Score: Int = 0,
        durationSeconds: Int64 = 600,
        endTime: String? = nil
    ) -> SpectatorMatch {
        SpectatorMatch(
            id: id,
            gameType: gameType,
            gameTypeName: gameTypeName,
            matchMode: matchMode,
            status: status,
            player1Id: player1Id,
            player1Name: player1Name,
            player1Avatar: player1Avatar,
            player2Id: player2Id,
            player2Name: player2Name,
            player2Avatar: player2Avatar,
            player1Score: player1Score,
            player2Score: player2Score,
            durationSeconds: durationSeconds,
            endTime: endTime
        )
    }

    static func itemJSON(
        id: Int64,
        status: Int = 1,
        gameType: Int = 3,
        gameTypeName: String = "中式八球",
        matchMode: String? = "practice",
        player1Id: Int64 = 11,
        player1Name: String = "甲",
        player1Avatar: String = "",
        player1Score: Int = 1,
        player2Id: Int64 = 22,
        player2Name: String = "乙",
        player2Avatar: String = "",
        player2Score: Int = 0,
        durationSeconds: Int64 = 600,
        endTime: String? = nil
    ) -> [String: Any] {
        var item: [String: Any] = [
            "id": id,
            "game_type": gameType,
            "game_type_name": gameTypeName,
            "status": status,
            "player1_id": player1Id,
            "player1_name": player1Name,
            "player1_avatar": player1Avatar,
            "player1_score": player1Score,
            "player2_id": player2Id,
            "player2_name": player2Name,
            "player2_avatar": player2Avatar,
            "player2_score": player2Score,
            "duration_seconds": durationSeconds,
        ]
        if let matchMode {
            item["match_mode"] = matchMode
        }
        if let endTime {
            item["end_time"] = endTime
        }
        return item
    }

    static func listResponse(_ items: [[String: Any]], total: Int? = nil) -> MockHTTPServer.Stub {
        .json([
            "success": true,
            "total": total ?? items.count,
            "list": items,
        ])
    }

    @MainActor
    static func makeGuestSession(_ server: MockHTTPServer) -> SessionStore {
        makeSessionStore(server, credentialStore: InMemoryCredentialStore(credentials: nil))
    }

    @MainActor
    static func makeSignedInSession(_ server: MockHTTPServer) async -> SessionStore {
        server.stub("GET", "/api/user/info", .json(AuthFixtures.userInfoResponse()))
        let session = makeSessionStore(
            server,
            credentialStore: InMemoryCredentialStore(credentials: AuthFixtures.credentials())
        )
        await session.restore()
        return session
    }
}
