import Combine
import Foundation

/// 好友列表：分页由 `PagedHistoryList` 负责，删除/拉黑成功后先用本地 overlay 移除。
@MainActor
final class FriendListModel: ObservableObject {
    enum Phase: Equatable {
        case idle
        case loading
        case loaded
        case empty
        case failed(String)
    }

    let list: PagedHistoryList<FriendInfo>

    @Published private(set) var removedIDs: Set<Int64> = []

    private let session: SessionStore
    private let activity: ActivityStore
    private let api: FriendAPI
    private var cancellables: Set<AnyCancellable> = []

    init(session: SessionStore, activity: ActivityStore, api: FriendAPI? = nil) {
        self.session = session
        self.activity = activity
        let api = api ?? FriendAPI(session: session)
        self.api = api
        list = PagedHistoryList(session: session) { page in
            let response = try await api.list(page: page)
            return (response.list ?? [], Int(response.total))
        }
        list.objectWillChange
            .sink { [weak self] in self?.objectWillChange.send() }
            .store(in: &cancellables)
    }

    var items: [FriendInfo] {
        list.items.filter { !removedIDs.contains($0.friendUserId) }
    }

    var phase: Phase {
        if case .failed(let message) = list.phase { return .failed(message) }
        if list.phase == .loading { return .loading }
        if items.isEmpty, list.loadedPage > 0 { return .empty }
        if list.phase == .idle { return .idle }
        return .loaded
    }

    var failure: LoadFailure? {
        if case .failed(let message) = list.phase {
            return LoadFailure(category: .business, message: message)
        }
        return nil
    }

    var refreshError: String? { list.refreshError }
    var loadMoreError: String? { list.loadMoreError }
    var isLoadingMore: Bool { list.isLoadingMore }
    var pendingCount: Int { activity.pendingFriendRequestCount }

    /// 已移除的好友不再计入总数，与小程序同步递减 total 的判断一致。
    var hasMore: Bool {
        guard list.loadedPage > 0 else { return false }
        return items.count < max(0, Int(list.total) - removedIDs.count)
    }

    func onAppear() async {
        guard session.identity != nil else {
            list.syncIdentity()
            removedIDs = []
            return
        }
        await list.loadIfNeeded()
        await activity.refreshIfNeeded()
    }

    func identityChanged() async {
        list.syncIdentity()
        removedIDs = []
        await onAppear()
    }

    func retry() async {
        await list.retry()
        await activity.refreshIfNeeded()
    }

    func refresh() async {
        await list.refresh()
        await activity.refresh()
    }

    func loadMore() async {
        await list.loadMore()
    }

    func delete(_ friend: FriendInfo) async -> Bool {
        await remove(friend) { [api] userId in
            _ = try await api.delete(friendUserId: userId)
        }
    }

    func addToBlacklist(_ friend: FriendInfo) async -> Bool {
        await remove(friend) { [api] userId in
            _ = try await api.addToBlacklist(friendUserId: userId)
        }
    }

    private func remove(
        _ friend: FriendInfo,
        action: (Int64) async throws -> Void
    ) async -> Bool {
        let userId = friend.friendUserId
        guard userId > 0, let identity = session.identity else { return false }
        do {
            try await action(userId)
            guard session.identity == identity else { return false }
            removedIDs.insert(userId)
            activity.markDirty()
            return true
        } catch {
            return false
        }
    }
}
