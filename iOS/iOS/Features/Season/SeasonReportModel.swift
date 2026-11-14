import Combine
import Foundation

/// 报告里的球种胜场行，按球种值升序展示。
struct SeasonTypeWinRow: Equatable, Identifiable {
    let type: Int
    let label: String
    let wins: Int

    var id: Int { type }
}

/// 赛季报告：一次直读，切换球种重读；结果绑定发起时的身份。
@MainActor
final class SeasonReportModel: ObservableObject {
    enum Phase: Equatable {
        case idle
        case loading
        case ready
        case empty
        case failed(LoadFailure)
    }

    @Published private(set) var phase: Phase = .idle
    @Published private(set) var report: SeasonReportData?
    @Published private(set) var currentGameType: Int

    let seasonId: Int64

    private let session: SessionStore
    private let api: SeasonAPI
    private var owner: SessionIdentity?
    private var requestToken = 0

    init(session: SessionStore, seasonId: Int64, gameType: Int) {
        self.session = session
        self.seasonId = seasonId
        self.api = SeasonAPI(session: session)
        self.currentGameType = GameTypeCatalog.isSupported(gameType) ? gameType : 3
        self.owner = session.identity
    }

    // MARK: - 展示值

    var isLoading: Bool { phase == .loading }
    var currentGameTypeLabel: String { GameTypeCatalog.label(for: currentGameType) }

    var winRate: Int {
        guard let record = report?.record, record.matchesPlayed > 0 else { return 0 }
        return Int((Double(record.wins) / Double(record.matchesPlayed) * 100).rounded())
    }

    var typeWinRows: [SeasonTypeWinRow] {
        (report?.winRateByType ?? [:])
            .compactMap { key, wins in
                guard let type = Int(key) else { return nil }
                return SeasonTypeWinRow(
                    type: type,
                    label: GameTypeCatalog.label(for: type, fallback: "球种\(String(type))"),
                    wins: max(0, wins)
                )
            }
            .sorted { $0.type < $1.type }
    }

    var trendText: String {
        let trend = report?.rankTrend ?? []
        guard !trend.isEmpty else { return "本赛季暂无可展示的段位趋势" }
        return "趋势记录：\(trend.map(String.init).joined(separator: " / "))"
    }

    // MARK: - 读取

    func loadIfNeeded() async {
        syncIdentity()
        guard session.identity != nil, phase == .idle else { return }
        await fetch()
    }

    func retry() async {
        syncIdentity()
        guard session.identity != nil, !isLoading else { return }
        await fetch()
    }

    /// 切球种回到加载态并重读；旧球种的在途结果按令牌丢弃。
    func selectGameType(_ gameType: Int) async {
        guard gameType != currentGameType, GameTypeCatalog.isSupported(gameType) else { return }
        currentGameType = gameType
        report = nil
        await fetch()
    }

    /// 身份变化时清空并回到可重新读取的状态。
    func syncIdentity() {
        let current = session.identity
        guard current != owner else { return }
        owner = current
        requestToken += 1
        report = nil
        phase = .idle
    }

    // MARK: - 私有

    private func fetch() async {
        guard let requestOwner = session.identity else { return }
        guard seasonId > 0 else {
            phase = .failed(LoadFailure(category: .notFound, message: "未找到赛季信息，请返回赛季页后重试。"))
            return
        }

        requestToken += 1
        let token = requestToken
        let requestedGameType = currentGameType
        phase = .loading

        do {
            let response = try await api.report(seasonId: seasonId, gameType: requestedGameType)
            guard token == requestToken,
                  session.identity == requestOwner,
                  currentGameType == requestedGameType else { return }
            report = response.report
            phase = response.report == nil ? .empty : .ready
        } catch {
            guard token == requestToken,
                  session.identity == requestOwner,
                  currentGameType == requestedGameType else { return }
            if isCancellation(error) {
                if phase == .loading { phase = .idle }
                return
            }
            let failure = LoadFailure(error)
            // 全局会话失效流程已接管重新登录，页面不重复展示失败态。
            guard failure.category != .session else { return }
            phase = .failed(failure)
        }
    }

    private func isCancellation(_ error: Error) -> Bool {
        if let apiError = error as? APIError, apiError.kind == .canceled { return true }
        return error is CancellationError
    }
}
