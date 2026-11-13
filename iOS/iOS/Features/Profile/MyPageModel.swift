import Combine
import Foundation

/// 「我的」页的读取与展示状态。概览与段位各自独立读取，互不阻塞。
@MainActor
final class MyPageModel: ObservableObject {
    /// 对应小程序 USER_OVERVIEW_CACHE_TTL。
    static let freshness: TimeInterval = 60

    let overview: ResourceLoader<UserOverview>
    let ranks: ResourceLoader<RankInfoMap>

    /// 页内临时切换的球种；0 表示跟随默认。
    @Published var selectedRankGameType = 0
    @Published private(set) var preferredGameType = 0
    @Published private(set) var floatRewardVisible = false

    enum CareerState: Equatable {
        case loading
        case ready
        case empty
        case error
    }

    private let session: SessionStore
    private let defaults: UserDefaults
    private let preference: GameTypePreference
    private let now: () -> Date
    private var cancellables: Set<AnyCancellable> = []
    private var rankPreferenceOwnerId: Int64 = 0

    init(session: SessionStore, defaults: UserDefaults = .standard, now: @escaping () -> Date = Date.init) {
        self.session = session
        self.defaults = defaults
        self.now = now

        let preference = GameTypePreference(defaults: defaults)
        self.preference = preference
        let api = ProfileAPI(session: session)
        overview = ResourceLoader(session: session, now: now) { try await api.overview() }
        ranks = ResourceLoader(session: session, now: now) { try await api.rankInfos() }

        let ownerId = session.identity?.userId ?? 0
        rankPreferenceOwnerId = ownerId
        preferredGameType = preference.read(userId: ownerId)

        // 两个 loader 不是本对象的 @Published 属性，需要把它们的变更转给页面。
        overview.objectWillChange
            .merge(with: ranks.objectWillChange)
            .sink { [weak self] in self?.objectWillChange.send() }
            .store(in: &cancellables)
        // 每次概览读取完成都重新判断悬浮奖励。
        overview.$phase
            .sink { [weak self] _ in self?.syncFloatRewardVisibility() }
            .store(in: &cancellables)
    }

    /// 页面出现：同步常用球种偏好；已登录时按新鲜度读取概览与段位。
    func onAppear() async {
        syncRankGamePreference()
        guard session.identity != nil else {
            resetForGuest()
            return
        }
        await loadHomepage()
    }

    /// 下拉刷新：两个资源都强制刷新。
    func refresh() async {
        guard session.identity != nil else {
            resetForGuest()
            return
        }
        async let overviewRefresh: Void = overview.refresh()
        async let rankRefresh: Void = ranks.refresh()
        _ = await (overviewRefresh, rankRefresh)
        syncFloatRewardVisibility()
    }

    /// 身份变化：清空页内状态并重新同步偏好。
    func onIdentityChanged() async {
        overview.syncIdentity()
        ranks.syncIdentity()
        selectedRankGameType = 0
        floatRewardVisible = false
        syncRankGamePreference()
        guard session.identity != nil else { return }
        await loadHomepage()
    }

    /// 在段位卡里选定常用球种（与设置页共用同一偏好）。
    func pickPreferredGameType(_ gameType: Int) {
        preferredGameType = preference.save(userId: session.identity?.userId ?? 0, gameType: gameType)
        selectedRankGameType = 0
    }

    /// 关闭悬浮奖励入口：按当前状态记 3 天稍后提醒。
    func snoozeFloatReward() {
        guard let status = overview.value?.availableFavoriteVenueReward,
              let userId = session.identity?.userId, userId > 0 else { return }
        defaults.set(
            FavoriteVenueReward.snoozeUntil(from: now()).timeIntervalSince1970,
            forKey: FavoriteVenueReward.floatSnoozeKey(userId: userId, status: status.status)
        )
        floatRewardVisible = false
    }

    // MARK: - 推导出的展示值

    var rankGameView: RankGameTypeView {
        UserHomepageRules.rankGameTypeView(
            preferredGameType: preferredGameType,
            rankInfoMap: ranks.value ?? [:]
        )
    }

    var currentRankGameType: Int {
        let view = rankGameView
        return view.tabs.contains { $0.value == selectedRankGameType }
            ? selectedRankGameType
            : view.primaryGameType
    }

    var showRankGamePicker: Bool {
        ranks.value != nil && rankGameView.needsPick
    }

    var rankCard: RankCardView {
        UserHomepageRules.rankCard(ranks.value?[currentRankGameType])
    }

    var isRankSkeleton: Bool {
        ranks.value == nil && ranks.isLoading
    }

    var memberIdentity: MemberIdentity {
        UserHomepageRules.selfMemberIdentity(
            isLoggedIn: session.identity != nil,
            memberStatus: overview.value?.availableMemberStatus,
            rankInfo: ranks.value?[currentRankGameType],
            rankLoading: ranks.isLoading && ranks.value == nil,
            now: now()
        )
    }

    var profileLevelText: String {
        UserHomepageRules.profileLevelText(
            memberStatus: overview.value?.availableMemberStatus,
            identity: memberIdentity,
            isLoading: isOverviewLoading,
            now: now()
        )
    }

    var reputationChip: ReputationChip {
        UserHomepageRules.reputationChip(overview.value?.availableReputation, isLoading: isOverviewLoading)
    }

    var careerState: CareerState {
        if let stats = overview.value?.availableStats {
            return stats.totalMatches > 0 ? .ready : .empty
        }
        switch overview.phase {
        case .failed, .loaded: return .error
        case .idle, .loading: return .loading
        }
    }

    var coreMetrics: [CoreMetric] {
        let stats = overview.value?.availableStats
        return UserHomepageRules.coreMetrics(stats: stats, isLoading: stats == nil)
    }

    var memberEntry: MemberEntry {
        UserHomepageRules.memberEntry(overview.value?.availableMemberStatus, now: now())
    }

    var compactRewardEntry: CompactRewardEntry {
        UserHomepageRules.compactRewardEntry(overview.value?.availableFavoriteVenueReward)
    }

    // MARK: - 私有

    private var isOverviewLoading: Bool {
        overview.isLoading || overview.isRefreshing
    }

    private func loadHomepage() async {
        async let overviewLoad: Void = overview.loadIfNeeded(maxAge: Self.freshness)
        async let rankLoad: Void = ranks.loadIfNeeded(maxAge: Self.freshness)
        _ = await (overviewLoad, rankLoad)
        syncFloatRewardVisibility()
    }

    private func resetForGuest() {
        overview.syncIdentity()
        ranks.syncIdentity()
        selectedRankGameType = 0
        floatRewardVisible = false
    }

    /// 设置页或开局弹窗改了常用球种、或换了账号时，段位卡回到新的常用球种。
    private func syncRankGamePreference() {
        let ownerId = session.identity?.userId ?? 0
        let preferred = preference.read(userId: ownerId)
        guard ownerId != rankPreferenceOwnerId || preferred != preferredGameType else { return }
        rankPreferenceOwnerId = ownerId
        preferredGameType = preferred
        selectedRankGameType = 0
    }

    private func syncFloatRewardVisibility() {
        guard let status = overview.value?.availableFavoriteVenueReward, status.enabled,
              let userId = session.identity?.userId, userId > 0 else {
            floatRewardVisible = false
            return
        }
        let state = status.status.trimmingCharacters(in: .whitespacesAndNewlines)
        guard state == "not_started" || state == "rejected" else {
            floatRewardVisible = false
            return
        }
        let key = FavoriteVenueReward.floatSnoozeKey(userId: userId, status: state)
        if FavoriteVenueReward.isSnoozed(until: snoozeUntil(key), now: now()) {
            floatRewardVisible = false
            return
        }
        // 过期的稍后提醒记录不再保留。
        if defaults.object(forKey: key) != nil {
            defaults.removeObject(forKey: key)
        }
        floatRewardVisible = true
    }

    private func snoozeUntil(_ key: String) -> Date? {
        guard let value = defaults.object(forKey: key) as? Double else { return nil }
        return Date(timeIntervalSince1970: value)
    }
}
