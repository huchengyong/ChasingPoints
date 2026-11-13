import UIKit
import XCTest
@testable import iOS

@MainActor
final class MemberIdentityTests: XCTestCase {
    func testGrowthLevelsMapOntoThreeTiers() {
        let identities = (1...5).map { MemberIdentity.resolve(vipLevel: $0, rankLevel: 0) }

        XCTAssertEqual(identities.map(\.tier), ["standard", "standard", "advanced", "advanced", "premium"])
        XCTAssertEqual(identities.map(\.tierLabel), ["标准", "标准", "进阶", "进阶", "尊享"])
        XCTAssertEqual(identities.map(\.badgeText), ["VIP1", "VIP2", "VIP3", "VIP4", "VIP5"])
        XCTAssertTrue(identities.allSatisfy(\.showBadge))
    }

    func testMissingOrInvalidVipLevelHasNoMemberIdentity() {
        for vipLevel in [0, -1] {
            let identity = MemberIdentity.resolve(vipLevel: vipLevel, rankLevel: 6)
            XCTAssertFalse(identity.showBadge, "vipLevel=\(vipLevel)")
            XCTAssertEqual(identity.vipLevel, 0)
            XCTAssertEqual(identity.badgeText, "")
            XCTAssertEqual(identity.tier, "")
            XCTAssertNil(identity.frameURL, "非会员即使有段位也没有头像框")
        }
    }

    func testLevelsAboveMaximumBecomeTopTier() {
        let identity = MemberIdentity.resolve(vipLevel: 9, rankLevel: 0)
        XCTAssertEqual(identity.vipLevel, 5)
        XCTAssertEqual(identity.badgeText, "VIP5")
    }

    func testFrameFollowsRankLevelAndIsNilWhenUnknown() {
        XCTAssertEqual(
            MemberIdentity.resolve(vipLevel: 3, rankLevel: 4).frameURL?.absoluteString,
            "https://cdn.zhuifen.cn/static/images/avatar-frames/frame_platinum.png"
        )

        for rankLevel in [0, 7] {
            let identity = MemberIdentity.resolve(vipLevel: 3, rankLevel: rankLevel)
            XCTAssertNil(identity.frameURL, "rankLevel=\(rankLevel)")
            XCTAssertTrue(identity.showBadge, "没有段位的会员仍显示标识")
        }
    }

    func testAllSixRankFramesAreDistinctCDNAssets() {
        let urls = (1...6).compactMap { MemberIdentity.resolve(vipLevel: 3, rankLevel: $0).frameURL?.absoluteString }

        XCTAssertEqual(urls.count, 6)
        XCTAssertEqual(Set(urls).count, 6)
        XCTAssertTrue(urls.allSatisfy { $0.hasPrefix("https://cdn.zhuifen.cn/static/images/avatar-frames/") })
    }

    func testAvatarMetricsScaleFrameAndBadgeWithAvatar() {
        let metrics = MemberAvatarMetrics.resolve(size: 96)

        XCTAssertEqual(metrics.avatarSize, 96)
        XCTAssertEqual(metrics.boxSize, 96)
        XCTAssertEqual(metrics.frameSize, 156)
        XCTAssertEqual(metrics.badgeFontSize, 13)
        XCTAssertEqual(metrics.badgeHeight, 20)
        XCTAssertEqual(metrics.badgePaddingX, 6)
    }

    func testReserveFrameKeepsAvatarAndReservesBox() {
        let metrics = MemberAvatarMetrics.resolve(size: 96, reserveFrame: true)

        XCTAssertEqual(metrics.avatarSize, 96)
        XCTAssertEqual(metrics.boxSize, 124)
    }

    func testFitFrameShrinksAvatarIntoRequestedFootprint() {
        let fitted = MemberAvatarMetrics.resolve(size: 224, fitFrame: true, hasFrame: true)

        XCTAssertEqual(fitted.avatarSize, 138)
        XCTAssertEqual(fitted.boxSize, 224)
        XCTAssertEqual(fitted.frameSize, 224)
        XCTAssertLessThanOrEqual(fitted.avatarSize * 1.625, fitted.frameSize + 1, "头像框不得溢出给定占位")

        let unframed = MemberAvatarMetrics.resolve(size: 224, fitFrame: true, hasFrame: false)
        XCTAssertEqual(unframed.avatarSize, 224)
        XCTAssertEqual(unframed.boxSize, 224)
        XCTAssertEqual(unframed.frameSize, 364)
    }

    func testBadgeFontSizeIsClamped() {
        XCTAssertEqual(MemberAvatarMetrics.resolve(size: 40).badgeFontSize, 9)
        XCTAssertEqual(MemberAvatarMetrics.resolve(size: 240).badgeFontSize, 13)
        XCTAssertEqual(MemberAvatarMetrics.resolve(size: 0).avatarSize, 0)
    }

    func testDefaultAvatarAssetNameFollowsUserId() {
        XCTAssertEqual(DefaultAvatar.assetName(userId: 1), "DefaultAvatar01")
        XCTAssertEqual(DefaultAvatar.assetName(userId: 7), "DefaultAvatar07")
        XCTAssertEqual(DefaultAvatar.assetName(userId: 15), "DefaultAvatar15")
        XCTAssertEqual(DefaultAvatar.assetName(userId: 16), "DefaultAvatar01")
        XCTAssertEqual(DefaultAvatar.assetName(userId: 0), "DefaultAvatar01")
        XCTAssertEqual(DefaultAvatar.assetName(userId: -3), "DefaultAvatar01")
    }

    func testDefaultAvatarAssetsAreBundled() {
        for index in 1...15 {
            let name = String(format: "DefaultAvatar%02d", index)
            XCTAssertNotNil(UIImage(named: name), "缺少头像资源 \(name)")
        }
    }
}
