import XCTest
@testable import iOS

final class AchievementPageRulesTests: XCTestCase {
    func testCategoryGroupsUseBackendCodesAndChineseLabels() {
        XCTAssertEqual(
            AchievementPageRules.categoryGroups.map(\.key),
            ["match", "wins", "streak", "tournament"]
        )
        XCTAssertEqual(AchievementPageRules.categoryLabel("wins"), "胜场")
        XCTAssertEqual(AchievementPageRules.categoryLabel("streak"), "连胜")
        XCTAssertEqual(AchievementPageRules.categoryLabel("special"), "特殊")
        XCTAssertEqual(AchievementPageRules.categoryLabel("match"), "对局")
        XCTAssertEqual(AchievementPageRules.categoryLabel("tournament"), "赛事")
        XCTAssertEqual(AchievementPageRules.categoryLabel(nil), "其他")
        XCTAssertEqual(AchievementPageRules.categoryEmoji("wins"), "🏅")
        XCTAssertEqual(AchievementPageRules.categoryEmoji("unknown"), "🎯")
    }

    func testListGroupsByCategoryAndHidesEmptyGroups() {
        let list = [
            def(id: 1, category: "wins"),
            def(id: 2, category: "streak"),
            def(id: 3, category: "wins"),
            def(id: 4, category: "tournament"),
        ]

        let groups = AchievementPageRules.groupAchievementsByCategory(list)
        XCTAssertEqual(groups.map(\.key), ["wins", "streak", "tournament"])
        XCTAssertEqual(groups.map { $0.items.count }, [2, 1, 1])
        XCTAssertEqual(groups[0].items.map(\.id), [1, 3])
    }

    func testCareerAchievementsSplitUniversalMilestonesFromSpecialties() {
        let list = [
            def(id: 1, category: "wins", gameType: 0, unlocked: true),
            def(id: 2, category: "match", gameType: 0),
            def(id: 3, category: "streak", gameType: 0),
            def(id: 4, category: "tournament", gameType: 0),
            def(id: 5, category: "special", gameType: 2, unlocked: true),
            def(id: 6, category: "special", gameType: 2),
            def(id: 7, category: "special", gameType: 4, unlocked: true),
        ]

        let universalGroups = AchievementPageRules.groupUniversalAchievements(list)
        XCTAssertEqual(universalGroups.map(\.key), ["match", "wins", "streak", "tournament"])
        XCTAssertEqual(universalGroups.map { $0.items.map(\.id) }, [[2], [1], [3], [4]])
        XCTAssertEqual(AchievementPageRules.filterSpecialtyAchievements(list, gameType: 1).map(\.id), [])
        XCTAssertEqual(AchievementPageRules.filterSpecialtyAchievements(list, gameType: 2).map(\.id), [5, 6])
        XCTAssertEqual(
            AchievementPageRules.filterSpecialtyAchievements(list, gameType: 2, unlockedOnly: true).map(\.id),
            [5]
        )
        XCTAssertEqual(AchievementPageRules.filterSpecialtyAchievements(list, gameType: 3).map(\.id), [])
        XCTAssertEqual(AchievementPageRules.filterSpecialtyAchievements(list, gameType: 4).map(\.id), [7])
        XCTAssertEqual(AchievementPageRules.filterSpecialtyAchievements(list, gameType: 99).map(\.id), [])
        XCTAssertEqual(AchievementPageRules.gameTypeLabel(1), "斯诺克")
        XCTAssertEqual(AchievementPageRules.gameTypeLabel(2), "九球追分")
        XCTAssertEqual(AchievementPageRules.gameTypeLabel(3), "中式八球")
        XCTAssertEqual(AchievementPageRules.gameTypeLabel(4), "美式九球")
        XCTAssertEqual(AchievementPageRules.gameTypeLabel(0), "通用")
        XCTAssertEqual(
            AchievementPageRules.fallbackEmoji(def(id: 1, category: "special", gameType: 1)),
            "🔴"
        )
    }

    func testTitleSourcesDisplayChineseLabelsWhileKeepingStableClasses() {
        XCTAssertEqual(AchievementPageRules.titleSourceLabel("achievement"), "成就")
        XCTAssertEqual(AchievementPageRules.titleSourceLabel("season"), "赛季")
        XCTAssertEqual(AchievementPageRules.titleSourceLabel("tournament"), "赛事")
        XCTAssertEqual(AchievementPageRules.titleSourceLabel("成就"), "成就")
        XCTAssertEqual(AchievementPageRules.titleSourceLabel(nil), "其他")
        XCTAssertEqual(AchievementPageRules.titleSourceClass("achievement"), "achievement")
        XCTAssertEqual(AchievementPageRules.titleSourceClass("赛季"), "season")
        XCTAssertEqual(AchievementPageRules.titleSourceClass(nil), "default")
    }

    func testTitleOptionsPutTheEquippedTitleFirstWithoutDisturbingApiOrder() {
        let input = [
            title(id: 1, name: "胜场新星"),
            title(id: 2, name: "S3 冠军", equipped: true),
            title(id: 3, name: "城市杯四强"),
        ]

        XCTAssertEqual(AchievementPageRules.sortTitleOptions(input).map(\.id), [2, 1, 3])
        XCTAssertEqual(input.map(\.id), [1, 2, 3])
    }

    func testTitleSourceDescriptionsUseSourceTypeAndReadableOriginName() {
        XCTAssertEqual(
            AchievementPageRules.titleSourceDescription(title(id: 1, source: "achievement", sourceRefName: "五连胜成就")),
            "生涯成就 · 五连胜成就"
        )
        XCTAssertEqual(
            AchievementPageRules.titleSourceDescription(title(id: 2, source: "season", sourceRefName: "S3")),
            "赛季荣誉 · S3"
        )
        XCTAssertEqual(
            AchievementPageRules.titleSourceDescription(title(id: 3, source: "tournament", sourceRefName: "")),
            "赛事荣誉"
        )
    }

    func testTitleSelectionResolvesNoOpEquipAndUnequipActions() {
        XCTAssertEqual(
            AchievementPageRules.resolveTitleSelection(currentTitleId: 2, selectedTitleId: 2),
            .close
        )
        XCTAssertEqual(
            AchievementPageRules.resolveTitleSelection(currentTitleId: 2, selectedTitleId: 3),
            .equip(titleId: 3)
        )
        XCTAssertEqual(
            AchievementPageRules.resolveTitleSelection(currentTitleId: 2, selectedTitleId: 0),
            .unequip(titleId: 2)
        )
        XCTAssertEqual(
            AchievementPageRules.resolveTitleSelection(currentTitleId: 0, selectedTitleId: 0),
            .close
        )
    }

    func testDetailProgressMatchesUnlockedAndMissingThresholdRules() {
        XCTAssertEqual(AchievementPageRules.detailProgressPercent(progress: 3, threshold: 10, unlocked: false), 30)
        XCTAssertEqual(AchievementPageRules.detailProgressPercent(progress: 10, threshold: 10, unlocked: false), 100)
        XCTAssertEqual(AchievementPageRules.detailProgressPercent(progress: 1, threshold: 0, unlocked: false), 100)
        XCTAssertEqual(AchievementPageRules.detailProgressPercent(progress: 0, threshold: 0, unlocked: false), 0)
        XCTAssertEqual(AchievementPageRules.detailProgressPercent(progress: 0, threshold: 10, unlocked: true), 100)
        XCTAssertEqual(AchievementPageRules.detailProgressPercent(progress: 15, threshold: 10, unlocked: false), 100)
    }

    func testHonorAndChallengeEmojiFallbacks() {
        XCTAssertEqual(AchievementPageRules.honorEmoji("season"), "👑")
        XCTAssertEqual(AchievementPageRules.honorEmoji("tournament"), "🏆")
        XCTAssertEqual(AchievementPageRules.honorEmoji("achievement"), "🏅")
        XCTAssertEqual(AchievementPageRules.challengeEmoji("season_wins_10"), "🔥")
        XCTAssertEqual(AchievementPageRules.challengeEmoji("season_tournament_finish_1"), "🏆")
        XCTAssertEqual(AchievementPageRules.challengeEmoji("season_matches_20"), "🎱")
    }

    // MARK: - 辅助

    private func def(
        id: Int64,
        category: String = "wins",
        gameType: Int = 0,
        unlocked: Bool = false
    ) -> AchievementDef {
        AchievementDef(
            id: id,
            key: "k\(id)",
            name: "成就 \(id)",
            description: "",
            icon: "",
            category: category,
            gameType: gameType,
            rewardTitleName: nil,
            threshold: 10,
            progress: 0,
            unlocked: unlocked,
            unlockedAt: nil
        )
    }

    private func title(
        id: Int64,
        name: String = "称号",
        source: String = "achievement",
        sourceRefName: String = "",
        equipped: Bool = false
    ) -> TitleInfo {
        TitleInfo(
            id: id,
            titleName: name,
            source: source,
            sourceType: source,
            sourceRefId: nil,
            sourceRefName: sourceRefName,
            equipped: equipped,
            grantedAt: nil,
            createdAt: "2026-08-01 09:00:00"
        )
    }
}
