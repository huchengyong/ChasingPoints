import Combine
import Foundation

/// 荣誉墙页面状态：本人按球种读取，好友视图由服务端收窄。
///
/// - 只采纳最新一次请求；切换球种、退出或切号后，在途结果一律丢弃。
/// - 已有内容时刷新或切换失败保留旧内容，只给轻提示。
@MainActor
final class HonorWallModel: ObservableObject {
    @Published private(set) var wall: HonorWallResponse?
    @Published private(set) var isLoading = false
    @Published private(set) var isRefreshing = false
    @Published private(set) var isHistoryLoading = false
    @Published private(set) var failure: LoadFailure?
    /// 需要以轻提示告知用户的刷新或称号操作结果。
    @Published private(set) var notice: String?
    @Published private(set) var currentGameType: Int
    @Published private(set) var activeTab: HonorWallTab
    @Published private(set) var selectedHistorySeasonId: Int64
    @Published private(set) var titleList: [TitleInfo] = []
    @Published private(set) var isTitleListLoading = false
    @Published private(set) var isTitleListLoaded = false
    @Published private(set) var isTitleListFailed = false
    @Published private(set) var isTitleSubmitting = false

    private let session: SessionStore
    private let api: AchievementAPI
    private let targetUserId: Int64
    private let historyPageSize = 20
    private var historyPage = 1
    private var owner: SessionIdentity?
    private var requestToken = 0
    private var titleRequestToken = 0

    init(
        session: SessionStore,
        gameType: Int,
        userId: Int64 = 0,
        tab: HonorWallTab = .career,
        historySeasonId: Int64 = 0
    ) {
        self.session = session
        self.api = AchievementAPI(session: session)
        self.targetUserId = max(0, userId)
        self.currentGameType = HonorWallRules.normalizeGameType(gameType)
        self.activeTab = tab
        self.selectedHistorySeasonId = max(0, historySeasonId)
        self.owner = session.identity
    }

    // MARK: - 展示值

    var hasLoaded: Bool { wall != nil }
    var isInitialLoading: Bool { isLoading && wall == nil }
    var isInitialFailure: Bool { wall == nil && failure != nil && !isLoading && session.identity != nil }

    var loadError: HonorWallLoadError? {
        failure.map { HonorWallRules.loadError(category: $0.category) }
    }

    var isSelf: Bool { wall?.viewerScope != "friend" }
    var tabs: [HonorWallTab] { HonorWallRules.tabs(viewerScope: wall?.viewerScope ?? "self") }
    var currentGameTypeLabel: String { GameTypeCatalog.label(for: currentGameType) }

    var profileName: String {
        if let name = wall?.profile.nickname, !name.isEmpty { return name }
        return isSelf ? "我的荣誉" : "球友"
    }

    var profileKicker: String { isSelf ? "我的荣誉墙" : "好友公开荣誉" }
    var profileNote: String { isSelf ? "生涯永久累计，赛季挑战独立刷新" : "仅展示已获得成就与永久荣誉" }

    var equippedTitleName: String { wall?.equippedTitle?.titleName ?? "暂未佩戴称号" }

    var equippedTitleSource: String {
        guard let title = wall?.equippedTitle else {
            return isSelf ? "选择一个已获称号进行展示" : "TA 暂未佩戴称号"
        }
        return AchievementPageRules.titleSourceDescription(title)
    }

    var activeTitleActionText: String { wall?.equippedTitle == nil ? "选择" : "更换" }

    var summaryItems: [HonorWallSummaryItem] {
        HonorWallRules.careerSummaryItems(
            wall?.summary ?? emptySummary,
            gameType: currentGameType,
            gameTypeLabel: currentGameTypeLabel
        )
    }

    var recentHonors: [HonorWallRecentHonor] {
        (wall?.recentHonors ?? []).map(HonorWallRules.recentHonor)
    }

    var universalGroups: [AchievementGroup] {
        AchievementPageRules.groupUniversalAchievements(wall?.careerAchievements ?? [])
    }

    var specialtyAchievements: [AchievementDef] {
        AchievementPageRules.filterSpecialtyAchievements(
            wall?.careerAchievements ?? [],
            gameType: currentGameType,
            unlockedOnly: !isSelf
        )
    }

    var upcomingSection: HonorWallUpcomingSection {
        HonorWallRules.upcomingSection(
            wall?.careerAchievements ?? [],
            gameType: currentGameType,
            viewerScope: wall?.viewerScope ?? "self"
        )
    }

    var upcomingEmptyText: String {
        HonorWallRules.upcomingEmptyText(upcomingSection, gameTypeLabel: currentGameTypeLabel)
    }

    var currentSeasonState: HonorWallSeasonState {
        HonorWallRules.currentSeasonState(wall?.currentSeason, seasonState: wall?.seasonState ?? "")
    }

    var currentSeasonName: String { wall?.currentSeason?.seasonName ?? "当前赛季" }
    var seasonDateText: String { HonorWallRules.seasonDateText(wall?.currentSeason) }

    var currentChallenges: [HonorWallChallengeItem] {
        (wall?.currentSeason?.challenges ?? []).map(HonorWallRules.challengeItem)
    }

    var archivedChallenges: [HonorWallChallengeItem] {
        (wall?.history.challengeRecords ?? []).map(HonorWallRules.challengeItem)
    }

    var historyHonors: [HonorItem] { wall?.history.honors ?? [] }
    var historyTotalText: String { String(wall?.history.total ?? 0) }
    var historySeasonOptions: [HonorWallSeasonOption] {
        HonorWallRules.historySeasonOptions(historyHonors)
    }

    var hasMoreHistory: Bool { Int64(historyHonors.count) < (wall?.history.total ?? 0) }
    var currentTitleId: Int64 { wall?.equippedTitle?.id ?? 0 }
    var titleOptions: [TitleInfo] { AchievementPageRules.sortTitleOptions(titleList) }

    // MARK: - 读取

    func loadIfNeeded() async {
        syncIdentity()
        guard session.identity != nil, wall == nil, !isLoading else { return }
        await fetch(appendHistory: false)
    }

    func retry() async {
        syncIdentity()
        guard session.identity != nil, !isLoading, !isRefreshing else { return }
        await fetch(appendHistory: false)
    }

    func refresh() async {
        syncIdentity()
        guard session.identity != nil, !isLoading, !isRefreshing else { return }
        historyPage = 1
        await fetch(appendHistory: false)
    }

    func switchTab(_ tab: HonorWallTab) {
        activeTab = tab
    }

    func selectGameType(_ gameType: Int) async {
        let normalized = HonorWallRules.normalizeGameType(gameType)
        guard normalized != currentGameType else { return }
        currentGameType = normalized
        historyPage = 1
        failure = nil
        await fetch(appendHistory: false)
    }

    func selectHistorySeason(_ seasonId: Int64) async {
        guard seasonId != selectedHistorySeasonId else { return }
        selectedHistorySeasonId = seasonId
        historyPage = 1
        failure = nil
        await fetch(appendHistory: false)
    }

    func loadMoreHistory() async {
        guard hasMoreHistory, !isHistoryLoading, !isLoading, !isRefreshing else { return }
        historyPage += 1
        await fetch(appendHistory: true)
    }

    func clearNotice() {
        notice = nil
    }

    // MARK: - 称号

    func prepareTitleSelector() async {
        guard isSelf else { return }
        if isTitleListLoaded {
            syncEquippedTitleFromList()
            return
        }
        await loadTitles()
    }

    func loadTitles() async {
        guard !isTitleListLoading, let requestOwner = session.identity else { return }
        titleRequestToken += 1
        let token = titleRequestToken
        isTitleListLoading = true
        isTitleListFailed = false

        do {
            let response = try await api.titles()
            guard token == titleRequestToken, session.identity == requestOwner else { return }
            titleList = response.list
            isTitleListLoaded = true
            isTitleListLoading = false
            syncEquippedTitleFromList()
        } catch {
            guard token == titleRequestToken, session.identity == requestOwner else { return }
            isTitleListLoading = false
            isTitleListFailed = true
        }
    }

    /// 返回是否关闭称号选择面板：同选项或提交成功关闭，提交失败留在面板内。
    func selectTitle(_ title: TitleInfo?) async -> Bool {
        guard !isTitleSubmitting else { return false }
        let selection = AchievementPageRules.resolveTitleSelection(
            currentTitleId: currentTitleId,
            selectedTitleId: title?.id ?? 0
        )
        let titleId: Int64
        let equip: Bool
        switch selection {
        case .close:
            return true
        case .equip(let id):
            titleId = id
            equip = true
        case .unequip(let id):
            titleId = id
            equip = false
        }

        let requestOwner = session.identity
        isTitleSubmitting = true
        defer { isTitleSubmitting = false }

        do {
            try await api.equipTitle(titleId: titleId, equip: equip)
            guard session.identity == requestOwner else { return false }
            applyTitleSelection(selection)
            return true
        } catch {
            guard session.identity == requestOwner else { return false }
            notice = error.localizedDescription.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty
                ? "称号切换失败"
                : error.localizedDescription
            return false
        }
    }

    func selectNoTitle() async -> Bool {
        await selectTitle(nil)
    }

    // MARK: - 私有

    private var emptySummary: HonorWallSummary {
        HonorWallSummary(
            careerUnlocked: 0,
            careerTotal: 0,
            universalUnlocked: 0,
            universalTotal: 0,
            specialtyGameType: 3,
            specialtyUnlocked: 0,
            specialtyTotal: 0,
            seasonHonors: 0,
            tournamentHonors: 0
        )
    }

    private func fetch(appendHistory: Bool) async {
        guard let requestOwner = session.identity else { return }
        requestToken += 1
        let token = requestToken
        let requestedGameType = currentGameType
        let requestedSeasonId = selectedHistorySeasonId
        let requestedUserId = targetUserId

        if appendHistory {
            isHistoryLoading = true
        } else if wall == nil {
            isLoading = true
        } else {
            isRefreshing = true
        }
        failure = nil

        do {
            let response = try await api.honorWall(
                userId: requestedUserId,
                gameType: requestedGameType,
                historySeasonId: requestedSeasonId,
                historyPage: historyPage,
                historyPageSize: historyPageSize
            )
            guard token == requestToken,
                  session.identity == requestOwner,
                  currentGameType == requestedGameType,
                  selectedHistorySeasonId == requestedSeasonId else { return }

            var next = response
            if appendHistory, let existing = wall {
                next.history.honors = existing.history.honors + response.history.honors
            }
            wall = next
            if !tabs.contains(activeTab) { activeTab = .career }
            isLoading = false
            isRefreshing = false
            isHistoryLoading = false
        } catch {
            guard token == requestToken,
                  session.identity == requestOwner,
                  currentGameType == requestedGameType,
                  selectedHistorySeasonId == requestedSeasonId else { return }
            isLoading = false
            isRefreshing = false
            isHistoryLoading = false
            if isCancellation(error) { return }
            if appendHistory { historyPage = max(1, historyPage - 1) }

            let loadFailure = LoadFailure(error)
            // 全局会话失效流程已接管清理与重新登录，页面不重复展示失败态。
            guard loadFailure.category != .session else { return }
            if wall == nil {
                failure = loadFailure
            } else {
                let message = error.localizedDescription.trimmingCharacters(in: .whitespacesAndNewlines)
                notice = message.isEmpty ? "荣誉墙更新失败" : message
            }
        }
    }

    private func syncEquippedTitleFromList() {
        wall?.equippedTitle = titleList.first { $0.equipped }
    }

    private func applyTitleSelection(_ selection: TitleSelectionAction) {
        switch selection {
        case .close:
            break
        case .equip(let titleId):
            titleList = titleList.map { title in
                var copy = title
                copy.equipped = title.id == titleId
                return copy
            }
            let equipped = titleList.first { $0.equipped }
            wall?.equippedTitle = equipped
            notice = "已佩戴「\(equipped?.titleName ?? "")」"
        case .unequip:
            titleList = titleList.map { title in
                var copy = title
                copy.equipped = false
                return copy
            }
            wall?.equippedTitle = nil
            notice = "已停止展示称号"
        }
    }

    /// 身份变化时作废在途请求并清空本页状态；相同身份不做任何事。
    func syncIdentity() {
        let current = session.identity
        guard current != owner else { return }
        owner = current
        requestToken += 1
        titleRequestToken += 1
        wall = nil
        isLoading = false
        isRefreshing = false
        isHistoryLoading = false
        failure = nil
        notice = nil
        historyPage = 1
        titleList = []
        isTitleListLoading = false
        isTitleListLoaded = false
        isTitleListFailed = false
        isTitleSubmitting = false
    }

    private func isCancellation(_ error: Error) -> Bool {
        if let apiError = error as? APIError, apiError.kind == .canceled { return true }
        return error is CancellationError
    }
}
