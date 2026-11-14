import Combine
import Foundation

/// PK 报表页，对应小程序 `subPages/social/pkReport.vue`。
@MainActor
final class PKReportModel: ObservableObject {
    enum Phase: Equatable {
        case loading
        case ready
        case empty
        case failed(LoadFailure)
    }

    let target: FriendHomepageTarget

    @Published private(set) var phase: Phase = .loading
    @Published private(set) var stats: H2HStats?
    @Published private(set) var history: [MatchListItem] = []
    @Published private(set) var opponent: H2HOpponent?
    /// 已有数据时的刷新失败横幅；接口不可用时保留已选数据。
    @Published private(set) var refreshError: LoadFailure?

    private let session: SessionStore
    private let api: PKReportAPI
    private var owner: SessionIdentity?
    private var requestToken = 0
    private var hasLoaded = false

    init(session: SessionStore, target: FriendHomepageTarget, api: PKReportAPI? = nil) {
        self.session = session
        self.target = target
        self.api = api ?? PKReportAPI(session: session)
    }

    // MARK: - 展示

    var myName: String {
        let name = (session.user?.nickname ?? "").trimmingCharacters(in: .whitespacesAndNewlines)
        return name.isEmpty ? "我" : name
    }

    var myAvatar: String { session.user?.avatar ?? "" }

    var opponentName: String {
        let name = (opponent?.name ?? target.name).trimmingCharacters(in: .whitespacesAndNewlines)
        return name.isEmpty ? "对手" : name
    }

    var opponentAvatar: String {
        let avatar = opponent?.avatar ?? target.avatar
        return avatar.isEmpty ? target.avatar : avatar
    }

    var hero: PKReportHero {
        PKReportRules.hero(stats: stats, opponentName: opponentName, history: history)
    }

    var evidenceList: [String] {
        PKReportRules.evidence(stats: stats, history: history)
    }

    var latestMatchSummary: String? {
        PKReportRules.latestMatchSummary(history.first)
    }

    var totalMatches: Int { stats?.totalMatches ?? 0 }

    var gameTypeLabel: String {
        guard let label = history.first?.gameTypeName, !label.isEmpty else {
            return PKReportRules.defaultGameTypeLabel
        }
        return label
    }

    var posterPayload: PKPosterPayload {
        PKReportRules.posterPayload(
            hero: hero,
            evidence: evidenceList,
            stats: stats,
            userName: myName,
            opponentName: opponentName,
            gameTypeLabel: gameTypeLabel
        )
    }

    var shareSummary: String {
        PKReportRules.shareSummary(hero: hero, evidence: evidenceList, totalMatches: totalMatches)
    }

    var posterEmptyText: String {
        PKReportRules.posterEmptyText(totalMatches: totalMatches)
    }

    var failureFeedback: LoadFailure.Feedback? {
        guard case .failed(let failure) = phase else { return nil }
        return failure.feedback(resource: "PK 报表")
    }

    // MARK: - 读取

    func onAppear() async {
        syncIdentity()
        guard session.identity != nil else {
            phase = .failed(LoadFailure(category: .session, message: "登录后查看 PK 报表"))
            return
        }
        guard !hasLoaded else { return }
        await load()
    }

    func retry() async {
        syncIdentity()
        await load()
    }

    private func load() async {
        guard session.identity != nil else { return }
        requestToken += 1
        let token = requestToken
        let requestOwner = session.identity
        if !hasLoaded { phase = .loading }

        do {
            let response = try await api.overview(opponentId: target.userId, opponentName: target.name)
            guard token == requestToken, session.identity == requestOwner else { return }

            if response.hidden {
                opponent = nil
                stats = nil
                history = []
                hasLoaded = true
                refreshError = nil
                phase = .empty
                return
            }
            if response.availability?["stats"] == false || response.availability?["history"] == false {
                let failure = LoadFailure(category: .business, message: PKReportRules.partialFailureMessage)
                if hasLoaded {
                    refreshError = failure
                } else {
                    phase = .failed(failure)
                }
                return
            }

            opponent = response.opponent
            stats = response.stats
            history = response.list ?? []
            hasLoaded = true
            refreshError = nil
            phase = (stats?.totalMatches ?? 0) > 0 ? .ready : .empty
        } catch {
            guard token == requestToken, session.identity == requestOwner else { return }
            let failure = LoadFailure(error)
            if hasLoaded {
                refreshError = failure
            } else {
                phase = .failed(failure)
            }
        }
    }

    /// 身份变化时作废在途请求并清空数据。
    private func syncIdentity() {
        let current = session.identity
        guard current != owner else { return }
        owner = current
        requestToken += 1
        hasLoaded = false
        stats = nil
        history = []
        opponent = nil
        refreshError = nil
        phase = .loading
    }
}
