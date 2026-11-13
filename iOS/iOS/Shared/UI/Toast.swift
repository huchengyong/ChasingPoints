import Combine
import SwiftUI

/// 轻提示，对应小程序的 uni.showToast({ icon: 'none' })。
@MainActor
final class ToastCenter: ObservableObject {
    @Published private(set) var message: String?

    private let duration: TimeInterval
    private var dismissTask: Task<Void, Never>?

    init(duration: TimeInterval = 2) {
        self.duration = duration
    }

    /// 显示一条提示；新提示替换旧提示并重新计时；空白字符串忽略。
    func show(_ message: String) {
        guard !message.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty else { return }
        self.message = message
        dismissTask?.cancel()
        AccessibilityNotification.Announcement(message).post()
        let duration = self.duration
        dismissTask = Task { [weak self] in
            try? await Task.sleep(nanoseconds: UInt64(max(0, duration) * 1_000_000_000))
            guard !Task.isCancelled else { return }
            self?.message = nil
            self?.dismissTask = nil
        }
    }

    func dismiss() {
        dismissTask?.cancel()
        dismissTask = nil
        message = nil
    }
}

extension View {
    /// 在根视图上叠加 Toast 层。
    func toastOverlay(_ center: ToastCenter) -> some View {
        overlay(alignment: .bottom) {
            ToastLayer(center: center)
        }
    }
}

private struct ToastLayer: View {
    @ObservedObject var center: ToastCenter

    var body: some View {
        ZStack {
            if let message = center.message {
                Text(message)
                    .font(.system(size: 14))
                    .foregroundStyle(.white)
                    .multilineTextAlignment(.center)
                    .lineLimit(2)
                    .padding(.horizontal, 20)
                    .padding(.vertical, 12)
                    .background(Color.black.opacity(0.78), in: Capsule())
                    .padding(.horizontal, 32)
                    .accessibilityHidden(true)
                    .transition(.opacity)
            }
        }
        .padding(.bottom, 96)
        .animation(.easeInOut(duration: 0.2), value: center.message)
        .allowsHitTesting(false)
    }
}
