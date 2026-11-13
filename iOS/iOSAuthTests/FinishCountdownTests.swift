import Foundation
import XCTest
@testable import iOS

/// 逐条对应小程序 `uniapp/tests/finish-confirmation.test.mjs` 里与 `finish-countdown.js` 相关的用例
/// （读取 `.vue` 源码、编译 SFC 与滑块手势的用例不搬）。
@MainActor
final class FinishCountdownTests: XCTestCase {
    private let receivedAt = Date(timeIntervalSince1970: 1_800_000_000)

    private func pending(
        finishState: String = "pending_confirmation",
        secondsLeft: Int = 25,
        isRequester: Bool = false,
        opponentName: String = "小李",
        myScore: Int = 3,
        opponentScore: Int = 5,
        canWithdrawFinish: Bool = false,
        canConfirmFinish: Bool = true,
        canDisputeFinish: Bool = true
    ) -> FinishConfirmInput {
        FinishConfirmInput(
            finishState: finishState,
            secondsLeft: secondsLeft,
            isRequester: isRequester,
            opponentName: opponentName,
            myScore: myScore,
            opponentScore: opponentScore,
            canWithdrawFinish: canWithdrawFinish,
            canConfirmFinish: canConfirmFinish,
            canDisputeFinish: canDisputeFinish
        )
    }

    func testCountdownStartsFromTheServerRemainingSecondsAndOnlyUsesLocalElapsedTime() {
        // 手机时钟比服务端快一小时也不影响：只用收到快照后的本地经过时间。
        let anchor = FinishCountdown.makeAnchor(remainingSeconds: 30, receivedAt: receivedAt)
        XCTAssertEqual(FinishCountdown.secondsLeft(anchor, now: receivedAt), 30)
        XCTAssertEqual(FinishCountdown.secondsLeft(anchor, now: receivedAt.addingTimeInterval(0.999)), 30)
        XCTAssertEqual(FinishCountdown.secondsLeft(anchor, now: receivedAt.addingTimeInterval(1)), 29)
        XCTAssertEqual(FinishCountdown.secondsLeft(anchor, now: receivedAt.addingTimeInterval(29.999)), 1)
        XCTAssertEqual(FinishCountdown.secondsLeft(anchor, now: receivedAt.addingTimeInterval(30)), 0)
        XCTAssertEqual(FinishCountdown.secondsLeft(anchor, now: receivedAt.addingTimeInterval(90)), 0)
        XCTAssertEqual(
            FinishCountdown.secondsLeft(anchor, now: receivedAt.addingTimeInterval(-60)),
            30,
            "本地时钟回拨不增加剩余时间"
        )
        XCTAssertEqual(
            FinishCountdown.secondsLeft(
                FinishCountdown.makeAnchor(remainingSeconds: 12, receivedAt: receivedAt),
                now: receivedAt.addingTimeInterval(5)
            ),
            7,
            "后续快照重新校准"
        )
        // JS 传非法秒数时会归零；Swift 的窄类型用 0 表达同样的起点。
        XCTAssertEqual(
            FinishCountdown.secondsLeft(
                FinishCountdown.makeAnchor(remainingSeconds: 0, receivedAt: receivedAt),
                now: receivedAt
            ),
            0
        )
        XCTAssertEqual(FinishCountdown.secondsLeft(nil, now: receivedAt), 0)
    }

    func testTheOtherPlayerGetsTheConfirmDialogWithCountdownSwipeConfirmAndDispute() {
        let view = FinishCountdown.resolveConfirmView(pending())
        XCTAssertTrue(view.visible)
        XCTAssertEqual(view.role, "responder")
        XCTAssertFalse(view.finishing)
        XCTAssertEqual(view.title, "小李申请结束对局")
        XCTAssertEqual(view.desc, "比分 3 : 5。25 秒内未处理，将按此比分自动结束，结束后不能再提出异议。")
        XCTAssertTrue(view.showConfirm)
        XCTAssertTrue(view.showDispute)
        XCTAssertFalse(view.showWithdraw)
    }

    func testTheRequesterSeesOnlyTheCountdownAndUndoNeverAConfirmEntry() {
        let view = FinishCountdown.resolveConfirmView(pending(
            isRequester: true,
            opponentName: "小王",
            myScore: 5,
            opponentScore: 3,
            canWithdrawFinish: true
        ))
        XCTAssertEqual(view.role, "requester")
        XCTAssertEqual(view.title, "等待小王确认")
        XCTAssertEqual(view.secondsLeft, 25)
        XCTAssertEqual(view.desc, "30秒内无人处理，将按 5 : 3 自动结束，之后不能再提出异议。")
        XCTAssertTrue(view.showWithdraw)
        XCTAssertFalse(view.showConfirm, "发起方没有确认入口")
        XCTAssertFalse(view.showDispute)
    }

    func testAtZeroOrOnceTheServerStopsOfferingActionsBothSidesOnlySeeFinishing() {
        let inputs = [
            pending(secondsLeft: 0),
            pending(secondsLeft: 12, canConfirmFinish: false, canDisputeFinish: false),
            pending(
                secondsLeft: 0,
                isRequester: true,
                canWithdrawFinish: true,
                canConfirmFinish: false,
                canDisputeFinish: false
            ),
        ]
        for input in inputs {
            let view = FinishCountdown.resolveConfirmView(input)
            XCTAssertTrue(view.visible)
            XCTAssertTrue(view.finishing, "\(input)")
            XCTAssertEqual(view.title, "正在结束…")
            XCTAssertNotNil(
                view.desc.range(
                    of: #"^确认期已满，正在按 \d+ : \d+ 结束对局。$"#,
                    options: .regularExpression
                )
            )
            XCTAssertFalse(view.showConfirm || view.showDispute || view.showWithdraw)
        }
        XCTAssertFalse(FinishCountdown.resolveConfirmView(FinishConfirmInput(finishState: "none")).visible)
        XCTAssertFalse(FinishCountdown.resolveConfirmView().visible)
    }

    func testSlideNoteExplainsThe30SecondConfirmationForRankedAndDirectFinishForPractice() {
        XCTAssertEqual(
            FinishCountdown.buildSlideNote(ranked: true, opponentName: "小王", myScore: 5, opponentScore: 3),
            "松手后需小王在30秒内确认，期间你可撤销；无人处理则按 5 : 3 自动结束，之后不能再提出异议。"
        )
        XCTAssertEqual(
            FinishCountdown.buildSlideNote(ranked: false, opponentName: "小王", myScore: 5, opponentScore: 3),
            "滑到终点后松手即可结束，无需对方确认。"
        )
        XCTAssertEqual(FinishCountdown.buildSlideNote(ranked: false), "滑到终点后松手即可结束，无需对方确认。")
        XCTAssertEqual(FinishCountdown.confirmSeconds, 30)
    }

    func testRankedResultPageTellsWhoStartedTheFinishAndHowItEnded() {
        let base = FinishMatchDetailInput(
            matchMode: "ranked",
            player1Id: 1001,
            player2Id: 2002,
            opponentName: "小王"
        )
        var player1Confirmed = base
        player1Confirmed.completionSource = "player_confirmed"
        player1Confirmed.completedByUserId = 2002
        player1Confirmed.isPlayer1 = true
        XCTAssertEqual(FinishCountdown.resolveMethodText(player1Confirmed), "你发起 · 小王确认")

        var player2Confirmed = base
        player2Confirmed.opponentName = "小李"
        player2Confirmed.completionSource = "player_confirmed"
        player2Confirmed.completedByUserId = 2002
        player2Confirmed.isPlayer1 = false
        XCTAssertEqual(FinishCountdown.resolveMethodText(player2Confirmed), "小李发起 · 你确认")

        var player1TimedOut = base
        player1TimedOut.completionSource = "finish_timeout"
        player1TimedOut.completedByUserId = 1001
        player1TimedOut.isPlayer1 = true
        XCTAssertEqual(
            FinishCountdown.resolveMethodText(player1TimedOut),
            "你发起 · 30秒未处理，自动结束"
        )

        var player2TimedOut = base
        player2TimedOut.opponentName = "小李"
        player2TimedOut.completionSource = "finish_timeout"
        player2TimedOut.completedByUserId = 1001
        player2TimedOut.isPlayer1 = false
        XCTAssertEqual(
            FinishCountdown.resolveMethodText(player2TimedOut),
            "小李发起 · 30秒未处理，自动结束"
        )

        var practice = base
        practice.matchMode = "practice"
        practice.completionSource = "player_direct"
        practice.completedByUserId = 1001
        practice.isPlayer1 = true

        var direct = base
        direct.completionSource = "player_direct"
        direct.completedByUserId = 1001
        direct.isPlayer1 = true

        var referee = base
        referee.completionSource = "referee"
        referee.completedByUserId = 3003
        referee.isPlayer1 = true

        var refereeTimeout = base
        refereeTimeout.completionSource = "finish_timeout"
        refereeTimeout.completedByUserId = 3003
        refereeTimeout.isPlayer1 = true

        var unknown = base
        unknown.completionSource = "unknown"
        unknown.isPlayer1 = true

        for match in [practice, direct, referee, refereeTimeout, unknown] {
            XCTAssertEqual(FinishCountdown.resolveMethodText(match), "", "\(match)")
        }
        XCTAssertEqual(FinishCountdown.resolveMethodText(nil), "")
    }

    func testLateResponsesAreDroppedAfterSwitchingAccountOrAuthGeneration() {
        let session = FinishCountdown.captureSession(userId: 1001, authGeneration: 3)
        XCTAssertTrue(FinishCountdown.isSameSession(session, userId: 1001, authGeneration: 3))
        XCTAssertFalse(FinishCountdown.isSameSession(session, userId: 2002, authGeneration: 3))
        XCTAssertFalse(FinishCountdown.isSameSession(session, userId: 1001, authGeneration: 4))
        XCTAssertFalse(FinishCountdown.isSameSession(nil, userId: 1001, authGeneration: 3))
    }
}
