import Foundation
@testable import iOS

/// 扫码开局 / 执裁 / 选择球友的虚构夹具，字段与 `backend/chasing_points.api` 的实际响应一致。
enum MatchStartFixtures {
    static let inviteToken = "secret.token"
    static let refereeJoinToken = "referee-secret"

    static func startResponse(
        success: Bool = true,
        action: String? = "created",
        blockReason: String? = nil,
        message: String? = nil,
        matchId: Int64? = 55,
        match: CurrentMatchInfo? = nil,
        ongoingMatch: CurrentMatchInfo? = nil
    ) -> StartMatchResponse {
        StartMatchResponse(
            success: success,
            action: action,
            blockReason: blockReason,
            message: message,
            matchId: matchId,
            match: match,
            ongoingMatch: ongoingMatch
        )
    }

    static func startHandoff(
        identity: SessionIdentity,
        inviteToken: String = inviteToken,
        opponent: MatchScanOpponent = MatchScanOpponent(id: 9, nickname: "对手乙", avatar: ""),
        preset: ScanRematchPreset? = nil,
        rematch: ScanRematch? = nil
    ) -> ScanHandoff {
        ScanHandoff(
            identity: identity,
            mode: .start,
            inviteToken: inviteToken,
            opponent: opponent,
            refereeJoin: nil,
            preset: preset,
            rematch: rematch
        )
    }

    static func refereeHandoff(identity: SessionIdentity) -> ScanHandoff {
        ScanHandoff(
            identity: identity,
            mode: .referee,
            inviteToken: nil,
            opponent: nil,
            refereeJoin: RefereeJoin(matchId: 77, joinToken: refereeJoinToken),
            preset: nil,
            rematch: nil
        )
    }

    static let refereePreviewJSON: [String: Any] = [
        "player1_id": 1,
        "player1_name": "甲",
        "player1_avatar": "",
        "player2_id": 2,
        "player2_name": "乙",
        "player2_avatar": "",
        "player1_score": 3,
        "player2_score": 2,
        "game_type": 3,
        "game_type_name": "中式八球",
        "match_mode": "ranked",
        "visibility": "public",
        "status": 1,
        "status_text": "进行中",
    ]

    static let refereePreview = RefereePreviewInfo(
        player1Id: 1,
        player1Name: "甲",
        player1Avatar: "",
        player2Id: 2,
        player2Name: "乙",
        player2Avatar: "",
        player1Score: 3,
        player2Score: 2,
        gameType: 3,
        gameTypeName: "中式八球",
        matchMode: "ranked",
        visibility: "public",
        status: 1,
        statusText: "进行中"
    )

    static func opponentCandidatesJSON() -> [String: Any] {
        [
            "success": true,
            "list": [
                ["user_id": 20, "name": "好友A", "avatar": "friend-a.png", "source": "friend"],
                ["user_id": 30, "name": "最近B", "avatar": "b.png", "source": "recent"],
                ["user_id": 40, "name": "最近C", "avatar": "c.png", "source": "recent"],
            ],
        ]
    }
}

/// 测试内可改写的身份盒子：用于模拟扫码期间切号 / 退出的迟到结果。
final class IdentityBox {
    var value: SessionIdentity?

    init(_ value: SessionIdentity?) {
        self.value = value
    }
}

/// 异步闸门：让测试在扫码请求挂起时切换账号，再放行结果。
actor AsyncGate {
    private var continuation: CheckedContinuation<Void, Never>?
    private var isOpen = false

    func wait() async {
        if isOpen { return }
        await withCheckedContinuation { continuation = $0 }
    }

    func open() {
        isOpen = true
        continuation?.resume()
        continuation = nil
    }
}
