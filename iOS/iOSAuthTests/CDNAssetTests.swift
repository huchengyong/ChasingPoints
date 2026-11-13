import Foundation
import XCTest
@testable import iOS

@MainActor
final class CDNAssetTests: XCTestCase {
    func testCDNPathsMapToTheCDN() {
        let paths = [
            "/static/images/ranks/rank_gold.png",
            "/static/images/achievements/wins_1.png",
            "/static/images/avatar-frames/frame_king.png",
            "/static/images/default-event-cover.png",
            "/static/images/wst-default-event-cover.png",
            "/static/images/gear/gear-cue-pool-default.png",
        ]

        for path in paths {
            XCTAssertEqual(CDNAsset.url(for: path)?.absoluteString, "https://cdn.zhuifen.cn" + path, path)
            XCTAssertTrue(CDNAsset.isCDNPath(path), path)
        }
    }

    func testOtherPathsHaveNoCDNURL() {
        let paths = [
            "/static/logo.png",
            "/static/images/default-avatars/ball-01.png",
            "/static/images/ranks/../logo.png",
            "/static/images/gear/../logo.png",
            "/static/images/gear/sub/a.png",
            "",
        ]

        for path in paths {
            XCTAssertNil(CDNAsset.url(for: path), path)
            XCTAssertFalse(CDNAsset.isCDNPath(path), path)
        }
    }

    func testAbsoluteURLsPassThroughAndAreIdempotent() {
        let absolute = "https://cdn.zhuifen.cn/avatar/1.png"
        XCTAssertEqual(CDNAsset.url(for: absolute)?.absoluteString, absolute)
        XCTAssertEqual(CDNAsset.url(for: "http://example.test/a.png")?.absoluteString, "http://example.test/a.png")

        let resolved = CDNAsset.url(for: "/static/images/ranks/rank_gold.png")
        XCTAssertEqual(resolved.flatMap { CDNAsset.url(for: $0.absoluteString) }, resolved)
    }
}
