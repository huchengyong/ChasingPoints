import Combine
import Foundation

/// 选择球友：服务端已按「好友优先、最近对手按最近对局」去重排序，页面直接展示。
@MainActor
final class OpponentSelectorModel: ObservableObject {
    enum Phase: Equatable {
        case idle
        case loading
        case loaded
        case empty
        case failed(LoadFailure)
    }

    @Published private(set) var phase: Phase = .idle
    @Published private(set) var opponents: [OpponentCandidate] = []

    private let session: SessionStore
    private let api: MatchStartAPI
    private var requestToken = 0

    init(session: SessionStore, api: MatchStartAPI? = nil) {
        self.session = session
        self.api = api ?? MatchStartAPI(session: session)
    }

    func loadIfNeeded() async {
        guard phase == .idle else { return }
        await fetch()
    }

    func retry() async {
        guard phase != .loading else { return }
        await fetch()
    }

    private func fetch() async {
        requestToken += 1
        let token = requestToken
        let owner = session.identity
        phase = .loading

        do {
            let candidates = try await api.opponentCandidates()
            // 迟到的响应只采纳最新一次请求，且必须仍属于发起时的账号。
            guard token == requestToken, session.identity == owner else { return }
            opponents = candidates
            phase = candidates.isEmpty ? .empty : .loaded
        } catch {
            guard token == requestToken, session.identity == owner else { return }
            opponents = []
            phase = .failed(LoadFailure(error))
        }
    }
}
