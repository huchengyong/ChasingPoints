import SwiftUI

struct LoginView: View {
    @EnvironmentObject var session: SessionStore
    @Environment(\.colorScheme) private var colorScheme
    @Environment(\.dismiss) private var dismiss

    @State private var phone = ""
    @State private var code = ""
    @State private var hasAgreed = false
    @State private var isSending = false
    @State private var isLoggingIn = false
    @State private var resendAfter: Date?
    @State private var feedback: String?
    @State private var isError = false
    @FocusState private var focusedField: Field?

    private enum Field {
        case phone, code
    }

    private let danger = Color(red: 239.0 / 255, green: 68.0 / 255, blue: 68.0 / 255)

    private var isPhoneValid: Bool {
        phone.range(of: #"^1[3-9][0-9]{9}$"#, options: .regularExpression) != nil
    }

    private var isCodeValid: Bool {
        code.count == 6 && code.allSatisfy { "0123456789".contains($0) }
    }

    private var isBusy: Bool { isSending || isLoggingIn }

    var body: some View {
        let palette = AppTheme.palette(for: colorScheme)

        ScrollView {
            VStack(alignment: .leading, spacing: 0) {
                HStack(spacing: 10) {
                    Image("BrandLogo")
                        .resizable()
                        .scaledToFit()
                        .frame(width: 42, height: 42)
                        .clipShape(RoundedRectangle(cornerRadius: 12))
                        .accessibilityHidden(true)

                    VStack(alignment: .leading, spacing: 4) {
                        Text("追分竞技")
                            .font(.system(size: 24, weight: .bold))
                            .foregroundStyle(palette.textPrimary)
                        Text("CHASING POINTS")
                            .font(.system(size: 11, weight: .semibold))
                            .tracking(2)
                            .foregroundStyle(AppTheme.primary)
                    }
                }
                .padding(.bottom, 64)

                Text("手机号登录")
                    .font(.system(size: 30, weight: .bold))
                    .foregroundStyle(palette.textPrimary)
                Text("未注册手机号验证后将自动创建账号")
                    .font(.system(size: 14))
                    .foregroundStyle(palette.textSecondary)
                    .padding(.top, 10)
                    .padding(.bottom, 32)

                Text("手机号")
                    .font(.system(size: 14, weight: .medium))
                    .foregroundStyle(palette.textPrimary)

                HStack(spacing: 14) {
                    Text("+86")
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
                            feedback = nil
                        }
                }
                .font(.system(size: 16))
                .padding(.horizontal, 16)
                .frame(height: 54)
                .background(palette.subtle, in: RoundedRectangle(cornerRadius: 12))
                .overlay {
                    RoundedRectangle(cornerRadius: 12)
                        .stroke(!phone.isEmpty && !isPhoneValid ? danger : palette.border)
                }
                .padding(.top, 10)

                if !phone.isEmpty && !isPhoneValid {
                    Text("请输入正确的手机号")
                        .font(.system(size: 12))
                        .foregroundStyle(danger)
                        .padding(.top, 6)
                }

                Text("短信验证码")
                    .font(.system(size: 14, weight: .medium))
                    .foregroundStyle(palette.textPrimary)
                    .padding(.top, 24)

                HStack(spacing: 8) {
                    TextField("请输入6位验证码", text: $code)
                        .keyboardType(.numberPad)
                        .textContentType(.oneTimeCode)
                        .foregroundStyle(palette.textPrimary)
                        .focused($focusedField, equals: .code)
                        .disabled(isBusy)
                        .onChange(of: code) { _, value in
                            code = String(value.filter { "0123456789".contains($0) }.prefix(6))
                            feedback = nil
                        }

                    TimelineView(.periodic(from: .now, by: 1)) { context in
                        let remaining = secondsUntilResend(at: context.date)

                        Button {
                            focusedField = nil
                            Task { await sendCode() }
                        } label: {
                            Text(remaining > 0 ? "\(remaining)s 后重试" : isSending ? "发送中..." : "获取验证码")
                                .font(.system(size: 14, weight: .semibold))
                                .fixedSize()
                        }
                        .foregroundStyle(AppTheme.primary)
                        .disabled(!isPhoneValid || remaining > 0 || isBusy)
                        .opacity(!isPhoneValid || remaining > 0 || isBusy ? 0.5 : 1)
                    }
                }
                .font(.system(size: 16))
                .padding(.horizontal, 16)
                .frame(height: 54)
                .background(palette.subtle, in: RoundedRectangle(cornerRadius: 12))
                .overlay {
                    RoundedRectangle(cornerRadius: 12)
                        .stroke(!code.isEmpty && !isCodeValid ? danger : palette.border)
                }
                .padding(.top, 10)

                if !code.isEmpty && !isCodeValid {
                    Text("请输入6位验证码")
                        .font(.system(size: 12))
                        .foregroundStyle(danger)
                        .padding(.top, 6)
                }

                VStack(alignment: .leading, spacing: 8) {
                    Button {
                        hasAgreed.toggle()
                        feedback = nil
                    } label: {
                        Label("我已阅读并同意", systemImage: hasAgreed ? "checkmark.square.fill" : "square")
                            .font(.system(size: 13))
                    }
                    .buttonStyle(.plain)
                    .foregroundStyle(hasAgreed ? AppTheme.primary : palette.textSecondary)
                    .disabled(isBusy)
                    .accessibilityLabel("我已阅读并同意用户协议和隐私政策")
                    .accessibilityValue(hasAgreed ? "已同意" : "未同意")

                    HStack(spacing: 4) {
                        Link("《用户协议》", destination: URL(string: "https://www.zhuifen.cn/agreement")!)
                        Text("和")
                            .foregroundStyle(palette.textSecondary)
                        Link("《隐私政策》", destination: URL(string: "https://www.zhuifen.cn/privacy")!)
                    }
                    .font(.system(size: 13))
                    .tint(AppTheme.primary)
                }
                .padding(.top, 28)

                Button {
                    focusedField = nil
                    Task { await logIn() }
                } label: {
                    Text(isLoggingIn ? "正在安全登录…" : "登录 / 注册")
                        .font(.system(size: 16, weight: .semibold))
                        .frame(maxWidth: .infinity)
                        .frame(height: 52)
                }
                .foregroundStyle(.white)
                .background(AppTheme.primary, in: RoundedRectangle(cornerRadius: 12))
                .disabled(!isPhoneValid || !isCodeValid || isBusy)
                .opacity(!isPhoneValid || !isCodeValid || isBusy ? 0.55 : 1)
                .padding(.top, 28)

                if let feedback {
                    Text(feedback)
                        .font(.system(size: 13))
                        .foregroundStyle(isError ? danger : palette.textSecondary)
                        .padding(.top, 14)
                        .accessibilityAddTraits(.updatesFrequently)
                }
            }
            .frame(maxWidth: 440)
            .padding(.horizontal, 24)
            .padding(.top, 40)
            .padding(.bottom, 32)
            .frame(maxWidth: .infinity)
        }
        .scrollDismissesKeyboard(.interactively)
        .background(palette.page)
    }

    private func secondsUntilResend(at date: Date) -> Int {
        guard let resendAfter else { return 0 }
        return max(0, Int(ceil(resendAfter.timeIntervalSince(date))))
    }

    @MainActor
    private func sendCode() async {
        guard isPhoneValid, secondsUntilResend(at: .now) == 0, !isBusy else { return }
        let requestedPhone = phone
        isSending = true
        feedback = nil
        defer { isSending = false }

        do {
            try await session.sendSMS(phone: requestedPhone)
            resendAfter = Date().addingTimeInterval(60)
            feedback = "验证码已发送，请注意查收"
            isError = false
        } catch {
            feedback = error.localizedDescription
            isError = true
        }
    }

    @MainActor
    private func logIn() async {
        guard isPhoneValid, isCodeValid, !isBusy else { return }
        guard hasAgreed else {
            feedback = "请先阅读并同意用户协议和隐私政策"
            isError = true
            return
        }

        isLoggingIn = true
        feedback = nil
        defer { isLoggingIn = false }

        do {
            try await session.login(phone: phone, code: code)
            dismiss()
        } catch {
            feedback = error.localizedDescription
            isError = true
        }
    }
}
