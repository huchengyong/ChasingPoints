import SwiftUI

/// 单场复盘（小程序 `matchReplay`）：会员权益页，锁定态只有局数与功能清单。
struct MatchReplayView: View {
    @EnvironmentObject private var session: SessionStore
    @EnvironmentObject private var navigator: AppNavigator
    @Environment(\.colorScheme) private var colorScheme
    @Environment(\.dismiss) private var dismiss
    @StateObject private var model: MatchReplayModel

    init(session: SessionStore, matchId: Int64, model: MatchReplayModel? = nil) {
        _model = StateObject(wrappedValue: model ?? MatchReplayModel(session: session, matchId: matchId))
    }

    private var palette: AppPalette { AppTheme.palette(for: colorScheme) }

    var body: some View {
        Group {
            if session.identity == nil {
                StateMessageView(
                    icon: "lock.fill",
                    title: "登录后查看单场复盘",
                    actionTitle: "去登录"
                ) {
                    navigator.openLogin()
                }
            } else {
                content
            }
        }
        .background(palette.page.ignoresSafeArea())
        .navigationTitle("单场复盘")
        .navigationBarTitleDisplayMode(.inline)
        .task { await model.load() }
        .onChange(of: session.identity) { _, _ in
            Task { await model.load() }
        }
    }

    @ViewBuilder
    private var content: some View {
        switch model.phase {
        case .loading:
            VStack(spacing: 10) {
                ProgressView()
                Text("加载单场复盘...")
                    .font(.system(size: 14))
                    .foregroundStyle(palette.textSecondary)
            }
            .frame(maxWidth: .infinity, maxHeight: .infinity)
        case .failed(let failure):
            errorState(failure)
        case .loaded(let view):
            page(view)
        }
    }

    private func errorState(_ failure: LoadFailure) -> some View {
        let feedback = failure.feedback(resource: "单场复盘")
        return StateMessageView(
            icon: "info.circle",
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

    private func page(_ view: ReplayView) -> some View {
        ScrollView {
            VStack(spacing: 14) {
                if let refreshFailure = model.refreshFailure {
                    refreshBanner(refreshFailure.feedback(resource: "单场复盘").description)
                }
                if view.locked {
                    lockedContent(view)
                } else {
                    replayContent(view)
                }
            }
            .padding(16)
            .padding(.bottom, 24)
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
    private func lockedContent(_ view: ReplayView) -> some View {
        VStack(alignment: .leading, spacing: 10) {
            Text("会员权益")
                .font(.system(size: 11, weight: .semibold))
                .tracking(1)
                .foregroundStyle(palette.brandStrong)
            Text("单场复盘")
                .font(.system(size: 22, weight: .bold))
                .foregroundStyle(palette.textPrimary)
            Text(view.wholeMatch
                 ? "回看这场九球追分的整场走势、双方得分构成与犯规。"
                 : "回看这场\(view.gameTypeLabel)每一局的走势、关键局和胜局方式。")
                .font(.system(size: 13))
                .foregroundStyle(palette.textSecondary)
                .fixedSize(horizontal: false, vertical: true)
            if !view.wholeMatch {
                Text("这场共打了 \(String(view.frameCount)) 局")
                    .font(.system(size: 12, weight: .semibold))
                    .foregroundStyle(palette.brandStrong)
                    .padding(.top, 2)
            }
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .appCard(padding: 16)

        VStack(alignment: .leading, spacing: 12) {
            CardTitle("完整复盘包含")
            ForEach(view.features ?? [], id: \.self) { feature in
                HStack(spacing: 8) {
                    Image(systemName: "checkmark")
                        .font(.system(size: 12, weight: .bold))
                        .foregroundStyle(AppTheme.primary)
                    Text(feature)
                        .font(.system(size: 13))
                        .foregroundStyle(palette.textPrimary)
                }
            }
            if view.showPurchase {
                Button("开通会员") {
                    navigator.open(.memberCenter)
                }
                .buttonStyle(.appPill)
                .frame(maxWidth: .infinity)
                .padding(.top, 4)
            } else {
                Text("完整复盘为会员权益")
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
    private func replayContent(_ view: ReplayView) -> some View {
        heroCard(view)

        if let notice = view.notice, !notice.isEmpty {
            Text(notice)
                .font(.system(size: 12))
                .foregroundStyle(palette.textSecondary)
                .frame(maxWidth: .infinity, alignment: .leading)
                .padding(10)
                .background(palette.brandTint, in: RoundedRectangle(cornerRadius: 8))
        }

        let keyFrames = view.keyFrames ?? []
        if !keyFrames.isEmpty {
            keyFramesCard(keyFrames)
        }

        let summaryRows = view.summaryRows ?? []
        if !summaryRows.isEmpty {
            VStack(alignment: .leading, spacing: 10) {
                CardTitle("整场数据")
                compareTable(rows: summaryRows, labelHead: "")
            }
            .frame(maxWidth: .infinity, alignment: .leading)
            .appCard(padding: 14)
        }

        let sequence = view.sequence ?? []
        if !sequence.isEmpty {
            sequenceCard(sequence, winTypeRows: view.winTypeRows ?? [])
        }

        let frames = view.frames ?? []
        if !frames.isEmpty {
            framesCard(frames, wholeMatch: view.wholeMatch)
        }

        if view.empty == true {
            Text(view.wholeMatch ? "这场对局没有可展示的得分走势" : "这场对局没有可展示的局记录")
                .font(.system(size: 12))
                .foregroundStyle(palette.textMuted)
                .frame(maxWidth: .infinity)
        }
    }

    private func heroCard(_ view: ReplayView) -> some View {
        VStack(alignment: .leading, spacing: 8) {
            Text(view.title ?? "")
                .font(.system(size: 12, weight: .semibold))
                .foregroundStyle(palette.brandStrong)
            Text(view.scoreText ?? "")
                .font(.system(size: 30, weight: .bold).monospacedDigit())
                .foregroundStyle(palette.textPrimary)
            Text(view.wholeMatch
                 ? "左为我，右为对手"
                 : "共 \(String(view.frameCount)) 局 · 左为我，右为对手")
                .font(.system(size: 12))
                .foregroundStyle(palette.textSecondary)
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .appCard(padding: 16)
    }

    private func keyFramesCard(_ frames: [ReplayFrameView]) -> some View {
        VStack(alignment: .leading, spacing: 12) {
            CardTitle("关键局")
            ForEach(Array(frames.enumerated()), id: \.offset) { _, frame in
                VStack(alignment: .leading, spacing: 5) {
                    HStack(spacing: 6) {
                        Text(frame.title)
                            .font(.system(size: 13, weight: .semibold))
                            .foregroundStyle(palette.textPrimary)
                        ForEach(Array(frame.tags.enumerated()), id: \.offset) { _, tag in
                            TagLabel(tag: tag)
                        }
                    }
                    Text(keyDetail(frame))
                        .font(.system(size: 12))
                        .foregroundStyle(palette.textSecondary)
                }
                .frame(maxWidth: .infinity, alignment: .leading)
            }
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .appCard(padding: 14)
    }

    private func keyDetail(_ frame: ReplayFrameView) -> String {
        var parts = [frame.winnerText]
        if !frame.scoreText.isEmpty { parts.append(frame.scoreText) }
        if !frame.deficitText.isEmpty { parts.append(frame.deficitText) }
        return parts.joined(separator: " · ")
    }

    private func sequenceCard(_ sequence: [ReplaySequenceItem], winTypeRows: [ReplaySummaryRow]) -> some View {
        VStack(alignment: .leading, spacing: 10) {
            CardTitle("逐局胜负")
            HStack(spacing: 6) {
                ForEach(Array(sequence.enumerated()), id: \.offset) { _, item in
                    Text(String(item.frameNo))
                        .font(.system(size: 12, weight: .semibold))
                        .foregroundStyle(item.winner == "me" ? Color.white : palette.textSecondary)
                        .frame(width: 28, height: 28)
                        .background(
                            item.winner == "me" ? AppTheme.primary : palette.subtle,
                            in: RoundedRectangle(cornerRadius: 7)
                        )
                }
            }
            Text("金色为我赢，灰色为对手赢")
                .font(.system(size: 11))
                .foregroundStyle(palette.textMuted)
            if !winTypeRows.isEmpty {
                compareTable(rows: winTypeRows, labelHead: "胜局方式")
                    .padding(.top, 2)
            }
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .appCard(padding: 14)
    }

    private func framesCard(_ frames: [ReplayFrameView], wholeMatch: Bool) -> some View {
        VStack(alignment: .leading, spacing: 14) {
            CardTitle(wholeMatch ? "得分走势" : "每局详情")
            ForEach(Array(frames.enumerated()), id: \.offset) { _, frame in
                frameBlock(frame)
            }
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .appCard(padding: 14)
    }

    private func frameBlock(_ frame: ReplayFrameView) -> some View {
        VStack(alignment: .leading, spacing: 7) {
            HStack(spacing: 6) {
                Text(frame.title)
                    .font(.system(size: 13, weight: .semibold))
                    .foregroundStyle(palette.textPrimary)
                Text(frame.winnerText)
                    .font(.system(size: 12, weight: .semibold))
                    .foregroundStyle(winnerColor(frame.winner))
                if !frame.scoreText.isEmpty {
                    Text(frame.scoreText)
                        .font(.system(size: 12, weight: .semibold).monospacedDigit())
                        .foregroundStyle(palette.textPrimary)
                }
                if !frame.winTypeText.isEmpty {
                    Text(frame.winTypeText)
                        .font(.system(size: 11))
                        .foregroundStyle(palette.brandStrong)
                }
                ForEach(Array(frame.tags.enumerated()), id: \.offset) { _, tag in
                    TagLabel(tag: tag)
                }
            }

            let meta = [frame.durationText, frame.deficitText].filter { !$0.isEmpty }
            if !meta.isEmpty {
                Text(meta.joined(separator: " · "))
                    .font(.system(size: 11))
                    .foregroundStyle(palette.textMuted)
            }

            if frame.trend.hasData {
                trendChart(frame.trend)
            }

            ForEach(Array(frame.details.enumerated()), id: \.offset) { _, detail in
                compareRow(label: detail.label, mine: detail.mine, opponent: detail.opponent, small: true)
            }
        }
        .frame(maxWidth: .infinity, alignment: .leading)
    }

    private func trendChart(_ trend: ReplayTrend) -> some View {
        VStack(spacing: 6) {
            HStack(alignment: .center, spacing: 2) {
                ForEach(Array(trend.columns.enumerated()), id: \.offset) { _, column in
                    VStack(spacing: 0) {
                        ZStack(alignment: .bottom) {
                            Color.clear
                            if column.direction == "up" {
                                Capsule()
                                    .fill(palette.brandStrong)
                                    .frame(height: barHeight(column.heightPercent))
                            }
                        }
                        .frame(height: 38)

                        Rectangle()
                            .fill(palette.border)
                            .frame(height: 1)

                        ZStack(alignment: .top) {
                            Color.clear
                            if column.direction == "down" {
                                Capsule()
                                    .fill(palette.textMuted)
                                    .frame(height: barHeight(column.heightPercent))
                            }
                        }
                        .frame(height: 38)
                    }
                    .frame(maxWidth: .infinity)
                }
            }
            .frame(height: 77)

            Text("向上为我领先，向下为对手领先（最大分差 \(String(trend.maxAbs))）")
                .font(.system(size: 11))
                .foregroundStyle(palette.textMuted)
        }
        .padding(.vertical, 4)
    }

    private func barHeight(_ percent: Double) -> CGFloat {
        max(2, CGFloat(percent) / 100 * 38)
    }

    // MARK: - 对比表

    private func compareTable(rows: [ReplaySummaryRow], labelHead: String) -> some View {
        VStack(spacing: 8) {
            HStack(spacing: 4) {
                Text("我")
                    .frame(width: 52, alignment: .leading)
                Text(labelHead)
                    .frame(maxWidth: .infinity)
                Text("对手")
                    .frame(width: 52, alignment: .trailing)
            }
            .font(.system(size: 11, weight: .semibold))
            .foregroundStyle(palette.textMuted)

            ForEach(Array(rows.enumerated()), id: \.offset) { _, row in
                compareRow(label: row.label, mine: row.mine, opponent: row.opponent, small: false)
            }
        }
    }

    private func compareRow(label: String, mine: String, opponent: String, small: Bool) -> some View {
        HStack(spacing: 4) {
            Text(mine)
                .font(.system(size: small ? 12 : 14, weight: .semibold).monospacedDigit())
                .foregroundStyle(palette.textPrimary)
                .frame(width: 52, alignment: .leading)
            Text(label)
                .font(.system(size: small ? 11 : 12))
                .foregroundStyle(palette.textSecondary)
                .frame(maxWidth: .infinity)
            Text(opponent)
                .font(.system(size: small ? 12 : 14, weight: .semibold).monospacedDigit())
                .foregroundStyle(palette.textPrimary)
                .frame(width: 52, alignment: .trailing)
        }
    }

    private func winnerColor(_ winner: String) -> Color {
        switch winner {
        case "me": palette.brandStrong
        case "opponent": palette.textSecondary
        default: palette.textMuted
        }
    }
}

private struct CardTitle: View {
    @Environment(\.colorScheme) private var colorScheme
    let text: String

    init(_ text: String) {
        self.text = text
    }

    var body: some View {
        Text(text)
            .font(.system(size: 16, weight: .semibold))
            .foregroundStyle(AppTheme.palette(for: colorScheme).textPrimary)
    }
}

private struct TagLabel: View {
    @Environment(\.colorScheme) private var colorScheme
    let tag: ReplayTag

    var body: some View {
        let palette = AppTheme.palette(for: colorScheme)
        Text(tag.label)
            .font(.system(size: 10, weight: .semibold))
            .foregroundStyle(palette.brandStrong)
            .padding(.horizontal, 6)
            .padding(.vertical, 2)
            .background(palette.brandTint, in: Capsule())
    }
}
