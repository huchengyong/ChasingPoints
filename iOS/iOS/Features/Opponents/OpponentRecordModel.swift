import Combine
import Foundation

/// 过往对手页：列表分页由 `PagedHistoryList` 负责，总对手数/总胜场跟随每页响应更新。
@MainActor
final class OpponentRecordModel: ObservableObject {
    static let pageSize = OpponentAPI.pageSize

    let targetUserId: Int64
    let targetName: String
    let targetAvatar: String

    /// 搜索框内容；与小程序一致，刷新时保留。
    @Published var keyword = ""
    @Published private(set) var totalOpponents = 0
    @Published private(set) var totalWins = 0

    private let session: SessionStore
    private let api: OpponentAPI
    /// 只采纳最新一次响应的统计，避免旧搜索的结果覆盖新结果。
    private var statsToken = 0
    private var cancellables: Set<AnyCancellable> = []

    lazy var list: PagedHistoryList<OpponentRecord> = {
        let list = PagedHistoryList<OpponentRecord>(session: session) { [weak self] page in
            guard let self else { return ([], 0) }
            return try await self.loadPage(page)
        }
        // 列表状态机不是本对象的 @Published 属性，需要把变更转给页面。
        list.objectWillChange
            .sink { [weak self] in self?.objectWillChange.send() }
            .store(in: &cancellables)
        return list
    }()

    init(
        session: SessionStore,
        targetUserId: Int64 = 0,
        targetName: String = "",
        targetAvatar: String = ""
    ) {
        self.session = session
        self.targetUserId = targetUserId
        self.targetName = targetName
        self.targetAvatar = targetAvatar
        api = OpponentAPI(session: session)
    }

    var viewModel: OpponentRecordViewModel {
        OpponentRecordRules.viewModel(targetName: targetName, targetUserId: targetUserId)
    }

    var statsSummary: OpponentStatsSummary {
        OpponentRecordRules.statsSummary(
            totalOpponents: totalOpponents,
            totalWins: totalWins,
            targetName: targetName,
            targetUserId: targetUserId
        )
    }

    var cards: [OpponentCard] {
        OpponentRecordRules.cards(
            opponents: list.items,
            subjectName: viewModel.subjectName
        )
    }

    func onAppear() async {
        guard session.identity != nil else {
            list.syncIdentity()
            resetStats()
            return
        }
        await list.loadIfNeeded()
    }

    func search() async {
        await list.refresh()
    }

    func retry() async {
        await list.retry()
    }

    func onIdentityChanged() async {
        list.syncIdentity()
        resetStats()
        await onAppear()
    }

    private func loadPage(_ page: Int) async throws -> (items: [OpponentRecord], total: Int) {
        statsToken += 1
        let token = statsToken
        let owner = session.identity
        let response = try await api.page(page: page, keyword: keyword, targetUserId: targetUserId)
        if token == statsToken, session.identity == owner {
            totalOpponents = response.totalOpponents
            totalWins = response.totalWins
        }
        return (response.list ?? [], Int(response.total))
    }

    private func resetStats() {
        statsToken += 1
        totalOpponents = 0
        totalWins = 0
    }
}
