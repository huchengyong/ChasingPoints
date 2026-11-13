import Foundation
import XCTest
@testable import iOS

@MainActor
final class SettingsProfileRulesTests: XCTestCase {
    func testFormatSettingsPhoneKeepsPrefixAndMasksMainlandNumber() {
        XCTAssertEqual(SettingsProfileRules.formatSettingsPhone("188****5657"), "+86 188****5657")
        XCTAssertEqual(SettingsProfileRules.formatSettingsPhone("13800138000"), "+86 138****8000")
        XCTAssertEqual(SettingsProfileRules.formatSettingsPhone(""), "")
    }

    func testBuildQiniuAvatarObjectKeyScopesByUserAndDate() {
        let timestamp = ISO8601DateFormatter().date(from: "2026-04-09T08:20:00Z")!
        let key = SettingsProfileRules.buildQiniuAvatarObjectKey(
            userId: 18,
            filePath: "/tmp/avatar.png",
            timestamp: timestamp
        )

        XCTAssertTrue(key.hasPrefix("avatars/18/20260409/"))
        XCTAssertTrue(key.hasSuffix(".png"))
        XCTAssertTrue(key.contains("1775722800000"))
    }

    func testNormalizeQiniuUploadResultJoinsDomainAndKey() {
        XCTAssertEqual(
            SettingsProfileRules.normalizeQiniuUploadResult(
                domain: "https://cdn.example.com/",
                key: "avatars/18/20260409/a.png"
            ),
            "https://cdn.example.com/avatars/18/20260409/a.png"
        )
        XCTAssertEqual(SettingsProfileRules.normalizeQiniuUploadResult(domain: "", key: "a"), "")
        XCTAssertEqual(SettingsProfileRules.normalizeQiniuUploadResult(domain: "https://cdn", key: ""), "")
    }

    func testAvatarCompressionKeepsLongestEdgeWithinLimit() {
        XCTAssertEqual(SettingsProfileRules.avatarImageMaxEdge, 512)
        XCTAssertEqual(SettingsProfileRules.avatarImageQuality, 80)

        XCTAssertEqual(
            SettingsProfileRules.getAvatarCompressionDimensions(width: 2400, height: 1200),
            SettingsProfileRules.AvatarDimensions(width: 512, height: 256)
        )
        XCTAssertEqual(
            SettingsProfileRules.getAvatarCompressionDimensions(width: 1200, height: 2400),
            SettingsProfileRules.AvatarDimensions(width: 256, height: 512)
        )
        XCTAssertEqual(
            SettingsProfileRules.getAvatarCompressionDimensions(width: 400, height: 400),
            SettingsProfileRules.AvatarDimensions(width: 400, height: 400)
        )
        XCTAssertEqual(
            SettingsProfileRules.getAvatarCompressionDimensions(width: 0, height: 0),
            SettingsProfileRules.AvatarDimensions(width: 512, height: 512)
        )
    }

    func testFileExtensionFallsBackToJpg() {
        XCTAssertEqual(SettingsProfileRules.getFileExtensionFromPath("/tmp/a.PNG?x=1"), ".png")
        XCTAssertEqual(SettingsProfileRules.getFileExtensionFromPath("/tmp/avatar"), ".jpg")
        XCTAssertEqual(SettingsProfileRules.getFileExtensionFromPath("/tmp/avatar."), ".jpg")
        XCTAssertEqual(SettingsProfileRules.getFileExtensionFromPath("/tmp/avatar.tar.gz"), ".gz")
    }
}
