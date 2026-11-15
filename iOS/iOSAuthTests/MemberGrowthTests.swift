import Foundation
import XCTest
@testable import iOS

@MainActor
final class MemberGrowthTests: XCTestCase {
    private let now = ProfileFixtures.time("2026-04-11T10:00:00+08:00")

    func testActiveMemberSeesGrowthProgress() {
        let card = MemberGrowthCard.resolve(
            ProfileFixtures.memberStatusModel(
                isActive: true,
                memberExpiresAt: "2026-05-30 10:00:00",
                growthPoints: 78,
                todayGrowthCount: 3
            ),
            now: now
        )

        XCTAssertEqual(card.statusText, "成长中")
        XCTAssertEqual(card.level, 3)
        XCTAssertEqual(card.levelLabel, "Lv3")
        XCTAssertEqual(card.growthPoints, 78)
        XCTAssertEqual(card.growthPointsText, "成长值 78")
        XCTAssertEqual(card.todayGrowthCount, 3)
        XCTAssertEqual(card.todayProgressText, "今日已计入 3/8")
        XCTAssertEqual(card.nextLevelText, "距 Lv4 还差 42 场")
        XCTAssertEqual(card.nextLevel, 4)
        XCTAssertFalse(card.frozen)
        XCTAssertEqual(card.progressPercent, 48)
    }

    func testExpiredMembershipFreezesGrowth() {
        let card = MemberGrowthCard.resolve(
            ProfileFixtures.memberStatusModel(
                isActive: false,
                memberExpiresAt: "2026-03-31 10:00:00",
                growthPoints: 78,
                todayGrowthCount: 1
            ),
            now: now
        )

        XCTAssertEqual(card.levelLabel, "Lv3")
        XCTAssertEqual(card.statusText, "成长已冻结")
        XCTAssertTrue(card.frozen)
        XCTAssertTrue(card.description.contains("续开后恢复"), card.description)
        XCTAssertEqual(card.nextLevelText, "续开后，再完成 42 场可升到 Lv4")
    }

    func testUnopenedMemberSeesGuidance() {
        let card = MemberGrowthCard.resolve(ProfileFixtures.memberStatusModel(memberExpiresAt: nil), now: now)

        XCTAssertEqual(card.levelLabel, "Lv1")
        XCTAssertEqual(card.growthPointsText, "成长值 0")
        XCTAssertEqual(card.todayProgressText, "今日已计入 0/8")
        XCTAssertEqual(card.nextLevelText, "开通VIP后，完成真实对局可累计成长")
        XCTAssertEqual(card.description, "开通VIP后，完成真实对局可累计成长。")
        XCTAssertFalse(card.frozen)
    }

    func testHighestLevelHasNoNextLevelAndFullProgress() {
        let card = MemberGrowthCard.resolve(
            ProfileFixtures.memberStatusModel(
                isActive: true,
                memberExpiresAt: "2026-05-30 10:00:00",
                growthPoints: 360
            ),
            now: now
        )

        XCTAssertEqual(card.levelLabel, "Lv5")
        XCTAssertNil(card.nextLevel)
        XCTAssertEqual(card.progressPercent, 100)
        XCTAssertEqual(card.nextLevelText, "当前已达到最高成长等级")
    }

    func testDailyCapAndExplicitRemainingPointsWin() {
        let card = MemberGrowthCard.resolve(
            ProfileFixtures.memberStatusModel(
                isActive: true,
                memberExpiresAt: "2026-05-30 10:00:00",
                growthPoints: 78,
                todayGrowthCount: 9,
                growthDailyCap: 3,
                remainingGrowthPoints: 5
            ),
            now: now
        )

        XCTAssertEqual(card.dailyCap, 3)
        XCTAssertEqual(card.todayGrowthCount, 3, "今日已计入不超过每日上限")
        XCTAssertEqual(card.todayProgressText, "今日已计入 3/3")
        XCTAssertEqual(card.nextLevelText, "距 Lv4 还差 5 场")
    }
}
