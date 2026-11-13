import SwiftUI

/// 未迁移页面的占位。
struct MigrationPlaceholderView: View {
    @Environment(\.colorScheme) private var colorScheme

    let title: String

    init(title: String) {
        self.title = title
    }

    var body: some View {
        let palette = AppTheme.palette(for: colorScheme)

        VStack(spacing: 16) {
            Image(systemName: "hammer.fill")
                .font(.system(size: 44))
                .foregroundStyle(AppTheme.primary)
            Text("\(title)页面正在迁移")
                .font(.title3.bold())
                .foregroundStyle(palette.textPrimary)
            Text("此功能尚未在原生版开放")
                .font(.subheadline)
                .foregroundStyle(palette.textSecondary)
        }
        .padding(24)
        .frame(maxWidth: .infinity, maxHeight: .infinity)
        .background(palette.page.ignoresSafeArea())
        .navigationTitle(title)
        .navigationBarTitleDisplayMode(.inline)
    }
}
