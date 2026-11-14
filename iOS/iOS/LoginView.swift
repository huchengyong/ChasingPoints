import SwiftUI

struct LoginView: View {
    @EnvironmentObject var session: SessionStore
    @Environment(\.colorScheme) private var colorScheme
    @Environment(\.dismiss) private var dismiss

    @State private var phone = ""
    @State private var code = ""
    @State private var isSending = false
    @State private var resendAfter: Date?
    @State private var message: String?
    @State private var isError = false
    @StateObject private var agreement: EntryAgreementModel
    @StateObject private var feedback: AuthFeedbackModel
    @FocusState private var focusedField: Field?

    private enum Field {
        case phone, code
    }

    private let danger = AppTheme.danger

    init(
        agreement: EntryAgreementModel? = nil,
        feedback: AuthFeedbackModel? = nil,
        initialError: String? = nil
    ) {
        _agreement = StateObject(wrappedValue: agreement ?? EntryAgreementModel())
        _feedback = StateObject(wrappedValue: feedback ?? AuthFeedbackModel())
        _message = State(initialValue: initialError)
        _isError = State(initialValue: initialError != nil)
    }

    private var isBusy: Bool { isSending || feedback.isAuthenticating }
    private var phoneError: String { EntryFunnelRules.phoneError(phone) }
    private var codeError: String { EntryFunnelRules.codeError(code) }
    private var canSubmit: Bool {
        EntryFunnelRules.canAttemptLogin(phone: phone, code: code, isLogging: feedback.isAuthenticating)
    }

    var body: some View {
        let palette = AppTheme.palette(for: colorScheme)

        ZStack {
            ScrollView {
                VStack(alignment: .leading, spacing: 0) {
                    brandRow(palette)

                    Text("手机号登录")
                        .font(.system(size: 25, weight: .heavy))
                        .foregroundStyle(palette.textPrimary)
                        .padding(.top, 28)
                    Text("未注册手机号验证后将自动创建账号")
                        .font(.system(size: 12.5))
                        .foregroundStyle(palette.textSecondary)
                        .padding(.top, 7)

                    phoneField(palette)
                        .padding(.top, 26)

                    codeField(palette)
                        .padding(.top, 13)

                    agreementBlock(palette)
                        .padding(.top, 2)

                    loginButton(palette)
                        .padding(.top, 8)

                    if feedback.feedback.visible {
                        authenticationFeedback(palette)
                            .padding(.top, 9)
                    }

                    if let message {
                        Text(message)
                            .font(.system(size: 12))
                            .foregroundStyle(isError ? danger : palette.textSecondary)
                            .padding(.top, 12)
                            .accessibilityAddTraits(.updatesFrequently)
                    }
                }
                .frame(maxWidth: 440)
                .padding(.horizontal, 20)
                .padding(.top, 12)
                .padding(.bottom, 16)
                .frame(maxWidth: .infinity)
            }
            .scrollDismissesKeyboard(.interactively)

            if agreement.isSheetPresented {
                AgreementConsentSheet(
                    onClose: { agreement.closeSheet() },
                    onAgree: handleAgreementAccepted
                )
            }
        }
        .background(palette.page)
        .animation(.easeInOut(duration: 0.2), value: agreement.isSheetPresented)
        .task(id: feedback.isAuthenticating) {
            guard feedback.isAuthenticating else { return }
            while !Task.isCancelled {
                try? await Task.sleep(for: .milliseconds(200))
                feedback.tick()
            }
        }
        .onDisappear {
            agreement.deactivate(current: .login, remaining: [])
        }
    }

    private func brandRow(_ palette: AppPalette) -> some View {
        HStack(spacing: 10) {
            Image("BrandLogo")
                .resizable()
                .scaledToFit()
                .frame(width: 42, height: 42)
                .clipShape(RoundedRectangle(cornerRadius: 12))
                .accessibilityHidden(true)

            VStack(alignment: .leading, spacing: 2) {
                Text("追分竞技")
                    .font(.system(size: 18, weight: .heavy))
                    .foregroundStyle(palette.textPrimary)
                Text("ZHUIFEN")
                    .font(.system(size: 11, weight: .semibold))
                    .tracking(3)
                    .foregroundStyle(palette.textMuted)
            }
        }
    }

    private func phoneField(_ palette: AppPalette) -> some View {
        VStack(alignment: .leading, spacing: 6) {
            Text("手机号")
                .font(.system(size: 12.5, weight: .bold))
                .foregroundStyle(palette.textSecondary)

            HStack(spacing: 11) {
                Text("+86")
                    .font(.system(size: 13.5, weight: .bold))
                    .foregroundStyle(palette.textPrimary)
                Rectangle()
                    .fill(palette.border)
                    .frame(width: 1, height: 22)
                TextField("请输入手机号", text: $phone)
                    .keyboardType(.phonePad)
                    .textContentType(.telephoneNumber)
                    .foregroundStyle(palette.textPrimary)
                    .focused($focusedField, equals: .phone)
                    .disabled(isBusy)
                    .onChange(of: phone) { _, value in
                        phone = String(value.filter { "0123456789".contains($0) }.prefix(11))
                        message = nil
                    }
            }
            .font(.system(size: 14))
            .padding(.horizontal, 12)
            .frame(height: 48)
            .background(palette.subtle, in: RoundedRectangle(cornerRadius: 12))
            .overlay {
                RoundedRectangle(cornerRadius: 12)
                    .stroke(phoneError.isEmpty ? palette.border : danger)
            }

            if !phoneError.isEmpty {
                Text(phoneError)
                    .font(.system(size: 11))
                    .foregroundStyle(danger)
                    .padding(.leading, 3)
            }
        }
    }

    private func codeField(_ palette: AppPalette) -> some View {
        VStack(alignment: .leading, spacing: 6) {
            Text("短信验证码")
                .font(.system(size: 12.5, weight: .bold))
                .foregroundStyle(palette.textSecondary)

            HStack(spacing: 8) {
                TextField("请输入6位验证码", text: $code)
                    .keyboardType(.numberPad)
                    .textContentType(.oneTimeCode)
                    .foregroundStyle(palette.textPrimary)
                    .focused($focusedField, equals: .code)
                    .disabled(isBusy)
                    .onChange(of: code) { _, value in
                        code = String(value.filter { "0123456789".contains($0) }.prefix(6))
                        message = nil
                    }

                TimelineView(.periodic(from: .now, by: 1)) { context in
                    let remaining = secondsUntilResend(at: context.date)
                    let canSend = EntryFunnelRules.canRequestSMS(
                        phone: phone,
                        countdown: remaining,
                        isSending: isBusy
                    )

                    Button {
                        focusedField = nil
                        Task { await sendCode() }
                    } label: {
                        Text(sendCodeText(remaining))
                            .font(.system(size: 11.5, weight: .bold))
                            .fixedSize()
                    }
                    .foregroundStyle(palette.brandStrong)
                    .padding(.horizontal, 10)
                    .frame(height: 32)
                    .background(palette.brandTint, in: RoundedRectangle(cornerRadius: 9))
                    .disabled(!canSend)
                    .opacity(canSend ? 1 : 0.48)
                }
            }
            .font(.system(size: 14))
            .padding(.horizontal, 12)
            .frame(height: 48)
            .background(palette.subtle, in: RoundedRectangle(cornerRadius: 12))
            .overlay {
                RoundedRectangle(cornerRadius: 12)
                    .stroke(codeError.isEmpty ? palette.border : danger)
            }

            if !codeError.isEmpty {
                Text(codeError)
                    .font(.system(size: 11))
                    .foregroundStyle(danger)
                    .padding(.leading, 3)
            }
        }
    }

    private func agreementBlock(_ palette: AppPalette) -> some View {
        VStack(spacing: 0) {
            AgreementConsentRow(
                hasAgreed: agreement.hasAgreed,
                isDisabled: feedback.isAuthenticating,
                onToggle: { agreement.toggle() }
            )
            .frame(maxWidth: .infinity)

            HStack(spacing: 4) {
                Link("《用户协议》", destination: LegalDocumentLink.userAgreement)
                Text("和")
                    .foregroundStyle(palette.textMuted)
                Link("《隐私政策》", destination: LegalDocumentLink.privacyPolicy)
            }
            .font(.system(size: 11))
            .tint(palette.brandStrong)
            .padding(.vertical, 12)
            .disabled(feedback.isAuthenticating)
            .opacity(feedback.isAuthenticating ? 0.45 : 1)
        }
        .frame(maxWidth: .infinity)
    }

    private func loginButton(_ palette: AppPalette) -> some View {
        Button {
            focusedField = nil
            handleSubmit()
        } label: {
            Text(feedback.buttonText)
                .font(.system(size: 15, weight: .bold))
                .foregroundStyle(.white)
                .frame(maxWidth: .infinity)
                .frame(height: 48)
                .background(
                    LinearGradient(
                        colors: [palette.gradientStart, palette.gradientEnd],
                        startPoint: .topLeading,
                        endPoint: .bottomTrailing
                    ),
                    in: RoundedRectangle(cornerRadius: 9)
                )
                .shadow(color: palette.brandStrong.opacity(0.18), radius: 14, y: 7)
        }
        .buttonStyle(.plain)
        .disabled(!canSubmit)
        .opacity(canSubmit ? 1 : 0.45)
    }

    private func authenticationFeedback(_ palette: AppPalette) -> some View {
        HStack(spacing: 8) {
            Circle()
                .fill(AppTheme.primary)
                .frame(width: 8, height: 8)
                .accessibilityHidden(true)
            Text(feedback.feedback.message)
                .font(.system(size: 12, weight: .bold))
                .foregroundStyle(palette.brandStrong)
                .accessibilityAddTraits(.updatesFrequently)
        }
        .padding(.horizontal, 10)
        .padding(.vertical, 9)
        .frame(maxWidth: .infinity, alignment: .leading)
        .background(feedback.isSlow ? palette.brandTint : palette.subtle, in: RoundedRectangle(cornerRadius: 10))
        .overlay {
            RoundedRectangle(cornerRadius: 10)
                .stroke(palette.brandStrong.opacity(feedback.isSlow ? 0.5 : 0.24))
        }
    }

    private func sendCodeText(_ remaining: Int) -> String {
        if remaining > 0 {
            return "\(String(remaining))s 后重试"
        }
        return isSending ? "发送中..." : "获取验证码"
    }

    private func secondsUntilResend(at date: Date) -> Int {
        guard let resendAfter else { return 0 }
        return max(0, Int(ceil(resendAfter.timeIntervalSince(date))))
    }

    private func handleSubmit() {
        switch EntryFunnelRules.loginSubmitGate(
            phone: phone,
            code: code,
            isAgreed: agreement.hasAgreed,
            isLogging: feedback.isAuthenticating
        ) {
        case .blocked:
            break
        case .showAgreementSheet:
            agreement.requestConsent()
        case .submit:
            Task { await logIn() }
        }
    }

    private func handleAgreementAccepted() {
        agreement.accept()
        Task { await logIn() }
    }

    @MainActor
    private func sendCode() async {
        guard EntryFunnelRules.isPhoneValid(phone), secondsUntilResend(at: .now) == 0, !isBusy else { return }
        let requestedPhone = phone
        isSending = true
        message = nil
        defer { isSending = false }

        do {
            try await session.sendSMS(phone: requestedPhone)
            resendAfter = Date().addingTimeInterval(60)
            message = EntryFunnelRules.smsFeedback(phone: requestedPhone)
            isError = false
        } catch {
            message = EntryFunnelRules.authErrorMessage(error, fallback: "发送失败，请重试")
            isError = true
        }
    }

    @MainActor
    private func logIn() async {
        guard EntryFunnelRules.canAttemptLogin(phone: phone, code: code, isLogging: feedback.isAuthenticating) else {
            return
        }
        guard agreement.hasAgreed else {
            agreement.requestConsent()
            return
        }

        message = nil
        isError = false
        feedback.begin()
        defer { feedback.end() }

        do {
            try await session.login(phone: phone, code: code)
            agreement.markWelcomeViewed()
            dismiss()
        } catch {
            message = EntryFunnelRules.authErrorMessage(error, fallback: "登录失败，请重试")
            isError = true
        }
    }
}
