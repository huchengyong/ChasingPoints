import SwiftUI

/// 赛讯列表与详情共用的状态徽标、国旗与球员头像。
struct SaiXunStatusBadge: View {
    @Environment(\.colorScheme) private var colorScheme

    let tone: SaiXunStatusTone
    let label: String
    var showDot = false

    var body: some View {
        let palette = AppTheme.palette(for: colorScheme)
        let colors = colors(palette)

        HStack(spacing: 4) {
            if showDot {
                Circle()
                    .fill(colors.foreground)
                    .frame(width: 5, height: 5)
            }
            Text(label)
                .font(.system(size: 10, weight: .semibold))
                .lineLimit(1)
        }
        .foregroundStyle(colors.foreground)
        .padding(.horizontal, 7)
        .frame(height: 20)
        .background(colors.background, in: Capsule())
    }

    private func colors(_ palette: AppPalette) -> (foreground: Color, background: Color) {
        let tint = colorScheme == .dark ? 0.16 : 0.12
        switch tone {
        case .live: return (palette.success, palette.success.opacity(tint))
        case .upcoming: return (palette.info, palette.info.opacity(tint))
        case .ended: return (palette.textMuted, palette.subtle)
        case .canceled: return (AppTheme.danger, AppTheme.danger.opacity(tint))
        }
    }
}

/// 国旗：英格兰 / 苏格兰 / 威尔士 / 北爱尔兰用本地图片，其余用 emoji。
struct SaiXunFlagView: View {
    let flag: SaiXunFlag

    var body: some View {
        switch flag.type {
        case .image:
            if let name = Self.assetName(flag.value) {
                Image(name)
                    .resizable()
                    .aspectRatio(contentMode: .fit)
                    .frame(width: 16, height: 11)
            }
        case .emoji:
            Text(flag.value)
                .font(.system(size: 13))
        case .none:
            EmptyView()
        }
    }

    private static func assetName(_ path: String) -> String? {
        switch path {
        case "/static/flags/eng.png": "FlagENG"
        case "/static/flags/sco.png": "FlagSCO"
        case "/static/flags/wls.png": "FlagWLS"
        case "/static/flags/nir.png": "FlagNIR"
        default: nil
        }
    }
}

/// 球员头像：远程头像；官方球员缺头像时用 WST 剪影占位；其他用默认球头像。
struct SaiXunPlayerAvatar: View {
    @Environment(\.colorScheme) private var colorScheme

    let source: SaiXunImageSource
    let size: CGFloat

    var body: some View {
        let palette = AppTheme.palette(for: colorScheme)

        Group {
            switch source {
            case .remote(let url):
                RemoteImage(url: url) { Color.clear }
            case .officialPlayerPlaceholder:
                Image("WSTPlayerPlaceholder")
                    .resizable()
                    .scaledToFill()
            case .defaultAvatar:
                Image("DefaultAvatar01")
                    .resizable()
                    .scaledToFill()
            }
        }
        .frame(width: size, height: size)
        .background(palette.subtle)
        .clipShape(Circle())
        .accessibilityHidden(true)
    }
}
