import Combine
import Foundation

/// 单个资源的读取状态，结果绑定发起时的身份。
///
/// - 只采纳最新一次请求的结果；退出或切号后，在途结果一律丢弃。
/// - 不做跨页面缓存，由调用方决定实例的生命周期。
@MainActor
final class ResourceLoader<Value>: ObservableObject {
    enum Access {
        /// 必须登录：没有身份时不请求。
        case signedIn
        /// 访客可读：没有身份也请求。
        case anyone
    }

    enum Phase {
        case idle
        case loading
        case loaded(Value)
        case failed(String)
    }

    @Published private(set) var phase: Phase = .idle
    @Published private(set) var isRefreshing = false
    /// 已有数据时刷新失败的提示；数据保持不变。
    @Published private(set) var refreshError: String?

    private let session: SessionStore
    private let access: Access
    private let now: () -> Date
    private let load: () async throws -> Value
    private var owner: SessionIdentity?
    private var requestToken = 0
    /// 写操作成功后置位，下次出现时以刷新方式重读。
    private var isStale = false
    /// 最近一次成功采纳结果的时刻，供按时间判断新鲜度使用。
    private var lastLoadedAt: Date?

    init(
        session: SessionStore,
        access: Access = .signedIn,
        now: @escaping () -> Date = Date.init,
        load: @escaping () async throws -> Value
    ) {
        self.session = session
        self.access = access
        self.now = now
        self.load = load
    }

    var value: Value? {
        if case .loaded(let value) = phase { return value }
        return nil
    }

    var isLoading: Bool {
        if case .loading = phase { return true }
        return false
    }

    var failureMessage: String? {
        if case .failed(let message) = phase { return message }
        return nil
    }

    /// 身份变化时作废在途请求并清空状态；相同身份不做任何事。
    func syncIdentity() {
        let current = session.identity
        guard current != owner else { return }
        owner = current
        requestToken += 1
        phase = .idle
        isRefreshing = false
        refreshError = nil
        isStale = false
        lastLoadedAt = nil
    }

    /// 传 `maxAge` 时，已加载的数据超过该时长也会以刷新方式重读（旧数据保留到新结果返回）。
    func loadIfNeeded(maxAge: TimeInterval? = nil) async {
        syncIdentity()
        guard canRequest else { return }
        switch phase {
        case .idle:
            await fetch(isFirstLoad: true)
        case .loaded:
            guard isStale || isExpired(maxAge: maxAge) else { return }
            await fetch(isFirstLoad: false)
        default:
            return
        }
    }

    func refresh() async {
        syncIdentity()
        guard canRequest, !isRefreshing else { return }
        if case .loaded = phase {
            await fetch(isFirstLoad: false)
        } else {
            await fetch(isFirstLoad: true)
        }
    }

    func retry() async {
        syncIdentity()
        guard canRequest else { return }
        if case .failed = phase {
            await fetch(isFirstLoad: true)
        } else {
            await refresh()
        }
    }

    /// 仅做过期标记（写操作成功后由调用方调用），不发请求。
    func invalidate() {
        isStale = true
    }

    private var canRequest: Bool {
        access == .anyone || owner != nil
    }

    /// 未传 `maxAge` 或还没有成功记录时都不算过期。
    private func isExpired(maxAge: TimeInterval?) -> Bool {
        guard let maxAge, let lastLoadedAt else { return false }
        return now().timeIntervalSince(lastLoadedAt) > maxAge
    }

    private func fetch(isFirstLoad: Bool) async {
        requestToken += 1
        let token = requestToken
        let requestOwner = session.identity

        if isFirstLoad {
            phase = .loading
        } else {
            isRefreshing = true
        }

        do {
            let value = try await load()
            guard token == requestToken, session.identity == requestOwner else { return }
            phase = .loaded(value)
            isRefreshing = false
            refreshError = nil
            lastLoadedAt = now()
            // 只有刷新成功才算补上过期数据；失败或取消时保留标记，下次出现继续重读。
            isStale = false
        } catch {
            guard token == requestToken, session.identity == requestOwner else { return }
            isRefreshing = false
            // 页面离开导致的取消不是失败：首次加载回到可重新加载的状态，下次出现时再读。
            if isCancellation(error) {
                if isFirstLoad { phase = .idle }
                return
            }
            let message = error.localizedDescription
            if isFirstLoad {
                phase = .failed(message)
            } else {
                refreshError = message
            }
        }
    }

    private func isCancellation(_ error: Error) -> Bool {
        if let apiError = error as? APIError, apiError.kind == .canceled { return true }
        return error is CancellationError
    }
}

extension ResourceLoader.Phase: Equatable where Value: Equatable {}
