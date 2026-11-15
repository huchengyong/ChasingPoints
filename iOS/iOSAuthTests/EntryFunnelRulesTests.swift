import Foundation
import XCTest
@testable import iOS

@MainActor
final class EntryFunnelRulesTests: XCTestCase {
    // MARK: - entry-funnel.js 用例

    func testAuthenticationGateBlocksCompetingAuthActionsAndIgnoresResultsAfterUnload() {
        XCTAssertEqual(
            EntryFunnelRules.authenticationGate(isLoggingIn: false),
            .init(isAuthenticating: false, canStart: true, canLeave: true, shouldHandleResult: true)
        )
        XCTAssertEqual(
            EntryFunnelRules.authenticationGate(isLoggingIn: true),
            .init(isAuthenticating: true, canStart: false, canLeave: false, shouldHandleResult: true)
        )
        XCTAssertEqual(
            EntryFunnelRules.authenticationGate(isLoggingIn: false, isPageActive: false),
            .init(isAuthenticating: false, canStart: false, canLeave: false, shouldHandleResult: false)
        )
    }

    func testAuthenticationPhaseEscalatesOnlyActiveRequestsAfterSlowThreshold() {
        XCTAssertEqual(EntryFunnelRules.authenticationPhase(isAuthenticating: false, elapsedMs: 0), .idle)
        XCTAssertEqual(EntryFunnelRules.authenticationPhase(isAuthenticating: false, elapsedMs: 3000), .idle)
        XCTAssertEqual(EntryFunnelRules.authenticationPhase(isAuthenticating: true, elapsedMs: 0), .pending)
        XCTAssertEqual(EntryFunnelRules.authenticationPhase(isAuthenticating: true, elapsedMs: 2999), .pending)
        XCTAssertEqual(EntryFunnelRules.authenticationPhase(isAuthenticating: true, elapsedMs: 3000), .slow)
    }

    func testAuthenticationFeedbackKeepsPendingFeedbackVisibleAndUpgradesSlowRequests() {
        XCTAssertEqual(
            EntryFunnelRules.authenticationFeedback(isAuthenticating: false, isSlow: false),
            .init(visible: false, phase: .idle, message: "")
        )
        XCTAssertEqual(
            EntryFunnelRules.authenticationFeedback(isAuthenticating: true, isSlow: false),
            .init(visible: true, phase: .pending, message: "追分正在为您完成登录")
        )
        XCTAssertEqual(
            EntryFunnelRules.authenticationFeedback(isAuthenticating: true, isSlow: true),
            .init(visible: true, phase: .slow, message: "网络有些慢，追分竭尽全力为您继续尝试中")
        )
    }

    func testWelcomeActionsUsePhoneLoginCopyWithoutWechatOrHuaweiEntries() {
        XCTAssertEqual(EntryFunnelRules.welcomeActions.primaryText, "手机号登录 / 注册")
        XCTAssertEqual(EntryFunnelRules.welcomeActions.tertiaryText, "先逛逛")
    }

    func testIsPhoneValidAcceptsMainlandMobileNumbers() {
        XCTAssertTrue(EntryFunnelRules.isPhoneValid("13800138000"))
    }

    func testIsPhoneValidRejectsInvalidMobileNumbers() {
        XCTAssertFalse(EntryFunnelRules.isPhoneValid("123"))
        XCTAssertFalse(EntryFunnelRules.isPhoneValid(""))
    }

    func testIsCodeValidOnlyAcceptsSixDigitSMSCodes() {
        XCTAssertTrue(EntryFunnelRules.isCodeValid("123456"))
        XCTAssertFalse(EntryFunnelRules.isCodeValid("12345"))
        XCTAssertFalse(EntryFunnelRules.isCodeValid("1234567"))
        XCTAssertFalse(EntryFunnelRules.isCodeValid("12a456"))
    }

    func testCanRequestSMSRequiresValidPhoneAndIdleSendState() {
        XCTAssertTrue(EntryFunnelRules.canRequestSMS(phone: "13800138000", countdown: 0, isSending: false))
        XCTAssertFalse(EntryFunnelRules.canRequestSMS(phone: "13800138000", countdown: 10, isSending: false))
        XCTAssertFalse(EntryFunnelRules.canRequestSMS(phone: "13800138000", countdown: 0, isSending: true))
    }

    func testCanSubmitLoginRequiresValidFormAgreementAndIdleSubmitState() {
        XCTAssertTrue(EntryFunnelRules.canSubmitLogin(
            phone: "13800138000", code: "123456", isAgreed: true, isLogging: false
        ))
        XCTAssertFalse(EntryFunnelRules.canSubmitLogin(
            phone: "13800138000", code: "123456", isAgreed: false, isLogging: false
        ))
        XCTAssertFalse(EntryFunnelRules.canSubmitLogin(
            phone: "13800138000", code: "12345", isAgreed: true, isLogging: false
        ))
    }

    func testCanAttemptLoginIgnoresAgreementStateAndOnlyDependsOnFormReadiness() {
        XCTAssertTrue(EntryFunnelRules.canAttemptLogin(phone: "13800138000", code: "123456", isLogging: false))
        XCTAssertFalse(EntryFunnelRules.canAttemptLogin(phone: "13800138000", code: "12345", isLogging: false))
    }

    func testPhoneErrorOnlySurfacesAfterUserStartsTypingInvalidNumber() {
        XCTAssertEqual(EntryFunnelRules.phoneError(""), "")
        XCTAssertEqual(EntryFunnelRules.phoneError("13800138000"), "")
        XCTAssertEqual(EntryFunnelRules.phoneError("1380013"), "请输入正确的手机号")
    }

    func testCodeErrorOnlySurfacesAfterUserStartsTypingInvalidCode() {
        XCTAssertEqual(EntryFunnelRules.codeError(""), "")
        XCTAssertEqual(EntryFunnelRules.codeError("123456"), "")
        XCTAssertEqual(EntryFunnelRules.codeError("1234"), "请输入6位验证码")
    }

    func testMaskPhoneReturnsMaskedNumberForValidPhonesOnly() {
        XCTAssertEqual(EntryFunnelRules.maskPhone("13800138000"), "138****8000")
        XCTAssertEqual(EntryFunnelRules.maskPhone("123"), "")
    }

    func testSmsFeedbackPrefersMaskedDestinationWhenPhoneIsValid() {
        XCTAssertEqual(EntryFunnelRules.smsFeedback(phone: "13800138000"), "验证码已发送至 138****8000")
        XCTAssertEqual(EntryFunnelRules.smsFeedback(phone: "bad phone"), "验证码已发送，请注意查收")
    }

    func testAgreementStateResetsStaleAgreementWhenOpeningFreshFunnelSession() {
        XCTAssertFalse(EntryFunnelRules.resolveEntryFunnelAgreementState(
            storedAgreement: true, sessionActive: false
        ))
        XCTAssertTrue(EntryFunnelRules.resolveEntryFunnelAgreementState(
            storedAgreement: true, sessionActive: true
        ))
    }

    func testAgreementSessionClearsOnlyWhenNoFunnelPageRemainsInStack() {
        XCTAssertFalse(EntryFunnelRules.shouldClearEntryFunnelAgreementSession(
            current: .login,
            visibleRoutes: [.welcome]
        ))
        XCTAssertTrue(EntryFunnelRules.shouldClearEntryFunnelAgreementSession(
            current: .login,
            visibleRoutes: []
        ))
    }

    func testAuthErrorMessageTurnsNetworkTimeoutsIntoReadableLoginMessage() {
        let error = APIError(message: "request:fail timeout", kind: .network)
        XCTAssertEqual(
            EntryFunnelRules.authErrorMessage(error, fallback: "登录失败"),
            EntryFunnelRules.authNetworkErrorMessage
        )
        XCTAssertFalse(EntryFunnelRules.authNetworkErrorMessage.contains("timeout"))
    }

    func testAuthErrorMessageKeepsServerMessagesAndFallsBackForUnknownErrors() {
        let business = APIError(message: "验证码已过期，请重新获取", kind: .business)
        XCTAssertEqual(
            EntryFunnelRules.authErrorMessage(business, fallback: "登录失败"),
            "验证码已过期，请重新获取"
        )
        XCTAssertEqual(
            EntryFunnelRules.authErrorMessage(APIError(message: "", kind: .business), fallback: "登录失败，请重试"),
            "登录失败，请重试"
        )
    }

    // MARK: - 欢迎页与登录页补充用例

    func testWelcomeShowsOnlyOnFirstLaunchWithoutSession() {
        XCTAssertTrue(EntryFunnelRules.shouldShowWelcome(hasViewed: false, isSignedIn: false))
        XCTAssertFalse(EntryFunnelRules.shouldShowWelcome(hasViewed: true, isSignedIn: false))
        XCTAssertFalse(EntryFunnelRules.shouldShowWelcome(hasViewed: false, isSignedIn: true))
    }

    func testWelcomePrimaryRequestsAgreementOnlyWhenNotAgreed() {
        XCTAssertEqual(EntryFunnelRules.welcomePrimaryOutcome(isAgreed: false), .showAgreementSheet)
        XCTAssertEqual(EntryFunnelRules.welcomePrimaryOutcome(isAgreed: true), .openLogin)
    }

    func testLoginSubmitGateAsksForAgreementOnlyWhenFormIsReady() {
        XCTAssertEqual(
            EntryFunnelRules.loginSubmitGate(phone: "13800138000", code: "123456", isAgreed: false, isLogging: false),
            .showAgreementSheet
        )
        XCTAssertEqual(
            EntryFunnelRules.loginSubmitGate(phone: "13800138000", code: "123456", isAgreed: true, isLogging: false),
            .submit
        )
        XCTAssertEqual(
            EntryFunnelRules.loginSubmitGate(phone: "1380013", code: "123456", isAgreed: true, isLogging: false),
            .blocked
        )
        XCTAssertEqual(
            EntryFunnelRules.loginSubmitGate(phone: "13800138000", code: "123456", isAgreed: true, isLogging: true),
            .blocked
        )
    }

    func testLoginButtonTextFollowsAuthenticationPhase() {
        XCTAssertEqual(EntryFunnelRules.loginButtonText(isLogging: false, isSlow: false), "登录 / 注册")
        XCTAssertEqual(EntryFunnelRules.loginButtonText(isLogging: true, isSlow: false), "正在安全登录…")
        XCTAssertEqual(EntryFunnelRules.loginButtonText(isLogging: true, isSlow: true), "网络稍慢，正在继续…")
    }
}

@MainActor
final class EntryAgreementModelTests: XCTestCase {
    func testFreshEntrySessionDropsStaleAgreement() {
        let (defaults, name) = makeDefaults()
        defer { defaults.removePersistentDomain(forName: name) }
        defaults.set(true, forKey: EntryFunnelAgreementStore.agreementKey)
        defaults.set(false, forKey: EntryFunnelAgreementStore.sessionKey)

        let model = EntryAgreementModel(defaults: defaults)

        XCTAssertFalse(model.hasAgreed)
        XCTAssertNil(defaults.object(forKey: EntryFunnelAgreementStore.agreementKey))
        XCTAssertTrue(defaults.bool(forKey: EntryFunnelAgreementStore.sessionKey))
    }

    func testActiveEntrySessionRestoresAgreement() {
        let (defaults, name) = makeDefaults()
        defer { defaults.removePersistentDomain(forName: name) }
        defaults.set(true, forKey: EntryFunnelAgreementStore.agreementKey)
        defaults.set(true, forKey: EntryFunnelAgreementStore.sessionKey)

        XCTAssertTrue(EntryAgreementModel(defaults: defaults).hasAgreed)
    }

    func testTogglingAndAcceptingAgreementPersists() {
        let (defaults, name) = makeDefaults()
        defer { defaults.removePersistentDomain(forName: name) }
        let model = EntryAgreementModel(defaults: defaults)

        model.toggle()
        XCTAssertTrue(model.hasAgreed)
        XCTAssertTrue(defaults.bool(forKey: EntryFunnelAgreementStore.agreementKey))

        model.toggle()
        XCTAssertFalse(model.hasAgreed)
        XCTAssertFalse(defaults.bool(forKey: EntryFunnelAgreementStore.agreementKey))

        model.requestConsent()
        XCTAssertTrue(model.isSheetPresented)
        model.accept()
        XCTAssertTrue(model.hasAgreed)
        XCTAssertFalse(model.isSheetPresented)
        XCTAssertTrue(defaults.bool(forKey: EntryFunnelAgreementStore.agreementKey))
    }

    func testDeactivateKeepsAgreementWhileAnotherEntryPageRemains() {
        let (defaults, name) = makeDefaults()
        defer { defaults.removePersistentDomain(forName: name) }
        let model = EntryAgreementModel(defaults: defaults)
        model.toggle()

        model.deactivate(current: .welcome, remaining: [.login])
        XCTAssertTrue(defaults.bool(forKey: EntryFunnelAgreementStore.agreementKey))
        XCTAssertTrue(defaults.bool(forKey: EntryFunnelAgreementStore.sessionKey))

        model.deactivate(current: .login, remaining: [])
        XCTAssertNil(defaults.object(forKey: EntryFunnelAgreementStore.agreementKey))
        XCTAssertNil(defaults.object(forKey: EntryFunnelAgreementStore.sessionKey))
    }

    func testWelcomeViewedFlagHidesWelcomeForLaterLaunches() {
        let (defaults, name) = makeDefaults()
        defer { defaults.removePersistentDomain(forName: name) }
        let model = EntryAgreementModel(defaults: defaults)
        XCTAssertTrue(EntryFunnelRules.shouldShowWelcome(hasViewed: model.hasViewedWelcome, isSignedIn: false))

        model.markWelcomeViewed()

        XCTAssertTrue(model.hasViewedWelcome)
        XCTAssertFalse(EntryFunnelRules.shouldShowWelcome(hasViewed: model.hasViewedWelcome, isSignedIn: false))
    }

    func testAuthFeedbackModelSwitchesToSlowAfterThreeSecondsWithInjectedClock() {
        var now = Date(timeIntervalSince1970: 1_700_000_000)
        let model = AuthFeedbackModel(now: { now })

        model.begin()
        XCTAssertEqual(model.feedback.phase, .pending)
        XCTAssertEqual(model.feedback.message, "追分正在为您完成登录")

        now = now.addingTimeInterval(2.999)
        model.tick()
        XCTAssertEqual(model.feedback.phase, .pending)

        now = now.addingTimeInterval(0.001)
        model.tick()
        XCTAssertEqual(model.feedback.phase, .slow)
        XCTAssertEqual(model.feedback.message, "网络有些慢，追分竭尽全力为您继续尝试中")
        XCTAssertEqual(model.buttonText, "网络稍慢，正在继续…")

        model.end()
        XCTAssertFalse(model.feedback.visible)
        XCTAssertEqual(model.buttonText, "登录 / 注册")
    }

    private func makeDefaults() -> (UserDefaults, String) {
        let name = "EntryFunnelTests.\(UUID().uuidString)"
        let defaults = UserDefaults(suiteName: name)!
        defaults.removePersistentDomain(forName: name)
        return (defaults, name)
    }
}
