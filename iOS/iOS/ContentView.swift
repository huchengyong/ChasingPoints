import SwiftUI

struct ContentView: View {
    @EnvironmentObject private var session: SessionStore
    @EnvironmentObject private var navigator: AppNavigator
    @AppStorage("theme_mode") private var themeMode = "system"
    @State private var isWelcomePresented = false

    var body: some View {
        MainTabView(session: session)
            .preferredColorScheme(ThemeMode(rawValue: themeMode)?.colorScheme)
            .fullScreenCover(isPresented: $isWelcomePresented) {
                WelcomeView(
                    onOpenLogin: {
                        isWelcomePresented = false
                        navigator.openLogin()
                    },
                    onBrowse: { isWelcomePresented = false }
                )
            }
            .task {
                // 等启动恢复结束后再判断，已登录用户不会看到首次欢迎页。
                await session.restore()
                while session.isRestoring {
                    try? await Task.sleep(for: .milliseconds(50))
                }
                isWelcomePresented = EntryFunnelRules.shouldShowWelcome(
                    hasViewed: EntryFunnelAgreementStore().hasViewedWelcome,
                    isSignedIn: session.identity != nil
                )
            }
    }
}
