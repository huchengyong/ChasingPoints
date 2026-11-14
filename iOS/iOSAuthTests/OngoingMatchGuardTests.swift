import XCTest
@testable import iOS

/// 逐条对应小程序 `uniapp/tests/ongoing-match-guard.test.mjs`。
/// `isPlayingMatchRoute` / `shouldPromptOngoingMatch` / `buildPlayingRoute` 依赖原生端尚未迁移的
/// 前台提醒与进行中页面，且没有 URL 字符串消费方，本轮未搬。
@MainActor
final class OngoingMatchGuardTests: XCTestCase {
    func testCreatedMatchNavigatesWithScannedOpponent() {
        let outcome = OngoingMatchGuard.resolveStartMatchGuardAction(
            response: MatchStartFixtures.startResponse(action: "created", matchId: 18),
            selectedGameType: 2,
            scannedOpponent: StartMatchOpponent(nickname: "球友A", avatar: "avatar-a.png")
        )

        XCTAssertEqual(outcome, .navigate(GuardedMatch(
            matchId: 18,
            gameType: 2,
            opponentName: "球友A",
            opponentAvatar: "avatar-a.png",
            opponentId: 0,
            matchMode: "ranked",
            visibility: "public",
            challengeId: 0
        )))
    }

    func testResumeExistingUsesOngoingMatchPayload() {
        let outcome = OngoingMatchGuard.resolveStartMatchGuardAction(
            response: MatchStartFixtures.startResponse(
                action: "resume_existing",
                matchId: 28,
                ongoingMatch: OngoingMatchRef(
                    id: 28,
                    gameType: 3,
                    opponentName: "球友B",
                    opponentAvatar: "avatar-b.png"
                )
            ),
            selectedGameType: 2,
            scannedOpponent: StartMatchOpponent(nickname: "球友C", avatar: "avatar-c.png")
        )

        XCTAssertEqual(outcome, .resume(GuardedMatch(
            matchId: 28,
            gameType: 3,
            opponentName: "球友B",
            opponentAvatar: "avatar-b.png",
            opponentId: 0,
            matchMode: "ranked",
            visibility: "public",
            challengeId: 0
        )))
    }

    func testSelfBlockPromptsWithActionableMatch() {
        let outcome = OngoingMatchGuard.resolveStartMatchGuardAction(
            response: MatchStartFixtures.startResponse(
                action: "blocked",
                blockReason: "self_ongoing",
                message: "你还有一场中式八球未结束，请先结束之前的对局",
                matchId: 38,
                ongoingMatch: OngoingMatchRef(
                    id: 38,
                    gameType: 3,
                    opponentName: "球友D",
                    opponentAvatar: "avatar-d.png"
                )
            ),
            selectedGameType: 2,
            scannedOpponent: StartMatchOpponent(nickname: "球友E", avatar: "avatar-e.png")
        )

        XCTAssertEqual(outcome, .promptSelfOngoing(
            message: "你还有一场中式八球未结束，请先结束之前的对局",
            match: GuardedMatch(
                matchId: 38,
                gameType: 3,
                opponentName: "球友D",
                opponentAvatar: "avatar-d.png",
                opponentId: 0,
                matchMode: "ranked",
                visibility: "public",
                challengeId: 0
            )
        ))
    }

    func testOpponentBlockIsAToastWithoutNavigationTarget() {
        let outcome = OngoingMatchGuard.resolveStartMatchGuardAction(
            response: MatchStartFixtures.startResponse(
                action: "blocked",
                blockReason: "opponent_ongoing",
                message: "对手还有未结束的对局，暂时无法开始新的 PK"
            ),
            selectedGameType: 3,
            scannedOpponent: StartMatchOpponent(nickname: "球友F", avatar: "avatar-f.png")
        )

        XCTAssertEqual(outcome, .toastOpponentOngoing(message: "对手还有未结束的对局，暂时无法开始新的 PK"))
    }

    func testFailedResponseAndUnknownActionReportTheDefaultCreationError() {
        XCTAssertEqual(
            OngoingMatchGuard.resolveStartMatchGuardAction(
                response: MatchStartFixtures.startResponse(success: false, action: nil, message: "匹配二维码已失效"),
                selectedGameType: 3
            ),
            .error(message: "匹配二维码已失效")
        )
        XCTAssertEqual(
            OngoingMatchGuard.resolveStartMatchGuardAction(
                response: MatchStartFixtures.startResponse(success: false, action: nil, message: nil),
                selectedGameType: 3
            ),
            .error(message: "创建对局失败")
        )
        XCTAssertEqual(
            OngoingMatchGuard.resolveStartMatchGuardAction(
                response: MatchStartFixtures.startResponse(action: "unexpected"),
                selectedGameType: 3
            ),
            .error(message: "创建对局失败")
        )
    }

    func testNormalizeFallsBackToDefaultsForMissingFields() {
        XCTAssertEqual(OngoingMatchGuard.normalizeMatchPayload(nil), GuardedMatch(
            matchId: 0,
            gameType: 3,
            opponentName: "对手",
            opponentAvatar: "",
            opponentId: 0,
            matchMode: "ranked",
            visibility: "public",
            challengeId: 0
        ))

        XCTAssertEqual(
            OngoingMatchGuard.normalizeMatchPayload(
                OngoingMatchRef(matchId: 7),
                fallback: GuardedMatchFallback(
                    gameType: 4,
                    opponentName: "球友G",
                    opponentAvatar: "avatar-g.png",
                    opponentId: 8,
                    challengeId: 9
                )
            ),
            GuardedMatch(
                matchId: 7,
                gameType: 4,
                opponentName: "球友G",
                opponentAvatar: "avatar-g.png",
                opponentId: 8,
                matchMode: "ranked",
                visibility: "public",
                challengeId: 9
            )
        )
    }
}
