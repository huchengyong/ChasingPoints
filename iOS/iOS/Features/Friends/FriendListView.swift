import SwiftUI

/// 好友列表，对应小程序 `subPages/social/friendList.vue`。
struct FriendListView: View {
    @EnvironmentObject private var session: SessionStore
    @EnvironmentObject private var navigator: AppNavigator
    @EnvironmentObject private var toast: ToastCenter
    @EnvironmentObject private var activity: ActivityStore
    @Environment(\.colorScheme) private var colorScheme
    @Environment(\.dismiss) private var dismiss

    @StateObject private var model: FriendListModel
    @State private var confirm: Confirm?

    private enum Confirm: Identifiable {
        case delete(FriendInfo)
        case blacklist(FriendInfo)

        var id: String {
            switch self {
            case .delete(let friend): "delete-\(friend.friendUserId)"
            case .blacklist(let friend): "blacklist-\(friend.friendUserId)"
            }
        }
    }

    init(session: SessionStore, activity: ActivityStore, model: FriendListModel? = nil) {
        _model = StateObject(wrappedValue: model ?? FriendListModel(session: session, activity: activity))
    }

    private var palette: AppPalette { AppTheme.palette(for: colorScheme) }

    var body: some View {
        Group {
            if session.identity == nil {
                StateMessageView(
                    icon: "lock.fill",
                    title: "登录后查看好友",
                    actionTitle: "去登录"
                ) {
                    navigator.openLogin()
                }
            } else {
                content
            }
        }
        .background(palette.page.ignoresSafeArea())
        .navigationTitle("好友列表")
        .navigationBarTitleDisplayMode(.inline)
        .task { await model.onAppear() }
        .onChange(of: session.identity) { _, _ in
            Task { await model.identityChanged() }
        }
        .alert(item: $confirm) { item in
            switch item {
            case .delete(let friend):
                Alert(
                    title: Text("删除好友"),
                    message: Text(FriendRules.deleteConfirmMessage(friend.nickname)),
                    primaryButton: .destructive(Text("删除")) {
                        delete(friend)
                    },
                    secondaryButton: .cancel(Text("取消"))
                )
            case .blacklist(let friend):
                Alert(
                    title: Text("加入黑名单"),
                    message: Text(FriendRules.blacklistConfirmMessage(friend.nickname)),
                    primaryButton: .default(Text("确认拉黑")) {
                        blacklist(friend)
                    },
                    secondaryButton: .cancel(Text("取消"))
                )
            }
        }
    }

    private var content: some View {
        List {
            header
                .listRowInsets(EdgeInsets(top: 10, leading: 16, bottom: 6, trailing: 16))
                .listRowSeparator(.hidden)
                .listRowBackground(Color.clear)

            if let refreshError = model.refreshError {
                refreshBanner(refreshError)
                    .listRowInsets(EdgeInsets(top: 0, leading: 16, bottom: 6, trailing: 16))
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
                ForEach(model.items) { friend in
                    NavigationLink(value: FriendRules.homepageRoute(friend)) {
                        friendRow(friend)
                    }
                    .buttonStyle(.plain)
                    .listRowInsets(EdgeInsets(top: 5, leading: 16, bottom: 5, trailing: 16))
                    .listRowSeparator(.hidden)
                    .listRowBackground(Color.clear)
                    .swipeActions(edge: .trailing, allowsFullSwipe: false) {
                        Button {
                            confirm = .delete(friend)
                        } label: {
                            Text("删除")
                        }
                        .tint(AppTheme.danger)

                        Button {
                            confirm = .blacklist(friend)
                        } label: {
                            Text("拉黑")
                        }
                        .tint(palette.textSecondary)
                    }
                }

                if model.hasMore || model.isLoadingMore || model.loadMoreError != nil {
                    loadMoreRow
                }
            }
        }
        .listStyle(.plain)
        .scrollContentBackground(.hidden)
        .refreshable { await model.refresh() }
    }

    private var header: some View {
        HStack(spacing: 12) {
            Button {
                navigator.open(.addFriend)
            } label: {
                HStack(spacing: 8) {
                    Image(systemName: "magnifyingglass")
                        .font(.system(size: 15, weight: .semibold))
                    Text("搜索/添加好友")
                        .font(.system(size: 14))
                    Spacer(minLength: 0)
                }
                .foregroundStyle(palette.textMuted)
                .padding(.horizontal, 14)
                .frame(height: 36)
                .background(palette.subtle, in: Capsule())
            }
            .buttonStyle(.plain)

            Button {
                navigator.open(.friendRequests)
            } label: {
                Image(systemName: "person.crop.circle.fill")
                    .font(.system(size: 24))
                    .foregroundStyle(AppTheme.primary)
                    .overlay(alignment: .topTrailing) {
                        if let badge = FriendRules.pendingBadgeText(model.pendingCount) {
                            Text(badge)
                                .font(.system(size: 10, weight: .semibold))
                                .foregroundStyle(.white)
                                .padding(.horizontal, 4)
                                .frame(minWidth: 16, minHeight: 16)
                                .background(AppTheme.danger, in: Capsule())
                                .offset(x: 7, y: -5)
                        }
                    }
            }
            .buttonStyle(.plain)
            .accessibilityLabel("好友申请")
        }
        .padding(.vertical, 4)
    }

    private func friendRow(_ friend: FriendInfo) -> some View {
        HStack(spacing: 12) {
            MemberAvatarView(
                avatar: friend.avatar,
                userId: friend.friendUserId,
                vipLevel: friend.vipLevel,
                rankLevel: friend.rankLevel,
                size: 40,
                reserveFrame: true
            )

            VStack(alignment: .leading, spacing: 3) {
                HStack(spacing: 6) {
                    Text(FriendRules.displayName(friend.nickname))
                        .font(.system(size: 15, weight: .medium))
                        .foregroundStyle(palette.textPrimary)
                        .lineLimit(1)
                    if !friend.rankName.isEmpty {
                        Text(friend.rankName)
                            .font(.system(size: 11))
                            .foregroundStyle(AppTheme.warning)
                            .padding(.horizontal, 6)
                            .padding(.vertical, 1)
                            .background(AppTheme.warning.opacity(0.12), in: Capsule())
                    }
                }
                Text("ID: \(String(friend.friendUserId))")
                    .font(.system(size: 12))
                    .foregroundStyle(palette.textMuted)
            }

            Spacer(minLength: 0)

            Image(systemName: "chevron.right")
                .font(.system(size: 12, weight: .semibold))
                .foregroundStyle(palette.textMuted)
                .accessibilityHidden(true)
        }
        .padding(12)
        .background(palette.card, in: RoundedRectangle(cornerRadius: 12))
        .contentShape(Rectangle())
    }

    private var loadingRow: some View {
        VStack(spacing: 8) {
            ProgressView()
            Text("加载中...")
                .font(.system(size: 14))
                .foregroundStyle(palette.textMuted)
        }
        .frame(maxWidth: .infinity)
        .padding(.top, 120)
        .listRowSeparator(.hidden)
        .listRowBackground(Color.clear)
    }

    private var failureRow: some View {
        let feedback = model.failure?.feedback(resource: "好友列表")
            ?? LoadFailure.Feedback(title: "好友列表加载失败", description: "请稍后重试", actionText: "重试", retryable: true)

        return StateMessageView(
            icon: "exclamationmark.circle",
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
        .frame(minHeight: 320)
        .listRowSeparator(.hidden)
        .listRowBackground(Color.clear)
    }

    private var emptyRow: some View {
        VStack(spacing: 10) {
            Text("👥")
                .font(.system(size: 40))
                .accessibilityHidden(true)
            Text("还没有好友")
                .font(.system(size: 16, weight: .medium))
                .foregroundStyle(palette.textPrimary)
            Text("去添加好友，一起打球吧！")
                .font(.system(size: 13))
                .foregroundStyle(palette.textMuted)
            Button("添加好友") {
                navigator.open(.addFriend)
            }
            .buttonStyle(.appPill)
            .padding(.top, 8)
        }
        .frame(maxWidth: .infinity)
        .padding(.top, 100)
        .listRowSeparator(.hidden)
        .listRowBackground(Color.clear)
    }

    private var loadMoreRow: some View {
        Button {
            Task { await model.loadMore() }
        } label: {
            HStack {
                Spacer()
                if model.isLoadingMore {
                    ProgressView()
                } else if let error = model.loadMoreError {
                    Text(error)
                        .font(.system(size: 12))
                        .foregroundStyle(palette.textMuted)
                } else {
                    Text("加载更多")
                        .font(.system(size: 13))
                        .foregroundStyle(palette.textSecondary)
                }
                Spacer()
            }
            .padding(.vertical, 14)
        }
        .buttonStyle(.plain)
        .listRowSeparator(.hidden)
        .listRowBackground(Color.clear)
    }

    private func refreshBanner(_ message: String) -> some View {
        HStack(spacing: 8) {
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
        .background(palette.brandTint, in: RoundedRectangle(cornerRadius: 9))
    }

    private func delete(_ friend: FriendInfo) {
        Task {
            if await model.delete(friend) {
                toast.show("已删除")
            } else {
                toast.show("删除失败")
            }
        }
    }

    private func blacklist(_ friend: FriendInfo) {
        Task {
            if await model.addToBlacklist(friend) {
                toast.show("已加入黑名单")
            } else {
                toast.show("操作失败")
            }
        }
    }
}
