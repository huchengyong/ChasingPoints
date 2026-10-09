import Combine
import Foundation

/// 单场战报的读取状态。每次进入页面创建新实例，不做跨页面缓存。
///
/// - 战报详情失败时显示错误，不使用默认比分。
/// - 奖励摘要只在选手视角的已完成对局读取一次；失败只隐藏奖励区，不影响战报其余内容。
@MainActor
final class MatchReportModel: ObservableObject {
    enum Phase: Equatable {
        case loading
        case loaded(MatchReport)
        case failed(String)
    }

    @Published private(set) var phase: Phase = .loading
    @Published private(set) var reward: MatchRewardState = .hidden

    let matchId: Int64
    private let session: SessionStore
    private let api: MatchHistoryAPI
    private var requestToken = 0

    init(session: SessionStore, matchId: Int64) {
        self.session = session
        self.matchId = matchId
        self.api = MatchHistoryAPI(session: session)
    }

    func load() async {
        guard let identity = session.identity else { return }
        requestToken += 1
        let token = requestToken
        phase = .loading
        reward = .hidden

        let report: MatchReport
        do {
            report = try await api.report(matchId: matchId)
        } catch {
            guard token == requestToken, session.identity == identity else { return }
            // 页面离开导致的取消保持加载态，页面再次出现时会重新读取。
            if let apiError = error as? APIError, apiError.kind == .canceled { return }
            phase = .failed(error.localizedDescription)
            return
        }
        guard token == requestToken, session.identity == identity else { return }
        phase = .loaded(report)

        guard !report.isRefereeView, report.status == 2 else { return }
        do {
            let summary = try await api.rewardSummary(matchId: matchId)
            guard token == requestToken, session.identity == identity else { return }
            reward = summary.state
        } catch {
            guard token == requestToken, session.identity == identity else { return }
            reward = .hidden
        }
    }
}
