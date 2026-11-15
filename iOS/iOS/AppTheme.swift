import SwiftUI

extension Color {
    /// 设计规范以十六进制给色，避免逐通道换算带来的偏差。
    init(hex: UInt32, opacity: Double = 1) {
        self.init(
            .sRGB,
            red: Double((hex >> 16) & 0xFF) / 255,
            green: Double((hex >> 8) & 0xFF) / 255,
            blue: Double(hex & 0xFF) / 255,
            opacity: opacity
        )
    }
}

struct AppPalette {
    let page: Color
    let card: Color
    let subtle: Color
    let textPrimary: Color
    let textSecondary: Color
    let textMuted: Color
    let border: Color
    let brandStrong: Color
    let brandTint: Color
    let success: Color
    let info: Color
    let gradientStart: Color
    let gradientEnd: Color
    let heroColors: [Color]
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
    static let warning = Color(hex: 0xF97316)

    static func palette(for scheme: ColorScheme) -> AppPalette {
        if scheme == .dark {
            return AppPalette(
                page: Color(red: 20.0 / 255, green: 17.0 / 255, blue: 9.0 / 255),
                card: Color(red: 30.0 / 255, green: 24.0 / 255, blue: 13.0 / 255),
                subtle: Color(red: 36.0 / 255, green: 29.0 / 255, blue: 16.0 / 255),
                textPrimary: Color(red: 255.0 / 255, green: 247.0 / 255, blue: 225.0 / 255),
                textSecondary: Color(red: 215.0 / 255, green: 200.0 / 255, blue: 155.0 / 255),
                textMuted: Color(hex: 0x9F926E),
                border: Color(red: 58.0 / 255, green: 46.0 / 255, blue: 22.0 / 255),
                brandStrong: Color(hex: 0xF0C542),
                brandTint: Color(hex: 0xE0AE12, opacity: 0.20),
                success: Color(hex: 0x22C55E),
                info: Color(hex: 0x60A5FA),
                gradientStart: Color(hex: 0xE0AE12),
                gradientEnd: Color(hex: 0xA97500),
                heroColors: [Color(hex: 0x1E180D), Color(hex: 0x241D10)]
            )
        }

        return AppPalette(
            page: Color(red: 247.0 / 255, green: 244.0 / 255, blue: 236.0 / 255),
            card: .white,
            subtle: Color(red: 250.0 / 255, green: 248.0 / 255, blue: 242.0 / 255),
            textPrimary: Color(red: 35.0 / 255, green: 28.0 / 255, blue: 11.0 / 255),
            textSecondary: Color(red: 110.0 / 255, green: 98.0 / 255, blue: 66.0 / 255),
            textMuted: Color(hex: 0x9A8C67),
            border: Color(red: 233.0 / 255, green: 226.0 / 255, blue: 207.0 / 255),
            brandStrong: Color(hex: 0xC69200),
            brandTint: Color(hex: 0xE0AE12, opacity: 0.14),
            success: Color(hex: 0x18B05B),
            info: Color(hex: 0x3B82F6),
            gradientStart: Color(hex: 0xD9A617),
            gradientEnd: Color(hex: 0xBB8400),
            heroColors: [Color(hex: 0x3C2B08), Color(hex: 0x8F6507), Color(hex: 0xC69200)]
        )
    }
}
