import SwiftUI

/// 首次启动的欢迎页，对应 `pages/welcome/index.vue` 的手机号登录形态。
struct WelcomeView: View {
    @Environment(\.colorScheme) private var colorScheme
    @StateObject private var agreement: EntryAgreementModel

    private let onOpenLogin: () -> Void
    private let onBrowse: () -> Void

    init(
        agreement: EntryAgreementModel? = nil,
        onOpenLogin: @escaping () -> Void = {},
        onBrowse: @escaping () -> Void = {}
    ) {
        _agreement = StateObject(wrappedValue: agreement ?? EntryAgreementModel())
        self.onOpenLogin = onOpenLogin
        self.onBrowse = onBrowse
    }

    private var palette: AppPalette { AppTheme.palette(for: colorScheme) }

    private struct PreviewStat: Identifiable {
        let icon: String
        let title: String
        let value: String
        let detail: String

        var id: String { title }
    }

    private let previewStats = [
        PreviewStat(icon: "medal.fill", title: "今日战绩", value: "12 胜 5 负", detail: "最近 7 天稳定输出"),
        PreviewStat(icon: "flame.fill", title: "精彩战报", value: "3 份", detail: "刚结束的强强对局"),
        PreviewStat(icon: "person.2.fill", title: "常约对手", value: "8 位", detail: "常一起较量的球友"),
    ]

    var body: some View {
        ZStack {
            LinearGradient(
                colors: [palette.subtle, palette.page],
                startPoint: .top,
                endPoint: .bottom
            )
            .ignoresSafeArea()

            ScrollView {
                VStack(alignment: .leading, spacing: 0) {
                    brandRow

                    heroSection
                        .padding(.top, 20)

                    previewCard
                        .padding(.top, 12)

                    buttonSection
                        .padding(.top, 16)

                    agreementBlock
                        .padding(.top, 8)
                }
                .frame(maxWidth: 440)
                .padding(.horizontal, 20)
                .padding(.top, 12)
                .padding(.bottom, 12)
                .frame(maxWidth: .infinity)
            }

            if agreement.isSheetPresented {
                AgreementConsentSheet(
                    onClose: { agreement.closeSheet() },
                    onAgree: handleAgreementAccepted
                )
            }
        }
        .animation(.easeInOut(duration: 0.2), value: agreement.isSheetPresented)
    }

    private var brandRow: some View {
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

    private var heroSection: some View {
        VStack(alignment: .leading, spacing: 0) {
            Text("台球竞技记录")
                .font(.system(size: 10, weight: .bold))
                .tracking(2)
                .foregroundStyle(palette.brandStrong)
                .padding(.bottom, 7)

            Group {
                Text("每一杆，")
                Text("都值得被记录")
            }
            .font(.system(size: 27, weight: .heavy))
            .foregroundStyle(palette.textPrimary)

            VStack(alignment: .leading, spacing: 0) {
                Text("记录战绩、生成战报，")
                Text("找到真正旗鼓相当的对手。")
            }
            .font(.system(size: 14))
            .foregroundStyle(palette.textSecondary)
            .padding(.top, 8)
        }
    }

    private var previewCard: some View {
        VStack(spacing: 0) {
            ForEach(Array(previewStats.enumerated()), id: \.element.id) { index, stat in
                HStack(spacing: 10) {
                    RoundedRectangle(cornerRadius: 13)
                        .fill(palette.brandTint)
                        .frame(width: 40, height: 40)
                        .overlay {
                            Image(systemName: stat.icon)
                                .font(.system(size: 16, weight: .semibold))
                                .foregroundStyle(palette.brandStrong)
                        }

                    VStack(alignment: .leading, spacing: 2) {
                        Text(stat.title)
                            .font(.system(size: 15, weight: .semibold))
                            .foregroundStyle(palette.textPrimary)
                        Text(stat.detail)
                            .font(.system(size: 11))
                            .foregroundStyle(palette.textMuted)
                    }

                    Spacer(minLength: 8)

                    Text(stat.value)
                        .font(.system(size: 16, weight: .bold))
                        .foregroundStyle(palette.textPrimary)
                }
                .padding(.vertical, 15)
                .overlay(alignment: .top) {
                    if index > 0 {
                        Rectangle()
                            .fill(palette.border)
                            .frame(height: 1)
                    }
                }
            }
        }
        .padding(.horizontal, 14)
        .background(palette.card, in: RoundedRectangle(cornerRadius: 16))
        .overlay {
            RoundedRectangle(cornerRadius: 16).stroke(palette.border)
        }
        .overlay(alignment: .topTrailing) {
            Text("示例数据")
                .font(.system(size: 10, weight: .bold))
                .foregroundStyle(palette.brandStrong)
                .padding(.horizontal, 8)
                .padding(.vertical, 2)
                .background(palette.card, in: Capsule())
                .overlay {
                    Capsule().stroke(palette.brandStrong.opacity(0.35))
                }
                .offset(x: -14, y: -9)
        }
        .accessibilityHidden(true)
    }

    private var buttonSection: some View {
        VStack(spacing: 6) {
            Button(action: handlePrimary) {
                Text(EntryFunnelRules.welcomeActions.primaryText)
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
            .accessibilityHint("打开手机号登录页")

            Button(action: handleBrowse) {
                Text(EntryFunnelRules.welcomeActions.tertiaryText)
                    .font(.system(size: 12, weight: .semibold))
                    .foregroundStyle(palette.textMuted)
                    .frame(maxWidth: .infinity)
                    .frame(height: 34)
            }
            .buttonStyle(.plain)
        }
    }

    private var agreementBlock: some View {
        VStack(spacing: 0) {
            AgreementConsentRow(
                hasAgreed: agreement.hasAgreed,
                isDisabled: false,
                onToggle: { agreement.toggle() }
            )
            .frame(maxWidth: .infinity)

            HStack(spacing: 4) {
                NavigationLink("《用户协议》", value: AppRoute.userAgreement)
                Text("和")
                    .foregroundStyle(palette.textMuted)
                NavigationLink("《隐私政策》", value: AppRoute.privacyPolicy)
            }
            .font(.system(size: 11))
            .tint(palette.brandStrong)
            .padding(.vertical, 12)
        }
        .frame(maxWidth: .infinity)
    }

    private func handlePrimary() {
        switch EntryFunnelRules.welcomePrimaryOutcome(isAgreed: agreement.hasAgreed) {
        case .showAgreementSheet:
            agreement.requestConsent()
        case .openLogin:
            agreement.markWelcomeViewed()
            onOpenLogin()
        }
    }

    private func handleAgreementAccepted() {
        agreement.accept()
        agreement.markWelcomeViewed()
        onOpenLogin()
    }

    private func handleBrowse() {
        agreement.markWelcomeViewed()
        agreement.deactivate(current: .welcome, remaining: [])
        onBrowse()
    }
}
