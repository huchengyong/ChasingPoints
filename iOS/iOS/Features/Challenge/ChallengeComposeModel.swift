import Combine
import Foundation

/// 发起约球，对应小程序 `challengeCompose.vue`。时间基准始终来自服务端，失败时用北京时间兜底。
@MainActor
final class ChallengeComposeModel: ObservableObject {
    static let maxMessageLength = 200

    enum SubmitOutcome: Equatable {
        case success(challengeId: Int64)
        /// 已有待回应邀请；能拿到原约球 ID 时页面提供「查看并取消」。
        case pendingConflict(message: String, challengeId: Int64?)
        case failure(String)
        /// 身份或页面已变化，迟到的结果不弹提示、不导航。
        case discarded
    }

    @Published private(set) var opponent: ChallengeComposeTarget?
    @Published var gameType: Int
    @Published var matchMode: String
    @Published var message: String = "" {
        didSet {
            if message.count > Self.maxMessageLength {
                message = String(message.prefix(Self.maxMessageLength))
            }
            validationMessage = nil
        }
    }
    @Published private(set) var scheduledDate = ""
    @Published private(set) var startHour = 16
    @Published private(set) var endHour = 17
    @Published private(set) var serverTime = ""
    @Published private(set) var isSubmitting = false
    @Published private(set) var validationMessage: String?

    private let session: SessionStore
    private let activity: ActivityStore
    private let api: ChallengeAPI
    private let now: () -> Date
    private var requestToken = 0
    private var submitToken = 0
    private var slotInitialized = false

    init(
        session: SessionStore,
        activity: ActivityStore,
        target: ChallengeComposeTarget? = nil,
        api: ChallengeAPI? = nil,
        now: @escaping () -> Date = Date.init
    ) {
        self.session = session
        self.activity = activity
        self.api = api ?? ChallengeAPI(session: session)
        self.now = now
        opponent = target
        gameType = target?.gameType ?? 3
        matchMode = target?.matchMode == "practice" ? "practice" : "ranked"
    }

    var opponentName: String {
        let name = (opponent?.nickname ?? "").trimmingCharacters(in: .whitespacesAndNewlines)
        return name.isEmpty ? "球友" : name
    }

    var opponentAvatar: String { opponent?.avatar ?? "" }

    var dayOptions: [ChallengeRules.DayOption] {
        ChallengeRules.dayOptions(serverTime: serverTime)
    }

    var activeDayOffset: Int {
        max(ChallengeRules.dayOffsetOfScheduledDate(serverTime: serverTime, scheduledDate: scheduledDate) ?? 0, 0)
    }

    var startHourChoices: [Int] {
        ChallengeRules.availableStartHours(serverTime: serverTime, dayOffset: activeDayOffset)
    }

    var endHourChoices: [Int] {
        ChallengeRules.endHourOptions(startHour: startHour)
    }

    var matchModeHint: String {
        ChallengeRules.matchModeHint(matchMode)
    }

    var canSubmit: Bool {
        !isSubmitting && (opponent?.userId ?? 0) > 0
    }

    func onAppear() async {
        guard session.identity != nil else {
            resetForSignedOut()
            return
        }
        await loadServerTime()
    }

    func chooseDay(_ dayOffset: Int) {
        let date = ChallengeRules.computeScheduledDate(serverTime: serverTime, dayOffset: dayOffset)
        if !date.isEmpty { scheduledDate = date }
        let hours = ChallengeRules.availableStartHours(serverTime: serverTime, dayOffset: dayOffset)
        if let first = hours.first {
            startHour = first
            endHour = min(first + 1, 24)
        }
        validationMessage = nil
    }

    func setStartHour(_ hour: Int) {
        startHour = hour
        if endHour <= hour {
            endHour = min(hour + 1, 24)
        }
    }

    func setEndHour(_ hour: Int) {
        endHour = hour
    }

    func submit() async -> SubmitOutcome {
        guard !isSubmitting else { return .discarded }
        guard let opponent, opponent.userId > 0 else {
            validationMessage = "请先选择球友"
            return .failure("请先选择球友")
        }
        guard !scheduledDate.isEmpty else {
            validationMessage = "时间选择无效，请重试"
            return .failure("时间选择无效，请重试")
        }
        guard let identity = session.identity else { return .discarded }

        isSubmitting = true
        submitToken += 1
        let token = submitToken
        validationMessage = nil
        do {
            let response = try await api.send(
                SendChallengeBody(
                    toUserId: opponent.userId,
                    gameType: gameType,
                    matchMode: matchMode,
                    message: message,
                    scheduledDate: scheduledDate,
                    startHour: startHour,
                    endHour: endHour
                )
            )
            guard isCurrent(identity, token) else { return .discarded }
            isSubmitting = false
            guard response.success else {
                return .failure(response.message ?? "发送失败")
            }
            activity.markDirty()
            await activity.refreshIfDirty()
            return .success(challengeId: response.challengeId)
        } catch {
            guard isCurrent(identity, token) else { return .discarded }
            isSubmitting = false
            let apiMessage = (error as? APIError)?.message ?? ""
            if ChallengeRules.isPendingConflict(apiMessage) {
                // 响应只保留文案，改从共享活动状态定位自己仍待回应的那条约球。
                activity.markDirty()
                await activity.refreshIfDirty()
                guard isCurrent(identity, token) else { return .discarded }
                let current = activity.currentChallenge
                let pendingId = (current?.status == 0 && current?.fromUserId == identity.userId)
                    ? current?.id
                    : nil
                return .pendingConflict(message: apiMessage, challengeId: pendingId)
            }
            return .failure(apiMessage.isEmpty ? "发送失败" : apiMessage)
        }
    }

    // MARK: - 私有

    private func loadServerTime() async {
        requestToken += 1
        let token = requestToken
        let identity = session.identity
        do {
            let response = try await api.summary()
            guard token == requestToken, session.identity == identity else { return }
            if response.success, !response.serverTime.isEmpty {
                serverTime = response.serverTime
            }
        } catch {
            // 保留旧基准；提交时由服务端校验兜底。
        }
        guard token == requestToken, session.identity == identity else { return }
        if serverTime.isEmpty {
            serverTime = ChallengeRules.localServerTime(now: now())
        }
        guard !slotInitialized else { return }
        slotInitialized = true
        applyDefaultSlot()
    }

    private func applyDefaultSlot() {
        let slot = ChallengeRules.defaultSlot(serverTime: serverTime)
        startHour = slot.startHour
        endHour = slot.endHour
        if scheduledDate.isEmpty {
            scheduledDate = ChallengeRules.computeScheduledDate(serverTime: serverTime, dayOffset: slot.dayOffset)
        }
    }

    private func isCurrent(_ identity: SessionIdentity?, _ token: Int) -> Bool {
        token == submitToken && session.identity == identity
    }

    private func resetForSignedOut() {
        requestToken += 1
        submitToken += 1
        opponent = nil
        scheduledDate = ""
        serverTime = ""
        isSubmitting = false
        validationMessage = nil
        slotInitialized = false
    }
}
