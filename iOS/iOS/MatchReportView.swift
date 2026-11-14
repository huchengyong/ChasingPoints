import SwiftUI

/// 赛果页（小程序 `matchResult`）：结论、排位、奖励、本场用杆、裁判、亮点与底部操作。
/// `fromHistory` 只影响底部操作：历史记录进入时不提供「再来一局」，分享按钮改为「生成战绩海报」。
struct MatchReportView: View {
    @EnvironmentObject private var session: SessionStore
    @EnvironmentObject private var navigator: AppNavigator
    @EnvironmentObject private var toast: ToastCenter
    @Environment(\.colorScheme) private var colorScheme
    @Environment(\.dismiss) private var dismiss
    @StateObject private var model: MatchReportModel
    private let fromHistory: Bool

    init(session: SessionStore, matchId: Int64, fromHistory: Bool = false, model: MatchReportModel? = nil) {
        _model = StateObject(wrappedValue: model ?? MatchReportModel(session: session, matchId: matchId))
        self.fromHistory = fromHistory
    }

    private var palette: AppPalette { AppTheme.palette(for: colorScheme) }

    var body: some View {
        Group {
            if session.identity == nil {
                MatchHistoryStateMessage(icon: "lock.fill", title: "请登录后查看战报", detail: nil)
            } else {
                switch model.phase {
                case .loading:
                    VStack(spacing: 10) {
                        ProgressView()
                        Text("正在生成本场战报...")
                            .font(.system(size: 13))
                            .foregroundStyle(palette.textSecondary)
                    }
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
                        MatchResultContent(
                            report: report,
                            reward: rewardView,
                            session: session,
                            showGear: gearEnabled(report),
                            onHonorWall: { openHonorWall(report) }
                        )
                        .padding(20)
                    }
                }
            }
        }
        .background(palette.page.ignoresSafeArea())
        .navigationTitle(navigationTitle)
        .navigationBarTitleDisplayMode(.inline)
        .safeAreaInset(edge: .bottom) {
            if case .loaded(let report) = model.phase, session.identity != nil, isPlayerView(report) {
                footer(report)
            }
        }
        .task { await model.load() }
        .onChange(of: session.identity) { _, _ in
            // 退出或切号后不再展示旧账号的战报。
            dismiss()
        }
    }

    private var navigationTitle: String {
        if case .loaded(let report) = model.phase {
            return MatchRefereeView.resultViewConfig(
                viewerRole: report.viewerRole ?? "player1",
                status: report.status
            ).pageTitle
        }
        return "战报"
    }

    private var rewardView: MatchRewardView {
        MatchRewardView(
            state: model.reward,
            message: model.rewardMessage,
            shouldRetry: model.rewardShouldRetry
        )
    }

    private func isPlayerView(_ report: MatchReport) -> Bool {
        report.viewerRole != "referee"
    }

    private func gearEnabled(_ report: MatchReport) -> Bool {
        GearMatchRules.shouldShowMatchGear(
            status: report.status,
            viewerRole: report.viewerRole ?? "",
            isPlayer1: report.isPlayer1 ?? false
        )
    }

    // MARK: - 底部操作

    private func footer(_ report: MatchReport) -> some View {
        VStack(spacing: 8) {
            HStack(spacing: 8) {
                if !fromHistory {
                    Button("再来一局") { rematch(report) }
                        .buttonStyle(.appPrimary)
                }
                Button(fromHistory ? "生成战绩海报" : "分享战绩") {
                    navigator.open(.shareResult(matchId: report.id))
                }
                .buttonStyle(.appSecondary)
            }
            HStack(spacing: 20) {
                Button("查看交锋记录") { openH2H(report) }
                if MatchReplayEntry.canOpenMatchReplay(
                    status: report.status,
                    viewerRole: report.viewerRole ?? ""
                ) {
                    Button("单场复盘") { navigator.open(.matchReplay(matchId: report.id)) }
                }
            }
            .font(.system(size: 14))
            .foregroundStyle(palette.brandStrong)
            .padding(.vertical, 2)
        }
        .padding(.horizontal, 20)
        .padding(.vertical, 12)
        .background(palette.card)
    }

    private func rematch(_ report: MatchReport) {
        guard let context = MatchCoreFlow.rematch(report) else {
            toast.show("缺少对手信息，请重新扫码开局")
            return
        }
        // 只接受原对手的匹配码，球种与类型预填为上一场的设置。
        navigator.pendingRematch = context
        navigator.requestScan(isSignedIn: session.identity != nil)
    }

    private func openH2H(_ report: MatchReport) {
        let opponentId = report.opponentId ?? 0
        guard opponentId > 0 else {
            toast.show("暂无可用的交锋记录")
            return
        }
        navigator.open(.h2hRecord(
            opponentId: opponentId,
            targetUserId: 0,
            targetName: "",
            targetAvatar: "",
            opponentName: report.opponentName.isEmpty ? "对手" : report.opponentName
        ))
    }

    private func openHonorWall(_ report: MatchReport) {
        navigator.open(.achievementWall(gameType: MatchResultPage.honorWallGameType(report.gameType)))
    }
}

private struct MatchResultContent: View {
    @Environment(\.colorScheme) private var colorScheme
    let report: MatchReport
    let reward: MatchRewardView
    let session: SessionStore
    let showGear: Bool
    let onHonorWall: () -> Void

    private var palette: AppPalette { AppTheme.palette(for: colorScheme) }
    private var config: RefereeResultViewConfig {
        MatchRefereeView.resultViewConfig(viewerRole: report.viewerRole ?? "player1", status: report.status)
    }
    private var isReferee: Bool { report.isRefereeView }
    private var outcome: MatchOutcome {
        MatchResultPage.outcome(result: report.result, myScore: report.myScore, opponentScore: report.opponentScore)
    }
    private var isWin: Bool { !isReferee && outcome == .win }
    private var isDraw: Bool { !isReferee && outcome == .draw }
    private var scoreGap: Int { abs(report.myScore - report.opponentScore) }
    private var unit: String { MatchFormatting.scoreUnit(report.gameType) }
    private var gapLabel: String { report.gameType == 2 ? "分差" : "局差" }
    private var showRankChange: Bool {
        config.showRankChange && ((report.matchMode ?? "").isEmpty || report.matchMode == "ranked")
    }
    private var finishMethodText: String {
        FinishCountdown.resolveMethodText(MatchResultPage.makeFinishInput(report))
    }
    private var scoreView: ChasingScoreView? {
        guard report.gameType == 2 else { return nil }
        return ChasingScore.resolveView(report.scoreBreakdown, swap: report.isPlayer1 == false, neutral: isReferee)
    }
    private var rankingRights: RankingRightsSummary {
        MemberRankingRights.summary(MatchRankingRightsInput(
            matchMode: report.matchMode,
            rankRulesVersion: report.rankRulesVersion,
            achievements: report.achievements,
            myRankDetails: report.myRankDetails
        ))
    }

    var body: some View {
        VStack(spacing: 16) {
            heroCard
            if showRankChange {
                rankSection
            }
            if reward.visible {
                rewardSection
            }
            if showGear {
                MatchGearCardView(session: session, matchId: report.id, enabled: showGear)
            }
            if config.showRefereeCard, refereeCard.hasReferee {
                refereeSection
            }
            if config.showPersonalInsights {
                insightSection
                if let scoreView {
                    ChasingScoreCard(view: scoreView, showEvents: false)
                }
                statsSection
            }
        }
    }

    // MARK: - 结论

    private var resultTitle: String {
        if isReferee { return "执裁记录" }
        if isDraw { return "握手言和" }
        return isWin ? "拿下这一场" : "这场先记下"
    }

    private var resultTagText: String {
        if isReferee { return "裁判视角" }
        if isDraw { return "平局战报" }
        return isWin ? "胜利战报" : "复盘战报"
    }

    private var resultSubtitle: String {
        if isReferee { return "你已完成本场执裁，对局结果已录入系统。" }
        if isDraw { return "双方打成平手，下一场再分高下。" }
        // 整场认输按认输事件判定胜负，局分可能与胜负方向不一致，需要明确说明。
        if report.isMatchConcession {
            return isWin ? "对手认输整场，本场判你获胜。" : "你已认输整场，本场记为失利。"
        }
        if isWin {
            return scoreGap >= 3
                ? "整场节奏控制稳定，关键分兑现得更彻底。"
                : "比分咬得很紧，但你在收官阶段更稳。"
        }
        return scoreGap <= 1
            ? "差距很小，下一场把关键球处理得更坚决。"
            : "这场被对手带走节奏，回看数据后更容易找到突破口。"
    }

    private var scoreSummaryText: String {
        if isReferee { return report.gameType == 2 ? "本场最终比分" : "本场最终局数" }
        if isDraw { return report.gameType == 2 ? "双方比分持平" : "双方局数持平" }
        return isWin
            ? "领先 \(String(scoreGap)) \(unit)收下对局"
            : "落后 \(String(scoreGap)) \(unit)结束对局"
    }

    private var heroCard: some View {
        ReportCard {
            VStack(alignment: .leading, spacing: 14) {
                HStack {
                    Text(resultTagText)
                        .font(.system(size: 11, weight: .semibold))
                        .foregroundStyle(tagColor)
                        .padding(.horizontal, 8)
                        .padding(.vertical, 3)
                        .background(palette.subtle, in: Capsule())
                    Spacer()
                    Text(MatchResultPage.relativeTimeText(report))
                        .font(.system(size: 12))
                        .foregroundStyle(palette.textSecondary)
                }

                VStack(alignment: .leading, spacing: 6) {
                    Text(resultTitle)
                        .font(.system(size: 26, weight: .bold))
                        .foregroundStyle(palette.textPrimary)
                    Text(resultSubtitle)
                        .font(.system(size: 14))
                        .foregroundStyle(palette.textSecondary)
                }

                HStack(alignment: .top, spacing: 12) {
                    PlayerColumn(
                        name: report.myName.isEmpty ? "我" : report.myName,
                        avatar: report.myAvatar,
                        userId: leftUserId,
                        vipLevel: report.myVipLevel ?? 0,
                        rankLevel: report.myRankLevel ?? 0,
                        caption: isReferee ? "选手" : "\(formatWinRate(report.myWinRate)) 胜率"
                    )
                    VStack(spacing: 6) {
                        Text("\(String(report.myScore)) : \(String(report.opponentScore))")
                            .font(.system(size: 24, weight: .bold).monospacedDigit())
                            .foregroundStyle(palette.textPrimary)
                        Text(scoreSummaryText)
                            .font(.system(size: 11))
                            .foregroundStyle(palette.textSecondary)
                            .multilineTextAlignment(.center)
                    }
                    .padding(.top, 30)
                    .frame(maxWidth: .infinity)

                    PlayerColumn(
                        name: report.opponentName.isEmpty ? "对手" : report.opponentName,
                        avatar: report.opponentAvatar,
                        userId: rightUserId,
                        vipLevel: report.opponentVipLevel ?? 0,
                        rankLevel: report.opponentRankLevel ?? 0,
                        caption: isReferee ? "选手" : "\(formatWinRate(report.opponentWinRate)) 胜率"
                    )
                }

                HStack(spacing: 8) {
                    MetaPill(label: "项目", value: MatchFormatting.gameTypeName(report.gameType))
                    MetaPill(label: gapLabel, value: isDraw ? "平局" : "\(String(scoreGap)) \(unit)")
                    MetaPill(
                        label: MatchResultPage.timeMetaLabel(report),
                        value: MatchResultPage.fullTimeText(report)
                    )
                }

                if !finishMethodText.isEmpty {
                    Text("结束方式：\(finishMethodText)")
                        .font(.system(size: 12))
                        .foregroundStyle(palette.textSecondary)
                }
            }
        }
    }

    private var tagColor: Color {
        if isReferee { return palette.textSecondary }
        if isDraw { return palette.info }
        return isWin ? palette.brandStrong : palette.textSecondary
    }

    private func formatWinRate(_ value: Double) -> String {
        "\(String(Int(value.rounded())))%"
    }

    private var leftUserId: Int64 {
        let player1 = report.player1Id ?? 0
        let player2 = report.player2Id ?? 0
        if isReferee { return player1 }
        return report.isPlayer1 == true ? player1 : player2
    }

    private var rightUserId: Int64 {
        let player1 = report.player1Id ?? 0
        let player2 = report.player2Id ?? 0
        if isReferee { return player2 }
        return report.isPlayer1 == true ? player2 : player1
    }

    // MARK: - 排位

    private var rankSection: some View {
        ReportSection(title: "排位变化") {
            Text("这场对段位走势的直接影响")
                .font(.system(size: 12))
                .foregroundStyle(palette.textSecondary)
            switch report.rankSettlement {
            case .pending(let clock):
                unsettledRankColumns(placeholder: "待生效")
                rankNote(clock.map { "排位积分将于 \($0) 生效。" } ?? "排位积分将在异议期结束后生效。")
            case .voided:
                unsettledRankColumns(placeholder: "不计分")
                rankNote("本场已提出异议，不计排位积分。")
            case .counted:
                HStack(alignment: .top, spacing: 12) {
                    RankColumn(title: "我的排位分", change: report.myRankChange)
                    RankColumn(title: "对手排位分", change: report.opponentRankChange)
                }
                if let details = report.myRankDetails, !details.isEmpty {
                    VStack(spacing: 8) {
                        ForEach(Array(details.enumerated()), id: \.offset) { _, detail in
                            HStack {
                                Text(detail.label)
                                    .font(.system(size: 13))
                                    .foregroundStyle(palette.textSecondary)
                                Spacer()
                                Text(signedText(detail.value))
                                    .font(.system(size: 13, weight: .semibold).monospacedDigit())
                                    .foregroundStyle(palette.textPrimary)
                            }
                        }
                    }
                } else {
                    rankNote(myRankNote)
                }
                rankNote(opponentRankNote)
            }
        }
    }

    private var myRankNote: String {
        let change = report.myRankChange
        if change > 0 { return "本场胜利带来稳定上分。" }
        if change < 0 { return "本场失利导致排位小幅回落。" }
        return "本场对排位暂无直接影响，可能触发了平局或 0 分保护。"
    }

    private var opponentRankNote: String {
        let change = report.opponentRankChange
        if change > 0 { return "对手本场状态更受益于排位分规则。" }
        if change < 0 { return "对手本场排位分出现回撤。" }
        return "对手本场排位分保持不变。"
    }

    private func unsettledRankColumns(placeholder: String) -> some View {
        HStack(alignment: .top, spacing: 12) {
            RankColumn(title: "我的排位分", placeholder: placeholder)
            RankColumn(title: "对手排位分", placeholder: placeholder)
        }
    }

    private func rankNote(_ text: String) -> some View {
        Text(text)
            .font(.system(size: 12))
            .foregroundStyle(palette.textSecondary)
    }

    // MARK: - 奖励

    @ViewBuilder
    private var rewardSection: some View {
        if reward.state == .pending {
            ReportCard {
                HStack(alignment: .top, spacing: 12) {
                    Image(systemName: "arrow.triangle.2.circlepath")
                        .font(.system(size: 20))
                        .foregroundStyle(palette.textSecondary)
                    VStack(alignment: .leading, spacing: 4) {
                        Text("正在同步本场荣誉")
                            .font(.system(size: 15, weight: .semibold))
                            .foregroundStyle(palette.textPrimary)
                        Text(reward.message)
                            .font(.system(size: 13))
                            .foregroundStyle(palette.textSecondary)
                        Text(reward.shouldRetry ? "即将自动重试，请稍候…" : "稍后再次进入本页即可继续查看")
                            .font(.system(size: 12))
                            .foregroundStyle(palette.textMuted)
                    }
                    Spacer()
                }
            }
        } else {
            ReportCard {
                VStack(alignment: .leading, spacing: 12) {
                    HStack(alignment: .top) {
                        VStack(alignment: .leading, spacing: 4) {
                            Text("本场新解锁")
                                .font(.system(size: 12, weight: .semibold))
                                .foregroundStyle(palette.brandStrong)
                            Text("荣誉已永久记录")
                                .font(.system(size: 17, weight: .bold))
                                .foregroundStyle(palette.textPrimary)
                            Text(reward.message)
                                .font(.system(size: 12))
                                .foregroundStyle(palette.textSecondary)
                        }
                        Spacer()
                        Text("+\(String(reward.rewards.count))")
                            .font(.system(size: 18, weight: .bold))
                            .foregroundStyle(palette.brandStrong)
                    }

                    ForEach(reward.rewards) { item in
                        HStack(alignment: .top, spacing: 10) {
                            RewardIcon(url: item.iconURL)
                            VStack(alignment: .leading, spacing: 3) {
                                Text("新成就")
                                    .font(.system(size: 11))
                                    .foregroundStyle(palette.textMuted)
                                Text(item.displayName)
                                    .font(.system(size: 14, weight: .semibold))
                                    .foregroundStyle(palette.textPrimary)
                                Text(item.displayDescription)
                                    .font(.system(size: 12))
                                    .foregroundStyle(palette.textSecondary)
                                if let title = item.rewardTitleName, !title.isEmpty {
                                    Text("奖励称号 · \(title)")
                                        .font(.system(size: 11, weight: .semibold))
                                        .foregroundStyle(palette.brandStrong)
                                }
                            }
                        }
                    }

                    Button(action: onHonorWall) {
                        HStack(spacing: 4) {
                            Text("查看荣誉墙")
                            Image(systemName: "chevron.right")
                        }
                        .font(.system(size: 13, weight: .semibold))
                        .foregroundStyle(palette.brandStrong)
                    }
                }
            }
        }
    }

    // MARK: - 裁判

    private var refereeCard: RefereeIdentityCard {
        MatchRefereeView.identityCard(
            refereeBound: report.refereeBound ?? false,
            refereeUserId: report.refereeUserId ?? 0,
            refereeName: report.refereeName ?? "",
            refereeAvatar: report.refereeAvatar ?? "",
            refereeDurationSeconds: report.refereeDurationSeconds ?? 0,
            completedByUserId: report.completedByUserId ?? 0,
            completionSource: report.completionSource ?? "",
            status: report.status
        )
    }

    private var refereeSection: some View {
        ReportSection(title: refereeCard.neutralLabel) {
            HStack(spacing: 12) {
                MatchHistoryAvatar(urlString: refereeCard.refereeAvatar, size: 48)
                VStack(alignment: .leading, spacing: 4) {
                    Text(refereeCard.refereeName)
                        .font(.system(size: 15, weight: .semibold))
                        .foregroundStyle(palette.textPrimary)
                    if !refereeCard.refereeDurationText.isEmpty {
                        Text("执裁时长 \(refereeCard.refereeDurationText)")
                            .font(.system(size: 12))
                            .foregroundStyle(palette.textSecondary)
                    }
                }
                Spacer()
            }
            if refereeCard.hasReliableAttribution {
                Text("完成方式：\(refereeCard.completionLabel)")
                    .font(.system(size: 12))
                    .foregroundStyle(palette.textSecondary)
            }
        }
    }

    // MARK: - 亮点与数据

    private var highlightItems: [MatchSummaryItem] { report.summaryHighlights ?? [] }

    private var insightSection: some View {
        ReportSection(title: "本场亮点") {
            Text(
                highlightItems.isEmpty
                    ? "当前\(MatchFormatting.gameTypeName(report.gameType))模式暂无可展示亮点，但战绩已完成记录"
                    : "按\(MatchFormatting.gameTypeName(report.gameType))模式提炼这场最值得记住的表现"
            )
            .font(.system(size: 12))
            .foregroundStyle(palette.textSecondary)

            if highlightItems.isEmpty {
                Text("暂无特殊战绩，本场战绩已正常计入记录。")
                    .font(.system(size: 13))
                    .foregroundStyle(palette.textMuted)
            } else {
                ForEach(Array(highlightItems.enumerated()), id: \.offset) { _, item in
                    VStack(alignment: .leading, spacing: 2) {
                        Text(item.value)
                            .font(.system(size: 20, weight: .bold))
                            .foregroundStyle(toneColor(item.tone))
                        Text(item.label)
                            .font(.system(size: 12))
                            .foregroundStyle(palette.textSecondary)
                        if let desc = item.desc, !desc.isEmpty {
                            Text(desc)
                                .font(.system(size: 11))
                                .foregroundStyle(palette.textMuted)
                        }
                    }
                    .frame(maxWidth: .infinity, alignment: .leading)
                    .padding(10)
                    .background(palette.subtle, in: RoundedRectangle(cornerRadius: 8))
                }
            }

            VStack(alignment: .leading, spacing: 6) {
                HStack(spacing: 8) {
                    Text(rankingRights.badgeText)
                        .font(.system(size: 11, weight: .semibold))
                        .foregroundStyle(rankingToneColor(rankingRights.tone))
                    Text(rankingRights.title)
                        .font(.system(size: 14, weight: .semibold))
                        .foregroundStyle(palette.textPrimary)
                }
                Text(rankingRights.description)
                    .font(.system(size: 12))
                    .foregroundStyle(palette.textSecondary)
            }
            .frame(maxWidth: .infinity, alignment: .leading)
            .padding(12)
            .background(palette.brandTint, in: RoundedRectangle(cornerRadius: 10))
        }
    }

    private var statsSection: some View {
        ReportSection(title: "本场数据") {
            Text("关键表现对比与记录摘要")
                .font(.system(size: 12))
                .foregroundStyle(palette.textSecondary)
            ForEach(Array((report.summaryStats ?? []).enumerated()), id: \.offset) { _, item in
                HStack(alignment: .top) {
                    VStack(alignment: .leading, spacing: 2) {
                        Text(item.label)
                            .font(.system(size: 13))
                            .foregroundStyle(palette.textPrimary)
                        if let desc = item.desc, !desc.isEmpty {
                            Text(desc)
                                .font(.system(size: 11))
                                .foregroundStyle(palette.textMuted)
                        }
                    }
                    Spacer()
                    Text(item.value)
                        .font(.system(size: 14, weight: .semibold).monospacedDigit())
                        .foregroundStyle(toneColor(item.tone))
                }
            }
        }
    }

    private func toneColor(_ tone: String?) -> Color {
        switch tone {
        case "gold", "orange": AppTheme.primary
        case "red": AppTheme.danger
        case "green", "emerald": palette.success
        case "blue", "cyan": palette.info
        default: palette.textPrimary
        }
    }

    private func rankingToneColor(_ tone: String) -> Color {
        switch tone {
        case "gold": AppTheme.primary
        case "emerald", "green": palette.success
        case "blue": palette.info
        default: palette.textSecondary
        }
    }

    private func signedText(_ value: Int) -> String {
        value > 0 ? "+\(String(value))" : String(value)
    }
}

// MARK: - 卡片与小组件

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
        VStack(alignment: .leading, spacing: 10) {
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

private struct MetaPill: View {
    @Environment(\.colorScheme) private var colorScheme
    let label: String
    let value: String

    var body: some View {
        let palette = AppTheme.palette(for: colorScheme)
        VStack(alignment: .leading, spacing: 3) {
            Text(label)
                .font(.system(size: 10))
                .foregroundStyle(palette.textMuted)
            Text(value)
                .font(.system(size: 12, weight: .semibold))
                .foregroundStyle(palette.textPrimary)
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .padding(8)
        .background(palette.subtle, in: RoundedRectangle(cornerRadius: 8))
    }
}

private struct PlayerColumn: View {
    @Environment(\.colorScheme) private var colorScheme
    let name: String
    let avatar: String
    let userId: Int64
    let vipLevel: Int
    let rankLevel: Int
    let caption: String

    var body: some View {
        let palette = AppTheme.palette(for: colorScheme)
        VStack(spacing: 6) {
            MemberAvatarView(
                avatar: avatar,
                userId: userId,
                vipLevel: vipLevel,
                rankLevel: rankLevel,
                size: 46
            )
            Text(name)
                .font(.system(size: 14, weight: .semibold))
                .foregroundStyle(palette.textPrimary)
                .lineLimit(1)
            Text(caption)
                .font(.system(size: 11))
                .foregroundStyle(palette.textSecondary)
        }
        .frame(maxWidth: .infinity)
    }
}

private struct RankColumn: View {
    @Environment(\.colorScheme) private var colorScheme
    let title: String
    /// 非 nil 时用文字代替数值，用于排位尚未生效或已作废，避免显示成 +0。
    let placeholder: String?
    let change: Int

    init(title: String, change: Int) {
        self.title = title
        self.placeholder = nil
        self.change = change
    }

    init(title: String, placeholder: String) {
        self.title = title
        self.placeholder = placeholder
        self.change = 0
    }

    var body: some View {
        let palette = AppTheme.palette(for: colorScheme)
        VStack(alignment: .leading, spacing: 6) {
            Text(title)
                .font(.system(size: 12))
                .foregroundStyle(palette.textSecondary)
            if let placeholder {
                Text(placeholder)
                    .font(.system(size: 16, weight: .bold))
                    .foregroundStyle(palette.textSecondary)
            } else {
                Text(change > 0 ? "+\(String(change))" : String(change))
                    .font(.system(size: 20, weight: .bold).monospacedDigit())
                    .foregroundStyle(change > 0 ? AppTheme.primary : palette.textPrimary)
            }
        }
        .frame(maxWidth: .infinity, alignment: .leading)
    }
}

private struct RewardIcon: View {
    let url: URL?

    var body: some View {
        ZStack {
            RoundedRectangle(cornerRadius: 10)
                .fill(AppTheme.primary.opacity(0.14))
            Image(systemName: "trophy.fill")
                .foregroundStyle(AppTheme.primary)
            if let url {
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
