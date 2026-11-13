import SwiftUI

/// 规则详情：球种名称 + 规则 / 犯规 / 术语三个 Tab。
struct RuleDetailView: View {
    @Environment(\.colorScheme) private var colorScheme
    @StateObject private var model: RuleDetailModel

    init(session: SessionStore, category: String, model: RuleDetailModel? = nil) {
        _model = StateObject(wrappedValue: model ?? RuleDetailModel(session: session, category: category))
    }

    private var palette: AppPalette { AppTheme.palette(for: colorScheme) }

    var body: some View {
        VStack(spacing: 0) {
            header
            tabs
            LoadableContent(
                loader: model.loader(for: model.type),
                requiresLogin: false,
                resource: "规则内容"
            ) { items in
                if items.isEmpty {
                    StateMessageView(icon: "tray", title: "暂无内容")
                } else {
                    ScrollView {
                        LazyVStack(spacing: 10) {
                            ForEach(items.indices, id: \.self) { index in
                                contentCard(items[index], index: index)
                            }
                        }
                        .padding(.horizontal, 16)
                        .padding(.bottom, 20)
                    }
                }
            }
        }
        .background(palette.page.ignoresSafeArea())
        .navigationTitle(navigationTitle)
        .navigationBarTitleDisplayMode(.inline)
    }

    private var navigationTitle: String {
        "\(RuleCategory.label(for: model.category, fallback: "规则")) - 规则"
    }

    private var header: some View {
        HStack(spacing: 8) {
            Text(RuleCategory.emoji(for: model.category))
                .font(.system(size: 24))
            Text(RuleCategory.label(for: model.category, fallback: "规则"))
                .font(.system(size: 18, weight: .bold))
                .foregroundStyle(palette.textPrimary)
            Spacer(minLength: 0)
        }
        .padding(.horizontal, 16)
        .padding(.top, 16)
    }

    private var tabs: some View {
        HStack(spacing: 8) {
            ForEach(RuleContentType.allCases, id: \.self) { type in
                let isActive = model.type == type
                Button {
                    Task { await model.select(type) }
                } label: {
                    Text(type.label)
                        .font(.system(size: 14))
                        .foregroundStyle(isActive ? palette.brandStrong : palette.textSecondary)
                        .frame(maxWidth: .infinity)
                        .padding(.vertical, 8)
                        .background(
                            isActive ? palette.brandTint : palette.card,
                            in: RoundedRectangle(cornerRadius: 8)
                        )
                }
                .buttonStyle(.plain)
            }
        }
        .padding(.horizontal, 16)
        .padding(.vertical, 12)
    }

    private func contentCard(_ item: RuleContentItem, index: Int) -> some View {
        let isExpanded = model.expandedId == item.id

        return VStack(alignment: .leading, spacing: 0) {
            Button {
                model.toggle(item.id)
            } label: {
                HStack(spacing: 10) {
                    Text("\(String(index + 1))")
                        .font(.system(size: 12, weight: .semibold))
                        .foregroundStyle(palette.brandStrong)
                        .frame(width: 24, height: 24)
                        .background(palette.brandTint, in: Circle())
                    Text(item.title)
                        .font(.system(size: 14, weight: .medium))
                        .foregroundStyle(palette.textPrimary)
                        .frame(maxWidth: .infinity, alignment: .leading)
                        .multilineTextAlignment(.leading)
                    Image(systemName: isExpanded ? "chevron.up" : "chevron.down")
                        .font(.system(size: 12, weight: .semibold))
                        .foregroundStyle(palette.textMuted)
                }
            }
            .buttonStyle(.plain)

            if isExpanded {
                Text(item.content)
                    .font(.system(size: 13))
                    .foregroundStyle(palette.textSecondary)
                    .lineSpacing(6)
                    .frame(maxWidth: .infinity, alignment: .leading)
                    .padding(.top, 12)
            }
        }
        .appCard()
    }
}
