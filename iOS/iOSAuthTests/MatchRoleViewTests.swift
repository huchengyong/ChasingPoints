import Foundation
import XCTest
@testable import iOS

/// 逐条对应小程序 `uniapp/tests/match-role-view.test.mjs`。
@MainActor
final class MatchRoleViewTests: XCTestCase {
    func testResolvePlayingViewerUiKeepsRefereeAsTheSoleWriter() {
        let ui = MatchRoleView.resolvePlayingViewerUi(PlayingViewerUiInput(
            viewerRole: "referee",
            canScore: true,
            canUndo: true,
            canFinish: true,
            refereeName: "助教丙"
        ))

        XCTAssertEqual(ui, PlayingViewerUi(
            leftIdentity: "选手1",
            rightIdentity: "选手2",
            subtitleSuffix: "左选手1右选手2",
            roleLabel: "裁判视角",
            showActionPanel: true,
            showUndoButton: true,
            showFinishButton: true,
            readonlyHint: ""
        ))
    }

    func testResolvePlayingViewerUiTurnsPlayersIntoReadonlyViewersAfterRefereeBinding() {
        let ui = MatchRoleView.resolvePlayingViewerUi(PlayingViewerUiInput(
            viewerRole: "player1",
            canScore: false,
            canUndo: false,
            canFinish: false,
            refereeName: "助教丙"
        ))

        XCTAssertEqual(ui, PlayingViewerUi(
            leftIdentity: "我方",
            rightIdentity: "对手",
            subtitleSuffix: "左我右敌",
            roleLabel: "选手视角",
            showActionPanel: false,
            showUndoButton: false,
            showFinishButton: false,
            readonlyHint: "本场由助教丙负责记分"
        ))
    }

    func testResolvePlayingViewerUiExposesRankedFinishActionsFromServerCapabilities() {
        let lastAction = MatchLastAction(
            actionType: nil, actor: nil, scoreChange: nil, serverRevision: nil, description: "我方发起结束确认"
        )
        let ui = MatchRoleView.resolvePlayingViewerUi(PlayingViewerUiInput(
            viewerRole: "player2",
            canScore: false,
            canUndo: false,
            canFinish: false,
            canConfirmFinish: true,
            canDisputeFinish: true,
            finishState: "pending_confirmation",
            lastAction: lastAction
        ))

        XCTAssertEqual(ui, PlayingViewerUi(
            leftIdentity: "我方",
            rightIdentity: "对手",
            subtitleSuffix: "左我右敌",
            roleLabel: "选手视角",
            showActionPanel: false,
            showUndoButton: false,
            showFinishButton: false,
            readonlyHint: "等待对手处理结束确认",
            showFinishRequestButton: false,
            showFinishConfirmButton: true,
            showFinishDisputeButton: true,
            showFinishWithdrawButton: false,
            finishState: "pending_confirmation",
            lastAction: lastAction
        ))
    }
}
