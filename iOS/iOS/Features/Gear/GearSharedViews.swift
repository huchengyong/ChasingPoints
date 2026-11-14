import SwiftUI

/// 装备图的底板固定用浅色：球杆、皮头素材多为深色，暗色底板下会糊成一片。
enum GearStyle {
    static let imageBoard = Color(hex: 0xF8F5EE)
}

/// 装备图：优先目录图，加载失败回退按类别的默认图。
struct GearImageView: View {
    let source: GearImageSource

    var body: some View {
        ZStack {
            GearStyle.imageBoard
            let urls = GearDisplayRules.image(for: source)
            if let src = urls.src {
                AsyncImage(url: src) { phase in
                    if case .success(let image) = phase {
                        image.resizable().aspectRatio(contentMode: .fit)
                    } else {
                        fallback(urls.fallback)
                    }
                }
            } else {
                fallback(urls.fallback)
            }
        }
    }

    @ViewBuilder
    private func fallback(_ url: URL?) -> some View {
        if let url {
            AsyncImage(url: url) { phase in
                if case .success(let image) = phase {
                    image.resizable().aspectRatio(contentMode: .fit)
                }
            }
        }
    }
}

/// 流式换行布局：小数量的标签、筛选 chip 用它换行。
struct WrappingLayout: Layout {
    var spacing: CGFloat = 8

    func sizeThatFits(proposal: ProposedViewSize, subviews: Subviews, cache: inout ()) -> CGSize {
        let maxWidth = proposal.width ?? .infinity
        var x: CGFloat = 0
        var y: CGFloat = 0
        var rowHeight: CGFloat = 0
        for subview in subviews {
            let size = subview.sizeThatFits(.unspecified)
            if x > 0, x + size.width > maxWidth {
                x = 0
                y += rowHeight + spacing
                rowHeight = 0
            }
            x += size.width + spacing
            rowHeight = max(rowHeight, size.height)
        }
        return CGSize(width: maxWidth.isFinite ? maxWidth : max(0, x - spacing), height: y + rowHeight)
    }

    func placeSubviews(in bounds: CGRect, proposal: ProposedViewSize, subviews: Subviews, cache: inout ()) {
        var x = bounds.minX
        var y = bounds.minY
        var rowHeight: CGFloat = 0
        for subview in subviews {
            let size = subview.sizeThatFits(.unspecified)
            if x > bounds.minX, x + size.width > bounds.maxX {
                x = bounds.minX
                y += rowHeight + spacing
                rowHeight = 0
            }
            subview.place(at: CGPoint(x: x, y: y), proposal: ProposedViewSize(size))
            x += size.width + spacing
            rowHeight = max(rowHeight, size.height)
        }
    }
}

struct GearChip: View {
    @Environment(\.colorScheme) private var colorScheme

    let title: String
    var isActive = false

    var body: some View {
        let palette = AppTheme.palette(for: colorScheme)
        Text(title)
            .font(.system(size: 12, weight: isActive ? .semibold : .regular))
            .foregroundStyle(isActive ? .white : palette.textSecondary)
            .padding(.horizontal, 12)
            .padding(.vertical, 5)
            .background(isActive ? AppTheme.primary : palette.subtle, in: Capsule())
            .overlay { Capsule().stroke(isActive ? AppTheme.primary : palette.border) }
    }
}

struct GearCardHead: View {
    @Environment(\.colorScheme) private var colorScheme

    let title: String
    var caption: String?

    var body: some View {
        let palette = AppTheme.palette(for: colorScheme)
        HStack(alignment: .firstTextBaseline, spacing: 8) {
            Text(title)
                .font(.system(size: 15, weight: .bold))
                .foregroundStyle(palette.textPrimary)
            Spacer(minLength: 0)
            if let caption {
                Text(caption)
                    .font(.system(size: 11))
                    .foregroundStyle(palette.textMuted)
            }
        }
    }
}

/// 有数据时刷新失败：保留内容，顶部提示并允许重试。
struct GearRefreshBanner: View {
    @Environment(\.colorScheme) private var colorScheme

    let message: String
    let retry: () -> Void

    var body: some View {
        HStack(spacing: 10) {
            Text(message)
                .font(.system(size: 12))
                .foregroundStyle(AppTheme.danger)
            Spacer(minLength: 8)
            Button("重试", action: retry)
                .font(.system(size: 12, weight: .semibold))
                .foregroundStyle(AppTheme.danger)
        }
        .padding(.horizontal, 11)
        .padding(.vertical, 9)
        .background(AppTheme.danger.opacity(0.1), in: RoundedRectangle(cornerRadius: 8))
    }
}

struct GearNoticeCard: View {
    @Environment(\.colorScheme) private var colorScheme

    let title: String?
    let text: String

    var body: some View {
        let palette = AppTheme.palette(for: colorScheme)
        VStack(alignment: .leading, spacing: 4) {
            if let title {
                Text(title)
                    .font(.system(size: 13, weight: .semibold))
                    .foregroundStyle(palette.textPrimary)
            }
            Text(text)
                .font(.system(size: 12))
                .foregroundStyle(palette.textSecondary)
                .lineSpacing(2)
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .padding(.horizontal, 13)
        .padding(.vertical, 11)
        .background(palette.subtle, in: RoundedRectangle(cornerRadius: 8))
        .overlay { RoundedRectangle(cornerRadius: 8).stroke(palette.border) }
    }
}
