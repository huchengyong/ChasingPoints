import Combine
import Foundation

/// 赛讯 Tab 的列表状态，逐条对应 `pages/tournament/index.vue`。
///
/// - 访客可读，因此不使用要求登录的 `PagedHistoryList`。
/// - 只采纳最新一次请求：切换筛选、退出或切号后，在途结果一律丢弃。
/// - 筛选变化时页码回到 1 并清空列表；加载更多失败不推进页码。
@MainActor
final class SaiXunListModel: ObservableObject {
    enum Phase: Equatable {
        case idle
        case loading
        case loaded
        case empty
        case failed(LoadFailure)
    }

    /// 与页面常驻时的重读新鲜度一致。
    static let freshness: TimeInterval = 60
    static let pageSize = 10

    struct StatusTab: Equatable, Identifiable {
        let label: String
        let value: Int

        var id: Int { value }
    }

    struct GameTypeOption: Equatable, Identifiable {
        let label: String
        let value: Int

        var id: Int { value }
    }

    static let statusTabs: [StatusTab] = [
        StatusTab(label: "全部", value: -1),
        StatusTab(label: "进行中", value: 1),
        StatusTab(label: "即将开始", value: 0),
        StatusTab(label: "已结束", value: 2),
    ]

    static let gameTypeOptions: [GameTypeOption] = [
        GameTypeOption(label: "全部球种", value: 0),
        GameTypeOption(label: "斯诺克", value: 1),
        GameTypeOption(label: "中式八球", value: 3),
        GameTypeOption(label: "中式九球", value: 2),
    ]

    @Published private(set) var filter: EventNewsFilter
    @Published private(set) var items: [EventNewsInfo] = []
    @Published private(set) var phase: Phase = .idle
    @Published private(set) var total = 0
    @Published private(set) var hasMore = false
    @Published private(set) var isRefreshing = false
    @Published private(set) var isLoadingMore = false
    /// 已有数据时刷新失败的提示；列表内容保持不变。
    @Published private(set) var refreshError: String?

    private enum FetchKind {
        case first
        case refresh
        case more
    }

    private let session: SessionStore
    private let api: EventNewsAPI
    private let now: () -> Date
    private var page = 0
    private var owner: SessionIdentity?
    private var requestToken = 0
    private var lastLoadedAt: Date?

    init(session: SessionStore, now: @escaping () -> Date = Date.init) {
        let current = SaiXunFilter.buildCurrentYearFilter(now: now())
        self.session = session
        self.api = EventNewsAPI(session: session)
        self.now = now
        self.filter = EventNewsFilter(
            year: current.year,
            from: current.from,
            to: current.to,
            gameType: 0,
            status: -1
        )
        self.owner = session.identity
    }

    var sections: [SaiXunListSection] {
        SaiXunRules.buildSaiXunListSections(items)
    }

    func card(for item: EventNewsInfo) -> SaiXunCard {
        SaiXunRules.normalizeSaiXunCard(item, now: now())
    }

    var yearOptions: [Int] {
        SaiXunFilter.buildYearOptions(filter.year, now: now())
    }

    var selectedGameTypeLabel: String {
        Self.gameTypeOptions.first { $0.value == filter.gameType }?.label ?? Self.gameTypeOptions[0].label
    }

    var hasNarrowFilter: Bool {
        filter.status != -1 || filter.gameType != 0
    }

    var emptyTitle: String {
        switch filter.status {
        case 1: "当前没有进行中的赛事"
        case 0: "暂无即将开始的赛事"
        case 2: "暂无已结束的赛事"
        default: "这一年还没有赛讯"
        }
    }

    var emptyDescription: String {
        switch filter.status {
        case 1: "可以先看看即将开始的赛事。"
        case 0: "新赛程公布后会第一时间更新。"
        case 2: "换个年份或球种看看。"
        default: "换个年份看看，或稍后下拉刷新。"
        }
    }

    /// 身份变化时清空列表；在途请求因令牌失效而被丢弃。
    func syncIdentity() {
        let current = session.identity
        guard current != owner else { return }
        owner = current
        requestToken += 1
        resetList()
        phase = .idle
        isRefreshing = false
        isLoadingMore = false
        refreshError = nil
    }

    /// 页面出现时读取第一页：没有数据就加载，超过新鲜度则静默重读。
    func loadIfNeeded(maxAge: TimeInterval = SaiXunListModel.freshness) async {
        syncIdentity()
        switch phase {
        case .idle:
            await fetch(.first)
        case .loaded, .empty:
            guard isExpired(maxAge: maxAge) else { return }
            await fetch(.refresh)
        case .loading, .failed:
            return
        }
    }

    func refresh() async {
        syncIdentity()
        guard !isBusy else { return }
        await fetch(.refresh)
    }

    func retry() async {
        syncIdentity()
        guard !isBusy else { return }
        await fetch(page > 0 ? .refresh : .first)
    }

    func loadMore() async {
        syncIdentity()
        guard hasMore, !isBusy, page > 0 else { return }
        await fetch(.more)
    }

    func selectStatus(_ status: Int) async {
        guard status != filter.status else { return }
        await applyFilter { $0.status = status }
    }

    func selectYear(_ year: Int) async {
        guard year != filter.year else { return }
        let range = SaiXunFilter.buildYearDateRange(year)
        await applyFilter {
            $0.year = year
            $0.from = range.from
            $0.to = range.to
        }
    }

    func selectGameType(_ value: Int) async {
        guard value != filter.gameType else { return }
        await applyFilter { $0.gameType = value }
    }

    func resetNarrowFilter() async {
        guard hasNarrowFilter else { return }
        await applyFilter {
            $0.status = -1
            $0.gameType = 0
        }
    }

    /// 身份变化后由页面触发：清空并按当前筛选重新读取。
    func reloadForIdentityChange() async {
        syncIdentity()
        await fetch(.first)
    }

    private var isBusy: Bool {
        phase == .loading || isRefreshing || isLoadingMore
    }

    private func isExpired(maxAge: TimeInterval) -> Bool {
        guard let lastLoadedAt else { return false }
        return now().timeIntervalSince(lastLoadedAt) > maxAge
    }

    private func applyFilter(_ update: (inout EventNewsFilter) -> Void) async {
        update(&filter)
        // 先作废旧请求，避免旧筛选的迟到结果覆盖新列表。
        requestToken += 1
        resetList()
        phase = .idle
        isRefreshing = false
        isLoadingMore = false
        refreshError = nil
        await fetch(.first)
    }

    private func resetList() {
        items = []
        total = 0
        hasMore = false
        page = 0
        lastLoadedAt = nil
    }

    private func fetch(_ kind: FetchKind) async {
        requestToken += 1
        let token = requestToken
        let requestOwner = session.identity
        let requestedPage = kind == .more ? page + 1 : 1

        switch kind {
        case .first:
            phase = .loading
        case .refresh:
            if items.isEmpty {
                phase = .loading
            } else {
                isRefreshing = true
            }
        case .more:
            isLoadingMore = true
        }
        refreshError = nil

        do {
            let result = try await api.list(filter: filter, page: requestedPage, pageSize: Self.pageSize)
            guard token == requestToken, session.identity == requestOwner else { return }

            switch kind {
            case .more:
                items += result.items
            case .first, .refresh:
                items = result.items
            }
            page = requestedPage
            total = result.total
            hasMore = items.count < total
            phase = items.isEmpty ? .empty : .loaded
            lastLoadedAt = now()
            isRefreshing = false
            isLoadingMore = false
        } catch {
            guard token == requestToken, session.identity == requestOwner else { return }
            isRefreshing = false
            isLoadingMore = false
            // 页面离开导致的取消不是失败：列表为空时回到可重新加载的状态。
            if let apiError = error as? APIError, apiError.kind == .canceled {
                if kind != .more, items.isEmpty { phase = .idle }
                return
            }
            switch kind {
            case .first:
                phase = .failed(LoadFailure(error))
            case .refresh:
                if items.isEmpty {
                    phase = .failed(LoadFailure(error))
                } else {
                    refreshError = error.localizedDescription
                }
            case .more:
                break
            }
        }
    }
}
