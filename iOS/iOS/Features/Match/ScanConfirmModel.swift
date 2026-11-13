import Combine
import Foundation

/// 扫码确认页：开局选择球种 / 对局类型，或预览裁判码后确认执裁。
@MainActor
final class ScanConfirmModel: ObservableObject {
    enum Stage: Equatable {
        case loading
        case ready
        case expired
        case error
    }

    struct SelfOngoingPrompt: Equatable {
        let message: String
        let match: GuardedMatch
    }

    @Published private(set) var stage: Stage = .loading
    @Published private(set) var mode: ScanConfirmMode = .start
    @Published private(set) var errorMessage = ""
    @Published private(set) var opponent = MatchScanOpponent(id: 0, nickname: "对手", avatar: "")
    @Published private(set) var gameType = 0
    @Published private(set) var defaultGameType = 0
    @Published private(set) var setAsDefault = false
    @Published private(set) var modeIndex = 0
    @Published private(set) var submitting = false
    @Published private(set) var refereePreview: RefereePreviewInfo?
    @Published private(set) var joining = false
    /// 需要进入的进行中对局；视图消费后调用 `consumeNavigation()`。
    @Published private(set) var liveMatchId: Int64?
    /// 需要以轻提示告知用户的结果。
    @Published private(set) var toastMessage: String?
    /// 自己还有未结束的对局：弹窗确认后进入该对局。
    @Published private(set) var selfOngoingPrompt: SelfOngoingPrompt?

    private let session: SessionStore
    private let api: MatchStartAPI
    private let preferences: GameTypePreference
    private var pageIdentity: SessionIdentity?
    private var inviteToken = ""
    private var refereeJoin: RefereeJoin?
    private var didLoadRefereePreview = false

    /// 重新扫码时保留原始 rematch 上下文。
    private(set) var rematch: ScanRematch?

    init(
        session: SessionStore,
        handoff: ScanHandoff?,
        api: MatchStartAPI? = nil,
        preferences: GameTypePreference? = nil
    ) {
        self.session = session
        self.api = api ?? MatchStartAPI(session: session)
        self.preferences = preferences ?? GameTypePreference()
        apply(handoff: handoff)
    }

    var navigationTitle: String {
        mode == .referee ? "确认执裁" : "确认对局"
    }

    /// 默认球种本身不需要「设为默认」入口。
    var canSetDefault: Bool {
        gameType > 0 && gameType != defaultGameType
    }

    var refereeModeText: String {
        refereePreview?.matchMode == "ranked" ? "排位赛" : "练习赛"
    }

    /// 确认页只加载一次裁判预览；开局模式无需请求。
    func loadIfNeeded() async {
        guard mode == .referee, stage == .loading, !didLoadRefereePreview else { return }
        didLoadRefereePreview = true
        await loadRefereePreview()
    }

    func selectGameType(_ value: Int) {
        gameType = value
        setAsDefault = false
    }

    func toggleSetAsDefault() {
        guard canSetDefault else { return }
        setAsDefault.toggle()
    }

    func selectMode(_ index: Int) {
        guard StartMatch.modeOptions.indices.contains(index) else { return }
        modeIndex = index
    }

    /// 页面重新出现时身份已变化：旧凭证不再可用。
    func handleIdentityChange() {
        guard pageIdentity != nil, !isCurrentIdentity, stage != .expired else { return }
        showExpired("登录状态已变更，请重新扫码")
    }

    func start() async {
        guard gameType > 0, !submitting else { return }
        guard isCurrentIdentity, let identity = pageIdentity else {
            showExpired("登录状态已变更，请重新扫码")
            return
        }

        let selectedGameType = gameType
        let option = StartMatch.modeOptions.indices.contains(modeIndex)
            ? StartMatch.modeOptions[modeIndex]
            : StartMatch.modeOptions[0]
        if setAsDefault {
            defaultGameType = preferences.save(userId: identity.userId, gameType: selectedGameType)
            setAsDefault = false
        }
        submitting = true
        errorMessage = ""
        defer { submitting = false }

        let scannedOpponent = StartMatchOpponent(
            id: opponent.id,
            nickname: opponent.nickname,
            avatar: opponent.avatar
        )
        do {
            let response = try await api.startMatch(StartMatch.buildPayload(
                gameType: selectedGameType,
                opponent: scannedOpponent,
                matchMode: option.matchMode,
                visibility: option.visibility,
                inviteToken: inviteToken
            ))
            // 等待期间切号或会话失效，旧账号的开局结果不再跳转。
            guard pageIdentity == identity, isCurrentIdentity else { return }
            handleStartOutcome(OngoingMatchGuard.resolveStartMatchGuardAction(
                response: response,
                selectedGameType: selectedGameType,
                scannedOpponent: scannedOpponent
            ))
        } catch {
            guard pageIdentity == identity, isCurrentIdentity else { return }
            errorMessage = Self.failureMessage(error, fallback: "匹配失败，请重试")
        }
    }

    func join() async {
        guard !joining else { return }
        guard isCurrentIdentity, let identity = pageIdentity else {
            showExpired("登录状态已变更，请重新扫码")
            return
        }
        guard let join = refereeJoin else { return }

        joining = true
        defer { joining = false }

        do {
            let response = try await api.joinMatchReferee(join)
            guard pageIdentity == identity, isCurrentIdentity else { return }
            guard response.success else {
                errorMessage = Self.firstNonEmpty(response.message, "裁判码已失效，请重新扫码")
                stage = .error
                return
            }
            liveMatchId = (response.matchId ?? 0) != 0 ? response.matchId : join.matchId
        } catch {
            guard pageIdentity == identity, isCurrentIdentity else { return }
            errorMessage = Self.failureMessage(error, fallback: "加入裁判失败，请重新扫码")
            stage = .error
        }
    }

    func confirmSelfOngoing() {
        guard let prompt = selfOngoingPrompt else { return }
        selfOngoingPrompt = nil
        liveMatchId = prompt.match.matchId
    }

    func dismissSelfOngoing() {
        selfOngoingPrompt = nil
    }

    func consumeNavigation() {
        liveMatchId = nil
    }

    func consumeToast() {
        toastMessage = nil
    }

    /// 重新扫码拿到新的交接后就地重载，避免重复压栈。
    func apply(handoff: ScanHandoff?) {
        guard let handoff else {
            showExpired("扫码信息已失效，请重新扫码")
            return
        }

        pageIdentity = handoff.identity
        rematch = handoff.rematch
        mode = handoff.mode
        didLoadRefereePreview = false
        errorMessage = ""
        submitting = false
        joining = false
        liveMatchId = nil
        toastMessage = nil
        selfOngoingPrompt = nil
        refereePreview = nil

        if handoff.mode == .referee {
            refereeJoin = handoff.refereeJoin
            inviteToken = ""
            stage = .loading
            return
        }

        refereeJoin = nil
        inviteToken = handoff.inviteToken ?? ""
        opponent = handoff.opponent ?? MatchScanOpponent(id: 0, nickname: "对手", avatar: "")
        defaultGameType = preferences.read(userId: handoff.identity.userId)
        gameType = handoff.preset?.gameType ?? defaultGameType
        modeIndex = Self.resolveModeIndex(handoff.preset)
        stage = .ready
    }

    private var isCurrentIdentity: Bool {
        guard let pageIdentity else { return false }
        return session.identity == pageIdentity
    }

    private func showExpired(_ message: String) {
        stage = .expired
        errorMessage = message
        inviteToken = ""
        refereeJoin = nil
    }

    private func loadRefereePreview() async {
        guard let identity = pageIdentity, let refereeJoin else { return }
        stage = .loading
        do {
            let response = try await api.previewMatchReferee(refereeJoin)
            guard pageIdentity == identity, isCurrentIdentity else { return }
            guard response.success, let preview = response.preview else {
                errorMessage = Self.firstNonEmpty(response.message, "裁判码已失效，请重新扫码")
                stage = .error
                return
            }
            refereePreview = preview
            stage = .ready
        } catch {
            guard pageIdentity == identity, isCurrentIdentity else { return }
            errorMessage = Self.failureMessage(error, fallback: "裁判码已失效，请重新扫码")
            stage = .error
        }
    }

    private func handleStartOutcome(_ outcome: StartMatchGuardOutcome) {
        switch outcome {
        case .navigate(let match), .resume(let match):
            liveMatchId = match.matchId
        case .promptSelfOngoing(let message, let match):
            selfOngoingPrompt = SelfOngoingPrompt(message: message, match: match)
        case .toastOpponentOngoing(let message):
            toastMessage = message
        case .error(let message):
            errorMessage = message
        }
    }

    private static func resolveModeIndex(_ preset: ScanRematchPreset?) -> Int {
        guard let preset else { return 0 }
        let index = StartMatch.modeOptions.firstIndex {
            $0.matchMode == preset.matchMode && $0.visibility == preset.visibility
        }
        return index ?? 0
    }

    /// 业务失败（HTTP 200 + success:false）沿用服务端文案；网络/服务异常用页面兜底。
    private static func failureMessage(_ error: Error, fallback: String) -> String {
        guard let apiError = error as? APIError, apiError.kind == .business else { return fallback }
        return apiError.message
    }

    private static func firstNonEmpty(_ value: String?, _ fallback: String) -> String {
        guard let value, !value.isEmpty else { return fallback }
        return value
    }
}
