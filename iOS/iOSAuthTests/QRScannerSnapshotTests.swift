import SwiftUI
import XCTest
@testable import iOS

/// 扫码页相机不可用时的两种状态；`previewState` 不启动相机，只渲染。
@MainActor
final class QRScannerSnapshotTests: XCTestCase {
    private let size = CGSize(width: 393, height: 852)

    func testPermissionDeniedSnapshot() throws {
        try render("scanner-permission-denied", view: QRScannerView(previewState: .permissionDenied))
    }

    func testUnsupportedSnapshot() throws {
        try render("scanner-unsupported", view: QRScannerView(previewState: .unsupported))
    }

    private func render<Content: View>(_ name: String, view: Content) throws {
        for scheme in [ColorScheme.light, .dark] {
            let image = try renderSnapshot(name, colorScheme: scheme, size: size) {
                view
            }
            XCTAssertEqual(image.size, size)
        }
    }
}
