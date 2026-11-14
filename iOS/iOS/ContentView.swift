import SwiftUI

struct ContentView: View {
    @EnvironmentObject private var session: SessionStore
    @AppStorage("theme_mode") private var themeMode = "system"

    var body: some View {
        MainTabView(session: session)
            .preferredColorScheme(ThemeMode(rawValue: themeMode)?.colorScheme)
    }
}
