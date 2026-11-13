import Combine
import Foundation

/// 观赛大厅列表状态：标签 / 球种筛选、分页、下拉刷新与“我”的标记。
///
/// - 只采纳最新一次请求；切标签、换球种、退出或切号都会作废在途结果。
/// - 刷新失败保留已有列表，只在顶部提示；加载更多失败不推进页码。
@MainActor
final class SpectatorLobbyModel: ObservableObject {
    enum Phase: Equatable {
        case idle
        case loading
        case loaded
        case empty
        case failed(LoadFailure)
    }

    private enum FetchKind {
        case first
        case refresh
        case more
    }

    @Published private(set) var tab: SpectatorLobbyTab
    @Published private(set) var gameType = 0
    @Published private(set) var isLoggedIn: Bool
    @Published private(set) var matches: [SpectatorMatch] = []
    @Published private(set) var phase: Phase = .idle
    @Published private(set) var isRefreshing = false
    @Published private(set) var isLoadingMore = false
    /// 已有数据时刷新失败的分类提示；列表内容保持不变。
    @Published private(set) var refreshFailure: LoadFailure?
    /// 需要以轻提示告知用户的加载更多失败。
    @Published private(set) var notice: String?

    private(set) var total = 0
    /// 最近一次成功采纳的页码；0 表示还没有成功结果。
    private(set) var page = 0

    private let session: SessionStore
    private let api: SpectatorLobbyAPI
    private var owner: SessionIdentity?
    private var viewerId: Int64
    private var requestToken = 0

    init(session: SessionStore, api: SpectatorLobbyAPI? = nil) {
        let identity = session.identity
        self.session = session
        self.api = api ?? SpectatorLobbyAPI(session: session)
        self.tab = identity == nil ? .live : .friends
        self.isLoggedIn = identity != nil
        self.viewerId = identity?.userId ?? 0
        self.owner = identity
    }

    var tabs: [SpectatorLobbyTab] { SpectatorLobbyRules.tabs(isLoggedIn: isLoggedIn) }

    var selectedGameTypeLabel: String {
        SpectatorLobbyRules.gameTypeOptions.first { $0.value == gameType }?.label ?? "全部球种"
    }

    var cards: [SpectatorMatchCard] {
        matches.map { SpectatorLobbyRules.normalize($0, userId: viewerId) }
    }

    var sections: [SpectatorLobbySection] {
        SpectatorLobbyRules.sections(cards, tabValue: tab.rawValue)
    }

    var emptyState: SpectatorLobbyEmptyState {
        SpectatorLobbyRules.emptyState(scope: tab.scope, status: tab.status, gameType: gameType)
    }

    var hasMore: Bool { page > 0 && page * SpectatorLobbyRules.pageSize < total }

    /// 身份变化时清空列表并回到访客可见的标签；在途请求因令牌失效而被丢弃。
    func syncIdentity() {
        let current = session.identity
        guard current != owner else { return }
        owner = current
        isLoggedIn = current != nil
        viewerId = current?.userId ?? 0
        requestToken += 1
        if !isLoggedIn, tab == .friends {
            tab = .live
        }
        clearContent()
    }

    func loadIfNeeded() async {
        syncIdentity()
        guard phase == .idle else { return }
        await fetch(.first)
    }

    func retry() async {
        syncIdentity()
        guard !isBusy else { return }
        await fetch(.first)
    }

    func refresh() async {
        syncIdentity()
        guard !isBusy else { return }
        await fetch(.refresh)
    }

    func loadMore() async {
        syncIdentity()
        guard hasMore, phase == .loaded, !isBusy else { return }
        await fetch(.more)
    }

    func selectTab(_ next: SpectatorLobbyTab) async {
        guard next != tab else { return }
        tab = next
        await reload()
    }

    func selectGameType(_ next: Int) async {
        guard next != gameType else { return }
        gameType = next
        await reload()
    }

    func clearNotice() {
        notice = nil
    }

    private var isBusy: Bool {
        phase == .loading || isRefreshing || isLoadingMore
    }

    /// 切换筛选先清空旧列表并作废在途请求，避免新条件下短暂显示旧数据。
    private func reload() async {
        requestToken += 1
        clearContent()
        await fetch(.first)
    }

    private func clearContent() {
        matches = []
        total = 0
        page = 0
        phase = .idle
        isRefreshing = false
        isLoadingMore = false
        refreshFailure = nil
    }

    private func fetch(_ kind: FetchKind) async {
        requestToken += 1
        let token = requestToken
        let requestOwner = session.identity
        let requestedPage = kind == .more ? page + 1 : 1

        switch kind {
        case .first:
            phase = .loading
            refreshFailure = nil
        case .refresh:
            isRefreshing = true
            refreshFailure = nil
        case .more:
            isLoadingMore = true
        }

        do {
            let result = try await api.matches(
                scope: tab.scope,
                status: tab.status,
                gameType: gameType,
                page: requestedPage
            )
            guard token == requestToken, session.identity == requestOwner else { return }

            switch kind {
            case .first, .refresh:
                matches = result.items
            case .more:
                // 翻页期间新开局会让后续页出现重复对局，按 id 去重。
                let knownIds = Set(matches.map(\.id))
                matches += result.items.filter { !knownIds.contains($0.id) }
            }
            total = max(result.total, 0)
            page = requestedPage
            isRefreshing = false
            isLoadingMore = false
            phase = matches.isEmpty ? .empty : .loaded
        } catch {
            guard token == requestToken, session.identity == requestOwner else { return }
            isRefreshing = false
            isLoadingMore = false
            if let apiError = error as? APIError, apiError.kind == .canceled {
                if kind == .first { phase = .idle }
                return
            }

            let failure = LoadFailure(error)
            switch kind {
            case .first:
                phase = .failed(failure)
            case .refresh:
                if matches.isEmpty {
                    phase = .failed(failure)
                } else {
                    refreshFailure = failure
                }
            case .more:
                notice = "加载失败，请上拉重试"
            }
        }
    }
}
