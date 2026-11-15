import SwiftUI

/// 区块标题行：标题 + 可选小字说明 + 可选右侧「更多」。
struct SectionHeader: View {
    @Environment(\.colorScheme) private var colorScheme

    private let title: String
    private let caption: String?
    private let moreTitle: String?
    private let moreAction: (() -> Void)?

    init(
        title: String,
        caption: String? = nil,
        moreTitle: String? = nil,
        moreAction: (() -> Void)? = nil
    ) {
        self.title = title
        self.caption = caption
        self.moreTitle = moreTitle
        self.moreAction = moreAction
    }

    var body: some View {
        let palette = AppTheme.palette(for: colorScheme)

        HStack(alignment: .firstTextBaseline, spacing: 8) {
            Text(title)
                .font(.system(size: 18, weight: .semibold))
                .foregroundStyle(palette.textPrimary)
            if let caption {
                Text(caption)
                    .font(.system(size: 13))
                    .foregroundStyle(palette.textSecondary)
            }
            Spacer(minLength: 8)
            if let moreTitle, let moreAction {
                Button(moreTitle, action: moreAction)
                    .font(.system(size: 13, weight: .semibold))
                    .foregroundStyle(palette.brandStrong)
            }
        }
    }
}
