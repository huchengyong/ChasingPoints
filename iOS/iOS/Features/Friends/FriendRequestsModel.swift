import Combine
import Foundation

enum FriendActionOutcome: Equatable {
    case success
    case failure(String)
}

/// 好友申请：只展示待处理申请；接受/拒绝用状态覆盖，拉黑后本地移除。
@MainActor
final class FriendRequestsModel: ObservableObject {
    enum Phase: Equatable {
        case idle
        case loading
        case loaded
        case empty
        case failed(String)
    }

    let list: PagedHistoryList<FriendRequestInfo>

    @Published private(set) var statusOverrides: [Int64: Int] = [:]
    @Published private(set) var removedIDs: Set<Int64> = []
    @Published private(set) var busyIDs: Set<Int64> = []

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
            let response = try await api.requests(page: page)
            return (response.list ?? [], Int(response.total))
        }
        list.objectWillChange
            .sink { [weak self] in self?.objectWillChange.send() }
            .store(in: &cancellables)
    }

    var items: [FriendRequestInfo] {
        list.items.filter { !removedIDs.contains($0.id) }
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
    var hasMore: Bool {
        guard list.loadedPage > 0 else { return false }
        return items.count < max(0, Int(list.total) - removedIDs.count)
    }

    func status(of request: FriendRequestInfo) -> Int {
        statusOverrides[request.id] ?? request.status
    }

    func isBusy(_ request: FriendRequestInfo) -> Bool {
        busyIDs.contains(request.id)
    }

    func onAppear() async {
        guard session.identity != nil else {
            list.syncIdentity()
            resetOverlay()
            return
        }
        await list.loadIfNeeded()
        syncPendingCount()
    }

    func identityChanged() async {
        list.syncIdentity()
        resetOverlay()
        await onAppear()
    }

    func retry() async {
        await list.retry()
        syncPendingCount()
    }

    func refresh() async {
        await list.refresh()
        syncPendingCount()
    }

    func loadMore() async {
        await list.loadMore()
    }

    func accept(_ request: FriendRequestInfo) async -> FriendActionOutcome {
        let outcome = await perform(request) { [api] in
            _ = try await api.accept(requestId: request.id)
        }
        guard outcome == .success else { return outcome }
        statusOverrides[request.id] = 1
        decrementPendingCount()
        activity.markDirty()
        return .success
    }

    func reject(_ request: FriendRequestInfo) async -> FriendActionOutcome {
        let outcome = await perform(request) { [api] in
            _ = try await api.reject(requestId: request.id)
        }
        guard outcome == .success else { return outcome }
        statusOverrides[request.id] = 2
        decrementPendingCount()
        activity.markDirty()
        return .success
    }

    func rejectAndBlacklist(_ request: FriendRequestInfo) async -> FriendActionOutcome {
        guard request.applicantUserId > 0 else { return .failure("申请人信息异常") }
        let outcome = await perform(request) { [api] in
            _ = try await api.addToBlacklist(friendUserId: request.applicantUserId)
        }
        guard outcome == .success else { return outcome }
        removedIDs.insert(request.id)
        decrementPendingCount()
        activity.markDirty()
        return .success
    }

    private func perform(
        _ request: FriendRequestInfo,
        action: () async throws -> Void
    ) async -> FriendActionOutcome {
        let id = request.id
        guard !busyIDs.contains(id), let identity = session.identity else {
            return .failure("操作失败")
        }
        busyIDs.insert(id)
        defer { busyIDs.remove(id) }
        do {
            try await action()
            guard session.identity == identity else { return .failure("操作失败") }
            return .success
        } catch {
            return .failure(error.localizedDescription)
        }
    }

    /// 首屏/刷新成功后把待处理总数同步给消息 Tab 的角标。
    private func syncPendingCount() {
        guard list.loadedPage > 0 else { return }
        if case .failed = list.phase { return }
        activity.setPendingFriendRequestCount(Int(list.total))
    }

    private func decrementPendingCount() {
        activity.setPendingFriendRequestCount(activity.pendingFriendRequestCount - 1)
    }

    private func resetOverlay() {
        statusOverrides = [:]
        removedIDs = []
        busyIDs = []
    }
}
