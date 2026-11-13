import Combine
import Foundation

/// 表现分析页的读取状态。
///
/// - 不做跨页面缓存；本页按球种保留已读结果，切回已读球种不再请求。
/// - 切到未读球种时丢弃页面上的旧结果并显示加载态（对应小程序没有缓存时的行为）。
/// - 只采纳最新一次请求；退出或切号后，在途结果一律丢弃。
@MainActor
final class PerformanceModel: ObservableObject {
    @Published private(set) var currentGameType: Int
    @Published private(set) var data: PerformanceResponse?
    @Published private(set) var isFetching = false
    @Published private(set) var failure: LoadFailure?
    /// 已有内容时刷新失败的轻提示。
    @Published private(set) var refreshFailure: LoadFailure?

    private let session: SessionStore
    private let api: PerformanceAPI
    private var owner: SessionIdentity?
    private var requestToken = 0
    private var responses: [Int: PerformanceResponse] = [:]

    init(session: SessionStore, gameType: Int) {
        let initial = GameTypeCatalog.isSupported(gameType) ? gameType : 3
        self.session = session
        self.api = PerformanceAPI(session: session)
        self.currentGameType = initial
        self.owner = session.identity
    }

    // MARK: - 展示值

    var isInitialLoading: Bool { isFetching && data == nil }
    var isInitialError: Bool { data == nil && failure != nil && !isFetching }
    var isLocked: Bool { data?.locked == true }

    var failureFeedback: LoadFailure.Feedback? {
        failure?.feedback(resource: "表现分析")
    }

    var refreshMessage: String? {
        refreshFailure?.feedback(resource: "表现分析").description
    }

    /// 开通入口只由合规开关决定。
    var lockedView: PerformanceLockedView? {
        guard let data, data.locked else { return nil }
        return PerformanceRules.resolveLockedView(
            data,
            canShowPurchase: AppFlags.canShowPaidMembershipEntry
        )
    }

    var summaryView: PerformanceSummaryView {
        PerformanceRules.resolveSummaryView(data?.summary, sampleCount: data?.sampleCount ?? 0)
    }

    var notableViews: [PerformanceNotableMatchView] {
        [PerformanceRules.resolveNotableMatchView(data?.bestWin, kind: .bestWin),
         PerformanceRules.resolveNotableMatchView(data?.worstLoss, kind: .worstLoss)]
            .compactMap { $0 }
    }

    var chart: PerformanceMonthChart {
        PerformanceRules.buildMonthChart(data?.months)
    }

    // MARK: - 读取

    func loadIfNeeded() async {
        syncIdentity()
        guard session.identity != nil else { return }
        if let cached = responses[currentGameType] {
            data = cached
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

    func selectGameType(_ gameType: Int) async {
        guard gameType != currentGameType else { return }
        currentGameType = gameType
        failure = nil
        refreshFailure = nil

        if let cached = responses[gameType] {
            requestToken += 1
            isFetching = false
            data = cached
            return
        }

        // 旧球种的内容不能挂在新球种下面。
        data = nil
        guard session.identity != nil else { return }
        await fetch()
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
            let response = try await api.performance(gameType: requestedGameType)
            guard token == requestToken,
                  session.identity == requestOwner,
                  currentGameType == requestedGameType else { return }
            responses[requestedGameType] = response
            data = response
            isFetching = false
        } catch {
            guard token == requestToken,
                  session.identity == requestOwner,
                  currentGameType == requestedGameType else { return }
            isFetching = false
            if isCancellation(error) { return }
            if data == nil {
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
        data = nil
        isFetching = false
        failure = nil
        refreshFailure = nil
    }

    private func isCancellation(_ error: Error) -> Bool {
        if let apiError = error as? APIError, apiError.kind == .canceled { return true }
        return error is CancellationError
    }
}
