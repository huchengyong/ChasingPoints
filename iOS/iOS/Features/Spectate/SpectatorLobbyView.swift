import SwiftUI

/// 观赛大厅：状态标签、球种筛选与分页列表；访客可看公开大厅。
struct SpectatorLobbyView: View {
    @EnvironmentObject private var session: SessionStore
    @EnvironmentObject private var toast: ToastCenter
    @Environment(\.colorScheme) private var colorScheme
    @Environment(\.dismiss) private var dismiss
    @StateObject private var model: SpectatorLobbyModel

    init(session: SessionStore, model: SpectatorLobbyModel? = nil) {
        _model = StateObject(wrappedValue: model ?? SpectatorLobbyModel(session: session))
    }

    private var palette: AppPalette { AppTheme.palette(for: colorScheme) }

    var body: some View {
        VStack(spacing: 0) {
            filterBar
            content
        }
        .background(palette.page.ignoresSafeArea())
        .navigationTitle("观赛大厅")
        .navigationBarTitleDisplayMode(.inline)
        .task { await model.loadIfNeeded() }
        .onChange(of: session.identity) { _, _ in
            // 登录、退出或切号后按新身份重读，「我」的标记随当前用户变化。
            model.syncIdentity()
            Task { await model.loadIfNeeded() }
        }
        .onChange(of: model.notice) { _, notice in
            guard let notice else { return }
            toast.show(notice)
            model.clearNotice()
        }
    }

    private var filterBar: some View {
        HStack(spacing: 8) {
            HStack(spacing: 6) {
                ForEach(model.tabs) { tab in
                    tabButton(tab)
                }
            }

            Spacer(minLength: 8)

            Menu {
                ForEach(SpectatorLobbyRules.gameTypeOptions) { option in
                    Button(option.label) {
                        Task { await model.selectGameType(option.value) }
                    }
                }
            } label: {
                HStack(spacing: 3) {
                    Text(model.selectedGameTypeLabel)
                        .font(.system(size: 12, weight: model.gameType == 0 ? .medium : .semibold))
                    Image(systemName: "chevron.down")
                        .font(.system(size: 10, weight: .semibold))
                }
                .foregroundStyle(model.gameType == 0 ? palette.textSecondary : palette.brandStrong)
                .frame(height: 30)
            }
            .accessibilityLabel("球种筛选，当前\(model.selectedGameTypeLabel)")
        }
        .padding(.horizontal, 12)
        .padding(.vertical, 8)
        .background(palette.page)
    }

    private func tabButton(_ tab: SpectatorLobbyTab) -> some View {
        let isActive = model.tab == tab
        return Button {
            Task { await model.selectTab(tab) }
        } label: {
            Text(tab.label)
                .font(.system(size: 13, weight: isActive ? .semibold : .medium))
                .foregroundStyle(isActive ? .white : palette.textSecondary)
                .padding(.horizontal, 13)
                .frame(height: 30)
                .background(isActive ? AppTheme.primary : palette.card, in: Capsule())
                .overlay {
                    if !isActive {
                        Capsule().stroke(palette.border)
                    }
                }
        }
        .buttonStyle(.plain)
        .accessibilityAddTraits(isActive ? .isSelected : [])
    }

    @ViewBuilder
    private var content: some View {
        switch model.phase {
        case .idle, .loading:
            ProgressView()
                .frame(maxWidth: .infinity, maxHeight: .infinity)
        case .failed(let failure):
            failureState(failure)
        case .empty:
            emptyState
        case .loaded:
            list
        }
    }

    private var list: some View {
        ScrollView {
            LazyVStack(alignment: .leading, spacing: 8) {
                if let failure = model.refreshFailure {
                    refreshBanner(failure)
                }

                ForEach(model.sections, id: \.key) { section in
                    if !section.title.isEmpty {
                        Text(section.title)
                            .font(.system(size: 12, weight: .semibold))
                            .foregroundStyle(palette.textSecondary)
                            .padding(.horizontal, 2)
                            .padding(.top, 8)
                            .padding(.bottom, 2)
                    }

                    ForEach(section.items) { card in
                        cardLink(card)
                    }
                }

                footer
            }
            .padding(.horizontal, 12)
            .padding(.top, 2)
            .padding(.bottom, 20)
        }
        .refreshable { await model.refresh() }
    }

    @ViewBuilder
    private func cardLink(_ card: SpectatorMatchCard) -> some View {
        if let route = destination(for: card) {
            NavigationLink(value: route) {
                matchCard(card)
            }
            .buttonStyle(.plain)
            .task {
                guard card.id == model.cards.last?.id else { return }
                await model.loadMore()
            }
        } else {
            matchCard(card)
        }
    }

    private func matchCard(_ card: SpectatorMatchCard) -> some View {
        VStack(alignment: .leading, spacing: 6) {
            HStack(spacing: 6) {
                Text(card.metaText)
                    .font(.system(size: 11))
                    .foregroundStyle(palette.textMuted)
                    .lineLimit(1)
                Spacer(minLength: 6)
                statePill(card)
            }

            ForEach(Array(card.players.enumerated()), id: \.offset) { _, player in
                playerRow(player)
            }
        }
        .padding(.horizontal, 12)
        .padding(.vertical, 10)
        .background(palette.card, in: RoundedRectangle(cornerRadius: 12))
        .overlay {
            RoundedRectangle(cornerRadius: 12)
                .stroke(card.isMine ? AppTheme.primary.opacity(0.55) : palette.border)
        }
        .contentShape(RoundedRectangle(cornerRadius: 12))
        .accessibilityElement(children: .combine)
        .accessibilityLabel(
            "\(card.metaText)，\(card.statusText)，"
                + card.players.map { "\($0.name)\($0.score)" }.joined(separator: "，")
        )
    }

    private func statePill(_ card: SpectatorMatchCard) -> some View {
        HStack(spacing: 4) {
            if card.isLive {
                Circle()
                    .fill(palette.success)
                    .frame(width: 5, height: 5)
            }
            Text(card.statusText)
                .font(.system(size: 10, weight: .semibold))
        }
        .foregroundStyle(card.isLive ? palette.success : palette.textMuted)
        .padding(.horizontal, 7)
        .frame(height: 20)
        .background(card.isLive ? palette.success.opacity(0.12) : palette.subtle, in: Capsule())
    }

    private func playerRow(_ player: SpectatorPlayer) -> some View {
        let isWinner = player.outcome == .winner
        let isLoser = player.outcome == .loser

        return HStack(spacing: 6) {
            AvatarView(avatar: player.avatar, userId: player.id, size: 28)

            Text(player.name)
                .font(.system(size: 14, weight: isWinner ? .bold : .medium))
                .foregroundStyle(isLoser ? palette.textMuted : palette.textPrimary)
                .lineLimit(1)

            if player.isMe {
                Text("我")
                    .font(.system(size: 9, weight: .bold))
                    .foregroundStyle(palette.brandStrong)
                    .padding(.horizontal, 5)
                    .frame(height: 15)
                    .background(palette.brandTint, in: RoundedRectangle(cornerRadius: 4))
            }

            Spacer(minLength: 6)

            Text("\(player.score)")
                .font(.system(size: 17, weight: .bold).monospacedDigit())
                .foregroundStyle(isLoser ? palette.textMuted : (isWinner ? palette.brandStrong : palette.textPrimary))
        }
        .frame(minHeight: 32)
    }

    @ViewBuilder
    private var footer: some View {
        if model.hasMore {
            Text(model.isLoadingMore ? "正在加载..." : "上拉加载更多")
                .font(.system(size: 11))
                .foregroundStyle(palette.textMuted)
                .frame(maxWidth: .infinity)
                .padding(.top, 18)
        } else if model.total > SpectatorLobbyRules.pageSize {
            HStack(spacing: 8) {
                line
                Text("没有更多对局了")
                    .font(.system(size: 11))
                    .foregroundStyle(palette.textMuted)
                line
            }
            .frame(maxWidth: .infinity)
            .padding(.top, 18)
        }
    }

    private var line: some View {
        Rectangle()
            .fill(palette.border)
            .frame(width: 24, height: 1)
    }

    private var emptyState: some View {
        let state = model.emptyState
        let actionTitle = model.gameType != 0
            ? "查看全部球种"
            : (model.tab == .friends ? "看看大厅对局" : nil)

        return StateMessageView(
            icon: "eye",
            title: state.title,
            detail: state.subtitle,
            actionTitle: actionTitle
        ) {
            Task {
                if model.gameType != 0 {
                    await model.selectGameType(0)
                } else if model.tab == .friends {
                    await model.selectTab(.live)
                }
            }
        }
    }

    private func failureState(_ failure: LoadFailure) -> some View {
        let feedback = failure.feedback(resource: "对局大厅")
        return StateMessageView(
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
    }

    private func refreshBanner(_ failure: LoadFailure) -> some View {
        let feedback = failure.feedback(resource: "对局大厅")
        return HStack(spacing: 8) {
            Text(feedback.description)
                .font(.system(size: 12))
                .foregroundStyle(palette.textSecondary)
            Spacer(minLength: 8)
            Button("重试") {
                Task { await model.refresh() }
            }
            .font(.system(size: 12, weight: .semibold))
            .foregroundStyle(palette.brandStrong)
        }
        .padding(.horizontal, 10)
        .padding(.vertical, 9)
        .background(palette.card, in: RoundedRectangle(cornerRadius: 9))
        .overlay {
            RoundedRectangle(cornerRadius: 9).stroke(palette.border)
        }
    }

    private func destination(for card: SpectatorMatchCard) -> AppRoute? {
        let identity = session.identity
        switch SpectatorLobbyRules.destination(
            for: card,
            userId: identity?.userId ?? 0,
            isLoggedIn: identity != nil
        ) {
        case .report: return .matchReport(matchId: card.id)
        case .publicDetail: return .publicMatchDetail(matchId: card.id)
        case .live: return .liveMatch(matchId: card.id)
        case nil: return nil
        }
    }
}
