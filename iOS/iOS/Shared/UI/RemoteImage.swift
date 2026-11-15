import SwiftUI

/// 远程图片：地址为空或加载失败时保持占位内容。
struct RemoteImage<Placeholder: View>: View {
    private let url: URL?
    private let contentMode: ContentMode
    private let placeholder: () -> Placeholder

    init(
        url: URL?,
        contentMode: ContentMode = .fill,
        @ViewBuilder placeholder: @escaping () -> Placeholder
    ) {
        self.url = url
        self.contentMode = contentMode
        self.placeholder = placeholder
    }

    var body: some View {
        if let url {
            AsyncImage(url: url) { phase in
                if case .success(let image) = phase {
                    image
                        .resizable()
                        .aspectRatio(contentMode: contentMode)
                } else {
                    placeholder()
                }
            }
        } else {
            placeholder()
        }
    }
}
