import Combine
import SwiftUI
import UIKit

/// 「出示我的匹配二维码」弹层的读取状态。
@MainActor
final class MatchQRCodeModel: ObservableObject {
    enum Phase: Equatable {
        case loading
        case failed
        case ready(UIImage)
    }

    @Published private(set) var phase: Phase = .loading

    private let session: SessionStore
    private var requestToken = 0

    init(session: SessionStore) {
        self.session = session
    }

    /// 读取二维码：弹层关闭或身份变化后到达的结果一律丢弃。
    func load() async {
        requestToken += 1
        let token = requestToken
        let requestOwner = session.identity
        phase = .loading

        do {
            let code = try await ProfileAPI(session: session).matchQRCode()
            guard token == requestToken, session.identity == requestOwner else { return }
            guard let image = QRCodeImage.make(content: code.qrcodeData, size: 240) else {
                phase = .failed
                return
            }
            phase = .ready(image)
        } catch {
            guard token == requestToken, session.identity == requestOwner else { return }
            phase = .failed
        }
    }

    /// 关闭弹层：作废在途结果并清掉已生成的二维码。
    func reset() {
        requestToken += 1
        phase = .loading
    }
}

struct MatchQRCodeSheet: View {
    @Environment(\.dismiss) private var dismiss
    @StateObject private var model: MatchQRCodeModel

    init(session: SessionStore) {
        _model = StateObject(wrappedValue: MatchQRCodeModel(session: session))
    }

    var body: some View {
        MatchQRCodeContent(model: model) { close() }
            .presentationDetents([.medium])
            .task { await model.load() }
            .onDisappear { model.reset() }
    }

    private func close() {
        model.reset()
        dismiss()
    }
}

struct MatchQRCodeContent: View {
    @ObservedObject var model: MatchQRCodeModel
    let onClose: () -> Void

    @Environment(\.colorScheme) private var colorScheme

    private var palette: AppPalette { AppTheme.palette(for: colorScheme) }

    var body: some View {
        VStack(spacing: 0) {
            HStack(alignment: .top, spacing: 10) {
                VStack(alignment: .leading, spacing: 4) {
                    Text("出示我的匹配二维码")
                        .font(.system(size: 16, weight: .bold))
                        .foregroundStyle(palette.textPrimary)
                    Text("请让对手使用发起对局扫码入口识别")
                        .font(.system(size: 12))
                        .foregroundStyle(palette.textSecondary)
                }
                Spacer(minLength: 8)
                Button(action: onClose) {
                    Image(systemName: "xmark")
                        .font(.system(size: 12, weight: .semibold))
                        .foregroundStyle(palette.textSecondary)
                        .frame(width: 32, height: 32)
                        .background(palette.subtle, in: Circle())
                }
                .buttonStyle(.plain)
                .accessibilityLabel("关闭")
            }

            ZStack {
                RoundedRectangle(cornerRadius: 12).fill(palette.subtle)
                switch model.phase {
                case .loading:
                    VStack(spacing: 10) {
                        ProgressView()
                        Text("二维码生成中...")
                            .font(.system(size: 13))
                            .foregroundStyle(palette.textSecondary)
                    }
                case .failed:
                    VStack(spacing: 10) {
                        Text("二维码加载失败")
                            .font(.system(size: 13))
                            .foregroundStyle(AppTheme.danger)
                        Button {
                            Task { await model.load() }
                        } label: {
                            Text("重试")
                                .font(.system(size: 13, weight: .semibold))
                                .foregroundStyle(palette.brandStrong)
                                .padding(.horizontal, 14)
                                .frame(height: 34)
                                .background(palette.brandTint, in: Capsule())
                        }
                        .buttonStyle(.plain)
                    }
                case .ready(let image):
                    // 二维码必须保持白底黑码，暗色主题下也一样，否则扫不出来。
                    Image(uiImage: image)
                        .interpolation(.none)
                        .resizable()
                        .frame(width: 240, height: 240)
                        .padding(8)
                        .background(Color.white, in: RoundedRectangle(cornerRadius: 8))
                }
            }
            .frame(maxWidth: .infinity)
            .frame(minHeight: 240)
            .padding(.vertical, 14)

            Button(action: onClose) {
                Text("关闭")
                    .font(.system(size: 13, weight: .semibold))
                    .foregroundStyle(.white)
                    .frame(maxWidth: .infinity)
                    .frame(height: 40)
                    .background(AppTheme.primary, in: Capsule())
            }
            .buttonStyle(.plain)
        }
        .padding(16)
        .background(palette.card)
    }
}
