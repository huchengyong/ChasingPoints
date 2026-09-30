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
    @StateObject private var session = SessionStore()

    var body: some View {
        ContentView()
            .environmentObject(session)
            .task { await session.restore() }
    }
}
