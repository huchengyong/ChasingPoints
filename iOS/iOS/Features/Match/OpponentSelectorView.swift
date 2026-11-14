import SwiftUI

/// 选择球友：列出服务端候选，点击直接进入约球发起页。
struct OpponentSelectorView: View {
    @EnvironmentObject private var session: SessionStore
    @EnvironmentObject private var navigator: AppNavigator
    @Environment(\.colorScheme) private var colorScheme
    @StateObject private var model: OpponentSelectorModel

    init(session: SessionStore, model: OpponentSelectorModel? = nil) {
        _model = StateObject(wrappedValue: model ?? OpponentSelectorModel(session: session))
    }

    private var palette: AppPalette { AppTheme.palette(for: colorScheme) }

    var body: some View {
        ScrollView {
            VStack(spacing: 16) {
                intro
                content
            }
            .padding(.horizontal, 14)
            .padding(.vertical, 14)
        }
        .background(palette.page.ignoresSafeArea())
        .navigationTitle("选择球友")
        .navigationBarTitleDisplayMode(.inline)
        .task { await model.loadIfNeeded() }
    }

    private var intro: some View {
        VStack(alignment: .leading, spacing: 6) {
            Text("选择一位球友")
                .font(.system(size: 19, weight: .heavy))
                .foregroundStyle(.white)
            Text("好友优先展示，最近对手按最近对局排序。点头像直接进入约球发起页。")
                .font(.system(size: 12, weight: .regular))
                .foregroundStyle(.white.opacity(0.82))
                .lineSpacing(3)
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .padding(15)
        .background(Color(hex: 0x1E180D), in: RoundedRectangle(cornerRadius: 13))
    }

    @ViewBuilder
    private var content: some View {
        switch model.phase {
        case .idle, .loading:
            stateBlock {
                ProgressView()
                    .tint(AppTheme.primary)
                Text("加载球友中...")
                    .font(.system(size: 13))
                    .foregroundStyle(palette.textSecondary)
            }
        case .empty:
            stateBlock {
                Text("暂无可邀请的球友")
                    .font(.system(size: 17, weight: .bold))
                    .foregroundStyle(palette.textPrimary)
                Text("先添加好友或完成一场平台注册对局后再试。")
                    .font(.system(size: 13))
                    .foregroundStyle(palette.textSecondary)
                    .multilineTextAlignment(.center)
            }
        case .loaded:
            VStack(spacing: 8) {
                ForEach(model.opponents) { opponent in
                    opponentRow(opponent)
                }
            }
        case .failed(let failure):
            stateBlock {
                Text("加载球友失败")
                    .font(.system(size: 17, weight: .bold))
                    .foregroundStyle(palette.textPrimary)
                Text(failure.feedback(resource: "球友").description)
                    .font(.system(size: 13))
                    .foregroundStyle(palette.textSecondary)
                    .multilineTextAlignment(.center)
                Button("重试") { Task { await model.retry() } }
                    .buttonStyle(.appPill)
                    .padding(.top, 4)
            }
        }
    }

    private func stateBlock<Content: View>(@ViewBuilder content: () -> Content) -> some View {
        VStack(spacing: 10) {
            content()
        }
        .frame(maxWidth: .infinity)
        .padding(.top, 80)
    }

    private func opponentRow(_ opponent: OpponentCandidate) -> some View {
        Button {
            navigator.open(.challengeCompose(target: ChallengeComposeTarget(
                userId: opponent.userId,
                nickname: opponent.name,
                avatar: opponent.avatar
            )))
        } label: {
            HStack(spacing: 10) {
                AvatarView(avatar: opponent.avatar, userId: opponent.userId, size: 46)
                VStack(alignment: .leading, spacing: 3) {
                    Text(opponent.name)
                        .font(.system(size: 15, weight: .bold))
                        .foregroundStyle(palette.textPrimary)
                        .lineLimit(1)
                    Text(opponent.sourceText)
                        .font(.system(size: 11.5))
                        .foregroundStyle(palette.textMuted)
                }
                Spacer(minLength: 0)
                Image(systemName: "chevron.right")
                    .font(.system(size: 13, weight: .semibold))
                    .foregroundStyle(palette.textMuted)
            }
            .padding(11)
            .background(palette.card, in: RoundedRectangle(cornerRadius: 11))
            .overlay {
                RoundedRectangle(cornerRadius: 11).stroke(palette.border, lineWidth: 1)
            }
        }
        .buttonStyle(.plain)
    }
}
