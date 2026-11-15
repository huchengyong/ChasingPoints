import Combine
import Foundation

/// 承接页状态：loading 解析或兑换中；preview 未登录预览；expired 短码失效；error 网络或服务异常可重试。
@MainActor
final class LinkEntryModel: ObservableObject {
    enum Stage: Equatable {
        case loading
        case preview
        case expired
        case error
    }

    @Published private(set) var stage: Stage = .loading
    @Published private(set) var errorMessage = ""
    @Published private(set) var inviter = LinkEntryInviter(name: "", avatar: "")
    @Published private(set) var expiresAt = ""
    /// 视图消费后调用 `clearConfirmation()`，避免重复压栈。
    @Published private(set) var didConfirm = false

    private let flow: LinkEntryFlow
    private var started = false

    init(session: SessionStore, quickScan: QuickScanFlow) {
        flow = LinkEntryFlow(session: session, quickScan: quickScan)
    }

    init(flow: LinkEntryFlow) {
        self.flow = flow
    }

    func begin(code: String) async {
        started = true
        apply(await flow.begin(code))
    }

    func resume() async {
        guard started else { return }
        apply(await flow.resume())
    }

    func retry() async {
        stage = .loading
        let action = await flow.retry()
        // 被忽略（已有请求在途或登录态刚变化）时退回错误态，不让页面停在加载中。
        if case .ignore = action {
            stage = .error
            return
        }
        apply(action)
    }

    func clearConfirmation() {
        didConfirm = false
    }

    private func apply(_ action: LinkEntryAction) {
        switch action {
        case .awaitLogin(let inviter, let expiresAt):
            self.inviter = inviter
            self.expiresAt = expiresAt
            stage = .preview
        case .confirm:
            didConfirm = true
        case .expired(let message):
            errorMessage = message
            stage = .expired
        case .error(let message):
            errorMessage = message
            stage = .error
        case .ignore:
            break
        }
    }
}
