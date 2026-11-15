import Combine
import Foundation

/// 消息 Tab 角标：无、红点、数字文本。
enum MessageBadge: Equatable {
    case none
    case dot
    case text(String)
}

/// 全应用共享的活动状态，按身份隔离。读取来源是聚合接口 GET /api/user/bootstrap。
@MainActor
final class ActivityStore: ObservableObject {
    static let freshness: TimeInterval = 30

    @Published private(set) var currentMatch: CurrentMatchInfo?
    @Published private(set) var currentChallenge: ChallengeInfo?
    @Published private(set) var receivedChallengeCount = 0
    @Published private(set) var challengeServerTime = ""
    /// 约球分区读不到时为 false：页面不得把未知当成「没有约球」。
    @Published private(set) var challengeAvailable = true
    @Published private(set) var unreadCount = 0
    @Published private(set) var pendingFriendRequestCount = 0
    @Published private(set) var latestSeasonRollover: NotificationInfo?
    /// 用户数据版本：实时通道通知「战绩 / 段位等已变化」时 +1，页面据此决定是否重读。
    @Published private(set) var userDataRevision = 0
    @Published private(set) var isLoaded = false
    /// 每次标脏 +1；消息中心用它判断通知列表是否需要回源。
    @Published private(set) var dirtySeq = 0

    private let session: SessionStore
    private let now: () -> Date
    private var owner: SessionIdentity?
    private var loadedAt: Date?
    private var isDirty = true
    private var requestToken = 0
    private var loadSequence = 0
    private var inFlight: Task<Void, Never>?

    init(session: SessionStore, now: @escaping () -> Date = Date.init) {
        self.session = session
        self.now = now
    }

    // MARK: - 身份

    /// 身份变化时清空；没有身份时保持清空状态。
    func syncIdentity() {
        let current = session.identity
        guard current != owner else { return }
        owner = current
        requestToken += 1
        inFlight = nil
        resetState()
    }

    // MARK: - 读取

    /// 未读过、已标脏或超过 30 秒时读取；同一时刻只有一个在途读取（并发调用共用它）。
    func refreshIfNeeded() async {
        syncIdentity()
        guard session.identity != nil else { return }
        let expired = loadedAt.map { now().timeIntervalSince($0) >= Self.freshness } ?? true
        guard !isLoaded || isDirty || expired else { return }
        await performLoad()
    }

    /// 强制读取。
    func refresh() async {
        syncIdentity()
        guard session.identity != nil else { return }
        await performLoad()
    }

    /// 标脏后的补读：有在途读取先等它结束，仍然是脏的再读一次。
    func refreshIfDirty() async {
        syncIdentity()
        guard isDirty else { return }
        if let inFlight {
            await inFlight.value
        }
        guard session.identity == owner, isDirty else { return }
        await performLoad()
    }

    private func performLoad() async {
        if let inFlight {
            await inFlight.value
            return
        }
        guard let identity = session.identity else { return }
        let startDirtySeq = dirtySeq
        let token = requestToken
        loadSequence += 1
        let sequence = loadSequence
        let task = Task { [weak self] in
            guard let self else { return }
            await self.load(identity: identity, token: token, startDirtySeq: startDirtySeq)
        }
        inFlight = task
        await task.value
        if loadSequence == sequence {
            inFlight = nil
        }
    }

    private func load(identity: SessionIdentity, token: Int, startDirtySeq: Int) async {
        do {
            let response: ActivityBootstrapResponse = try await session.authorizedRequest("/api/user/bootstrap")
            guard token == requestToken, session.identity == identity, owner == identity else { return }
            apply(response, startDirtySeq: startDirtySeq)
        } catch {
            // 整体失败（网络 / 5xx）：状态不变、保持脏，等下一次补读。
            guard token == requestToken, session.identity == identity else { return }
            isDirty = true
        }
    }

    /// 逐条对应小程序 `applyBootstrap`：不可用分区保留旧值，任何分区不可用都保持脏。
    private func apply(_ response: ActivityBootstrapResponse, startDirtySeq: Int) {
        let availability = response.availability ?? [:]
        func available(_ scope: String) -> Bool { availability[scope] != false }

        if available("current_match") {
            currentMatch = response.currentMatch
        }
        if available("current_challenge") {
            currentChallenge = response.currentChallenge
        }
        if available("challenge_received_count") {
            receivedChallengeCount = max(0, response.receivedPendingChallengeCount)
        }
        let challengeScopes = ["current_challenge", "challenge_received_count"]
        if challengeScopes.contains(where: { availability[$0] != nil }) {
            challengeAvailable = challengeScopes.allSatisfy { availability[$0] != false }
        }
        if !response.serverTime.isEmpty {
            challengeServerTime = response.serverTime
        }
        if available("unread_count") {
            setUnreadCount(response.unreadCount)
        }
        if available("pending_friend_request_count") {
            setPendingFriendRequestCount(response.pendingFriendRequestCount)
        }
        if available("latest_season_rollover") {
            latestSeasonRollover = response.latestSeasonRollover
        }

        let requiredScopes = [
            "current_match",
            "current_challenge",
            "challenge_received_count",
            "unread_count",
            "pending_friend_request_count",
            "latest_season_rollover",
        ]
        let partial = requiredScopes.contains { availability[$0] == false }
        isLoaded = true
        // 读取期间发生的新标脏不得被这份快照覆盖。
        isDirty = partial || dirtySeq > startDirtySeq
        loadedAt = now()
    }

    // MARK: - 标脏与计数

    /// 写操作成功或实时通道通知后调用：标脏并递增序号；读取期间发生的标脏不会被该次响应清掉。
    func markDirty() {
        guard session.identity != nil else { return }
        isDirty = true
        dirtySeq += 1
    }

    func setUnreadCount(_ count: Int) {
        unreadCount = max(0, count)
    }

    func decrementUnread() {
        if unreadCount > 0 { unreadCount -= 1 }
    }

    func setPendingFriendRequestCount(_ count: Int) {
        pendingFriendRequestCount = max(0, count)
    }

    func clearLatestSeasonRollover() {
        latestSeasonRollover = nil
    }

    func bumpUserDataRevision() {
        userDataRevision += 1
    }

    /// 消息 Tab 角标，对应 `resolveMessageBadge`。
    var messageBadge: MessageBadge {
        guard session.identity != nil else { return .none }
        let unread = max(0, unreadCount)
        if unread > 0 {
            return .text(unread > 99 ? "99+" : String(unread))
        }
        if max(0, pendingFriendRequestCount) + max(0, receivedChallengeCount) > 0 {
            return .dot
        }
        return .none
    }

    private func resetState() {
        currentMatch = nil
        currentChallenge = nil
        receivedChallengeCount = 0
        challengeServerTime = ""
        challengeAvailable = true
        unreadCount = 0
        pendingFriendRequestCount = 0
        latestSeasonRollover = nil
        userDataRevision = 0
        isLoaded = false
        dirtySeq = 0
        loadedAt = nil
        isDirty = true
    }
}
