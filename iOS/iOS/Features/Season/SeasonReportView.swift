import SwiftUI

/// 赛季报告，对应小程序 `subPages/season/report.vue`。
struct SeasonReportView: View {
    @EnvironmentObject private var session: SessionStore
    @Environment(\.colorScheme) private var colorScheme
    @Environment(\.dismiss) private var dismiss
    @StateObject private var model: SeasonReportModel

    init(session: SessionStore, seasonId: Int64, gameType: Int, model: SeasonReportModel? = nil) {
        _model = StateObject(wrappedValue: model ?? SeasonReportModel(
            session: session,
            seasonId: seasonId,
            gameType: gameType
        ))
    }

    private var palette: AppPalette { AppTheme.palette(for: colorScheme) }

    var body: some View {
        Group {
            if session.identity == nil {
                StateMessageView(
                    icon: "lock.fill",
                    title: "登录后查看赛季报告",
                    actionTitle: "去登录"
                ) {
                    dismiss()
                }
            } else {
                switch model.phase {
                case .idle, .loading:
                    loadingState
                case .failed(let failure):
                    failureState(failure)
                case .empty:
                    StateMessageView(icon: "doc.text", title: "暂无赛季报告")
                case .ready:
                    content
                }
            }
        }
        .background(palette.page.ignoresSafeArea())
        .navigationTitle("赛季报告")
        .navigationBarTitleDisplayMode(.inline)
        .task { await model.loadIfNeeded() }
        .onChange(of: session.identity) { _, _ in
            Task { await model.loadIfNeeded() }
        }
    }

    // MARK: - 状态

    private var loadingState: some View {
        ProgressView()
            .frame(maxWidth: .infinity, maxHeight: .infinity)
            .padding(24)
    }

    private func failureState(_ failure: LoadFailure) -> some View {
        let feedback = failure.feedback(resource: "赛季报告")
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
            VStack(spacing: 12) {
                gameTypeTabs
                summaryCard
                if model.report?.record != nil {
                    rankCard
                }
                if !model.typeWinRows.isEmpty {
                    typeCard
                }
                if let achievements = model.report?.topAchievements, !achievements.isEmpty {
                    achievementCard(achievements)
                }
            }
            .padding(.horizontal, 12)
            .padding(.top, 10)
            .padding(.bottom, 28)
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
                        .foregroundStyle(isActive ? palette.textPrimary : palette.textSecondary)
                        .lineLimit(1)
                        .minimumScaleFactor(0.8)
                        .frame(maxWidth: .infinity)
                        .frame(height: 34)
                        .background(
                            isActive ? AnyShapeStyle(LinearGradient(
                                colors: [palette.gradientStart, palette.gradientEnd],
                                startPoint: .topLeading,
                                endPoint: .bottomTrailing
                            )) : AnyShapeStyle(palette.card),
                            in: Capsule()
                        )
                        .overlay {
                            Capsule().stroke(isActive ? .clear : palette.border)
                        }
                }
                .buttonStyle(.plain)
                .accessibilityAddTraits(isActive ? .isSelected : [])
            }
        }
    }

    private var summaryCard: some View {
        VStack(alignment: .leading, spacing: 12) {
            cardTitle("赛季总结")
            if let record = model.report?.record {
                HStack(spacing: 0) {
                    statItem(String(record.matchesPlayed), label: "总场次")
                    statItem(String(record.wins), label: "胜场")
                    statItem("\(String(model.winRate))%", label: "胜率")
                }
            }
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .appCard()
    }

    private var rankCard: some View {
        VStack(alignment: .leading, spacing: 10) {
            cardTitle("段位变化")
            if let record = model.report?.record {
                HStack(alignment: .center, spacing: 16) {
                    rankValue(String(record.startRankScore), label: "赛季初始", highlighted: false)
                    Text("→")
                        .font(.system(size: 18))
                        .foregroundStyle(palette.textMuted)
                    rankValue(
                        String(record.endRankScore),
                        label: "赛季结束",
                        highlighted: record.endRankScore > record.startRankScore
                    )
                }
                .frame(maxWidth: .infinity)
                Text("赛季峰值：\(String(record.peakRankScore)) 分")
                    .font(.system(size: 13))
                    .foregroundStyle(palette.textSecondary)
                    .frame(maxWidth: .infinity)
                if record.finalRank > 0 {
                    Text("最终排名：第 \(String(record.finalRank)) 名")
                        .font(.system(size: 13))
                        .foregroundStyle(palette.textSecondary)
                        .frame(maxWidth: .infinity)
                } else if record.qualified == false {
                    Text("本季未达标，未参与赛季排名")
                        .font(.system(size: 13))
                        .foregroundStyle(palette.textSecondary)
                        .frame(maxWidth: .infinity)
                }
                Text(model.trendText)
                    .font(.system(size: 13))
                    .foregroundStyle(palette.textSecondary)
                    .frame(maxWidth: .infinity)
            }
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .appCard()
    }

    private var typeCard: some View {
        VStack(alignment: .leading, spacing: 10) {
            cardTitle("各球种胜场")
            ForEach(model.typeWinRows) { row in
                HStack(spacing: 8) {
                    Text(row.label)
                        .font(.system(size: 12))
                        .foregroundStyle(palette.textPrimary)
                        .lineLimit(1)
                        .frame(width: 68, alignment: .leading)
                    GeometryReader { proxy in
                        ZStack(alignment: .leading) {
                            Capsule().fill(palette.border)
                            Capsule()
                                .fill(AppTheme.primary)
                                .frame(width: proxy.size.width * CGFloat(min(100, row.wins * 10)) / 100)
                        }
                    }
                    .frame(height: 8)
                    Text("\(String(row.wins))胜")
                        .font(.system(size: 12))
                        .foregroundStyle(palette.textSecondary)
                        .frame(width: 40, alignment: .trailing)
                }
            }
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .appCard()
    }

    private func achievementCard(_ achievements: [AchievementDef]) -> some View {
        VStack(alignment: .leading, spacing: 10) {
            cardTitle("本赛季解锁成就")
            ForEach(achievements) { achievement in
                HStack(alignment: .center, spacing: 8) {
                    RemoteImage(url: CDNAsset.url(for: achievement.icon), contentMode: .fit) {
                        Text("奖")
                            .font(.system(size: 14, weight: .bold))
                            .foregroundStyle(AppTheme.primary)
                    }
                    .frame(width: 34, height: 34)
                    .background(palette.brandTint, in: RoundedRectangle(cornerRadius: 9))
                    VStack(alignment: .leading, spacing: 2) {
                        Text(achievement.name)
                            .font(.system(size: 13, weight: .semibold))
                            .foregroundStyle(palette.textPrimary)
                            .lineLimit(1)
                        Text(achievement.description.isEmpty ? "本赛季达成的关键里程碑" : achievement.description)
                            .font(.system(size: 11))
                            .foregroundStyle(palette.textSecondary)
                            .lineLimit(2)
                    }
                    Spacer(minLength: 4)
                    Text(unlockTime(achievement.unlockedAt))
                        .font(.system(size: 11))
                        .foregroundStyle(palette.textMuted)
                }
            }
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .appCard()
    }

    private func rankValue(_ value: String, label: String, highlighted: Bool) -> some View {
        VStack(spacing: 2) {
            Text(value)
                .font(.system(size: 20, weight: .bold).monospacedDigit())
                .foregroundStyle(highlighted ? palette.success : palette.textPrimary)
            Text(label)
                .font(.system(size: 11))
                .foregroundStyle(palette.textSecondary)
        }
    }

    private func statItem(_ value: String, label: String) -> some View {
        VStack(spacing: 2) {
            Text(value)
                .font(.system(size: 24, weight: .heavy).monospacedDigit())
                .foregroundStyle(palette.textPrimary)
            Text(label)
                .font(.system(size: 12))
                .foregroundStyle(palette.textSecondary)
        }
        .frame(maxWidth: .infinity)
        .accessibilityElement(children: .combine)
    }

    private func cardTitle(_ title: String) -> some View {
        Text(title)
            .font(.system(size: 15, weight: .semibold))
            .foregroundStyle(palette.textPrimary)
    }

    private func unlockTime(_ value: String?) -> String {
        guard let value, !value.isEmpty else { return "已解锁" }
        return String(value.prefix(10))
    }
}
