//
//  iOSApp.swift
//  iOS
//
//  Created by WiseSearch on 2026/9/22.
//

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

private struct ApplicationRootView: View {
    @StateObject private var session: SessionStore
    @StateObject private var activity: ActivityStore
    @StateObject private var realtime: AppRealtime
    @StateObject private var navigator = AppNavigator()
    @StateObject private var toast = ToastCenter()
    @Environment(\.scenePhase) private var scenePhase

    private let networkMonitor: NetworkStatusMonitor

    init() {
        let session = SessionStore()
        let activity = ActivityStore(session: session)
        _session = StateObject(wrappedValue: session)
        _activity = StateObject(wrappedValue: activity)
        _realtime = StateObject(wrappedValue: AppRealtime(session: session, activity: activity))
        networkMonitor = NWPathNetworkMonitor()
    }

    var body: some View {
        ContentView()
            .environmentObject(session)
            .environmentObject(activity)
            .environmentObject(realtime)
            .environmentObject(navigator)
            .environmentObject(toast)
            .task { await session.restore() }
            .onAppear {
                networkMonitor.onStatusChange = { online in
                    realtime.setNetworkOnline(online)
                }
                networkMonitor.start()
                realtime.setForeground(scenePhase == .active)
            }
            .onChange(of: scenePhase) { _, phase in
                realtime.setForeground(phase == .active)
            }
            .onChange(of: session.identity) { _, _ in
                activity.syncIdentity()
                Task {
                    await realtime.handleIdentityChange()
                    await activity.refreshIfNeeded()
                }
            }
    }
}
