import SwiftUI

/// 设置类页面共用的列表行元素。
struct SettingsIconBadge: View {
    @Environment(\.colorScheme) private var colorScheme

    let symbol: String
    var tint: Color = AppTheme.primary

    private var palette: AppPalette { AppTheme.palette(for: colorScheme) }

    var body: some View {
        Image(systemName: symbol)
            .font(.system(size: 15))
            .foregroundStyle(tint)
            .frame(width: 32, height: 32)
            .background(palette.brandTint, in: RoundedRectangle(cornerRadius: 9))
            .accessibilityHidden(true)
    }
}

struct SettingsLabel: View {
    @Environment(\.colorScheme) private var colorScheme

    let symbol: String
    let title: String
    var subtitle: String?
    var tint: Color = AppTheme.primary
    var titleColor: Color?

    private var palette: AppPalette { AppTheme.palette(for: colorScheme) }

    var body: some View {
        HStack(spacing: 10) {
            SettingsIconBadge(symbol: symbol, tint: tint)

            VStack(alignment: .leading, spacing: 3) {
                Text(title)
                    .font(.system(size: 15))
                    .foregroundStyle(titleColor ?? palette.textPrimary)
                if let subtitle {
                    Text(subtitle)
                        .font(.system(size: 12))
                        .foregroundStyle(palette.textSecondary)
                        .fixedSize(horizontal: false, vertical: true)
                }
            }
        }
    }
}

struct SettingsChevron: View {
    @Environment(\.colorScheme) private var colorScheme

    var body: some View {
        Image(systemName: "chevron.right")
            .font(.system(size: 12, weight: .semibold))
            .foregroundStyle(AppTheme.palette(for: colorScheme).textMuted)
            .accessibilityHidden(true)
    }
}

/// 只展示开关状态，点击由所在行处理。
struct SettingsSwitch: View {
    @Environment(\.colorScheme) private var colorScheme

    let isOn: Bool

    private var palette: AppPalette { AppTheme.palette(for: colorScheme) }

    var body: some View {
        Capsule()
            .fill(isOn ? AppTheme.primary : palette.border)
            .frame(width: 44, height: 26)
            .overlay(alignment: isOn ? .trailing : .leading) {
                Circle()
                    .fill(.white)
                    .frame(width: 22, height: 22)
                    .padding(2)
                    .shadow(color: .black.opacity(0.16), radius: 2, y: 1)
            }
            .animation(.easeInOut(duration: 0.2), value: isOn)
            .accessibilityHidden(true)
    }
}

struct SettingsDivider: View {
    @Environment(\.colorScheme) private var colorScheme

    var body: some View {
        Rectangle()
            .fill(AppTheme.palette(for: colorScheme).border)
            .frame(height: 1)
            .padding(.leading, 58)
    }
}
