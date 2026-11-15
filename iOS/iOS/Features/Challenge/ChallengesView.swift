import SwiftUI

/// 约球列表，对应小程序 `subPages/social/challenges.vue`。
struct ChallengesView: View {
    @EnvironmentObject private var session: SessionStore
    @EnvironmentObject private var navigator: AppNavigator
    @EnvironmentObject private var toast: ToastCenter
    @Environment(\.colorScheme) private var colorScheme

    @StateObject private var model: ChallengesModel

    init(
        session: SessionStore,
        activity: ActivityStore,
        initialTab: ChallengeTab,
        model: ChallengesModel? = nil
    ) {
        _model = StateObject(wrappedValue: model ?? ChallengesModel(
            session: session,
            activity: activity,
            initialTab: initialTab
        ))
    }

    private var palette: AppPalette { AppTheme.palette(for: colorScheme) }

    var body: some View {
        Group {
            if session.identity == nil {
                StateMessageView(
                    icon: "lock.fill",
                    title: "登录后查看约球",
                    actionTitle: "去登录"
                ) {
                    navigator.openLogin()
                }
            } else {
                content
            }
        }
        .background(palette.page.ignoresSafeArea())
        .navigationTitle("约球")
        .navigationBarTitleDisplayMode(.inline)
        .task { await model.onAppear() }
        .onChange(of: session.identity) { _, _ in
            Task { await model.onAppear() }
        }
        .onChange(of: model.feedback) { _, value in
            guard let value else { return }
            toast.show(value)
            model.feedback = nil
        }
        .onChange(of: model.navigation) { _, value in
            guard let value else { return }
            model.navigation = nil
            switch value {
            case .waiting(let challengeId):
                navigator.open(.challengeWaiting(challengeId: challengeId))
            case .match(let matchId):
                navigator.open(.liveMatch(matchId: matchId))
            case .compose(let target):
                navigator.open(.challengeCompose(target: target))
            }
        }
        .alert(item: $model.prompt) { prompt in
            alert(for: prompt)
        }
    }

    private var content: some View {
        ScrollView {
            VStack(spacing: 12) {
                tip
                tabBar
                if let message = model.refreshError {
                    refreshBanner(message)
                }
                content(for: model.phase)
            }
            .padding(.horizontal, 16)
            .padding(.vertical, 14)
        }
    }
    private var tip: some View {
        HStack(alignment: .top, spacing: 8) {
            Image(systemName: "info.circle.fill")
                .font(.system(size: 14))
                .foregroundStyle(AppTheme.primary)
            Text("约好后双方各点一次进入即正式开局；未打成的约球不会产生比赛。")
                .font(.system(size: 12))
                .foregroundStyle(palette.textSecondary)
                .fixedSize(horizontal: false, vertical: true)
            Spacer(minLength: 0)
        }
        .padding(12)
        .background(palette.brandTint, in: RoundedRectangle(cornerRadius: 12))
    }

    private var tabBar: some View {
        HStack(spacing: 6) {
            tabButton("收到", tab: .received) {
                badge(model.receivedCount)
            }
            tabButton("发出", tab: .sent)
            tabButton("约球记录", tab: .history)
        }
        .padding(5)
        .background(palette.card, in: RoundedRectangle(cornerRadius: 14))
    }

    private func tabButton<Accessory: View>(
        _ title: String,
        tab: ChallengeTab,
        @ViewBuilder accessory: () -> Accessory = { EmptyView() }
    ) -> some View {
        Button {
            Task { await model.switchTab(tab) }
        } label: {
            HStack(spacing: 4) {
                Text(title)
                    .font(.system(size: 13, weight: model.tab == tab ? .semibold : .regular))
                    .foregroundStyle(model.tab == tab ? palette.brandStrong : palette.textMuted)
                accessory()
            }
            .frame(maxWidth: .infinity)
            .frame(height: 36)
            .background(
                model.tab == tab ? palette.brandTint : Color.clear,
                in: RoundedRectangle(cornerRadius: 10)
            )
            .contentShape(Rectangle())
        }
        .buttonStyle(.plain)
    }

    @ViewBuilder
    private func badge(_ count: Int) -> some View {
        if count > 0 {
            Text(String(count))
                .font(.system(size: 10, weight: .semibold))
                .foregroundStyle(.white)
                .padding(.horizontal, 5)
                .frame(minWidth: 16, minHeight: 16)
                .background(AppTheme.danger, in: Capsule())
        }
    }

    @ViewBuilder
    private func content(for phase: ChallengesModel.Phase) -> some View {
        switch phase {
        case .loading:
            VStack(spacing: 10) {
                ProgressView()
                Text("加载中...")
                    .font(.system(size: 13))
                    .foregroundStyle(palette.textMuted)
            }
            .frame(maxWidth: .infinity)
            .padding(.top, 90)
        case .failed(let failure):
            let feedback = failure.feedback(resource: "约球")
            VStack(spacing: 10) {
                Text(feedback.title)
                    .font(.system(size: 16, weight: .semibold))
                    .foregroundStyle(palette.textPrimary)
                Text(feedback.description)
                    .font(.system(size: 13))
                    .foregroundStyle(palette.textSecondary)
                    .multilineTextAlignment(.center)
                if feedback.retryable {
                    Button("重试") {
                        Task { await model.retry() }
                    }
                    .buttonStyle(.appPill)
                    .padding(.top, 4)
                }
            }
            .frame(maxWidth: .infinity)
            .padding(.top, 90)
        case .empty:
            VStack(spacing: 10) {
                Text("🎱")
                    .font(.system(size: 34))
                    .accessibilityHidden(true)
                Text(model.emptyText)
                    .font(.system(size: 13))
                    .foregroundStyle(palette.textMuted)
            }
            .frame(maxWidth: .infinity)
            .padding(.top, 90)
        case .loaded:
            LazyVStack(spacing: 12) {
                ForEach(model.items) { item in
                    card(item)
                }
                if model.tab == .history {
                    historyFooter
                }
            }
        }
    }

    private func card(_ item: ChallengeInfo) -> some View {
        let opponent = model.opponent(item)
        return VStack(alignment: .leading, spacing: 10) {
            Button {
                model.openDetail(item)
            } label: {
                HStack(spacing: 12) {
                    MemberAvatarView(
                        avatar: opponent.avatar,
                        userId: opponent.opponentId,
                        vipLevel: vipLevel(item),
                        size: 44,
                        reserveFrame: true
                    )
                    VStack(alignment: .leading, spacing: 3) {
                        Text(ChallengeRules.displayName(opponent.nickname))
                            .font(.system(size: 15, weight: .medium))
                            .foregroundStyle(palette.textPrimary)
                            .lineLimit(1)
                        Text("\(GameTypeCatalog.label(for: item.gameType)) · \(ChallengeRules.matchModeLabel(item.matchMode))")
                            .font(.system(size: 12))
                            .foregroundStyle(palette.textMuted)
                        Text(ChallengeRules.schedule(item, serverTime: model.serverTime))
                            .font(.system(size: 12))
                            .foregroundStyle(palette.textMuted)
                    }
                    Spacer(minLength: 6)
                    statusPill(item)
                }
                .contentShape(Rectangle())
            }
            .buttonStyle(.plain)

            if !item.message.isEmpty {
                Text("\"\(item.message)\"")
                    .font(.system(size: 12))
                    .foregroundStyle(palette.textSecondary)
                    .fixedSize(horizontal: false, vertical: true)
                    .padding(.horizontal, 10)
                    .padding(.vertical, 8)
                    .frame(maxWidth: .infinity, alignment: .leading)
                    .background(palette.subtle, in: RoundedRectangle(cornerRadius: 8))
            }

            actions(item)
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .appCard(padding: 14)
    }

    private func vipLevel(_ item: ChallengeInfo) -> Int {
        model.opponent(item).direction == .sent ? (item.toVipLevel ?? 0) : (item.fromVipLevel ?? 0)
    }

    private func statusPill(_ item: ChallengeInfo) -> some View {
        Text(model.statusLabel(item))
            .font(.system(size: 11))
            .foregroundStyle(statusColor(item))
            .padding(.horizontal, 10)
            .frame(height: 24)
            .background(statusColor(item).opacity(0.12), in: Capsule())
    }

    private func statusColor(_ item: ChallengeInfo) -> Color {
        switch item.status {
        case 0: return palette.brandStrong
        case 1, 6: return palette.info
        default: return palette.textSecondary
        }
    }

    @ViewBuilder
    private func actions(_ item: ChallengeInfo) -> some View {
        if model.tab == .received && item.status == 0 {
            HStack(spacing: 8) {
                actionButton("这次不了", primary: false) {
                    Task { await model.reject(item) }
                }
                actionButton("接受约球", primary: true) {
                    Task { await model.accept(item) }
                }
            }
        } else if ChallengeRules.isMineSentPending(item, userId: model.myUserId) {
            actionButton("取消约球", primary: false) {
                model.prompt = .cancel(item)
            }
        } else if item.status == 1 {
            HStack(spacing: 8) {
                actionButton("进入对局", primary: true) {
                    model.openWaiting(item)
                }
                actionButton("取消约球", primary: false) {
                    model.prompt = .cancel(item)
                }
            }
        } else if item.status == 6, (item.matchId ?? 0) > 0 {
            actionButton("查看对局", primary: true) {
                model.openMatch(item)
            }
        } else if ChallengeRules.isTerminal(item) {
            actionButton("再约一场", primary: false) {
                model.rematch(item)
            }
        }
    }

    private func actionButton(_ title: String, primary: Bool, action: @escaping () -> Void) -> some View {
        Button(action: action) {
            Text(title)
                .font(.system(size: 13, weight: primary ? .semibold : .regular))
                .foregroundStyle(primary ? Color.white : palette.textSecondary)
                .frame(maxWidth: .infinity)
                .frame(height: 34)
                .background(primary ? AppTheme.primary : palette.subtle, in: Capsule())
        }
        .buttonStyle(.plain)
        .disabled(model.isActing)
    }

    @ViewBuilder
    private var historyFooter: some View {
        if model.isLoadingMoreHistory {
            ProgressView()
                .padding(.vertical, 12)
        } else if let error = model.historyLoadMoreError {
            Button(error) {
                Task { await model.loadMoreHistory() }
            }
            .font(.system(size: 12))
            .foregroundStyle(palette.textMuted)
            .buttonStyle(.plain)
            .padding(.vertical, 12)
        } else if model.hasMoreHistory {
            Button("加载更多") {
                Task { await model.loadMoreHistory() }
            }
            .font(.system(size: 13))
            .foregroundStyle(palette.textSecondary)
            .buttonStyle(.plain)
            .padding(.vertical, 12)
        }
    }

    private func refreshBanner(_ message: String) -> some View {
        Button {
            Task { await model.retry() }
        } label: {
            HStack(spacing: 6) {
                Text(message)
                    .font(.system(size: 12))
                    .foregroundStyle(palette.textSecondary)
                    .lineLimit(1)
                Text("点击重试")
                    .font(.system(size: 12, weight: .semibold))
                    .foregroundStyle(palette.brandStrong)
                Spacer(minLength: 0)
            }
            .padding(.horizontal, 11)
            .padding(.vertical, 9)
            .background(palette.card.opacity(0.6), in: RoundedRectangle(cornerRadius: 9))
        }
        .buttonStyle(.plain)
    }

    private func alert(for prompt: ChallengesModel.Prompt) -> Alert {
        switch prompt {
        case .cancel(let item):
            return Alert(
                title: Text("取消本次约球？"),
                message: Text("取消后需要重新邀约才能再打。"),
                primaryButton: .destructive(Text("取消约球")) {
                    Task { await model.cancel(item) }
                },
                secondaryButton: .cancel(Text("暂不"))
            )
        case .closeOwn(let item):
            return Alert(
                title: Text("接受这个约球？"),
                message: Text("接受后会自动关闭你已发出的邀请。"),
                primaryButton: .default(Text("确认接受")) {
                    Task { await model.accept(item, confirmCloseOwn: true) }
                },
                secondaryButton: .cancel(Text("暂不接受"))
            )
        case .typeConflict(let message):
            return Alert(
                title: Text("类型不同，不能直接匹配"),
                message: Text(message),
                primaryButton: .default(Text("查看发出的邀请")) {
                    Task { await model.switchTab(.sent) }
                },
                secondaryButton: .cancel(Text("知道了"))
            )
        }
    }
}
