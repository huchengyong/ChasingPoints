import Foundation

/// 进行中页面的视角展示模型；只带结束确认能力时才带末尾六个字段。
struct PlayingViewerUi: Equatable {
    let leftIdentity: String
    let rightIdentity: String
    let subtitleSuffix: String
    let roleLabel: String
    let showActionPanel: Bool
    let showUndoButton: Bool
    let showFinishButton: Bool
    let readonlyHint: String
    let showFinishRequestButton: Bool?
    let showFinishConfirmButton: Bool?
    let showFinishDisputeButton: Bool?
    let showFinishWithdrawButton: Bool?
    let finishState: String?
    let lastAction: MatchLastAction?

    init(
        leftIdentity: String,
        rightIdentity: String,
        subtitleSuffix: String,
        roleLabel: String,
        showActionPanel: Bool,
        showUndoButton: Bool,
        showFinishButton: Bool,
        readonlyHint: String,
        showFinishRequestButton: Bool? = nil,
        showFinishConfirmButton: Bool? = nil,
        showFinishDisputeButton: Bool? = nil,
        showFinishWithdrawButton: Bool? = nil,
        finishState: String? = nil,
        lastAction: MatchLastAction? = nil
    ) {
        self.leftIdentity = leftIdentity
        self.rightIdentity = rightIdentity
        self.subtitleSuffix = subtitleSuffix
        self.roleLabel = roleLabel
        self.showActionPanel = showActionPanel
        self.showUndoButton = showUndoButton
        self.showFinishButton = showFinishButton
        self.readonlyHint = readonlyHint
        self.showFinishRequestButton = showFinishRequestButton
        self.showFinishConfirmButton = showFinishConfirmButton
        self.showFinishDisputeButton = showFinishDisputeButton
        self.showFinishWithdrawButton = showFinishWithdrawButton
        self.finishState = finishState
        self.lastAction = lastAction
    }
}

struct PlayingViewerUiInput: Equatable {
    var viewerRole: String = "player1"
    var canScore: Bool = true
    var canUndo: Bool = true
    var canFinish: Bool = true
    var refereeName: String = ""
    var canRequestFinish: Bool? = nil
    var canConfirmFinish: Bool? = nil
    var canDisputeFinish: Bool? = nil
    var canWithdrawFinish: Bool? = nil
    var finishState: String = "none"
    var lastAction: MatchLastAction? = nil
}

/// 逐条对应小程序 `utils/match-role-view.js`。
enum MatchRoleView {
    static func resolvePlayingViewerUi(_ input: PlayingViewerUiInput = PlayingViewerUiInput()) -> PlayingViewerUi {
        let hasFinishCapabilities = input.canRequestFinish == true
            || input.canConfirmFinish == true
            || input.canDisputeFinish == true
            || input.canWithdrawFinish == true
            || input.finishState != "none"
            || input.lastAction != nil

        func withFinishCapabilities(_ base: PlayingViewerUi) -> PlayingViewerUi {
            guard hasFinishCapabilities else { return base }
            return PlayingViewerUi(
                leftIdentity: base.leftIdentity,
                rightIdentity: base.rightIdentity,
                subtitleSuffix: base.subtitleSuffix,
                roleLabel: base.roleLabel,
                showActionPanel: base.showActionPanel,
                showUndoButton: base.showUndoButton,
                showFinishButton: base.showFinishButton,
                readonlyHint: input.finishState == "pending_confirmation" ? "等待对手处理结束确认" : base.readonlyHint,
                showFinishRequestButton: input.canRequestFinish ?? false,
                showFinishConfirmButton: input.canConfirmFinish ?? false,
                showFinishDisputeButton: input.canDisputeFinish ?? false,
                showFinishWithdrawButton: input.canWithdrawFinish ?? false,
                finishState: input.finishState,
                lastAction: input.lastAction
            )
        }

        if input.viewerRole == "referee" {
            return withFinishCapabilities(PlayingViewerUi(
                leftIdentity: "选手1",
                rightIdentity: "选手2",
                subtitleSuffix: "左选手1右选手2",
                roleLabel: "裁判视角",
                showActionPanel: input.canScore,
                showUndoButton: input.canUndo,
                showFinishButton: input.canFinish,
                readonlyHint: ""
            ))
        }

        return withFinishCapabilities(PlayingViewerUi(
            leftIdentity: "我方",
            rightIdentity: "对手",
            subtitleSuffix: "左我右敌",
            roleLabel: "选手视角",
            showActionPanel: input.canScore,
            showUndoButton: input.canUndo,
            showFinishButton: input.canFinish,
            readonlyHint: input.canScore ? "" : "本场由\(input.refereeName.isEmpty ? "裁判" : input.refereeName)负责记分"
        ))
    }
}
