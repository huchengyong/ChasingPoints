import SwiftUI

/// 好友申请，对应小程序 `subPages/social/friendRequests.vue`。
struct FriendRequestsView: View {
    @EnvironmentObject private var session: SessionStore
    @EnvironmentObject private var navigator: AppNavigator
    @EnvironmentObject private var toast: ToastCenter
    @EnvironmentObject private var activity: ActivityStore
    @Environment(\.colorScheme) private var colorScheme
    @Environment(\.dismiss) private var dismiss

    @StateObject private var model: FriendRequestsModel
    @State private var pendingBlacklist: FriendRequestInfo?

    init(session: SessionStore, activity: ActivityStore, model: FriendRequestsModel? = nil) {
        _model = StateObject(wrappedValue: model ?? FriendRequestsModel(session: session, activity: activity))
    }

    private var palette: AppPalette { AppTheme.palette(for: colorScheme) }

    var body: some View {
        Group {
            if session.identity == nil {
                StateMessageView(icon: "lock.fill", title: "登录后查看好友申请", actionTitle: "去登录") {
                    navigator.openLogin()
                }
            } else {
                content
            }
        }
        .background(palette.page.ignoresSafeArea())
        .navigationTitle("好友申请")
        .navigationBarTitleDisplayMode(.inline)
        .task { await model.onAppear() }
        .onChange(of: session.identity) { _, _ in
            Task { await model.identityChanged() }
        }
        .alert(
            "加入黑名单",
            isPresented: Binding(
                get: { pendingBlacklist != nil },
                set: { if !$0 { pendingBlacklist = nil } }
            ),
            presenting: pendingBlacklist
        ) { request in
            Button("确认拉黑") {
                blacklist(request)
            }
            Button("取消", role: .cancel) {}
        } message: { request in
            Text(FriendRules.rejectAndBlacklistConfirmMessage(request.nickname))
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
                    ForEach(model.items) { request in
                        requestCard(request)
                    }
                    if model.hasMore {
                        Button {
                            Task { await model.loadMore() }
                        } label: {
                            Text("加载更多")
                                .font(.system(size: 13))
                                .foregroundStyle(palette.textSecondary)
                                .frame(maxWidth: .infinity)
                                .padding(.vertical, 10)
                        }
                        .buttonStyle(.plain)
                    }
                }
            }
            .padding(.horizontal, 12)
            .padding(.vertical, 10)
        }
        .refreshable { await model.refresh() }
    }

    private func requestCard(_ request: FriendRequestInfo) -> some View {
        let message = FriendRules.requestMessage(request.message)
        let status = model.status(of: request)
        let busy = model.isBusy(request)

        return VStack(spacing: 0) {
            HStack(spacing: 10) {
                AvatarView(avatar: request.avatar, userId: request.applicantUserId, size: 48)

                VStack(alignment: .leading, spacing: 5) {
                    HStack(spacing: 8) {
                        Text(FriendRules.displayName(request.nickname))
                            .font(.system(size: 15, weight: .semibold))
                            .foregroundStyle(palette.textPrimary)
                            .lineLimit(1)
                        Spacer(minLength: 0)
                        Text(FriendRules.requestTime(request.createdAt))
                            .font(.system(size: 11))
                            .foregroundStyle(palette.textMuted)
                            .lineLimit(1)
                    }
                    Text(message.text)
                        .font(.system(size: 12))
                        .foregroundStyle(message.isMuted ? palette.textMuted : palette.textSecondary)
                        .lineLimit(1)
                }
            }

            if status == 0 {
                HStack(spacing: 6) {
                    Menu {
                        Button(role: .destructive) {
                            pendingBlacklist = request
                        } label: {
                            Label("拒绝并加入黑名单", systemImage: "person.crop.circle.badge.xmark")
                        }
                    } label: {
                        Image(systemName: "ellipsis")
                            .font(.system(size: 15))
                            .foregroundStyle(palette.textMuted)
                            .frame(width: 32, height: 32)
                            .contentShape(Rectangle())
                    }

                    Spacer(minLength: 0)

                    actionButton("拒绝", filled: false, busy: busy) {
                        reject(request)
                    }
                    actionButton("接受", filled: true, busy: busy) {
                        accept(request)
                    }
                }
                .padding(.top, 10)
            } else {
                HStack {
                    Spacer()
                    Text(FriendRules.requestStatusText(status))
                        .font(.system(size: 11, weight: .medium))
                        .foregroundStyle(status == 1 ? palette.success : palette.textMuted)
                }
                .padding(.top, 10)
            }
        }
        .padding(.horizontal, 12)
        .padding(.vertical, 12)
        .background(palette.card, in: RoundedRectangle(cornerRadius: 14))
        .overlay {
            RoundedRectangle(cornerRadius: 14).stroke(palette.brandTint, lineWidth: 1)
        }
    }

    private func actionButton(
        _ title: String,
        filled: Bool,
        busy: Bool,
        action: @escaping () -> Void
    ) -> some View {
        Button(action: action) {
            Text(title)
                .font(.system(size: 12, weight: .semibold))
                .foregroundStyle(filled ? Color.white : palette.textSecondary)
                .padding(.horizontal, 14)
                .frame(height: 32)
                .background {
                    if filled {
                        Capsule().fill(
                            LinearGradient(
                                colors: [palette.gradientStart, palette.gradientEnd],
                                startPoint: .topLeading,
                                endPoint: .bottomTrailing
                            )
                        )
                    } else {
                        Capsule()
                            .fill(palette.subtle)
                            .overlay(Capsule().stroke(palette.border, lineWidth: 1))
                    }
                }
        }
        .buttonStyle(.plain)
        .disabled(busy)
        .opacity(busy ? 0.6 : 1)
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
        let feedback = model.failure?.feedback(resource: "好友申请")
            ?? LoadFailure.Feedback(title: "好友申请加载失败", description: "请稍后重试", actionText: "重试", retryable: true)

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
            Text("📬")
                .font(.system(size: 40))
                .accessibilityHidden(true)
            Text("暂无好友请求")
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

    private func accept(_ request: FriendRequestInfo) {
        Task {
            switch await model.accept(request) {
            case .success: toast.show("已接受")
            case .failure(let message): toast.show(message)
            }
        }
    }

    private func reject(_ request: FriendRequestInfo) {
        Task {
            switch await model.reject(request) {
            case .success: toast.show("已拒绝")
            case .failure(let message): toast.show(message)
            }
        }
    }

    private func blacklist(_ request: FriendRequestInfo) {
        Task {
            switch await model.rejectAndBlacklist(request) {
            case .success: toast.show("已加入黑名单")
            case .failure(let message): toast.show(message)
            }
        }
    }
}
