import SwiftUI

struct AppPalette {
    let page: Color
    let card: Color
    let subtle: Color
    let textPrimary: Color
    let textSecondary: Color
    let border: Color
}

enum ThemeMode: String, CaseIterable {
    case system
    case light
    case dark

    var title: String {
        switch self {
        case .system: "跟随系统"
        case .light: "浅色"
        case .dark: "深色"
        }
    }

    var colorScheme: ColorScheme? {
        switch self {
        case .system: nil
        case .light: .light
        case .dark: .dark
        }
    }
}

enum AppTheme {
    static let primary = Color(red: 224.0 / 255, green: 174.0 / 255, blue: 18.0 / 255)
    static let danger = Color(red: 239.0 / 255, green: 68.0 / 255, blue: 68.0 / 255)

    static func palette(for scheme: ColorScheme) -> AppPalette {
        if scheme == .dark {
            return AppPalette(
                page: Color(red: 20.0 / 255, green: 17.0 / 255, blue: 9.0 / 255),
                card: Color(red: 30.0 / 255, green: 24.0 / 255, blue: 13.0 / 255),
                subtle: Color(red: 36.0 / 255, green: 29.0 / 255, blue: 16.0 / 255),
                textPrimary: Color(red: 255.0 / 255, green: 247.0 / 255, blue: 225.0 / 255),
                textSecondary: Color(red: 215.0 / 255, green: 200.0 / 255, blue: 155.0 / 255),
                border: Color(red: 58.0 / 255, green: 46.0 / 255, blue: 22.0 / 255)
            )
        }

        return AppPalette(
            page: Color(red: 247.0 / 255, green: 244.0 / 255, blue: 236.0 / 255),
            card: .white,
            subtle: Color(red: 250.0 / 255, green: 248.0 / 255, blue: 242.0 / 255),
            textPrimary: Color(red: 35.0 / 255, green: 28.0 / 255, blue: 11.0 / 255),
            textSecondary: Color(red: 110.0 / 255, green: 98.0 / 255, blue: 66.0 / 255),
            border: Color(red: 233.0 / 255, green: 226.0 / 255, blue: 207.0 / 255)
        )
    }
}
