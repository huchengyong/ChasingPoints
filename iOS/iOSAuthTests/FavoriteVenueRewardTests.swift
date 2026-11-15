import Foundation
import XCTest
@testable import iOS

@MainActor
final class FavoriteVenueRewardTests: XCTestCase {
    func testFormatDurationUsesWholeMonthsThenDays() {
        XCTAssertEqual(FavoriteVenueReward.formatDuration(rewardDays: 30), "1 个月VIP")
        XCTAssertEqual(FavoriteVenueReward.formatDuration(rewardDays: 60), "2 个月VIP")
        XCTAssertEqual(FavoriteVenueReward.formatDuration(rewardDays: 31), "31 天VIP")
    }

    func testClaimableOnlyWhenEnabledAndNotGranted() {
        XCTAssertTrue(FavoriteVenueReward.isClaimable(ProfileFixtures.favoriteVenueRewardModel()))
        XCTAssertFalse(FavoriteVenueReward.isClaimable(ProfileFixtures.favoriteVenueRewardModel(enabled: false)))
        XCTAssertFalse(
            FavoriteVenueReward.isClaimable(ProfileFixtures.favoriteVenueRewardModel(status: "reward_granted"))
        )
        XCTAssertFalse(FavoriteVenueReward.isClaimable(nil))
    }

    func testTaskCardReturnsEntryCopyForNotStartedUsers() {
        let card = FavoriteVenueReward.taskCard(
            ProfileFixtures.favoriteVenueRewardModel(status: "not_started")
        )

        XCTAssertTrue(card.visible)
        XCTAssertEqual(card.title, "补充常玩球馆，送 1 个月VIP")
        XCTAssertEqual(card.description, "首次有效补充并审核通过后，VIP会自动到账。")
        XCTAssertEqual(card.actionText, "去领取")
        XCTAssertEqual(card.statusText, "常玩球馆奖励")
    }

    func testTaskCardReturnsWaitingCopyForPendingReviewUsers() {
        let card = FavoriteVenueReward.taskCard(
            ProfileFixtures.favoriteVenueRewardModel(
                status: "pending_review",
                submittedVenueName: "南山台球会"
            )
        )

        XCTAssertTrue(card.visible)
        XCTAssertEqual(card.title, "常玩球馆审核中")
        XCTAssertEqual(card.description, "南山台球会 正在审核中，通过后会自动发放VIP。")
        XCTAssertEqual(card.actionText, "")
        XCTAssertEqual(card.statusText, "审核中")
    }

    func testTaskCardHidesGrantedRewards() {
        let card = FavoriteVenueReward.taskCard(
            ProfileFixtures.favoriteVenueRewardModel(status: "reward_granted")
        )

        XCTAssertFalse(card.visible)
        XCTAssertEqual(card.title, "")
        XCTAssertEqual(card.description, "")
        XCTAssertEqual(card.actionText, "")
    }

    func testTaskCardReturnsRetryCopyForRejectedUsers() {
        let card = FavoriteVenueReward.taskCard(
            ProfileFixtures.favoriteVenueRewardModel(status: "rejected", rejectReason: "地址信息不完整")
        )

        XCTAssertTrue(card.visible)
        XCTAssertEqual(card.title, "补充信息后可重新领取VIP")
        XCTAssertEqual(card.description, "本次未通过：地址信息不完整")
        XCTAssertEqual(card.actionText, "重新提交")
        XCTAssertEqual(card.statusText, "未通过")
    }

    func testTaskCardHidesUntouchedTasksWhenActivityIsDisabled() {
        let card = FavoriteVenueReward.taskCard(
            ProfileFixtures.favoriteVenueRewardModel(enabled: false, status: "not_started")
        )

        XCTAssertFalse(card.visible)
    }

    func testTaskCardDropsVIPPromisesWhenActivityIsDisabled() {
        let pending = FavoriteVenueReward.taskCard(
            ProfileFixtures.favoriteVenueRewardModel(
                enabled: false,
                status: "pending_review",
                submittedVenueName: "南山台球会"
            )
        )
        XCTAssertTrue(pending.visible)
        XCTAssertEqual(pending.description, "南山台球会 正在审核中，审核通过后会展示给附近球友。")

        let rejected = FavoriteVenueReward.taskCard(
            ProfileFixtures.favoriteVenueRewardModel(
                enabled: false,
                status: "rejected",
                rejectReason: "地址信息不完整"
            )
        )
        XCTAssertEqual(rejected.title, "补充信息后可重新提交")
    }

    func testTaskCardDoesNotUseNewUserWording() {
        let card = FavoriteVenueReward.taskCard(ProfileFixtures.favoriteVenueRewardModel())

        XCTAssertEqual(card.statusText, "常玩球馆奖励")
        XCTAssertFalse(card.title.contains("新用户福利"))
    }

    func testTaskCardWithoutStatusHidesEntry() {
        let card = FavoriteVenueReward.taskCard(nil)

        XCTAssertFalse(card.visible, "没有状态时按活动未开启处理")
        XCTAssertEqual(card.title, "")
    }

    func testFloatSnoozeKeyScopesByUserAndStatus() {
        XCTAssertEqual(
            FavoriteVenueReward.floatSnoozeKey(userId: 12, status: "not_started"),
            "favorite-venue-reward-float-snoozed-until:12:not_started"
        )
        XCTAssertEqual(
            FavoriteVenueReward.floatSnoozeKey(userId: 42, status: "rejected"),
            "favorite-venue-reward-float-snoozed-until:42:rejected"
        )
        XCTAssertNotEqual(
            FavoriteVenueReward.floatSnoozeKey(userId: 12, status: "not_started"),
            FavoriteVenueReward.floatSnoozeKey(userId: 12, status: "rejected")
        )
        XCTAssertNotEqual(
            FavoriteVenueReward.floatSnoozeKey(userId: 12, status: "not_started"),
            FavoriteVenueReward.floatSnoozeKey(userId: 18, status: "not_started")
        )
    }

    func testSnoozeUntilIsThreeDaysAfterNow() {
        let now = ProfileFixtures.time("2026-07-21T12:00:00+08:00")
        XCTAssertEqual(
            FavoriteVenueReward.snoozeUntil(from: now),
            now.addingTimeInterval(3 * 24 * 60 * 60)
        )
    }

    func testSnoozedOnlyWhileDeadlineIsInTheFuture() {
        let now = ProfileFixtures.time("2026-07-22T00:00:00+08:00")
        let futureDeadline = ProfileFixtures.time("2026-07-25T00:00:00+08:00")
        let pastDeadline = ProfileFixtures.time("2026-07-21T00:00:00+08:00")

        XCTAssertTrue(FavoriteVenueReward.isSnoozed(until: futureDeadline, now: now))
        XCTAssertFalse(FavoriteVenueReward.isSnoozed(until: pastDeadline, now: now))
        XCTAssertFalse(FavoriteVenueReward.isSnoozed(until: nil, now: now))
    }

    func testSnoozedIsFalseWhenDeadlineEqualsNow() {
        let now = ProfileFixtures.time("2026-07-24T12:00:00+08:00")
        XCTAssertFalse(FavoriteVenueReward.isSnoozed(until: now, now: now))
    }
}
