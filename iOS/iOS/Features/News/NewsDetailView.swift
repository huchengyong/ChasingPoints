import SwiftUI

/// 赛事详情：头部、冠军、赛程分栏与按天分组的对阵，逐条对应 `subPages/tournament/detail.vue`。
struct NewsDetailView: View {
    @EnvironmentObject private var session: SessionStore
    @Environment(\.colorScheme) private var colorScheme
    @Environment(\.dismiss) private var dismiss
    @StateObject private var model: SaiXunDetailModel

    init(session: SessionStore, eventId: Int64, model: SaiXunDetailModel? = nil) {
        _model = StateObject(wrappedValue: model ?? SaiXunDetailModel(session: session, eventId: eventId))
    }

    private var palette: AppPalette { AppTheme.palette(for: colorScheme) }

    var body: some View {
        ScrollView {
            LazyVStack(spacing: 0) {
                switch model.phase {
                case .idle, .loading:
                    skeleton
                case .failed(let failure):
                    failureState(failure)
                case .loaded:
                    if let detail = model.detail {
                        detailContent(detail)
                    } else {
                        skeleton
                    }
                }
            }
            .padding(.horizontal, 12)
            .padding(.top, 12)
            .padding(.bottom, 24)
        }
        .background(palette.page.ignoresSafeArea())
        .refreshable { await model.refresh() }
        .navigationTitle("赛事详情")
        .navigationBarTitleDisplayMode(.inline)
        .task {
            await model.loadIfNeeded()
            model.refreshDisplay()
            // 页面离开时任务取消，定时刷新随之停止。
            await model.runAutoRefresh()
        }
        .onChange(of: session.identity) { _, _ in
            Task { await model.reloadForIdentityChange() }
        }
    }

    // MARK: - 内容

    private func detailContent(_ detail: SaiXunDetail) -> some View {
        VStack(spacing: 0) {
            if let refreshError = model.refreshError {
                refreshBanner(refreshError)
                    .padding(.bottom, 10)
            }

            headerCard(detail)

            if let champion = detail.champion {
                championCard(champion)
                    .padding(.top, 10)
            }

            scheduleSection(detail)

            if !detail.card.sourceText.isEmpty {
                Text(verbatim: "数据来源：\(detail.card.sourceText)")
                    .font(.system(size: 10))
                    .foregroundStyle(palette.textMuted)
                    .frame(maxWidth: .infinity)
                    .padding(.top, 16)
            }
        }
    }

    private func headerCard(_ detail: SaiXunDetail) -> some View {
        let card = detail.card

        return VStack(alignment: .leading, spacing: 0) {
            RemoteImage(url: coverURL(card.coverImage)) {
                palette.subtle
            }
            .frame(height: 160)
            .clipped()

            VStack(alignment: .leading, spacing: 0) {
                HStack(spacing: 6) {
                    SaiXunStatusBadge(
                        tone: card.statusTone,
                        label: card.statusLabel,
                        showDot: card.statusTone == .live
                    )
                    tag(card.gameTypeText)
                    if !card.sourceText.isEmpty {
                        tag(card.sourceText)
                    }
                    Spacer(minLength: 0)
                }

                Text(card.title)
                    .font(.system(size: 20, weight: .bold))
                    .foregroundStyle(palette.textPrimary)
                    .padding(.top, 9)

                if !card.originalTitle.isEmpty {
                    Text(card.originalTitle)
                        .font(.system(size: 11))
                        .foregroundStyle(palette.textMuted)
                        .padding(.top, 3)
                }

                VStack(alignment: .leading, spacing: 5) {
                    factRow(icon: "calendar", text: detail.showTime ? detail.timeText : detail.dateText)
                    if !detail.regionText.isEmpty {
                        factRow(icon: "location", text: detail.regionText)
                    }
                }
                .padding(.top, 10)
            }
            .padding(.horizontal, 14)
            .padding(.top, 12)
            .padding(.bottom, 14)
        }
        .background(palette.card)
        .clipShape(RoundedRectangle(cornerRadius: 12))
        .overlay {
            RoundedRectangle(cornerRadius: 12).stroke(palette.border)
        }
    }

    private func tag(_ text: String) -> some View {
        Text(text)
            .font(.system(size: 10))
            .foregroundStyle(palette.textSecondary)
            .padding(.horizontal, 7)
            .frame(height: 20)
            .background(palette.subtle, in: Capsule())
    }

    private func factRow(icon: String, text: String) -> some View {
        HStack(spacing: 5) {
            Image(systemName: icon)
                .font(.system(size: 13))
                .foregroundStyle(palette.textMuted)
                .frame(width: 16)
            Text(text)
                .font(.system(size: 12))
                .foregroundStyle(palette.textSecondary)
        }
    }

    private func championCard(_ champion: SaiXunChampion) -> some View {
        let isDark = colorScheme == .dark
        let nameColor: Color = isDark ? palette.textPrimary : .white
        let secondaryColor: Color = isDark ? palette.textSecondary : .white.opacity(0.8)

        return HStack(spacing: 12) {
            SaiXunPlayerAvatar(source: champion.avatar, size: 60)
                .overlay {
                    Circle().stroke(AppTheme.primary.opacity(0.6), lineWidth: 1)
                }

            VStack(alignment: .leading, spacing: 3) {
                HStack(spacing: 3) {
                    Image(systemName: "medal.fill")
                        .font(.system(size: 14))
                        .foregroundStyle(Color(hex: 0xF0C542))
                    Text("冠军")
                        .font(.system(size: 11, weight: .semibold))
                        .foregroundStyle(Color(hex: 0xF0C542))
                }

                HStack(spacing: 5) {
                    SaiXunFlagView(flag: champion.flag)
                    Text(champion.name)
                        .font(.system(size: 18, weight: .heavy))
                        .foregroundStyle(nameColor)
                        .lineLimit(1)
                }

                if !champion.subName.isEmpty {
                    Text(champion.subName)
                        .font(.system(size: 11))
                        .foregroundStyle(secondaryColor)
                        .lineLimit(1)
                }

                Text(verbatim: "决赛 \(champion.scoreText) 胜 \(champion.opponentName)")
                    .font(.system(size: 11))
                    .foregroundStyle(secondaryColor)
                    .lineLimit(1)
            }

            Spacer(minLength: 0)
        }
        .padding(.horizontal, 14)
        .padding(.vertical, 12)
        .background(
            LinearGradient(
                colors: palette.heroColors,
                startPoint: .topLeading,
                endPoint: .bottomTrailing
            ),
            in: RoundedRectangle(cornerRadius: 12)
        )
    }

    // MARK: - 赛程

    private func scheduleSection(_ detail: SaiXunDetail) -> some View {
        VStack(alignment: .leading, spacing: 0) {
            HStack(alignment: .firstTextBaseline) {
                Text("赛程赛果")
                    .font(.system(size: 16, weight: .bold))
                    .foregroundStyle(palette.textPrimary)
                Spacer(minLength: 8)
                Text(detail.card.matchCountText)
                    .font(.system(size: 11))
                    .foregroundStyle(palette.textMuted)
            }
            .padding(.horizontal, 2)
            .padding(.top, 16)

            if detail.rounds.isEmpty {
                VStack(spacing: 5) {
                    Text("赛程待更新")
                        .font(.system(size: 14, weight: .semibold))
                        .foregroundStyle(palette.textPrimary)
                    Text("当前赛事还没有录入逐场赛程与赛果。")
                        .font(.system(size: 12))
                        .foregroundStyle(palette.textSecondary)
                }
                .frame(maxWidth: .infinity)
                .padding(.vertical, 20)
                .padding(.horizontal, 16)
                .background(palette.card, in: RoundedRectangle(cornerRadius: 12))
                .overlay {
                    RoundedRectangle(cornerRadius: 12).stroke(palette.border)
                }
                .padding(.top, 10)
            } else {
                scheduleTabs(detail.scheduleTabs)
                    .padding(.top, 6)

                let active = model.activeScheduleTab
                if active.matches.isEmpty {
                    Text(active.key == "ended" ? "还没有已结束的比赛" : "本赛事的比赛已全部结束")
                        .font(.system(size: 12))
                        .foregroundStyle(palette.textSecondary)
                        .frame(maxWidth: .infinity)
                        .padding(.vertical, 20)
                        .background(palette.card, in: RoundedRectangle(cornerRadius: 12))
                        .overlay {
                            RoundedRectangle(cornerRadius: 12).stroke(palette.border)
                        }
                        .padding(.top, 10)
                } else {
                    ForEach(model.visibleDays, id: \.key) { day in
                        VStack(alignment: .leading, spacing: 6) {
                            Text(day.label)
                                .font(.system(size: 12, weight: .semibold))
                                .foregroundStyle(palette.textSecondary)
                                .padding(.horizontal, 2)

                            ForEach(day.matches) { match in
                                matchCard(match)
                                    .task {
                                        guard match.id == active.matches.prefix(model.scheduleRenderLimit).last?.id else {
                                            return
                                        }
                                        model.showMoreScheduleMatches()
                                    }
                            }
                        }
                        .padding(.top, 12)
                    }

                    if model.hasMoreScheduleMatches {
                        Text("上拉加载更多")
                            .font(.system(size: 11))
                            .foregroundStyle(palette.textMuted)
                            .frame(maxWidth: .infinity)
                            .padding(.top, 12)
                            .padding(.bottom, 4)
                    }
                }
            }
        }
    }

    private func scheduleTabs(_ tabs: [SaiXunScheduleTab]) -> some View {
        let activeKey = model.activeScheduleTab.key

        return HStack(spacing: 0) {
            ForEach(tabs, id: \.key) { tab in
                let isActive = tab.key == activeKey
                Button {
                    model.selectScheduleTab(tab.key)
                } label: {
                    HStack(spacing: 4) {
                        Text(tab.label)
                            .font(.system(size: 13, weight: isActive ? .semibold : .medium))
                        Text(verbatim: String(tab.matches.count))
                            .font(.system(size: 11, weight: isActive ? .semibold : .regular))
                    }
                    .foregroundStyle(isActive ? .white : palette.textSecondary)
                    .frame(maxWidth: .infinity)
                    .frame(height: 30)
                    .background {
                        if isActive {
                            Capsule().fill(AppTheme.primary)
                        }
                    }
                }
                .buttonStyle(.plain)
                .accessibilityAddTraits(isActive ? .isSelected : [])
            }
        }
        .padding(3)
        .background(palette.card, in: Capsule())
        .overlay {
            Capsule().stroke(palette.border)
        }
    }

    private func matchCard(_ match: SaiXunMatch) -> some View {
        VStack(alignment: .leading, spacing: 5) {
            HStack(spacing: 6) {
                Text(match.clockText)
                    .font(.system(size: 11, weight: .semibold))
                    .foregroundStyle(palette.textSecondary)
                Text(match.roundName)
                    .font(.system(size: 10, weight: .semibold))
                    .foregroundStyle(palette.textPrimary)
                    .lineLimit(1)
                    .padding(.horizontal, 6)
                    .frame(height: 18)
                    .background(palette.brandTint, in: RoundedRectangle(cornerRadius: 4))
                if !match.metaText.isEmpty {
                    Text(match.metaText)
                        .font(.system(size: 10))
                        .foregroundStyle(palette.textMuted)
                }
                Spacer(minLength: 4)
                if match.isLive {
                    SaiXunStatusBadge(tone: .live, label: "进行中", showDot: true)
                } else if match.status == 3 {
                    SaiXunStatusBadge(tone: .canceled, label: "已取消")
                }
            }

            playerRow(
                name: match.homePlayerName,
                subName: match.homePlayerSubName,
                flag: match.homePlayerFlag,
                avatar: match.homePlayerAvatar,
                score: match.homeScore,
                showScore: match.showScore,
                state: playerState(match, side: 1)
            )
            playerRow(
                name: match.awayPlayerName,
                subName: match.awayPlayerSubName,
                flag: match.awayPlayerFlag,
                avatar: match.awayPlayerAvatar,
                score: match.awayScore,
                showScore: match.showScore,
                state: playerState(match, side: 2)
            )
        }
        .appCard(padding: 10)
        .overlay {
            if match.isLive {
                RoundedRectangle(cornerRadius: 16).stroke(palette.success)
            }
        }
    }

    private enum PlayerState {
        case winner
        case loser
        case normal
    }

    private func playerState(_ match: SaiXunMatch, side: Int) -> PlayerState {
        guard match.winnerSide == 1 || match.winnerSide == 2 else { return .normal }
        return match.winnerSide == side ? .winner : .loser
    }

    private func playerRow(
        name: String,
        subName: String,
        flag: SaiXunFlag,
        avatar: SaiXunImageSource,
        score: Int,
        showScore: Bool,
        state: PlayerState
    ) -> some View {
        HStack(spacing: 6) {
            SaiXunPlayerAvatar(source: avatar, size: 28)
            SaiXunFlagView(flag: flag)

            VStack(alignment: .leading, spacing: 1) {
                Text(name)
                    .font(.system(size: 14, weight: state == .winner ? .bold : .medium))
                    .foregroundStyle(nameColor(state))
                    .lineLimit(1)
                if !subName.isEmpty {
                    Text(subName)
                        .font(.system(size: 10))
                        .foregroundStyle(palette.textMuted)
                        .lineLimit(1)
                }
            }

            Spacer(minLength: 4)

            if showScore {
                Text(verbatim: String(score))
                    .font(.system(size: 17, weight: .bold))
                    .foregroundStyle(state == .winner ? palette.brandStrong : nameColor(state))
            }
        }
        .frame(minHeight: 32)
    }

    private func nameColor(_ state: PlayerState) -> Color {
        switch state {
        case .winner, .normal: palette.textPrimary
        case .loser: palette.textMuted
        }
    }

    private func coverURL(_ source: SaiXunImageSource) -> URL? {
        if case .remote(let url) = source { return url }
        return nil
    }

    // MARK: - 状态

    private var skeleton: some View {
        VStack(alignment: .leading, spacing: 8) {
            RoundedRectangle(cornerRadius: 12)
                .fill(palette.border)
                .frame(height: 160)
            skeletonLine(width: 0.56, height: 20)
            skeletonLine(width: 0.72, height: 12)
            skeletonLine(width: 0.4, height: 12)
        }
        .frame(maxWidth: .infinity, alignment: .leading)
    }

    private func skeletonLine(width: CGFloat, height: CGFloat) -> some View {
        GeometryReader { proxy in
            RoundedRectangle(cornerRadius: 6)
                .fill(palette.border)
                .frame(width: proxy.size.width * width, height: height)
        }
        .frame(height: height)
    }

    private func failureState(_ failure: LoadFailure) -> some View {
        let feedback = failure.feedback(resource: "赛事详情")

        return VStack(spacing: 12) {
            ZStack {
                Circle()
                    .fill(palette.brandTint)
                    .frame(width: 56, height: 56)
                Image(systemName: "calendar")
                    .font(.system(size: 24, weight: .medium))
                    .foregroundStyle(palette.brandStrong)
            }
            Text(feedback.title)
                .font(.system(size: 16, weight: .bold))
                .foregroundStyle(palette.textPrimary)
            Text(feedback.description)
                .font(.system(size: 12))
                .foregroundStyle(palette.textSecondary)
                .multilineTextAlignment(.center)

            HStack(spacing: 10) {
                Button(feedback.actionText) {
                    if feedback.retryable {
                        Task { await model.retry() }
                    } else {
                        dismiss()
                    }
                }
                .buttonStyle(.appPill)

                Button("返回上一页") {
                    dismiss()
                }
                .buttonStyle(.plain)
                .font(.system(size: 13, weight: .semibold))
                .foregroundStyle(palette.textSecondary)
                .padding(.horizontal, 16)
                .frame(height: 38)
                .background(palette.subtle, in: Capsule())
                .overlay {
                    Capsule().stroke(palette.border)
                }
            }
            .padding(.top, 4)
        }
        .padding(.horizontal, 24)
        .padding(.top, 80)
        .frame(maxWidth: .infinity)
    }

    private func refreshBanner(_ message: String) -> some View {
        HStack(spacing: 10) {
            Text(message)
                .font(.system(size: 12))
                .foregroundStyle(palette.textSecondary)
            Spacer(minLength: 8)
            Button("重试") {
                Task { await model.retry() }
            }
            .font(.system(size: 12, weight: .semibold))
            .foregroundStyle(palette.brandStrong)
            .buttonStyle(.plain)
        }
        .padding(.horizontal, 11)
        .padding(.vertical, 9)
        .background(palette.card, in: RoundedRectangle(cornerRadius: 9))
        .overlay {
            RoundedRectangle(cornerRadius: 9).stroke(palette.border)
        }
    }
}
