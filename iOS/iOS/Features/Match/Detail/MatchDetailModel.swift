import Combine
import Foundation

/// 对局详情（小程序 `matchDetail`）：三种入口共用一个页面，结果绑定发起时的会话身份。
///
/// - `mode == "spectate"` 或 `source == "history"` 读公开详情，其余入口读参与者详情；
/// - 进行中的对局连实时通道，比分推送就地更新；九球追分在推送后延迟静默补读得分明细；
/// - 对局结束改为重新读取详情，比分与局记录都来自同一份接口快照。
@MainActor
final class MatchDetailModel: ObservableObject {
    enum Phase: Equatable {
        case loading
        case failed(LoadFailure)
        case loaded(MatchDetailViewData)
    }

    @Published private(set) var phase: Phase = .loading

    let matchId: Int64
    /// `spectate` 时显示观战徽标并按无票据连接实时通道。
    let mode: String
    let source: String
    /// 战绩列表进入时表示「站在谁的角度看」，缺省 0 表示不换边。
    let perspectiveUserId: Int64

    private let session: SessionStore
    private let api: LiveMatchAPI
    private let channel: LiveMatchChannel
    private let refreshDelay: @Sendable () -> TimeInterval
    private let sleep: @Sendable (TimeInterval) async throws -> Void

    private var snapshot: LiveMatchSnapshot?
    private var messageToken: UUID?
    private var requestToken = 0
    private var refreshToken = 0
    private var refreshTask: Task<Void, Never>?
    private var isActive = false

    init(
        session: SessionStore,
        matchId: Int64,
        mode: String = "",
        source: String = "",
        perspectiveUserId: Int64 = 0,
        api: LiveMatchAPI? = nil,
        channel: LiveMatchChannel? = nil,
        refreshDelay: @escaping @Sendable () -> TimeInterval = { 0.4 + Double.random(in: 0..<0.6) },
        sleep: @escaping @Sendable (TimeInterval) async throws -> Void = { try await Task.sleep(for: .seconds($0)) }
    ) {
        self.session = session
        self.matchId = matchId
        self.mode = mode
        self.source = source
        self.perspectiveUserId = perspectiveUserId
        self.api = api ?? LiveMatchAPI(session: session)
        self.channel = channel ?? MatchRealtime(session: session)
        self.refreshDelay = refreshDelay
        self.sleep = sleep
    }

    var isSpectate: Bool { mode == "spectate" }

    /// 分享链接进来的访客：观战模式且未登录时给一条登录引导。
    var showsLoginHint: Bool { isSpectate && session.identity == nil }

    var viewData: MatchDetailViewData? {
        if case .loaded(let data) = phase { return data }
        return nil
    }

    // MARK: - 生命周期

    /// 快照测试会先手动调用；页面 `.task` 重复触发时直接返回，避免二次加载。
    func start() async {
        guard !isActive else { return }
        isActive = true
        messageToken = channel.observeLive { [weak self] message in
            self?.handle(message)
        }
        await load()
    }

    func stop() {
        isActive = false
        cancelRefresh()
        if let messageToken {
            channel.stopObservingLive(messageToken)
        }
        messageToken = nil
        channel.closeLive()
    }

    func load() async {
        guard matchId > 0 else {
            phase = .failed(LoadFailure(category: .notFound, message: "缺少对局信息"))
            return
        }
        requestToken += 1
        let token = requestToken
        cancelRefresh()
        let identity = session.identity
        phase = .loading

        do {
            let response = try await fetch()
            guard token == requestToken, session.identity == identity else { return }
            guard response.success, let match = response.match else {
                phase = .failed(LoadFailure(category: .business, message: "暂时无法查看这场对局"))
                return
            }
            apply(match)
            await connectIfLive()
        } catch {
            guard token == requestToken, session.identity == identity else { return }
            if let apiError = error as? APIError, apiError.kind == .canceled { return }
            phase = .failed(failure(for: error))
        }
    }

    /// 业务失败（HTTP 200 + `success=false`）不区分不存在 / 非参与者，统一按同一句提示。
    private func failure(for error: Error) -> LoadFailure {
        if let apiError = error as? APIError, apiError.kind == .business, apiError.statusCode == 200 {
            return LoadFailure(category: .business, message: "暂时无法查看这场对局")
        }
        return LoadFailure(error)
    }

    private func fetch() async throws -> LiveMatchDetailResponse {
        if MatchDetailRules.shouldUsePublicMatchDetail(mode: mode, source: source) {
            return try await api.publicDetail(matchId: matchId)
        }
        return try await api.detail(matchId: matchId)
    }

    private func apply(_ match: LiveMatchSnapshot) {
        snapshot = match
        phase = .loaded(MatchDetailRules.normalize(match, perspectiveUserId: perspectiveUserId))
    }

    // MARK: - 实时

    private func connectIfLive() async {
        guard isActive, snapshot?.status == 1, !channel.isConnected else { return }
        do {
            try await channel.openLive(matchId: matchId, allowAnonymous: isSpectate)
        } catch {
            // 实时连接失败保留已读到的详情，下次进入页面再连。
        }
    }

    private func handle(_ message: RealtimeMessage) {
        switch message.type {
        case WSMessageType.scoreUpdate:
            merge(message, isSync: false)
            scheduleRefresh()
        case WSMessageType.roundEnd:
            merge(message, isSync: false)
            scheduleRefresh()
            // 局记录不在比分推送里，向服务端要一次整场快照补齐。
            Task { await channel.requestLiveSync() }
        case WSMessageType.sync:
            merge(message, isSync: true)
            scheduleRefresh()
        case WSMessageType.matchEnd:
            // 结束时比分与局记录以详情接口为准，避免推送与列表短暂不一致。
            cancelRefresh()
            Task { await refreshSilently() }
        default:
            break
        }
    }

    private func merge(_ message: RealtimeMessage, isSync: Bool) {
        guard let payload = message.payload,
              let current = snapshot,
              let incoming = try? Self.snapshotDecoder.decode(LiveMatchSnapshot.self, from: payload) else {
            return
        }
        apply(MatchDetailRules.mergePush(incoming, into: current, isSync: isSync))
    }

    /// 只有九球追分的得分构成是页面加载时取的，比分推送后延迟补读，避开大量观战者同时请求。
    private func scheduleRefresh() {
        guard snapshot?.gameType == 2, snapshot?.status == 1 else { return }
        refreshTask?.cancel()
        // 新消息到达后，仍在途中的静默刷新已经过期，不能再回写。
        refreshToken += 1
        let delay = refreshDelay()
        refreshTask = Task { [weak self] in
            guard let self else { return }
            try? await self.sleep(delay)
            guard !Task.isCancelled else { return }
            await self.refreshSilently()
        }
    }

    private func cancelRefresh() {
        refreshTask?.cancel()
        refreshTask = nil
        refreshToken += 1
    }

    /// 不切整页 loading；失败保留旧数据。
    private func refreshSilently() async {
        guard matchId > 0, snapshot != nil else { return }
        refreshToken += 1
        let token = refreshToken
        let identity = session.identity
        guard let response = try? await fetch(),
              token == refreshToken,
              session.identity == identity,
              response.success,
              let match = response.match else {
            return
        }
        apply(match)
    }

    private static let snapshotDecoder: JSONDecoder = {
        let decoder = JSONDecoder()
        decoder.keyDecodingStrategy = .convertFromSnakeCase
        return decoder
    }()
}
