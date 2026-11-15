import SwiftUI

/// 赛季页，对应小程序 `subPages/season/index.vue`。
struct SeasonView: View {
    @EnvironmentObject private var session: SessionStore
    @Environment(\.colorScheme) private var colorScheme
    @Environment(\.dismiss) private var dismiss
    @StateObject private var model: SeasonModel

    init(session: SessionStore, model: SeasonModel? = nil) {
        _model = StateObject(wrappedValue: model ?? SeasonModel(session: session))
    }

    private var palette: AppPalette { AppTheme.palette(for: colorScheme) }

    var body: some View {
        Group {
            switch model.phase {
            case .idle, .loading:
                loadingState
            case .failed(let failure):
                failureState(failure)
            case .ready:
                content
            }
        }
        .background(palette.page.ignoresSafeArea())
        .navigationTitle("赛季")
        .navigationBarTitleDisplayMode(.inline)
        .task { await model.loadIfNeeded() }
        .onChange(of: session.identity) { _, _ in
            Task { await model.loadIfNeeded() }
        }
    }

    // MARK: - 状态

    private var loadingState: some View {
        VStack(spacing: 12) {
            ProgressView()
            Text("加载中...")
                .font(.system(size: 14))
                .foregroundStyle(palette.textMuted)
        }
        .frame(maxWidth: .infinity, maxHeight: .infinity)
        .padding(24)
    }

    private func failureState(_ failure: LoadFailure) -> some View {
        let feedback = failure.feedback(resource: "赛季信息")
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

    // MARK: - 内容

    private var content: some View {
        ScrollView {
            VStack(spacing: 10) {
                if let failure = model.refreshFailure {
                    refreshBanner(failure)
                }
                if model.isActiveSeason, let season = model.season {
                    seasonCard(season)
                } else {
                    noSeasonCard
                }
                historyEntry
                if model.isActiveSeason {
                    gameTypeTabs
                    if model.isSwitchingGameType {
                        switchingCard
                    } else if model.seasonDataUnavailable {
                        seasonDataUnavailableCard
                    } else {
                        if !model.isGuest {
                            recordSection
                        }
                        leaderboardSection
                    }
                }
            }
            .padding(.horizontal, 12)
            .padding(.top, 10)
            .padding(.bottom, 28)
        }
    }

    private func refreshBanner(_ failure: LoadFailure) -> some View {
        HStack(spacing: 10) {
            Text(failure.feedback(resource: "赛季信息").description)
                .font(.system(size: 12))
                .foregroundStyle(palette.brandStrong)
                .frame(maxWidth: .infinity, alignment: .leading)
            Button("重试") {
                Task { await model.retry() }
            }
            .font(.system(size: 12, weight: .semibold))
            .foregroundStyle(palette.brandStrong)
        }
        .padding(10)
        .background(palette.brandTint, in: RoundedRectangle(cornerRadius: 10))
    }

    private func seasonCard(_ season: SeasonInfo) -> some View {
        VStack(alignment: .leading, spacing: 10) {
            HStack(alignment: .center, spacing: 8) {
                Text(season.name)
                    .font(.system(size: 18, weight: .bold))
                    .foregroundStyle(.white)
                    .lineLimit(1)
                Spacer(minLength: 4)
                Text(model.statusText)
                    .font(.system(size: 11, weight: .semibold))
                    .foregroundStyle(.white)
                    .padding(.horizontal, 8)
                    .padding(.vertical, 3)
                    .background(statusColor(season.status), in: RoundedRectangle(cornerRadius: 5))
            }
            Text("\(season.startDate) ~ \(season.endDate)")
                .font(.system(size: 12))
                .foregroundStyle(.white.opacity(0.7))
            if season.status == 1 {
                VStack(alignment: .leading, spacing: 5) {
                    GeometryReader { proxy in
                        ZStack(alignment: .leading) {
                            Capsule().fill(.white.opacity(0.2))
                            Capsule()
                                .fill(palette.success)
                                .frame(width: proxy.size.width * CGFloat(min(100, max(0, model.seasonTimeline.progressPercent))) / 100)
                        }
                    }
                    .frame(height: 6)
                    Text("剩余 \(String(model.seasonTimeline.remainDays)) 天")
                        .font(.system(size: 11))
                        .foregroundStyle(.white.opacity(0.8))
                }
            }
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .padding(16)
        .background(
            LinearGradient(colors: palette.heroColors, startPoint: .topLeading, endPoint: .bottomTrailing),
            in: RoundedRectangle(cornerRadius: 16)
        )
    }

    private func statusColor(_ status: Int) -> Color {
        switch status {
        case 1: palette.success.opacity(0.6)
        case 2: Color.gray.opacity(0.35)
        default: .white.opacity(0.2)
        }
    }

    private var noSeasonCard: some View {
        VStack(spacing: 8) {
            Text("🏆")
                .font(.system(size: 30))
            Text(model.seasonStateView.title)
                .font(.system(size: 14, weight: .semibold))
                .foregroundStyle(palette.textPrimary)
            Text(model.seasonStateView.description)
                .font(.system(size: 12))
                .lineSpacing(4)
                .foregroundStyle(palette.textMuted)
                .multilineTextAlignment(.center)
        }
        .frame(maxWidth: .infinity)
        .padding(.vertical, 26)
        .appCard(padding: 12)
    }

    private var historyEntry: some View {
        NavigationLink(value: AppRoute.seasonHistory(gameType: model.currentGameType)) {
            HStack(spacing: 8) {
                Text("历届赛季")
                    .font(.system(size: 14, weight: .semibold))
                    .foregroundStyle(palette.textPrimary)
                Spacer(minLength: 4)
                Text("回顾往届排名与报告")
                    .font(.system(size: 12))
                    .foregroundStyle(palette.textMuted)
                Image(systemName: "chevron.right")
                    .font(.system(size: 11, weight: .semibold))
                    .foregroundStyle(palette.textMuted)
            }
            .padding(.horizontal, 14)
            .frame(height: 44)
        }
        .buttonStyle(.plain)
        .background(palette.card, in: RoundedRectangle(cornerRadius: 16))
        .overlay {
            RoundedRectangle(cornerRadius: 16).stroke(palette.border)
        }
    }

    private var gameTypeTabs: some View {
        HStack(spacing: 6) {
            ForEach(GameTypeCatalog.ordered) { option in
                let isActive = model.currentGameType == option.value
                Button {
                    Task { await model.selectGameType(option.value) }
                } label: {
                    Text(option.label)
                        .font(.system(size: 12, weight: isActive ? .semibold : .regular))
                        .foregroundStyle(isActive ? palette.brandStrong : palette.textSecondary)
                        .lineLimit(1)
                        .minimumScaleFactor(0.8)
                        .frame(maxWidth: .infinity)
                        .frame(height: 34)
                        .background(isActive ? palette.brandTint : palette.card, in: Capsule())
                        .overlay {
                            Capsule().stroke(isActive ? AppTheme.primary.opacity(0.22) : palette.border)
                        }
                }
                .buttonStyle(.plain)
                .accessibilityAddTraits(isActive ? .isSelected : [])
            }
        }
    }

    private var switchingCard: some View {
        Text("加载中...")
            .font(.system(size: 13))
            .foregroundStyle(palette.textMuted)
            .frame(maxWidth: .infinity)
            .padding(.vertical, 24)
            .appCard()
    }

    private var seasonDataUnavailableCard: some View {
        VStack(spacing: 4) {
            Text("赛季数据暂时没加载出来")
                .font(.system(size: 14, weight: .semibold))
                .foregroundStyle(palette.textPrimary)
            Text("排位成绩会照常记录，稍后重试即可")
                .font(.system(size: 12))
                .foregroundStyle(palette.textMuted)
            Button("重试") {
                Task { await model.retry() }
            }
            .font(.system(size: 13, weight: .semibold))
            .foregroundStyle(palette.brandStrong)
            .padding(.top, 4)
        }
        .frame(maxWidth: .infinity)
        .padding(.vertical, 12)
        .appCard()
    }

    // MARK: - 我的赛季数据

    @ViewBuilder
    private var recordSection: some View {
        if model.recordUnavailable {
            VStack(alignment: .leading, spacing: 10) {
                cardTitle("我的赛季数据")
                VStack(spacing: 4) {
                    Text("数据暂时没加载出来")
                        .font(.system(size: 14, weight: .semibold))
                        .foregroundStyle(palette.textPrimary)
                    Text("排位成绩会照常记录，稍后重试即可")
                        .font(.system(size: 12))
                        .foregroundStyle(palette.textMuted)
                    Button("重试") {
                        Task { await model.retry() }
                    }
                    .font(.system(size: 13, weight: .semibold))
                    .foregroundStyle(palette.brandStrong)
                    .padding(.top, 2)
                }
                .frame(maxWidth: .infinity)
                .padding(.vertical, 6)
            }
            .frame(maxWidth: .infinity, alignment: .leading)
            .appCard()
        } else if let record = model.myRecord {
            VStack(alignment: .leading, spacing: 10) {
                cardTitle("我的赛季数据")
                HStack(spacing: 0) {
                    statItem(String(record.matchesPlayed), label: "场次")
                    statItem(String(record.wins), label: "胜场")
                    statItem("\(String(model.winRate))%", label: "胜率")
                    statItem(String(record.peakRankScore), label: "峰值分")
                }
                if model.myRankAvailable && model.myRank > 0 {
                    Text("当前排名：第 \(String(model.myRank)) 名")
                        .font(.system(size: 13, weight: .medium))
                        .foregroundStyle(palette.brandStrong)
                        .frame(maxWidth: .infinity)
                } else if !model.qualification.qualified {
                    VStack(spacing: 3) {
                        Text("暂未达标，不参与赛季排名")
                            .font(.system(size: 13, weight: .medium))
                            .foregroundStyle(palette.textSecondary)
                        Text(model.qualification.progressText)
                            .font(.system(size: 11))
                            .foregroundStyle(palette.textMuted)
                    }
                    .frame(maxWidth: .infinity)
                }
                if !model.netChangeText.isEmpty {
                    Text("本季排位分 \(String(record.startRankScore)) → \(String(record.endRankScore))（\(model.netChangeText)）")
                        .font(.system(size: 11))
                        .foregroundStyle(palette.textMuted)
                        .frame(maxWidth: .infinity)
                }
            }
            .frame(maxWidth: .infinity, alignment: .leading)
            .appCard()
        } else {
            VStack(alignment: .leading, spacing: 4) {
                Text("本赛季还没有排位记录")
                    .font(.system(size: 14, weight: .semibold))
                    .foregroundStyle(palette.textPrimary)
                Text("完成一场\(model.currentGameTypeLabel)排位即可上榜")
                    .font(.system(size: 12))
                    .foregroundStyle(palette.textMuted)
                if !model.qualification.thresholdText.isEmpty {
                    Text(model.qualification.thresholdText)
                        .font(.system(size: 12))
                        .foregroundStyle(palette.textMuted)
                }
            }
            .frame(maxWidth: .infinity, alignment: .leading)
            .appCard()
        }
    }

    private func statItem(_ value: String, label: String) -> some View {
        VStack(spacing: 2) {
            Text(value)
                .font(.system(size: 20, weight: .bold).monospacedDigit())
                .foregroundStyle(palette.textPrimary)
            Text(label)
                .font(.system(size: 11))
                .foregroundStyle(palette.textMuted)
        }
        .frame(maxWidth: .infinity)
        .accessibilityElement(children: .combine)
    }

    // MARK: - 排行榜

    private var leaderboardSection: some View {
        VStack(alignment: .leading, spacing: 10) {
            cardTitle("赛季排行榜")
            if model.leaderboardUnavailable {
                VStack(spacing: 4) {
                    Text("榜单暂时没加载出来")
                        .font(.system(size: 14, weight: .semibold))
                        .foregroundStyle(palette.textPrimary)
                    Text("稍后重试即可查看最新排名")
                        .font(.system(size: 12))
                        .foregroundStyle(palette.textMuted)
                    Button("重试") {
                        Task { await model.retry() }
                    }
                    .font(.system(size: 13, weight: .semibold))
                    .foregroundStyle(palette.brandStrong)
                    .padding(.top, 2)
                }
                .frame(maxWidth: .infinity)
                .padding(.vertical, 6)
            } else if model.leaderboardRows.isEmpty {
                Text("暂无排行数据")
                    .font(.system(size: 14))
                    .foregroundStyle(palette.textMuted)
                    .frame(maxWidth: .infinity)
                    .padding(.vertical, 20)
            } else {
                VStack(spacing: 0) {
                    ForEach(model.leaderboardRows) { row in
                        leaderboardRow(row)
                        if row.id != model.leaderboardRows.last?.id {
                            Divider().overlay(palette.border)
                        }
                    }
                }
                if model.hasMoreLeaderboard {
                    Button {
                        Task { await model.loadMoreLeaderboard() }
                    } label: {
                        HStack(spacing: 6) {
                            if model.isLoadingMore {
                                ProgressView().controlSize(.small)
                            }
                            Text("加载更多")
                                .font(.system(size: 13))
                        }
                        .foregroundStyle(palette.brandStrong)
                        .frame(maxWidth: .infinity)
                        .padding(.vertical, 10)
                    }
                    .buttonStyle(.plain)
                    .disabled(model.isLoadingMore)
                } else {
                    Text("没有更多了")
                        .font(.system(size: 12))
                        .foregroundStyle(palette.textMuted)
                        .frame(maxWidth: .infinity)
                        .padding(.vertical, 10)
                }
            }
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .appCard()
    }

    private func leaderboardRow(_ row: SeasonLeaderboardRow) -> some View {
        HStack(spacing: 8) {
            Text(row.rank.text)
                .font(.system(size: 12, weight: .semibold))
                .foregroundStyle(rankTextColor(row.rank.rank))
                .frame(width: 26, height: 26)
                .background(rankBackground(row.rank.rank), in: Circle())
            AvatarView(avatar: row.item.avatar, userId: row.item.userId, size: 34)
            VStack(alignment: .leading, spacing: 2) {
                Text(row.item.nickname.isEmpty ? "球友" : row.item.nickname)
                    .font(.system(size: 13, weight: .medium))
                    .foregroundStyle(palette.textPrimary)
                    .lineLimit(1)
                Text("\(String(row.item.rankScore)) 分")
                    .font(.system(size: 11))
                    .foregroundStyle(palette.textMuted)
            }
            Spacer(minLength: 4)
            VStack(alignment: .trailing, spacing: 2) {
                Text("\(String(row.item.wins))胜")
                    .font(.system(size: 12, weight: .medium))
                    .foregroundStyle(palette.brandStrong)
                if row.rank.unqualified {
                    Text("未达标")
                        .font(.system(size: 10))
                        .foregroundStyle(palette.textMuted)
                }
            }
        }
        .padding(.vertical, 7)
        .accessibilityElement(children: .combine)
    }

    private func rankTextColor(_ rank: Int) -> Color {
        switch rank {
        case 1: Color(hex: 0xD97706)
        case 2: palette.textSecondary
        case 3: Color(hex: 0xC2410C)
        default: palette.textMuted
        }
    }

    private func rankBackground(_ rank: Int) -> Color {
        switch rank {
        case 1: Color(hex: 0xFEF3C7)
        case 2: palette.subtle
        case 3: Color(hex: 0xFED7AA)
        default: palette.subtle
        }
    }

    private func cardTitle(_ title: String) -> some View {
        Text(title)
            .font(.system(size: 15, weight: .semibold))
            .foregroundStyle(palette.textPrimary)
    }
}
