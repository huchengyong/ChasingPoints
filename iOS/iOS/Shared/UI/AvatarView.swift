import SwiftUI

/// 普通头像：有 http(s) 头像地址用远程图，否则用按 userId 取的默认球头像。
struct AvatarView: View {
    private let avatar: String
    private let userId: Int64
    private let size: CGFloat

    init(avatar: String, userId: Int64, size: CGFloat) {
        self.avatar = avatar
        self.userId = userId
        self.size = size
    }

    var body: some View {
        ZStack {
            Image(DefaultAvatar.assetName(userId: userId))
                .resizable()
                .scaledToFill()

            if avatar.hasPrefix("http://") || avatar.hasPrefix("https://"),
               let url = URL(string: avatar) {
                RemoteImage(url: url) {
                    Color.clear
                }
            }
        }
        .frame(width: size, height: size)
        .clipShape(Circle())
        .accessibilityHidden(true)
    }
}
