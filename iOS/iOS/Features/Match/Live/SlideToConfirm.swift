import SwiftUI

/// 逐条对应小程序 `utils/slide-to-confirm.js`：从左端滑块拖到轨道终点并松手才算完成。
enum SlideToConfirm {
    static let completeRatio = 0.98

    static func progress(deltaX: CGFloat, maxTrack: CGFloat) -> Double {
        guard deltaX.isFinite, maxTrack.isFinite, maxTrack > 0 else { return 0 }
        let clamped = min(max(deltaX, 0), maxTrack)
        return Double(clamped / maxTrack)
    }

    /// 允许 1–2px 误差。
    static func isComplete(deltaX: CGFloat, maxTrack: CGFloat) -> Bool {
        progress(deltaX: deltaX, maxTrack: maxTrack) >= completeRatio
    }

    static func shouldCommitOnRelease(deltaX: CGFloat, maxTrack: CGFloat) -> Bool {
        isComplete(deltaX: deltaX, maxTrack: maxTrack)
    }
}

/// 滑动确认控件；VoiceOver 下整块作为按钮，双击即完成确认。
struct SlideToConfirmView: View {
    let label: String
    let accessibilityLabelText: String
    var isEnabled = true
    let onCommit: () -> Void

    @Environment(\.colorScheme) private var colorScheme
    @State private var dragOffset: CGFloat = 0

    private let thumbSize: CGFloat = 52

    var body: some View {
        let palette = AppTheme.palette(for: colorScheme)

        GeometryReader { proxy in
            let maxTrack = max(0, proxy.size.width - thumbSize)
            ZStack(alignment: .leading) {
                Capsule().fill(palette.subtle)
                Capsule().stroke(palette.border)
                Text(label)
                    .font(.system(size: 13, weight: .medium))
                    .foregroundStyle(palette.textSecondary)
                    .frame(maxWidth: .infinity)
                    .padding(.leading, thumbSize)
                Circle()
                    .fill(AppTheme.primary)
                    .frame(width: thumbSize, height: thumbSize)
                    .overlay {
                        Image(systemName: "arrow.right")
                            .font(.system(size: 16, weight: .bold))
                            .foregroundStyle(.white)
                    }
                    .offset(x: min(max(dragOffset, 0), maxTrack))
                    .gesture(
                        DragGesture(minimumDistance: 1)
                            .onChanged { value in
                                guard isEnabled else { return }
                                dragOffset = min(max(value.translation.width, 0), maxTrack)
                            }
                            .onEnded { _ in
                                guard isEnabled else { return }
                                if SlideToConfirm.shouldCommitOnRelease(deltaX: dragOffset, maxTrack: maxTrack) {
                                    onCommit()
                                }
                                withAnimation(.easeOut(duration: 0.18)) { dragOffset = 0 }
                            }
                    )
            }
        }
        .frame(height: thumbSize)
        .opacity(isEnabled ? 1 : 0.5)
        .accessibilityElement(children: .ignore)
        .accessibilityLabel(accessibilityLabelText)
        .accessibilityAddTraits(.isButton)
        .accessibilityAction {
            guard isEnabled else { return }
            onCommit()
        }
    }
}
