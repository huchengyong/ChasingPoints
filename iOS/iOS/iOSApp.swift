//
//  iOSApp.swift
//  iOS
//
//  Created by WiseSearch on 2026/9/22.
//

import SwiftUI

@main
struct iOSApp: App {
    @StateObject private var session = SessionStore()

    var body: some Scene {
        WindowGroup {
            ContentView()
                .environmentObject(session)
                .task { await session.restore() }
        }
    }
}
