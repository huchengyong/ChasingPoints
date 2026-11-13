import SwiftUI

/// 信誉明细，对应小程序 `subPages/user/reputation.vue`。
struct ReputationView: View {
    @EnvironmentObject private var session: SessionStore
    @EnvironmentObject private var navigator: AppNavigator
    @Environment(\.colorScheme) private var colorScheme
    @Environment(\.dismiss) private var dismiss
    @StateObject private var model: ReputationModel

    init(session: SessionStore, model: ReputationModel? = nil) {
        _model = StateObject(wrappedValue: model ?? ReputationModel(session: session))
    }

    private var palette: AppPalette { AppTheme.palette(for: colorScheme) }

    var body: some View {
        Group {
            if session.identity == nil {
                StateMessageView(
                    icon: "lock.fill",
                    title: "请登录后查看信誉明细",
                    actionTitle: "去登录"
                ) {
                    navigator.openLogin()
                }
            } else {
                content
            }
        }
        .background(palette.page.ignoresSafeArea())
        .navigationTitle("信誉明细")
        .navigationBarTitleDisplayMode(.inline)
        .task { await model.onAppear() }
        .onChange(of: session.identity) { _, _ in
            Task { await model.onIdentityChanged() }
        }
    }

    private var content: some View {
        ScrollView {
            VStack(spacing: 12) {
                summaryCard
                logsSection
            }
            .padding(.horizontal, 12)
            .padding(.vertical, 12)
        }
        .refreshable { await model.refresh() }
    }

    private var summaryCard: some View {
        VStack(alignment: .leading, spacing: 10) {
            HStack(alignment: .top, spacing: 12) {
                VStack(alignment: .leading, spacing: 4) {
                    Text("当前信誉")
                        .font(.system(size: 13))
                        .foregroundStyle(palette.textSecondary)
                    Text(model.displayScore)
                        .font(.system(size: 38, weight: .heavy).monospacedDigit())
                        .foregroundStyle(palette.textPrimary)
                }

                Spacer(minLength: 8)

                Text(model.displayStatusText)
                    .font(.system(size: 13, weight: .semibold))
                    .foregroundStyle(statusTone)
                    .padding(.horizontal, 10)
                    .padding(.vertical, 5)
                    .background(statusTone.opacity(0.16), in: Capsule())
            }

            if let banUntil = model.banUntilText {
                Text("禁赛至 \(banUntil)")
                    .font(.system(size: 13, weight: .semibold))
                    .foregroundStyle(AppTheme.danger)
            }

            Text("异常比赛会扣减信誉，信誉会按规则恢复")
                .font(.system(size: 13))
                .foregroundStyle(palette.textSecondary)

            if let feedback = model.statusFeedback {
                HStack(alignment: .firstTextBaseline, spacing: 8) {
                    Text(feedback.description)
                        .font(.system(size: 13))
                        .foregroundStyle(palette.textSecondary)
                    Spacer(minLength: 8)
                    Button(feedback.actionText) {
                        if feedback.retryable {
                            Task { await model.retryStatus() }
                        } else {
                            dismiss()
                        }
                    }
                    .font(.system(size: 13, weight: .semibold))
                    .foregroundStyle(palette.brandStrong)
                }
            }
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .appCard()
    }

    private var statusTone: Color {
        if model.status.value == nil { return palette.textSecondary }
        return model.isRestricted ? AppTheme.danger : palette.success
    }

    private var logsSection: some View {
        VStack(alignment: .leading, spacing: 12) {
            HStack(alignment: .firstTextBaseline, spacing: 8) {
                VStack(alignment: .leading, spacing: 4) {
                    Text("信誉记录")
                        .font(.system(size: 18, weight: .semibold))
                        .foregroundStyle(palette.textPrimary)
                    Text("只展示你自己的信誉变更流水")
                        .font(.system(size: 13))
                        .foregroundStyle(palette.textSecondary)
                }
                Spacer(minLength: 8)
                if model.logs.total > 0 {
                    Text("共 \(String(model.logs.total)) 条")
                        .font(.system(size: 13))
                        .foregroundStyle(palette.textSecondary)
                }
            }

            logsContent
        }
        .frame(maxWidth: .infinity, alignment: .leading)
    }

    @ViewBuilder
    private var logsContent: some View {
        if model.logs.items.isEmpty {
            switch model.logs.phase {
            case .failed(let message):
                let feedback = model.logsFeedback(for: message)
                logStateCard(
                    icon: "exclamationmark.triangle",
                    title: feedback.title,
                    description: feedback.description,
                    actionTitle: feedback.actionText
                ) {
                    if feedback.retryable {
                        Task { await model.retryLogs() }
                    } else {
                        dismiss()
                    }
                }
            case .empty:
                logStateCard(icon: "list.bullet", title: "暂时还没有信誉变更记录", description: nil)
            case .idle, .loading, .loaded:
                VStack(spacing: 10) {
                    ProgressView()
                    Text("正在加载信誉记录")
                        .font(.system(size: 14))
                        .foregroundStyle(palette.textSecondary)
                }
                .frame(maxWidth: .infinity)
                .padding(.vertical, 28)
                .appCard()
            }
        } else {
            VStack(spacing: 10) {
                if let refreshError = model.logs.refreshError {
                    HStack(spacing: 8) {
                        Text(refreshError)
                            .font(.system(size: 13))
                            .foregroundStyle(palette.textSecondary)
                        Spacer(minLength: 8)
                        Button("重试") { Task { await model.retryLogs() } }
                            .font(.system(size: 13, weight: .semibold))
                            .foregroundStyle(palette.brandStrong)
                    }
                    .padding(.horizontal, 4)
                }

                ForEach(model.logs.items) { item in
                    logRow(item)
                        .onAppear {
                            if item.id == model.logs.items.last?.id {
                                Task { await model.logs.loadMore() }
                            }
                        }
                }

                footer
            }
        }
    }

    private func logRow(_ item: ReputationLog) -> some View {
        VStack(alignment: .leading, spacing: 8) {
            HStack(alignment: .firstTextBaseline, spacing: 8) {
                HStack(spacing: 8) {
                    Text(item.displayReason)
                        .font(.system(size: 15, weight: .semibold))
                        .foregroundStyle(palette.textPrimary)
                    Text(item.displayType)
                        .font(.system(size: 12))
                        .foregroundStyle(palette.textSecondary)
                        .padding(.horizontal, 8)
                        .padding(.vertical, 2)
                        .background(palette.subtle, in: RoundedRectangle(cornerRadius: 6))
                }
                Spacer(minLength: 8)
                Text(item.changeScoreText)
                    .font(.system(size: 18, weight: .bold).monospacedDigit())
                    .foregroundStyle(scoreColor(item.changeScore))
            }

            HStack(spacing: 8) {
                Text("变更前 \(String(item.beforeScore))")
                Text("→")
                Text("变更后 \(String(item.afterScore))")
            }
            .font(.system(size: 13))
            .foregroundStyle(palette.textSecondary)

            HStack(spacing: 8) {
                Text(logTime(item.createdAt))
                    .font(.system(size: 12))
                    .foregroundStyle(palette.textMuted)
                Spacer(minLength: 8)
                if item.isMatchLog, let matchId = item.matchId {
                    Text("对局ID \(String(matchId))")
                        .font(.system(size: 12))
                        .foregroundStyle(palette.textMuted)
                }
            }
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .appCard(padding: 14)
        .accessibilityElement(children: .combine)
    }

    private func scoreColor(_ score: Int) -> Color {
        if score > 0 { return palette.success }
        if score < 0 { return AppTheme.danger }
        return palette.textSecondary
    }

    private func logTime(_ text: String) -> String {
        text.isEmpty ? "--" : AppFormat.relativeTime(text)
    }

    @ViewBuilder
    private var footer: some View {
        if model.logs.isLoadingMore {
            Text("加载更多...")
                .font(.system(size: 13))
                .foregroundStyle(palette.textSecondary)
                .frame(maxWidth: .infinity)
                .padding(.vertical, 8)
        } else if !model.logs.hasMore {
            Text("没有更多了")
                .font(.system(size: 13))
                .foregroundStyle(palette.textSecondary)
                .frame(maxWidth: .infinity)
                .padding(.vertical, 8)
        }
    }

    private func logStateCard(
        icon: String,
        title: String,
        description: String?,
        actionTitle: String? = nil,
        action: (() -> Void)? = nil
    ) -> some View {
        VStack(spacing: 10) {
            Image(systemName: icon)
                .font(.system(size: 28))
                .foregroundStyle(palette.textMuted)
                .accessibilityHidden(true)
            Text(title)
                .font(.system(size: 15, weight: .semibold))
                .foregroundStyle(palette.textPrimary)
                .multilineTextAlignment(.center)
            if let description {
                Text(description)
                    .font(.system(size: 13))
                    .foregroundStyle(palette.textSecondary)
                    .multilineTextAlignment(.center)
            }
            if let actionTitle, let action {
                Button(actionTitle, action: action)
                    .buttonStyle(.appPill)
                    .padding(.top, 2)
            }
        }
        .padding(20)
        .frame(maxWidth: .infinity)
        .appCard(padding: 0)
    }
}
