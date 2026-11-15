import Charts
import SwiftUI

/// 竞技分析，对应小程序 `subPages/user/statsDetail.vue`。
struct StatsDetailView: View {
    @EnvironmentObject private var session: SessionStore
    @EnvironmentObject private var navigator: AppNavigator
    @Environment(\.colorScheme) private var colorScheme
    @Environment(\.dismiss) private var dismiss
    @StateObject private var model: StatsDetailModel

    init(session: SessionStore, gameType: Int, model: StatsDetailModel? = nil) {
        _model = StateObject(wrappedValue: model ?? StatsDetailModel(session: session, gameType: gameType))
    }

    private var palette: AppPalette { AppTheme.palette(for: colorScheme) }

    var body: some View {
        Group {
            if session.identity == nil {
                StateMessageView(
                    icon: "lock.fill",
                    title: "登录后查看竞技分析",
                    actionTitle: "去登录"
                ) {
                    navigator.openLogin()
                }
            } else {
                content
            }
        }
        .background(palette.page.ignoresSafeArea())
        .navigationTitle("竞技分析")
        .navigationBarTitleDisplayMode(.inline)
        .task { await model.loadIfNeeded() }
        .onChange(of: session.identity) { _, _ in
            Task { await model.reloadForIdentityChange() }
        }
    }

    @ViewBuilder
    private var content: some View {
        if model.isInitialLoading {
            VStack(spacing: 10) {
                ProgressView()
                Text("加载统计数据...")
                    .font(.system(size: 14))
                    .foregroundStyle(palette.textSecondary)
            }
            .frame(maxWidth: .infinity, maxHeight: .infinity)
        } else if model.isInitialError {
            errorState
        } else {
            ScrollView {
                VStack(spacing: 12) {
                    if let message = model.refreshMessage {
                        refreshBanner(message)
                    }
                    gameTypeSection
                    performanceEntry
                    trendSection
                    rankSection
                    highScoreSection
                    durationSection
                    opponentSection
                }
                .padding(.horizontal, 12)
                .padding(.vertical, 12)
            }
        }
    }

    private var errorState: some View {
        let feedback = model.failureFeedback
            ?? LoadFailure(category: .business, message: "").feedback(resource: "统计数据")

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

    private func refreshBanner(_ message: String) -> some View {
        HStack(alignment: .firstTextBaseline, spacing: 8) {
            Text(message)
                .font(.system(size: 13))
                .foregroundStyle(palette.textSecondary)
            Spacer(minLength: 8)
            Button("重试") {
                Task { await model.retry() }
            }
            .font(.system(size: 13, weight: .semibold))
            .foregroundStyle(palette.brandStrong)
        }
        .padding(12)
        .background(palette.brandTint, in: RoundedRectangle(cornerRadius: 10))
    }

    // MARK: - 分球种统计

    private var gameTypeSection: some View {
        VStack(alignment: .leading, spacing: 12) {
            sectionTitle("分球种统计")
            gameTypeTabs

            LazyVGrid(columns: [GridItem(.flexible()), GridItem(.flexible())], spacing: 0) {
                statItem("\(StatsDetailRules.formatPercent(stats?.winRate ?? 0))%", "胜率", color: palette.brandStrong)
                statItem(String(stats?.totalMatches ?? 0), "总场次", color: palette.textPrimary)
                statItem(String(stats?.wins ?? 0), "胜场", color: palette.success)
                statItem(String(stats?.losses ?? 0), "负场", color: AppTheme.danger)
            }

            winRateBar
        }
        .appCard(padding: 14)
    }

    private var stats: StatsGameTypeStats? { model.currentStats }

    private var gameTypeTabs: some View {
        HStack(spacing: 4) {
            ForEach(GameTypeCatalog.ordered) { option in
                let isActive = model.currentGameType == option.value
                let isPending = model.isRefreshing && isActive

                Button {
                    Task { await model.selectGameType(option.value) }
                } label: {
                    HStack(spacing: 4) {
                        Text(option.label)
                            .lineLimit(1)
                            .minimumScaleFactor(0.8)
                        if isPending {
                            ProgressView()
                                .controlSize(.mini)
                                .tint(.white)
                        }
                    }
                    .font(.system(size: 12, weight: isActive ? .semibold : .regular))
                    .foregroundStyle(isActive ? Color.white : palette.textSecondary)
                    .frame(maxWidth: .infinity)
                    .frame(height: 32)
                    .background(isActive ? AppTheme.primary : palette.subtle, in: RoundedRectangle(cornerRadius: 8))
                }
                .buttonStyle(.plain)
                .opacity(isPending ? 0.78 : 1)
                .accessibilityAddTraits(isActive ? .isSelected : [])
            }
        }
    }

    private func statItem(_ value: String, _ label: String, color: Color) -> some View {
        VStack(spacing: 2) {
            Text(value)
                .font(.system(size: 20, weight: .bold).monospacedDigit())
                .foregroundStyle(color)
            Text(label)
                .font(.system(size: 12))
                .foregroundStyle(palette.textMuted)
        }
        .frame(maxWidth: .infinity)
        .padding(.vertical, 8)
        .accessibilityElement(children: .combine)
    }

    private var winRateBar: some View {
        GeometryReader { proxy in
            ZStack(alignment: .leading) {
                Capsule().fill(AppTheme.danger.opacity(0.18))
                Capsule()
                    .fill(palette.success)
                    .frame(width: proxy.size.width * CGFloat(min(100, max(0, stats?.winRate ?? 0))) / 100)
            }
        }
        .frame(height: 8)
        .accessibilityHidden(true)
    }

    // MARK: - 表现分析入口

    private var performanceEntry: some View {
        Button {
            navigator.open(.performance(gameType: model.currentGameType))
        } label: {
            HStack(spacing: 10) {
                VStack(alignment: .leading, spacing: 4) {
                    Text("表现分析")
                        .font(.system(size: 15, weight: .bold))
                        .foregroundStyle(palette.textPrimary)
                    Text("按对手实力与赛制长度，看清你近 90 天真实打出的水平")
                        .font(.system(size: 12))
                        .foregroundStyle(palette.textSecondary)
                        .multilineTextAlignment(.leading)
                }
                Spacer(minLength: 4)
                Image(systemName: "chevron.right")
                    .font(.system(size: 13, weight: .semibold))
                    .foregroundStyle(palette.textMuted)
            }
            .appCard(padding: 14)
        }
        .buttonStyle(.plain)
    }

    // MARK: - 近期趋势

    private var trendSection: some View {
        let calendar = model.trendCalendar

        return VStack(alignment: .leading, spacing: 12) {
            headerWithMonthSwitch(title: "近期趋势", monthTitle: calendar.title)
            Text("本月\(String(calendar.summary.matchCount))场，胜率\(String(calendar.summary.winRate))%")
                .font(.system(size: 13))
                .foregroundStyle(palette.textSecondary)

            LazyVGrid(columns: Array(repeating: GridItem(.flexible(), spacing: 4), count: 7), spacing: 4) {
                ForEach(calendar.weekdays, id: \.self) { weekday in
                    Text(weekday)
                        .font(.system(size: 11))
                        .foregroundStyle(palette.textMuted)
                }
                ForEach(calendar.cells) { cell in
                    trendCell(cell)
                }
            }

            HStack(spacing: 12) {
                legendDot(color: palette.success, text: "胜")
                legendDot(color: AppTheme.danger, text: "负")
            }
        }
        .appCard(padding: 14)
    }

    private func trendCell(_ cell: StatsTrendCell) -> some View {
        VStack(alignment: .leading, spacing: 2) {
            Text(String(cell.day))
                .font(.system(size: 10))
                .foregroundStyle(palette.textSecondary)
            if cell.isCurrentMonth && cell.hasMatches {
                if cell.winCount > 0 {
                    resultRow(color: palette.success, count: cell.winCount)
                }
                if cell.lossCount > 0 {
                    resultRow(color: AppTheme.danger, count: cell.lossCount)
                }
            }
        }
        .frame(maxWidth: .infinity, minHeight: 40, alignment: .topLeading)
        .padding(3)
        .background(
            cell.hasMatches ? palette.brandTint : palette.subtle,
            in: RoundedRectangle(cornerRadius: 5)
        )
        .opacity(cell.isCurrentMonth ? 1 : 0.35)
    }

    private func resultRow(color: Color, count: Int) -> some View {
        HStack(spacing: 3) {
            Circle()
                .fill(color)
                .frame(width: 5, height: 5)
            Text("×\(String(count))")
                .font(.system(size: 9))
                .foregroundStyle(palette.textSecondary)
        }
    }

    private func legendDot(color: Color, text: String) -> some View {
        HStack(spacing: 4) {
            RoundedRectangle(cornerRadius: 2)
                .fill(color)
                .frame(width: 8, height: 8)
            Text(text)
                .font(.system(size: 11))
                .foregroundStyle(palette.textMuted)
        }
    }

    // MARK: - 段位分变化

    private var rankSection: some View {
        let chart = model.rankDeltaChart
        let trend = model.rankTrend

        return VStack(alignment: .leading, spacing: 12) {
            headerWithMonthSwitch(title: "段位分变化", monthTitle: chart.title)

            HStack(alignment: .center, spacing: 16) {
                VStack(spacing: 2) {
                    Text(String(trend.currentScore))
                        .font(.system(size: 28, weight: .heavy).monospacedDigit())
                        .foregroundStyle(palette.brandStrong)
                    Text("当前分")
                        .font(.system(size: 12))
                        .foregroundStyle(palette.textMuted)
                }
                Spacer(minLength: 8)
                rankMeta(String(trend.peakScore), "最高分", color: palette.textPrimary)
                rankMeta(netDeltaText(chart.summary.netDelta), "本月净变", color: netDeltaColor(chart.summary.netDelta))
            }

            if chart.summary.upDays + chart.summary.downDays > 0 {
                rankChart(chart)
            } else {
                Text("暂无段位变化记录")
                    .font(.system(size: 13))
                    .foregroundStyle(palette.textMuted)
                    .frame(maxWidth: .infinity)
                    .padding(.vertical, 16)
            }
        }
        .appCard(padding: 14)
    }

    private func rankMeta(_ value: String, _ label: String, color: Color) -> some View {
        VStack(spacing: 2) {
            Text(value)
                .font(.system(size: 16, weight: .semibold).monospacedDigit())
                .foregroundStyle(color)
            Text(label)
                .font(.system(size: 11))
                .foregroundStyle(palette.textMuted)
        }
    }

    private func netDeltaText(_ delta: Int) -> String {
        delta > 0 ? "+\(String(delta))" : String(delta)
    }

    private func netDeltaColor(_ delta: Int) -> Color {
        if delta > 0 { return palette.success }
        if delta < 0 { return AppTheme.danger }
        return palette.textPrimary
    }

    private func rankChart(_ chart: StatsRankDeltaChart) -> some View {
        let tickDays = chart.days.filter(\.showTick).map(\.day)
        let lastDay = chart.days.count

        return Chart {
            RuleMark(y: .value("零线", 0))
                .foregroundStyle(palette.border)
            ForEach(chart.days) { day in
                if day.hasChange {
                    BarMark(
                        x: .value("日", day.day),
                        y: .value("变化", day.delta)
                    )
                    .foregroundStyle(day.isUp ? palette.success : AppTheme.danger)
                    .cornerRadius(2)
                }
            }
        }
        .chartXScale(domain: 1...max(lastDay, 1))
        .chartYScale(domain: -chart.maxAbsDelta...chart.maxAbsDelta)
        .chartXAxis {
            AxisMarks(values: tickDays) { value in
                AxisValueLabel {
                    if let day = value.as(Int.self) {
                        Text("\(String(day))日")
                            .font(.system(size: 9))
                            .foregroundStyle(palette.textMuted)
                    }
                }
            }
        }
        .chartYAxis(.hidden)
        .chartLegend(.hidden)
        .frame(height: 170)
        .accessibilityLabel("段位分变化柱状图")
    }

    // MARK: - 单杆最高分 / 对局时长 / 强弱对手

    private var highScoreSection: some View {
        let highScore = model.highScore

        return VStack(alignment: .leading, spacing: 12) {
            sectionTitle("单杆最高分")
            HStack(alignment: .center, spacing: 24) {
                VStack(spacing: 2) {
                    Text(String(highScore.personalBest))
                        .font(.system(size: 32, weight: .heavy).monospacedDigit())
                        .foregroundStyle(AppTheme.primary)
                    Text("个人最佳")
                        .font(.system(size: 12))
                        .foregroundStyle(palette.textMuted)
                }
                Text("本月最佳：\(String(highScore.monthBest))")
                    .font(.system(size: 13))
                    .foregroundStyle(palette.textSecondary)
                Spacer(minLength: 0)
            }
        }
        .appCard(padding: 14)
    }

    private var durationSection: some View {
        let duration = model.duration

        return VStack(alignment: .leading, spacing: 12) {
            sectionTitle("对局时长")
            HStack(spacing: 0) {
                durationItem(StatsDetailRules.formatDuration(seconds: duration.average), "平均", color: palette.textPrimary)
                durationItem(StatsDetailRules.formatDuration(seconds: duration.fastest), "最快", color: palette.success)
                durationItem(StatsDetailRules.formatDuration(seconds: duration.longest), "最长", color: AppTheme.primary)
            }
        }
        .appCard(padding: 14)
    }

    private func durationItem(_ value: String, _ label: String, color: Color) -> some View {
        VStack(spacing: 2) {
            Text(value)
                .font(.system(size: 18, weight: .bold).monospacedDigit())
                .foregroundStyle(color)
                .lineLimit(1)
                .minimumScaleFactor(0.8)
            Text(label)
                .font(.system(size: 12))
                .foregroundStyle(palette.textMuted)
        }
        .frame(maxWidth: .infinity)
        .accessibilityElement(children: .combine)
    }

    private var opponentSection: some View {
        let rows = model.opponentRows

        return VStack(alignment: .leading, spacing: 12) {
            sectionTitle("强弱对手分析")
            if rows.isEmpty {
                Text("暂无对手分析数据")
                    .font(.system(size: 13))
                    .foregroundStyle(palette.textMuted)
                    .frame(maxWidth: .infinity)
                    .padding(.vertical, 16)
            } else {
                VStack(spacing: 10) {
                    ForEach(rows) { row in
                        opponentRow(row)
                    }
                }
            }
        }
        .appCard(padding: 14)
    }

    private func opponentRow(_ row: StatsOpponentStrengthRow) -> some View {
        let isGood = row.winRate >= 50
        let color = isGood ? palette.success : AppTheme.danger

        return HStack(spacing: 8) {
            Text(row.tierName)
                .font(.system(size: 12))
                .foregroundStyle(palette.textSecondary)
                .lineLimit(1)
                .frame(width: 88, alignment: .leading)
            GeometryReader { proxy in
                ZStack(alignment: .leading) {
                    Capsule().fill(palette.subtle)
                    Capsule()
                        .fill(color)
                        .frame(width: proxy.size.width * CGFloat(min(100, max(0, row.winRate))) / 100)
                }
            }
            .frame(height: 10)
            Text("\(StatsDetailRules.formatPercent(row.winRate))%")
                .font(.system(size: 12, weight: .semibold).monospacedDigit())
                .foregroundStyle(color)
                .frame(width: 54, alignment: .trailing)
        }
        .accessibilityElement(children: .combine)
    }

    // MARK: - 公共

    private func sectionTitle(_ title: String) -> some View {
        Text(title)
            .font(.system(size: 15, weight: .bold))
            .foregroundStyle(palette.textPrimary)
    }

    private func headerWithMonthSwitch(title: String, monthTitle: String) -> some View {
        HStack(spacing: 8) {
            sectionTitle(title)
            Spacer(minLength: 8)
            monthSwitch(title: monthTitle)
        }
    }

    private func monthSwitch(title: String) -> some View {
        HStack(spacing: 6) {
            monthButton(systemName: "chevron.left", label: "上个月", offset: -1)
            Text(title)
                .font(.system(size: 12))
                .foregroundStyle(palette.textSecondary)
                .frame(minWidth: 66)
                .monospacedDigit()
            monthButton(systemName: "chevron.right", label: "下个月", offset: 1)
        }
    }

    private func monthButton(systemName: String, label: String, offset: Int) -> some View {
        Button {
            model.shiftMonth(by: offset)
        } label: {
            Image(systemName: systemName)
                .font(.system(size: 12, weight: .bold))
                .foregroundStyle(palette.textSecondary)
                .frame(width: 26, height: 26)
                .background(palette.subtle, in: Circle())
        }
        .buttonStyle(.plain)
        .accessibilityLabel(label)
    }
}
