import Combine
import Foundation

/// 欢迎页与登录页共用的协议勾选状态；离开漏斗的最后一个页面时清空。
@MainActor
final class EntryAgreementModel: ObservableObject {
    @Published private(set) var hasAgreed: Bool
    @Published private(set) var isSheetPresented = false

    private let store: EntryFunnelAgreementStore

    init(defaults: UserDefaults = .standard) {
        let store = EntryFunnelAgreementStore(defaults: defaults)
        self.store = store
        hasAgreed = store.activate()
    }

    var hasViewedWelcome: Bool { store.hasViewedWelcome }

    func toggle() {
        hasAgreed.toggle()
        store.save(hasAgreed)
    }

    func requestConsent() {
        isSheetPresented = true
    }

    func closeSheet() {
        isSheetPresented = false
    }

    func accept() {
        hasAgreed = true
        store.save(true)
        isSheetPresented = false
    }

    func markWelcomeViewed() {
        store.markWelcomeViewed()
    }

    func deactivate(current: EntryFunnelPage, remaining: [EntryFunnelPage]) {
        store.deactivate(current: current, remaining: remaining)
    }
}

/// 登录中的反馈文案：三秒后从「完成登录」升级为「网络有些慢」。
@MainActor
final class AuthFeedbackModel: ObservableObject {
    @Published private(set) var isAuthenticating = false
    @Published private(set) var isSlow = false

    private var startedAt: Date?
    private let now: () -> Date

    init(now: @escaping () -> Date = Date.init) {
        self.now = now
    }

    var feedback: EntryFunnelRules.AuthenticationFeedback {
        EntryFunnelRules.authenticationFeedback(isAuthenticating: isAuthenticating, isSlow: isSlow)
    }

    var buttonText: String {
        EntryFunnelRules.loginButtonText(isLogging: isAuthenticating, isSlow: isSlow)
    }

    func begin() {
        isAuthenticating = true
        isSlow = false
        startedAt = now()
    }

    /// 页面按固定间隔调用；规则只用开始时间与会话状态判断是否进入慢速阶段。
    func tick() {
        guard isAuthenticating, let startedAt else { return }
        let elapsedMs = max(0, Int(now().timeIntervalSince(startedAt) * 1000))
        isSlow = EntryFunnelRules.authenticationPhase(
            isAuthenticating: true,
            elapsedMs: elapsedMs
        ) == .slow
    }

    func end() {
        isAuthenticating = false
        isSlow = false
        startedAt = nil
    }
}
