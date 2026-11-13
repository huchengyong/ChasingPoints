import Foundation
import XCTest
@testable import iOS

@MainActor
final class QRCodeImageTests: XCTestCase {
    func testGeneratedCodeCanBeDetectedBack() throws {
        let content = "https://zhuifen.cn/invite?code=ABC123&from=我的页"
        let image = try XCTUnwrap(QRCodeImage.make(content: content, size: 240))

        XCTAssertGreaterThanOrEqual(image.size.width, 240)
        XCTAssertEqual(image.size.width, image.size.height)

        let ciImage = try XCTUnwrap(CIImage(image: image))
        let detector = try XCTUnwrap(CIDetector(
            ofType: CIDetectorTypeQRCode,
            context: nil,
            options: [CIDetectorAccuracy: CIDetectorAccuracyHigh]
        ))
        let messages = detector.features(in: ciImage).compactMap { ($0 as? CIQRCodeFeature)?.messageString }
        XCTAssertEqual(messages.first, content)
    }

    func testEmptyContentReturnsNil() {
        XCTAssertNil(QRCodeImage.make(content: "", size: 240))
        XCTAssertNil(QRCodeImage.make(content: "https://zhuifen.cn/invite?code=ABC123", size: 0))
    }
}
