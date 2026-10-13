import SwiftUI

/// 只读战报。本期不展示再来一局、海报、交锋记录与荣誉墙入口。
struct MatchReportView: View {
    @EnvironmentObject private var session: SessionStore
    @Environment(\.colorScheme) private var colorScheme
    @Environment(\.dismiss) private var dismiss
    @StateObject private var model: MatchReportModel

    init(session: SessionStore, matchId: Int64) {
        _model = StateObject(wrappedValue: MatchReportModel(session: session, matchId: matchId))
    }

    private var palette: AppPalette { AppTheme.palette(for: colorScheme) }

    var body: some View {
        Group {
            if session.identity == nil {
                MatchHistoryStateMessage(icon: "lock.fill", title: "请登录后查看战报", detail: nil)
            } else {
                switch model.phase {
                case .loading:
                    ProgressView()
                        .frame(maxWidth: .infinity, maxHeight: .infinity)
                case .failed(let message):
                    MatchHistoryStateMessage(
                        icon: "exclamationmark.triangle",
                        title: "战报加载失败",
                        detail: message,
                        actionTitle: "重试",
                        action: { Task { await model.load() } }
                    )
                case .loaded(let report):
                    ScrollView {
                        MatchReportContent(report: report, reward: model.reward)
                            .padding(20)
                    }
                }
            }
        }
        .background(palette.page.ignoresSafeArea())
        .navigationTitle(navigationTitle)
        .navigationBarTitleDisplayMode(.inline)
        .task { await model.load() }
        .onChange(of: session.identity) { _, _ in
            // 退出或切号后不再展示旧账号的战报。
            dismiss()
        }
    }

    private var navigationTitle: String {
        if case .loaded(let report) = model.phase, report.isRefereeView {
            return "执裁详情"
        }
        return "对局战报"
    }
}

private struct MatchReportContent: View {
    @Environment(\.colorScheme) private var colorScheme
    let report: MatchReport
    let reward: MatchRewardState

    private var palette: AppPalette { AppTheme.palette(for: colorScheme) }
    private var isReferee: Bool { report.isRefereeView }
    private var outcome: MatchOutcome { report.outcome }
    private var scoreGap: Int { abs(report.myScore - report.opponentScore) }
    private var unit: String { MatchFormatting.scoreUnit(report.gameType) }
    private var createdAtText: String { MatchTimeFormatting.absolute(report.createdAt) }

    var body: some View {
        VStack(spacing: 16) {
            heroCard
            playersCard
            if !isReferee {
                rankCard
                rewardCard
            }
            if report.refereeBound == true, (report.refereeUserId ?? 0) > 0 {
                refereeCard
            }
            if let highlights = report.summaryHighlights, !highlights.isEmpty {
                highlightCard(highlights)
            }
            if let stats = report.summaryStats, !stats.isEmpty {
                statsCard(stats)
            }
        }
    }

    // MARK: - 结论

    private var heroCard: some View {
        ReportCard {
            VStack(alignment: .leading, spacing: 10) {
                Text(isReferee ? "裁判视角" : "复盘战报")
                    .font(.system(size: 12, weight: .semibold))
                    .foregroundStyle(palette.textSecondary)
                Text(isReferee ? "执裁记录" : outcome.title)
                    .font(.system(size: 26, weight: .bold))
                    .foregroundStyle(palette.textPrimary)
                Text(summaryText)
                    .font(.system(size: 14))
                    .foregroundStyle(palette.textSecondary)

                Divider()
                HStack(alignment: .top) {
                    MetaItem(label: "项目", value: MatchFormatting.gameTypeName(report.gameType))
                    Spacer()
                    if !isReferee {
                        MetaItem(
                            label: report.gameType == 2 ? "分差" : "局差",
                            value: scoreGap == 0 ? "平局" : "\(scoreGap) \(unit)"
                        )
                        Spacer()
                    }
                    MetaItem(label: "结束时间", value: createdAtText)
                }
            }
        }
    }

    private var summaryText: String {
        if isReferee { return "你已完成本场执裁，对局结果已录入系统。" }
        // 整场认输按认输事件判定胜负，局分可能与胜负方向不一致。
        if report.isMatchConcession {
            return outcome == .win ? "对手认输整场，本场判你获胜" : "你已认输整场，本场记为失利"
        }
        switch outcome {
        case .win: return "领先 \(scoreGap) \(unit)收下对局"
        case .loss: return "落后 \(scoreGap) \(unit)结束对局"
        case .draw: return "双方比分持平"
        }
    }

    // MARK: - 双方

    private var playersCard: some View {
        ReportCard {
            HStack(alignment: .top, spacing: 12) {
                PlayerColumn(
                    name: report.myName,
                    avatarUrl: report.myAvatar,
                    score: report.myScore,
                    caption: isReferee ? "选手" : winRateText(report.myWinRate)
                )
                Text("-")
                    .font(.system(size: 20, weight: .semibold))
                    .foregroundStyle(palette.textSecondary)
                    .padding(.top, 36)
                PlayerColumn(
                    name: report.opponentName,
                    avatarUrl: report.opponentAvatar,
                    score: report.opponentScore,
                    caption: isReferee ? "选手" : winRateText(report.opponentWinRate)
                )
            }
            .frame(maxWidth: .infinity)
        }
    }

    private func winRateText(_ rate: Double) -> String {
        "\(Int(rate.rounded()))% 胜率"
    }

    // MARK: - 排位与奖励（仅选手视角）

    private var rankCard: some View {
        ReportSection(title: "排位变化") {
            HStack(alignment: .top, spacing: 12) {
                RankColumn(title: "我的排位分", change: report.myRankChange)
                RankColumn(title: "对手排位分", change: report.opponentRankChange)
            }
            if let details = report.myRankDetails, !details.isEmpty {
                VStack(spacing: 8) {
                    ForEach(Array(details.enumerated()), id: \.offset) { _, detail in
                        HStack {
                            Text(detail.label)
                                .font(.system(size: 14))
                                .foregroundStyle(palette.textSecondary)
                            Spacer()
                            Text(signedText(detail.value))
                                .font(.system(size: 14, weight: .semibold).monospacedDigit())
                                .foregroundStyle(palette.textPrimary)
                        }
                    }
                }
                .padding(.top, 4)
            }
        }
    }

    @ViewBuilder
    private var rewardCard: some View {
        switch reward {
        case .hidden:
            EmptyView()
        case .pending:
            ReportSection(title: "荣誉奖励") {
                Text("本场荣誉奖励仍在处理中")
                    .font(.system(size: 14))
                    .foregroundStyle(palette.textSecondary)
            }
        case .ready(let items):
            ReportSection(title: "本场首次解锁 \(items.count) 项荣誉") {
                VStack(alignment: .leading, spacing: 12) {
                    ForEach(items) { item in
                        HStack(alignment: .top, spacing: 12) {
                            RewardIcon(urlString: item.icon)
                            VStack(alignment: .leading, spacing: 4) {
                                Text(item.achievementName)
                                    .font(.system(size: 15, weight: .semibold))
                                    .foregroundStyle(palette.textPrimary)
                                Text(item.description)
                                    .font(.system(size: 13))
                                    .foregroundStyle(palette.textSecondary)
                                if let title = item.rewardTitleName, !title.isEmpty {
                                    Text("获得称号：\(title)")
                                        .font(.system(size: 12, weight: .semibold))
                                        .foregroundStyle(palette.textPrimary)
                                }
                            }
                        }
                    }
                }
            }
        }
    }

    // MARK: - 裁判与亮点

    private var refereeCard: some View {
        ReportSection(title: "本场裁判") {
            HStack(spacing: 12) {
                MatchHistoryAvatar(urlString: report.refereeAvatar ?? "", size: 48)
                VStack(alignment: .leading, spacing: 4) {
                    Text(report.refereeName ?? "裁判")
                        .font(.system(size: 16, weight: .semibold))
                        .foregroundStyle(palette.textPrimary)
                    if let duration = MatchFormatting.duration(seconds: report.refereeDurationSeconds) {
                        Text("执裁时长 \(duration)")
                            .font(.system(size: 13))
                            .foregroundStyle(palette.textSecondary)
                    }
                }
                Spacer()
            }
            if report.status == 2, MatchFormatting.hasReliableCompletionSource(report.completionSource) {
                Text("完成方式：\(MatchFormatting.completionSourceLabel(report.completionSource))")
                    .font(.system(size: 13))
                    .foregroundStyle(palette.textSecondary)
                    .padding(.top, 4)
            }
        }
    }

    private func highlightCard(_ items: [MatchSummaryItem]) -> some View {
        ReportSection(title: "本场亮点") {
            VStack(alignment: .leading, spacing: 12) {
                ForEach(Array(items.enumerated()), id: \.offset) { _, item in
                    VStack(alignment: .leading, spacing: 4) {
                        Text(item.label)
                            .font(.system(size: 13))
                            .foregroundStyle(palette.textSecondary)
                        Text(item.value)
                            .font(.system(size: 20, weight: .bold))
                            .foregroundStyle(toneColor(item.tone))
                        if let desc = item.desc, !desc.isEmpty {
                            Text(desc)
                                .font(.system(size: 12))
                                .foregroundStyle(palette.textSecondary)
                        }
                    }
                }
            }
        }
    }

    private func statsCard(_ items: [MatchSummaryItem]) -> some View {
        ReportSection(title: "关键数据") {
            VStack(spacing: 10) {
                ForEach(Array(items.enumerated()), id: \.offset) { _, item in
                    HStack {
                        Text(item.label)
                            .font(.system(size: 14))
                            .foregroundStyle(palette.textSecondary)
                        Spacer()
                        Text(item.value)
                            .font(.system(size: 14, weight: .semibold).monospacedDigit())
                            .foregroundStyle(palette.textPrimary)
                    }
                }
            }
        }
    }

    /// 亮点颜色只使用 `AppTheme` 已有的品牌与危险色，其余语义回落到正文色。
    private func toneColor(_ tone: String?) -> Color {
        switch tone {
        case "gold", "orange": AppTheme.primary
        case "red": AppTheme.danger
        default: palette.textPrimary
        }
    }

    private func signedText(_ value: Int) -> String {
        value > 0 ? "+\(value)" : "\(value)"
    }
}

private struct ReportCard<Content: View>: View {
    @Environment(\.colorScheme) private var colorScheme
    @ViewBuilder let content: () -> Content

    var body: some View {
        content()
            .frame(maxWidth: .infinity, alignment: .leading)
            .padding(20)
            .background(AppTheme.palette(for: colorScheme).card, in: RoundedRectangle(cornerRadius: 16))
    }
}

private struct ReportSection<Content: View>: View {
    @Environment(\.colorScheme) private var colorScheme
    let title: String
    @ViewBuilder let content: () -> Content

    var body: some View {
        let palette = AppTheme.palette(for: colorScheme)
        VStack(alignment: .leading, spacing: 12) {
            Text(title)
                .font(.system(size: 16, weight: .semibold))
                .foregroundStyle(palette.textPrimary)
            content()
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .padding(20)
        .background(palette.card, in: RoundedRectangle(cornerRadius: 16))
    }
}

private struct MetaItem: View {
    @Environment(\.colorScheme) private var colorScheme
    let label: String
    let value: String

    var body: some View {
        let palette = AppTheme.palette(for: colorScheme)
        VStack(alignment: .leading, spacing: 4) {
            Text(label)
                .font(.system(size: 12))
                .foregroundStyle(palette.textSecondary)
            Text(value)
                .font(.system(size: 14, weight: .semibold))
                .foregroundStyle(palette.textPrimary)
        }
    }
}

private struct PlayerColumn: View {
    @Environment(\.colorScheme) private var colorScheme
    let name: String
    let avatarUrl: String
    let score: Int
    let caption: String

    var body: some View {
        let palette = AppTheme.palette(for: colorScheme)
        VStack(spacing: 8) {
            MatchHistoryAvatar(urlString: avatarUrl, size: 56)
            Text(name)
                .font(.system(size: 15, weight: .semibold))
                .foregroundStyle(palette.textPrimary)
                .lineLimit(1)
            Text("\(score)")
                .font(.system(size: 34, weight: .bold).monospacedDigit())
                .foregroundStyle(palette.textPrimary)
            Text(caption)
                .font(.system(size: 12))
                .foregroundStyle(palette.textSecondary)
        }
        .frame(maxWidth: .infinity)
    }
}

private struct RankColumn: View {
    @Environment(\.colorScheme) private var colorScheme
    let title: String
    let change: Int

    var body: some View {
        let palette = AppTheme.palette(for: colorScheme)
        VStack(alignment: .leading, spacing: 6) {
            Text(title)
                .font(.system(size: 13))
                .foregroundStyle(palette.textSecondary)
            Text(change > 0 ? "+\(change)" : "\(change)")
                .font(.system(size: 22, weight: .bold).monospacedDigit())
                .foregroundStyle(change > 0 ? AppTheme.primary : palette.textPrimary)
        }
        .frame(maxWidth: .infinity, alignment: .leading)
    }
}

private struct RewardIcon: View {
    @Environment(\.colorScheme) private var colorScheme
    let urlString: String

    var body: some View {
        ZStack {
            RoundedRectangle(cornerRadius: 10)
                .fill(AppTheme.primary.opacity(0.14))
            Image(systemName: "trophy.fill")
                .foregroundStyle(AppTheme.primary)
            if !urlString.isEmpty, let url = URL(string: urlString) {
                AsyncImage(url: url) { image in
                    image.resizable().scaledToFit()
                } placeholder: {
                    Color.clear
                }
                .padding(6)
            }
        }
        .frame(width: 44, height: 44)
        .accessibilityHidden(true)
    }
}
