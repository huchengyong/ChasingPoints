import Foundation
import XCTest
@testable import iOS

@MainActor
final class UserHomepageRulesTests: XCTestCase {
    private let now = ProfileFixtures.time("2026-07-20T12:00:00+08:00")
    private let activeMember = ProfileFixtures.memberStatusModel(
        isActive: true,
        memberExpiresAt: "2026-08-01 12:00:00"
    )

    // MARK: - 生涯指标

    func testCoreMetricsCoverAllGameTypes() {
        let metrics = UserHomepageRules.coreMetrics(
            stats: ProfileFixtures.statsModel(
                totalMatches: 86, wins: 51, losses: 35, winRate: 59.3, maxWinStreak: 7
            ),
            isLoading: false
        )

        XCTAssertEqual(metrics.map(\.key), ["total", "wins", "win-rate", "max-streak"])
        XCTAssertEqual(metrics.map(\.label), ["场次", "胜场", "胜率", "最高连胜"])
        XCTAssertEqual(metrics.map(\.value), ["86", "51", "59.3%", "7"])
    }

    func testCoreMetricsShowPlaceholdersWhileLoading() {
        let metrics = UserHomepageRules.coreMetrics(stats: nil, isLoading: true)

        XCTAssertEqual(metrics.map(\.value), ["—", "—", "—", "—"])
        XCTAssertEqual(metrics.map(\.id), ["total", "wins", "win-rate", "max-streak"])
    }

    func testWinRateDropsTrailingZeroForWholeNumbers() {
        XCTAssertEqual(
            UserHomepageRules.coreMetrics(stats: ProfileFixtures.statsModel(winRate: 50), isLoading: false)[2].value,
            "50%"
        )
        XCTAssertEqual(
            UserHomepageRules.coreMetrics(stats: ProfileFixtures.statsModel(winRate: 66.7), isLoading: false)[2].value,
            "66.7%"
        )
        XCTAssertEqual(
            UserHomepageRules.coreMetrics(stats: ProfileFixtures.statsModel(winRate: 0), isLoading: false)[2].value,
            "0%"
        )
    }

    // MARK: - 段位卡

    func testRankCardShowsScoreGapToNextRank() {
        let card = UserHomepageRules.rankCard(ProfileFixtures.rankInfoModel(
            level: 3, name: "黄金", icon: "gold.png", rankScore: 1320,
            nextName: "铂金", nextScore: 1500, progress: 64
        ))

        XCTAssertTrue(card.ranked)
        XCTAssertEqual(card.name, "黄金")
        XCTAssertEqual(card.score, 1320)
        XCTAssertEqual(card.progress, 64)
        XCTAssertEqual(card.hint, "距铂金还差 180 分")
    }

    func testRankCardOnlyLevelSixIsTheMaxRank() {
        let diamond = UserHomepageRules.rankCard(ProfileFixtures.rankInfoModel(
            level: 5, name: "钻石", rankScore: 2400, nextName: "王者", nextScore: 3000, progress: 40
        ))
        XCTAssertEqual(diamond.hint, "距王者还差 600 分")
        XCTAssertEqual(diamond.progress, 40)

        let king = UserHomepageRules.rankCard(ProfileFixtures.rankInfoModel(
            level: 6, name: "王者", rankScore: 3200, nextScore: 3000, progress: 100
        ))
        XCTAssertEqual(king.hint, "已达到最高段位")
        XCTAssertEqual(king.progress, 100)
    }

    func testUnrankedCardUsesPlaceholderCopy() {
        let card = UserHomepageRules.rankCard(nil)

        XCTAssertFalse(card.ranked)
        XCTAssertEqual(card.name, "未定级")
        XCTAssertEqual(card.score, 0)
        XCTAssertEqual(card.progress, 0)
        XCTAssertEqual(card.hint, "完成首场排位赛后开始定级")
    }

    func testRankCardIconIsResolvedToTheCDN() {
        let gold = UserHomepageRules.rankCard(ProfileFixtures.rankInfoModel(
            name: "黄金", icon: "/static/images/ranks/rank_gold.png"
        ))
        XCTAssertEqual(
            gold.iconURL?.absoluteString,
            "https://cdn.zhuifen.cn/static/images/ranks/rank_gold.png"
        )

        XCTAssertEqual(
            UserHomepageRules.rankCard(nil).iconURL?.absoluteString,
            "https://cdn.zhuifen.cn/static/images/ranks/rank_bronze.png"
        )
    }

    // MARK: - 段位卡球种

    func testRankGameTypePrefersSavedGameTypeAndKeepsPlayedOnesSwitchable() {
        let view = UserHomepageRules.rankGameTypeView(
            preferredGameType: 1,
            rankInfoMap: [
                1: ProfileFixtures.rankInfoModel(totalWins: 0, totalLosses: 0),
                2: ProfileFixtures.rankInfoModel(totalWins: 30, totalLosses: 10),
                3: ProfileFixtures.rankInfoModel(totalWins: 0, totalLosses: 0),
                4: ProfileFixtures.rankInfoModel(totalWins: 1, totalLosses: 0),
            ]
        )

        XCTAssertEqual(view.primaryGameType, 1)
        XCTAssertEqual(view.primaryLabel, "斯诺克")
        XCTAssertEqual(view.tabs.map(\.value), [2, 1, 4])
        XCTAssertFalse(view.needsPick)
    }

    func testRankGameTypeFallsBackToMostPlayedThenScoreThenFixedOrder() {
        XCTAssertEqual(UserHomepageRules.rankGameTypeView(
            preferredGameType: 0,
            rankInfoMap: [
                1: ProfileFixtures.rankInfoModel(rankScore: 1800, totalWins: 3, totalLosses: 2),
                2: ProfileFixtures.rankInfoModel(totalWins: 20, totalLosses: 25),
                3: ProfileFixtures.rankInfoModel(rankScore: 2400, totalWins: 10, totalLosses: 5),
                4: ProfileFixtures.rankInfoModel(totalWins: 0, totalLosses: 0),
            ]
        ).primaryGameType, 2)

        XCTAssertEqual(UserHomepageRules.rankGameTypeView(
            preferredGameType: 0,
            rankInfoMap: [
                1: ProfileFixtures.rankInfoModel(rankScore: 1300, totalWins: 5, totalLosses: 5),
                2: ProfileFixtures.rankInfoModel(rankScore: 1200, totalWins: 4, totalLosses: 6),
                3: ProfileFixtures.rankInfoModel(totalWins: 0, totalLosses: 0),
                4: ProfileFixtures.rankInfoModel(totalWins: 0, totalLosses: 0),
            ]
        ).primaryGameType, 1)

        XCTAssertEqual(UserHomepageRules.rankGameTypeView(
            preferredGameType: 0,
            rankInfoMap: [
                1: ProfileFixtures.rankInfoModel(rankScore: 1200, totalWins: 5, totalLosses: 5),
                2: ProfileFixtures.rankInfoModel(rankScore: 1200, totalWins: 4, totalLosses: 6),
                3: ProfileFixtures.rankInfoModel(totalWins: 0, totalLosses: 0),
                4: ProfileFixtures.rankInfoModel(totalWins: 0, totalLosses: 0),
            ]
        ).primaryGameType, 2)
    }

    func testRankGameTypeHidesSwitcherForSingleGameTypePlayers() {
        let view = UserHomepageRules.rankGameTypeView(
            preferredGameType: 0,
            rankInfoMap: [
                1: ProfileFixtures.rankInfoModel(totalWins: 0, totalLosses: 0),
                2: ProfileFixtures.rankInfoModel(totalWins: 12, totalLosses: 3),
                3: ProfileFixtures.rankInfoModel(totalWins: 0, totalLosses: 0),
                4: ProfileFixtures.rankInfoModel(totalWins: 0, totalLosses: 0),
            ]
        )

        XCTAssertEqual(view.primaryGameType, 2)
        XCTAssertTrue(view.tabs.isEmpty)
        XCTAssertFalse(view.needsPick)
    }

    func testRankGameTypeAsksNewPlayersToPickOnlyWhenNothingIsSavedOrPlayed() {
        let fresh = UserHomepageRules.rankGameTypeView(
            preferredGameType: 0,
            rankInfoMap: Dictionary(
                uniqueKeysWithValues: [1, 2, 3, 4].map {
                    ($0, ProfileFixtures.rankInfoModel(totalWins: 0, totalLosses: 0))
                }
            )
        )
        XCTAssertEqual(fresh.primaryGameType, 3)
        XCTAssertTrue(fresh.tabs.isEmpty)
        XCTAssertTrue(fresh.needsPick)

        XCTAssertFalse(UserHomepageRules.rankGameTypeView(preferredGameType: 4, rankInfoMap: [:]).needsPick)
        XCTAssertEqual(UserHomepageRules.rankGameTypeView(preferredGameType: 99, rankInfoMap: [:]).primaryGameType, 3)
    }

    // MARK: - 信誉与会员入口

    func testReputationChipNamesTheDimensionAndFlagsABan() {
        XCTAssertEqual(
            UserHomepageRules.reputationChip(ProfileFixtures.reputationModel(status: "good", statusText: "良好"), isLoading: false),
            ReputationChip(text: "信誉良好", restricted: false)
        )
        XCTAssertEqual(
            UserHomepageRules.reputationChip(
                ProfileFixtures.reputationModel(status: "restricted", statusText: "禁赛"), isLoading: false
            ),
            ReputationChip(text: "禁赛", restricted: true)
        )
        XCTAssertEqual(UserHomepageRules.reputationChip(nil, isLoading: true).text, "信誉同步中")
        XCTAssertEqual(UserHomepageRules.reputationChip(nil, isLoading: false).text, "信誉暂不可用")
    }

    func testMemberEntryOnlyLabelsActiveOrExpiredMemberships() {
        XCTAssertEqual(
            UserHomepageRules.memberEntry(activeMember, now: now),
            MemberEntry(state: "active", text: "2026-08-01 到期")
        )
        XCTAssertEqual(
            UserHomepageRules.memberEntry(
                ProfileFixtures.memberStatusModel(isActive: true, memberExpiresAt: "2026-06-20 12:00:00"),
                now: now
            ),
            MemberEntry(state: "expired", text: "已到期")
        )
        XCTAssertEqual(
            UserHomepageRules.memberEntry(ProfileFixtures.memberStatusModel(memberExpiresAt: nil), now: now),
            MemberEntry(state: "none", text: "")
        )
        XCTAssertEqual(UserHomepageRules.memberEntry(nil, now: now), MemberEntry(state: "none", text: ""))
    }

    // MARK: - 会员身份

    func testSelfIdentityShowsRankFrameAndGrowthTierBadgeForActiveMembers() {
        let identity = UserHomepageRules.selfMemberIdentity(
            isLoggedIn: true,
            memberStatus: ProfileFixtures.memberStatusModel(
                isActive: true, memberExpiresAt: "2026-08-01 12:00:00", growthLevel: 3
            ),
            rankInfo: ProfileFixtures.rankInfoModel(level: 5),
            rankLoading: false,
            now: now
        )

        XCTAssertEqual(
            identity.frameURL?.absoluteString,
            "https://cdn.zhuifen.cn/static/images/avatar-frames/frame_diamond.png"
        )
        XCTAssertEqual(identity.badgeText, "VIP3")
        XCTAssertEqual(identity.tierLabel, "进阶")

        // 没有显式等级时按成长值换算，和会员中心一致。
        XCTAssertEqual(UserHomepageRules.selfMemberIdentity(
            isLoggedIn: true,
            memberStatus: ProfileFixtures.memberStatusModel(
                isActive: true, memberExpiresAt: "2026-08-01 12:00:00", growthPoints: 300
            ),
            rankInfo: ProfileFixtures.rankInfoModel(level: 1),
            rankLoading: false,
            now: now
        ).badgeText, "VIP5")
    }

    func testSelfIdentityHidesForGuestsOrdinaryExpiredAndFailedMembers() {
        let states: [MemberStatus?] = [
            nil,
            ProfileFixtures.memberStatusModel(isActive: false, memberExpiresAt: "2026-08-01 12:00:00", growthLevel: 4),
            ProfileFixtures.memberStatusModel(isActive: true, memberExpiresAt: "2026-07-20 12:00:00", growthLevel: 4),
            ProfileFixtures.memberStatusModel(isActive: true, memberExpiresAt: "invalid", growthLevel: 4),
        ]

        for (index, memberStatus) in states.enumerated() {
            let identity = UserHomepageRules.selfMemberIdentity(
                isLoggedIn: true,
                memberStatus: memberStatus,
                rankInfo: ProfileFixtures.rankInfoModel(level: 6),
                rankLoading: false,
                now: now
            )
            XCTAssertFalse(identity.showBadge, "state=\(index)")
            XCTAssertNil(identity.frameURL, "state=\(index)")
        }

        let guest = UserHomepageRules.selfMemberIdentity(
            isLoggedIn: false,
            memberStatus: ProfileFixtures.memberStatusModel(
                isActive: true, memberExpiresAt: "2026-08-01 12:00:00", growthLevel: 4
            ),
            rankInfo: ProfileFixtures.rankInfoModel(level: 6),
            rankLoading: false,
            now: now
        )
        XCTAssertFalse(guest.showBadge)
        XCTAssertNil(guest.frameURL)
    }

    func testSelfIdentityWaitsForRealRankAndDropsStaleSwitchData() {
        let member = ProfileFixtures.memberStatusModel(
            isActive: true, memberExpiresAt: "2026-08-01 12:00:00", growthLevel: 2
        )
        let stale: [(rankInfo: RankInfo?, rankLoading: Bool)] = [
            (nil, true),
            (ProfileFixtures.rankInfoModel(level: 5), true),
            (nil, false),
        ]

        for state in stale {
            let identity = UserHomepageRules.selfMemberIdentity(
                isLoggedIn: true,
                memberStatus: member,
                rankInfo: state.rankInfo,
                rankLoading: state.rankLoading,
                now: now
            )
            XCTAssertNil(identity.frameURL)
            // 段位没读到不影响会员标识本身。
            XCTAssertEqual(identity.badgeText, "VIP2")
        }
    }

    // MARK: - 资料头部等级标签

    func testProfileLevelTextCoversLoadingBadgeAndGrowthLevels() {
        XCTAssertEqual(
            UserHomepageRules.profileLevelText(
                memberStatus: nil,
                identity: MemberIdentity.resolve(vipLevel: 0, rankLevel: 0),
                isLoading: true,
                now: now
            ),
            "等级同步中"
        )

        let identity = MemberIdentity.resolve(vipLevel: 3, rankLevel: 5)
        XCTAssertEqual(
            UserHomepageRules.profileLevelText(
                memberStatus: activeMember, identity: identity, isLoading: false, now: now
            ),
            "VIP3 · 进阶"
        )

        XCTAssertEqual(
            UserHomepageRules.profileLevelText(
                memberStatus: ProfileFixtures.memberStatusModel(growthPoints: 78),
                identity: MemberIdentity.resolve(vipLevel: 0, rankLevel: 0),
                isLoading: false,
                now: now
            ),
            "Lv3"
        )
    }

    // MARK: - 常玩球馆奖励紧凑入口

    func testCompactRewardEntryMarksNotStartedAndRejectedAsFloatable() {
        let notStarted = UserHomepageRules.compactRewardEntry(ProfileFixtures.favoriteVenueRewardModel())
        XCTAssertTrue(notStarted.visible)
        XCTAssertTrue(notStarted.floatable)
        XCTAssertFalse(notStarted.inline)
        XCTAssertEqual(notStarted.state, "not_started")
        XCTAssertEqual(notStarted.action, "favorite-venue")
        XCTAssertEqual(notStarted.title, "免费VIP赠送")
        XCTAssertEqual(notStarted.description, "绑定你常去的球馆，审核通过后可免费领取 1 个月VIP")
        XCTAssertEqual(notStarted.actionText, "去领取")

        let rejected = UserHomepageRules.compactRewardEntry(
            ProfileFixtures.favoriteVenueRewardModel(status: "rejected", rejectReason: "地址不完整")
        )
        XCTAssertTrue(rejected.visible)
        XCTAssertTrue(rejected.floatable)
        XCTAssertFalse(rejected.inline)
        XCTAssertEqual(rejected.state, "rejected")
        XCTAssertEqual(rejected.action, "favorite-venue")
        XCTAssertEqual(rejected.title, "补充信息后可重新领取VIP")
        XCTAssertEqual(rejected.actionText, "重新提交")
    }

    func testCompactRewardEntryMarksPendingReviewAsInlineReadOnly() {
        let pending = UserHomepageRules.compactRewardEntry(
            ProfileFixtures.favoriteVenueRewardModel(
                status: "pending_review", submittedVenueName: "测试球馆"
            )
        )

        XCTAssertTrue(pending.visible)
        XCTAssertFalse(pending.floatable)
        XCTAssertTrue(pending.inline)
        XCTAssertEqual(pending.action, "")
        XCTAssertEqual(pending.actionText, "")
        XCTAssertEqual(pending.statusText, "审核中")
    }

    func testCompactRewardEntryHidesGrantedAndDisabledStates() {
        let granted = UserHomepageRules.compactRewardEntry(
            ProfileFixtures.favoriteVenueRewardModel(status: "reward_granted")
        )
        XCTAssertFalse(granted.visible)
        XCTAssertFalse(granted.floatable)
        XCTAssertFalse(granted.inline)

        let disabled = UserHomepageRules.compactRewardEntry(
            ProfileFixtures.favoriteVenueRewardModel(enabled: false)
        )
        XCTAssertFalse(disabled.visible)
        XCTAssertFalse(disabled.floatable)
        XCTAssertFalse(disabled.inline)
    }
}
