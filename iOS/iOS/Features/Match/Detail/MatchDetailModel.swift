import Combine
import Foundation

/// 对局详情（小程序 `matchDetail`）：公开详情只读接口，结果绑定发起时的会话身份。
@MainActor
final class MatchDetailModel: ObservableObject {
    enum Phase: Equatable {
        case loading
        case failed(String)
        case loaded(MatchDetailViewData)
    }

    @Published private(set) var phase: Phase = .loading

    let matchId: Int64
    private let session: SessionStore
    private let isSpectate: Bool
    private let api: LiveMatchAPI
    private var requestToken = 0

    init(session: SessionStore, matchId: Int64, isSpectate: Bool) {
        self.session = session
        self.matchId = matchId
        self.isSpectate = isSpectate
        self.api = LiveMatchAPI(session: session)
    }

    /// 分享链接进来的访客：观战模式且未登录时给一条登录引导。
    var showsLoginHint: Bool { isSpectate && session.identity == nil }

    var viewData: MatchDetailViewData? {
        if case .loaded(let data) = phase { return data }
        return nil
    }

    func load() async {
        guard matchId > 0 else {
            phase = .failed("缺少对局信息")
            return
        }
        requestToken += 1
        let token = requestToken
        let identity = session.identity
        phase = .loading

        let publicDetail = MatchDetailRules.shouldUsePublicMatchDetail(
            mode: isSpectate ? "spectate" : ""
        )
        do {
            let response = publicDetail
                ? try await api.publicDetail(matchId: matchId)
                : try await api.detail(matchId: matchId)
            guard token == requestToken, session.identity == identity else { return }
            guard response.success, let snapshot = response.match else {
                phase = .failed("暂时无法查看这场对局")
                return
            }
            phase = .loaded(MatchDetailRules.normalize(
                snapshot,
                perspectiveUserId: identity?.userId ?? 0
            ))
        } catch {
            guard token == requestToken, session.identity == identity else { return }
            if let apiError = error as? APIError, apiError.kind == .canceled { return }
            phase = .failed(error.localizedDescription)
        }
    }
}
