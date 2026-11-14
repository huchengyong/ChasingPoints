import SwiftUI

/// 消息 Tab，对应小程序 `pages/message/index.vue`。
struct MessageCenterView: View {
    @EnvironmentObject private var session: SessionStore
    @EnvironmentObject private var navigator: AppNavigator
    @EnvironmentObject private var toast: ToastCenter
    @EnvironmentObject private var activity: ActivityStore
    @Environment(\.colorScheme) private var colorScheme
    @Environment(\.scenePhase) private var scenePhase

    @StateObject private var model: MessageCenterModel

    init(session: SessionStore, activity: ActivityStore, model: MessageCenterModel? = nil) {
        _model = StateObject(wrappedValue: model ?? MessageCenterModel(session: session, activity: activity))
    }

    private var palette: AppPalette { AppTheme.palette(for: colorScheme) }

    var body: some View {
        Group {
            if session.identity == nil {
                guestState
            } else {
                notificationList
            }
        }
        .background(palette.page.ignoresSafeArea())
        .navigationTitle("消息")
        .navigationBarTitleDisplayMode(.inline)
        .task { await model.onAppear() }
        .onChange(of: session.identity) { _, _ in
            Task { await model.identityChanged() }
        }
        .onChange(of: scenePhase) { _, phase in
            if phase == .active {
                Task { await model.onAppear() }
            }
        }
    }

    // MARK: - 未登录

    private var guestState: some View {
        VStack(spacing: 0) {
            Image(systemName: "bubble.left.and.bubble.right")
                .font(.system(size: 32))
                .foregroundStyle(palette.textMuted)
                .frame(width: 56, height: 56)
                .background(palette.brandTint, in: Circle())
                .accessibilityHidden(true)

            Text("登录后查看消息")
                .font(.system(size: 16, weight: .bold))
                .foregroundStyle(palette.textPrimary)
                .padding(.top, 14)

            Text("好友申请、约球邀请和比赛提醒都会在这里")
                .font(.system(size: 13))
                .foregroundStyle(palette.textSecondary)
                .multilineTextAlignment(.center)
                .padding(.top, 6)

            Button("登录 / 注册") {
                navigator.openLogin()
            }
            .buttonStyle(.appPill)
            .padding(.top, 20)
        }
        .padding(.horizontal, 24)
        .padding(.top, 100)
        .frame(maxWidth: .infinity, maxHeight: .infinity, alignment: .top)
    }

    // MARK: - 通知

    private var notificationList: some View {
        List {
            entryCard
                .listRowInsets(EdgeInsets(top: 12, leading: 16, bottom: 4, trailing: 16))
                .listRowSeparator(.hidden)
                .listRowBackground(Color.clear)

            topBar
                .listRowInsets(EdgeInsets(top: 8, leading: 20, bottom: 4, trailing: 20))
                .listRowSeparator(.hidden)
                .listRowBackground(Color.clear)

            if let refreshError = model.refreshError {
                refreshBanner(refreshError)
                    .listRowInsets(EdgeInsets(top: 4, leading: 16, bottom: 8, trailing: 16))
                    .listRowSeparator(.hidden)
                    .listRowBackground(Color.clear)
            }

            switch model.phase {
            case .idle, .loading:
                loadingRow
            case .failed:
                failureRow
            case .empty:
                emptyRow
            case .loaded:
                ForEach(model.items) { item in
                    notificationRow(item)
                        .listRowInsets(EdgeInsets(top: 6, leading: 16, bottom: 6, trailing: 16))
                        .listRowSeparator(.hidden)
                        .listRowBackground(Color.clear)
                        .swipeActions(edge: .trailing, allowsFullSwipe: true) {
                            Button(role: .destructive) {
                                delete(item)
                            } label: {
                                Label("删除", systemImage: "trash")
                            }
                        }
                }
                if model.hasMore {
                    loadMoreRow
                }
            }
        }
        .listStyle(.plain)
        .scrollContentBackground(.hidden)
        .refreshable { await model.refresh() }
    }

    private var entryCard: some View {
        VStack(spacing: 0) {
            ForEach(Array(model.entries.enumerated()), id: \.element.id) { index, entry in
                if index > 0 {
                    Divider().overlay(palette.border)
                }
                Button {
                    navigator.open(entry.route)
                } label: {
                    entryRow(entry)
                }
                .buttonStyle(.plain)
            }
        }
        .appCard(padding: 0)
    }

    private func entryRow(_ entry: MessageEntry) -> some View {
        HStack(spacing: 10) {
            Image(systemName: entry.symbol)
                .font(.system(size: 18))
                .foregroundStyle(palette.brandStrong)
                .frame(width: 36, height: 36)
                .background(palette.brandTint, in: Circle())
                .accessibilityHidden(true)

            VStack(alignment: .leading, spacing: 3) {
                Text(entry.title)
                    .font(.system(size: 15, weight: .semibold))
                    .foregroundStyle(palette.textPrimary)
                Text(entry.desc)
                    .font(.system(size: 12))
                    .foregroundStyle(palette.textMuted)
            }

            Spacer(minLength: 8)

            if !entry.countText.isEmpty {
                Text(entry.countText)
                    .font(.system(size: 11, weight: .semibold))
                    .foregroundStyle(.white)
                    .padding(.horizontal, 6)
                    .frame(minWidth: 20, minHeight: 20)
                    .background(AppTheme.danger, in: Capsule())
            }

            Image(systemName: "chevron.right")
                .font(.system(size: 12, weight: .semibold))
                .foregroundStyle(palette.textMuted)
                .accessibilityHidden(true)
        }
        .padding(.horizontal, 12)
        .padding(.vertical, 13)
        .contentShape(Rectangle())
    }

    private var topBar: some View {
        HStack {
            Text("全部消息")
                .font(.system(size: 13, weight: .semibold))
                .foregroundStyle(palette.textSecondary)
            Spacer()
            Button("全部已读") {
                Task {
                    if await model.markAllRead() {
                        toast.show("已全部标记为已读")
                    } else {
                        toast.show("操作失败")
                    }
                }
            }
            .font(.system(size: 13))
            .foregroundStyle(palette.brandStrong)
        }
    }

    private func refreshBanner(_ message: String) -> some View {
        HStack(spacing: 10) {
            Text(message)
                .font(.system(size: 12))
                .foregroundStyle(palette.textSecondary)
                .lineLimit(2)
            Spacer(minLength: 0)
            Button("重试") {
                Task { await model.retry() }
            }
            .font(.system(size: 12, weight: .semibold))
            .foregroundStyle(palette.brandStrong)
        }
        .padding(.horizontal, 11)
        .padding(.vertical, 9)
        .background(palette.card, in: RoundedRectangle(cornerRadius: 9))
        .overlay {
            RoundedRectangle(cornerRadius: 9).stroke(palette.border, lineWidth: 1)
        }
    }

    private func notificationRow(_ item: NotificationInfo) -> some View {
        let unread = !model.isRead(item)
        return Button {
            tap(item)
        } label: {
            HStack(alignment: .top, spacing: 8) {
                Text(typeIcon(item.type))
                    .font(.system(size: 14))
                    .frame(width: 32, height: 32)
                    .background(typeTint(item.type), in: Circle())
                    .accessibilityHidden(true)

                VStack(alignment: .leading, spacing: 4) {
                    HStack(spacing: 4) {
                        Text(item.title)
                            .font(.system(size: 14, weight: .semibold))
                            .foregroundStyle(palette.textPrimary)
                            .lineLimit(1)
                        if unread {
                            Circle()
                                .fill(AppTheme.primary)
                                .frame(width: 6, height: 6)
                                .accessibilityHidden(true)
                        }
                    }
                    Text(item.content)
                        .font(.system(size: 13))
                        .foregroundStyle(palette.textSecondary)
                        .lineLimit(1)
                    Text(item.createdAt)
                        .font(.system(size: 11))
                        .foregroundStyle(palette.textMuted)
                }

                Spacer(minLength: 0)
            }
            .padding(12)
            .frame(maxWidth: .infinity, alignment: .leading)
            .background(unread ? palette.subtle : palette.card, in: RoundedRectangle(cornerRadius: 10))
            .overlay {
                RoundedRectangle(cornerRadius: 10).stroke(palette.border, lineWidth: 1)
            }
            .overlay(alignment: .leading) {
                if unread {
                    Rectangle()
                        .fill(AppTheme.primary)
                        .frame(width: 3)
                        .clipShape(RoundedRectangle(cornerRadius: 2))
                }
            }
            .contentShape(Rectangle())
        }
        .buttonStyle(.plain)
    }

    private var loadingRow: some View {
        HStack {
            Spacer()
            ProgressView()
            Spacer()
        }
        .padding(.vertical, 20)
        .listRowSeparator(.hidden)
        .listRowBackground(Color.clear)
    }

    private var failureRow: some View {
        let feedback = model.failure?.feedback(resource: "消息中心")
            ?? LoadFailure.Feedback(title: "消息中心加载失败", description: "请稍后重试", actionText: "重试", retryable: true)
        return StateMessageView(
            icon: "exclamationmark.circle",
            title: feedback.title,
            detail: feedback.description,
            actionTitle: feedback.actionText,
            action: { Task { await model.retry() } }
        )
        .frame(minHeight: 300)
        .listRowSeparator(.hidden)
        .listRowBackground(Color.clear)
    }

    private var emptyRow: some View {
        VStack(spacing: 6) {
            Text("🔔")
                .font(.system(size: 40))
                .accessibilityHidden(true)
            Text("暂无消息")
                .font(.system(size: 15))
                .foregroundStyle(palette.textSecondary)
            Text("有新消息时会在这里通知你")
                .font(.system(size: 12))
                .foregroundStyle(palette.textMuted)
        }
        .frame(maxWidth: .infinity)
        .padding(.top, 80)
        .padding(.bottom, 40)
        .listRowSeparator(.hidden)
        .listRowBackground(Color.clear)
    }

    private var loadMoreRow: some View {
        Button {
            Task { await model.loadMore() }
        } label: {
            HStack {
                Spacer()
                if model.list.isLoadingMore {
                    ProgressView()
                } else {
                    Text("加载更多")
                        .font(.system(size: 13))
                        .foregroundStyle(palette.brandStrong)
                }
                Spacer()
            }
            .padding(.vertical, 16)
        }
        .buttonStyle(.plain)
        .listRowSeparator(.hidden)
        .listRowBackground(Color.clear)
    }

    // MARK: - 操作

    private func tap(_ item: NotificationInfo) {
        Task {
            await model.markRead(item)
            if let route = MessageCenterRules.notificationRoute(type: item.type, data: item.data) {
                navigator.open(route)
            }
        }
    }

    private func delete(_ item: NotificationInfo) {
        Task {
            if await model.delete(item) {
                toast.show("已删除")
            } else {
                toast.show("删除失败")
            }
        }
    }

    private func typeIcon(_ type: String) -> String {
        switch type {
        case "challenge": "🎯"
        case "tournament": "🏆"
        case "friend_request": "👥"
        case "follow": "⭐"
        case "match_result": "🏁"
        case "rank_change": "📊"
        case "season_rollover": "🔄"
        case "system": "📢"
        default: "🔔"
        }
    }

    private func typeTint(_ type: String) -> Color {
        switch type {
        case "match_result", "rank_change":
            palette.info.opacity(0.12)
        case "season_rollover":
            palette.success.opacity(0.14)
        case "system":
            palette.subtle
        default:
            palette.brandTint
        }
    }
}
