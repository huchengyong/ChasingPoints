import Combine
import Foundation

/// 信誉明细页：信誉状态与变更流水各自独立读取，互不阻塞。
@MainActor
final class ReputationModel: ObservableObject {
    let status: ResourceLoader<ReputationStatus>
    let logs: PagedHistoryList<ReputationLog>

    private let session: SessionStore
    private var cancellables: Set<AnyCancellable> = []

    init(session: SessionStore) {
        self.session = session
        let api = ReputationAPI(session: session)
        status = ResourceLoader(session: session) { try await api.status() }
        logs = PagedHistoryList(session: session) { page in try await api.logsPage(page: page) }

        // 两个子状态机不是本对象的 @Published 属性，需要把变更转给页面。
        status.objectWillChange
            .merge(with: logs.objectWillChange)
            .sink { [weak self] in self?.objectWillChange.send() }
            .store(in: &cancellables)
    }

    func onAppear() async {
        guard session.identity != nil else {
            status.syncIdentity()
            logs.syncIdentity()
            return
        }
        async let statusLoad: Void = status.loadIfNeeded()
        async let logsLoad: Void = logs.loadIfNeeded()
        _ = await (statusLoad, logsLoad)
    }

    func refresh() async {
        guard session.identity != nil else { return }
        async let statusRefresh: Void = status.refresh()
        async let logsRefresh: Void = logs.refresh()
        _ = await (statusRefresh, logsRefresh)
    }

    func onIdentityChanged() async {
        status.syncIdentity()
        logs.syncIdentity()
        await onAppear()
    }

    func retryStatus() async {
        await status.retry()
    }

    func retryLogs() async {
        if logs.items.isEmpty {
            await logs.retry()
        } else {
            await logs.refresh()
        }
    }

    // MARK: - 展示值

    /// 与小程序 `displayScore` 一致：没有数值时显示 `--`。
    var displayScore: String {
        guard let score = status.value?.score else { return "--" }
        return String(score)
    }

    var displayStatusText: String {
        if let text = status.value?.statusText, !text.isEmpty { return text }
        return status.isLoading ? "加载中" : "暂不可用"
    }

    var isRestricted: Bool { status.value?.status == "restricted" }

    var banUntilText: String? {
        guard let banUntil = status.value?.banUntil else { return nil }
        let text = AppFormat.dateTime(banUntil)
        return text.isEmpty ? nil : text
    }

    var statusFeedback: LoadFailure.Feedback? {
        guard case .failed = status.phase else { return nil }
        return status.failure?.feedback(resource: "信誉状态")
    }

    /// 首页失败时用分类提示；`PagedHistoryList` 只保留文案，这里按业务失败补分类。
    func logsFeedback(for message: String) -> LoadFailure.Feedback {
        LoadFailure(category: .business, message: message).feedback(resource: "信誉记录")
    }
}
