import Combine
import Foundation

/// 等待对手页，对应小程序 `subPages/match/challengeWaiting.vue`。
/// 五秒轮询由页面 `.task` 驱动，睡眠可注入以便测试。
@MainActor
final class ChallengeWaitingModel: ObservableObject {
    static let pollInterval: TimeInterval = 5

    enum Phase: Equatable {
        case loading
        case loaded
        case failed
    }

    enum Prompt: Identifiable, Equatable {
        case closeOwn
        case typeConflict(message: String)
        case confirmEarly(schedule: String)
        case cancel
        case abandon
        case ongoingMatch(message: String, matchId: Int64)

        var id: String {
            switch self {
            case .closeOwn: "close-own"
            case .typeConflict: "type-conflict"
            case .confirmEarly: "confirm-early"
            case .cancel: "cancel"
            case .abandon: "abandon"
            case .ongoingMatch: "ongoing-match"
            }
        }
    }

    enum Navigation: Equatable {
        case match(Int64)
    }

    @Published private(set) var phase: Phase = .loading
    @Published private(set) var challenge: ChallengeInfo?
    @Published private(set) var serverTime = ""
    @Published private(set) var isActing = false
    @Published var prompt: Prompt?
    @Published var feedback: String?
    @Published var navigation: Navigation?

    private let session: SessionStore
    private let activity: ActivityStore
    private let challengeId: Int64
    private let api: ChallengeAPI
    private let sleep: (TimeInterval) async throws -> Void
    private var requestToken = 0
    private var isActive = true

    init(
        session: SessionStore,
        activity: ActivityStore,
        challengeId: Int64,
        api: ChallengeAPI? = nil,
        sleep: @escaping (TimeInterval) async throws -> Void = { try await Task.sleep(for: .seconds($0)) }
    ) {
        self.session = session
        self.activity = activity
        self.challengeId = challengeId
        self.api = api ?? ChallengeAPI(session: session)
        self.sleep = sleep
    }

    // MARK: - 展示

    var myUserId: Int64 { session.identity?.userId ?? 0 }

    var opponentUserId: Int64 {
        guard let challenge else { return 0 }
        return challenge.fromUserId == myUserId ? challenge.toUserId : challenge.fromUserId
    }

    var gameTypeLabel: String {
        guard let challenge else { return "台球" }
        return GameTypeCatalog.label(for: challenge.gameType)
    }

    var matchModeLabel: String {
        ChallengeRules.matchModeLabel(challenge?.matchMode)
    }

    var scheduleLabel: String {
        ChallengeRules.schedule(challenge, serverTime: serverTime)
    }

    /// 服务端时间判断过期：worker 未运行时也能把到期约球显示为「已失效」。
    var expiredNow: Bool {
        guard let challenge else { return false }
        return ChallengeRules.isExpired(
            status: challenge.status,
            expiresAt: challenge.expiresAt,
            serverTime: serverTime
        )
    }

    var metaHint: String {
        guard let challenge else { return "" }
        if expiredNow { return "已失效" }
        if challenge.status == 1 { return "返回或锁屏后仍会等待；暂时不打，可以退出等待" }
        if challenge.status == 0 { return ChallengeRules.expiry(challenge, serverTime: serverTime) }
        return ""
    }

    var statusLabel: String {
        guard let challenge else { return "" }
        switch challenge.status {
        case 2: return "对方这次不了"
        case 3: return "已失效"
        case 4: return "约球已取消"
        case 5: return "对方已放弃，本次约球已结束"
        case 7: return "已完成"
        case 8: return "对局已取消"
        default: return ChallengeRules.statusLabel(challenge, userId: myUserId)
        }
    }

    func isWaiting(_ userId: Int64) -> Bool {
        challenge?.waitingUserId == userId && userId > 0
    }

    // MARK: - 读取

    /// 页面出现：先读一次，随后每 5 秒刷新；任务取消（离开页面）即停止。
    func start() async {
        isActive = true
        await load()
        while !Task.isCancelled, isActive {
            do {
                try await sleep(Self.pollInterval)
            } catch {
                return
            }
            guard !Task.isCancelled, isActive else { return }
            if !isActing { await load() }
        }
    }

    func stop() {
        isActive = false
    }

    func identityChanged() async {
        phase = .loading
        challenge = nil
        serverTime = ""
        prompt = nil
        feedback = nil
        navigation = nil
        await load()
    }

    func load() async {
        guard challengeId > 0, let owner = session.identity else {
            phase = .failed
            return
        }
        requestToken += 1
        let token = requestToken
        if challenge == nil { phase = .loading }

        async let detailRequest = api.detail(id: challengeId)
        let summary = try? await api.summary()

        do {
            let detail = try await detailRequest
            guard isCurrent(token, owner) else { return }
            if let summary, summary.success, !summary.serverTime.isEmpty {
                serverTime = summary.serverTime
            }
            if detail.success, let item = detail.challenge {
                applyDetail(item)
                phase = .loaded
            } else if challenge == nil {
                phase = .failed
            }
        } catch {
            guard isCurrent(token, owner) else { return }
            if challenge == nil { phase = .failed }
        }
    }

    private func applyDetail(_ detail: ChallengeInfo) {
        let previous = challenge
        if let previous,
           previous.status != detail.status
           || previous.waitingUserId != detail.waitingUserId
           || previous.matchId != detail.matchId {
            activity.markDirty()
        }
        challenge = detail
        // 只有已见过详情后才自动跳转：首读拿到已开局记录也按页内状态处理。
        if previous != nil, detail.status == 6, let matchId = detail.matchId, matchId > 0 {
            navigation = .match(matchId)
        }
    }

    // MARK: - 写操作

    func accept(confirmCloseOwn: Bool = false) async {
        guard !isActing, let owner = session.identity else { return }
        isActing = true
        defer { isActing = false }
        do {
            let response = try await api.accept(challengeId: challengeId, confirmCloseOwnPending: confirmCloseOwn)
            guard isCurrentAction(owner) else { return }
            if let matchId = response.matchId, matchId > 0 {
                activity.markDirty()
                navigation = .match(matchId)
                return
            }
            activity.markDirty()
            feedback = "已接受，双方进入后正式开局"
            await load()
        } catch {
            guard isCurrentAction(owner) else { return }
            let apiError = error as? APIError
            let message = apiError?.message ?? ""
            let failure = apiError?.decodeBody(ChallengeFailureDetails.self)
            if failure?.needConfirmCloseOwn == true {
                prompt = .closeOwn
            } else if failure?.typeConflict == true {
                prompt = .typeConflict(message: message)
            } else {
                feedback = message.isEmpty ? "接受失败" : message
                await load()
            }
        }
    }

    func reject() async {
        guard !isActing, let owner = session.identity else { return }
        isActing = true
        defer { isActing = false }
        do {
            _ = try await api.reject(challengeId: challengeId)
            guard isCurrentAction(owner) else { return }
            activity.markDirty()
            feedback = "已拒绝"
            await load()
        } catch {
            guard isCurrentAction(owner) else { return }
            let message = (error as? APIError)?.message ?? ""
            feedback = message.isEmpty ? "拒绝失败" : message
            await load()
        }
    }

    func cancel() async {
        let api = self.api
        let challengeId = self.challengeId
        await handleAction({ try await api.cancel(challengeId: challengeId) }, successText: "约球已取消")
    }

    func abandon() async {
        let api = self.api
        let challengeId = self.challengeId
        await handleAction({ try await api.abandon(challengeId: challengeId) }, successText: "本次约球已结束")
    }

    func leave() async {
        let api = self.api
        let challengeId = self.challengeId
        await handleAction({ try await api.leave(challengeId: challengeId) }, successText: "已退出等待，约球保留")
    }

    private func handleAction(
        _ request: @escaping () async throws -> ChallengeActionResponse,
        successText: String
    ) async {
        guard !isActing, let owner = session.identity else { return }
        isActing = true
        defer { isActing = false }
        do {
            let response = try await request()
            guard isCurrentAction(owner) else { return }
            if let matchId = response.matchId, matchId > 0 {
                activity.markDirty()
                navigation = .match(matchId)
                return
            }
            activity.markDirty()
            feedback = successText
            await load()
        } catch {
            guard isCurrentAction(owner) else { return }
            let message = (error as? APIError)?.message ?? ""
            feedback = message.isEmpty ? "操作失败" : message
            await load()
        }
    }

    func enter(confirmEarly: Bool = false) async {
        guard !isActing, let owner = session.identity else { return }
        let opponentWaiting = isWaiting(opponentUserId)
        isActing = true
        defer { isActing = false }
        do {
            let response = try await api.enter(challengeId: challengeId, confirmEarly: confirmEarly)
            guard isCurrentAction(owner) else { return }
            switch response.action {
            case "self_ongoing":
                if let matchId = response.matchId, matchId > 0 {
                    prompt = .ongoingMatch(message: response.message ?? "请先处理当前比赛", matchId: matchId)
                }
            case "match_created":
                if let matchId = response.matchId, matchId > 0 {
                    activity.markDirty()
                    navigation = .match(matchId)
                }
            case "waiting":
                activity.markDirty()
                feedback = opponentWaiting ? "对方刚退出等待，你已成为等待方" : "已进入等待"
                await load()
            default:
                await load()
            }
        } catch {
            guard isCurrentAction(owner) else { return }
            let apiError = error as? APIError
            let message = apiError?.message ?? ""
            if apiError?.decodeBody(ChallengeFailureDetails.self)?.action == "confirm_early" {
                prompt = .confirmEarly(schedule: scheduleLabel)
            } else {
                feedback = message.isEmpty ? "进入失败" : message
                await load()
            }
        }
    }

    // MARK: - 私有

    private func isCurrent(_ token: Int, _ owner: SessionIdentity) -> Bool {
        token == requestToken && session.identity == owner
    }

    private func isCurrentAction(_ owner: SessionIdentity) -> Bool {
        isActive && session.identity == owner
    }
}
