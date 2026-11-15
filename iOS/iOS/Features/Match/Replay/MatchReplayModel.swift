import Combine
import Foundation

/// 单场复盘（小程序 `matchReplay`）：会员权益读取，结果绑定发起时的会话身份。
///
/// - 服务端对不存在、未完成、非参与者统一返回 success=false，不区分原因。
/// - 已有内容时刷新失败只提示，不清空旧复盘。
@MainActor
final class MatchReplayModel: ObservableObject {
    enum Phase: Equatable {
        case loading
        case failed(LoadFailure)
        case loaded(ReplayView)
    }

    @Published private(set) var phase: Phase = .loading
    @Published private(set) var refreshFailure: LoadFailure?

    let matchId: Int64
    private let session: SessionStore
    private let api: MatchReplayAPI
    private var requestToken = 0

    init(session: SessionStore, matchId: Int64) {
        self.session = session
        self.matchId = matchId
        self.api = MatchReplayAPI(session: session)
    }

    var view: ReplayView? {
        if case .loaded(let view) = phase { return view }
        return nil
    }

    func load() async {
        guard matchId > 0 else {
            phase = .failed(LoadFailure(category: .notFound, message: "缺少对局信息"))
            return
        }
        guard let identity = session.identity else { return }
        requestToken += 1
        let token = requestToken
        let isRefresh = view != nil
        if isRefresh {
            refreshFailure = nil
        } else {
            phase = .loading
        }

        do {
            let response = try await api.replay(matchId: matchId)
            guard token == requestToken, session.identity == identity else { return }
            guard response.success else {
                throw APIError(message: "暂时无法查看这场对局的复盘", statusCode: 404)
            }
            phase = .loaded(MemberReplayView.resolveReplayView(
                response,
                canShowPurchase: AppFlags.canShowPaidMembershipEntry
            ))
            refreshFailure = nil
        } catch {
            guard token == requestToken, session.identity == identity else { return }
            if let apiError = error as? APIError, apiError.kind == .canceled { return }
            // 服务端对本接口的 success=false 只表示未完成、不存在或非参与者，统一按资源不存在处理。
            let failure = (error as? APIError)?.kind == .business
                ? LoadFailure(category: .notFound, message: error.localizedDescription)
                : LoadFailure(error)
            if isRefresh {
                refreshFailure = failure
            } else {
                phase = .failed(failure)
            }
        }
    }

    func retry() async {
        await load()
    }
}
