import SwiftUI

/// 五个主 Tab：首页 / 赛讯 / 扫码 / 消息 / 我的；扫码只触发流程，不切换 Tab。
struct MainTabView: View {
    @EnvironmentObject private var session: SessionStore
    @EnvironmentObject private var navigator: AppNavigator
    @EnvironmentObject private var toast: ToastCenter
    @EnvironmentObject private var activity: ActivityStore
    @Environment(\.colorScheme) private var colorScheme
    @Environment(\.scenePhase) private var scenePhase

    private var palette: AppPalette { AppTheme.palette(for: colorScheme) }

    private var messageBadge: Text? {
        switch activity.messageBadge {
        case .none: nil
        case .dot: Text("")
        case .text(let value): Text(value)
        }
    }

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
                NewsHomeView(session: session)
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
                MessageCenterView(session: session, activity: activity)
                    .navigationDestination(for: AppRoute.self) { AppRouteDestination(route: $0) }
            }
            .tabItem {
                Image(navigator.selectedTab == .messages ? "TabMessageSelected" : "TabMessage")
                    .renderingMode(.original)
                Text(MainTab.messages.title)
            }
            .badge(messageBadge)
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
        .sheet(isPresented: $navigator.isScanPresented) {
            NavigationStack {
                MigrationPlaceholderView(title: "扫码")
                    .toolbar {
                        ToolbarItem(placement: .topBarTrailing) {
                            Button("关闭") { navigator.isScanPresented = false }
                        }
                    }
            }
        }
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
        .onAppear {
            Task { await activity.refreshIfNeeded() }
        }
        .onChange(of: scenePhase) { _, phase in
            if phase == .active {
                Task { await activity.refreshIfNeeded() }
            }
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
