import Charts
import SwiftUI

/// 表现分析，对应小程序 `subPages/user/performance.vue`。
struct PerformanceView: View {
    @EnvironmentObject private var session: SessionStore
    @EnvironmentObject private var navigator: AppNavigator
    @Environment(\.colorScheme) private var colorScheme
    @Environment(\.dismiss) private var dismiss
    @StateObject private var model: PerformanceModel

    init(session: SessionStore, gameType: Int, model: PerformanceModel? = nil) {
        _model = StateObject(wrappedValue: model ?? PerformanceModel(session: session, gameType: gameType))
    }

    private var palette: AppPalette { AppTheme.palette(for: colorScheme) }

    var body: some View {
        Group {
            if session.identity == nil {
                StateMessageView(
                    icon: "lock.fill",
                    title: "登录后查看表现分析",
                    actionTitle: "去登录"
                ) {
                    navigator.openLogin()
                }
            } else {
                VStack(spacing: 0) {
                    gameTypeTabs
                    content
                }
            }
        }
        .background(palette.page.ignoresSafeArea())
        .navigationTitle("表现分析")
        .navigationBarTitleDisplayMode(.inline)
        .task { await model.loadIfNeeded() }
        .onChange(of: session.identity) { _, _ in
            Task { await model.reloadForIdentityChange() }
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
                        .foregroundStyle(isActive ? Color.white : palette.textSecondary)
                        .lineLimit(1)
                        .minimumScaleFactor(0.8)
                        .frame(maxWidth: .infinity)
                        .frame(height: 34)
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
        }
        .padding(.horizontal, 12)
        .padding(.top, 12)
        .padding(.bottom, 4)
    }

    @ViewBuilder
    private var content: some View {
        if model.isInitialLoading {
            VStack(spacing: 10) {
                ProgressView()
                Text("加载表现分析...")
                    .font(.system(size: 14))
                    .foregroundStyle(palette.textSecondary)
            }
            .frame(maxWidth: .infinity, maxHeight: .infinity)
        } else if model.isInitialError {
            errorState
        } else if model.data != nil {
            page
        } else {
            VStack(spacing: 10) {
                ProgressView()
                Text("加载表现分析...")
                    .font(.system(size: 14))
                    .foregroundStyle(palette.textSecondary)
            }
            .frame(maxWidth: .infinity, maxHeight: .infinity)
        }
    }

    private var page: some View {
        ScrollView {
            VStack(spacing: 12) {
                if let message = model.refreshMessage {
                    refreshBanner(message)
                }
                if let locked = model.lockedView {
                    lockedContent(locked)
                } else {
                    memberContent
                }
                Text(PerformanceRules.note)
                    .font(.system(size: 11))
                    .foregroundStyle(palette.textMuted)
                    .multilineTextAlignment(.center)
                    .padding(.horizontal, 8)
            }
            .padding(.horizontal, 12)
            .padding(.top, 6)
            .padding(.bottom, 24)
        }
    }

    private var errorState: some View {
        let feedback = model.failureFeedback
            ?? LoadFailure(category: .business, message: "").feedback(resource: "表现分析")

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
        .background(AppTheme.danger.opacity(0.1), in: RoundedRectangle(cornerRadius: 10))
    }

    // MARK: - 锁定态

    @ViewBuilder
    private func lockedContent(_ locked: PerformanceLockedView) -> some View {
        VStack(alignment: .leading, spacing: 10) {
            Text("会员权益")
                .font(.system(size: 11, weight: .semibold))
                .tracking(1)
                .foregroundStyle(palette.brandStrong)
            Text("看清你真实打出的水平")
                .font(.system(size: 20, weight: .bold))
                .foregroundStyle(palette.textPrimary)
            Text("表现分按你每场的对手实力与赛制长度，反推出这段时间的实际水平。")
                .font(.system(size: 12))
                .foregroundStyle(palette.textSecondary)
                .fixedSize(horizontal: false, vertical: true)
            Text(locked.hasSamples
                 ? "近 90 天已有 \(locked.sampleText) 场可分析的排位"
                 : "完成排位对局后开始积累分析样本")
                .font(.system(size: 12, weight: .semibold))
                .foregroundStyle(palette.brandStrong)
                .padding(.top, 2)
        }
        .appCard(padding: 16)

        VStack(alignment: .leading, spacing: 12) {
            cardTitle("\(locked.preview.monthText)表现分预览")
            HStack(alignment: .firstTextBaseline, spacing: 3) {
                if !locked.preview.qualifier.isEmpty {
                    Text(locked.preview.qualifier)
                        .font(.system(size: 13, weight: .semibold))
                        .foregroundStyle(palette.textSecondary)
                }
                Text(locked.preview.available ? locked.preview.value : "—")
                    .font(.system(size: 40, weight: .heavy).monospacedDigit())
                    .foregroundStyle(locked.preview.available ? palette.textPrimary : palette.textMuted)
            }
            Text(locked.preview.hint)
                .font(.system(size: 12))
                .foregroundStyle(palette.textSecondary)
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .appCard(padding: 16)

        VStack(alignment: .leading, spacing: 12) {
            cardTitle("完整分析包含")
            ForEach(locked.features, id: \.self) { feature in
                HStack(spacing: 8) {
                    Image(systemName: "checkmark")
                        .font(.system(size: 12, weight: .bold))
                        .foregroundStyle(AppTheme.primary)
                    Text(feature)
                        .font(.system(size: 13))
                        .foregroundStyle(palette.textPrimary)
                }
            }
            if locked.showPurchase {
                Button("开通会员") {
                    navigator.open(.memberCenter)
                }
                .buttonStyle(.appPill)
                .frame(maxWidth: .infinity)
                .padding(.top, 4)
            } else {
                Text("完整分析为会员权益")
                    .font(.system(size: 12))
                    .foregroundStyle(palette.textMuted)
                    .padding(.top, 2)
            }
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .appCard(padding: 16)
    }

    // MARK: - 会员态

    @ViewBuilder
    private var memberContent: some View {
        summaryCard
        if !model.notableViews.isEmpty {
            notableCard
        }
        chartCard
    }

    private var summaryCard: some View {
        let summary = model.summaryView

        return VStack(alignment: .leading, spacing: 12) {
            cardTitle("近 90 天表现分")
            HStack(alignment: .firstTextBaseline, spacing: 3) {
                if summary.rating.available, !summary.rating.qualifier.isEmpty {
                    Text(summary.rating.qualifier)
                        .font(.system(size: 13, weight: .semibold))
                        .foregroundStyle(palette.textSecondary)
                }
                Text(summary.rating.available ? summary.rating.value : "—")
                    .font(.system(size: 40, weight: .heavy).monospacedDigit())
                    .foregroundStyle(summary.rating.available ? palette.textPrimary : palette.textMuted)
            }
            if !summary.ratingHint.isEmpty {
                Text(summary.ratingHint)
                    .font(.system(size: 12))
                    .foregroundStyle(palette.textSecondary)
            }

            if summary.available {
                HStack(spacing: 0) {
                    summaryCell(String(summary.wins), "实际胜场", color: palette.textPrimary)
                    summaryCell(summary.expectedText, "期望胜场", color: palette.textPrimary)
                    summaryCell(summary.diffText, "差值", color: toneColor(summary.tone))
                }
                Text("\(summary.headline)（\(summary.sampleText) 场样本）")
                    .font(.system(size: 12))
                    .foregroundStyle(palette.textSecondary)
                    .fixedSize(horizontal: false, vertical: true)
            } else {
                Text("近 90 天还没有可分析的排位样本")
                    .font(.system(size: 12))
                    .foregroundStyle(palette.textSecondary)
            }
        }
        .appCard(padding: 16)
    }

    private func summaryCell(_ value: String, _ label: String, color: Color) -> some View {
        VStack(spacing: 2) {
            Text(value)
                .font(.system(size: 18, weight: .bold).monospacedDigit())
                .foregroundStyle(color)
            Text(label)
                .font(.system(size: 11))
                .foregroundStyle(palette.textMuted)
        }
        .frame(maxWidth: .infinity)
        .accessibilityElement(children: .combine)
    }

    private func toneColor(_ tone: PerformanceTone) -> Color {
        switch tone {
        case .up: palette.success
        case .down: AppTheme.danger
        case .flat: palette.textSecondary
        }
    }

    private var notableCard: some View {
        VStack(alignment: .leading, spacing: 12) {
            cardTitle("值得回看的两场")
            VStack(spacing: 10) {
                ForEach(model.notableViews) { item in
                    Button {
                        navigator.open(.matchDetail(matchId: item.matchId, source: nil, perspectiveUserId: nil))
                    } label: {
                        HStack(spacing: 10) {
                            VStack(alignment: .leading, spacing: 3) {
                                Text(item.title)
                                    .font(.system(size: 13, weight: .bold))
                                    .foregroundStyle(item.kind == .bestWin ? palette.success : AppTheme.danger)
                                Text(item.headline)
                                    .font(.system(size: 13, weight: .semibold))
                                    .foregroundStyle(palette.textPrimary)
                                Text(item.detail)
                                    .font(.system(size: 12))
                                    .foregroundStyle(palette.textSecondary)
                                    .fixedSize(horizontal: false, vertical: true)
                                if !item.playedAt.isEmpty {
                                    Text(AppFormat.dateTime(item.playedAt))
                                        .font(.system(size: 11))
                                        .foregroundStyle(palette.textMuted)
                                }
                            }
                            Spacer(minLength: 4)
                            Image(systemName: "chevron.right")
                                .font(.system(size: 12, weight: .semibold))
                                .foregroundStyle(palette.textMuted)
                        }
                    }
                    .buttonStyle(.plain)
                }
            }
        }
        .appCard(padding: 16)
    }

    private var chartCard: some View {
        let chart = model.chart

        return VStack(alignment: .leading, spacing: 12) {
            cardTitle("最近 6 个月实力曲线")
            if chart.hasData {
                monthChart(chart)
                Text(PerformanceRules.monthFootnote)
                    .font(.system(size: 11))
                    .foregroundStyle(palette.textMuted)
                    .fixedSize(horizontal: false, vertical: true)
                HStack(spacing: 16) {
                    legendSwatch(color: palette.brandStrong, text: "当月表现分", isBar: true)
                    legendSwatch(color: palette.info, text: "月末排位分", isBar: false)
                }
            } else {
                Text("最近 6 个月还没有可画出曲线的排位记录")
                    .font(.system(size: 13))
                    .foregroundStyle(palette.textMuted)
                    .frame(maxWidth: .infinity)
                    .padding(.vertical, 16)
            }
        }
        .appCard(padding: 16)
    }

    private func monthChart(_ chart: PerformanceMonthChart) -> some View {
        VStack(spacing: 6) {
            Chart {
                ForEach(chart.bars) { bar in
                    if let rating = bar.ratingValue {
                        RectangleMark(
                            x: .value("月份", bar.label),
                            yStart: .value("下限", chart.scaleMin),
                            yEnd: .value("表现分", rating)
                        )
                        .foregroundStyle(bar.isCurrent ? AppTheme.primary : palette.brandStrong)
                        .cornerRadius(3)
                    }
                    if bar.dotPercent != nil, bar.endRankScore > 0 {
                        PointMark(
                            x: .value("月份", bar.label),
                            y: .value("月末排位分", bar.endRankScore)
                        )
                        .foregroundStyle(palette.info)
                        .symbolSize(46)
                    }
                }
            }
            .chartYScale(domain: chart.scaleMin...chart.scaleMax)
            .chartXAxis {
                AxisMarks { value in
                    AxisValueLabel {
                        if let label = value.as(String.self) {
                            Text(label)
                                .font(.system(size: 10))
                                .foregroundStyle(palette.textMuted)
                        }
                    }
                }
            }
            .chartYAxis(.hidden)
            .chartLegend(.hidden)
            .frame(height: 170)
            .accessibilityLabel("最近 6 个月表现分与月末排位分")

            HStack(alignment: .top, spacing: 4) {
                ForEach(chart.bars) { bar in
                    VStack(spacing: 2) {
                        Text(bar.isCurrent ? "本月" : bar.label)
                            .font(.system(size: 10))
                            .foregroundStyle(palette.textMuted)
                        Text(bar.ratingText)
                            .font(.system(size: 11, weight: .semibold).monospacedDigit())
                            .foregroundStyle(bar.status == .ready ? palette.textPrimary : palette.textMuted)
                            .lineLimit(1)
                        if !bar.note.isEmpty {
                            Text(bar.note)
                                .font(.system(size: 9))
                                .foregroundStyle(palette.textMuted)
                        }
                    }
                    .frame(maxWidth: .infinity)
                }
            }
        }
    }

    private func legendSwatch(color: Color, text: String, isBar: Bool) -> some View {
        HStack(spacing: 5) {
            if isBar {
                RoundedRectangle(cornerRadius: 2)
                    .fill(color)
                    .frame(width: 8, height: 10)
            } else {
                Circle()
                    .fill(color)
                    .frame(width: 7, height: 7)
            }
            Text(text)
                .font(.system(size: 11))
                .foregroundStyle(palette.textMuted)
        }
    }

    private func cardTitle(_ text: String) -> some View {
        Text(text)
            .font(.system(size: 15, weight: .bold))
            .foregroundStyle(palette.textPrimary)
    }
}
