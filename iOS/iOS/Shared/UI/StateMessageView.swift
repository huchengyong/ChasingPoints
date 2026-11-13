import SwiftUI

/// 空态 / 错误 / 登录引导：图标 → 标题 → 说明 → 可选操作。
struct StateMessageView: View {
    @Environment(\.colorScheme) private var colorScheme

    private let icon: String
    private let title: String
    private let detail: String?
    private let actionTitle: String?
    private let action: (() -> Void)?

    init(
        icon: String,
        title: String,
        detail: String? = nil,
        actionTitle: String? = nil,
        action: (() -> Void)? = nil
    ) {
        self.icon = icon
        self.title = title
        self.detail = detail
        self.actionTitle = actionTitle
        self.action = action
    }

    var body: some View {
        let palette = AppTheme.palette(for: colorScheme)

        VStack(spacing: 12) {
            Image(systemName: icon)
                .font(.system(size: 40))
                .foregroundStyle(palette.textMuted)
                .accessibilityHidden(true)

            VStack(spacing: 6) {
                Text(title)
                    .font(.system(size: 17, weight: .semibold))
                    .foregroundStyle(palette.textPrimary)
                if let detail {
                    Text(detail)
                        .font(.system(size: 14))
                        .foregroundStyle(palette.textSecondary)
                        .multilineTextAlignment(.center)
                }
            }
            .accessibilityElement(children: .combine)

            if let actionTitle, let action {
                Button(actionTitle, action: action)
                    .buttonStyle(.appPill)
                    .padding(.top, 4)
            }
        }
        .padding(24)
        .frame(maxWidth: .infinity, maxHeight: .infinity)
    }
}
