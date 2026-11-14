import Combine
import Foundation

/// 赛事详情的展示数据：赛事头部、赛程分栏与冠军，逐条对应 `subPages/tournament/detail.vue` 的 `normalizeEventView`。
struct SaiXunDetail: Equatable {
    let card: SaiXunCard
    let dateText: String
    let timeText: String
    let showTime: Bool
    let regionText: String
    let rounds: [SaiXunRound]
    let scheduleTabs: [SaiXunScheduleTab]
    let champion: SaiXunChampion?
}

enum SaiXunDetailRules {
    /// 赛讯缺字段时沿用赛事实体：名称、封面、地点与日期都以赛事数据补齐。
    static func build(
        _ view: EventNewsView,
        now: Date,
        calendar: Calendar = SaiXunCalendar.shanghai
    ) -> SaiXunDetail? {
        guard let event = view.eventNews else { return nil }
        let tournament = view.tournament

        let merged = EventNewsInfo(
            id: event.id,
            title: event.title,
            sourceType: event.sourceType,
            sourceName: event.sourceName,
            sourceUrl: event.sourceUrl,
            coverImage: nonEmpty(event.coverImage) ?? nonEmpty(tournament?.coverImage) ?? "",
            summary: event.summary,
            content: event.content,
            description: event.description,
            tournamentId: event.tournamentId,
            tournamentName: nonEmpty(event.tournamentName)
                ?? nonEmpty(tournament?.name)
                ?? nonEmpty(event.title)
                ?? "",
            gameType: event.gameType,
            country: SaiXunRules.localizeCountryName(
                nonEmpty(event.country) ?? nonEmpty(tournament?.country) ?? ""
            ),
            city: nonEmpty(event.city) ?? nonEmpty(tournament?.city) ?? "",
            venue: nonEmpty(event.venue) ?? nonEmpty(tournament?.venueName) ?? "",
            startDate: nonEmpty(event.startDate) ?? nonEmpty(tournament?.startDate) ?? "",
            endDate: nonEmpty(event.endDate) ?? nonEmpty(tournament?.endDate) ?? "",
            startTime: nonEmpty(event.startTime) ?? nonEmpty(tournament?.startTime),
            endTime: nonEmpty(event.endTime) ?? nonEmpty(tournament?.endTime),
            status: event.status,
            currentRoundText: event.currentRoundText,
            latestResultText: event.latestResultText,
            matchCount: view.matches.count,
            sortTime: event.sortTime,
            published: event.published,
            publishedAt: event.publishedAt,
            createdAt: event.createdAt,
            updatedAt: event.updatedAt
        )

        let card = SaiXunRules.normalizeSaiXunCard(merged, now: now, calendar: calendar)
        let rounds = SaiXunRules.buildSaiXunDetailRounds(view.matches, now: now, calendar: calendar)
        return SaiXunDetail(
            card: card,
            dateText: SaiXunRules.formatEventDateRange(merged.startDate, merged.endDate, calendar: calendar),
            timeText: SaiXunRules.formatEventTimeRange(
                merged.startTime,
                merged.endTime,
                now: now,
                calendar: calendar
            ),
            showTime: !SaiXunText.trimmed(merged.startTime).isEmpty
                || !SaiXunText.trimmed(merged.endTime).isEmpty,
            regionText: SaiXunRules.buildEventRegionText(merged),
            rounds: rounds,
            scheduleTabs: SaiXunRules.buildSaiXunScheduleTabs(rounds),
            champion: SaiXunRules.resolveSaiXunChampion(rounds)
        )
    }

    private static func nonEmpty(_ value: String?) -> String? {
        let text = SaiXunText.trimmed(value)
        return text.isEmpty ? nil : text
    }
}

/// 赛事详情状态。
///
/// - 访客可读；只采纳最新一次请求，退出或切号后的迟到结果不得改动状态。
/// - 定时刷新只重算展示（开赛时间、进行中状态），不重新请求；进行中的赛事才需要。
@MainActor
final class SaiXunDetailModel: ObservableObject {
    enum Phase: Equatable {
        case idle
        case loading
        case loaded
        case failed(LoadFailure)
    }

    /// 小程序 `setInterval(..., 30 * 1000)`。
    static let autoRefreshInterval: TimeInterval = 30

    @Published private(set) var phase: Phase = .idle
    @Published private(set) var detail: SaiXunDetail?
    @Published private(set) var isRefreshing = false
    /// 已有数据时刷新失败的提示；内容保持不变。
    @Published private(set) var refreshError: String?
    @Published private(set) var selectedScheduleTabKey = ""
    @Published private(set) var scheduleRenderLimit = SaiXunRules.schedulePageSize
    /// 最近一次重算展示的时刻，供测试观察定时刷新。
    private(set) var lastRenderedAt: Date?

    private let session: SessionStore
    private let api: EventNewsAPI
    private let eventId: Int64
    private let now: () -> Date
    private let interval: TimeInterval
    private let sleep: (TimeInterval) async throws -> Void
    private var payload: EventNewsView?
    private var owner: SessionIdentity?
    private var requestToken = 0

    init(
        session: SessionStore,
        eventId: Int64,
        now: @escaping () -> Date = Date.init,
        interval: TimeInterval = SaiXunDetailModel.autoRefreshInterval,
        sleep: @escaping (TimeInterval) async throws -> Void = { try await Task.sleep(for: .seconds($0)) }
    ) {
        self.session = session
        self.api = EventNewsAPI(session: session)
        self.eventId = eventId
        self.now = now
        self.interval = interval
        self.sleep = sleep
        self.owner = session.identity
    }

    var isLive: Bool {
        detail?.card.statusTone == .live
    }

    /// 定时重渲染后保留用户手动选择的栏目；比赛状态随时间变化，场次会在两栏之间移动。
    var activeScheduleTab: SaiXunScheduleTab {
        let tabs = detail?.scheduleTabs ?? []
        let key = SaiXunRules.resolveSaiXunScheduleTabKey(tabs, selectedKey: selectedScheduleTabKey)
        return tabs.first { $0.key == key } ?? SaiXunScheduleTab(key: key, label: "", matches: [])
    }

    var visibleDays: [SaiXunScheduleDay] {
        SaiXunRules.buildSaiXunScheduleDays(activeScheduleTab.matches, limit: scheduleRenderLimit)
    }

    var hasMoreScheduleMatches: Bool {
        activeScheduleTab.matches.count > scheduleRenderLimit
    }

    /// 身份变化时清空内容；在途请求因令牌失效而被丢弃。
    func syncIdentity() {
        let current = session.identity
        guard current != owner else { return }
        owner = current
        requestToken += 1
        payload = nil
        detail = nil
        phase = .idle
        isRefreshing = false
        refreshError = nil
        selectedScheduleTabKey = ""
        scheduleRenderLimit = SaiXunRules.schedulePageSize
        lastRenderedAt = nil
    }

    func loadIfNeeded() async {
        syncIdentity()
        guard phase == .idle else { return }
        await fetch()
    }

    func refresh() async {
        syncIdentity()
        guard !isRefreshing else { return }
        await fetch()
    }

    func retry() async {
        syncIdentity()
        await fetch()
    }

    func reloadForIdentityChange() async {
        syncIdentity()
        await fetch()
    }

    func selectScheduleTab(_ key: String) {
        guard key != activeScheduleTab.key else { return }
        selectedScheduleTabKey = key
        scheduleRenderLimit = SaiXunRules.schedulePageSize
    }

    func showMoreScheduleMatches() {
        guard hasMoreScheduleMatches else { return }
        scheduleRenderLimit += SaiXunRules.schedulePageSize
    }

    /// 页面重新出现时按当前时间重算展示，对应小程序 `onShow` 里的 `renderEventView(Date.now())`。
    func refreshDisplay() {
        render(at: now())
    }

    /// 进行中的赛事每 30 秒按当前时间重算一次展示；页面离开（任务取消）或不再进行中即停止。
    func runAutoRefresh() async {
        while !Task.isCancelled {
            guard isLive else { return }
            do {
                try await sleep(interval)
            } catch {
                return
            }
            guard !Task.isCancelled, isLive else { return }
            render(at: now())
        }
    }

    private func fetch() async {
        guard eventId > 0 else {
            phase = .failed(LoadFailure(category: .notFound, message: "未找到该赛讯，请返回赛事列表后重试。"))
            return
        }

        requestToken += 1
        let token = requestToken
        let requestOwner = session.identity
        if detail == nil {
            phase = .loading
        } else {
            isRefreshing = true
        }
        refreshError = nil

        do {
            let view = try await api.view(eventId: eventId)
            guard token == requestToken, session.identity == requestOwner else { return }
            guard let picked = view.pickedPayload,
                  let built = SaiXunDetailRules.build(picked, now: now()) else {
                applyFailure(LoadFailure(category: .notFound, message: "赛事数据不存在或已下线"))
                return
            }
            payload = picked
            detail = built
            lastRenderedAt = now()
            phase = .loaded
            isRefreshing = false
        } catch {
            guard token == requestToken, session.identity == requestOwner else { return }
            isRefreshing = false
            if let apiError = error as? APIError, apiError.kind == .canceled {
                if detail == nil { phase = .idle }
                return
            }
            applyFailure(LoadFailure(error))
        }
    }

    private func render(at date: Date) {
        guard let payload, let rebuilt = SaiXunDetailRules.build(payload, now: date) else { return }
        detail = rebuilt
        lastRenderedAt = date
    }

    private func applyFailure(_ failure: LoadFailure) {
        if detail == nil {
            phase = .failed(failure)
        } else {
            refreshError = failure.message
        }
    }
}
