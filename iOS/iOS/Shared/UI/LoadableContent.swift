import SwiftUI

/// 把 ResourceLoader 的各状态画出来：加载中、失败（带重试）、未登录引导、已加载内容 + 刷新失败提示。
struct LoadableContent<Value, Content: View>: View {
    @EnvironmentObject private var session: SessionStore
    @EnvironmentObject private var navigator: AppNavigator
    @Environment(\.colorScheme) private var colorScheme
    @ObservedObject private var loader: ResourceLoader<Value>

    private let requiresLogin: Bool
    private let loginPrompt: String
    private let content: (Value) -> Content

    init(
        loader: ResourceLoader<Value>,
        requiresLogin: Bool = true,
        loginPrompt: String = "登录后查看",
        @ViewBuilder content: @escaping (Value) -> Content
    ) {
        _loader = ObservedObject(wrappedValue: loader)
        self.requiresLogin = requiresLogin
        self.loginPrompt = loginPrompt
        self.content = content
    }

    var body: some View {
        Group {
            if requiresLogin && session.identity == nil {
                StateMessageView(icon: "lock.fill", title: loginPrompt, actionTitle: "去登录") {
                    navigator.openLogin()
                }
            } else {
                switch loader.phase {
                case .idle, .loading:
                    ProgressView()
                        .frame(maxWidth: .infinity, maxHeight: .infinity)
                case .failed(let message):
                    StateMessageView(
                        icon: "exclamationmark.triangle",
                        title: "加载失败",
                        detail: message,
                        actionTitle: "重试"
                    ) {
                        Task { await loader.retry() }
                    }
                case .loaded(let value):
                    VStack(spacing: 0) {
                        if let refreshError = loader.refreshError {
                            refreshBanner(refreshError)
                        }
                        content(value)
                    }
                }
            }
        }
        .task { await loader.loadIfNeeded() }
        .onChange(of: session.identity) { _, _ in
            loader.syncIdentity()
            Task { await loader.loadIfNeeded() }
        }
    }

    private func refreshBanner(_ message: String) -> some View {
        let palette = AppTheme.palette(for: colorScheme)

        return HStack(spacing: 8) {
            Text(message)
                .font(.system(size: 14))
                .foregroundStyle(palette.textSecondary)
            Spacer(minLength: 8)
            Button("重试") {
                Task { await loader.retry() }
            }
            .font(.system(size: 14, weight: .semibold))
            .foregroundStyle(palette.brandStrong)
        }
        .padding(.horizontal, 16)
    }
}
