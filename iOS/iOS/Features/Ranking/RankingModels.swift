import Combine
import Foundation

struct LeaderboardItem: Decodable, Equatable, Identifiable {
    let rank: Int
    let userId: Int64
    let nickname: String
    let avatar: String
    let rankLevel: Int
    let rankName: String
    let rankScore: Int
    let winRate: Int
    /// 旧服务端不返回会员等级。
    let vipLevel: Int?

    var id: Int64 { userId }
}

struct LeaderboardPage: Decodable, Equatable {
    let total: Int
    let topThree: [LeaderboardItem]
    let myRanking: LeaderboardItem?
    let list: [LeaderboardItem]
}

/// 领奖台位置：显示顺序为第二名、第一名、第三名。
struct PodiumSlot: Equatable, Identifiable {
    let placement: String
    let rank: Int
    let user: LeaderboardItem?

    var id: String { placement }

    /// 对应小程序 buildLeaderboardPodiumSlots：固定三个位置，缺位留空。
    static func build(topThree: [LeaderboardItem]) -> [PodiumSlot] {
        let placements = ["second", "first", "third"]
        let sourceIndex = ["second": 1, "first": 0, "third": 2]
        let normalized = Array(topThree.prefix(3))
        return placements.map { placement in
            let index = sourceIndex[placement] ?? 0
            return PodiumSlot(
                placement: placement,
                rank: index + 1,
                user: index < normalized.count ? normalized[index] : nil
            )
        }
    }
}

/// 排行榜读取与展示状态。
///
/// - 只采纳最新一次请求；切换球种、退出或切号后，在途结果一律丢弃。
/// - 前三名与「我的排名」只在第一页或刷新时更新，加载更多只追加列表。
/// - 刷新失败保留已有数据，只在顶部提示；加载更多失败不推进页码。
@MainActor
final class RankingModel: ObservableObject {
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

    @Published private(set) var phase: Phase = .idle
    @Published private(set) var topThree: [LeaderboardItem] = []
    @Published private(set) var list: [LeaderboardItem] = []
    @Published private(set) var myRanking: LeaderboardItem?
    @Published private(set) var total = 0
    @Published private(set) var currentGameType: Int
    @Published private(set) var isRefreshing = false
    @Published private(set) var isLoadingMore = false
    /// 已有数据时刷新 / 加载更多失败的提示；列表内容保持不变。
    @Published private(set) var refreshFailure: LoadFailure?

    /// 最近一次成功采纳的页码；0 表示还没有成功结果。
    private(set) var page = 0

    private let session: SessionStore
    private let api: RankingAPI
    private var owner: SessionIdentity?
    private var requestToken = 0

    init(session: SessionStore, gameType: Int = 3) {
        self.session = session
        self.api = RankingAPI(session: session)
        self.currentGameType = gameType
        self.owner = session.identity
    }

    var hasMore: Bool {
        topThree.count + list.count < total
    }

    /// 身份变化时清空并回到可重新读取的状态；在途请求因令牌失效而被丢弃。
    func syncIdentity() {
        let current = session.identity
        guard current != owner else { return }
        owner = current
        requestToken += 1
        topThree = []
        list = []
        myRanking = nil
        total = 0
        page = 0
        phase = .idle
        isRefreshing = false
        isLoadingMore = false
        refreshFailure = nil
    }

    func loadIfNeeded() async {
        syncIdentity()
        guard phase == .idle else { return }
        await fetch(.first)
    }

    func retry() async {
        syncIdentity()
        guard !isBusy else { return }
        await fetch(.refresh)
    }

    func refresh() async {
        syncIdentity()
        guard !isBusy else { return }
        await fetch(.refresh)
    }

    func loadMore() async {
        syncIdentity()
        guard phase == .loaded, hasMore, !isBusy else { return }
        await fetch(.more)
    }

    /// 切球种回到第一页；旧球种的在途结果由请求令牌丢弃。
    func selectGameType(_ gameType: Int) async {
        guard gameType != currentGameType else { return }
        currentGameType = gameType
        await fetch(.refresh)
    }

    private var isBusy: Bool {
        phase == .loading || isRefreshing || isLoadingMore
    }

    private func fetch(_ kind: FetchKind) async {
        requestToken += 1
        let token = requestToken
        let requestOwner = session.identity
        let requestedGameType = currentGameType
        let requestedPage = kind == .more ? page + 1 : 1

        switch kind {
        case .first:
            phase = .loading
            isRefreshing = false
            isLoadingMore = false
        case .refresh:
            if phase == .loaded {
                isRefreshing = true
            } else {
                phase = .loading
                isRefreshing = false
            }
            isLoadingMore = false
        case .more:
            isLoadingMore = true
        }
        refreshFailure = nil

        do {
            let result = try await api.leaderboard(gameType: requestedGameType, page: requestedPage)
            guard token == requestToken, session.identity == requestOwner else { return }

            if requestedPage == 1 {
                // 前三名与「我的排名」只在第一页或刷新时更新。
                topThree = result.topThree
                myRanking = result.myRanking
                list = result.list
            } else {
                list += result.list
            }
            total = max(result.total, 0)
            page = requestedPage
            isRefreshing = false
            isLoadingMore = false
            refreshFailure = nil
            phase = topThree.isEmpty && list.isEmpty ? .empty : .loaded
        } catch {
            guard token == requestToken, session.identity == requestOwner else { return }
            isRefreshing = false
            isLoadingMore = false
            // 页面离开导致的取消不是失败：没有内容时回到可重新加载的状态。
            if let apiError = error as? APIError, apiError.kind == .canceled {
                if requestedPage == 1, phase != .loaded {
                    phase = .idle
                }
                return
            }

            let failure = LoadFailure(error)
            if topThree.isEmpty && list.isEmpty {
                phase = .failed(failure)
            } else {
                phase = .loaded
                refreshFailure = failure
            }
        }
    }
}
