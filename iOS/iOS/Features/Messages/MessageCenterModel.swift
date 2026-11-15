import Combine
import Foundation

/// 消息中心：顶部入口计数来自 ActivityStore，通知列表按身份与活动状态脏标记决定是否回源。
@MainActor
final class MessageCenterModel: ObservableObject {
    enum Phase: Equatable {
        case idle
        case loading
        case loaded
        case empty
        case failed(String)
    }

    let list: PagedHistoryList<NotificationInfo>

    @Published private(set) var readIDs: Set<Int64> = []
    @Published private(set) var deletedIDs: Set<Int64> = []
    @Published private(set) var isAllRead = false

    private let session: SessionStore
    private let activity: ActivityStore
    private let api: MessageCenterAPI
    private var loadedIdentityKey = ""
    private var requestedIdentityKey = ""
    private var loadedScopeVersion = -1

    init(session: SessionStore, activity: ActivityStore) {
        self.session = session
        self.activity = activity
        let api = MessageCenterAPI(session: session)
        self.api = api
        list = PagedHistoryList(session: session) { page in
            let identity = session.identity
            let response = try await api.list(page: page)
            if session.identity == identity {
                activity.setUnreadCount(response.unreadCount)
            }
            return (response.list, Int(response.total))
        }
    }

    // MARK: - 展示值

    /// 叠加本地已读与删除：服务端结果不变，界面先反映成功的写操作。
    var items: [NotificationInfo] {
        list.items.compactMap { item in
            guard !deletedIDs.contains(item.id) else { return nil }
            var copy = item
            if isAllRead || readIDs.contains(item.id) {
                copy.isRead = true
            }
            return copy
        }
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
    var hasMore: Bool { list.hasMore }

    var entries: [MessageEntry] {
        MessageCenterRules.buildMessageEntries(
            friendRequestCount: activity.pendingFriendRequestCount,
            challengeCount: activity.receivedChallengeCount
        )
    }

    func isRead(_ item: NotificationInfo) -> Bool {
        isAllRead || item.isRead || readIDs.contains(item.id)
    }

    // MARK: - 加载

    func onAppear() async {
        await activity.refreshIfNeeded()
        await reloadIfNeeded()
    }

    func identityChanged() async {
        reset()
        await onAppear()
    }

    func retry() async {
        guard let identity = session.identity else { return }
        await loadFirstPage(identityKey: identityKey(identity))
    }

    /// 下拉刷新同时重读通知首屏与活动快照，对应小程序的 onPullDownRefresh。
    func refresh() async {
        guard let identity = session.identity else { return }
        await loadFirstPage(identityKey: identityKey(identity))
        await activity.refresh()
    }

    func loadMore() async {
        await list.loadMore()
    }

    private func reloadIfNeeded() async {
        guard let identity = session.identity else {
            reset()
            return
        }
        let key = identityKey(identity)
        if !loadedIdentityKey.isEmpty, loadedIdentityKey != key {
            reset()
        }
        let decision = MessageCenterRules.resolveNotificationReload(
            identityKey: key,
            loadedIdentityKey: loadedIdentityKey,
            requestedIdentityKey: requestedIdentityKey,
            loadedScopeVersion: loadedScopeVersion,
            scopeVersion: activity.dirtySeq
        )
        guard decision == .load else { return }
        await loadFirstPage(identityKey: key)
    }

    private func loadFirstPage(identityKey key: String) async {
        guard let currentIdentity = session.identity, identityKey(currentIdentity) == key else { return }
        // 以请求开始时的失效版本为准：加载期间到达的新通知会让下一次显示再刷新。
        let scopeVersion = activity.dirtySeq
        requestedIdentityKey = key
        await list.refresh()
        guard let latestIdentity = session.identity, identityKey(latestIdentity) == key else { return }
        // 失败时也要清掉，否则下次显示会被误判为「该身份的加载仍在进行」而不再回源。
        requestedIdentityKey = ""
        if case .failed = list.phase { return }
        loadedIdentityKey = key
        loadedScopeVersion = scopeVersion
        readIDs = []
        deletedIDs = []
        isAllRead = false
    }

    private func reset() {
        list.syncIdentity()
        loadedIdentityKey = ""
        requestedIdentityKey = ""
        loadedScopeVersion = -1
        readIDs = []
        deletedIDs = []
        isAllRead = false
    }

    private func identityKey(_ identity: SessionIdentity) -> String {
        "\(identity.userId):\(identity.generation)"
    }

    // MARK: - 写操作

    /// 标记已读；失败时保持原状（JS 也只在成功后计数）。
    func markRead(_ item: NotificationInfo) async {
        guard !isRead(item), let identity = session.identity else { return }
        do {
            try await api.markRead(id: item.id)
            guard session.identity == identity else { return }
            readIDs.insert(item.id)
            activity.decrementUnread()
        } catch {}
    }

    func markAllRead() async -> Bool {
        guard let identity = session.identity else { return false }
        do {
            try await api.markAllRead()
            guard session.identity == identity else { return false }
            isAllRead = true
            activity.setUnreadCount(0)
            return true
        } catch {
            return false
        }
    }

    func delete(_ item: NotificationInfo) async -> Bool {
        guard let identity = session.identity else { return false }
        do {
            try await api.delete(id: item.id)
            guard session.identity == identity else { return false }
            if !isRead(item) {
                activity.decrementUnread()
            }
            deletedIDs.insert(item.id)
            return true
        } catch {
            return false
        }
    }
}
