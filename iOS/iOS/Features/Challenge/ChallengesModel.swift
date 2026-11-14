import Combine
import Foundation

/// 约球列表，对应小程序 `subPages/social/challenges.vue`：收到 / 发出 / 约球记录三个分栏。
@MainActor
final class ChallengesModel: ObservableObject {
    enum Phase: Equatable {
        case loading
        case loaded
        case empty
        case failed(LoadFailure)
    }

    enum Prompt: Identifiable, Equatable {
        case cancel(ChallengeInfo)
        case closeOwn(ChallengeInfo)
        case typeConflict(message: String)

        var id: String {
            switch self {
            case .cancel(let item): "cancel-\(String(item.id))"
            case .closeOwn(let item): "close-own-\(String(item.id))"
            case .typeConflict: "type-conflict"
            }
        }
    }

    enum Navigation: Equatable {
        case waiting(Int64)
        case match(Int64)
        case compose(ChallengeComposeTarget)
    }

    @Published private(set) var tab: ChallengeTab
    @Published private(set) var activeItems: [ChallengeInfo] = []
    @Published private(set) var historyItems: [ChallengeInfo] = []
    @Published private(set) var nextBeforeId: Int64 = 0
    @Published private(set) var serverTime = ""
    @Published private(set) var activeLoading = true
    @Published private(set) var historyLoading = true
    @Published private(set) var isLoadingMoreHistory = false
    @Published private(set) var isActing = false
    @Published private(set) var activeFailure: LoadFailure?
    @Published private(set) var historyFailure: LoadFailure?
    @Published private(set) var activeRefreshError: String?
    @Published private(set) var historyRefreshError: String?
    @Published private(set) var historyLoadMoreError: String?
    @Published var prompt: Prompt?
    @Published var feedback: String?
    @Published var navigation: Navigation?

    private let session: SessionStore
    private let activity: ActivityStore
    private let api: ChallengeAPI
    private var owner: SessionIdentity?
    private var activeLoaded = false
    private var historyLoaded = false
    private var historyDirty = true
    private var historyDirtySeq = 0
    private var activeToken = 0
    private var historyToken = 0

    init(session: SessionStore, activity: ActivityStore, initialTab: ChallengeTab, api: ChallengeAPI? = nil) {
        self.session = session
        self.activity = activity
        self.tab = initialTab
        self.api = api ?? ChallengeAPI(session: session)
        self.owner = session.identity
    }

    // MARK: - 展示

    var myUserId: Int64 { session.identity?.userId ?? 0 }

    var receivedCount: Int {
        activeItems.filter { ChallengeRules.isMineReceivedPending($0, userId: myUserId) }.count
    }

    var items: [ChallengeInfo] {
        switch tab {
        case .received:
            return activeItems.filter { ChallengeRules.isMineReceivedPending($0, userId: myUserId) }
        case .sent:
            return activeItems.filter {
                ChallengeRules.isMineSentPending($0, userId: myUserId) || $0.status == 1 || $0.status == 6
            }
        case .history:
            return historyItems
        }
    }

    var phase: Phase {
        if tab == .history {
            if historyLoading && !historyLoaded { return .loading }
            if let historyFailure, historyItems.isEmpty { return .failed(historyFailure) }
            return historyItems.isEmpty ? .empty : .loaded
        }
        if activeLoading && !activeLoaded { return .loading }
        if let activeFailure, items.isEmpty { return .failed(activeFailure) }
        return items.isEmpty ? .empty : .loaded
    }

    /// 当前分栏的刷新失败提示；已有数据时保留列表。
    var refreshError: String? {
        tab == .history ? historyRefreshError : activeRefreshError
    }

    var emptyText: String { ChallengeRules.emptyText(tab: tab) }

    var hasMoreHistory: Bool { nextBeforeId > 0 }

    func opponent(_ item: ChallengeInfo) -> ChallengeListItemView {
        ChallengeRules.opponent(item, userId: myUserId)
    }

    func statusLabel(_ item: ChallengeInfo) -> String {
        ChallengeRules.statusLabel(item, userId: myUserId)
    }

    // MARK: - 读取

    func onAppear() async {
        syncIdentity()
        await reload()
    }

    func retry() async {
        syncIdentity()
        await reload()
    }

    func switchTab(_ value: ChallengeTab) async {
        tab = value
        syncIdentity()
        if value == .history {
            if !historyLoaded || historyDirty {
                await loadHistory()
            }
        } else if !activeLoaded {
            await loadActive()
        }
    }

    func loadMoreHistory() async {
        guard !isLoadingMoreHistory, nextBeforeId > 0, let owner = session.identity else { return }
        let cursor = nextBeforeId
        isLoadingMoreHistory = true
        defer { isLoadingMoreHistory = false }
        do {
            let response = try await api.history(beforeId: cursor)
            guard session.identity == owner, nextBeforeId == cursor, tab == .history else { return }
            historyItems.append(contentsOf: response.list)
            nextBeforeId = response.nextBeforeId ?? 0
            historyLoadMoreError = nil
        } catch {
            guard session.identity == owner else { return }
            historyLoadMoreError = errorMessage(error)
        }
    }

    private func reload() async {
        await loadActive()
        if historyLoaded || tab == .history {
            await loadHistory()
        }
    }

    private func loadActive() async {
        guard let owner = session.identity else {
            activeLoading = false
            activeLoaded = false
            return
        }
        activeToken += 1
        let token = activeToken
        do {
            if !activeLoaded { activeLoading = true }
            let pending = try await api.pending()
            let summary = try? await api.summary()
            guard token == activeToken, session.identity == owner else { return }
            activeItems = pending.list
            if let summary, summary.success, !summary.serverTime.isEmpty {
                serverTime = summary.serverTime
            }
            activeLoaded = true
            activeFailure = nil
            activeRefreshError = nil
            activeLoading = false
        } catch {
            guard token == activeToken, session.identity == owner else { return }
            activeLoading = false
            if activeLoaded {
                activeRefreshError = errorMessage(error)
            } else {
                activeFailure = LoadFailure(error)
            }
        }
    }

    private func loadHistory() async {
        guard let owner = session.identity else {
            historyLoading = false
            historyLoaded = false
            return
        }
        historyToken += 1
        let token = historyToken
        let startDirtySeq = historyDirtySeq
        if !historyLoaded { historyLoading = true }
        var shouldReload = false
        do {
            let response = try await api.history()
            guard token == historyToken, session.identity == owner else { return }
            historyItems = response.list
            nextBeforeId = response.nextBeforeId ?? 0
            historyLoaded = true
            historyFailure = nil
            historyRefreshError = nil
            historyLoadMoreError = nil
            // 读取期间发生的新写入不得被这份响应覆盖。
            historyDirty = historyDirtySeq > startDirtySeq
        } catch {
            guard token == historyToken, session.identity == owner else { return }
            if historyLoaded {
                historyRefreshError = errorMessage(error)
            } else {
                historyFailure = LoadFailure(error)
            }
        }
        historyLoading = false
        // 读取期间发生过新写入时补读一次，失败则保留 dirty 交给重试入口。
        if historyLoaded, historyDirty, historyDirtySeq > startDirtySeq, tab == .history {
            shouldReload = true
        }
        if shouldReload {
            await loadHistory()
        }
    }

    // MARK: - 写操作

    func accept(_ item: ChallengeInfo, confirmCloseOwn: Bool = false) async {
        guard !isActing, let owner = session.identity else { return }
        isActing = true
        defer { isActing = false }
        do {
            let response = try await api.accept(challengeId: item.id, confirmCloseOwnPending: confirmCloseOwn)
            guard session.identity == owner else { return }
            markActivityDirty()
            if let matchId = response.matchId, matchId > 0 {
                navigation = .match(matchId)
                return
            }
            feedback = confirmCloseOwn ? "已接受" : "已接受，双方进入后正式开局"
            navigation = .waiting(response.challengeId > 0 ? response.challengeId : item.id)
            await reload()
        } catch {
            guard session.identity == owner else { return }
            let message = (error as? APIError)?.message ?? ""
            if ChallengeRules.isNeedConfirmCloseOwn(message) {
                prompt = .closeOwn(item)
            } else if ChallengeRules.isTypeConflict(message) {
                prompt = .typeConflict(message: message)
            } else {
                feedback = message.isEmpty ? "接受失败" : message
            }
        }
    }

    func reject(_ item: ChallengeInfo) async {
        guard !isActing, let owner = session.identity else { return }
        isActing = true
        defer { isActing = false }
        do {
            _ = try await api.reject(challengeId: item.id)
            guard session.identity == owner else { return }
            markActivityDirty()
            feedback = "已拒绝"
            await reload()
        } catch {
            guard session.identity == owner else { return }
            let message = (error as? APIError)?.message ?? ""
            feedback = message.isEmpty ? "操作失败" : message
        }
    }

    func cancel(_ item: ChallengeInfo) async {
        let api = self.api
        await handleAction(item, request: { try await api.cancel(challengeId: item.id) }, successText: "约球已取消")
    }

    private func handleAction(
        _ item: ChallengeInfo,
        request: @escaping () async throws -> ChallengeActionResponse,
        successText: String
    ) async {
        guard !isActing, let owner = session.identity else { return }
        isActing = true
        defer { isActing = false }
        do {
            let response = try await request()
            guard session.identity == owner else { return }
            markActivityDirty()
            if let matchId = response.matchId, matchId > 0 {
                navigation = .match(matchId)
                return
            }
            feedback = successText
            await reload()
        } catch {
            guard session.identity == owner else { return }
            let message = (error as? APIError)?.message ?? ""
            feedback = message.isEmpty ? "操作失败" : message
        }
    }

    // MARK: - 导航

    func openWaiting(_ item: ChallengeInfo) {
        navigation = .waiting(item.id)
    }

    func openMatch(_ item: ChallengeInfo) {
        guard let matchId = item.matchId, matchId > 0 else { return }
        navigation = .match(matchId)
    }

    func openDetail(_ item: ChallengeInfo) {
        if item.status == 1 || item.status == 6 {
            openWaiting(item)
        }
    }

    func rematch(_ item: ChallengeInfo) {
        let opponent = opponent(item)
        navigation = .compose(ChallengeComposeTarget(
            userId: opponent.opponentId,
            nickname: opponent.nickname,
            avatar: opponent.avatar,
            gameType: item.gameType,
            matchMode: item.matchMode == "practice" ? "practice" : "ranked"
        ))
    }

    // MARK: - 私有

    private func markActivityDirty() {
        activity.markDirty()
        historyDirty = true
        historyDirtySeq += 1
    }

    private func errorMessage(_ error: Error) -> String {
        let message = (error as? APIError)?.message ?? ""
        return message.isEmpty ? "加载失败" : message
    }

    private func syncIdentity() {
        let current = session.identity
        guard current != owner else { return }
        owner = current
        activeToken += 1
        historyToken += 1
        activeItems = []
        historyItems = []
        nextBeforeId = 0
        serverTime = ""
        activeLoaded = false
        historyLoaded = false
        activeLoading = true
        historyLoading = true
        isLoadingMoreHistory = false
        activeFailure = nil
        historyFailure = nil
        activeRefreshError = nil
        historyRefreshError = nil
        historyLoadMoreError = nil
        historyDirty = true
        historyDirtySeq = 0
    }
}
