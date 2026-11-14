import Combine
import Foundation

/// 黑名单列表：移出成功后本地移除，空态由剩余数据推导。
@MainActor
final class BlacklistModel: ObservableObject {
    enum Phase: Equatable {
        case idle
        case loading
        case loaded
        case empty
        case failed(String)
    }

    let list: PagedHistoryList<FriendBlacklistItem>

    @Published private(set) var removedIDs: Set<Int64> = []
    @Published private(set) var busyIDs: Set<Int64> = []

    private let session: SessionStore
    private let api: FriendAPI
    private var cancellables: Set<AnyCancellable> = []

    init(session: SessionStore, api: FriendAPI? = nil) {
        self.session = session
        let api = api ?? FriendAPI(session: session)
        self.api = api
        list = PagedHistoryList(session: session) { page in
            let response = try await api.blacklist(page: page)
            return (response.list ?? [], Int(response.total))
        }
        list.objectWillChange
            .sink { [weak self] in self?.objectWillChange.send() }
            .store(in: &cancellables)
    }

    var items: [FriendBlacklistItem] {
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
    var isLoadingMore: Bool { list.isLoadingMore }
    var hasMore: Bool {
        guard list.loadedPage > 0 else { return false }
        return items.count < max(0, Int(list.total) - removedIDs.count)
    }

    func isBusy(_ item: FriendBlacklistItem) -> Bool {
        busyIDs.contains(item.friendUserId)
    }

    func onAppear() async {
        guard session.identity != nil else {
            list.syncIdentity()
            removedIDs = []
            busyIDs = []
            return
        }
        await list.loadIfNeeded()
    }

    func identityChanged() async {
        list.syncIdentity()
        removedIDs = []
        busyIDs = []
        await onAppear()
    }

    func retry() async {
        await list.retry()
    }

    func refresh() async {
        await list.refresh()
    }

    func loadMore() async {
        await list.loadMore()
    }

    func remove(_ item: FriendBlacklistItem) async -> FriendActionOutcome {
        let userId = item.friendUserId
        guard userId > 0, !busyIDs.contains(userId), let identity = session.identity else {
            return .failure("操作失败")
        }
        busyIDs.insert(userId)
        defer { busyIDs.remove(userId) }
        do {
            _ = try await api.removeFromBlacklist(friendUserId: userId)
            guard session.identity == identity else { return .failure("操作失败") }
            removedIDs.insert(userId)
            return .success
        } catch {
            return .failure(error.localizedDescription)
        }
    }
}
