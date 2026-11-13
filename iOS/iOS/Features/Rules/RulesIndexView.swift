import SwiftUI

/// 规则百科首页：球种入口、术语词典与规则搜索。
struct RulesIndexView: View {
    @Environment(\.colorScheme) private var colorScheme
    @Environment(\.dismiss) private var dismiss
    @StateObject private var search: RulesSearchModel
    @State private var expandedResultId: Int64?

    init(session: SessionStore, search: RulesSearchModel? = nil, expandedResultId: Int64? = nil) {
        _search = StateObject(wrappedValue: search ?? RulesSearchModel(api: RulesAPI(session: session)))
        _expandedResultId = State(initialValue: expandedResultId)
    }

    private var palette: AppPalette { AppTheme.palette(for: colorScheme) }

    var body: some View {
        VStack(spacing: 0) {
            searchBar
                .padding(.horizontal, 16)
                .padding(.vertical, 12)

            if search.phase == .idle {
                catalog
            } else {
                searchResults
            }
        }
        .background(palette.page.ignoresSafeArea())
        .navigationTitle("规则百科")
        .navigationBarTitleDisplayMode(.inline)
    }

    private var searchBar: some View {
        HStack(spacing: 8) {
            Image(systemName: "magnifyingglass")
                .font(.system(size: 16))
                .foregroundStyle(palette.textMuted)
                .accessibilityHidden(true)
            TextField("搜索规则、术语...", text: $search.keyword)
                .font(.system(size: 14))
                .foregroundStyle(palette.textPrimary)
                .submitLabel(.search)
                .autocorrectionDisabled()
                .textInputAutocapitalization(.never)
                .onSubmit { Task { await search.search() } }
        }
        .padding(.horizontal, 12)
        .padding(.vertical, 10)
        .background(palette.card, in: RoundedRectangle(cornerRadius: 8))
        .overlay {
            RoundedRectangle(cornerRadius: 8).stroke(palette.border)
        }
    }

    private var catalog: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 12) {
                SectionHeader(title: "选择球种")
                ForEach(RuleCategory.ordered.indices, id: \.self) { index in
                    categoryCard(RuleCategory.ordered[index])
                }

                SectionHeader(title: "更多")
                    .padding(.top, 8)

                NavigationLink(value: AppRoute.glossary) {
                    HStack(spacing: 10) {
                        Text("📚")
                            .font(.system(size: 24))
                        VStack(alignment: .leading, spacing: 3) {
                            Text("术语词典")
                                .font(.system(size: 15, weight: .semibold))
                                .foregroundStyle(palette.textPrimary)
                            Text("台球专业术语大全")
                                .font(.system(size: 12))
                                .foregroundStyle(palette.textMuted)
                        }
                        Spacer(minLength: 8)
                        Image(systemName: "chevron.right")
                            .font(.system(size: 14, weight: .semibold))
                            .foregroundStyle(palette.textMuted)
                    }
                    .appCard()
                }
                .buttonStyle(.plain)
            }
            .padding(.horizontal, 16)
            .padding(.bottom, 20)
        }
    }

    private func categoryCard(_ category: (key: String, name: String, desc: String, emoji: String)) -> some View {
        NavigationLink(value: AppRoute.ruleDetail(category: category.key)) {
            HStack(spacing: 10) {
                Text(category.emoji)
                    .font(.system(size: 24))
                VStack(alignment: .leading, spacing: 3) {
                    Text(category.name)
                        .font(.system(size: 15, weight: .semibold))
                        .foregroundStyle(palette.textPrimary)
                    Text(category.desc)
                        .font(.system(size: 12))
                        .foregroundStyle(palette.textMuted)
                }
                Spacer(minLength: 8)
                Image(systemName: "chevron.right")
                    .font(.system(size: 14, weight: .semibold))
                    .foregroundStyle(palette.textMuted)
            }
            .appCard()
        }
        .buttonStyle(.plain)
    }

    private var searchResults: some View {
        VStack(alignment: .leading, spacing: 12) {
            SectionHeader(title: "搜索结果", moreTitle: "清除") {
                expandedResultId = nil
                search.clear()
            }
            .padding(.horizontal, 16)

            switch search.phase {
            case .idle:
                EmptyView()
            case .loading:
                ProgressView()
                    .frame(maxWidth: .infinity, maxHeight: .infinity)
            case .failed(let failure):
                failureState(failure.feedback(resource: "规则搜索"))
            case .loaded(let items):
                ScrollView {
                    LazyVStack(spacing: 10) {
                        ForEach(items) { item in
                            resultRow(item)
                        }
                    }
                    .padding(.horizontal, 16)
                    .padding(.bottom, 20)
                }
            case .empty:
                Text("未找到相关内容")
                    .font(.system(size: 14))
                    .foregroundStyle(palette.textMuted)
                    .frame(maxWidth: .infinity, maxHeight: .infinity)
            }
        }
    }

    private func resultRow(_ item: RuleContentItem) -> some View {
        let isExpanded = expandedResultId == item.id

        return Button {
            expandedResultId = isExpanded ? nil : item.id
        } label: {
            VStack(alignment: .leading, spacing: 8) {
                HStack(spacing: 8) {
                    Text(item.title)
                        .font(.system(size: 14))
                        .foregroundStyle(palette.textPrimary)
                        .frame(maxWidth: .infinity, alignment: .leading)
                        .multilineTextAlignment(.leading)
                    Image(systemName: isExpanded ? "chevron.up" : "chevron.down")
                        .font(.system(size: 12, weight: .semibold))
                        .foregroundStyle(palette.textMuted)
                }
                if isExpanded {
                    Text(item.content)
                        .font(.system(size: 13))
                        .foregroundStyle(palette.textSecondary)
                        .lineSpacing(6)
                        .frame(maxWidth: .infinity, alignment: .leading)
                        .multilineTextAlignment(.leading)
                }
            }
            .appCard(padding: 14)
        }
        .buttonStyle(.plain)
    }

    private func failureState(_ feedback: LoadFailure.Feedback) -> some View {
        StateMessageView(
            icon: "exclamationmark.triangle",
            title: feedback.title,
            detail: feedback.description,
            actionTitle: feedback.actionText
        ) {
            if feedback.retryable {
                Task { await search.search() }
            } else {
                dismiss()
            }
        }
    }
}
