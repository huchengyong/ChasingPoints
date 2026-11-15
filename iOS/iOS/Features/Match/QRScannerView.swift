import AVFoundation
import SwiftUI
import UIKit

enum QRScannerState: Equatable {
    case preparing
    case scanning
    case permissionDenied
    case unsupported
}

/// 相机扫码页：只识别二维码，扫到首个结果就停止并只回调一次。
/// `previewState` 只用于快照测试，传入后不启动相机。
struct QRScannerView: View {
    @Environment(\.colorScheme) private var colorScheme
    @State private var scanner = QRCodeScanner()
    @State private var state: QRScannerState
    @State private var manualCode = ""
    @State private var isTorchOn = false

    private let previewState: QRScannerState?
    private let onResult: (ScanOutcome) -> Void

    init(previewState: QRScannerState? = nil, onResult: @escaping (ScanOutcome) -> Void = { _ in }) {
        self.previewState = previewState
        self.onResult = onResult
        _state = State(initialValue: previewState ?? .preparing)
    }

    var body: some View {
        ZStack {
            Color.black.ignoresSafeArea()

            switch state {
            case .preparing:
                ProgressView()
                    .tint(.white)
            case .scanning:
                CameraPreview(session: scanner.session)
                    .ignoresSafeArea()
                ScannerOverlay()
                VStack {
                    topBar
                    Spacer()
                }
            case .permissionDenied:
                stateBlock(
                    title: "需要相机权限才能扫码",
                    detail: "请到系统设置中开启相机权限后重试。",
                    actionTitle: "去设置",
                    action: openSettings
                )
            case .unsupported:
                stateBlock(
                    title: "当前端暂不支持扫码",
                    detail: "请使用支持相机的设备扫码。",
                    actionTitle: "关闭",
                    showsManualInput: true,
                    action: { onResult(.cancelled) }
                )
            }
        }
        .task { await prepare() }
        .onDisappear { scanner.stop() }
        .statusBarHidden()
    }

    private var topBar: some View {
        HStack {
            Button {
                onResult(.cancelled)
            } label: {
                Image(systemName: "xmark")
                    .font(.system(size: 16, weight: .semibold))
                    .foregroundStyle(.white)
                    .frame(width: 40, height: 40)
                    .background(.black.opacity(0.45), in: Circle())
            }
            .buttonStyle(.plain)
            .accessibilityLabel("关闭")

            Spacer()

            Button {
                isTorchOn = scanner.toggleTorch()
            } label: {
                Image(systemName: isTorchOn ? "bolt.fill" : "bolt.slash")
                    .font(.system(size: 16, weight: .semibold))
                    .foregroundStyle(.white)
                    .frame(width: 40, height: 40)
                    .background(.black.opacity(0.45), in: Circle())
            }
            .buttonStyle(.plain)
            .accessibilityLabel(isTorchOn ? "关闭手电筒" : "打开手电筒")
        }
        .padding(.horizontal, 16)
        .padding(.top, 12)
    }

    #if DEBUG
    /// 模拟器没有相机：允许手动粘贴二维码内容，便于联调。
    private var manualInput: some View {
        VStack(spacing: 12) {
            TextField("粘贴二维码内容", text: $manualCode)
                .textFieldStyle(.plain)
                .textInputAutocapitalization(.never)
                .autocorrectionDisabled()
                .foregroundStyle(.white)
                .padding(12)
                .background(.white.opacity(0.14), in: RoundedRectangle(cornerRadius: 12))
                .background {
                    RoundedRectangle(cornerRadius: 12)
                        .strokeBorder(.white.opacity(0.35), lineWidth: 1)
                }

            Button("确认") {
                let value = manualCode.trimmingCharacters(in: .whitespacesAndNewlines)
                guard !value.isEmpty else { return }
                onResult(.scanned(value))
            }
            .buttonStyle(.appPill)
        }
    }
    #endif

    private func stateBlock(
        title: String,
        detail: String,
        actionTitle: String,
        showsManualInput: Bool = false,
        action: @escaping () -> Void
    ) -> some View {
        VStack(spacing: 14) {
            Image(systemName: "camera.fill")
                .font(.system(size: 40))
                .foregroundStyle(.white.opacity(0.6))
                .accessibilityHidden(true)

            Text(title)
                .font(.system(size: 17, weight: .semibold))
                .foregroundStyle(.white)

            Text(detail)
                .font(.system(size: 14))
                .foregroundStyle(.white.opacity(0.72))
                .multilineTextAlignment(.center)

            #if DEBUG
            if showsManualInput {
                manualInput
            }
            #endif

            Button(actionTitle, action: action)
                .buttonStyle(.appPill)
        }
        .padding(24)
        .frame(maxWidth: .infinity)
    }

    private func prepare() async {
        guard previewState == nil else { return }

        let granted: Bool
        switch AVCaptureDevice.authorizationStatus(for: .video) {
        case .authorized:
            granted = true
        case .notDetermined:
            granted = await AVCaptureDevice.requestAccess(for: .video)
        default:
            granted = false
        }
        guard granted else {
            state = .permissionDenied
            return
        }
        guard scanner.isAvailable else {
            state = .unsupported
            return
        }

        do {
            try scanner.configure { code in
                onResult(.scanned(code))
            }
            state = .scanning
            scanner.start()
        } catch {
            state = .unsupported
        }
    }

    private func openSettings() {
        guard let url = URL(string: UIApplication.openSettingsURLString) else { return }
        UIApplication.shared.open(url)
    }
}

private struct ScannerOverlay: View {
    var body: some View {
        VStack(spacing: 20) {
            RoundedRectangle(cornerRadius: 18)
                .stroke(.white, lineWidth: 2)
                .frame(width: 240, height: 240)
            Text("将二维码放入框内")
                .font(.system(size: 14))
                .foregroundStyle(.white.opacity(0.82))
        }
        .accessibilityHidden(true)
    }
}

/// 相机预览层：`AVCaptureVideoPreviewLayer` 作为 UIView 的 backing layer。
private struct CameraPreview: UIViewRepresentable {
    let session: AVCaptureSession

    func makeUIView(context: Context) -> PreviewView {
        let view = PreviewView()
        view.videoPreviewLayer.session = session
        view.videoPreviewLayer.videoGravity = .resizeAspectFill
        return view
    }

    func updateUIView(_ uiView: PreviewView, context: Context) {}

    final class PreviewView: UIView {
        override class var layerClass: AnyClass { AVCaptureVideoPreviewLayer.self }
        var videoPreviewLayer: AVCaptureVideoPreviewLayer { layer as! AVCaptureVideoPreviewLayer }
    }
}

private enum QRScannerError: Error {
    case unsupported
}

/// `AVCaptureSession` 的薄封装：配置、启停与手电筒。
final class QRCodeScanner: NSObject, AVCaptureMetadataOutputObjectsDelegate {
    let session = AVCaptureSession()

    private let queue = DispatchQueue(label: "com.chasingpoints.qr-scanner")
    private var device: AVCaptureDevice?
    private var onCode: ((String) -> Void)?
    private var hasResult = false

    var isAvailable: Bool { AVCaptureDevice.default(for: .video) != nil }

    func configure(onCode: @escaping (String) -> Void) throws {
        guard let device = AVCaptureDevice.default(for: .video) else {
            throw QRScannerError.unsupported
        }
        let input = try AVCaptureDeviceInput(device: device)
        let output = AVCaptureMetadataOutput()

        session.beginConfiguration()
        defer { session.commitConfiguration() }

        guard session.canAddInput(input), session.canAddOutput(output) else {
            throw QRScannerError.unsupported
        }
        session.addInput(input)
        session.addOutput(output)
        output.setMetadataObjectsDelegate(self, queue: queue)
        output.metadataObjectTypes = [.qr]

        self.device = device
        self.onCode = onCode
    }

    func start() {
        let session = session
        queue.async {
            guard !session.isRunning else { return }
            session.startRunning()
        }
    }

    func stop() {
        let session = session
        queue.async {
            guard session.isRunning else { return }
            session.stopRunning()
        }
    }

    @discardableResult
    func toggleTorch() -> Bool {
        guard let device, device.hasTorch, (try? device.lockForConfiguration()) != nil else { return false }
        defer { device.unlockForConfiguration() }
        device.torchMode = device.torchMode == .on ? .off : .on
        return device.torchMode == .on
    }

    func metadataOutput(
        _ output: AVCaptureMetadataOutput,
        didOutput metadataObjects: [AVMetadataObject],
        from connection: AVCaptureConnection
    ) {
        guard !hasResult,
              let object = metadataObjects.first as? AVMetadataMachineReadableCodeObject,
              let value = object.stringValue,
              !value.isEmpty else { return }
        hasResult = true
        stop()
        let onCode = onCode
        DispatchQueue.main.async {
            onCode?(value)
        }
    }
}
