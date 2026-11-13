import Foundation
import XCTest
@testable import iOS

/// 逐条对应小程序 `uniapp/tests/start-match.test.mjs`。
@MainActor
final class StartMatchTests: XCTestCase {
    func testValidateStartMatchPayloadRejectsEmptyOpponentId() {
        XCTAssertEqual(
            StartMatch.validate(StartMatchRequest(gameType: 3, opponentId: 0, opponentName: "球友A")),
            "请选择有效的平台对手"
        )
    }

    func testValidateStartMatchPayloadAcceptsPositiveOpponentId() {
        XCTAssertEqual(
            StartMatch.validate(StartMatchRequest(gameType: 3, opponentId: 2001, opponentName: "球友A")),
            ""
        )
    }

    func testValidateStartMatchPayloadSupportsFlexibleSnookerFormatsAndLegacyRequests() {
        XCTAssertEqual(
            StartMatch.validate(StartMatchRequest(
                gameType: 1, opponentId: 2001, snookerFormat: "free", snookerTargetWins: 0
            )),
            ""
        )
        XCTAssertEqual(
            StartMatch.validate(StartMatchRequest(
                gameType: 1, opponentId: 2001, snookerFormat: "free", snookerTargetWins: 1
            )),
            "自由局数不能设置目标胜局"
        )
        XCTAssertEqual(
            StartMatch.validate(StartMatchRequest(
                gameType: 1, opponentId: 2001, snookerFormat: "race_to", snookerTargetWins: 10
            )),
            ""
        )
        XCTAssertEqual(
            StartMatch.validate(StartMatchRequest(
                gameType: 1, opponentId: 2001, snookerFormat: "race_to", snookerTargetWins: 26
            )),
            "抢N局目标必须在1至25之间"
        )
        XCTAssertEqual(
            StartMatch.validate(StartMatchRequest(
                gameType: 1, opponentId: 2001, bestOfFrames: 4, startingActor: 1
            )),
            "斯诺克总局数必须为正奇数"
        )
        XCTAssertEqual(
            StartMatch.validate(StartMatchRequest(
                gameType: 1, opponentId: 2001, bestOfFrames: 3, startingActor: 0
            )),
            "请选择首局开球方"
        )
        XCTAssertEqual(
            StartMatch.validate(StartMatchRequest(
                gameType: 1, opponentId: 2001, bestOfFrames: 7, startingActor: 2
            )),
            ""
        )
        XCTAssertEqual(
            StartMatch.validate(StartMatchRequest(gameType: 1, opponentId: 2001, snookerRulesVersion: 1)),
            ""
        )
        XCTAssertEqual(
            StartMatch.validate(StartMatchRequest(gameType: 1, opponentId: 2001, snookerRulesVersion: 3)),
            "不支持的斯诺克规则版本"
        )
    }

    func testNormalizeStartMatchOptionsDefaultsPracticeToPrivateAndRankedToPublic() {
        XCTAssertEqual(
            StartMatch.normalizeOptions(matchMode: "practice"),
            NormalizedStartMatchOptions(matchMode: "practice", visibility: "private")
        )
        XCTAssertEqual(
            StartMatch.normalizeOptions(matchMode: "ranked"),
            NormalizedStartMatchOptions(matchMode: "ranked", visibility: "public")
        )
    }

    func testBuildStartMatchPayloadNeverSendsChallengeIdMatchesComeFromBothPlayersEntering() {
        // JS 还会传入 challengeId；Swift 的请求体没有这个字段，因此天然不会下发。
        XCTAssertEqual(
            StartMatch.buildPayload(
                gameType: 3,
                opponent: StartMatchOpponent(id: 2001, nickname: "球友A", avatar: "a.png"),
                matchMode: "practice",
                visibility: "public"
            ),
            StartMatchRequest(
                gameType: 3,
                opponentId: 2001,
                opponentName: "球友A",
                opponentAvatar: "a.png",
                matchMode: "practice",
                visibility: "public",
                matchFormat: "free",
                targetWins: 0
            )
        )
    }

    func testSignedInviteStartPayloadOmitsMutableOpponentIdentityFields() {
        let payload = StartMatch.buildPayload(
            gameType: 3,
            opponent: StartMatchOpponent(id: 2001, nickname: "仅用于预览"),
            matchMode: "practice",
            inviteToken: "signed.invite"
        )
        XCTAssertEqual(payload, StartMatchRequest(
            gameType: 3,
            inviteToken: "signed.invite",
            matchMode: "practice",
            visibility: "private",
            matchFormat: "free",
            targetWins: 0
        ))
        XCTAssertEqual(StartMatch.validate(payload), "")
    }

    func testValidateStartMatchPayloadSupportsPoolFreeAndRaceToFormats() {
        XCTAssertEqual(
            StartMatch.validate(StartMatchRequest(
                gameType: 3, opponentId: 2001, matchFormat: "free", targetWins: 0
            )),
            ""
        )
        XCTAssertEqual(
            StartMatch.validate(StartMatchRequest(
                gameType: 4, opponentId: 2001, matchFormat: "race_to", targetWins: 65
            )),
            ""
        )
        XCTAssertEqual(
            StartMatch.validate(StartMatchRequest(
                gameType: 3, opponentId: 2001, matchFormat: "free", targetWins: 1
            )),
            "自由局数不能设置目标胜局"
        )
        XCTAssertEqual(
            StartMatch.validate(StartMatchRequest(
                gameType: 4, opponentId: 2001, matchFormat: "race_to", targetWins: 66
            )),
            "抢N局目标必须在1至65之间"
        )
    }

    func testBuildStartMatchPayloadDefaultsNewSnookerMatchesToFreeFormat() {
        XCTAssertEqual(
            StartMatch.buildPayload(
                gameType: 1,
                opponent: StartMatchOpponent(id: 2001, nickname: "球友A")
            ),
            StartMatchRequest(
                gameType: 1,
                opponentId: 2001,
                opponentName: "球友A",
                opponentAvatar: "",
                matchMode: "ranked",
                visibility: "public",
                snookerRulesVersion: 2,
                snookerFormat: "free",
                snookerTargetWins: 0
            )
        )
    }

    func testScanStartPayloadKeepsTheSignedInviteAndNeverSendsChallengeId() {
        XCTAssertEqual(
            StartMatch.buildPayload(
                gameType: 3,
                opponent: StartMatchOpponent(userId: 2001, nickname: "扫码昵称", avatar: "scan.png"),
                matchMode: "practice",
                visibility: "private",
                inviteToken: "signed.invite"
            ),
            StartMatchRequest(
                gameType: 3,
                inviteToken: "signed.invite",
                matchMode: "practice",
                visibility: "private",
                matchFormat: "free",
                targetWins: 0
            )
        )
    }
}
