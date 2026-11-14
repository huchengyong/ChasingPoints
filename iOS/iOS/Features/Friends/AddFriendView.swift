import SwiftUI

/// 添加好友，对应小程序 `subPages/social/addFriend.vue`。
struct AddFriendView: View {
    @EnvironmentObject private var session: SessionStore
    @EnvironmentObject private var navigator: AppNavigator
    @EnvironmentObject private var toast: ToastCenter
    @Environment(\.colorScheme) private var colorScheme
    @Environment(\.dismiss) private var dismiss

    @StateObject private var model: AddFriendModel

    init(session: SessionStore, model: AddFriendModel? = nil) {
        _model = StateObject(wrappedValue: model ?? AddFriendModel(session: session))
    }

    private var palette: AppPalette { AppTheme.palette(for: colorScheme) }

    var body: some View {
        VStack(spacing: 0) {
            searchBar
            Group {
                if session.identity == nil {
                    StateMessageView(icon: "lock.fill", title: "登录后添加好友", actionTitle: "去登录") {
                        navigator.openLogin()
                    }
                } else {
                    resultContent
                }
            }
        }
        .background(palette.page.ignoresSafeArea())
        .navigationTitle("添加好友")
        .navigationBarTitleDisplayMode(.inline)
        .onChange(of: model.notice) { _, notice in
            guard let notice else { return }
            toast.show(notice)
            model.clearNotice()
        }
    }

    private var searchBar: some View {
        HStack(spacing: 8) {
            HStack(spacing: 6) {
                Image(systemName: "magnifyingglass")
                    .font(.system(size: 15, weight: .semibold))
                    .foregroundStyle(palette.textMuted)
                    .accessibilityHidden(true)

                TextField("搜索昵称或手机号", text: $model.keyword)
                    .font(.system(size: 14))
                    .foregroundStyle(palette.textPrimary)
                    .submitLabel(.search)
                    .textInputAutocapitalization(.never)
                    .autocorrectionDisabled()
                    .onSubmit { Task { await model.search() } }

                if !model.keyword.isEmpty {
                    Button {
                        model.clear()
                    } label: {
                        Image(systemName: "xmark.circle.fill")
                            .font(.system(size: 15))
                            .foregroundStyle(palette.textMuted)
                    }
                    .buttonStyle(.plain)
                    .accessibilityLabel("清空搜索")
                }
            }
            .padding(.horizontal, 12)
            .frame(height: 38)
            .background(palette.subtle, in: Capsule())

            Button {
                Task { await model.search() }
            } label: {
                Text("搜索")
                    .font(.system(size: 14))
                    .foregroundStyle(.white)
                    .padding(.horizontal, 14)
                    .frame(height: 38)
                    .background(
                        LinearGradient(
                            colors: [palette.gradientStart, palette.gradientEnd],
                            startPoint: .topLeading,
                            endPoint: .bottomTrailing
                        ),
                        in: Capsule()
                    )
            }
            .buttonStyle(.plain)
        }
        .padding(.horizontal, 12)
        .padding(.vertical, 10)
        .background(palette.card)
    }

    @ViewBuilder
    private var resultContent: some View {
        switch model.phase {
        case .idle:
            guideState
        case .searching:
            loadingState
        case .failed:
            failureState
        case .empty:
            emptyState
        case .loaded:
            resultList
        }
    }

    private var guideState: some View {
        VStack(spacing: 10) {
            Text("🔍")
                .font(.system(size: 40))
                .accessibilityHidden(true)
            Text("输入昵称或手机号搜索用户")
                .font(.system(size: 14))
                .foregroundStyle(palette.textMuted)
        }
        .frame(maxWidth: .infinity, maxHeight: .infinity)
        .padding(.top, 120)
        .frame(maxHeight: .infinity, alignment: .top)
    }

    private var loadingState: some View {
        VStack(spacing: 8) {
            ProgressView()
            Text("搜索中...")
                .font(.system(size: 14))
                .foregroundStyle(palette.textMuted)
        }
        .frame(maxWidth: .infinity, maxHeight: .infinity)
    }

    private var failureState: some View {
        let feedback = model.failure?.feedback(resource: "搜索结果")
            ?? LoadFailure.Feedback(title: "搜索用户失败", description: "请稍后重试", actionText: "重试", retryable: true)

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
    }

    private var emptyState: some View {
        VStack {
            Text("未找到相关用户")
                .font(.system(size: 14))
                .foregroundStyle(palette.textMuted)
        }
        .frame(maxWidth: .infinity)
        .padding(.top, 100)
        .frame(maxHeight: .infinity, alignment: .top)
    }

    private var resultList: some View {
        ScrollView {
            LazyVStack(spacing: 8) {
                ForEach(model.results) { item in
                    resultRow(item)
                }
            }
            .padding(.horizontal, 12)
            .padding(.vertical, 10)
        }
    }

    private func resultRow(_ item: SearchUserItem) -> some View {
        let status = model.status(of: item)
        let busy = model.isBusy(item)

        return HStack(spacing: 10) {
            AvatarView(avatar: item.avatar, userId: item.userId, size: 44)

            VStack(alignment: .leading, spacing: 3) {
                Text(FriendRules.displayName(item.nickname))
                    .font(.system(size: 15, weight: .medium))
                    .foregroundStyle(palette.textPrimary)
                    .lineLimit(1)
                Text("ID: \(String(item.userId))")
                    .font(.system(size: 12))
                    .foregroundStyle(palette.textMuted)
            }

            Spacer(minLength: 0)

            Button {
                add(item)
            } label: {
                Text(buttonTitle(status: status, busy: busy))
                    .font(.system(size: 12, weight: .semibold))
                    .foregroundStyle(buttonTextColor(status))
                    .padding(.horizontal, 14)
                    .frame(height: 30)
                    .background { buttonBackground(status) }
            }
            .buttonStyle(.plain)
            .disabled(status != .addable || busy)
            .opacity(busy ? 0.6 : 1)
        }
        .padding(12)
        .background(palette.card, in: RoundedRectangle(cornerRadius: 12))
    }

    private func buttonTitle(status: FriendSearchStatus, busy: Bool) -> String {
        switch status {
        case .friend: "已是好友"
        case .pending: "已发送"
        case .addable: busy ? "发送中…" : "添加"
        }
    }

    private func buttonTextColor(_ status: FriendSearchStatus) -> Color {
        switch status {
        case .friend: palette.success
        case .pending: palette.textMuted
        case .addable: .white
        }
    }

    @ViewBuilder
    private func buttonBackground(_ status: FriendSearchStatus) -> some View {
        switch status {
        case .friend:
            Capsule().fill(palette.success.opacity(0.12))
        case .pending:
            Capsule().fill(palette.subtle)
        case .addable:
            Capsule().fill(
                LinearGradient(
                    colors: [palette.gradientStart, palette.gradientEnd],
                    startPoint: .topLeading,
                    endPoint: .bottomTrailing
                )
            )
        }
    }

    private func add(_ item: SearchUserItem) {
        Task {
            switch await model.sendRequest(to: item) {
            case .success: toast.show("请求已发送")
            case .failure(let message): toast.show(message)
            }
        }
    }
}
