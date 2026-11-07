import Combine
import Foundation

/// 单个分栏的分页读取状态。
///
/// - 首次显示加载第一页；已采纳结果后，切换或返回不重复请求。
/// - 只采纳最新一次请求的结果；身份变化后，在途结果一律丢弃。
/// - 页码只在成功后推进，加载更多失败不会跳过页。
@MainActor
final class PagedHistoryList<Item: Identifiable>: ObservableObject {
    enum Phase: Equatable {
        case idle
        case loading
        case loaded
        case empty
        case failed(String)
    }

    typealias PageLoader = (_ page: Int) async throws -> (items: [Item], total: Int)

    private enum Kind {
        case first
        case refresh
        case more
    }

    @Published private(set) var items: [Item] = []
    @Published private(set) var phase: Phase = .idle
    @Published private(set) var isRefreshing = false
    @Published private(set) var isLoadingMore = false
    /// 下拉刷新失败但已有数据时的提示；列表内容保持不变。
    @Published private(set) var refreshError: String?
    @Published private(set) var loadMoreError: String?
    private(set) var total = 0
    /// 最近一次成功采纳的页码；0 表示还没有成功结果。
    private(set) var loadedPage = 0

    private let session: SessionStore
    private let loadPage: PageLoader
    private var owner: SessionIdentity?
    private var requestToken = 0

    init(session: SessionStore, loadPage: @escaping PageLoader) {
        self.session = session
        self.loadPage = loadPage
    }

    var hasMore: Bool {
        loadedPage > 0 && items.count < total
    }

    /// 身份变化时清空全部状态；在途请求因令牌失效而被丢弃。
    func syncIdentity() {
        let current = session.identity
        guard current != owner else { return }
        owner = current
        requestToken += 1
        items = []
        total = 0
        loadedPage = 0
        phase = .idle
        isRefreshing = false
        isLoadingMore = false
        refreshError = nil
        loadMoreError = nil
    }

    func loadIfNeeded() async {
        syncIdentity()
        guard owner != nil, loadedPage == 0, phase != .loading else { return }
        await fetch(.first)
    }

    func retry() async {
        syncIdentity()
        guard owner != nil else { return }
        if loadedPage == 0 {
            await fetch(.first)
        } else {
            await refresh()
        }
    }

    func refresh() async {
        syncIdentity()
        guard owner != nil, !isRefreshing else { return }
        await fetch(.refresh)
    }

    func loadMore() async {
        syncIdentity()
        guard owner != nil, hasMore, !isLoadingMore, !isRefreshing else { return }
        await fetch(.more)
    }

    private func fetch(_ kind: Kind) async {
        let page = kind == .more ? loadedPage + 1 : 1
        requestToken += 1
        let token = requestToken
        let requestOwner = owner

        switch kind {
        case .first:
            phase = .loading
        case .refresh:
            isRefreshing = true
            loadMoreError = nil
        case .more:
            isLoadingMore = true
            loadMoreError = nil
        }

        do {
            let result = try await loadPage(page)
            guard token == requestToken, session.identity == requestOwner else { return }
            switch kind {
            case .first, .refresh:
                items = result.items
                loadedPage = 1
            case .more:
                items.append(contentsOf: result.items)
                loadedPage = page
            }
            total = result.total
            refreshError = nil
            isRefreshing = false
            isLoadingMore = false
            phase = items.isEmpty ? .empty : .loaded
        } catch {
            guard token == requestToken, session.identity == requestOwner else { return }
            isRefreshing = false
            isLoadingMore = false
            // 页面离开导致的取消不是失败：首页回到可重新加载的状态，下次出现时再请求。
            if let apiError = error as? APIError, apiError.kind == .canceled {
                if kind == .first { phase = .idle }
                return
            }
            let message = error.localizedDescription
            switch kind {
            case .first:
                phase = .failed(message)
            case .refresh:
                if items.isEmpty {
                    phase = .failed(message)
                } else {
                    refreshError = message
                }
            case .more:
                loadMoreError = message
            }
        }
    }
}
