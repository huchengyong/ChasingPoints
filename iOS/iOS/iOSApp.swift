//
//  iOSApp.swift
//  iOS
//
//  Created by WiseSearch on 2026/9/22.
//

import Combine
import Foundation
import SwiftUI

@main
struct iOSApp: App {
    var body: some Scene {
        WindowGroup {
            if ProcessInfo.processInfo.environment["CHASING_POINTS_AUTH_TEST_HOST"] == "1" {
                Color.clear
            } else {
                ApplicationRootView()
            }
        }
    }
}

/// 只构造一次的应用对象；视图结构体被重新初始化时不会重复读 Keychain。
private final class AppDependencies: ObservableObject {
    let session: SessionStore
    let activity: ActivityStore
    let realtime: AppRealtime
    let gear: GearStore

    init() {
        let session = SessionStore()
        let activity = ActivityStore(session: session)
        self.session = session
        self.activity = activity
        self.realtime = AppRealtime(session: session, activity: activity)
        self.gear = GearStore(session: session)
    }
}

private struct ApplicationRootView: View {
    @StateObject private var dependencies = AppDependencies()
    @StateObject private var navigator = AppNavigator()
    @StateObject private var toast = ToastCenter()
    @Environment(\.scenePhase) private var scenePhase

    var body: some View {
        ContentView()
            .environmentObject(dependencies.session)
            .environmentObject(dependencies.activity)
            .environmentObject(dependencies.realtime)
            .environmentObject(dependencies.gear)
            .environmentObject(navigator)
            .environmentObject(toast)
            .task { await dependencies.session.restore() }
            .onAppear {
                dependencies.realtime.startNetworkMonitoring()
                dependencies.realtime.setForeground(scenePhase == .active)
            }
            .onChange(of: scenePhase) { _, phase in
                dependencies.realtime.setForeground(phase == .active)
            }
            .onChange(of: dependencies.session.identity) { _, _ in
                dependencies.activity.syncIdentity()
                dependencies.gear.syncIdentity()
                Task {
                    await dependencies.realtime.handleIdentityChange()
                    await dependencies.activity.refreshIfNeeded()
                }
            }
            .onOpenURL { url in
                openEntryLink(url.absoluteString)
            }
            .onContinueUserActivity(NSUserActivityTypeBrowsingWeb) { activity in
                guard let url = activity.webpageURL else { return }
                openEntryLink(url.absoluteString)
            }
    }

    /// 只有入口链接会改变页面；其他链接交给系统默认处理。
    private func openEntryLink(_ url: String) {
        guard let route = LinkEntryRouting.route(for: url) else { return }
        navigator.open(route)
    }
}
