import Foundation
import XCTest
@testable import iOS

/// 迁移自 `uniapp/tests/member-center.test.mjs` 与 `member-growth-center.test.mjs` 的非支付用例。
@MainActor
final class MemberCenterRulesTests: XCTestCase {
    private let now = ProfileFixtures.time("2026-11-05T10:00:00+08:00")

    // MARK: - 摘要

    func testComplianceSummaryStaysNeutralBeforeAnyReward() {
        let summary = MemberCenterRules.resolveSummary(
            ProfileFixtures.memberStatusModel(isActive: false, memberExpiresAt: nil),
            now: now,
            complianceMode: true
        )

        XCTAssertEqual(summary.statusText, "暂无权益")
        XCTAssertFalse(summary.description.contains("注册"))
    }

    func testSummaryTreatsExpiryAsUTC8() {
        let summary = MemberCenterRules.resolveSummary(
            ProfileFixtures.memberStatusModel(
                isActive: true,
                memberExpiresAt: "2026-12-31 00:30:00",
                growthLevel: 2
            ),
            now: now,
            complianceMode: false
        )

        XCTAssertEqual(summary.statusText, "VIP中")
        XCTAssertTrue(summary.description.contains("展示「标准」外观"))
        XCTAssertFalse(summary.description.contains("倍率"))
        XCTAssertFalse(summary.description.contains("110%"))
        XCTAssertFalse(summary.description.contains("高光"))
    }

    func testSummaryReturnsExpiredCopyAfterMembershipEnds() {
        let summary = MemberCenterRules.resolveSummary(
            ProfileFixtures.memberStatusModel(
                isActive: false,
                memberExpiresAt: "2026-11-02 10:00:00",
                growthLevel: 3
            ),
            now: now,
            complianceMode: false
        )

        XCTAssertEqual(summary.statusText, "已到期")
        XCTAssertEqual(summary.title, "VIP已到期")
        XCTAssertTrue(summary.description.contains("2026-11-02 10:00:00"))
        XCTAssertTrue(summary.description.contains("「进阶」外观"))
        XCTAssertFalse(summary.description.contains("倍率"))
        XCTAssertFalse(summary.description.contains("120%"))
        XCTAssertFalse(summary.description.contains("高光"))
    }

    func testSummaryNeverAdvertisesRankingMultipliers() {
        let states = [
            ProfileFixtures.memberStatusModel(
                isActive: true,
                memberExpiresAt: "2026-12-31 00:30:00",
                growthLevel: 2
            ),
            ProfileFixtures.memberStatusModel(
                isActive: true,
                memberExpiresAt: "2026-12-31 00:30:00",
                growthLevel: 5
            ),
            ProfileFixtures.memberStatusModel(
                isActive: false,
                memberExpiresAt: "2026-11-02 10:00:00",
                growthLevel: 3
            ),
            ProfileFixtures.memberStatusModel(isActive: false, memberExpiresAt: nil),
        ]
        for status in states {
            let summary = MemberCenterRules.resolveSummary(status, now: now, complianceMode: false)
            let text = "\(summary.title) \(summary.description)"
            for word in ["高光", "倍率", "计入排位"] {
                XCTAssertFalse(text.contains(word), text)
            }
            XCTAssertNil(text.range(of: #"\d+%"#, options: .regularExpression), text)
        }

        let unopened = MemberCenterRules.resolveSummary(
            ProfileFixtures.memberStatusModel(isActive: false, memberExpiresAt: nil),
            now: now,
            complianceMode: false
        )
        XCTAssertTrue(unopened.description.contains("成长等级解锁更高阶外观"))
        let premium = MemberCenterRules.resolveSummary(states[1], now: now, complianceMode: false)
        XCTAssertTrue(premium.description.contains("「尊享」外观"))
    }

    func testComplianceSummaryAvoidsPaymentWording() {
        let summary = MemberCenterRules.resolveSummary(
            ProfileFixtures.memberStatusModel(isActive: true, memberExpiresAt: "2026-12-31 00:30:00"),
            now: now,
            complianceMode: true
        )

        XCTAssertEqual(summary.statusText, "VIP权益中")
        for word in ["月卡", "开通", "续费"] {
            XCTAssertFalse(summary.title.contains(word))
        }
        XCTAssertTrue(summary.description.contains("有效期"))
        XCTAssertEqual(summary.primaryActionText, "查看权益")
    }

    func testSummaryIncludesGrowthProgressForActiveMembers() {
        let summary = MemberCenterRules.resolveSummary(
            ProfileFixtures.memberStatusModel(
                isActive: true,
                memberExpiresAt: "2026-05-30 10:00:00",
                growthPoints: 78,
                todayGrowthCount: 3
            ),
            now: ProfileFixtures.time("2026-04-11T10:00:00+08:00"),
            complianceMode: false
        )

        XCTAssertEqual(summary.statusText, "VIP中")
        XCTAssertTrue(summary.description.contains("Lv3"))
        XCTAssertTrue(summary.description.contains("今日已计入 3/8"))
    }

    // MARK: - 入口卡

    func testEntryCardShowsUnopenedCopyForFirstTimeUser() {
        let card = MemberCenterRules.resolveEntryCard(
            ProfileFixtures.memberStatusModel(isActive: false, memberExpiresAt: nil),
            now: now,
            complianceMode: false
        )

        XCTAssertEqual(card.statusText, "未开通")
        XCTAssertEqual(card.title, "月卡VIP")
        XCTAssertEqual(card.actionText, "立即开通")
    }

    func testEntryCardShowsActiveCopyForValidMember() {
        let card = MemberCenterRules.resolveEntryCard(
            ProfileFixtures.memberStatusModel(isActive: true, memberExpiresAt: "2026-12-31 10:00:00"),
            now: now,
            complianceMode: false
        )

        XCTAssertEqual(card.statusText, "VIP中")
        XCTAssertEqual(card.title, "VIP有效期至 2026-12-31 10:00:00")
        XCTAssertEqual(card.actionText, "立即续费")
    }

    func testComplianceEntryCardHidesPriceAndPurchaseCopy() {
        let card = MemberCenterRules.resolveEntryCard(
            ProfileFixtures.memberStatusModel(isActive: false, memberExpiresAt: nil),
            now: now,
            complianceMode: true
        )

        XCTAssertFalse(card.visible)
        XCTAssertEqual(card.statusText, "")
        XCTAssertEqual(card.title, "")
        XCTAssertEqual(card.description, "")
        XCTAssertEqual(card.actionText, "")
        XCTAssertEqual(card.eyebrow, "")
        XCTAssertEqual(card.priceText, "")
    }

    func testComplianceEntryCardKeepsActiveCopyOnTheEntry() {
        let card = MemberCenterRules.resolveEntryCard(
            ProfileFixtures.memberStatusModel(isActive: true, memberExpiresAt: "2026-12-31 10:00:00"),
            now: now,
            complianceMode: true
        )

        XCTAssertEqual(card.statusText, "VIP权益中")
        XCTAssertEqual(card.title, "VIP权益已生效")
        XCTAssertEqual(card.description, "查看当前权益明细和有效期。")
        XCTAssertEqual(card.actionText, "查看权益")
        XCTAssertEqual(card.priceText, "")
    }

    func testComplianceEntryCardShowsEndedCopyWhenHistoryExists() {
        let card = MemberCenterRules.resolveEntryCard(
            ProfileFixtures.memberStatusModel(isActive: false, memberExpiresAt: "2026-11-02 10:00:00"),
            now: now,
            complianceMode: true
        )

        XCTAssertTrue(card.visible)
        XCTAssertEqual(card.statusText, "已结束")
        XCTAssertEqual(card.title, "获赠VIP已结束")
        for word in ["月卡", "开通", "续费"] {
            XCTAssertFalse(card.title.contains(word))
        }
        XCTAssertEqual(card.actionText, "查看权益")
        XCTAssertEqual(card.eyebrow, "VIP权益")
        XCTAssertEqual(card.priceText, "")
    }

    // MARK: - 身份预览

    func testIdentityPreviewShowsCurrentTierOnlyForActiveMembers() {
        let preview = MemberCenterRules.resolveIdentityPreview(
            ProfileFixtures.memberStatusModel(
                isActive: true,
                memberExpiresAt: "2026-12-31 00:30:00",
                growthLevel: 4
            ),
            now: now,
            rankLevel: 5,
            complianceMode: false
        )

        XCTAssertTrue(preview.active)
        XCTAssertEqual(preview.level, 4)
        XCTAssertEqual(preview.identity.tierLabel, "进阶")
        XCTAssertEqual(preview.identity.badgeText, "VIP4")
        XCTAssertEqual(preview.identity.frameURL?.lastPathComponent, "frame_diamond.png")
        XCTAssertEqual(
            preview.tiers.map { [$0.label, $0.levelText, "\($0.current)", "\($0.reached)"] },
            [
                ["标准", "Lv1–Lv2", "false", "true"],
                ["进阶", "Lv3–Lv4", "true", "true"],
                ["尊享", "Lv5", "false", "false"],
            ]
        )
        XCTAssertEqual(preview.tiers.map(\.badgeText), ["VIP1", "VIP3", "VIP5"])
        XCTAssertTrue(preview.hint.contains("进阶"))
        XCTAssertTrue(preview.hint.contains("排行榜、对局和好友列表"))
    }

    func testIdentityPreviewHidesAfterExpiryAndExplainsRenewal() {
        let expired = MemberCenterRules.resolveIdentityPreview(
            ProfileFixtures.memberStatusModel(
                isActive: false,
                memberExpiresAt: "2026-11-02 10:00:00",
                growthLevel: 3
            ),
            now: now,
            rankLevel: 4,
            complianceMode: false
        )

        XCTAssertFalse(expired.active)
        XCTAssertTrue(expired.frozen)
        XCTAssertFalse(expired.identity.showBadge)
        XCTAssertNil(expired.identity.frameURL)
        XCTAssertFalse(expired.tiers.contains(where: \.current))
        XCTAssertEqual(expired.level, 3)
        XCTAssertTrue(expired.hint.contains("已隐藏"))
        XCTAssertTrue(expired.hint.contains("续开"))
        XCTAssertTrue(expired.hint.contains("「进阶」"))

        let never = MemberCenterRules.resolveIdentityPreview(
            ProfileFixtures.memberStatusModel(isActive: false, memberExpiresAt: nil),
            now: now,
            rankLevel: 0,
            complianceMode: false
        )
        XCTAssertFalse(never.frozen)
        XCTAssertFalse(never.identity.showBadge)
        XCTAssertTrue(never.hint.contains("开通VIP后"))
    }

    func testComplianceIdentityPreviewAvoidsPurchaseWording() {
        let texts = [
            MemberCenterRules.resolveIdentityPreview(
                ProfileFixtures.memberStatusModel(isActive: false, memberExpiresAt: nil),
                now: now,
                rankLevel: 0,
                complianceMode: true
            ).hint,
            MemberCenterRules.resolveIdentityPreview(
                ProfileFixtures.memberStatusModel(
                    isActive: false,
                    memberExpiresAt: "2026-11-02 10:00:00",
                    growthLevel: 2
                ),
                now: now,
                rankLevel: 0,
                complianceMode: true
            ).hint,
        ]
        for text in texts {
            for word in ["开通", "订阅", "续开", "续费", "支付"] {
                XCTAssertFalse(text.contains(word), text)
            }
        }
        XCTAssertTrue(texts[0].contains("获得VIP权益后"))
    }

    // MARK: - 套餐与价格

    func testPlanCardsMarkOnlyTheSelectedPlan() {
        let cards = MemberCenterRules.resolvePlanCards(
            [
                MemberPlan(planCode: "member_monthly", planName: "月卡VIP"),
                MemberPlan(planCode: "member_yearly", planName: "年卡VIP"),
            ],
            selectedPlanCode: "member_monthly"
        )

        XCTAssertTrue(cards[0].selected)
        XCTAssertFalse(cards[1].selected)
    }

    func testFormatPriceUsesFen() {
        XCTAssertEqual(MemberCenterRules.formatPrice(priceFen: 1900), "¥19.00")
    }
}
