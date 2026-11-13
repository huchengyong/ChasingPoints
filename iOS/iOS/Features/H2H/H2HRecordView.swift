import SwiftUI

/// 交锋记录，对应小程序 `subPages/user/h2hRecord.vue`。
struct H2HRecordView: View {
    @EnvironmentObject private var session: SessionStore
    @EnvironmentObject private var navigator: AppNavigator
    @EnvironmentObject private var toast: ToastCenter
    @Environment(\.colorScheme) private var colorScheme
    @Environment(\.dismiss) private var dismiss
    @StateObject private var model: H2HRecordModel

    init(
        session: SessionStore,
        targetUserId: Int64 = 0,
        targetName: String = "",
        targetAvatar: String = "",
        opponentId: Int64 = 0,
        opponentName: String = "",
        model: H2HRecordModel? = nil
    ) {
        _model = StateObject(wrappedValue: model ?? H2HRecordModel(
            session: session,
            targetUserId: targetUserId,
            targetName: targetName,
            targetAvatar: targetAvatar,
            opponentId: opponentId,
            opponentName: opponentName
        ))
    }

    private var palette: AppPalette { AppTheme.palette(for: colorScheme) }

    var body: some View {
        Group {
            if session.identity == nil {
                StateMessageView(
                    icon: "lock.fill",
                    title: "请登录后查看交锋记录",
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
        .safeAreaInset(edge: .bottom) {
            if session.identity != nil, model.showSummaryCard {
                actionBar
            }
        }
        .task {
            await model.onAppear()
            handleLoadFailure()
        }
        .onChange(of: session.identity) { _, _ in
            Task {
                await model.onIdentityChanged()
                handleLoadFailure()
            }
        }
    }

    private var content: some View {
        ScrollView {
            VStack(spacing: 12) {
                if model.showSummaryCard {
                    summaryCard
                }
                stateContent
            }
            .padding(.horizontal, 12)
            .padding(.vertical, 12)
        }
        .refreshable { await model.reload() }
    }

    @ViewBuilder
    private var stateContent: some View {
        if model.isHidden {
            StateMessageView(
                icon: "lock.fill",
                title: H2HRules.hiddenTitle,
                detail: H2HRules.hiddenDescription
            )
            .frame(maxHeight: 360)
        } else {
            switch model.pageStatus {
            case .loading:
                VStack(spacing: 10) {
                    ProgressView()
                    Text("加载中...")
                        .font(.system(size: 14))
                        .foregroundStyle(palette.textSecondary)
                }
                .frame(maxWidth: .infinity)
                .padding(.vertical, 60)
            case .error:
                errorState
            case .partial, .ready, .empty:
                historySection
            }
        }
    }

    private var errorState: some View {
        let feedback = model.failureFeedback ?? LoadFailure(category: .business, message: "")
            .feedback(resource: "交锋数据")

        return StateMessageView(
            icon: "exclamationmark.triangle",
            title: feedback.title,
            detail: feedback.description,
            actionTitle: feedback.actionText
        ) {
            if feedback.retryable {
                Task {
                    await model.retry()
                    handleLoadFailure()
                }
            } else {
                dismiss()
            }
        }
        .frame(maxHeight: 360)
    }

    // MARK: - 概览

    private var summaryCard: some View {
        let hero = model.hero

        return VStack(alignment: .leading, spacing: 14) {
            HStack(alignment: .top, spacing: 8) {
                avatarSection(
                    name: hero.leftName,
                    avatar: hero.leftAvatar,
                    userId: model.subjectUserId,
                    showsMeBadge: false
                )

                Text("VS")
                    .font(.system(size: 13, weight: .bold))
                    .foregroundStyle(palette.textMuted)
                    .padding(.top, 22)

                avatarSection(
                    name: hero.rightName,
                    avatar: hero.rightAvatar,
                    userId: model.resolvedOpponentId,
                    showsMeBadge: model.viewModel.isOpponentMe
                )
            }
            .frame(maxWidth: .infinity)

            VStack(spacing: 6) {
                Text(hero.title)
                    .font(.system(size: 17, weight: .semibold))
                    .foregroundStyle(palette.textPrimary)
                    .multilineTextAlignment(.center)
                Text(hero.subtitle)
                    .font(.system(size: 13))
                    .foregroundStyle(palette.textSecondary)
                    .multilineTextAlignment(.center)
            }
            .frame(maxWidth: .infinity)

            Text(hero.scoreText)
                .font(.system(size: 32, weight: .heavy).monospacedDigit())
                .foregroundStyle(palette.textPrimary)
                .frame(maxWidth: .infinity)

            HStack(spacing: 8) {
                ForEach(hero.badges, id: \.self) { badge in
                    Text(badge)
                        .font(.system(size: 12))
                        .foregroundStyle(palette.textSecondary)
                        .padding(.horizontal, 10)
                        .padding(.vertical, 4)
                        .background(palette.subtle, in: Capsule())
                }
            }
            .frame(maxWidth: .infinity)

            HStack(spacing: 0) {
                statItem(label: "总场次", value: String(model.stats?.totalMatches ?? 0))
                statItem(label: "平均分差", value: avgDiffText)
                statItem(label: "最长连胜", value: String(model.stats?.maxWinStreak ?? 0))
            }
        }
        .frame(maxWidth: .infinity)
        .appCard()
    }

    private var avgDiffText: String {
        let diff = model.stats?.avgScoreDiff ?? 0
        return "\(diff > 0 ? "+" : "")\(String(format: "%.1f", diff))"
    }

    private func avatarSection(
        name: String,
        avatar: String,
        userId: Int64,
        showsMeBadge: Bool
    ) -> some View {
        VStack(spacing: 6) {
            AvatarView(avatar: avatar, userId: userId, size: 56)
            HStack(spacing: 4) {
                Text(name)
                    .font(.system(size: 13, weight: .semibold))
                    .foregroundStyle(palette.textPrimary)
                    .lineLimit(1)
                if showsMeBadge {
                    Text("我")
                        .font(.system(size: 10, weight: .semibold))
                        .foregroundStyle(palette.brandStrong)
                        .padding(.horizontal, 6)
                        .padding(.vertical, 1)
                        .background(palette.brandTint, in: Capsule())
                }
            }
        }
        .frame(maxWidth: .infinity)
    }

    private func statItem(label: String, value: String) -> some View {
        VStack(spacing: 4) {
            Text(label)
                .font(.system(size: 12))
                .foregroundStyle(palette.textSecondary)
            Text(value)
                .font(.system(size: 18, weight: .bold).monospacedDigit())
                .foregroundStyle(palette.textPrimary)
        }
        .frame(maxWidth: .infinity)
    }

    private var actionBar: some View {
        Button(model.hero.primaryActionText) {
            navigator.open(model.primaryActionRoute)
        }
        .buttonStyle(.appPrimary)
        .padding(.horizontal, 16)
        .padding(.vertical, 10)
        .background(.ultraThinMaterial)
    }

    // MARK: - 比赛历史

    private var historySection: some View {
        VStack(alignment: .leading, spacing: 12) {
            VStack(alignment: .leading, spacing: 10) {
                Text("比赛历史")
                    .font(.system(size: 18, weight: .semibold))
                    .foregroundStyle(palette.textPrimary)

                HStack(spacing: 8) {
                    monthButton(systemName: "chevron.left", label: "上个月", offset: -1)
                    Text(model.calendar.title)
                        .font(.system(size: 15, weight: .semibold))
                        .foregroundStyle(palette.textPrimary)
                        .frame(minWidth: 84)
                    monthButton(systemName: "chevron.right", label: "下个月", offset: 1)
                    Spacer(minLength: 8)
                    viewModeSwitch
                }
            }

            if let message = model.partialMessage {
                HStack(alignment: .firstTextBaseline, spacing: 8) {
                    Text(message)
                        .font(.system(size: 13))
                        .foregroundStyle(palette.textSecondary)
                    Spacer(minLength: 8)
                    Button("重新加载") {
                        Task {
                            await model.reload()
                            handleLoadFailure()
                        }
                    }
                    .font(.system(size: 13, weight: .semibold))
                    .foregroundStyle(palette.brandStrong)
                }
                .padding(12)
                .background(AppTheme.warning.opacity(0.12), in: RoundedRectangle(cornerRadius: 10))
            }

            if model.historyViewMode == .calendar {
                calendarCard
            } else {
                listPanel
            }
        }
    }

    private func monthButton(systemName: String, label: String, offset: Int) -> some View {
        Button {
            Task {
                await model.shiftMonth(by: offset)
                handleLoadFailure()
            }
        } label: {
            Image(systemName: systemName)
                .font(.system(size: 13, weight: .semibold))
                .foregroundStyle(palette.textSecondary)
                .frame(width: 30, height: 30)
                .background(palette.subtle, in: Circle())
        }
        .buttonStyle(.plain)
        .accessibilityLabel(label)
    }

    private var viewModeSwitch: some View {
        HStack(spacing: 0) {
            viewModeButton("月历", mode: .calendar)
            viewModeButton("列表", mode: .list)
        }
        .padding(3)
        .background(palette.subtle, in: Capsule())
    }

    private func viewModeButton(_ title: String, mode: H2HHistoryViewMode) -> some View {
        Button {
            model.historyViewMode = mode
        } label: {
            Text(title)
                .font(.system(size: 13, weight: .semibold))
                .foregroundStyle(model.historyViewMode == mode ? palette.textPrimary : palette.textSecondary)
                .padding(.horizontal, 12)
                .padding(.vertical, 5)
                .background(
                    model.historyViewMode == mode ? palette.card : Color.clear,
                    in: Capsule()
                )
        }
        .buttonStyle(.plain)
        .accessibilityAddTraits(model.historyViewMode == mode ? .isSelected : [])
    }

    private var calendarCard: some View {
        VStack(spacing: 10) {
            HStack(spacing: 0) {
                ForEach(H2HCalendar.weekdays, id: \.self) { weekday in
                    Text(weekday)
                        .font(.system(size: 12))
                        .foregroundStyle(palette.textSecondary)
                        .frame(maxWidth: .infinity)
                }
            }

            LazyVGrid(columns: Array(repeating: GridItem(.flexible(), spacing: 4), count: 7), spacing: 6) {
                ForEach(model.calendar.cells, id: \.dateKey) { cell in
                    calendarCell(cell)
                }
            }

            HStack(spacing: 20) {
                legendItem(color: palette.success, text: "胜")
                legendItem(color: AppTheme.danger, text: "负")
            }
            .frame(maxWidth: .infinity)
        }
        .appCard()
    }

    private func calendarCell(_ cell: H2HCalendarCell) -> some View {
        VStack(spacing: 2) {
            Text(String(cell.day))
                .font(.system(size: 12))
                .foregroundStyle(cell.isCurrentMonth ? palette.textPrimary : palette.textMuted)

            if cell.isCurrentMonth && cell.hasMatches {
                if cell.winCount > 0 {
                    resultLine(color: palette.success, count: cell.winCount)
                }
                if cell.lossCount > 0 {
                    resultLine(color: AppTheme.danger, count: cell.lossCount)
                }
            }
        }
        .frame(maxWidth: .infinity, minHeight: 38)
        .background(
            cell.isCurrentMonth && cell.hasMatches ? palette.brandTint : Color.clear,
            in: RoundedRectangle(cornerRadius: 6)
        )
        .accessibilityElement(children: .ignore)
        .accessibilityLabel(cellAccessibilityLabel(cell))
    }

    private func resultLine(color: Color, count: Int) -> some View {
        HStack(spacing: 2) {
            Circle()
                .fill(color)
                .frame(width: 5, height: 5)
            Text("×\(String(count))")
                .font(.system(size: 9))
                .foregroundStyle(palette.textSecondary)
        }
    }

    private func legendItem(color: Color, text: String) -> some View {
        HStack(spacing: 6) {
            Circle()
                .fill(color)
                .frame(width: 7, height: 7)
            Text(text)
                .font(.system(size: 12))
                .foregroundStyle(palette.textSecondary)
        }
    }

    private func cellAccessibilityLabel(_ cell: H2HCalendarCell) -> String {
        guard cell.isCurrentMonth, cell.hasMatches else { return "\(cell.day)日，无对局" }
        return "\(cell.day)日，\(String(cell.winCount))胜\(String(cell.lossCount))负"
    }

    private var listPanel: some View {
        VStack(spacing: 10) {
            if model.matchCards.isEmpty {
                Text("本月暂无对局记录")
                    .font(.system(size: 14))
                    .foregroundStyle(palette.textSecondary)
                    .frame(maxWidth: .infinity)
                    .padding(.vertical, 28)
                    .appCard()
            } else {
                ForEach(model.matchCards) { card in
                    matchCard(card)
                        .onAppear {
                            if card.id == model.matchCards.last?.id {
                                Task { await model.loadMore() }
                            }
                        }
                }

                footer
            }
        }
    }

    @ViewBuilder
    private func matchCard(_ card: H2HMatchCard) -> some View {
        if card.id != 0 {
            NavigationLink(value: AppRoute.matchReport(matchId: card.id)) {
                H2HMatchCardRow(card: card)
            }
            .buttonStyle(.plain)
        } else {
            H2HMatchCardRow(card: card)
        }
    }

    @ViewBuilder
    private var footer: some View {
        if model.isLoadingMore {
            Text("加载中...")
                .font(.system(size: 13))
                .foregroundStyle(palette.textSecondary)
                .frame(maxWidth: .infinity)
                .padding(.vertical, 8)
        } else if !model.hasMore {
            Text("没有更多了")
                .font(.system(size: 13))
                .foregroundStyle(palette.textSecondary)
                .frame(maxWidth: .infinity)
                .padding(.vertical, 8)
        }
    }

    private func handleLoadFailure() {
        guard let action = model.loadFailureAction else { return }
        toast.show(action.toastMessage)
        dismiss()
    }
}

private struct H2HMatchCardRow: View {
    @Environment(\.colorScheme) private var colorScheme
    let card: H2HMatchCard

    private var palette: AppPalette { AppTheme.palette(for: colorScheme) }

    var body: some View {
        HStack(alignment: .top, spacing: 12) {
            VStack(alignment: .leading, spacing: 6) {
                Text(H2HRules.matchDateText(card.dateText))
                    .font(.system(size: 13))
                    .foregroundStyle(palette.textSecondary)
                Text(card.gameTypeText)
                    .font(.system(size: 15, weight: .semibold))
                    .foregroundStyle(palette.textPrimary)
                HStack(spacing: 6) {
                    ForEach(card.tags, id: \.self) { tag in
                        Text(tag)
                            .font(.system(size: 11))
                            .foregroundStyle(palette.textSecondary)
                            .padding(.horizontal, 8)
                            .padding(.vertical, 2)
                            .background(palette.subtle, in: RoundedRectangle(cornerRadius: 6))
                    }
                }
            }

            Spacer(minLength: 8)

            VStack(alignment: .trailing, spacing: 4) {
                Text(card.scoreText)
                    .font(.system(size: 17, weight: .bold).monospacedDigit())
                    .foregroundStyle(palette.textPrimary)
                HStack(spacing: 6) {
                    Text(card.resultText)
                        .font(.system(size: 12, weight: .semibold))
                        .foregroundStyle(resultColor)
                    Text(card.diffText)
                        .font(.system(size: 12))
                        .foregroundStyle(palette.textSecondary)
                }
            }

            Image(systemName: "chevron.right")
                .font(.system(size: 13, weight: .semibold))
                .foregroundStyle(palette.textMuted)
                .padding(.top, 18)
        }
        .padding(14)
        .frame(maxWidth: .infinity, alignment: .leading)
        .background(palette.card, in: RoundedRectangle(cornerRadius: 16))
        .overlay {
            RoundedRectangle(cornerRadius: 16).stroke(palette.border, lineWidth: 1)
        }
        .accessibilityElement(children: .combine)
    }

    private var resultColor: Color {
        switch card.result {
        case 1: palette.success
        case 2: AppTheme.danger
        default: palette.textSecondary
        }
    }
}
