import SwiftUI

/// 术语词典：本地过滤术语名称与解释。
struct GlossaryView: View {
    @Environment(\.colorScheme) private var colorScheme
    @StateObject private var loader: ResourceLoader<[GlossaryItem]>
    @State private var filterText = ""

    init(session: SessionStore, loader: ResourceLoader<[GlossaryItem]>? = nil) {
        if let loader {
            _loader = StateObject(wrappedValue: loader)
        } else {
            let api = RulesAPI(session: session)
            _loader = StateObject(wrappedValue: ResourceLoader(session: session, access: .anyone) {
                try await api.glossary()
            })
        }
    }

    private var palette: AppPalette { AppTheme.palette(for: colorScheme) }

    private var isFiltering: Bool {
        !filterText.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty
    }

    var body: some View {
        VStack(spacing: 0) {
            searchBar
                .padding(.horizontal, 16)
                .padding(.vertical, 12)

            LoadableContent(loader: loader, requiresLogin: false, resource: "术语词典") { items in
                let filtered = GlossaryFilter.filter(items, keyword: filterText)
                if filtered.isEmpty {
                    StateMessageView(
                        icon: isFiltering ? "magnifyingglass" : "tray",
                        title: isFiltering ? "未找到匹配术语" : "暂无术语数据"
                    )
                } else {
                    ScrollView {
                        LazyVStack(spacing: 12) {
                            ForEach(filtered) { item in
                                termCard(item)
                            }
                        }
                        .padding(.horizontal, 16)
                        .padding(.bottom, 20)
                    }
                }
            }
        }
        .background(palette.page.ignoresSafeArea())
        .navigationTitle("术语词典")
        .navigationBarTitleDisplayMode(.inline)
    }

    private var searchBar: some View {
        HStack(spacing: 8) {
            Image(systemName: "magnifyingglass")
                .font(.system(size: 16))
                .foregroundStyle(palette.textMuted)
                .accessibilityHidden(true)
            TextField("搜索术语...", text: $filterText)
                .font(.system(size: 14))
                .foregroundStyle(palette.textPrimary)
                .autocorrectionDisabled()
                .textInputAutocapitalization(.never)
        }
        .padding(.horizontal, 12)
        .padding(.vertical, 10)
        .background(palette.card, in: RoundedRectangle(cornerRadius: 8))
        .overlay {
            RoundedRectangle(cornerRadius: 8).stroke(palette.border)
        }
    }

    private func termCard(_ item: GlossaryItem) -> some View {
        VStack(alignment: .leading, spacing: 6) {
            Text(item.title)
                .font(.system(size: 15, weight: .semibold))
                .foregroundStyle(palette.textPrimary)
            Text(item.content)
                .font(.system(size: 13))
                .foregroundStyle(palette.textSecondary)
                .lineSpacing(5)
            if !item.category.isEmpty {
                Text(RuleCategory.label(for: item.category))
                    .font(.system(size: 11))
                    .foregroundStyle(palette.brandStrong)
                    .padding(.horizontal, 8)
                    .padding(.vertical, 2)
                    .background(palette.brandTint, in: RoundedRectangle(cornerRadius: 6))
            }
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .appCard()
    }
}
