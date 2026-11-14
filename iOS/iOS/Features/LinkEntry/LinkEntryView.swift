import SwiftUI

/// 入口链接承接页，对应小程序 `subPages/match/linkEntry.vue`。
struct LinkEntryView: View {
    @EnvironmentObject private var session: SessionStore
    @EnvironmentObject private var navigator: AppNavigator
    @Environment(\.colorScheme) private var colorScheme

    @StateObject private var model: LinkEntryModel
    @State private var didStart = false

    private let url: String
    private let usesInjectedModel: Bool

    /// 注入模型（测试 / 快照）已带好状态，页面不再按 URL 自动开始。
    init(session: SessionStore, url: String, quickScan: QuickScanFlow, model: LinkEntryModel? = nil) {
        self.url = url
        usesInjectedModel = model != nil
        _model = StateObject(wrappedValue: model ?? LinkEntryModel(session: session, quickScan: quickScan))
    }

    private var palette: AppPalette { AppTheme.palette(for: colorScheme) }

    var body: some View {
        ScrollView {
            content
                .frame(maxWidth: .infinity)
                .padding(.horizontal, 24)
                .padding(.top, 64)
        }
        .background(palette.page.ignoresSafeArea())
        .navigationTitle("追分邀请")
        .navigationBarTitleDisplayMode(.inline)
        .task {
            guard !usesInjectedModel, !didStart else { return }
            didStart = true
            await start()
        }
        .onChange(of: session.identity) { _, _ in
            Task {
                await model.resume()
                consumeConfirmation()
            }
        }
        .onChange(of: model.didConfirm) { _, _ in
            consumeConfirmation()
        }
    }

    @ViewBuilder
    private var content: some View {
        switch model.stage {
        case .loading:
            loadingBlock
        case .preview:
            previewBlock
        case .expired:
            expiredBlock
        case .error:
            errorBlock
        }
    }

    private var loadingBlock: some View {
        VStack(spacing: 12) {
            ProgressView()
            Text("正在打开邀请…")
                .font(.system(size: 14))
                .foregroundStyle(palette.textSecondary)
        }
        .padding(.top, 120)
    }

    private var expiredBlock: some View {
        VStack(spacing: 16) {
            VStack(spacing: 8) {
                Text("邀请已失效")
                    .font(.system(size: 17, weight: .bold))
                    .foregroundStyle(palette.textPrimary)
                Text(model.errorMessage)
                    .font(.system(size: 14))
                    .foregroundStyle(AppTheme.danger)
                    .multilineTextAlignment(.center)
            }
            Button("返回首页") { goHome() }
                .buttonStyle(.appPrimary)
        }
        .padding(.top, 80)
    }

    private var errorBlock: some View {
        VStack(spacing: 16) {
            VStack(spacing: 8) {
                Text("暂时无法打开邀请")
                    .font(.system(size: 17, weight: .bold))
                    .foregroundStyle(palette.textPrimary)
                Text(model.errorMessage)
                    .font(.system(size: 14))
                    .foregroundStyle(AppTheme.danger)
                    .multilineTextAlignment(.center)
            }
            Button("重试") {
                Task {
                    await model.retry()
                    consumeConfirmation()
                }
            }
            .buttonStyle(.appPrimary)
            Button("返回首页") { goHome() }
                .buttonStyle(.appSecondary)
        }
        .padding(.top, 80)
    }

    private var previewBlock: some View {
        VStack(spacing: 0) {
            VStack(spacing: 8) {
                inviterAvatar
                Text(model.inviter.name.isEmpty ? "球友" : model.inviter.name)
                    .font(.system(size: 18, weight: .semibold))
                    .foregroundStyle(palette.textPrimary)
                    .lineLimit(1)
                Text("邀请你打一局")
                    .font(.system(size: 14))
                    .foregroundStyle(palette.textSecondary)
                remainingText
            }
            .frame(maxWidth: .infinity)
            .padding(.vertical, 28)
            .background(palette.card, in: RoundedRectangle(cornerRadius: 16))
            .overlay {
                RoundedRectangle(cornerRadius: 16).stroke(palette.border)
            }
            .padding(.bottom, 32)

            Button("登录并开局") { navigator.open(.login) }
                .buttonStyle(.appPrimary)

            Text("登录后直接进入对局确认，无需再扫码")
                .font(.system(size: 12))
                .foregroundStyle(palette.textMuted)
                .padding(.top, 14)
        }
    }

    private var inviterAvatar: some View {
        ZStack {
            Circle().fill(palette.subtle)
            if !model.inviter.avatar.isEmpty, let url = URL(string: model.inviter.avatar) {
                RemoteImage(url: url) {
                    personIcon
                }
                .frame(width: 80, height: 80)
                .clipShape(Circle())
            } else {
                personIcon
            }
        }
        .frame(width: 80, height: 80)
        .accessibilityHidden(true)
    }

    private var personIcon: some View {
        Image(systemName: "person.fill")
            .font(.system(size: 28))
            .foregroundStyle(palette.textMuted)
    }

    /// 短码 5 分钟有效，剩余时间每 30 秒刷新一次即可。
    private var remainingText: some View {
        TimelineView(.periodic(from: .now, by: 30)) { context in
            let text = LinkEntryRules.remainingText(expiresAt: model.expiresAt, now: context.date)
            if !text.isEmpty {
                Text(text)
                    .font(.system(size: 12))
                    .foregroundStyle(palette.textMuted)
            }
        }
    }

    private func start() async {
        switch LinkEntryRouting.landing(for: EntryLink.parse(url)) {
        case .invite(let code):
            await model.begin(code: code)
            consumeConfirmation()
        case .publicMatch(let matchId):
            navigator.popToRoot(navigator.selectedTab)
            navigator.open(.publicMatchDetail(matchId: matchId))
        case .home:
            goHome()
        }
    }

    private func consumeConfirmation() {
        guard model.didConfirm else { return }
        model.clearConfirmation()
        navigator.popToRoot(navigator.selectedTab)
        navigator.open(.scanConfirm)
    }

    private func goHome() {
        navigator.popToRoot(.home)
        navigator.select(.home, isSignedIn: true)
    }
}
