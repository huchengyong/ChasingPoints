import Foundation

/// 欢迎页与登录页的规则，逐条对应 `uniapp/utils/entry-funnel.js`；微信、华为与绑定手机号分支不迁移。
enum EntryFunnelRules {
    /// 对应 AUTH_SLOW_FEEDBACK_DELAY。
    static let authSlowFeedbackDelayMs = 3000
    /// 对应 AUTH_NETWORK_ERROR_MESSAGE。
    static let authNetworkErrorMessage = "网络不太稳定，登录超时，请检查网络后重试"

    enum AuthenticationPhase: Equatable {
        case idle
        case pending
        case slow
    }

    struct AuthenticationGate: Equatable {
        let isAuthenticating: Bool
        let canStart: Bool
        let canLeave: Bool
        let shouldHandleResult: Bool
    }

    struct AuthenticationFeedback: Equatable {
        let visible: Bool
        let phase: AuthenticationPhase
        let message: String
    }

    struct WelcomeActions: Equatable {
        let primaryText: String
        let tertiaryText: String
    }

    enum WelcomePrimaryOutcome: Equatable {
        case showAgreementSheet
        case openLogin
    }

    enum LoginSubmitGate: Equatable {
        /// 表单未就绪，按钮不可提交。
        case blocked
        /// 表单就绪但未同意协议，先弹协议确认。
        case showAgreementSheet
        case submit
    }

    /// 原生端固定手机号登录，取 JS 中非微信端的结果。
    static let welcomeActions = WelcomeActions(primaryText: "手机号登录 / 注册", tertiaryText: "先逛逛")

    static func shouldShowWelcome(hasViewed: Bool, isSignedIn: Bool) -> Bool {
        !hasViewed && !isSignedIn
    }

    static func welcomePrimaryOutcome(isAgreed: Bool) -> WelcomePrimaryOutcome {
        isAgreed ? .openLogin : .showAgreementSheet
    }

    static func loginSubmitGate(phone: String, code: String, isAgreed: Bool, isLogging: Bool) -> LoginSubmitGate {
        guard canAttemptLogin(phone: phone, code: code, isLogging: isLogging) else { return .blocked }
        return isAgreed ? .submit : .showAgreementSheet
    }

    static func loginButtonText(isLogging: Bool, isSlow: Bool) -> String {
        guard isLogging else { return "登录 / 注册" }
        return isSlow ? "网络稍慢，正在继续…" : "正在安全登录…"
    }

    static func authenticationGate(isLoggingIn: Bool, isPageActive: Bool = true) -> AuthenticationGate {
        let isAuthenticating = isLoggingIn
        let canInteract = isPageActive && !isAuthenticating
        return AuthenticationGate(
            isAuthenticating: isAuthenticating,
            canStart: canInteract,
            canLeave: canInteract,
            shouldHandleResult: isPageActive
        )
    }

    static func authenticationPhase(isAuthenticating: Bool, elapsedMs: Int) -> AuthenticationPhase {
        guard isAuthenticating else { return .idle }
        return elapsedMs >= authSlowFeedbackDelayMs ? .slow : .pending
    }

    static func authenticationFeedback(isAuthenticating: Bool, isSlow: Bool) -> AuthenticationFeedback {
        guard isAuthenticating else {
            return AuthenticationFeedback(visible: false, phase: .idle, message: "")
        }
        if isSlow {
            return AuthenticationFeedback(
                visible: true,
                phase: .slow,
                message: "网络有些慢，追分竭尽全力为您继续尝试中"
            )
        }
        return AuthenticationFeedback(visible: true, phase: .pending, message: "追分正在为您完成登录")
    }

    static func authErrorMessage(_ error: Error, fallback: String) -> String {
        if let apiError = error as? APIError, apiError.kind == .network {
            return authNetworkErrorMessage
        }
        let message = (error as? APIError)?.message ?? error.localizedDescription
        return message.isEmpty ? fallback : message
    }

    static func isPhoneValid(_ phone: String) -> Bool {
        let value = phone.trimmingCharacters(in: .whitespacesAndNewlines)
        return value.range(of: #"^1[3-9][0-9]{9}$"#, options: .regularExpression) != nil
    }

    static func isCodeValid(_ code: String) -> Bool {
        let value = code.trimmingCharacters(in: .whitespacesAndNewlines)
        return value.count == 6 && value.allSatisfy { "0123456789".contains($0) }
    }

    static func canRequestSMS(phone: String, countdown: Int, isSending: Bool) -> Bool {
        isPhoneValid(phone) && countdown <= 0 && !isSending
    }

    static func canAttemptLogin(phone: String, code: String, isLogging: Bool) -> Bool {
        isPhoneValid(phone) && isCodeValid(code) && !isLogging
    }

    static func canSubmitLogin(phone: String, code: String, isAgreed: Bool, isLogging: Bool) -> Bool {
        canAttemptLogin(phone: phone, code: code, isLogging: isLogging) && isAgreed
    }

    static func phoneError(_ phone: String) -> String {
        let value = phone.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !value.isEmpty else { return "" }
        return isPhoneValid(value) ? "" : "请输入正确的手机号"
    }

    static func codeError(_ code: String) -> String {
        let value = code.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !value.isEmpty else { return "" }
        return isCodeValid(value) ? "" : "请输入6位验证码"
    }

    static func maskPhone(_ phone: String) -> String {
        let value = phone.trimmingCharacters(in: .whitespacesAndNewlines)
        guard isPhoneValid(value) else { return "" }
        return "\(value.prefix(3))****\(value.suffix(4))"
    }

    static func smsFeedback(phone: String) -> String {
        let masked = maskPhone(phone)
        return masked.isEmpty ? "验证码已发送，请注意查收" : "验证码已发送至 \(masked)"
    }

    /// 登录成功后：从其他页面进入的返回上一页，直接打开登录页的回首页。
    static func postLoginNavigation(pageCount: Int) -> PostLoginNavigation {
        pageCount > 1 ? .back : .home
    }

    static func resolveEntryFunnelAgreementState(storedAgreement: Bool, sessionActive: Bool) -> Bool {
        guard sessionActive else { return false }
        return storedAgreement
    }

    static func shouldClearEntryFunnelAgreementSession(
        current: EntryFunnelPage,
        visibleRoutes: [EntryFunnelPage]
    ) -> Bool {
        !visibleRoutes.contains { $0 != current }
    }
}

enum PostLoginNavigation: Equatable {
    case back
    case home
}

/// 入口漏斗内的页面，对应小程序中 `ENTRY_FUNNEL_ROUTES` 的两条路由。
enum EntryFunnelPage: Hashable {
    case welcome
    case login
}

/// 欢迎页与登录页共用的协议勾选持久化；键名与小程序一致。
struct EntryFunnelAgreementStore {
    static let agreementKey = "entry_funnel_agreement_accepted"
    static let sessionKey = "entry_funnel_session_active"
    static let welcomeViewedKey = "welcome_page_viewed"

    private let defaults: UserDefaults

    init(defaults: UserDefaults = .standard) {
        self.defaults = defaults
    }

    var hasViewedWelcome: Bool {
        defaults.bool(forKey: Self.welcomeViewedKey)
    }

    /// 进入入口页：恢复本会话内的勾选；新会话丢弃上次遗留的勾选。
    func activate() -> Bool {
        let sessionActive = defaults.bool(forKey: Self.sessionKey)
        let agreed = EntryFunnelRules.resolveEntryFunnelAgreementState(
            storedAgreement: defaults.bool(forKey: Self.agreementKey),
            sessionActive: sessionActive
        )
        defaults.set(true, forKey: Self.sessionKey)
        if !sessionActive {
            defaults.removeObject(forKey: Self.agreementKey)
        }
        return agreed
    }

    func save(_ agreed: Bool) {
        defaults.set(agreed, forKey: Self.agreementKey)
    }

    func markWelcomeViewed() {
        defaults.set(true, forKey: Self.welcomeViewedKey)
    }

    /// 离开入口页：漏斗内已无其他页面时清掉协议状态。
    func deactivate(current: EntryFunnelPage, remaining: [EntryFunnelPage]) {
        guard EntryFunnelRules.shouldClearEntryFunnelAgreementSession(current: current, visibleRoutes: remaining) else {
            return
        }
        defaults.removeObject(forKey: Self.agreementKey)
        defaults.removeObject(forKey: Self.sessionKey)
    }
}
