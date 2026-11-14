import Combine
import Foundation

/// 历届赛季：登录后按球种分页读取，切换球种与身份变化丢弃在途结果。
///
/// - 只有成功才推进分页游标；加载更多失败保留已读条目，重试继续同一页。
/// - 空页立即停止分页，避免数据收缩时无限加载。
@MainActor
final class SeasonHistoryModel: ObservableObject {
    enum Phase: Equatable {
        case idle
        case loading
        case ready
        case empty
        case failed(LoadFailure)
    }

    @Published private(set) var phase: Phase = .idle
    @Published private(set) var items: [SeasonHistoryRow] = []
    @Published private(set) var currentGameType: Int
    @Published private(set) var isLoadingMore = false
    @Published private(set) var isSwitchingGameType = false
    @Published private(set) var refreshFailure: LoadFailure?
    @Published private(set) var hasMore = false

    private let session: SessionStore
    private let api: SeasonAPI
    private var owner: SessionIdentity?
    private var requestToken = 0
    private var page = 1
    private var total: Int64 = 0
    private var lastPageCount = 0

    init(session: SessionStore, gameType: Int) {
        self.session = session
        self.api = SeasonAPI(session: session)
        self.currentGameType = GameTypeCatalog.isSupported(gameType) ? gameType : 3
        self.owner = session.identity
    }

    var currentGameTypeLabel: String { GameTypeCatalog.label(for: currentGameType) }

    // MARK: - 读取

    func loadIfNeeded() async {
        syncIdentity()
        guard session.identity != nil, phase == .idle else { return }
        await fetch(page: 1, append: false, switching: false)
    }

    func retry() async {
        syncIdentity()
        guard session.identity != nil, !isBusy else { return }
        await fetch(page: 1, append: false, switching: false)
    }

    /// 切球种只替换列表区域，赛季页保持一致的分页起点。
    func selectGameType(_ gameType: Int) async {
        guard gameType != currentGameType, GameTypeCatalog.isSupported(gameType) else { return }
        currentGameType = gameType
        resetPagination()
        isSwitchingGameType = true
        await fetch(page: 1, append: false, switching: true)
    }

    func loadMore() async {
        guard refreshFailure != nil || hasMore, !isBusy else { return }
        await fetch(page: page + 1, append: true, switching: false)
    }

    /// 身份变化时作废在途请求并清空本页状态；相同身份不做任何事。
    func syncIdentity() {
        let current = session.identity
        guard current != owner else { return }
        owner = current
        requestToken += 1
        isSwitchingGameType = false
        resetPagination()
        phase = .idle
    }

    // MARK: - 私有

    private var isBusy: Bool {
        phase == .loading || isLoadingMore || isSwitchingGameType
    }

    private func fetch(page requestedPage: Int, append: Bool, switching: Bool) async {
        guard let requestOwner = session.identity else { return }
        requestToken += 1
        let token = requestToken
        let requestedGameType = currentGameType
        if append {
            isLoadingMore = true
        } else if !switching {
            phase = .loading
        }
        refreshFailure = nil

        do {
            let response = try await api.history(gameType: requestedGameType, page: requestedPage)
            guard token == requestToken,
                  session.identity == requestOwner,
                  currentGameType == requestedGameType else { return }
            let rows = response.list.compactMap { SeasonHistoryRules.row($0, gameType: requestedGameType) }
            items = append ? items + rows : rows
            total = response.total
            page = requestedPage
            lastPageCount = rows.count
            hasMore = SeasonHistoryRules.hasMore(
                loaded: items.count,
                total: total,
                lastPageCount: lastPageCount
            )
            isLoadingMore = false
            isSwitchingGameType = false
            phase = items.isEmpty ? .empty : .ready
        } catch {
            guard token == requestToken,
                  session.identity == requestOwner,
                  currentGameType == requestedGameType else { return }
            isLoadingMore = false
            isSwitchingGameType = false
            if isCancellation(error) {
                if phase == .loading { phase = .idle }
                return
            }
            let failure = LoadFailure(error)
            // 全局会话失效流程已接管重新登录，页面不重复展示失败态。
            guard failure.category != .session else { return }
            if append {
                // 已读条目保留，重试仍请求同一页。
                refreshFailure = failure
            } else {
                items = []
                hasMore = false
                phase = .failed(failure)
            }
        }
    }

    private func resetPagination() {
        items = []
        total = 0
        page = 1
        lastPageCount = 0
        hasMore = false
        refreshFailure = nil
        isLoadingMore = false
        phase = .idle
    }

    private func isCancellation(_ error: Error) -> Bool {
        if let apiError = error as? APIError, apiError.kind == .canceled { return true }
        return error is CancellationError
    }
}
