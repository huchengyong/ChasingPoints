import SwiftUI

/// 块级主按钮：撑满宽度、高 48、圆角 12、品牌金背景、白字。
struct PrimaryButtonStyle: ButtonStyle {
    func makeBody(configuration: Configuration) -> some View {
        StyledBody(configuration: configuration)
    }

    private struct StyledBody: View {
        @Environment(\.isEnabled) private var isEnabled
        let configuration: Configuration

        var body: some View {
            configuration.label
                .font(.system(size: 16, weight: .semibold))
                .foregroundStyle(.white)
                .frame(maxWidth: .infinity)
                .frame(height: 48)
                .background(AppTheme.primary, in: RoundedRectangle(cornerRadius: 12))
                .opacity(isEnabled ? (configuration.isPressed ? 0.85 : 1) : 0.5)
        }
    }
}

/// 块级次按钮：subtle 背景 + border 描边 + brandStrong 文字。
struct SecondaryButtonStyle: ButtonStyle {
    func makeBody(configuration: Configuration) -> some View {
        StyledBody(configuration: configuration)
    }

    private struct StyledBody: View {
        @Environment(\.isEnabled) private var isEnabled
        @Environment(\.colorScheme) private var colorScheme
        let configuration: Configuration

        var body: some View {
            let palette = AppTheme.palette(for: colorScheme)

            configuration.label
                .font(.system(size: 16, weight: .semibold))
                .foregroundStyle(palette.brandStrong)
                .frame(maxWidth: .infinity)
                .frame(height: 48)
                .background(palette.subtle, in: RoundedRectangle(cornerRadius: 12))
                .overlay {
                    RoundedRectangle(cornerRadius: 12).stroke(palette.border)
                }
                .opacity(isEnabled ? (configuration.isPressed ? 0.85 : 1) : 0.5)
        }
    }
}

/// 紧凑胶囊按钮：高 36、水平内边距 20、品牌金背景、白字 14 semibold。
struct PillButtonStyle: ButtonStyle {
    func makeBody(configuration: Configuration) -> some View {
        StyledBody(configuration: configuration)
    }

    private struct StyledBody: View {
        @Environment(\.isEnabled) private var isEnabled
        let configuration: Configuration

        var body: some View {
            configuration.label
                .font(.system(size: 14, weight: .semibold))
                .foregroundStyle(.white)
                .padding(.horizontal, 20)
                .frame(height: 36)
                .background(AppTheme.primary, in: Capsule())
                .opacity(isEnabled ? (configuration.isPressed ? 0.85 : 1) : 0.5)
        }
    }
}

/// 块级破坏性按钮：danger 文字 + card 背景 + border 描边。
struct DangerButtonStyle: ButtonStyle {
    func makeBody(configuration: Configuration) -> some View {
        StyledBody(configuration: configuration)
    }

    private struct StyledBody: View {
        @Environment(\.isEnabled) private var isEnabled
        @Environment(\.colorScheme) private var colorScheme
        let configuration: Configuration

        var body: some View {
            let palette = AppTheme.palette(for: colorScheme)

            configuration.label
                .font(.system(size: 16, weight: .semibold))
                .foregroundStyle(AppTheme.danger)
                .frame(maxWidth: .infinity)
                .frame(height: 48)
                .background(palette.card, in: RoundedRectangle(cornerRadius: 12))
                .overlay {
                    RoundedRectangle(cornerRadius: 12).stroke(palette.border)
                }
                .opacity(isEnabled ? (configuration.isPressed ? 0.85 : 1) : 0.5)
        }
    }
}

extension ButtonStyle where Self == PrimaryButtonStyle {
    static var appPrimary: PrimaryButtonStyle { PrimaryButtonStyle() }
}

extension ButtonStyle where Self == SecondaryButtonStyle {
    static var appSecondary: SecondaryButtonStyle { SecondaryButtonStyle() }
}

extension ButtonStyle where Self == PillButtonStyle {
    static var appPill: PillButtonStyle { PillButtonStyle() }
}

extension ButtonStyle where Self == DangerButtonStyle {
    static var appDanger: DangerButtonStyle { DangerButtonStyle() }
}
