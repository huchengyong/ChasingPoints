import SwiftUI

extension View {
    /// 普通卡片：surface-card 背景 + 1px border-default 描边 + 16 圆角。
    func appCard(padding: CGFloat = 16) -> some View {
        modifier(AppCardModifier(padding: padding))
    }
}

private struct AppCardModifier: ViewModifier {
    @Environment(\.colorScheme) private var colorScheme

    let padding: CGFloat

    func body(content: Content) -> some View {
        let palette = AppTheme.palette(for: colorScheme)

        content
            .padding(padding)
            .background(palette.card, in: RoundedRectangle(cornerRadius: 16))
            .overlay {
                RoundedRectangle(cornerRadius: 16).stroke(palette.border, lineWidth: 1)
            }
    }
}
