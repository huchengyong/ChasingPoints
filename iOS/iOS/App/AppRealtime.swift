import Combine
import Foundation
import Network

/// 把用户实时通道接到应用生命周期；只在正式启动路径里创建，测试宿主不创建。
@MainActor
final class AppRealtime: ObservableObject {
    private let session: SessionStore
    private let activity: ActivityStore
    private let user: UserRealtime
    private var connectedIdentity: SessionIdentity?
    private var foreground = true
    private var online = true

    init(session: SessionStore, activity: ActivityStore, user: UserRealtime? = nil) {
        self.session = session
        self.activity = activity
        let realtime = user ?? UserRealtime(session: session)
        self.user = realtime

        // 会话失效是一条带凭据的普通请求：由 SessionStore 走标准 401 / SESSION_INVALID 清理。
        realtime.sessionInvalidHandler = { [weak self] _ in
            guard let self else { return }
            Task { await self.activity.refresh() }
        }
        realtime.on(WSMessageType.notificationUpdate) { [weak self] message in
            self?.handleNotificationUpdate(message)
        }
        realtime.on(WSMessageType.userDataUpdated) { [weak self] message in
            self?.handleUserDataUpdated(message)
        }
        realtime.on(WSMessageType.rankInfoUpdated) { [weak self] message in
            self?.handleRankInfoUpdated(message)
        }
    }

    /// 身份出现 → 连接；身份消失或变化 → 断开后按新身份重连。
    func handleIdentityChange() async {
        guard let identity = session.identity else {
            user.disconnect()
            connectedIdentity = nil
            return
        }
        if connectedIdentity == identity, user.isConnected { return }
        if connectedIdentity != nil || user.target != nil {
            user.disconnect()
        }
        connectedIdentity = identity
        await connectIfPossible()
    }

    func setForeground(_ value: Bool) {
        guard foreground != value else { return }
        foreground = value
        user.setForeground(value)
        if value {
            Task { await connectIfPossible() }
        } else {
            // 小程序进入后台时同时断开，回到前台再重新连接。
            user.disconnect()
            connectedIdentity = nil
        }
    }

    func setNetworkOnline(_ value: Bool) {
        guard online != value else { return }
        online = value
        user.setNetworkOnline(value)
        if value {
            Task { await connectIfPossible() }
        } else {
            connectedIdentity = nil
        }
    }

    private func connectIfPossible() async {
        guard let identity = session.identity else {
            user.disconnect()
            connectedIdentity = nil
            return
        }
        guard foreground, online else { return }
        do {
            try await user.connect(authGeneration: identity.generation)
            connectedIdentity = identity
        } catch {
            connectedIdentity = nil
        }
    }

    // MARK: - 实时消息

    private func handleNotice(_ message: RealtimeMessage) -> NotificationUpdatePayload? {
        decode(NotificationUpdatePayload.self, message.payload)
    }

    private func handleNotificationUpdate(_ message: RealtimeMessage) {
        guard session.identity != nil else { return }
        if let unreadCount = handleNotice(message)?.unreadCount {
            activity.setUnreadCount(unreadCount)
        }
        activity.markDirty()
        Task { await activity.refreshIfDirty() }
    }

    private func handleRankInfoUpdated(_ message: RealtimeMessage) {
        handleUserDataUpdated(message)
    }

    private func handleUserDataUpdated(_ message: RealtimeMessage) {
        guard session.identity != nil else { return }
        let payload = decode(UserDataUpdatedPayload.self, message.payload)
        if let count = payload?.pendingFriendRequestCount {
            activity.setPendingFriendRequestCount(count)
        }
        activity.bumpUserDataRevision()
        activity.markDirty()
    }

    private func decode<T: Decodable>(_ type: T.Type, _ data: Data?) -> T? {
        guard let data else { return nil }
        let decoder = JSONDecoder()
        decoder.keyDecodingStrategy = .convertFromSnakeCase
        return try? decoder.decode(type, from: data)
    }
}

private struct NotificationUpdatePayload: Decodable {
    let unreadCount: Int?
    let category: String?
}

private struct UserDataUpdatedPayload: Decodable {
    let scopes: [String]?
    let pendingFriendRequestCount: Int?
    let competitiveRevision: Int64?
}

/// 网络状态监听的可注入包装：测试不依赖真实网络。
@MainActor
protocol NetworkStatusMonitor: AnyObject {
    var onStatusChange: ((Bool) -> Void)? { get set }
    func start()
}

/// 以 `NWPathMonitor` 实现；回调固定回主线程。
@MainActor
final class NWPathNetworkMonitor: NetworkStatusMonitor {
    var onStatusChange: ((Bool) -> Void)?

    private let monitor = NWPathMonitor()
    private let queue = DispatchQueue(label: "ChasingPoints.NetworkStatus")
    private var started = false

    func start() {
        guard !started else { return }
        started = true
        monitor.pathUpdateHandler = { [weak self] path in
            let online = path.status == .satisfied
            DispatchQueue.main.async {
                self?.onStatusChange?(online)
            }
        }
        monitor.start(queue: queue)
    }

    deinit {
        monitor.cancel()
    }
}
