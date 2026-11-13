import Combine
import Foundation

/// 竞技分析页的读取状态。
///
/// - 只采纳最新一次请求；切换球种、退出或切号后，在途结果一律丢弃。
/// - 本页按球种保留已读结果：切回已读球种直接复用，不再请求。
/// - 切到未读球种时保留上一球种的内容挂载，Tab 显示刷新中（与小程序一致），新数据到了再替换。
@MainActor
final class StatsDetailModel: ObservableObject {
    @Published private(set) var currentGameType: Int
    @Published private(set) var selectedMonthKey: String
    @Published private(set) var overview: StatsOverviewResponse?
    @Published private(set) var isFetching = false
    @Published private(set) var hasLoadedOnce = false
    @Published private(set) var failure: LoadFailure?
    /// 已有内容时刷新失败的轻提示。
    @Published private(set) var refreshFailure: LoadFailure?

    private let session: SessionStore
    private let api: StatsAPI
    private let now: () -> Date
    private var owner: SessionIdentity?
    private var requestToken = 0
    private var responses: [Int: StatsOverviewResponse] = [:]

    init(session: SessionStore, gameType: Int, now: @escaping () -> Date = Date.init) {
        // 不支持的球种按页面默认球种读取。
        let initial = GameTypeCatalog.isSupported(gameType) ? gameType : 3
        self.session = session
        self.api = StatsAPI(session: session)
        self.now = now
        self.currentGameType = initial
        self.selectedMonthKey = AppFormat.monthKey(now())
        self.owner = session.identity
    }

    // MARK: - 展示值

    var loadingMode: StatsLoadingMode {
        StatsDetailRules.loadingMode(hasLoadedOnce: hasLoadedOnce, isFetching: isFetching)
    }

    var isInitialLoading: Bool { loadingMode == .initial }
    var isRefreshing: Bool { loadingMode == .refreshing }
    var isInitialError: Bool { !hasLoadedOnce && failure != nil && !isFetching }

    var failureFeedback: LoadFailure.Feedback? {
        failure?.feedback(resource: "统计数据")
    }

    var refreshMessage: String? {
        refreshFailure?.feedback(resource: "统计数据").description
    }

    var currentStats: StatsGameTypeStats? {
        StatsDetailRules.normalizeGameTypeStats(overview?.byGameType)[currentGameType]
    }

    var duration: StatsDurationValues {
        StatsDetailRules.normalizeDurationStats(overview?.duration)
    }

    var opponentRows: [StatsOpponentStrengthRow] {
        StatsDetailRules.normalizeOpponentStrengthStats(overview?.opponentStrength)
    }

    var highScore: StatsHighScoreSummary {
        StatsDetailRules.normalizeHighScoreStats(overview?.singleHighScores, now: now())
    }

    var trendCalendar: StatsTrendCalendar {
        StatsDetailRules.trendCalendar(
            matches: overview?.recentTrend ?? [],
            monthKey: selectedMonthKey,
            now: now()
        )
    }

    var rankDeltaChart: StatsRankDeltaChart {
        StatsDetailRules.rankDeltaChart(
            points: overview?.rankScoreTrend ?? [],
            monthKey: selectedMonthKey,
            now: now()
        )
    }

    var rankTrend: StatsRankTrend {
        StatsDetailRules.rankTrend(points: overview?.rankScoreTrend ?? [])
    }

    // MARK: - 读取

    func loadIfNeeded() async {
        syncIdentity()
        guard session.identity != nil else { return }
        if let cached = responses[currentGameType] {
            overview = cached
            return
        }
        guard !isFetching else { return }
        await fetch()
    }

    func retry() async {
        syncIdentity()
        guard session.identity != nil, !isFetching else { return }
        await fetch()
    }

    func reloadForIdentityChange() async {
        syncIdentity()
        guard session.identity != nil else { return }
        await fetch()
    }

    /// 切换球种：已读球种直接复用本页缓存，未读球种保留旧内容并进入刷新态。
    func selectGameType(_ gameType: Int) async {
        guard gameType != currentGameType else { return }
        currentGameType = gameType
        failure = nil
        refreshFailure = nil

        if let cached = responses[gameType] {
            // 作废上一球种的在途请求，避免它晚到后覆盖缓存结果。
            requestToken += 1
            isFetching = false
            overview = cached
            return
        }

        guard session.identity != nil else { return }
        await fetch()
    }

    func shiftMonth(by offset: Int) {
        selectedMonthKey = StatsDetailRules.shiftMonthKey(selectedMonthKey, offset: offset, now: now())
    }

    // MARK: - 私有

    private func fetch() async {
        guard let requestOwner = session.identity else { return }
        requestToken += 1
        let token = requestToken
        let requestedGameType = currentGameType
        isFetching = true
        failure = nil
        refreshFailure = nil

        do {
            let response = try await api.overview(gameType: requestedGameType)
            guard token == requestToken,
                  session.identity == requestOwner,
                  currentGameType == requestedGameType else { return }
            let merged = response.merged(with: responses[requestedGameType])
            responses[requestedGameType] = merged
            overview = merged
            hasLoadedOnce = true
            isFetching = false
        } catch {
            guard token == requestToken,
                  session.identity == requestOwner,
                  currentGameType == requestedGameType else { return }
            isFetching = false
            if isCancellation(error) { return }
            if overview == nil {
                failure = LoadFailure(error)
            } else {
                refreshFailure = LoadFailure(error)
            }
        }
    }

    /// 身份变化时作废在途请求并清空本页缓存；相同身份不做任何事。
    private func syncIdentity() {
        let current = session.identity
        guard current != owner else { return }
        owner = current
        requestToken += 1
        responses = [:]
        overview = nil
        isFetching = false
        hasLoadedOnce = false
        failure = nil
        refreshFailure = nil
    }

    private func isCancellation(_ error: Error) -> Bool {
        if let apiError = error as? APIError, apiError.kind == .canceled { return true }
        return error is CancellationError
    }
}
