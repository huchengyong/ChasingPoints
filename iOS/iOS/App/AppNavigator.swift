import Combine
import SwiftUI

enum MainTab: Int, CaseIterable, Hashable {
    case home, news, scan, messages, me

    var title: String {
        switch self {
        case .home: "首页"
        case .news: "赛讯"
        case .scan: "扫码"
        case .messages: "消息"
        case .me: "我的"
        }
    }
}

/// 主 Tab 选中态与各 Tab 的导航栈。扫码不是页面：点它不切换 Tab，只触发扫码流程。
@MainActor
final class AppNavigator: ObservableObject {
    @Published private(set) var selectedTab: MainTab = .home
    @Published var isScanPresented = false
    /// 赛果页「再来一局」带入的上下文，扫码流程取走后立即清空。
    @Published var pendingRematch: ScanRematch?
    @Published private var paths: [MainTab: NavigationPath] = [:]

    func path(for tab: MainTab) -> NavigationPath {
        guard tab != .scan else { return NavigationPath() }
        return paths[tab] ?? NavigationPath()
    }

    func setPath(_ path: NavigationPath, for tab: MainTab) {
        guard tab != .scan else { return }
        paths[tab] = path
    }

    /// 点击 Tab：`.scan` 走 requestScan，其余切换选中态（已选中的 Tab 再点一次回到该 Tab 根页面）。
    func select(_ tab: MainTab, isSignedIn: Bool) {
        guard tab != .scan else {
            requestScan(isSignedIn: isSignedIn)
            return
        }
        if selectedTab == tab {
            popToRoot(tab)
        } else {
            selectedTab = tab
        }
    }

    /// 在当前选中的 Tab 里打开页面。
    func open(_ route: AppRoute) {
        guard selectedTab != .scan else { return }
        var path = paths[selectedTab] ?? NavigationPath()
        path.append(route)
        paths[selectedTab] = path
    }

    /// 切到指定 Tab 并回到根页面后打开页面（用于跨 Tab 跳转）。
    func open(_ route: AppRoute, in tab: MainTab) {
        guard tab != .scan else { return }
        selectedTab = tab
        var path = NavigationPath()
        path.append(route)
        paths[tab] = path
    }

    /// 切到「我的」、回到根页面并打开登录页。
    func openLogin() {
        open(.login, in: .me)
    }

    /// 扫码入口：未登录 → openLogin()；已登录 → isScanPresented = true。
    func requestScan(isSignedIn: Bool) {
        if isSignedIn {
            isScanPresented = true
        } else {
            openLogin()
        }
    }

    func popToRoot(_ tab: MainTab) {
        guard tab != .scan else { return }
        paths[tab] = NavigationPath()
    }

    /// 退出登录、会话失效或切换账号后调用：清空所有 Tab 的导航栈并关闭扫码。
    func reset() {
        paths = [:]
        isScanPresented = false
        pendingRematch = nil
    }
}
