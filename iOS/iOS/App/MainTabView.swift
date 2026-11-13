import SwiftUI

/// 五个主 Tab：首页 / 赛讯 / 扫码 / 消息 / 我的；扫码只触发流程，不切换 Tab。
struct MainTabView: View {
    @EnvironmentObject private var session: SessionStore
    @EnvironmentObject private var navigator: AppNavigator
    @EnvironmentObject private var toast: ToastCenter
    @Environment(\.colorScheme) private var colorScheme
    @State private var pendingScanOutcome: ScanOutcome?

    private var palette: AppPalette { AppTheme.palette(for: colorScheme) }

    var body: some View {
        TabView(selection: selectedTabBinding) {
            NavigationStack(path: pathBinding(for: .home)) {
                MigrationPlaceholderView(title: "追分")
                    .navigationDestination(for: AppRoute.self) { AppRouteDestination(route: $0) }
            }
            .tabItem {
                Image(navigator.selectedTab == .home ? "TabHomeSelected" : "TabHome")
                    .renderingMode(.original)
                Text(MainTab.home.title)
            }
            .tag(MainTab.home)

            NavigationStack(path: pathBinding(for: .news)) {
                MigrationPlaceholderView(title: "赛讯")
                    .navigationDestination(for: AppRoute.self) { AppRouteDestination(route: $0) }
            }
            .tabItem {
                Image(navigator.selectedTab == .news ? "TabNewsSelected" : "TabNews")
                    .renderingMode(.original)
                Text(MainTab.news.title)
            }
            .tag(MainTab.news)

            Color.clear
                .tabItem {
                    Image("TabScan")
                        .renderingMode(.original)
                    Text(MainTab.scan.title)
                }
                .tag(MainTab.scan)

            NavigationStack(path: pathBinding(for: .messages)) {
                MigrationPlaceholderView(title: "消息")
                    .navigationDestination(for: AppRoute.self) { AppRouteDestination(route: $0) }
            }
            .tabItem {
                Image(navigator.selectedTab == .messages ? "TabMessageSelected" : "TabMessage")
                    .renderingMode(.original)
                Text(MainTab.messages.title)
            }
            .tag(MainTab.messages)

            NavigationStack(path: pathBinding(for: .me)) {
                MyPage(session: session)
                    .navigationDestination(for: AppRoute.self) { AppRouteDestination(route: $0) }
            }
            .tabItem {
                Image(navigator.selectedTab == .me ? "TabMeSelected" : "TabMe")
                    .renderingMode(.original)
                Text(MainTab.me.title)
            }
            .tag(MainTab.me)
        }
        .tint(AppTheme.primary)
        .toolbarBackground(palette.card, for: .tabBar)
        .toolbarBackground(.visible, for: .tabBar)
        .fullScreenCover(isPresented: $navigator.isScanPresented) {
            QRScannerView { outcome in
                pendingScanOutcome = outcome
                navigator.isScanPresented = false
            }
        }
        .onChange(of: navigator.isScanPresented) { _, isPresented in
            guard !isPresented, let outcome = pendingScanOutcome else { return }
            pendingScanOutcome = nil
            Task { await handleScan(outcome) }
        }
        .task { QuickScanFlow.shared.bind(session: session) }
        .alert("登录状态已失效", isPresented: sessionExpiredBinding) {
            Button("重新登录") {
                session.acknowledgeSessionExpired()
                navigator.openLogin()
            }
            Button("暂不登录", role: .cancel) {
                session.acknowledgeSessionExpired()
            }
        } message: {
            Text(session.sessionExpiredNotice ?? "请重新登录")
        }
        .onChange(of: session.identity) { oldValue, newValue in
            // 退出、失效或切号时清空导航栈；从访客变为登录不清栈。
            guard oldValue != nil else { return }
            if newValue == nil || oldValue?.userId != newValue?.userId {
                navigator.reset()
            }
        }
        .toastOverlay(toast)
    }

    /// 相机回调先落到状态再关页，避免 sheet 关闭与回调时序互相覆盖。
    @MainActor
    private func handleScan(_ outcome: ScanOutcome) async {
        guard let result = try? await QuickScanFlow.shared.start(scan: { outcome }) else {
            return
        }
        switch result {
        case .login:
            navigator.openLogin()
        case .confirm:
            navigator.open(.scanConfirm)
        case .error(let message):
            toast.show(message ?? "扫码失败")
        case .mismatch:
            toast.show("请扫描指定对手的匹配码")
        case .spectate:
            // 观赛详情尚未迁移：关闭扫码页即可。
            break
        case .busy, .cancelled, .stale:
            break
        }
    }

    private var selectedTabBinding: Binding<MainTab> {
        Binding(
            get: { navigator.selectedTab },
            set: { navigator.select($0, isSignedIn: session.identity != nil) }
        )
    }

    private var sessionExpiredBinding: Binding<Bool> {
        Binding(
            get: { session.sessionExpiredNotice != nil },
            set: { presented in
                if !presented { session.acknowledgeSessionExpired() }
            }
        )
    }

    private func pathBinding(for tab: MainTab) -> Binding<NavigationPath> {
        Binding(
            get: { navigator.path(for: tab) },
            set: { navigator.setPath($0, for: tab) }
        )
    }
}
