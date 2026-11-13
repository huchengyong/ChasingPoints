import Combine
import Foundation

/// 交锋记录页：概览（统计 + 当页历史）与同月后续分页共用一套请求令牌，只采纳最新一次结果。
@MainActor
final class H2HRecordModel: ObservableObject {
    static let pageSize = H2HAPI.pageSize

    let targetUserId: Int64
    let targetName: String
    let targetAvatar: String
    let opponentId: Int64
    let opponentName: String

    @Published private(set) var opponent: H2HOpponent?
    @Published private(set) var stats: H2HStats?
    @Published private(set) var history: [MatchListItem] = []
    @Published private(set) var total = 0
    @Published private(set) var hasMore = false
    @Published private(set) var isHidden = false
    @Published private(set) var isLoadingMore = false
    /// 部分分区失败或加载更多失败的提示；页面用横幅展示已获取内容。
    @Published private(set) var partialMessage: String?
    /// 首屏失败；保留分类供错误态选择「重新加载」还是「返回」。
    @Published private(set) var failure: LoadFailure?
    @Published private(set) var statsLoaded = false
    @Published private(set) var historyLoaded = false
    @Published private(set) var selectedMonthKey: String
    @Published var historyViewMode: H2HHistoryViewMode = .calendar
    /// 好友关系失效时的提示；页面负责提示并返回上一页。
    @Published private(set) var loadFailureAction: H2HLoadFailureAction?

    private let session: SessionStore
    private let api: H2HAPI
    private var owner: SessionIdentity?
    private var requestToken = 0
    private var loadedPage = 0
    private var isFetching = false

    init(
        session: SessionStore,
        targetUserId: Int64 = 0,
        targetName: String = "",
        targetAvatar: String = "",
        opponentId: Int64 = 0,
        opponentName: String = "",
        monthKey: String? = nil
    ) {
        self.session = session
        self.targetUserId = targetUserId
        self.targetName = targetName
        self.targetAvatar = targetAvatar
        self.opponentId = opponentId
        self.opponentName = opponentName
        selectedMonthKey = monthKey ?? H2HRules.currentMonthKey()
        api = H2HAPI(session: session)
    }

    // MARK: - 展示值

    private var resolvedOpponentName: String {
        if let name = opponent?.name, !name.isEmpty { return name }
        return opponentName
    }

    /// 概览可能不返回 opponent（旧版服务端），用路由带入的名字补齐英雄区。
    private var displayedOpponent: H2HOpponent? {
        if let opponent, !opponent.name.isEmpty { return opponent }
        return resolvedOpponentName.isEmpty
            ? nil
            : H2HOpponent(id: opponentId, name: resolvedOpponentName, avatar: "")
    }

    var viewModel: H2HViewModel {
        H2HRules.viewModel(
            targetUserId: targetUserId,
            targetName: targetName,
            opponentId: opponentId > 0 ? opponentId : (opponent?.id ?? 0),
            currentUserId: session.user?.id ?? 0,
            opponentName: resolvedOpponentName
        )
    }

    var subjectAvatar: String {
        targetUserId > 0 ? targetAvatar : (session.user?.avatar ?? "")
    }

    var subjectUserId: Int64 {
        targetUserId > 0 ? targetUserId : (session.user?.id ?? 0)
    }

    /// 路由带进来的对手 ID 优先，概览返回的对手用于补齐。
    var resolvedOpponentId: Int64 {
        opponentId > 0 ? opponentId : (opponent?.id ?? 0)
    }

    var hero: H2HHero {
        H2HRules.hero(
            stats: stats,
            opponent: displayedOpponent,
            history: history,
            subjectName: viewModel.subjectName,
            subjectAvatar: subjectAvatar
        )
    }

    var pageStatus: H2HPageStatus {
        H2HRules.pageStatus(
            statsLoaded: statsLoaded,
            historyLoaded: historyLoaded,
            totalMatches: stats?.totalMatches ?? 0,
            historyLength: history.count,
            hasError: failure != nil || partialMessage != nil
        )
    }

    var showSummaryCard: Bool {
        H2HRules.shouldShowSummaryCard(pageStatus: pageStatus, statsLoaded: statsLoaded)
    }

    var failureFeedback: LoadFailure.Feedback? {
        failure?.feedback(resource: "交锋数据")
    }

    var calendar: H2HCalendar {
        H2HRules.calendar(history: history, monthKey: selectedMonthKey)
    }

    var matchCards: [H2HMatchCard] {
        H2HRules.matchCards(history)
    }

    var primaryActionRoute: AppRoute {
        .challenges(showHistory: false)
    }

    // MARK: - 读取

    func onAppear() async {
        syncIdentity()
        guard session.identity != nil else { return }
        // 已采纳过结果（含空/隐藏）时不重复读取；月份切换与重试走 reload。
        guard !statsLoaded, !historyLoaded, !isHidden, failure == nil, !isFetching else { return }
        await load()
    }

    func reload() async {
        syncIdentity()
        guard session.identity != nil else { return }
        await load()
    }

    func retry() async {
        await reload()
    }

    func onIdentityChanged() async {
        syncIdentity()
        await onAppear()
    }

    func shiftMonth(by offset: Int) async {
        selectedMonthKey = H2HRules.shiftMonthKey(selectedMonthKey, offset: offset)
        await reload()
    }

    func loadMore() async {
        syncIdentity()
        guard session.identity != nil, !isHidden, historyLoaded, hasMore, !isLoadingMore else { return }
        let page = loadedPage + 1
        requestToken += 1
        let token = requestToken
        let requestOwner = session.identity
        isLoadingMore = true

        do {
            let response = try await api.history(
                page: page,
                targetUserId: targetUserId,
                opponentId: opponentId > 0 ? opponentId : (opponent?.id ?? 0),
                opponentName: resolvedOpponentName,
                monthKey: selectedMonthKey
            )
            guard token == requestToken, session.identity == requestOwner else { return }
            isLoadingMore = false
            if response.hidden {
                isHidden = true
                history = []
                total = 0
                hasMore = false
                partialMessage = nil
                return
            }
            let list = response.list ?? []
            loadedPage = page
            if response.total > 0 { total = Int(response.total) }
            history.append(contentsOf: list)
            hasMore = history.count < total
            historyLoaded = true
            partialMessage = nil
        } catch {
            guard token == requestToken, session.identity == requestOwner else { return }
            isLoadingMore = false
            if isCancellation(error) { return }
            partialMessage = error.localizedDescription
        }
    }

    private func load() async {
        requestToken += 1
        let token = requestToken
        let requestOwner = session.identity
        isFetching = true

        // 与小程序一致：切月/重载先回到加载态，不让上个月的统计和列表继续展示。
        statsLoaded = false
        historyLoaded = false
        isHidden = false
        stats = nil
        history = []
        total = 0
        hasMore = false
        loadedPage = 0
        failure = nil
        partialMessage = nil
        loadFailureAction = nil

        do {
            let response = try await api.overview(
                targetUserId: targetUserId,
                opponentId: opponentId,
                opponentName: resolvedOpponentName,
                monthKey: selectedMonthKey
            )
            guard token == requestToken, session.identity == requestOwner else { return }
            isFetching = false
            apply(response)
        } catch {
            guard token == requestToken, session.identity == requestOwner else { return }
            isFetching = false
            if isCancellation(error) { return }
            failure = LoadFailure(error)
            loadFailureAction = H2HRules.loadFailureAction(
                targetUserId: targetUserId,
                message: error.localizedDescription
            )
        }
    }

    private func apply(_ response: H2HOverviewResponse) {
        // 对方隐藏战绩是正常业务结果，不展示统计与列表，也不当作加载失败。
        if response.hidden {
            isHidden = true
            opponent = nil
            stats = nil
            history = []
            total = 0
            hasMore = false
            return
        }

        opponent = response.opponent

        if response.availability?["stats"] != false {
            stats = response.stats
            statsLoaded = true
        }
        if response.availability?["history"] != false {
            let list = response.list ?? []
            total = Int(response.total)
            history = list
            hasMore = response.hasMore || list.count < total
            loadedPage = 1
            historyLoaded = true
        }
        if !statsLoaded || !historyLoaded {
            partialMessage = "部分交锋数据加载失败"
        }
    }

    /// 身份变化时作废在途请求并清空数据；相同身份不做任何事。
    private func syncIdentity() {
        let current = session.identity
        guard current != owner else { return }
        owner = current
        requestToken += 1
        isFetching = false
        isLoadingMore = false
        stats = nil
        opponent = nil
        history = []
        total = 0
        hasMore = false
        loadedPage = 0
        isHidden = false
        statsLoaded = false
        historyLoaded = false
        partialMessage = nil
        failure = nil
        loadFailureAction = nil
    }

    private func isCancellation(_ error: Error) -> Bool {
        if let apiError = error as? APIError, apiError.kind == .canceled { return true }
        return error is CancellationError
    }
}
