import Combine
import Foundation

/// 段位说明页的读取状态。
///
/// - 只采纳最新一次请求；切换球种、退出或切号后，在途结果一律丢弃。
/// - 已有内容时切换球种保留旧内容并标记刷新中，新数据到了再替换。
@MainActor
final class RankExplainModel: ObservableObject {
    @Published private(set) var currentGameType: Int
    @Published private(set) var loadedGameType: Int
    @Published private(set) var rankInfo: RankInfo?
    @Published private(set) var rankList: [RankItem] = []
    @Published private(set) var rulesMeta = RankRulesMeta.legacy
    @Published private(set) var isFetching = false
    @Published private(set) var hasLoadedOnce = false
    @Published private(set) var hasLoadError = false
    /// 已有内容时刷新失败的轻提示。
    @Published private(set) var notice: String?

    private let session: SessionStore
    private let now: () -> Date
    private var owner: SessionIdentity?
    private var requestToken = 0

    init(session: SessionStore, gameType: Int, now: @escaping () -> Date = Date.init) {
        // 不支持的球种按页面默认球种读取。
        let initial = GameTypeCatalog.isSupported(gameType) ? gameType : 3
        self.session = session
        self.currentGameType = initial
        self.loadedGameType = initial
        self.now = now
        self.owner = session.identity
    }

    var isInitialLoading: Bool { isFetching && !hasLoadedOnce }
    var isRefreshing: Bool { isFetching && hasLoadedOnce }
    var isInitialError: Bool { !hasLoadedOnce && hasLoadError && !isFetching }

    var heroModel: RankHeroModel? {
        RankExplainRules.heroModel(rankInfo: rankInfo, rankList: rankList)
    }

    var growthPath: [RankGrowthStep] {
        RankExplainRules.growthPath(
            rankList: rankList,
            currentLevel: rankInfo?.level ?? 0,
            rankScore: rankInfo?.rankScore ?? 0
        )
    }

    var currentGameTypeLabel: String {
        GameTypeCatalog.label(for: loadedGameType)
    }

    var rankRules: RankRules {
        RankExplainRules.rules(
            rulesVersion: rulesMeta.rulesVersion,
            rulesEffectiveAt: rulesMeta.rulesEffectiveAt,
            now: now()
        )
    }

    /// 身份变化时清空内容；在途请求因令牌失效而被丢弃。
    func syncIdentity() {
        let current = session.identity
        guard current != owner else { return }
        owner = current
        requestToken += 1
        rankInfo = nil
        rankList = []
        rulesMeta = RankRulesMeta.legacy
        isFetching = false
        hasLoadedOnce = false
        hasLoadError = false
        notice = nil
    }

    func loadIfNeeded() async {
        syncIdentity()
        guard session.identity != nil, !isFetching, !hasLoadedOnce, !hasLoadError else { return }
        await fetch()
    }

    /// 首次加载失败后重新加载。
    func retry() async {
        syncIdentity()
        guard session.identity != nil, !isFetching else { return }
        hasLoadError = false
        await fetch()
    }

    /// 切换球种：保留旧内容，Tab 显示 pending 态，成功后整体替换。
    func selectGameType(_ gameType: Int) async {
        guard gameType != currentGameType else { return }
        currentGameType = gameType
        guard session.identity != nil else { return }
        await fetch()
    }

    func reloadForIdentityChange() async {
        syncIdentity()
        guard session.identity != nil else { return }
        await fetch()
    }

    func clearNotice() {
        notice = nil
    }

    private func fetch() async {
        guard let requestOwner = session.identity else { return }
        requestToken += 1
        let token = requestToken
        let requestedGameType = currentGameType
        isFetching = true
        hasLoadError = false

        let profile = ProfileAPI(session: session)
        let api = RankExplainAPI(session: session)
        async let rankInfoTask = profile.rankInfos()
        async let configsTask = api.configs()

        do {
            let map = try await rankInfoTask
            let configs = try await configsTask
            guard token == requestToken, session.identity == requestOwner else { return }
            guard let nextRankInfo = map[requestedGameType] else {
                throw APIError(message: "段位数据加载失败", kind: .invalidResponse)
            }

            rankInfo = nextRankInfo
            rankList = configs.rankList
            rulesMeta = configs.rules
            loadedGameType = requestedGameType
            hasLoadedOnce = true
            hasLoadError = false
            isFetching = false
            notice = nil
        } catch {
            guard token == requestToken, session.identity == requestOwner else { return }
            isFetching = false
            if let apiError = error as? APIError, apiError.kind == .canceled { return }

            if hasLoadedOnce {
                // 已有成功内容：保留旧内容，回到已加载的球种并轻提示。
                currentGameType = loadedGameType
                let message = error.localizedDescription
                notice = message.isEmpty ? "获取数据失败" : message
            } else {
                hasLoadError = true
            }
        }
    }
}
