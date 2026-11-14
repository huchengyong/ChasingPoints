import Combine
import Foundation

/// 单场战报的读取状态。每次进入页面创建新实例，不做跨页面缓存。
///
/// - 战报详情失败时显示错误，不使用默认比分。
/// - 奖励摘要只在选手视角的已完成对局读取；`pending` 时最多重试到 `MatchResultPage.rewardMaxAttempts` 次。
@MainActor
final class MatchReportModel: ObservableObject {
    enum Phase: Equatable {
        case loading
        case loaded(MatchReport)
        case failed(LoadFailure)
    }

    @Published private(set) var phase: Phase = .loading
    @Published private(set) var reward: MatchRewardState = .hidden
    @Published private(set) var rewardMessage = ""
    @Published private(set) var rewardShouldRetry = false

    let matchId: Int64
    private let session: SessionStore
    private let api: MatchHistoryAPI
    private var requestToken = 0
    private var retryDelay: @Sendable () async throws -> Void

    init(
        session: SessionStore,
        matchId: Int64,
        retryDelay: @escaping @Sendable () async throws -> Void = {
            try await Task.sleep(nanoseconds: UInt64(MatchResultPage.rewardRetryDelayMilliseconds) * 1_000_000)
        }
    ) {
        self.session = session
        self.matchId = matchId
        self.api = MatchHistoryAPI(session: session)
        self.retryDelay = retryDelay
    }

    func load() async {
        guard let identity = session.identity else { return }
        requestToken += 1
        let token = requestToken
        phase = .loading
        resetReward()

        let report: MatchReport
        do {
            report = try await api.report(matchId: matchId)
        } catch {
            guard token == requestToken, session.identity == identity else { return }
            // 页面离开导致的取消保持加载态，页面再次出现时会重新读取。
            if let apiError = error as? APIError, apiError.kind == .canceled { return }
            phase = .failed(LoadFailure(error))
            return
        }
        guard token == requestToken, session.identity == identity else { return }
        phase = .loaded(report)

        guard !report.isRefereeView, report.status == 2 else { return }
        await loadReward(token: token, identity: identity)
    }

    /// 奖励只是赛果页的一块内容：失败或取消都不影响战报主体。
    private func loadReward(token: Int, identity: SessionIdentity) async {
        for attempt in 1...MatchResultPage.rewardMaxAttempts {
            let summary: MatchRewardSummary
            do {
                summary = try await api.rewardSummary(matchId: matchId)
            } catch {
                guard token == requestToken, session.identity == identity else { return }
                if let apiError = error as? APIError, apiError.kind == .canceled { return }
                resetReward()
                return
            }
            guard token == requestToken, session.identity == identity else { return }

            let view = MatchResultPage.resolveRewardSummary(summary, attempt: attempt)
            reward = view.state
            rewardMessage = view.message
            rewardShouldRetry = view.shouldRetry
            guard view.shouldRetry else { return }

            do {
                try await retryDelay()
            } catch {
                return
            }
            guard token == requestToken, session.identity == identity else { return }
        }
    }

    private func resetReward() {
        reward = .hidden
        rewardMessage = ""
        rewardShouldRetry = false
    }
}
