import SwiftUI

/// 应用内静态协议页，对应小程序 `subPages/agreement/userAgreement.vue` / `privacyPolicy.vue`。
struct AgreementView: View {
    @Environment(\.colorScheme) private var colorScheme

    let document: AgreementDocument

    private var palette: AppPalette { AppTheme.palette(for: colorScheme) }

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 22) {
                VStack(spacing: 8) {
                    Text(document.title)
                        .font(.system(size: 22, weight: .bold))
                        .foregroundStyle(palette.textPrimary)
                        .multilineTextAlignment(.center)
                    Text(document.updatedAt)
                        .font(.system(size: 12))
                        .foregroundStyle(palette.textMuted)
                }
                .frame(maxWidth: .infinity)

                ForEach(Array(document.sections.enumerated()), id: \.offset) { _, section in
                    VStack(alignment: .leading, spacing: 8) {
                        Text(section.title)
                            .font(.system(size: 15, weight: .semibold))
                            .foregroundStyle(palette.textPrimary)
                        ForEach(Array(section.paragraphs.enumerated()), id: \.offset) { _, paragraph in
                            Text(paragraph)
                                .font(.system(size: 14))
                                .foregroundStyle(palette.textSecondary)
                                .lineSpacing(6)
                                .fixedSize(horizontal: false, vertical: true)
                        }
                    }
                }
            }
            .padding(.horizontal, 20)
            .padding(.vertical, 18)
        }
        .background(palette.page.ignoresSafeArea())
        .navigationTitle(document.navigationTitle)
        .navigationBarTitleDisplayMode(.inline)
    }
}
