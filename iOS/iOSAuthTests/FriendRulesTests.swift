import Foundation
import XCTest
@testable import iOS

/// 好友规则测试，搬运 `uniapp/tests/friend-entry.test.mjs` 的用例。
@MainActor
final class FriendRulesTests: XCTestCase {
    private let now = AppFormat.parseDateTime("2026-10-11 12:00:00")!

    func testFriendUserIdPrefersBackendUserId() {
        XCTAssertEqual(FriendRules.friendUserId(userId: 18, id: 3), 18)
        XCTAssertEqual(FriendRules.friendUserId(userId: 0, id: 3), 3)
        XCTAssertEqual(FriendRules.friendUserId(userId: 0, id: 0), 0)
    }

    func testFriendListDecodesCanonicalUserId() throws {
        let data = try JSONSerialization.data(withJSONObject: [
            FriendFixtures.friend(id: 3, userId: 28, nickname: "球友 A"),
            FriendFixtures.friend(id: 4, userId: 0, nickname: "球友 B"),
        ])
        let decoder = JSONDecoder()
        decoder.keyDecodingStrategy = .convertFromSnakeCase
        let items = try decoder.decode([FriendInfo].self, from: data)

        XCTAssertEqual(items[0].friendUserId, 28)
        XCTAssertEqual(items[1].friendUserId, 4, "user_id 缺失时退回关系记录 id")
    }

    func testHomepageRouteCarriesIdentityNeededByLiteHomepage() {
        let friend = FriendInfo(
            id: 3,
            userId: 66,
            nickname: "球友 D",
            avatar: "",
            rankLevel: 4,
            rankName: "星耀",
            vipLevel: 3,
            createdAt: "2026-10-01 12:00:00"
        )

        XCTAssertEqual(
            FriendRules.homepageRoute(friend),
            .friendHomepage(target: FriendHomepageTarget(
                userId: 66,
                name: "球友 D",
                avatar: "",
                rankName: "星耀",
                vipLevel: 3,
                rankLevel: 4
            ))
        )
    }

    func testOpponentRecordRouteCarriesTargetContext() {
        let friend = FriendInfo(
            id: 3,
            userId: 52,
            nickname: "球友 B",
            avatar: "",
            rankLevel: 0,
            rankName: "",
            vipLevel: 0,
            createdAt: "2026-10-01 12:00:00"
        )

        XCTAssertEqual(
            FriendRules.opponentRecordRoute(friend),
            .opponentRecord(target: OpponentRecordTarget(userId: 52, name: "球友 B", avatar: ""))
        )
    }

    func testSearchStatusMapsFriendAndPending() {
        XCTAssertEqual(FriendRules.searchStatus(isFriend: true, hasPendingRequest: true), .friend)
        XCTAssertEqual(FriendRules.searchStatus(isFriend: false, hasPendingRequest: true), .pending)
        XCTAssertEqual(FriendRules.searchStatus(isFriend: false, hasPendingRequest: false), .addable)
    }

    func testApplicantUserIdPrefersFromUserId() {
        XCTAssertEqual(FriendRules.applicantUserId(fromUserId: 52, userId: 18), 52)
        XCTAssertEqual(FriendRules.applicantUserId(fromUserId: 0, userId: 18), 18)
    }

    func testConfirmCopyFallsBackToBallFriend() {
        XCTAssertEqual(FriendRules.displayName(""), "球友")
        XCTAssertEqual(FriendRules.deleteConfirmMessage("球友 B"), "确定删除好友「球友 B」吗？")
        XCTAssertEqual(
            FriendRules.blacklistConfirmMessage("球友 B"),
            "确定将「球友 B」加入黑名单吗？加入后会自动删除好友，并阻止彼此再次搜索和添加。可在 设置 - 黑名单 中移出。"
        )
        XCTAssertEqual(
            FriendRules.rejectAndBlacklistConfirmMessage(""),
            "确定拒绝并拉黑「球友」吗？拉黑后对方将无法再搜索或添加你。"
        )
        XCTAssertEqual(
            FriendRules.unblacklistConfirmMessage("球友 B"),
            "确定将「球友 B」移出黑名单吗？移出后对方可以重新搜索并添加你。"
        )
    }

    func testRequestMessageAndStatusCopy() {
        XCTAssertEqual(FriendRules.requestMessage("").text, "请求添加你为好友")
        XCTAssertTrue(FriendRules.requestMessage("").isMuted)
        XCTAssertEqual(FriendRules.requestMessage("一起打球").text, "一起打球")
        XCTAssertFalse(FriendRules.requestMessage("一起打球").isMuted)
        XCTAssertEqual(FriendRules.requestStatusText(1), "已接受")
        XCTAssertEqual(FriendRules.requestStatusText(2), "已拒绝")
    }

    func testRequestAndBlockedTime() {
        XCTAssertEqual(FriendRules.requestTime("2026-10-08 12:00:00", now: now), "3天前")
        XCTAssertEqual(FriendRules.blockedTime("2026-10-08 12:00:00", now: now), "3天前拉黑")
        XCTAssertEqual(FriendRules.requestTime("", now: now), "")
        XCTAssertEqual(FriendRules.blockedTime("不是时间", now: now), "不是时间")
    }

    func testRemoveBlacklistKeepsOtherItems() throws {
        let items = [
            FriendBlacklistItem(userId: 66, nickname: "A", avatar: "", createdAt: ""),
            FriendBlacklistItem(userId: 67, nickname: "B", avatar: "", createdAt: ""),
        ]

        XCTAssertEqual(FriendRules.removeBlacklist(items, userId: 66).map(\.userId), [67])
    }

    func testPendingBadgeCapsAt99Plus() {
        XCTAssertNil(FriendRules.pendingBadgeText(0))
        XCTAssertNil(FriendRules.pendingBadgeText(-3))
        XCTAssertEqual(FriendRules.pendingBadgeText(5), "5")
        XCTAssertEqual(FriendRules.pendingBadgeText(99), "99")
        XCTAssertEqual(FriendRules.pendingBadgeText(100), "99+")
    }
}
