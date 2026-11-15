import Combine
import Foundation

/// 赛季页状态：登录用户读概览，访客只读公开的当前赛季与排行榜。
///
/// - 只采纳最新一次请求；切换球种、退出或切号后，在途结果一律丢弃。
/// - 赛季信息与球种无关，切换球种时保留卡片，只替换下方记录与榜单。
@MainActor
final class SeasonModel: ObservableObject {
    enum Phase: Equatable {
        case idle
        case loading
        case ready
        case failed(LoadFailure)
    }

    @Published private(set) var phase: Phase = .idle
    @Published private(set) var season: SeasonInfo?
    @Published private(set) var seasonState = "not_started"
    @Published private(set) var myRecord: SeasonRecordInfo?
    @Published private(set) var leaderboard: [SeasonLeaderboardItem] = []
    @Published private(set) var recordUnavailable = false
    @Published private(set) var leaderboardUnavailable = false
    @Published private(set) var myRank = 0
    @Published private(set) var myRankAvailable = false
    @Published private(set) var currentGameType: Int
    @Published private(set) var isRefreshing = false
    @Published private(set) var isSwitchingGameType = false
    @Published private(set) var isLoadingMore = false
    /// 已有内容时刷新 / 切换球种失败的提示；赛季卡片保持不变。
    @Published private(set) var refreshFailure: LoadFailure?

    private(set) var page = 1
    private(set) var hasMoreLeaderboard = false

    private let session: SessionStore
    private let api: SeasonAPI
    private var owner: SessionIdentity?
    private var requestToken = 0

    init(session: SessionStore, gameType: Int = 3) {
        self.session = session
        self.api = SeasonAPI(session: session)
        self.currentGameType = GameTypeCatalog.isSupported(gameType) ? gameType : 3
        self.owner = session.identity
    }

    // MARK: - 展示值

    var isGuest: Bool { session.identity == nil }
    var isActiveSeason: Bool { seasonState == "active" && season != nil }
    var currentGameTypeLabel: String { GameTypeCatalog.label(for: currentGameType) }

    var seasonStateView: HonorWallSeasonState {
        let current = season.map {
            HonorWallSeasonInfo(
                seasonId: $0.id,
                seasonName: $0.name,
                startDate: $0.startDate,
                endDate: $0.endDate,
                status: $0.status,
                gameType: currentGameType,
                challenges: []
            )
        }
        return HonorWallRules.currentSeasonState(current, seasonState: seasonState)
    }

    var seasonTimeline: SeasonLifecycleRules.Timeline {
        SeasonLifecycleRules.resolveTimeline(season)
    }

    var statusText: String {
        switch season?.status {
        case 0: "未开始"
        case 1: "进行中"
        case 2: "已结束"
        default: "未知"
        }
    }

    var qualification: SeasonQualification {
        SeasonQualificationRules.qualification(season: season, record: myRecord)
    }

    var netChangeText: String { SeasonQualificationRules.netChange(myRecord) }

    var winRate: Int {
        guard let record = myRecord, record.matchesPlayed > 0 else { return 0 }
        return Int((Double(record.wins) / Double(record.matchesPlayed) * 100).rounded())
    }

    var leaderboardRows: [SeasonLeaderboardRow] {
        leaderboard.map { SeasonLeaderboardRow(item: $0, rank: SeasonQualificationRules.leaderboardRank($0)) }
    }

    var seasonDataUnavailable: Bool { recordUnavailable && leaderboardUnavailable }

    // MARK: - 读取

    /// 身份变化时清空并回到可重新读取的状态；在途请求因令牌失效而被丢弃。
    func syncIdentity() {
        let current = session.identity
        guard current != owner else { return }
        owner = current
        requestToken += 1
        reset()
    }

    func loadIfNeeded() async {
        syncIdentity()
        guard phase == .idle else { return }
        await fetch(isSwitching: false)
    }

    func retry() async {
        syncIdentity()
        guard !isBusy else { return }
        await fetch(isSwitching: false)
    }

    /// 切球种回到第一页；赛季信息与球种标签保持不动。
    func selectGameType(_ gameType: Int) async {
        guard gameType != currentGameType, GameTypeCatalog.isSupported(gameType) else { return }
        currentGameType = gameType
        clearGameTypeData()
        page = 1
        hasMoreLeaderboard = true
        isSwitchingGameType = true
        await fetch(isSwitching: true)
    }

    func loadMoreLeaderboard() async {
        guard let seasonId = season?.id, seasonId > 0,
              isActiveSeason, hasMoreLeaderboard, !isBusy else { return }
        let requestOwner = session.identity
        requestToken += 1
        let token = requestToken
        let requestedGameType = currentGameType
        let requestedPage = page + 1
        isLoadingMore = true

        do {
            let response = try await api.leaderboard(
                seasonId: seasonId,
                gameType: requestedGameType,
                page: requestedPage
            )
            guard token == requestToken,
                  session.identity == requestOwner,
                  currentGameType == requestedGameType else { return }
            leaderboard = SeasonQualificationRules.mergeLeaderboard(leaderboard, response.list)
            page = requestedPage
            hasMoreLeaderboard = Int64(requestedPage * SeasonAPI.leaderboardPageSize) < response.total
            leaderboardUnavailable = false
            isLoadingMore = false
        } catch {
            guard token == requestToken,
                  session.identity == requestOwner,
                  currentGameType == requestedGameType else { return }
            isLoadingMore = false
            if isCancellation(error) { return }
            refreshFailure = LoadFailure(error)
        }
    }

    // MARK: - 私有

    private var isBusy: Bool {
        phase == .loading || isRefreshing || isSwitchingGameType || isLoadingMore
    }

    private func fetch(isSwitching: Bool) async {
        guard let requestOwner = session.identity else {
            await fetchPublic(isSwitching: isSwitching)
            return
        }
        requestToken += 1
        let token = requestToken
        let requestedGameType = currentGameType

        if isSwitching {
            // 只替换下方数据卡片，整页与赛季卡片保持可见。
        } else if season == nil {
            phase = .loading
        } else {
            isRefreshing = true
        }
        refreshFailure = nil

        do {
            let response = try await api.overview(gameType: requestedGameType)
            guard token == requestToken,
                  session.identity == requestOwner,
                  currentGameType == requestedGameType else { return }

            season = response.season
            seasonState = response.seasonState.isEmpty
                ? (response.season == nil ? "not_started" : "active")
                : response.seasonState
            isRefreshing = false
            isSwitchingGameType = false

            guard isActiveSeason else {
                clearGameTypeData()
                hasMoreLeaderboard = false
                phase = .ready
                return
            }

            let availability = response.availability ?? [:]
            recordUnavailable = availability["record"] == false
            leaderboardUnavailable = availability["leaderboard"] == false
            myRecord = recordUnavailable ? nil : response.record
            myRankAvailable = !recordUnavailable && availability["my_rank"] != false
            myRank = myRankAvailable ? max(0, response.myRank) : 0
            leaderboard = leaderboardUnavailable ? [] : response.leaderboard
            page = 1
            hasMoreLeaderboard = !leaderboardUnavailable
                && leaderboard.count >= SeasonAPI.leaderboardPageSize
            phase = .ready
        } catch {
            guard token == requestToken,
                  session.identity == requestOwner,
                  currentGameType == requestedGameType else { return }
            isRefreshing = false
            isSwitchingGameType = false
            if isCancellation(error) {
                if phase == .loading { phase = .idle }
                return
            }

            let failure = LoadFailure(error)
            if isSwitching || season != nil {
                // 切换球种时数据区已清空，失败必须明确标出，不能被当作「未参加排位」。
                if isSwitching {
                    recordUnavailable = true
                    leaderboardUnavailable = true
                }
                if phase == .loading { phase = .ready }
                refreshFailure = failure
            } else {
                phase = .failed(failure)
            }
        }
    }

    /// 访客：当前赛季与排行榜公开可读，不读取任何个人数据。
    private func fetchPublic(isSwitching: Bool) async {
        let requestOwner = session.identity
        requestToken += 1
        let token = requestToken
        let requestedGameType = currentGameType

        if isSwitching {
            // 保留赛季卡片，只等下方榜单。
        } else if season == nil {
            phase = .loading
        } else {
            isRefreshing = true
        }
        refreshFailure = nil

        do {
            let current = try await api.current()
            guard token == requestToken,
                  session.identity == requestOwner,
                  currentGameType == requestedGameType else { return }

            season = current.season
            seasonState = current.seasonState.isEmpty
                ? (current.season == nil ? "not_started" : "active")
                : current.seasonState
            clearGameTypeData()
            isRefreshing = false
            isSwitchingGameType = false
            page = 1
            hasMoreLeaderboard = false

            if isActiveSeason, let seasonId = season?.id, seasonId > 0 {
                let response = try await api.leaderboard(
                    seasonId: seasonId,
                    gameType: requestedGameType,
                    page: 1
                )
                guard token == requestToken,
                      session.identity == requestOwner,
                      currentGameType == requestedGameType else { return }
                leaderboard = response.list
                hasMoreLeaderboard = response.list.count >= SeasonAPI.leaderboardPageSize
            }
            phase = .ready
        } catch {
            guard token == requestToken,
                  session.identity == requestOwner,
                  currentGameType == requestedGameType else { return }
            isRefreshing = false
            isSwitchingGameType = false
            if isCancellation(error) {
                if phase == .loading { phase = .idle }
                return
            }

            let failure = LoadFailure(error)
            if phase == .loading {
                phase = .failed(failure)
            } else {
                phase = .ready
                refreshFailure = failure
            }
        }
    }

    private func clearGameTypeData() {
        myRecord = nil
        leaderboard = []
        recordUnavailable = false
        leaderboardUnavailable = false
        myRank = 0
        myRankAvailable = false
    }

    private func reset() {
        phase = .idle
        season = nil
        seasonState = "not_started"
        clearGameTypeData()
        isRefreshing = false
        isSwitchingGameType = false
        isLoadingMore = false
        refreshFailure = nil
        page = 1
        hasMoreLeaderboard = false
    }

    private func isCancellation(_ error: Error) -> Bool {
        if let apiError = error as? APIError, apiError.kind == .canceled { return true }
        return error is CancellationError
    }
}
