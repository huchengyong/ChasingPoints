import Combine
import Foundation
import UIKit

struct LiveMatchRoundWinGroup: Identifiable, Equatable {
    let actor: Int
    let label: String
    let options: [ScoringActionOption]

    var id: Int { actor }
}

struct LiveMatchJiuqiuGroup: Identifiable, Equatable {
    let actor: Int
    let label: String
    let foulScoringLabel: String
    let options: [ScoringActionOption]

    var id: Int { actor }
}

struct LiveMatchMenuOption: Identifiable, Equatable {
    enum Kind: Equatable {
        case settings
        case concedeMatch
        case finishMatch
    }

    let kind: Kind
    let title: String

    var id: Kind { kind }
}

/// 实时计分页的核心状态：三种入口读取、修订号收敛、角色能力、统一写锁、结束流程与赛场消息。
@MainActor
final class LiveMatchModel: ObservableObject {
    enum Phase: Equatable {
        case loading
        case ready
        case failed(String)
    }

    private enum FinishSubmission {
        case request
        case confirm
        case dispute
        case withdraw
    }

    // MARK: - 依赖

    private let session: SessionStore
    private let api: LiveMatchAPI
    private let channel: LiveMatchChannel
    private let now: () -> Date
    private let sleep: (TimeInterval) async throws -> Void
    private var onMatchSettled: () -> Void
    private var onRankInvalidated: () -> Void
    let entry: LiveMatchEntry
    let matchId: Int64

    // MARK: - 生命周期内部状态

    private var pageIdentity: SessionIdentity?
    private var didStart = false
    private var loadToken = 0
    private var messageToken: UUID?
    private var settleTask: Task<Void, Never>?
    private var didScheduleSettle = false
    private var qrToken = 0
    private var isLeaving = false
    private var isVisible = true
    private var writeInFlight = false
    private var finishActionBusy = false
    private var pendingFinishAction: SnookerFrameAction?
    private var hasNavigatedToResult = false
    private var choiceHandler: ((Int) -> Void)?
    private var pendingConcessionScope: String?
    private var pendingAwardWinner: Int?

    // MARK: - 页面状态

    @Published private(set) var phase: Phase = .loading
    @Published private(set) var gameType = 3
    @Published private(set) var gameTypeName = "中式八球"
    @Published private(set) var matchMode = "ranked"
    @Published private(set) var status: Int?
    @Published private(set) var myScore = 0
    @Published private(set) var opponentScore = 0
    @Published private(set) var currentFrameMyScore = 0
    @Published private(set) var currentFrameOpponentScore = 0
    @Published private(set) var currentFrameStarted = true
    @Published private(set) var currentRound = 1
    @Published private(set) var totalRounds = 0
    @Published private(set) var serverRevision: Int64 = 0
    @Published private(set) var durationSeconds: Int64 = 0
    @Published private(set) var myInfo = LiveMatchPlayerInfo()
    @Published private(set) var opponentInfo = LiveMatchPlayerInfo()
    @Published private(set) var isPlayer1 = true
    @Published private(set) var viewerRole = "player1"
    @Published private(set) var refereeBound = false
    @Published private(set) var refereeUserId: Int64 = 0
    @Published private(set) var refereeName = ""
    @Published private(set) var refereeAvatar = ""
    @Published private(set) var canScore = false
    @Published private(set) var canUndo = false
    @Published private(set) var canFinish = false
    @Published private(set) var canRequestFinish = false
    @Published private(set) var canConfirmFinish = false
    @Published private(set) var canDisputeFinish = false
    @Published private(set) var canWithdrawFinish = false
    @Published private(set) var finishState = "none"
    @Published private(set) var finishRequestedBy: Int64 = 0
    @Published private(set) var challengeId: Int64 = 0
    @Published private(set) var lastAction: MatchLastAction?
    @Published private(set) var isSyncing = false
    @Published private(set) var isConnected = false

    // 赛制（中八、美九与斯诺克共用同一套编辑交互）
    @Published private(set) var matchFormat = ""
    @Published private(set) var targetWins = 0
    @Published private(set) var canChangeMatchFormat = false
    @Published private(set) var isFormatEditorShown = false
    @Published private(set) var draftMatchFormat = PoolMatchFormat.free
    @Published private(set) var draftTargetWins = 10
    @Published private(set) var isFormatSaving = false
    @Published private(set) var formatError = ""
    @Published private(set) var snookerRulesVersion = 0
    @Published private(set) var snookerFormat = SnookerMatchFormat.free
    @Published private(set) var snookerTargetWins = 0
    @Published private(set) var canChangeSnookerFormat = false
    @Published private(set) var isSnookerFormatEditorShown = false
    @Published private(set) var draftSnookerFormat = SnookerMatchFormat.free
    @Published private(set) var draftSnookerTargetWins = 3
    @Published private(set) var isSnookerFormatSaving = false
    @Published private(set) var snookerFormatError = ""

    // 斯诺克 V1：红球与清彩进度由服务端下发。
    @Published private(set) var snookerRedBallCount = 0
    @Published private(set) var snookerClearanceStarted = false
    @Published private(set) var snookerClearedColors: [Int] = []
    @Published private(set) var snookerExpectedClearanceScore = 0
    @Published private(set) var snookerClearanceCompleted = false
    @Published private(set) var snookerScoringActor = 1
    @Published private(set) var snookerFoulingActor = 1

    // 斯诺克 V2：服务端权威局面与犯规编辑器草稿。
    @Published private(set) var snookerV2State = SnookerV2State()
    @Published private(set) var isSnookerFoulEditorShown = false
    @Published private(set) var snookerFoulPenalty = 4
    @Published private(set) var snookerFoulBallOnValue = 2
    @Published private(set) var snookerFoulRedsRemoved = 0
    @Published private(set) var snookerFoulResolution = "incoming_plays"
    @Published private(set) var snookerFoulAndMiss = false
    @Published private(set) var snookerMissSequenceEligible = false
    @Published private(set) var snookerFreeBallAwarded = false
    @Published private(set) var snookerFoulCueBallInHand = false
    @Published private(set) var choice: LiveMatchChoice?

    // 结束确认
    @Published private(set) var nowValue: Date
    private var finishCountdownAnchor: FinishCountdownAnchor?
    @Published private(set) var confirm: LiveMatchConfirm?
    @Published private(set) var toastMessage: String?
    @Published private(set) var destination: LiveMatchDestination?
    @Published private(set) var shouldDismiss = false

    // 裁判二维码
    @Published private(set) var isRefereeQRCodePresented = false
    @Published private(set) var isRefereeQRCodeLoading = false
    @Published private(set) var refereeQRCodeContent = ""
    @Published private(set) var refereeQRCodeImage: UIImage?
    @Published private(set) var refereeQRCodeError = ""

    init(
        session: SessionStore,
        matchId: Int64,
        entry: LiveMatchEntry = .automatic,
        api: LiveMatchAPI? = nil,
        channel: LiveMatchChannel? = nil,
        now: @escaping () -> Date = Date.init,
        sleep: @escaping (TimeInterval) async throws -> Void = { try await Task.sleep(for: .seconds($0)) },
        onMatchSettled: @escaping () -> Void = {},
        onRankInvalidated: @escaping () -> Void = {}
    ) {
        self.session = session
        self.matchId = matchId
        self.entry = entry
        self.api = api ?? LiveMatchAPI(session: session)
        self.channel = channel ?? MatchRealtime(session: session)
        self.now = now
        self.sleep = sleep
        self.onMatchSettled = onMatchSettled
        self.onRankInvalidated = onRankInvalidated
        self.pageIdentity = session.identity
        self.nowValue = now()
    }

    /// 结算回调依赖环境里的 `ActivityStore`，只能在视图拿到环境后补上。
    func setSettlementHandlers(
        onMatchSettled: @escaping () -> Void,
        onRankInvalidated: @escaping () -> Void
    ) {
        self.onMatchSettled = onMatchSettled
        self.onRankInvalidated = onRankInvalidated
    }

    // MARK: - 展示派生

    var isReferee: Bool { viewerRole == "referee" }
    var isSpectator: Bool { viewerRole == "spectator" }
    var isSnookerV2: Bool { gameType == 1 && snookerRulesVersion == 2 }
    var isRoundWinMode: Bool { gameType == 3 || gameType == 4 }
    var isPoolFormatVisible: Bool {
        PoolMatchFormat.isPoolMatchFormatGameType(gameType)
            && !matchFormat.isEmpty
            && matchFormat != PoolMatchFormat.legacy
    }

    var viewerUi: PlayingViewerUi {
        if isSpectator {
            return PlayingViewerUi(
                leftIdentity: "选手1",
                rightIdentity: "选手2",
                subtitleSuffix: "左选手1右选手2",
                roleLabel: "观赛视角",
                showActionPanel: false,
                showUndoButton: false,
                showFinishButton: false,
                readonlyHint: ""
            )
        }
        return MatchRoleView.resolvePlayingViewerUi(PlayingViewerUiInput(
            viewerRole: viewerRole,
            canScore: canScore,
            canUndo: canUndo,
            canFinish: canFinish,
            refereeName: refereeName,
            canRequestFinish: canRequestFinish,
            canConfirmFinish: canConfirmFinish,
            canDisputeFinish: canDisputeFinish,
            canWithdrawFinish: canWithdrawFinish,
            finishState: finishState,
            lastAction: lastAction
        ))
    }

    var gameSubtitle: String {
        LiveMatchText.gameSubtitle(gameType: gameType, isSnookerV2: isSnookerV2, isReferee: isReferee)
    }

    var scoreboardSubtitle: String { "\(gameSubtitle) · \(viewerUi.subtitleSuffix)" }

    var matchModeText: String { matchMode == "ranked" ? "排位赛" : "练习赛" }

    var durationText: String { LiveMatchText.duration(durationSeconds) }

    var currentRoundText: String {
        LiveMatchText.roundText(gameType: gameType, currentFrameStarted: currentFrameStarted, currentRound: currentRound)
    }

    var syncStatusText: String { isSyncing ? "比分同步中" : "实时同步中" }

    var actionPanelDescription: String {
        LiveMatchText.actionPanelDescription(
            canScore: canScore,
            readonlyHint: viewerUi.readonlyHint,
            gameType: gameType,
            isSnookerV2: isSnookerV2,
            isReferee: isReferee
        )
    }

    var readonlyTitle: String { isSpectator ? "当前为只读观赛" : "当前为只读" }
    var readonlyDescription: String { viewerUi.readonlyHint.isEmpty ? "请等待裁判完成记分操作" : viewerUi.readonlyHint }

    var matchFormatLabel: String { PoolMatchFormat.label(matchFormat, targetWins: targetWins) }
    var matchFormatHint: String { PoolMatchFormat.hint(matchFormat, targetWins: targetWins) }
    var draftMatchFormatHint: String { PoolMatchFormat.hint(draftMatchFormat, targetWins: draftTargetWins) }

    private var isFreeNormalFinish: Bool {
        (isSnookerV2 && snookerFormat == SnookerMatchFormat.free)
            || (isPoolFormatVisible && matchFormat == PoolMatchFormat.free)
    }

    var finishButtonLabel: String { LiveMatchText.finishButtonLabel(isFreeNormalFinish: isFreeNormalFinish) }
    var finishRequestButtonLabel: String { LiveMatchText.finishRequestButtonLabel(isFreeNormalFinish: isFreeNormalFinish) }

    var showInviteRefereeAction: Bool {
        !isReferee && !isSpectator && !refereeBound
    }

    var roundWinActionOptions: [ScoringActionOption] {
        MatchScoringTarget.buildRoundWinActionOptions(gameType: gameType, viewerRole: viewerRole, uiActor: 2)
    }

    var refereeRoundWinGroups: [LiveMatchRoundWinGroup] {
        [1, 2].map { actor in
            LiveMatchRoundWinGroup(
                actor: actor,
                label: MatchScoringTarget.resolveScoringParticipantLabel(viewerRole: "referee", uiActor: actor),
                options: MatchScoringTarget.buildRoundWinActionOptions(
                    gameType: gameType,
                    viewerRole: "referee",
                    uiActor: actor
                )
            )
        }
    }

    var jiuqiuWinOptions: [ScoringActionOption] {
        MatchScoringTarget.buildJiuqiuWinOptions(viewerRole: viewerRole, uiActor: 2)
    }

    var jiuqiuNormalOption: ScoringActionOption { jiuqiuWinOptions[0] }
    var jiuqiuGoldOptions: ArraySlice<ScoringActionOption> { jiuqiuWinOptions.dropFirst() }

    var refereeJiuqiuGroups: [LiveMatchJiuqiuGroup] {
        [1, 2].map { actor in
            LiveMatchJiuqiuGroup(
                actor: actor,
                label: MatchScoringTarget.resolveScoringParticipantLabel(viewerRole: "referee", uiActor: actor),
                foulScoringLabel: MatchScoringTarget.resolveScoringParticipantLabel(
                    viewerRole: "referee",
                    uiActor: actor == 1 ? 2 : 1
                ),
                options: MatchScoringTarget.buildJiuqiuWinOptions(viewerRole: "referee", uiActor: actor)
            )
        }
    }

    // MARK: - 斯诺克派生

    var viewerFixedActor: Int { isReferee ? 0 : (isPlayer1 ? 1 : 2) }

    var snookerFormatLabel: String {
        SnookerMatchFormat.label(snookerFormat, targetWins: snookerTargetWins)
    }

    var snookerFormatHint: String {
        SnookerMatchFormat.hint(snookerFormat, targetWins: snookerTargetWins)
    }

    var draftSnookerFormatHint: String {
        SnookerMatchFormat.hint(draftSnookerFormat, targetWins: draftSnookerTargetWins)
    }

    var canStartNextSnookerFrameValue: Bool {
        SnookerMatchFormat.canStartNextSnookerFrame(
            format: snookerFormat,
            targetWins: snookerTargetWins,
            myScore: myScore,
            opponentScore: opponentScore
        )
    }

    var snookerRedBallText: String {
        "红球 \(String(snookerRedBallCount))/15"
    }

    var snookerRedBallRemaining: Int { max(15 - snookerRedBallCount, 0) }

    var isSnookerRedDisabled: Bool {
        gameType == 1 && (snookerRedBallCount >= 15 || !currentFrameStarted)
    }

    private var snookerClearedColorNames: [String] {
        snookerClearedColors.compactMap(Self.snookerColorName)
    }

    var snookerNextClearanceColorName: String {
        Self.snookerColorName(snookerExpectedClearanceScore)
    }

    var snookerRedHintText: String {
        if !currentFrameStarted {
            return "当前局已结束，可开始下一局，或直接结束整场比赛"
        }
        if snookerClearanceCompleted {
            return "清彩阶段已完成，请确认本局胜方"
        }
        if snookerClearanceStarted {
            let clearedText = snookerClearedColorNames.isEmpty ? "" : "，已完成：\(snookerClearedColorNames.joined(separator: "、"))"
            let nextText = snookerNextClearanceColorName.isEmpty ? "" : "，下一颗：\(snookerNextClearanceColorName)"
            return "已进入清彩阶段，彩球需按顺序计分\(clearedText)\(nextText)"
        }
        if isSnookerRedDisabled { return "本局红球已打完，请先记录最后一颗红球后的彩球" }
        if snookerRedBallRemaining <= 2 { return "本局红球还剩 \(String(snookerRedBallRemaining)) 颗" }
        return "斯诺克最多打完15套红球"
    }

    func isSnookerScoreDisabled(_ score: Int) -> Bool {
        guard gameType == 1 else { return false }
        if !currentFrameStarted { return true }
        if score == 1 { return isSnookerRedDisabled }
        if !snookerClearanceStarted { return false }
        if snookerExpectedClearanceScore > 0 {
            return score != snookerExpectedClearanceScore
        }
        return false
    }

    var snookerScoreTitle: String {
        isReferee
            ? "给\(MatchScoringTarget.resolveScoringParticipantLabel(viewerRole: "referee", uiActor: snookerScoringActor))计分"
            : "给对手计分"
    }

    var snookerFoulTitle: String {
        if isReferee {
            let foulingLabel = MatchScoringTarget.resolveScoringParticipantLabel(
                viewerRole: "referee",
                uiActor: snookerFoulingActor
            )
            let scoringLabel = MatchScoringTarget.resolveScoringParticipantLabel(
                viewerRole: "referee",
                uiActor: snookerFoulingActor == 1 ? 2 : 1
            )
            return "\(foulingLabel)犯规，给\(scoringLabel)加分"
        }
        return "对手犯规，给我方加分"
    }

    var snookerV2PhaseLabel: String { SnookerV2.phaseLabel(snookerV2State) }
    var snookerV2BallOnLabel: String { SnookerV2.ballOnLabel(snookerV2State) }
    var snookerV2StrikerLabel: String { snookerActorLabel(snookerV2State.striker) }
    var snookerV2PotOptions: [SnookerBall] { SnookerV2.potOptions(snookerV2State) }
    var snookerColorOptions: [SnookerBall] { Array(SnookerV2.balls.dropFirst()) }
    var snookerFoulResolutions: [(value: String, label: String)] {
        [
            ("incoming_plays", "非犯规方从现状击球"),
            ("offender_plays_from_left", "犯规方从现状继续"),
            ("offender_replays_original", "复原球位，由犯规方重打"),
        ]
    }

    var canRespondToSnookerConcession: Bool {
        let pendingActor = snookerV2State.pendingConcessionActor
        return pendingActor > 0 && (isReferee || viewerFixedActor == (pendingActor == 1 ? 2 : 1))
    }

    func canStartRespottedBlack(_ actor: Int) -> Bool {
        canScore && snookerV2State.respottedBlackPending && snookerV2State.pendingConcessionActor == 0
            && (actor == 1 || actor == 2)
    }

    var isSnookerV2ScorePanelVisible: Bool {
        currentFrameStarted && snookerV2State.pendingConcessionActor == 0
    }

    var isChallengeMatchFinishSlider: Bool {
        challengeId > 0
            && finishState != "pending_confirmation"
            && (viewerUi.showFinishButton || (viewerUi.showFinishRequestButton ?? false))
    }

    var finishSlideNote: String {
        FinishCountdown.buildSlideNote(
            ranked: canRequestFinish,
            opponentName: opponentInfo.nickname,
            myScore: myScore,
            opponentScore: opponentScore
        )
    }

    var finishSecondsLeft: Int {
        FinishCountdown.secondsLeft(finishCountdownAnchor, now: nowValue)
    }

    var finishConfirmView: FinishConfirmView {
        FinishCountdown.resolveConfirmView(FinishConfirmInput(
            finishState: finishState,
            secondsLeft: finishSecondsLeft,
            isRequester: finishRequestedBy > 0 && finishRequestedBy == (pageIdentity?.userId ?? 0),
            opponentName: opponentInfo.nickname,
            myScore: myScore,
            opponentScore: opponentScore,
            canWithdrawFinish: canWithdrawFinish,
            canConfirmFinish: canConfirmFinish,
            canDisputeFinish: canDisputeFinish
        ))
    }

    var showFinishConfirmDialog: Bool {
        let view = finishConfirmView
        return view.visible && view.role == "responder" && !view.finishing
    }

    var showFinishConfirmPanel: Bool {
        finishConfirmView.visible && !showFinishConfirmDialog
    }

    var menuOptions: [LiveMatchMenuOption] {
        if isChallengeMatchFinishSlider {
            return [LiveMatchMenuOption(kind: .settings, title: "对局设置")]
        }
        return [
            LiveMatchMenuOption(kind: .settings, title: "对局设置"),
            LiveMatchMenuOption(
                kind: isSnookerV2 ? .concedeMatch : .finishMatch,
                title: isSnookerV2 ? "认输整场比赛" : "放弃对局"
            ),
        ]
    }

    // MARK: - 入口

    /// 快照测试会先手动调用；页面 `.task` 重复触发时直接返回，避免二次加载。
    func start() async {
        guard !didStart else { return }
        didStart = true
        messageToken = channel.observeLive { [weak self] message in
            self?.handle(message)
        }
        await load(showLoading: true)
    }

    func retry() async {
        await load(showLoading: true)
    }

    private func load(showLoading: Bool) async {
        loadToken += 1
        let token = loadToken
        let identity = session.identity
        pageIdentity = identity
        if showLoading {
            phase = .loading
        }

        do {
            let (detail, isPublic) = try await fetchDetail()
            guard token == loadToken, session.identity == identity else { return }
            guard let match = detail.match else {
                throw APIError(message: "对局不存在或无权查看", kind: .business)
            }
            applyDetail(match, isPublic: isPublic)

            if shouldCheckCurrentMatch {
                let current = try? await api.current()
                guard token == loadToken, session.identity == identity else { return }
                let currentMatchId: Int64? = current?.success == true
                    ? (current?.match?.id ?? 0)
                    : nil
                if MatchNavigation.shouldLeavePlayingPage(
                    pageMatchId: matchId,
                    detailStatus: match.status,
                    currentMatchId: currentMatchId
                ) {
                    leaveForHistory()
                    return
                }
                if let currentMatch = current?.match, currentMatch.id == matchId {
                    applyCurrentMatch(currentMatch)
                }
            }

            phase = .ready
            await connectChannel()
        } catch {
            guard token == loadToken, session.identity == identity else { return }
            guard showLoading else { return }
            let message = (error as? APIError)?.message ?? "加载对局失败，请重试"
            phase = .failed(message)
        }
    }

    private var shouldCheckCurrentMatch: Bool {
        switch entry {
        case .spectator: false
        case .participant, .automatic: !isSpectator && session.identity != nil
        }
    }

    /// 本人 / 裁判先读详情，无权限时回退公开详情；访客直接读公开详情。
    /// 公开详情不带身份字段，返回 `true` 让调用方按观赛视角处理。
    private func fetchDetail() async throws -> (response: LiveMatchDetailResponse, isPublic: Bool) {
        switch entry {
        case .spectator:
            return (try await api.publicDetail(matchId: matchId), true)
        case .participant:
            return (try await api.detail(matchId: matchId), false)
        case .automatic:
            guard session.identity != nil else {
                return (try await api.publicDetail(matchId: matchId), true)
            }
            do {
                let detail = try await api.detail(matchId: matchId)
                if detail.success, detail.match != nil { return (detail, false) }
            } catch let error as APIError where error.kind == .business || error.kind == .unauthorized {
                // 登录用户看别人的公开对局：详情接口无权限时按观赛读取。
            }
            return (try await api.publicDetail(matchId: matchId), true)
        }
    }

    private func connectChannel() async {
        guard !isLeaving, phase == .ready else { return }
        do {
            try await channel.openLive(matchId: matchId, allowAnonymous: isSpectator)
            isConnected = channel.isConnected
        } catch {
            isConnected = false
        }
    }

    // MARK: - 快照应用

    private func applyDetail(_ snapshot: LiveMatchSnapshot, isPublic: Bool = false) {
        if let role = snapshot.viewerRole, !role.isEmpty {
            viewerRole = role
        } else if isPublic {
            viewerRole = "spectator"
        }
        isPlayer1 = viewerRole == "spectator"
            ? true
            : (snapshot.isPlayer1 ?? (viewerRole == "player2" ? false : true))
        if let value = snapshot.gameType, value > 0 {
            gameType = value
        }
        gameTypeName = snapshot.gameTypeName?.isEmpty == false
            ? (snapshot.gameTypeName ?? LiveMatchText.gameTitle(gameType))
            : LiveMatchText.gameTitle(gameType)
        if let mode = snapshot.matchMode, !mode.isEmpty {
            matchMode = mode
        }
        if let value = snapshot.challengeId {
            challengeId = value
        }
        if let value = snapshot.status {
            status = value
        }
        updateRevision(snapshot.serverRevision)
        applyViewerCapabilities(snapshot)
        applyScores(snapshot)
        applyFrameScores(snapshot)
        if let value = snapshot.currentRound {
            currentRound = value
        }
        if let value = snapshot.totalRounds {
            totalRounds = value
        }
        if let value = snapshot.durationSeconds {
            durationSeconds = value
        }
        applyPlayerInfos(snapshot)
        applySnookerRoundState(snapshot)
        applyPoolMatchFormatState(snapshot)
    }

    /// 推送 / 写响应快照：修订号不前进的一律丢弃，避免旧响应覆盖新状态。
    private func applySnapshot(_ snapshot: LiveMatchSnapshot) {
        guard shouldApplyIncomingRevision(snapshot.serverRevision) else { return }
        updateRevision(snapshot.serverRevision)
        if let value = snapshot.challengeId {
            challengeId = value
        }
        applyViewerCapabilities(snapshot)
        if let mode = snapshot.matchMode, !mode.isEmpty {
            matchMode = mode
        }
        if let value = snapshot.status {
            status = value
        }
        applyScores(snapshot)
        applyFrameScores(snapshot)
        if let value = snapshot.currentRound, value > 0 {
            currentRound = value
        }
        if let value = snapshot.totalRounds {
            totalRounds = value
        }
        applySnookerRoundState(snapshot)
        applyPoolMatchFormatState(snapshot)
    }

    private func applyViewerCapabilities(_ snapshot: LiveMatchSnapshot) {
        if let role = snapshot.viewerRole, !role.isEmpty {
            viewerRole = role
        }
        if let value = snapshot.refereeBound {
            refereeBound = value
            if value { closeRefereeQRCode() }
        }
        if let value = snapshot.refereeUserId { refereeUserId = value }
        if let value = snapshot.refereeName { refereeName = value }
        if let value = snapshot.refereeAvatar { refereeAvatar = value }
        if let value = snapshot.canScore { canScore = value }
        if let value = snapshot.canUndo { canUndo = value }
        if let value = snapshot.canFinish { canFinish = value }
        if let value = snapshot.canRequestFinish { canRequestFinish = value }
        if let value = snapshot.canConfirmFinish { canConfirmFinish = value }
        if let value = snapshot.canDisputeFinish { canDisputeFinish = value }
        if let value = snapshot.canWithdrawFinish { canWithdrawFinish = value }
        if let value = snapshot.finishState {
            finishState = value.isEmpty ? "none" : value
            if finishState != "pending_confirmation" {
                didScheduleSettle = false
                settleTask?.cancel()
                settleTask = nil
            }
        }
        if let value = snapshot.finishRequestedBy {
            finishRequestedBy = value
        }
        if let value = snapshot.finishRequestRemainingSeconds {
            finishCountdownAnchor = FinishCountdown.makeAnchor(
                remainingSeconds: value,
                receivedAt: now()
            )
            nowValue = now()
            didScheduleSettle = false
            settleTask?.cancel()
            settleTask = nil
        }
        if let value = snapshot.lastAction {
            lastAction = value
        }
        if let value = snapshot.canChangeMatchFormat {
            canChangeMatchFormat = value
            if !value { isFormatEditorShown = false }
        }
        if let value = snapshot.canChangeSnookerFormat {
            canChangeSnookerFormat = value
            if !value { isSnookerFormatEditorShown = false }
        }
    }

    private func applyScores(_ snapshot: LiveMatchSnapshot) {
        if let my = snapshot.myScore, let opponent = snapshot.opponentScore {
            myScore = my
            opponentScore = opponent
            return
        }
        if snapshot.player1Score != nil || snapshot.player2Score != nil {
            applyServerScores(snapshot.player1Score ?? 0, snapshot.player2Score ?? 0)
        }
    }

    private func applyFrameScores(_ snapshot: LiveMatchSnapshot) {
        if let my = snapshot.currentFrameMyScore {
            currentFrameMyScore = my
        } else if snapshot.currentFramePlayer1Score != nil || snapshot.currentFramePlayer2Score != nil {
            applyServerCurrentFrameScores(
                snapshot.currentFramePlayer1Score ?? 0,
                snapshot.currentFramePlayer2Score ?? 0
            )
        }
        if let opponent = snapshot.currentFrameOpponentScore {
            currentFrameOpponentScore = opponent
        }
        if let started = snapshot.currentFrameStarted {
            currentFrameStarted = started
        }
    }

    private func applyPlayerInfos(_ snapshot: LiveMatchSnapshot) {
        if viewerRole == "spectator" {
            myInfo = LiveMatchPlayerInfo(
                userId: snapshot.player1Id ?? 0,
                nickname: Self.firstNonEmpty(snapshot.player1Name, "选手1"),
                avatar: snapshot.player1Avatar ?? ""
            )
            opponentInfo = LiveMatchPlayerInfo(
                userId: snapshot.player2Id ?? 0,
                nickname: Self.firstNonEmpty(snapshot.player2Name, "选手2"),
                avatar: snapshot.player2Avatar ?? ""
            )
            return
        }

        if snapshot.isPlayer1 == true {
            myInfo.userId = snapshot.player1Id ?? myInfo.userId
        } else if snapshot.isPlayer1 == false {
            myInfo.userId = snapshot.player2Id ?? myInfo.userId
        }
        myInfo.nickname = Self.firstNonEmpty(snapshot.myName, Self.firstNonEmpty(myInfo.nickname, "我"))
        myInfo.avatar = snapshot.myAvatar ?? myInfo.avatar
        myInfo.vipLevel = snapshot.myVipLevel ?? myInfo.vipLevel
        myInfo.rankLevel = snapshot.myRankLevel ?? myInfo.rankLevel
        myInfo.winRate = Int((snapshot.myWinRate ?? Double(myInfo.winRate)).rounded())
        myInfo.maxScore = snapshot.myMaxScore ?? myInfo.maxScore

        opponentInfo.userId = snapshot.opponentId ?? opponentInfo.userId
        opponentInfo.nickname = Self.firstNonEmpty(snapshot.opponentName, Self.firstNonEmpty(opponentInfo.nickname, "对手"))
        opponentInfo.avatar = snapshot.opponentAvatar ?? opponentInfo.avatar
        opponentInfo.vipLevel = snapshot.opponentVipLevel ?? opponentInfo.vipLevel
        opponentInfo.rankLevel = snapshot.opponentRankLevel ?? opponentInfo.rankLevel
        opponentInfo.winRate = Int((snapshot.opponentWinRate ?? Double(opponentInfo.winRate)).rounded())
        opponentInfo.maxScore = snapshot.opponentMaxScore ?? opponentInfo.maxScore
    }

    private func applySnookerRoundState(_ snapshot: LiveMatchSnapshot) {
        // V1 的红球/清彩字段每次快照都要覆盖；它们与规则版本无关。
        snookerRedBallCount = snapshot.redBallCount ?? 0
        snookerClearanceStarted = snapshot.snookerClearanceStarted == true
        snookerClearedColors = snapshot.snookerClearedColors ?? []
        snookerExpectedClearanceScore = snapshot.snookerExpectedClearanceScore ?? 0
        snookerClearanceCompleted = snapshot.snookerClearanceCompleted == true

        guard let version = snapshot.snookerRulesVersion else { return }
        snookerRulesVersion = version
        let legacy = snapshot.bestOfFrames ?? 0
        let state = SnookerMatchFormat.normalize(SnookerMatchFormatInput(
            snookerFormat: snapshot.snookerFormat ?? (legacy > 0 ? SnookerMatchFormat.raceTo : SnookerMatchFormat.free),
            snookerTargetWins: snapshot.snookerTargetWins ?? (legacy > 0 ? legacy / 2 + 1 : 0)
        ))
        snookerFormat = state.format
        snookerTargetWins = state.targetWins
        if let value = snapshot.canChangeSnookerFormat {
            canChangeSnookerFormat = value
            if !value { isSnookerFormatEditorShown = false }
        }

        snookerV2State = SnookerV2.normalizeState(SnookerV2SnapshotInput(
            snookerRulesVersion: snapshot.snookerRulesVersion,
            bestOfFrames: snapshot.bestOfFrames,
            startingActor: snapshot.startingActor,
            snookerPhase: snapshot.snookerPhase,
            snookerBallOn: snapshot.snookerBallOn,
            snookerStriker: snapshot.snookerStriker,
            snookerVisitNo: snapshot.snookerVisitNo,
            snookerCurrentBreak: snapshot.snookerCurrentBreak,
            snookerRedsRemaining: snapshot.snookerRedsRemaining,
            snookerFreeBallAvailable: snapshot.snookerFreeBallAvailable,
            snookerCueBallInHand: snapshot.snookerCueBallInHand,
            snookerMissWarningActive: snapshot.snookerMissWarningActive,
            snookerRespottedBlackPending: snapshot.snookerRespottedBlackPending,
            snookerPendingConcessionActor: snapshot.snookerPendingConcessionActor,
            snookerPendingConcessionScope: snapshot.snookerPendingConcessionScope,
            snookerFrameEndReason: snapshot.snookerFrameEndReason,
            currentFrameStarted: snapshot.currentFrameStarted,
            status: snapshot.status
        ))
        if snookerRulesVersion == 2 {
            let hideEditor = !snookerV2State.currentFrameStarted
                || snookerV2State.pendingConcessionActor > 0
                || ["ended", "respotted_black_pending"].contains(snookerV2State.phase)
            if hideEditor {
                isSnookerFoulEditorShown = false
            } else if !isSnookerFoulEditorShown {
                // 裁判正在编辑时保留草稿，避免推送把刚选的罚分冲回默认值。
                let target = SnookerV2.ballOnValue(snookerV2State)
                snookerFoulBallOnValue = target >= 2 ? target : 2
                snookerFoulPenalty = SnookerV2.minimumPenalty(snookerV2State, ballOnValue: snookerFoulBallOnValue)
                snookerFoulRedsRemoved = 0
            }
        }
    }

    private func applyPoolMatchFormatState(_ snapshot: LiveMatchSnapshot) {
        guard PoolMatchFormat.isPoolMatchFormatGameType(snapshot.gameType ?? gameType) else {
            matchFormat = ""
            targetWins = 0
            canChangeMatchFormat = false
            isFormatEditorShown = false
            return
        }
        if snapshot.matchFormat != nil || snapshot.targetWins != nil {
            let state = PoolMatchFormat.normalize(PoolMatchFormatInput(
                matchFormat: snapshot.matchFormat,
                targetWins: snapshot.targetWins
            ))
            matchFormat = state.format
            targetWins = state.targetWins
        }
        if let value = snapshot.canChangeMatchFormat {
            canChangeMatchFormat = value
            if !value { isFormatEditorShown = false }
        }
    }

    private func applyCurrentMatch(_ match: CurrentMatchInfo) {
        updateRevision(match.serverRevision)
        currentRound = match.currentRound
        if PoolMatchFormat.isPoolMatchFormatGameType(match.gameType) {
            if let format = match.matchFormat, let target = match.targetWins {
                let state = PoolMatchFormat.normalize(PoolMatchFormatInput(
                    matchFormat: format,
                    targetWins: target
                ))
                matchFormat = state.format
                targetWins = state.targetWins
            }
            if let value = match.canChangeMatchFormat {
                canChangeMatchFormat = value
            }
        }
    }

    // MARK: - 修订号

    private func updateRevision(_ value: Int64?) {
        guard let value, value >= 0 else { return }
        serverRevision = max(serverRevision, value)
    }

    /// 与小程序一致：缺失 / 非正数放行，只有低于本地修订的快照才丢弃。
    private func shouldApplyIncomingRevision(_ value: Int64?) -> Bool {
        guard let value else { return true }
        if value <= 0 { return true }
        return value >= serverRevision
    }

    private func applyServerScores(_ player1Score: Int, _ player2Score: Int) {
        if isPlayer1 {
            myScore = player1Score
            opponentScore = player2Score
        } else {
            myScore = player2Score
            opponentScore = player1Score
        }
    }

    private func applyServerCurrentFrameScores(_ player1Score: Int, _ player2Score: Int) {
        if isPlayer1 {
            currentFrameMyScore = player1Score
            currentFrameOpponentScore = player2Score
        } else {
            currentFrameMyScore = player2Score
            currentFrameOpponentScore = player1Score
        }
    }

    // MARK: - 实时消息

    private func handle(_ message: RealtimeMessage) {
        switch message.type {
        case WSMessageType.sync:
            guard let snapshot = decodeSnapshot(message.payload) else { return }
            handleSync(snapshot)
        case WSMessageType.scoreUpdate:
            guard let snapshot = decodeSnapshot(message.payload) else { return }
            handleScoreUpdate(snapshot)
        case WSMessageType.roundEnd:
            guard let snapshot = decodeSnapshot(message.payload) else { return }
            handleRoundEnd(snapshot)
        case WSMessageType.roundStart:
            guard let snapshot = decodeSnapshot(message.payload) else { return }
            handleRoundStart(snapshot)
        case WSMessageType.matchEnd:
            guard let snapshot = decodeSnapshot(message.payload) else { return }
            handleMatchEnd(snapshot)
        case WSMessageType.matchRoleChanged:
            Task { await handleRoleChanged() }
        case WSMessageType.matchFinishRequest,
            WSMessageType.matchFinishConfirm,
            WSMessageType.matchFinishDispute,
            WSMessageType.matchFinishWithdraw,
            WSMessageType.matchFinishExpired,
            WSMessageType.matchFinishTimeout:
            guard let snapshot = decodeFinishSnapshot(message.payload) else { return }
            handleFinishStateUpdate(snapshot)
        default:
            break
        }
    }

    private func decodeSnapshot(_ payload: Data?) -> LiveMatchSnapshot? {
        guard let payload else { return nil }
        return try? Self.snapshotDecoder.decode(LiveMatchSnapshot.self, from: payload)
    }

    /// 结束状态消息把完整快照放在 `snapshot` 里，顶层只有修订号与结束状态；
    /// 只读顶层会丢掉确认/异议能力与倒计时。
    private func decodeFinishSnapshot(_ payload: Data?) -> LiveMatchSnapshot? {
        guard let payload else { return nil }
        struct Envelope: Decodable {
            let snapshot: LiveMatchSnapshot?
        }
        if let envelope = try? Self.snapshotDecoder.decode(Envelope.self, from: payload),
           let snapshot = envelope.snapshot {
            return snapshot
        }
        return try? Self.snapshotDecoder.decode(LiveMatchSnapshot.self, from: payload)
    }

    private static let snapshotDecoder: JSONDecoder = {
        let decoder = JSONDecoder()
        decoder.keyDecodingStrategy = .convertFromSnakeCase
        return decoder
    }()

    private func handleScoreUpdate(_ snapshot: LiveMatchSnapshot) {
        guard shouldApplyIncomingRevision(snapshot.serverRevision) else { return }
        applySnapshot(snapshot)
        // 推送顶层的 my_score / opponent_score 恒为选手1视角，按 player1/player2 还原当前视角。
        applyServerScores(
            snapshot.player1Score ?? snapshot.myScore ?? 0,
            snapshot.player2Score ?? snapshot.opponentScore ?? 0
        )
    }

    private func handleRoundEnd(_ snapshot: LiveMatchSnapshot) {
        guard shouldApplyIncomingRevision(snapshot.serverRevision) else { return }
        applySnapshot(snapshot)
        applyServerScores(
            snapshot.player1Score ?? snapshot.myScore ?? 0,
            snapshot.player2Score ?? snapshot.opponentScore ?? 0
        )

        if isSnookerV2 {
            toast("本局已判给\(snookerActorLabel(snapshot.winner ?? 0))")
            return
        }
        let isMyWin = isPlayer1 ? snapshot.actor == "me" : snapshot.actor == "opponent"
        if isReferee {
            toast("本局已判给\(snapshot.winner == 1 ? "选手1" : "选手2")")
        } else {
            toast(isMyWin ? "你赢得本局" : "对手赢得本局")
        }
    }

    private func handleRoundStart(_ snapshot: LiveMatchSnapshot) {
        guard shouldApplyIncomingRevision(snapshot.serverRevision) else { return }
        updateRevision(snapshot.serverRevision)
        applyServerCurrentFrameScores(
            snapshot.currentFramePlayer1Score ?? 0,
            snapshot.currentFramePlayer2Score ?? 0
        )
        currentFrameStarted = snapshot.currentFrameStarted != false
        currentRound = snapshot.currentRound ?? currentRound
        applySnapshot(snapshot)
        toast("第\(currentRound)局开始")
    }

    private func handleSync(_ snapshot: LiveMatchSnapshot) {
        guard shouldApplyIncomingRevision(snapshot.serverRevision) else {
            isSyncing = false
            return
        }
        applySnapshot(snapshot)
        if snapshot.player1Score != nil || snapshot.player2Score != nil {
            applyServerScores(snapshot.player1Score ?? 0, snapshot.player2Score ?? 0)
        }
        applyServerCurrentFrameScores(
            snapshot.currentFramePlayer1Score ?? 0,
            snapshot.currentFramePlayer2Score ?? 0
        )
        currentFrameStarted = snapshot.currentFrameStarted != false
        if let round = snapshot.currentRound { currentRound = round }
        isSyncing = false
        if snapshot.status == 2 {
            invalidateRankAfterSettlement(snapshot)
            navigateToResultOnce()
        }
    }

    private func handleFinishStateUpdate(_ snapshot: LiveMatchSnapshot) {
        guard shouldApplyIncomingRevision(snapshot.serverRevision) else { return }
        applySnapshot(snapshot)
        invalidateRankAfterSettlement(snapshot)
        toast(finishState == "pending_confirmation" ? "对局等待结束确认" : "对局结束状态已更新")
    }

    private func handleMatchEnd(_ snapshot: LiveMatchSnapshot) {
        guard shouldApplyIncomingRevision(snapshot.serverRevision) else { return }
        if let mode = snapshot.matchMode, !mode.isEmpty {
            matchMode = mode
        }
        if let player1 = snapshot.player1Score, let player2 = snapshot.player2Score {
            applyServerScores(player1, player2)
        }
        invalidateRankAfterSettlement(snapshot)
        onMatchSettled()
        updateRevision(snapshot.serverRevision)
        navigateToResultOnce()
    }

    private func handleRoleChanged() async {
        let token = loadToken
        let identity = session.identity
        do {
            let detail = try await api.detail(matchId: matchId)
            guard token == loadToken, session.identity == identity, let match = detail.match else { return }
            applyDetail(match)
            if MatchNavigation.shouldLeavePlayingPage(
                pageMatchId: matchId,
                detailStatus: match.status,
                currentMatchId: nil
            ) {
                leaveForHistory()
                return
            }
            if channel.isConnected {
                await channel.requestLiveSync()
            }
            toast(canScore ? "你已接管本场记分" : (viewerUi.readonlyHint.isEmpty ? "本场已切换裁判记分" : viewerUi.readonlyHint))
        } catch {
            // 角色变化的刷新失败保留当前视图，等待下一次同步修正。
        }
    }

    func snookerActorLabel(_ actor: Int) -> String {
        guard actor == 1 || actor == 2 else { return "待确定" }
        if isReferee { return actor == 1 ? "选手1" : "选手2" }
        let myActor = isPlayer1 ? 1 : 2
        return actor == myActor ? "我方" : "对手"
    }

    // MARK: - 写操作

    private func actionPayload(_ fields: [String: MatchActionValue] = [:]) -> [String: MatchActionValue] {
        var payload = fields
        payload["match_id"] = .int(Int(matchId))
        return MatchAction.buildPayload(payload, baseRevision: Int(serverRevision))
    }

    /// 所有普通写操作共用一把锁：进行中重复点击直接忽略，避免同一动作发两次。
    @discardableResult
    private func send(
        _ path: String,
        payload: [String: MatchActionValue],
        successMessage: String?,
        failureMessage: String,
        businessFailureMessage: String? = nil,
        success: ((LiveMatchWriteResponse) -> Void)? = nil
    ) async -> LiveMatchWriteResponse? {
        guard !writeInFlight else { return nil }
        writeInFlight = true
        isSyncing = true
        let identity = session.identity
        defer {
            writeInFlight = false
            isSyncing = false
        }

        do {
            let response = try await api.perform(path, payload: payload)
            guard session.identity == identity else { return nil }
            if response.success == true {
                applyWriteResponse(response)
                invalidateRankAfterSettlement(response)
                if let successMessage {
                    toast(successMessage.isEmpty ? (response.message ?? "操作成功") : successMessage)
                }
                success?(response)
            } else {
                applyWriteResponse(response)
                toast(response.message ?? businessFailureMessage ?? failureMessage)
            }
            return response
        } catch {
            guard session.identity == identity else { return nil }
            toast(Self.writeFailureMessage(error, businessFallback: businessFailureMessage, networkFallback: failureMessage))
            return nil
        }
    }

    private func applyWriteResponse(_ response: LiveMatchWriteResponse) {
        if let snapshot = response.snapshot {
            applySnapshot(snapshot)
            return
        }
        guard shouldApplyIncomingRevision(response.serverRevision) else { return }
        updateRevision(response.serverRevision)
        if let value = response.myScore { myScore = value }
        if let value = response.opponentScore { opponentScore = value }
        if let value = response.currentFrameStarted { currentFrameStarted = value }
        if let value = response.currentFrameMyScore { currentFrameMyScore = value }
        if let value = response.currentFrameOpponentScore { currentFrameOpponentScore = value }
        if let value = response.finishState {
            finishState = value.isEmpty ? "none" : value
        }
        if let value = response.finishRequestedBy { finishRequestedBy = value }
    }

    func addRoundWin(winType: String, score: Int = 1, winnerActor: Int = 2) async {
        guard ensure(canScore, "当前只有裁判可以记分") else { return }
        let winner = MatchScoringTarget.resolveMatchApiActor(
            viewerRole: viewerRole,
            isPlayer1: isPlayer1,
            uiActor: winnerActor
        )
        await send(
            "/api/match/round/end",
            payload: actionPayload([
                "winner": .int(winner),
                "win_type": .string(winType),
                "score": .int(score),
            ]),
            successMessage: MatchScoringTarget.resolveWinFeedback(viewerRole: viewerRole, uiActor: winnerActor),
            failureMessage: "操作失败"
        )
    }

    func foul(actor: Int) async {
        guard ensure(canScore, "当前只有裁判可以记分") else { return }
        let apiActor = MatchScoringTarget.resolveMatchApiActor(
            viewerRole: viewerRole,
            isPlayer1: isPlayer1,
            uiActor: actor
        )
        await send(
            "/api/match/foul",
            payload: actionPayload(["actor": .int(apiActor)]),
            successMessage: MatchScoringTarget.resolveFoulFeedback(
                viewerRole: viewerRole,
                foulingActor: actor,
                score: 1
            ),
            failureMessage: "操作失败",
            businessFailureMessage: "比分已更新，请重试"
        )
    }

    // MARK: - 斯诺克 V1

    func selectSnookerScoringActor(_ actor: Int) {
        snookerScoringActor = actor == 1 ? 1 : 2
    }

    func selectSnookerFoulingActor(_ actor: Int) {
        snookerFoulingActor = actor == 1 ? 1 : 2
    }

    func addSnookerScore(_ score: Int) async {
        guard ensure(canScore, "当前只有裁判可以记分") else { return }
        guard currentFrameStarted else {
            toast("请先开始下一局")
            return
        }
        if score == 1 && isSnookerRedDisabled {
            toast("本局红球已打完，请改为彩球计分")
            return
        }
        if score > 1 && isSnookerScoreDisabled(score) {
            toast(snookerNextClearanceColorName.isEmpty
                ? "\(Self.snookerColorName(score))当前不可计分"
                : "请先记 \(snookerNextClearanceColorName)")
            return
        }
        let uiActor = isReferee ? snookerScoringActor : 2
        await send(
            "/api/match/score",
            payload: actionPayload([
                "actor": .int(MatchScoringTarget.resolveMatchApiActor(
                    viewerRole: viewerRole,
                    isPlayer1: isPlayer1,
                    uiActor: uiActor
                )),
                "score": .int(score),
            ]),
            successMessage: MatchScoringTarget.resolveScoreFeedback(
                viewerRole: viewerRole,
                uiActor: uiActor,
                score: score
            ),
            failureMessage: "操作失败",
            businessFailureMessage: "比分已更新，请重试"
        )
    }

    func foulByScore(_ score: Int) async {
        guard ensure(canScore, "当前只有裁判可以记分") else { return }
        let uiActor = isReferee ? snookerFoulingActor : 2
        await send(
            "/api/match/foul",
            payload: actionPayload([
                "actor": .int(MatchScoringTarget.resolveMatchApiActor(
                    viewerRole: viewerRole,
                    isPlayer1: isPlayer1,
                    uiActor: uiActor
                )),
                "score": .int(score),
            ]),
            successMessage: MatchScoringTarget.resolveFoulFeedback(
                viewerRole: viewerRole,
                foulingActor: uiActor,
                score: score
            ),
            failureMessage: "操作失败",
            businessFailureMessage: "比分已更新，请重试"
        )
    }

    // MARK: - 斯诺克 V2

    private func submitSnookerV2Stroke(_ fields: [String: MatchActionValue], successMessage: String) async -> Bool {
        guard ensure(canScore, "当前只有裁判可以记分") else { return false }
        guard isSnookerV2 else { return false }
        let response = await send(
            "/api/match/snooker/stroke",
            payload: actionPayload(fields),
            successMessage: successMessage,
            failureMessage: "操作失败",
            businessFailureMessage: "局面已更新，请重试"
        ) { [weak self] response in
            if response.snapshot?.status == 2 { self?.navigateToResultOnce() }
        }
        return response?.success == true
    }

    private func submitSnookerV2FrameAction(_ fields: [String: MatchActionValue], successMessage: String) async -> Bool {
        guard ensure(canScore, "当前只有裁判可以处理局动作") else { return false }
        guard isSnookerV2 else { return false }
        let response = await send(
            "/api/match/snooker/frame-action",
            payload: actionPayload(fields),
            successMessage: successMessage,
            failureMessage: "操作失败",
            businessFailureMessage: "局面已更新，请重试"
        ) { [weak self] response in
            if response.snapshot?.status == 2 { self?.navigateToResultOnce() }
        }
        return response?.success == true
    }

    private func potFields(_ payload: SnookerV2PotPayload) -> [String: MatchActionValue] {
        [
            "actor": .int(payload.actor),
            "outcome": .string(payload.outcome),
            "ball_on_value": .int(payload.ballOnValue),
            "potted_reds": .int(payload.pottedReds),
            "ball_on_potted": .bool(payload.ballOnPotted),
            "free_ball_value": .int(payload.freeBallValue),
            "free_ball_potted": .bool(payload.freeBallPotted),
        ]
    }

    private func noScoreFields(_ payload: SnookerV2NoScorePayload) -> [String: MatchActionValue] {
        var fields: [String: MatchActionValue] = [
            "actor": .int(payload.actor),
            "outcome": .string(payload.outcome),
            "ball_on_value": .int(payload.ballOnValue),
        ]
        if let freeBallValue = payload.freeBallValue {
            fields["free_ball_value"] = .int(freeBallValue)
        }
        return fields
    }

    private func foulFields(_ payload: SnookerV2FoulPayload) -> [String: MatchActionValue] {
        [
            "actor": .int(payload.actor),
            "outcome": .string(payload.outcome),
            "ball_on_value": .int(payload.ballOnValue),
            "penalty": .int(payload.penalty),
            "reds_removed": .int(payload.redsRemoved),
            "foul_resolution": .string(payload.foulResolution),
            "free_ball_awarded": .bool(payload.freeBallAwarded),
            "foul_and_miss": .bool(payload.foulAndMiss),
            "miss_sequence_eligible": .bool(payload.missSequenceEligible),
            "cue_ball_in_hand": .bool(payload.cueBallInHand),
        ]
    }

    func potSnookerV2Ball(_ ball: SnookerBall) async {
        let actor = snookerV2State.striker
        guard actor != 0 else { return }
        if ball.value == 1 {
            // 单击红球默认进 1 颗；一杆多颗另走单独入口。
            await submitSnookerV2Stroke(
                potFields(SnookerV2.potPayload(state: snookerV2State, actor: actor, pottedReds: 1)),
                successMessage: "已记录1颗红球"
            )
            return
        }
        await submitSnookerV2Stroke(
            potFields(SnookerV2.potPayload(
                state: snookerV2State,
                actor: actor,
                ballOnValue: ball.value,
                ballOnPotted: true
            )),
            successMessage: "已记录\(ball.label) \(String(ball.value))分"
        )
    }

    func startSnookerV2MultiRed() {
        let actor = snookerV2State.striker
        guard actor != 0 else { return }
        chooseSnookerCount(
            minimum: 2,
            maximum: max(0, snookerV2State.redsRemaining),
            label: { "\(String($0)) 颗红球入袋（\(String($0))分）" }
        ) { [weak self] count in
            guard let self, let count else { return }
            Task {
                await self.submitSnookerV2Stroke(
                    self.potFields(SnookerV2.potPayload(state: self.snookerV2State, actor: actor, pottedReds: count)),
                    successMessage: "已记录\(String(count))颗红球"
                )
            }
        }
    }

    func startSnookerV2FreeBall() {
        let state = snookerV2State
        guard state.freeBallAvailable, state.striker != 0 else { return }
        if state.ballOn == "color_choice" {
            presentChoice(options: snookerColorOptions.map { "本杆以\($0.label)为目标球" }) { [weak self] index in
                guard let self, self.snookerColorOptions.indices.contains(index) else { return }
                self.continueFreeBall(ballOnValue: self.snookerColorOptions[index].value)
            }
            return
        }
        continueFreeBall(ballOnValue: SnookerV2.ballOnValue(state))
    }

    private func continueFreeBall(ballOnValue: Int) {
        let state = snookerV2State
        let freeOptions = SnookerV2.freeBallOptions(state, ballOnValue: ballOnValue)
        guard !freeOptions.isEmpty else { return }
        presentChoice(options: freeOptions.map { "指定\($0.label)为自由球" }) { [weak self] index in
            guard let self, freeOptions.indices.contains(index) else { return }
            let freeBallValue = freeOptions[index].value
            if state.ballOn == "red" {
                self.presentFreeBallResultForRed(
                    ballOnValue: ballOnValue,
                    freeBallValue: freeBallValue
                )
            } else {
                self.presentFreeBallResultForColor(
                    ballOnValue: ballOnValue,
                    freeBallValue: freeBallValue
                )
            }
        }
    }

    private func presentFreeBallResultForRed(ballOnValue: Int, freeBallValue: Int) {
        presentChoice(options: ["自由球入袋", "自由球未入袋"]) { [weak self] index in
            guard let self else { return }
            let freeBallPotted = index == 0
            self.chooseSnookerCount(
                minimum: 0,
                maximum: self.snookerV2State.redsRemaining,
                label: { $0 == 0 ? "没有真实红球入袋" : "\(String($0)) 颗真实红球入袋" }
            ) { [weak self] pottedReds in
                guard let self, let pottedReds else { return }
                if !freeBallPotted && pottedReds == 0 {
                    Task {
                        await self.submitSnookerV2Stroke(
                            self.noScoreFields(SnookerV2.noScorePayload(
                                state: self.snookerV2State,
                                actor: self.snookerV2State.striker,
                                ballOnValue: ballOnValue,
                                freeBallValue: freeBallValue
                            )),
                            successMessage: "自由球击球未进，上手已结束"
                        )
                    }
                    return
                }
                Task {
                    await self.submitSnookerV2Stroke(
                        self.potFields(SnookerV2.potPayload(
                            state: self.snookerV2State,
                            actor: self.snookerV2State.striker,
                            ballOnValue: ballOnValue,
                            pottedReds: pottedReds,
                            freeBallValue: freeBallValue,
                            freeBallPotted: freeBallPotted
                        )),
                        successMessage: "自由球结果已记录"
                    )
                }
            }
        }
    }

    private func presentFreeBallResultForColor(ballOnValue: Int, freeBallValue: Int) {
        presentChoice(options: [
            "仅自由球入袋",
            "仅真实目标球入袋",
            "自由球与真实目标球同时入袋",
            "均未入袋",
        ]) { [weak self] index in
            guard let self else { return }
            if index == 3 {
                Task {
                    await self.submitSnookerV2Stroke(
                        self.noScoreFields(SnookerV2.noScorePayload(
                            state: self.snookerV2State,
                            actor: self.snookerV2State.striker,
                            ballOnValue: ballOnValue,
                            freeBallValue: freeBallValue
                        )),
                        successMessage: "自由球击球未进，上手已结束"
                    )
                }
                return
            }
            Task {
                await self.submitSnookerV2Stroke(
                    self.potFields(SnookerV2.potPayload(
                        state: self.snookerV2State,
                        actor: self.snookerV2State.striker,
                        ballOnValue: ballOnValue,
                        freeBallValue: freeBallValue,
                        freeBallPotted: index != 1,
                        ballOnPotted: index != 0
                    )),
                    successMessage: "自由球结果已记录"
                )
            }
        }
    }

    func snookerV2NoScore() {
        let state = snookerV2State
        guard state.striker != 0 else { return }
        if state.ballOn == "color_choice" {
            presentChoice(options: snookerColorOptions.map { "本杆以\($0.label)为目标球" }) { [weak self] index in
                guard let self, self.snookerColorOptions.indices.contains(index) else { return }
                self.submitSnookerV2NoScore(ballOnValue: self.snookerColorOptions[index].value)
            }
            return
        }
        submitSnookerV2NoScore(ballOnValue: SnookerV2.ballOnValue(state))
    }

    private func submitSnookerV2NoScore(ballOnValue: Int) {
        Task {
            await submitSnookerV2Stroke(
                noScoreFields(SnookerV2.noScorePayload(
                    state: snookerV2State,
                    actor: snookerV2State.striker,
                    ballOnValue: ballOnValue
                )),
                successMessage: "上手已结束"
            )
        }
    }

    func toggleSnookerFoulEditor() {
        guard currentFrameStarted, snookerV2State.pendingConcessionActor == 0 else { return }
        isSnookerFoulEditorShown.toggle()
    }

    func setSnookerFoulPenalty(_ value: Int) {
        snookerFoulPenalty = value
    }

    func selectSnookerFoulBallOnValue(_ value: Int) {
        snookerFoulBallOnValue = value
    }

    func adjustSnookerFoulReds(_ change: Int) {
        snookerFoulRedsRemoved = min(
            max(0, snookerV2State.redsRemaining),
            max(0, snookerFoulRedsRemoved + change)
        )
    }

    func selectSnookerFoulResolution(_ value: String) {
        snookerFoulResolution = value
    }

    func toggleSnookerFoulAndMiss() { snookerFoulAndMiss.toggle() }
    func toggleSnookerMissSequenceEligible() { snookerMissSequenceEligible.toggle() }
    func toggleSnookerFreeBallAwarded() { snookerFreeBallAwarded.toggle() }
    func toggleSnookerFoulCueBallInHand() { snookerFoulCueBallInHand.toggle() }

    func submitSnookerV2Foul() async {
        let minimum = SnookerV2.minimumPenalty(snookerV2State, ballOnValue: snookerFoulBallOnValue)
        guard snookerFoulPenalty >= minimum else {
            toast("当前目标球至少罚\(String(minimum))分")
            return
        }
        let submitted = await submitSnookerV2Stroke(
            foulFields(SnookerV2.foulPayload(
                state: snookerV2State,
                actor: snookerV2State.striker,
                penalty: snookerFoulPenalty,
                ballOnValue: snookerFoulBallOnValue,
                redsRemoved: snookerFoulRedsRemoved,
                resolution: snookerFoulResolution,
                freeBallAwarded: snookerFreeBallAwarded,
                foulAndMiss: snookerFoulAndMiss,
                missSequenceEligible: snookerMissSequenceEligible,
                cueBallInHand: snookerFoulCueBallInHand
            )),
            successMessage: "已记录犯规\(String(snookerFoulPenalty))分"
        )
        guard submitted else { return }
        isSnookerFoulEditorShown = false
        snookerFoulAndMiss = false
        snookerMissSequenceEligible = false
        snookerFreeBallAwarded = false
        snookerFoulCueBallInHand = false
    }

    func startRespottedBlack(_ actor: Int) async {
        guard canStartRespottedBlack(actor) else { return }
        await submitSnookerV2FrameAction(
            ["actor": .int(actor), "action": .string("start_respotted_black")],
            successMessage: "\(snookerActorLabel(actor))先打重置黑球"
        )
    }

    func offerSnookerConcession(scope: String) {
        let actor = viewerFixedActor
        guard actor != 0 else { return }
        pendingConcessionScope = scope
        confirm = LiveMatchConfirm(
            kind: .offerSnookerConcession,
            title: scope == "match" ? "认输整场比赛" : "提出本局认输",
            message: scope == "match"
                ? "对方接受后，本场比赛将立即结束。"
                : "仅在需要对手犯规罚分才能反超时允许认输本局。",
            confirmTitle: "确认提出",
            cancelTitle: "取消"
        )
    }

    func respondToSnookerConcession(accept: Bool) async {
        let pendingActor = snookerV2State.pendingConcessionActor
        guard pendingActor == 1 || pendingActor == 2 else { return }
        let actor = pendingActor == 1 ? 2 : 1
        await submitSnookerV2FrameAction(
            [
                "actor": .int(actor),
                "action": .string(accept ? "accept_concession" : "reject_concession"),
            ],
            successMessage: accept ? "已接受认输" : "已拒绝认输"
        )
    }

    func awardSnookerFrame(winner: Int) {
        guard isReferee, winner == 1 || winner == 2 else { return }
        pendingAwardWinner = winner
        confirm = LiveMatchConfirm(
            kind: .awardSnookerFrame,
            title: "判给\(snookerActorLabel(winner))本局",
            message: "仅用于裁判依规则作出的判局决定，确认后会立即结束当前局。",
            confirmTitle: "确认判局",
            cancelTitle: "取消"
        )
    }

    func recordRefereeAcceptedConcession() {
        guard isReferee else { return }
        presentChoice(options: ["选手1认输", "选手2认输"]) { [weak self] actorIndex in
            guard let self, actorIndex == 0 || actorIndex == 1 else { return }
            self.presentChoice(options: ["认输本局", "认输整场比赛"]) { [weak self] scopeIndex in
                guard let self, scopeIndex == 0 || scopeIndex == 1 else { return }
                let concedingActor = actorIndex == 0 ? 1 : 2
                let scope = scopeIndex == 0 ? "frame" : "match"
                Task {
                    let offered = await self.submitSnookerV2FrameAction(
                        [
                            "actor": .int(concedingActor),
                            "action": .string("offer_concession"),
                            "scope": .string(scope),
                        ],
                        successMessage: "已记录认输提议"
                    )
                    guard offered else { return }
                    await self.submitSnookerV2FrameAction(
                        [
                            "actor": .int(concedingActor == 1 ? 2 : 1),
                            "action": .string("accept_concession"),
                        ],
                        successMessage: "已记录双方接受认输"
                    )
                }
            }
        }
    }

    // MARK: - 弹层选择

    private func presentChoice(title: String = "", options: [String], handler: @escaping (Int) -> Void) {
        choiceHandler = handler
        choice = LiveMatchChoice(
            title: title,
            options: options.map { LiveMatchChoice.Option(title: $0) }
        )
    }

    func selectChoice(_ index: Int) {
        let handler = choiceHandler
        choice = nil
        choiceHandler = nil
        guard index >= 0 else { return }
        handler?(index)
    }

    func cancelChoice() {
        choice = nil
        choiceHandler = nil
    }

    private func chooseSnookerCount(
        minimum: Int,
        maximum: Int,
        label: @escaping (Int) -> String,
        completion: @escaping (Int?) -> Void
    ) {
        let pages = SnookerV2.countPages(minimum: minimum, maximum: maximum)
        guard let firstPage = pages.first else {
            completion(nil)
            return
        }
        let selectValue: ([Int]) -> Void = { [weak self] values in
            guard let self else { return }
            self.presentChoice(options: values.map(label)) { index in
                completion(values.indices.contains(index) ? values[index] : nil)
            }
        }
        guard pages.count > 1 else {
            selectValue(firstPage)
            return
        }
        presentChoice(options: pages.map { "\(String($0[0]))–\(String($0[$0.count - 1])) 颗" }) { index in
            guard pages.indices.contains(index) else {
                completion(nil)
                return
            }
            selectValue(pages[index])
        }
    }

    func undo() async {
        guard ensure(canUndo, "当前只有裁判可以撤销操作") else { return }
        await send(
            "/api/match/undo",
            payload: actionPayload(),
            successMessage: nil,
            failureMessage: "撤销失败"
        ) { [weak self] response in
            guard let self else { return }
            Task { await self.channel.requestLiveSync() }
            self.toast(response.message?.isEmpty == false ? response.message! : "已撤销")
        }
    }

    func requestNextRound() {
        guard ensure(canScore, "当前只有裁判可以开始下一局") else { return }
        if isSnookerV2 {
            guard !currentFrameStarted else {
                toast("当前局尚未按规则结束")
                return
            }
            showNextRoundConfirm("确定开始新的一局吗？")
            return
        }

        let action = SnookerFrame.nextFrameAction(SnookerFrameSettleInput(
            currentFrameStarted: currentFrameStarted,
            myFrameScore: currentFrameMyScore,
            opponentFrameScore: currentFrameOpponentScore
        ))
        if action.action == "blocked" {
            toast(action.message ?? "无法开始下一局")
            return
        }
        if action.action == "settle_and_start_next_round" {
            let label = MatchScoringTarget.resolveScoringParticipantLabel(
                viewerRole: viewerRole,
                uiActor: action.winner ?? 2
            )
            showNextRoundConfirm("将按当前局比分 \(currentFrameMyScore):\(currentFrameOpponentScore) 自动判定\(label)赢下本局，并开始下一局。")
            return
        }
        showNextRoundConfirm("确定开始新的一局吗？")
    }

    private func showNextRoundConfirm(_ message: String) {
        confirm = LiveMatchConfirm(
            kind: .nextRound,
            title: "开始下一局",
            message: message,
            confirmTitle: "确定",
            cancelTitle: "取消"
        )
    }

    private func executeNextRound() async {
        guard !writeInFlight else { return }
        if isSnookerV2 {
            await startNextRoundRequest()
            return
        }
        let action = SnookerFrame.nextFrameAction(SnookerFrameSettleInput(
            currentFrameStarted: currentFrameStarted,
            myFrameScore: currentFrameMyScore,
            opponentFrameScore: currentFrameOpponentScore
        ))
        if action.action == "blocked" {
            toast(action.message ?? "无法开始下一局")
            return
        }
        if action.action == "settle_and_start_next_round" {
            guard let winner = action.winner else { return }
            writeInFlight = true
            isSyncing = true
            defer {
                writeInFlight = false
                isSyncing = false
            }
            let identity = session.identity
            do {
                let response = try await api.endRound(actionPayload([
                    "winner": .int(MatchScoringTarget.resolveMatchApiActor(
                        viewerRole: viewerRole,
                        isPlayer1: isPlayer1,
                        uiActor: winner
                    )),
                    "win_type": .string("normal"),
                    "score": .int(1),
                ]))
                guard session.identity == identity else { return }
                applyWriteResponse(response)
                guard response.success == true else {
                    toast(response.message ?? "本局结算失败")
                    return
                }
            } catch {
                guard session.identity == identity else { return }
                toast(Self.failureMessage(error, fallback: "操作失败"))
                return
            }
        }
        await startNextRoundRequest()
    }

    private func startNextRoundRequest() async {
        await send(
            "/api/match/round/start",
            payload: actionPayload(),
            successMessage: nil,
            failureMessage: "操作失败"
        ) { [weak self] response in
            guard let self else { return }
            if let round = response.roundNo {
                self.toast("第\(round)局开始")
            }
        }
    }

    // MARK: - 赛制

    func toggleFormatEditor() {
        guard canChangeMatchFormat, !isFormatSaving else { return }
        isFormatEditorShown.toggle()
        guard isFormatEditorShown else { return }
        draftMatchFormat = matchFormat == PoolMatchFormat.raceTo ? PoolMatchFormat.raceTo : PoolMatchFormat.free
        draftTargetWins = targetWins > 0 ? targetWins : 10
        formatError = ""
    }

    func selectFormatMode(_ format: String) {
        draftMatchFormat = format == PoolMatchFormat.raceTo ? PoolMatchFormat.raceTo : PoolMatchFormat.free
        formatError = ""
    }

    func setDraftTargetWins(_ value: Int) {
        draftTargetWins = min(max(value, 1), PoolMatchFormat.maxTargetWins)
    }

    func saveMatchFormat() async {
        guard canChangeMatchFormat, !isFormatSaving else { return }
        isFormatSaving = true
        formatError = ""
        let identity = session.identity
        let target = draftMatchFormat == PoolMatchFormat.raceTo ? draftTargetWins : 0
        defer { isFormatSaving = false }

        do {
            let response = try await api.updateMatchFormat(
                matchId: matchId,
                format: draftMatchFormat,
                targetWins: target,
                baseRevision: serverRevision
            )
            guard session.identity == identity else { return }
            if let snapshot = response.snapshot {
                applySnapshot(snapshot)
            } else if response.success != true {
                await load(showLoading: false)
            }
            guard response.success == true else {
                formatError = response.message ?? "赛制保存失败，请重试"
                return
            }
            isFormatEditorShown = false
            toast("赛制已更新")
        } catch {
            guard session.identity == identity else { return }
            formatError = Self.failureMessage(error, fallback: "赛制保存失败，请重试")
        }
    }

    // MARK: - 斯诺克赛制

    func toggleSnookerFormatEditor() {
        guard canChangeSnookerFormat, !isSnookerFormatSaving else { return }
        isSnookerFormatEditorShown.toggle()
        guard isSnookerFormatEditorShown else { return }
        draftSnookerFormat = snookerFormat == SnookerMatchFormat.raceTo
            ? SnookerMatchFormat.raceTo
            : SnookerMatchFormat.free
        draftSnookerTargetWins = snookerTargetWins > 0 ? snookerTargetWins : 3
        snookerFormatError = ""
    }

    func selectSnookerFormatMode(_ format: String) {
        draftSnookerFormat = format == SnookerMatchFormat.raceTo
            ? SnookerMatchFormat.raceTo
            : SnookerMatchFormat.free
        snookerFormatError = ""
    }

    func setDraftSnookerTargetWins(_ value: Int) {
        draftSnookerTargetWins = min(max(value, 1), SnookerMatchFormat.maxTargetWins)
    }

    func saveSnookerFormat() async {
        guard canChangeSnookerFormat, !isSnookerFormatSaving else { return }
        isSnookerFormatSaving = true
        snookerFormatError = ""
        let identity = session.identity
        let target = draftSnookerFormat == SnookerMatchFormat.raceTo ? draftSnookerTargetWins : 0
        defer { isSnookerFormatSaving = false }

        do {
            let response = try await api.updateSnookerFormat(
                matchId: matchId,
                format: draftSnookerFormat,
                targetWins: target,
                baseRevision: serverRevision
            )
            guard session.identity == identity else { return }
            if let snapshot = response.snapshot {
                applySnapshot(snapshot)
            } else if response.success != true {
                await load(showLoading: false)
            }
            guard response.success == true else {
                snookerFormatError = response.message ?? "赛制保存失败，请重试"
                return
            }
            isSnookerFormatEditorShown = false
            toast("赛制已更新")
        } catch {
            guard session.identity == identity else { return }
            snookerFormatError = Self.failureMessage(error, fallback: "赛制保存失败，请重试")
        }
    }

    // MARK: - 结束流程

    func requestFinish() {
        guard !writeInFlight, !finishActionBusy else { return }
        if isPoolFormatVisible, matchFormat == PoolMatchFormat.free,
           !PoolMatchFormat.canFinishFreePoolMatch(format: matchFormat, myScore: myScore, opponentScore: opponentScore) {
            return
        }
        guard ensure(canRequestFinish, "当前无法发起结束确认") else { return }
        confirm = LiveMatchConfirm(
            kind: .requestFinish,
            title: "发起结束确认",
            message: "比分将冻结，对手需在30秒内确认；期间你可撤销，无人处理则按当前比分自动结束，之后不能再提出异议。",
            confirmTitle: "发起结束",
            cancelTitle: "取消"
        )
    }

    func confirmFinish() async {
        guard ensure(canConfirmFinish, "当前没有可确认的结束请求") else { return }
        await submitFinish(.confirm)
    }

    func disputeFinish() async {
        guard ensure(canDisputeFinish, "当前没有可异议的结束请求") else { return }
        await submitFinish(.dispute)
    }

    func withdrawFinish() async {
        guard ensure(canWithdrawFinish, "当前没有可撤销的结束请求") else { return }
        await submitFinish(.withdraw)
    }

    private func submitFinish(_ submission: FinishSubmission) async {
        guard !finishActionBusy, !writeInFlight else { return }
        finishActionBusy = true
        isSyncing = true
        let identity = session.identity
        defer {
            finishActionBusy = false
            isSyncing = false
        }

        let request: (path: String, success: String, failure: String) = switch submission {
        case .request: ("/api/match/finish/request", "已发起结束，等待对方确认", "发起结束失败")
        case .confirm: ("/api/match/finish/confirm", "已确认结束并完成结算", "确认结束失败")
        case .dispute: ("/api/match/finish/dispute", "已提出异议，对局继续", "提出异议失败")
        case .withdraw: ("/api/match/finish/withdraw", "已撤销结束，对局继续", "撤销失败")
        }

        do {
            let response = try await api.perform(request.path, payload: actionPayload())
            guard session.identity == identity else { return }
            applyFinishActionResponse(response, successMessage: request.success)
        } catch {
            guard session.identity == identity else { return }
            toast(Self.failureMessage(error, fallback: request.failure))
        }
    }

    private func applyFinishActionResponse(_ response: LiveMatchWriteResponse, successMessage: String) {
        if response.success == true {
            applyWriteResponse(response)
            invalidateRankAfterSettlement(response)
            toast(successMessage)
            if response.snapshot?.status == 2 {
                navigateToResultOnce()
            }
            return
        }
        applyWriteResponse(response)
        toast(response.message ?? "操作失败")
        // 确认期已满被拒时，服务端已按冻结比分结束本场，直接进入赛果。
        if response.snapshot?.status == 2 {
            navigateToResultOnce()
        }
    }

    func finishMatch() {
        guard !writeInFlight, !finishActionBusy else { return }
        if isSnookerV2, snookerFormat == SnookerMatchFormat.free,
           !SnookerMatchFormat.canFinishFreeSnookerMatch(
               format: snookerFormat,
               currentFrameStarted: currentFrameStarted,
               myScore: myScore,
               opponentScore: opponentScore
           ) {
            return
        }
        if isPoolFormatVisible, matchFormat == PoolMatchFormat.free,
           !PoolMatchFormat.canFinishFreePoolMatch(format: matchFormat, myScore: myScore, opponentScore: opponentScore) {
            return
        }
        guard ensure(canFinish, "当前只有裁判可以结束对局") else { return }

        let action: SnookerFrameAction
        if gameType == 1 {
            action = SnookerFrame.finishMatchAction(SnookerFrameSettleInput(
                currentFrameStarted: currentFrameStarted,
                myFrameScore: currentFrameMyScore,
                opponentFrameScore: currentFrameOpponentScore
            ))
        } else {
            action = SnookerFrameAction(action: "finish_match", winner: nil, message: nil)
        }
        if action.action == "blocked" {
            toast(action.message ?? "无法结束对局")
            return
        }
        pendingFinishAction = action

        if isReferee {
            confirm = LiveMatchConfirm(
                kind: .finishMatch,
                title: "确认最终比分并完成结算",
                message: "双方最终比分：\(myScore) : \(opponentScore)\n\n完成后立即结算且无需选手确认",
                confirmTitle: "确认完成",
                cancelTitle: "继续记分"
            )
            return
        }

        let message = action.action == "settle_and_finish_match"
            ? "确定要结束本次对局吗？\n\n按当前局比分自动判定后结束"
            : "确定要结束本次对局吗？"
        confirm = LiveMatchConfirm(
            kind: .finishMatch,
            title: "确认结束",
            message: message,
            confirmTitle: "确定",
            cancelTitle: "取消"
        )
    }

    private func executeFinishMatch(_ action: SnookerFrameAction) async {
        guard !writeInFlight else { return }
        writeInFlight = true
        isSyncing = true
        let identity = session.identity
        defer {
            writeInFlight = false
            isSyncing = false
            pendingFinishAction = nil
        }

        if action.action == "settle_and_finish_match", let winner = action.winner {
            do {
                let response = try await api.endRound(actionPayload([
                    "winner": .int(MatchScoringTarget.resolveMatchApiActor(
                        viewerRole: viewerRole,
                        isPlayer1: isPlayer1,
                        uiActor: winner
                    )),
                    "win_type": .string("normal"),
                    "score": .int(1),
                ]))
                guard session.identity == identity else { return }
                applyWriteResponse(response)
                guard response.success == true else {
                    toast(response.message ?? "本局结算失败")
                    return
                }
            } catch {
                guard session.identity == identity else { return }
                toast(Self.failureMessage(error, fallback: "操作失败"))
                return
            }
        }

        do {
            let response = try await api.perform("/api/match/finish", payload: actionPayload())
            guard session.identity == identity else { return }
            if response.success != true {
                applyWriteResponse(response)
                if let message = response.message, message.contains("revision") {
                    toast("比分已变化，请重新核对后完成")
                } else {
                    toast(response.message ?? "操作失败")
                }
                return
            }
            applyWriteResponse(response)
            invalidateRankAfterSettlement(response)
            onMatchSettled()
            navigateToResultOnce()
        } catch {
            guard session.identity == identity else { return }
            toast(Self.failureMessage(error, fallback: "操作失败"))
        }
    }

    /// 约球底部滑块：确认弹层里的滑块只确认结束，底部滑块按排位/练习分流。
    func commitSlide() async {
        if showFinishConfirmDialog, finishConfirmView.showConfirm {
            await confirmFinish()
            return
        }
        if canRequestFinish {
            guard !writeInFlight, !finishActionBusy else { return }
            if isPoolFormatVisible, matchFormat == PoolMatchFormat.free,
               !PoolMatchFormat.canFinishFreePoolMatch(format: matchFormat, myScore: myScore, opponentScore: opponentScore) {
                return
            }
            guard ensure(canRequestFinish, "当前无法发起结束确认") else { return }
            await submitFinish(.request)
            return
        }

        if gameType == 1 {
            let action = SnookerFrame.finishMatchAction(SnookerFrameSettleInput(
                currentFrameStarted: currentFrameStarted,
                myFrameScore: currentFrameMyScore,
                opponentFrameScore: currentFrameOpponentScore
            ))
            if action.action == "blocked" {
                toast(action.message ?? "无法结束对局")
                return
            }
            await executeFinishMatch(action)
            return
        }
        await executeFinishMatch(SnookerFrameAction(action: "finish_match", winner: nil, message: nil))
    }

    // MARK: - 倒计时

    /// 视图每秒调用；归零后等待服务端结束消息，宽限期满仍无消息就回源读取。
    func tick() {
        nowValue = now()
        guard finishState == "pending_confirmation" else {
            didScheduleSettle = false
            settleTask?.cancel()
            settleTask = nil
            return
        }
        guard finishSecondsLeft <= 0, !didScheduleSettle, !isLeaving else { return }
        didScheduleSettle = true
        let delay = Double(FinishCountdown.settleGraceMs) / 1000
        settleTask = Task { [weak self] in
            guard let self else { return }
            try? await self.sleep(delay)
            guard !Task.isCancelled else { return }
            self.settleTask = nil
            guard self.finishState == "pending_confirmation", !self.isLeaving else { return }
            await self.load(showLoading: false)
        }
    }

    // MARK: - 裁判二维码

    func openRefereeQRCode() async {
        guard showInviteRefereeAction else { return }
        isRefereeQRCodePresented = true
        await loadRefereeQRCode()
    }

    func refreshRefereeQRCode() async {
        guard isRefereeQRCodePresented else { return }
        await loadRefereeQRCode()
    }

    private func loadRefereeQRCode() async {
        qrToken += 1
        let token = qrToken
        isRefereeQRCodeLoading = true
        refereeQRCodeContent = ""
        refereeQRCodeImage = nil
        refereeQRCodeError = ""

        do {
            let response = try await api.refereeQRCode(matchId: matchId)
            guard token == qrToken else { return }
            if response.success, let content = response.qrcodeData, !content.isEmpty {
                refereeQRCodeContent = content
                refereeQRCodeImage = QRCodeImage.make(content: content, size: 240)
                isRefereeQRCodeLoading = false
                return
            }
            refereeQRCodeError = response.message?.isEmpty == false ? response.message! : "生成裁判码失败"
            toast(refereeQRCodeError)
        } catch {
            guard token == qrToken else { return }
            refereeQRCodeError = "生成裁判码失败，请点击刷新"
            toast("生成裁判码失败")
        }
        isRefereeQRCodeLoading = false
    }

    func closeRefereeQRCode() {
        qrToken += 1
        isRefereeQRCodePresented = false
        isRefereeQRCodeLoading = false
        refereeQRCodeContent = ""
        refereeQRCodeImage = nil
        refereeQRCodeError = ""
    }

    // MARK: - 返回 / 菜单

    func requestBack() {
        confirm = LiveMatchConfirm(
            kind: .back,
            title: "提示",
            message: "对局进行中，确定要退出吗？",
            confirmTitle: "退出",
            cancelTitle: "继续"
        )
    }

    func selectMenu(_ option: LiveMatchMenuOption) {
        switch option.kind {
        case .settings:
            toggleFormatEditor()
        case .finishMatch:
            // 放弃对局与中式八球/美式九球的结束入口是同一动作。
            finishMatch()
        case .concedeMatch:
            offerSnookerConcession(scope: "match")
        }
    }

    func acceptConfirm() {
        guard let confirm else { return }
        self.confirm = nil
        switch confirm.kind {
        case .finishMatch:
            guard let action = pendingFinishAction else { return }
            Task { await executeFinishMatch(action) }
        case .requestFinish:
            Task { await submitFinish(.request) }
        case .nextRound:
            Task { await executeNextRound() }
        case .offerSnookerConcession:
            guard let scope = pendingConcessionScope else { return }
            pendingConcessionScope = nil
            let actor = viewerFixedActor
            guard actor != 0 else { return }
            Task {
                await submitSnookerV2FrameAction(
                    [
                        "actor": .int(actor),
                        "action": .string("offer_concession"),
                        "scope": .string(scope),
                    ],
                    successMessage: "认输提议已发送"
                )
            }
        case .awardSnookerFrame:
            guard let winner = pendingAwardWinner else { return }
            pendingAwardWinner = nil
            Task {
                await submitSnookerV2FrameAction(
                    [
                        "actor": .int(winner),
                        "action": .string("award_frame"),
                        "winner": .int(winner),
                        "reason": .string("裁判判局"),
                    ],
                    successMessage: "已判给\(snookerActorLabel(winner))"
                )
            }
        case .back:
            shouldDismiss = true
        }
    }

    func cancelConfirm() {
        confirm = nil
        pendingFinishAction = nil
        pendingConcessionScope = nil
        pendingAwardWinner = nil
    }

    // MARK: - 页面可见性 / 身份

    func setVisible(_ visible: Bool) {
        guard isVisible != visible else { return }
        isVisible = visible
        channel.setLiveActive(visible)
        guard visible, !isLeaving, phase == .ready else { return }
        Task { await resume() }
    }

    private func resume() async {
        await load(showLoading: false)
        guard isVisible, !isLeaving, phase == .ready else { return }
        if channel.isConnected {
            await channel.requestLiveSync()
        } else {
            await connectChannel()
        }
    }

    /// 页面被关闭：停止通道与倒计时，迟到响应由身份 / 令牌校验丢弃。
    func stop() {
        isLeaving = true
        settleTask?.cancel()
        settleTask = nil
        channel.setLiveActive(false)
        channel.closeLive()
        if let messageToken {
            channel.stopObservingLive(messageToken)
            self.messageToken = nil
        }
    }

    func handleIdentityChange() {
        guard session.identity != pageIdentity else { return }
        loadToken += 1
        settleTask?.cancel()
        settleTask = nil
        channel.closeLive()
        if let messageToken {
            channel.stopObservingLive(messageToken)
            self.messageToken = nil
        }
        isLeaving = false
        pageIdentity = session.identity

        guard session.identity != nil else {
            phase = .failed("登录状态已变更，请返回重试")
            return
        }
        resetForReload()
        messageToken = channel.observeLive { [weak self] message in
            self?.handle(message)
        }
        Task { await load(showLoading: true) }
    }

    private func resetForReload() {
        phase = .loading
        myScore = 0
        opponentScore = 0
        currentFrameMyScore = 0
        currentFrameOpponentScore = 0
        currentFrameStarted = true
        currentRound = 1
        serverRevision = 0
        viewerRole = "player1"
        canScore = false
        canUndo = false
        canFinish = false
        canRequestFinish = false
        canConfirmFinish = false
        canDisputeFinish = false
        canWithdrawFinish = false
        finishState = "none"
        finishRequestedBy = 0
        finishCountdownAnchor = nil
        confirm = nil
        pendingFinishAction = nil
        destination = nil
        challengeId = 0
        lastAction = nil
        matchFormat = ""
        targetWins = 0
        canChangeMatchFormat = false
        isFormatEditorShown = false
        formatError = ""
        snookerRulesVersion = 0
        snookerFormat = SnookerMatchFormat.free
        snookerTargetWins = 0
        canChangeSnookerFormat = false
        isSnookerFormatEditorShown = false
        draftSnookerFormat = SnookerMatchFormat.free
        draftSnookerTargetWins = 3
        snookerFormatError = ""
        snookerRedBallCount = 0
        snookerClearanceStarted = false
        snookerClearedColors = []
        snookerExpectedClearanceScore = 0
        snookerClearanceCompleted = false
        snookerScoringActor = 1
        snookerFoulingActor = 1
        snookerV2State = SnookerV2State()
        isSnookerFoulEditorShown = false
        snookerFoulPenalty = 4
        snookerFoulBallOnValue = 2
        snookerFoulRedsRemoved = 0
        snookerFoulResolution = "incoming_plays"
        snookerFoulAndMiss = false
        snookerMissSequenceEligible = false
        snookerFreeBallAwarded = false
        snookerFoulCueBallInHand = false
        choice = nil
        choiceHandler = nil
        pendingConcessionScope = nil
        pendingAwardWinner = nil
        closeRefereeQRCode()
    }

    // MARK: - 结算 / 跳转

    private func invalidateRankAfterSettlement(_ snapshot: LiveMatchSnapshot) {
        if let mode = snapshot.matchMode, !mode.isEmpty {
            matchMode = mode
        }
        if LiveMatchSettlement.shouldInvalidateRank(
            isParticipant: viewerRole == "player1" || viewerRole == "player2",
            matchMode: matchMode,
            status: snapshot.status ?? status,
            result: snapshot.result,
            player1Score: snapshot.player1Score,
            player2Score: snapshot.player2Score,
            myScore: snapshot.myScore ?? myScore,
            opponentScore: snapshot.opponentScore ?? opponentScore
        ) {
            onRankInvalidated()
        }
    }

    private func invalidateRankAfterSettlement(_ response: LiveMatchWriteResponse) {
        if let snapshot = response.snapshot {
            invalidateRankAfterSettlement(snapshot)
            return
        }
        if LiveMatchSettlement.shouldInvalidateRank(
            isParticipant: viewerRole == "player1" || viewerRole == "player2",
            matchMode: matchMode,
            status: status,
            result: response.result,
            player1Score: response.player1Score,
            player2Score: response.player2Score,
            myScore: response.myScore ?? myScore,
            opponentScore: response.opponentScore ?? opponentScore
        ) {
            onRankInvalidated()
        }
    }

    private func navigateToResultOnce() {
        guard !isLeaving else { return }
        guard MatchNavigation.consumeResultNavigationGuard(&hasNavigatedToResult) else { return }
        destination = .result(matchId)
    }

    private func leaveForHistory() {
        guard !isLeaving else { return }
        isLeaving = true
        channel.closeLive()
        destination = .matchHistory
    }

    func consumeDestination() {
        destination = nil
    }

    func consumeToast() {
        toastMessage = nil
    }

    func consumeDismiss() {
        shouldDismiss = false
    }

    // MARK: - 工具

    private func ensure(_ allowed: Bool, _ message: String) -> Bool {
        guard allowed else {
            toast(message.isEmpty ? (viewerUi.readonlyHint.isEmpty ? "当前由裁判负责记分" : viewerUi.readonlyHint) : message)
            return false
        }
        return true
    }

    private func toast(_ message: String) {
        toastMessage = message
    }

    private static func firstNonEmpty(_ value: String?, _ fallback: String) -> String {
        guard let value, !value.isEmpty else { return fallback }
        return value
    }

    static func snookerColorName(_ score: Int) -> String {
        switch score {
        case 2: "黄球"
        case 3: "绿球"
        case 4: "咖啡球"
        case 5: "蓝球"
        case 6: "粉球"
        case 7: "黑球"
        default: ""
        }
    }

    /// 业务失败（HTTP 200 + success:false）沿用服务端文案；网络/服务异常用页面兜底。
    private static func failureMessage(_ error: Error, fallback: String) -> String {
        guard let apiError = error as? APIError, apiError.kind == .business, !apiError.message.isEmpty else {
            return fallback
        }
        return apiError.message
    }

    /// 请求层把 `success:false` 转成 APIError 时，服务端没给文案会填通用兜底；
    /// 这种情况下页面用自己的业务提示，避免把冲突误报成网络错误。
    private static func writeFailureMessage(
        _ error: Error,
        businessFallback: String?,
        networkFallback: String
    ) -> String {
        guard let apiError = error as? APIError, apiError.kind == .business else {
            return failureMessage(error, fallback: networkFallback)
        }
        if apiError.message == APIClient.genericBusinessMessage, let businessFallback {
            return businessFallback
        }
        return failureMessage(error, fallback: networkFallback)
    }
}
