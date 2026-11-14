import SwiftUI

/// 小程序里没有原生协议页路由，沿用现有外链。
enum LegalDocumentLink {
    static let userAgreement = URL(string: "https://www.zhuifen.cn/agreement")!
    static let privacyPolicy = URL(string: "https://www.zhuifen.cn/privacy")!
}

/// 未勾选协议时点主按钮弹出的底部确认层，对应 `agreementConsentSheet.vue`。
struct AgreementConsentSheet: View {
    @Environment(\.colorScheme) private var colorScheme

    let onClose: () -> Void
    let onAgree: () -> Void

    private var palette: AppPalette { AppTheme.palette(for: colorScheme) }

    var body: some View {
        ZStack(alignment: .bottom) {
            Color.black.opacity(0.42)
                .ignoresSafeArea()
                .onTapGesture(perform: onClose)
                .accessibilityHidden(true)

            VStack(spacing: 0) {
                Capsule()
                    .fill(palette.textMuted.opacity(0.35))
                    .frame(width: 44, height: 4)
                    .padding(.bottom, 14)

                Text("请阅读并同意以下条款")
                    .font(.system(size: 17, weight: .bold))
                    .foregroundStyle(palette.textPrimary)

                Text("继续使用前，请先阅读并同意《用户协议》和《隐私政策》。")
                    .font(.system(size: 13.5))
                    .foregroundStyle(palette.textSecondary)
                    .multilineTextAlignment(.center)
                    .padding(.top, 8)

                HStack(spacing: 5) {
                    Link("用户协议", destination: LegalDocumentLink.userAgreement)
                    Text("和")
                        .foregroundStyle(palette.textMuted)
                    Link("隐私政策", destination: LegalDocumentLink.privacyPolicy)
                }
                .font(.system(size: 13))
                .tint(palette.brandStrong)
                .padding(.top, 12)

                Button(action: onAgree) {
                    Text("同意并继续")
                        .font(.system(size: 16, weight: .bold))
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
                }
                .padding(.top, 16)
            }
            .padding(.horizontal, 16)
            .padding(.top, 12)
            .padding(.bottom, 16)
            .background {
                UnevenRoundedRectangle(topLeadingRadius: 16, topTrailingRadius: 16)
                    .fill(palette.card)
                    .ignoresSafeArea(edges: .bottom)
            }
        }
        .transition(.opacity)
    }
}

/// 欢迎页与登录页共用的协议勾选行。
struct AgreementConsentRow: View {
    @Environment(\.colorScheme) private var colorScheme

    let hasAgreed: Bool
    let isDisabled: Bool
    let onToggle: () -> Void

    private var palette: AppPalette { AppTheme.palette(for: colorScheme) }

    var body: some View {
        Button(action: onToggle) {
            HStack(spacing: 5) {
                RoundedRectangle(cornerRadius: 4)
                    .fill(hasAgreed ? AppTheme.primary : palette.card)
                    .overlay {
                        RoundedRectangle(cornerRadius: 4)
                            .stroke(hasAgreed ? AppTheme.primary : palette.brandStrong, lineWidth: 1)
                    }
                    .overlay {
                        if hasAgreed {
                            Image(systemName: "checkmark")
                                .font(.system(size: 10, weight: .bold))
                                .foregroundStyle(.white)
                        }
                    }
                    .frame(width: 18, height: 18)

                Text("我已阅读并同意")
                    .font(.system(size: 11))
                    .foregroundStyle(palette.textMuted)
            }
            .padding(.vertical, 12)
            .contentShape(Rectangle())
        }
        .buttonStyle(.plain)
        .disabled(isDisabled)
        .opacity(isDisabled ? 0.45 : 1)
        .accessibilityLabel("我已阅读并同意用户协议和隐私政策")
        .accessibilityValue(hasAgreed ? "已同意" : "未同意")
    }
}
