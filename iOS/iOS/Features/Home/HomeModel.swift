import Combine
import Foundation

/// 首页读取与展示状态：各区块独立读取，单个失败只影响自己。
///
/// - 公开数据按 60 秒新鲜度重读；附近球房另有 3 分钟定位节流。
/// - 所有结果绑定发起时的身份，只采纳最新一次请求。
@MainActor
final class HomeModel: ObservableObject {
    /// 与小程序公共读取 TTL 一致。
    static let freshness: TimeInterval = 60

    @Published private(set) var leaderboardPhase: HomeSectionPhase = .idle
    @Published private(set) var leaderboardTopThree: [LeaderboardItem] = []
    @Published private(set) var myRanking: LeaderboardItem?

    @Published private(set) var eventPhase: HomeSectionPhase = .idle
    @Published private(set) var topEventNews: SaiXunCard?

    @Published private(set) var livePhase: HomeSectionPhase = .idle
    @Published private(set) var liveMatches: [SpectatorMatch] = []

    @Published private(set) var nearbyPhase: HomeSectionPhase = .idle
    @Published private(set) var nearbyVenues: [VenueInfo] = []
    @Published private(set) var nearbyLocationIssue: HomeLocationIssue?

    @Published private(set) var isRefreshing = false

    private let session: SessionStore
    private let location: LocationProviding
    private let api: HomeAPI
    private let now: () -> Date

    private var owner: SessionIdentity?
    private var leaderboardToken = 0
    private var eventToken = 0
    private var liveToken = 0
    private var nearbyToken = 0

    private var leaderboardLoadedAt: Date?
    private var eventLoadedAt: Date?
    private var liveLoadedAt: Date?
    private var nearbyAttemptAt: Date?

    init(
        session: SessionStore,
        location: LocationProviding? = nil,
        now: @escaping () -> Date = Date.init
    ) {
        self.session = session
        self.location = location ?? SystemLocationProvider()
        self.api = HomeAPI(session: session)
        self.now = now
        self.owner = session.identity
    }

    // MARK: - 生命周期

    /// 页面出现：按新鲜度重读公开数据；失败过的区块也在这里重试。
    func onAppear() async {
        syncIdentity()
        async let leaderboard: Void = loadLeaderboard(force: false)
        async let event: Void = loadEventNews(force: false)
        async let live: Void = loadLiveMatches(force: false)
        async let nearby: Void = loadNearby(force: false)
        _ = await (leaderboard, event, live, nearby)
    }

    /// 下拉刷新：全部强制重读。
    func refresh() async {
        syncIdentity()
        guard !isRefreshing else { return }
        isRefreshing = true
        async let leaderboard: Void = loadLeaderboard(force: true)
        async let event: Void = loadEventNews(force: true)
        async let live: Void = loadLiveMatches(force: true)
        async let nearby: Void = loadNearby(force: true)
        _ = await (leaderboard, event, live, nearby)
        isRefreshing = false
    }

    /// 登录 / 退出 / 切号后重读与身份相关的区块。
    func onIdentityChanged() async {
        guard session.identity != owner else { return }
        owner = session.identity
        leaderboardToken += 1
        leaderboardPhase = .idle
        leaderboardTopThree = []
        myRanking = nil
        leaderboardLoadedAt = nil
        await loadLeaderboard(force: true)
    }

    func retryLeaderboard() async { await loadLeaderboard(force: true) }
    func retryEventNews() async { await loadEventNews(force: true) }
    func retryLiveMatches() async { await loadLiveMatches(force: true) }
    func retryNearby() async { await loadNearby(force: true) }

    // MARK: - 推导状态

    var isPageLoading: Bool {
        leaderboardPhase == .idle || leaderboardPhase == .loading
            || eventPhase == .idle || eventPhase == .loading
            || livePhase == .idle || livePhase == .loading
    }

    /// 整页错误只由排行榜与赛事共同决定，与小程序的全页状态一致。
    var pageFailure: LoadFailure.Feedback? {
        guard case .failed(let failure) = leaderboardPhase,
              eventPhase.isFailed else { return nil }
        return failure.feedback(resource: "首页内容")
    }

    var hasPublicContent: Bool {
        !leaderboardTopThree.isEmpty || topEventNews != nil || !liveMatches.isEmpty || !nearbyVenues.isEmpty
    }

    /// 附近球房的失败提示，对应模板 nearbyVenueFeedback。
    var nearbyFeedback: LoadFailure.Feedback? {
        switch nearbyLocationIssue {
        case .denied:
            return LoadFailure.Feedback(
                title: "未开启定位权限",
                description: "开启定位权限后可查看附近球房",
                actionText: "重试",
                retryable: true
            )
        case .unavailable:
            return LoadFailure.Feedback(
                title: "暂时无法获取位置",
                description: "请检查定位服务后重试",
                actionText: "重试",
                retryable: true
            )
        case nil:
            guard case .failed(let failure) = nearbyPhase else { return nil }
            return failure.feedback(resource: "附近球房")
        }
    }

    var venueEmptyAction: HomeVenueEmptyAction { HomeRules.venueEmptyAction(rewardClaimable: false) }

    // MARK: - 读取

    private func syncIdentity() {
        owner = session.identity
    }

    private func loadLeaderboard(force: Bool) async {
        if !force {
            if leaderboardPhase == .loading { return }
            if leaderboardPhase == .loaded, !isExpired(leaderboardLoadedAt, maxAge: Self.freshness) { return }
        }
        leaderboardToken += 1
        let token = leaderboardToken
        let requestOwner = session.identity
        if leaderboardPhase != .loaded { leaderboardPhase = .loading }

        do {
            let summary = try await api.leaderboardSummary()
            guard token == leaderboardToken, session.identity == requestOwner else { return }
            leaderboardTopThree = Array(summary.topThree.prefix(3))
            myRanking = summary.myRanking
            leaderboardPhase = .loaded
            leaderboardLoadedAt = now()
        } catch {
            guard token == leaderboardToken, session.identity == requestOwner else { return }
            if isCancellation(error) {
                if leaderboardPhase == .loading { leaderboardPhase = .idle }
                return
            }
            leaderboardPhase = .failed(LoadFailure(error))
        }
    }

    private func loadEventNews(force: Bool) async {
        if !force {
            if eventPhase == .loading { return }
            if eventPhase == .loaded, !isExpired(eventLoadedAt, maxAge: Self.freshness) { return }
        }
        eventToken += 1
        let token = eventToken
        let requestOwner = session.identity
        if eventPhase != .loaded { eventPhase = .loading }

        do {
            let item = try await api.topEventNews()
            guard token == eventToken, session.identity == requestOwner else { return }
            topEventNews = item.map { SaiXunRules.normalizeSaiXunCard($0, now: now()) }
            eventPhase = .loaded
            eventLoadedAt = now()
        } catch {
            guard token == eventToken, session.identity == requestOwner else { return }
            if isCancellation(error) {
                if eventPhase == .loading { eventPhase = .idle }
                return
            }
            eventPhase = .failed(LoadFailure(error))
        }
    }

    private func loadLiveMatches(force: Bool) async {
        if !force {
            if livePhase == .loading { return }
            if livePhase == .loaded, !isExpired(liveLoadedAt, maxAge: Self.freshness) { return }
        }
        liveToken += 1
        let token = liveToken
        let requestOwner = session.identity
        if livePhase != .loaded { livePhase = .loading }

        do {
            let items = try await api.livePreview()
            guard token == liveToken, session.identity == requestOwner else { return }
            liveMatches = items
            livePhase = .loaded
            liveLoadedAt = now()
        } catch {
            guard token == liveToken, session.identity == requestOwner else { return }
            if isCancellation(error) {
                if livePhase == .loading { livePhase = .idle }
                return
            }
            // 失败时保留上一次的列表，只在区块内提示。
            livePhase = .failed(LoadFailure(error))
        }
    }

    private func loadNearby(force: Bool) async {
        guard nearbyPhase != .loading else { return }
        guard HomeRules.shouldReloadNearby(force: force, lastAttemptAt: nearbyAttemptAt, now: now()) else {
            return
        }
        nearbyAttemptAt = now()
        nearbyToken += 1
        let token = nearbyToken
        let requestOwner = session.identity
        nearbyPhase = .loading
        nearbyLocationIssue = nil

        do {
            let coordinate = try await location.currentLocation()
            guard token == nearbyToken, session.identity == requestOwner else { return }
            guard let params = HomeRules.nearbyVenueParams(coordinate) else {
                nearbyLocationIssue = .unavailable
                nearbyPhase = .loaded
                return
            }
            let venues = try await api.nearbyVenues(params)
            guard token == nearbyToken, session.identity == requestOwner else { return }
            nearbyVenues = venues
            nearbyPhase = .loaded
        } catch {
            guard token == nearbyToken, session.identity == requestOwner else { return }
            if let locationError = error as? LocationError {
                nearbyLocationIssue = locationError == .denied ? .denied : .unavailable
                nearbyPhase = .loaded
                return
            }
            if isCancellation(error) {
                nearbyPhase = .idle
                return
            }
            nearbyPhase = .failed(LoadFailure(error))
        }
    }

    private func isExpired(_ loadedAt: Date?, maxAge: TimeInterval) -> Bool {
        guard let loadedAt else { return true }
        return now().timeIntervalSince(loadedAt) >= maxAge
    }

    private func isCancellation(_ error: Error) -> Bool {
        if let apiError = error as? APIError, apiError.kind == .canceled { return true }
        return error is CancellationError
    }
}

private extension HomeSectionPhase {
    var isFailed: Bool {
        if case .failed = self { return true }
        return false
    }
}
