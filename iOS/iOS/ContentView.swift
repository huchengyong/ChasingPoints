import SwiftUI

struct ContentView: View {
    @AppStorage("theme_mode") private var themeMode = "system"

    var body: some View {
        MainTabView()
            .preferredColorScheme(ThemeMode(rawValue: themeMode)?.colorScheme)
    }
}
