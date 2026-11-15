import SwiftUI

/// 过往对手，对应小程序 `subPages/user/opponentRecord.vue`。
struct OpponentRecordView: View {
    @EnvironmentObject private var session: SessionStore
    @EnvironmentObject private var navigator: AppNavigator
    @Environment(\.colorScheme) private var colorScheme
    @Environment(\.dismiss) private var dismiss
    @StateObject private var model: OpponentRecordModel

    init(session: SessionStore, target: OpponentRecordTarget? = nil, model: OpponentRecordModel? = nil) {
        _model = StateObject(wrappedValue: model ?? OpponentRecordModel(
            session: session,
            targetUserId: target?.userId ?? 0,
            targetName: target?.name ?? "",
            targetAvatar: target?.avatar ?? ""
        ))
    }

    private var palette: AppPalette { AppTheme.palette(for: colorScheme) }

    var body: some View {
        Group {
            if session.identity == nil {
                StateMessageView(
                    icon: "lock.fill",
                    title: model.viewModel.loginHint,
                    actionTitle: "去登录"
                ) {
                    navigator.openLogin()
                }
            } else {
                content
            }
        }
        .background(palette.page.ignoresSafeArea())
        .navigationTitle(model.viewModel.navigationTitle)
        .navigationBarTitleDisplayMode(.inline)
        .task { await model.onAppear() }
        .onChange(of: session.identity) { _, _ in
            Task { await model.onIdentityChanged() }
        }
    }

    private var content: some View {
        ScrollView {
            VStack(spacing: 12) {
                searchBox
                statsCard
                listSection
            }
            .padding(.horizontal, 12)
            .padding(.vertical, 12)
        }
        .refreshable { await model.list.refresh() }
    }

    private var searchBox: some View {
        HStack(spacing: 8) {
            Image(systemName: "magnifyingglass")
                .font(.system(size: 15, weight: .semibold))
                .foregroundStyle(palette.textMuted)
                .accessibilityHidden(true)
            TextField(model.viewModel.searchPlaceholder, text: $model.keyword)
                .font(.system(size: 15))
                .foregroundStyle(palette.textPrimary)
                .submitLabel(.search)
                .textInputAutocapitalization(.never)
                .autocorrectionDisabled()
                .onSubmit { Task { await model.search() } }
        }
        .padding(.horizontal, 14)
        .padding(.vertical, 11)
        .background(palette.card, in: RoundedRectangle(cornerRadius: 12))
        .overlay {
            RoundedRectangle(cornerRadius: 12).stroke(palette.border, lineWidth: 1)
        }
    }

    private var statsCard: some View {
        let summary = model.statsSummary

        return VStack(alignment: .leading, spacing: 14) {
            VStack(alignment: .leading, spacing: 4) {
                Text(summary.title)
                    .font(.system(size: 18, weight: .semibold))
                    .foregroundStyle(palette.textPrimary)
                Text(summary.subtitle)
                    .font(.system(size: 13))
                    .foregroundStyle(palette.textSecondary)
            }

            HStack(spacing: 0) {
                statItem(label: "总对手数", value: model.totalOpponents)
                Rectangle()
                    .fill(palette.border)
                    .frame(width: 1, height: 32)
                statItem(label: "总胜场", value: model.totalWins)
            }
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .appCard()
    }

    private func statItem(label: String, value: Int) -> some View {
        VStack(spacing: 4) {
            Text(label)
                .font(.system(size: 12))
                .foregroundStyle(palette.textSecondary)
            Text(String(value))
                .font(.system(size: 20, weight: .bold).monospacedDigit())
                .foregroundStyle(palette.textPrimary)
        }
        .frame(maxWidth: .infinity)
    }

    @ViewBuilder
    private var listSection: some View {
        if model.list.items.isEmpty {
            switch model.list.phase {
            case .failed(let message):
                let feedback = LoadFailure(category: .business, message: message).feedback(resource: "过往对手")
                StateMessageView(
                    icon: "exclamationmark.triangle",
                    title: feedback.title,
                    detail: feedback.description,
                    actionTitle: feedback.actionText
                ) {
                    if feedback.retryable {
                        Task { await model.retry() }
                    } else {
                        dismiss()
                    }
                }
                .frame(maxHeight: 320)
            case .empty:
                StateMessageView(
                    icon: "person.2",
                    title: emptyText,
                    detail: emptyHint
                )
                .frame(maxHeight: 320)
            case .idle, .loading, .loaded:
                VStack(spacing: 10) {
                    ProgressView()
                    Text("加载中...")
                        .font(.system(size: 14))
                        .foregroundStyle(palette.textSecondary)
                }
                .frame(maxWidth: .infinity)
                .padding(.vertical, 28)
            }
        } else {
            VStack(spacing: 10) {
                if let refreshError = model.list.refreshError {
                    HStack(spacing: 8) {
                        Text(refreshError)
                            .font(.system(size: 13))
                            .foregroundStyle(palette.textSecondary)
                        Spacer(minLength: 8)
                        Button("重试") { Task { await model.list.refresh() } }
                            .font(.system(size: 13, weight: .semibold))
                            .foregroundStyle(palette.brandStrong)
                    }
                    .padding(.horizontal, 4)
                }

                ForEach(model.cards) { card in
                    NavigationLink(value: route(for: card)) {
                        OpponentCardRow(card: card)
                    }
                    .buttonStyle(.plain)
                    .onAppear {
                        if card.id == model.list.items.last?.id {
                            Task { await model.list.loadMore() }
                        }
                    }
                }

                footer
            }
        }
    }

    private var emptyText: String {
        isSearching ? "没有找到匹配的对手" : model.viewModel.emptyText
    }

    private var emptyHint: String {
        isSearching ? "换个昵称试试，或者下拉刷新后再看" : model.viewModel.emptyHint
    }

    private var isSearching: Bool {
        !model.keyword.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty
    }

    private func route(for card: OpponentCard) -> AppRoute {
        OpponentRecordRules.h2hRoute(
            opponent: card.opponent,
            targetUserId: model.targetUserId,
            targetName: model.targetName,
            targetAvatar: model.targetAvatar
        )
    }

    @ViewBuilder
    private var footer: some View {
        if model.list.isLoadingMore {
            Text("加载中...")
                .font(.system(size: 13))
                .foregroundStyle(palette.textSecondary)
                .frame(maxWidth: .infinity)
                .padding(.vertical, 8)
        } else if !model.list.hasMore {
            Text("没有更多了")
                .font(.system(size: 13))
                .foregroundStyle(palette.textSecondary)
                .frame(maxWidth: .infinity)
                .padding(.vertical, 8)
        }
    }
}

private struct OpponentCardRow: View {
    @Environment(\.colorScheme) private var colorScheme
    let card: OpponentCard

    private var palette: AppPalette { AppTheme.palette(for: colorScheme) }

    var body: some View {
        HStack(alignment: .top, spacing: 12) {
            AvatarView(
                avatar: card.opponent.avatar,
                userId: card.opponent.id,
                size: 44
            )

            VStack(alignment: .leading, spacing: 6) {
                HStack(spacing: 8) {
                    Text(card.opponent.name)
                        .font(.system(size: 16, weight: .semibold))
                        .foregroundStyle(palette.textPrimary)
                        .lineLimit(1)
                    Text(card.relationshipBadge)
                        .font(.system(size: 11, weight: .semibold))
                        .foregroundStyle(toneColor)
                        .padding(.horizontal, 8)
                        .padding(.vertical, 2)
                        .background(toneColor.opacity(0.14), in: Capsule())
                }

                Text(card.relationshipText)
                    .font(.system(size: 13))
                    .foregroundStyle(palette.textSecondary)

                ViewThatFits(in: .horizontal) {
                    HStack(spacing: 6) {
                        chip(card.recordText)
                        chip(card.lastMatchText)
                    }
                    VStack(alignment: .leading, spacing: 4) {
                        chip(card.recordText)
                        chip(card.lastMatchText)
                    }
                }
            }

            Spacer(minLength: 8)

            VStack(alignment: .trailing, spacing: 6) {
                Text(card.winRateText)
                    .font(.system(size: 15, weight: .bold).monospacedDigit())
                    .foregroundStyle(toneColor)
                Text(card.sampleText)
                    .font(.system(size: 12))
                    .foregroundStyle(palette.textSecondary)
                Text("查看交锋")
                    .font(.system(size: 12, weight: .semibold))
                    .foregroundStyle(palette.brandStrong)
            }
        }
        .padding(14)
        .frame(maxWidth: .infinity, alignment: .leading)
        .background(palette.card, in: RoundedRectangle(cornerRadius: 16))
        .overlay {
            RoundedRectangle(cornerRadius: 16).stroke(palette.border, lineWidth: 1)
        }
        .accessibilityElement(children: .combine)
    }

    private func chip(_ text: String) -> some View {
        Text(text)
            .font(.system(size: 11))
            .foregroundStyle(palette.textSecondary)
            // 胶囊内文字必须单行；放不下时由外层整体换行。
            .lineLimit(1)
            .fixedSize(horizontal: true, vertical: false)
            .padding(.horizontal, 8)
            .padding(.vertical, 3)
            .background(palette.subtle, in: RoundedRectangle(cornerRadius: 6))
    }

    private var toneColor: Color {
        switch card.tone {
        case .lead: palette.success
        case .trail: AppTheme.danger
        case .balanced: palette.brandStrong
        case .waiting: palette.textSecondary
        }
    }
}
