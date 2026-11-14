import SwiftUI

/// 黑名单，对应小程序 `subPages/social/blacklist.vue`。
struct BlacklistView: View {
    @EnvironmentObject private var session: SessionStore
    @EnvironmentObject private var navigator: AppNavigator
    @EnvironmentObject private var toast: ToastCenter
    @Environment(\.colorScheme) private var colorScheme
    @Environment(\.dismiss) private var dismiss

    @StateObject private var model: BlacklistModel
    @State private var pendingRemove: FriendBlacklistItem?

    init(session: SessionStore, model: BlacklistModel? = nil) {
        _model = StateObject(wrappedValue: model ?? BlacklistModel(session: session))
    }

    private var palette: AppPalette { AppTheme.palette(for: colorScheme) }

    var body: some View {
        Group {
            if session.identity == nil {
                StateMessageView(icon: "lock.fill", title: "登录后查看黑名单", actionTitle: "去登录") {
                    navigator.openLogin()
                }
            } else {
                content
            }
        }
        .background(palette.page.ignoresSafeArea())
        .navigationTitle("黑名单")
        .navigationBarTitleDisplayMode(.inline)
        .task { await model.onAppear() }
        .onChange(of: session.identity) { _, _ in
            Task { await model.identityChanged() }
        }
        .alert(
            "移出黑名单",
            isPresented: Binding(
                get: { pendingRemove != nil },
                set: { if !$0 { pendingRemove = nil } }
            ),
            presenting: pendingRemove
        ) { item in
            Button("移出") {
                remove(item)
            }
            Button("取消", role: .cancel) {}
        } message: { item in
            Text(FriendRules.unblacklistConfirmMessage(item.nickname))
        }
    }

    private var content: some View {
        ScrollView {
            LazyVStack(spacing: 9) {
                if let refreshError = model.refreshError {
                    refreshBanner(refreshError)
                }

                switch model.phase {
                case .idle, .loading:
                    loadingState
                case .failed:
                    failureState
                case .empty:
                    emptyState
                case .loaded:
                    ForEach(model.items) { item in
                        row(item)
                            .task {
                                guard item.friendUserId == model.items.last?.friendUserId else { return }
                                await model.loadMore()
                            }
                    }
                    if model.hasMore {
                        Text("上拉加载更多")
                            .font(.system(size: 13))
                            .foregroundStyle(palette.textSecondary)
                            .frame(maxWidth: .infinity)
                            .padding(.vertical, 10)
                    }
                }
            }
            .padding(.horizontal, 12)
            .padding(.vertical, 10)
        }
        .refreshable { await model.refresh() }
    }

    private func row(_ item: FriendBlacklistItem) -> some View {
        let busy = model.isBusy(item)

        return HStack(spacing: 10) {
            MemberAvatarView(
                avatar: item.avatar,
                userId: item.friendUserId,
                size: 44,
                reserveFrame: true
            )

            VStack(alignment: .leading, spacing: 4) {
                Text(FriendRules.displayName(item.nickname))
                    .font(.system(size: 15, weight: .semibold))
                    .foregroundStyle(palette.textPrimary)
                    .lineLimit(1)
                Text(FriendRules.blockedTime(item.createdAt))
                    .font(.system(size: 11))
                    .foregroundStyle(palette.textMuted)
            }

            Spacer(minLength: 0)

            Button {
                pendingRemove = item
            } label: {
                Text("移出")
                    .font(.system(size: 12, weight: .semibold))
                    .foregroundStyle(palette.textSecondary)
                    .padding(.horizontal, 14)
                    .frame(height: 32)
                    .background(palette.subtle, in: Capsule())
                    .overlay(Capsule().stroke(palette.border, lineWidth: 1))
            }
            .buttonStyle(.plain)
            .disabled(busy)
            .opacity(busy ? 0.6 : 1)
        }
        .padding(12)
        .background(palette.card, in: RoundedRectangle(cornerRadius: 14))
        .overlay {
            RoundedRectangle(cornerRadius: 14).stroke(palette.brandTint, lineWidth: 1)
        }
    }

    private var loadingState: some View {
        VStack(spacing: 8) {
            ProgressView()
            Text("加载中...")
                .font(.system(size: 14))
                .foregroundStyle(palette.textMuted)
        }
        .frame(maxWidth: .infinity)
        .padding(.top, 120)
    }

    private var failureState: some View {
        let feedback = model.failure?.feedback(resource: "黑名单")
            ?? LoadFailure.Feedback(title: "黑名单加载失败", description: "请稍后重试", actionText: "重试", retryable: true)

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
    }

    private var emptyState: some View {
        VStack(spacing: 10) {
            Text("🚫")
                .font(.system(size: 40))
                .accessibilityHidden(true)
            Text("黑名单是空的")
                .font(.system(size: 14))
                .foregroundStyle(palette.textMuted)
        }
        .frame(maxWidth: .infinity)
        .padding(.top, 140)
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

    private func remove(_ item: FriendBlacklistItem) {
        Task {
            switch await model.remove(item) {
            case .success: toast.show("已移出黑名单")
            case .failure(let message): toast.show(message)
            }
        }
    }
}
