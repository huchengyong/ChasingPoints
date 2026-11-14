import Combine
import Foundation

/// VIP 中心页面状态：概览提供会员身份与常玩球馆奖励，段位只用于挑选头像框。
@MainActor
final class MemberCenterModel: ObservableObject {
    static let freshness: TimeInterval = 60

    let overview: ResourceLoader<UserOverview>
    let ranks: ResourceLoader<RankInfoMap>

    @Published private(set) var preferredGameType: Int

    private let session: SessionStore
    private let defaults: UserDefaults
    private let now: () -> Date
    private var cancellables: Set<AnyCancellable> = []

    init(session: SessionStore, defaults: UserDefaults = .standard, now: @escaping () -> Date = Date.init) {
        self.session = session
        self.defaults = defaults
        self.now = now
        self.preferredGameType = GameTypePreference(defaults: defaults).read(userId: session.identity?.userId ?? 0)

        let api = ProfileAPI(session: session)
        overview = ResourceLoader(session: session, now: now) { try await api.overview() }
        ranks = ResourceLoader(session: session, now: now) { try await api.rankInfos() }

        // 两个 loader 不是本对象的 @Published 属性，需要把变更转给页面。
        overview.objectWillChange
            .merge(with: ranks.objectWillChange)
            .sink { [weak self] in self?.objectWillChange.send() }
            .store(in: &cancellables)
    }

    /// 头像框跟随默认球种的当前段位；没有偏好时按中式八球（3）。
    var currentGameType: Int { preferredGameType > 0 ? preferredGameType : 3 }

    var memberStatus: MemberStatus? { overview.value?.availableMemberStatus }

    var rankLevel: Int { ranks.value?[currentGameType]?.level ?? 0 }

    var summary: MemberCenterSummary {
        MemberCenterRules.resolveSummary(memberStatus, now: now(), complianceMode: AppFlags.complianceMode)
    }

    var entryCard: MemberEntryCard {
        MemberCenterRules.resolveEntryCard(memberStatus, now: now(), complianceMode: AppFlags.complianceMode)
    }

    var growthCard: MemberGrowthCard {
        MemberGrowthCard.resolve(memberStatus, now: now())
    }

    var identityPreview: MemberIdentityPreview {
        MemberCenterRules.resolveIdentityPreview(
            memberStatus,
            now: now(),
            rankLevel: rankLevel,
            complianceMode: AppFlags.complianceMode
        )
    }

    var venueRewardClaimable: Bool {
        FavoriteVenueReward.isClaimable(overview.value?.availableFavoriteVenueReward)
    }

    var venueRewardText: String {
        FavoriteVenueReward.formatDuration(
            rewardDays: FavoriteVenueReward.rewardDays(overview.value?.availableFavoriteVenueReward)
        )
    }

    func onAppear() async {
        syncRankGamePreference()
        guard session.identity != nil else {
            overview.syncIdentity()
            ranks.syncIdentity()
            return
        }
        async let overviewLoad: Void = overview.loadIfNeeded(maxAge: Self.freshness)
        async let rankLoad: Void = ranks.loadIfNeeded(maxAge: Self.freshness)
        _ = await (overviewLoad, rankLoad)
    }

    func refresh() async {
        guard session.identity != nil else { return }
        async let overviewRefresh: Void = overview.refresh()
        async let rankRefresh: Void = ranks.refresh()
        _ = await (overviewRefresh, rankRefresh)
    }

    func onIdentityChanged() async {
        overview.syncIdentity()
        ranks.syncIdentity()
        syncRankGamePreference()
        guard session.identity != nil else { return }
        await onAppear()
    }

    /// 设置页改了常用球种或换了账号时，头像框回到新的常用球种。
    private func syncRankGamePreference() {
        preferredGameType = GameTypePreference(defaults: defaults).read(userId: session.identity?.userId ?? 0)
    }
}
