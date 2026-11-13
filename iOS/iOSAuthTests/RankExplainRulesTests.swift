import XCTest
@testable import iOS

@MainActor
final class RankExplainRulesTests: XCTestCase {
    private let rankList = RankingFixtures.rankList

    // MARK: - 成长路径

    func testGrowthPathReturnsSixRanksFromKingToBronzeWithAdjacentRanges() {
        let path = RankExplainRules.growthPath(rankList: rankList, currentLevel: 3, rankScore: 1280)

        XCTAssertEqual(path.count, 6)
        XCTAssertEqual(path.map(\.level), [6, 5, 4, 3, 2, 1])
        XCTAssertEqual(path.map(\.rangeText), [
            "2500 分以上",
            "2000–2499 分",
            "1500–1999 分",
            "1000–1499 分",
            "500–999 分",
            "0–499 分",
        ])
    }

    func testGrowthPathMarksCurrentReachedAndPendingForGoldUser() {
        let path = RankExplainRules.growthPath(rankList: rankList, currentLevel: 3, rankScore: 1280)
        let stateByLevel = Dictionary(uniqueKeysWithValues: path.map { ($0.level, $0.state) })

        XCTAssertEqual(stateByLevel[3], .current)
        XCTAssertEqual(stateByLevel[1], .reached)
        XCTAssertEqual(stateByLevel[2], .reached)
        XCTAssertEqual(stateByLevel[4], .pending)
        XCTAssertEqual(stateByLevel[5], .pending)
        XCTAssertEqual(stateByLevel[6], .pending)
        XCTAssertEqual(path.filter { $0.state == .current }.count, 1)
        XCTAssertEqual(path.first { $0.level == 3 }?.stateText, "当前")
        XCTAssertEqual(path.first { $0.level == 1 }?.stateText, "已达成")
        XCTAssertEqual(path.first { $0.level == 6 }?.stateText, "待晋级")
    }

    // MARK: - 英雄卡

    func testHeroModelComputesInLevelProgressAndRemainingScore() throws {
        let hero = try XCTUnwrap(RankExplainRules.heroModel(
            rankInfo: RankingFixtures.goldRankInfo(),
            rankList: rankList
        ))

        XCTAssertFalse(hero.isMax)
        XCTAssertEqual(hero.nextName, "铂金大师")
        XCTAssertEqual(hero.earned, 280)
        XCTAssertEqual(hero.span, 500)
        XCTAssertEqual(hero.remaining, 220)
        XCTAssertEqual(hero.progressPercent, 56)
    }

    func testHeroModelReturnsMaxStateForKingWithoutNextRank() throws {
        let hero = try XCTUnwrap(RankExplainRules.heroModel(
            rankInfo: RankingFixtures.kingRankInfo(),
            rankList: rankList
        ))

        XCTAssertTrue(hero.isMax)
        XCTAssertEqual(hero.progressPercent, 100)
        XCTAssertNil(hero.remaining)
        XCTAssertNil(hero.nextName)
        XCTAssertNil(hero.earned)
        XCTAssertNil(hero.span)
    }

    func testHeroModelIsNilWhenRankInfoMissing() {
        XCTAssertNil(RankExplainRules.heroModel(rankInfo: nil, rankList: rankList))
    }

    func testHeroModelExposesRemainingPlacementMatches() throws {
        let withPlacement = try XCTUnwrap(RankExplainRules.heroModel(
            rankInfo: RankingFixtures.goldRankInfo(placementRemaining: 7),
            rankList: rankList
        ))
        XCTAssertEqual(withPlacement.placementRemaining, 7)

        let withoutPlacement = try XCTUnwrap(RankExplainRules.heroModel(
            rankInfo: RankingFixtures.rankInfo(level: 3, name: "黄金球手", rankScore: 1300),
            rankList: rankList
        ))
        XCTAssertEqual(withoutPlacement.placementRemaining, 0)
    }

    // MARK: - 规则文案

    func testLegacyScoringCopyMatchesLegacySettlement() {
        let rules = RankExplainRules.legacyRules
        let text = ruleText(rules)

        XCTAssertTrue(text.contains("+8～40"))
        XCTAssertTrue(text.contains("+500"))
        XCTAssertTrue(text.contains("败方基础扣分"))
        XCTAssertTrue(text.contains("第 4～6 场按 30%"))
        XCTAssertTrue(text.contains("第 7 场起不计"))
        XCTAssertTrue(text.contains("最低 +2"))
        XCTAssertTrue(text.contains("单日最多 200 分；输球不计"))
        XCTAssertTrue(rules.example.formula.contains("+20"))
        XCTAssertTrue(rules.example.formula.contains("+16"))
        XCTAssertFalse(text.contains("失败减免"))
        XCTAssertFalse(text.contains("减免到 -2"))
    }

    func testRatingV2ScoringCopyLocksSharedFormulaExamples() {
        let rules = RankExplainRules.ratingV2Rules
        let text = ruleText(rules)

        XCTAssertTrue(text.contains("±45"))
        XCTAssertTrue(text.contains("+68"))
        XCTAssertTrue(text.contains("只扣 21"))
        XCTAssertTrue(text.contains("抢 1 为 ±20，抢 3 为 ±35，抢 5 为 ±45，抢 9 为 ±60"))
        XCTAssertTrue(text.contains("5:0 与 5:4 分值相同"))
        XCTAssertTrue(text.contains("前 10 场为定级赛，涨跌加倍"))
        XCTAssertTrue(text.contains("涨跌减半"))
        XCTAssertTrue(text.contains("第 3 场按 80%，第 4～6 场按 30%，双方同比例"))
        XCTAssertTrue(text.contains("第 7 场起不计"))
        XCTAssertTrue(text.contains("每天最多上涨 500"))
        XCTAssertTrue(text.contains("不改变排位分，会员与普通用户规则相同"))
        XCTAssertTrue(text.contains("打满 5 场、遇到至少 3 位不同对手"))
        XCTAssertTrue(text.contains("不做重置"))
        XCTAssertFalse(text.contains("VIP"))
        XCTAssertFalse(text.contains("倍率"))
    }

    // MARK: - 规则切换

    func testRulesSwitchByEffectiveTimeEvenWhenCachedConfigStillSaysLegacy() throws {
        let effectiveAt = "2026-11-01T00:00:00+08:00"

        let scheduled = RankExplainRules.rules(
            rulesVersion: 1,
            rulesEffectiveAt: effectiveAt,
            now: try date("2026-10-31T23:59:59+08:00")
        )
        XCTAssertEqual(scheduled.version, 1)
        XCTAssertTrue(scheduled.notice.contains("2026年11月1日起生效"))
        XCTAssertTrue(scheduled.notice.contains("重算你的历史排位"))

        let active = RankExplainRules.rules(
            rulesVersion: 1,
            rulesEffectiveAt: effectiveAt,
            now: try date("2026-11-01T00:00:00+08:00")
        )
        XCTAssertEqual(active.version, 2)
        XCTAssertTrue(active.notice.contains("已于2026年11月1日生效"))

        let unconfigured = RankExplainRules.rules(
            rulesVersion: 1,
            rulesEffectiveAt: "",
            now: try date("2026-10-31T00:00:00+08:00")
        )
        XCTAssertEqual(unconfigured.version, 1)
        XCTAssertEqual(unconfigured.notice, "")
    }

    func testRulesVersionTwoIsActiveWithoutEffectiveTime() throws {
        let rules = RankExplainRules.rules(
            rulesVersion: 2,
            rulesEffectiveAt: "",
            now: try date("2026-10-31T00:00:00+08:00")
        )

        XCTAssertEqual(rules.version, 2)
        XCTAssertEqual(rules.notice, "")
    }

    // MARK: - CDN

    func testRankIconsResolveToCDN() throws {
        let hero = try XCTUnwrap(RankExplainRules.heroModel(
            rankInfo: RankingFixtures.goldRankInfo(),
            rankList: rankList
        ))
        let path = RankExplainRules.growthPath(rankList: rankList, currentLevel: 3, rankScore: 1280)

        XCTAssertEqual(
            CDNAsset.url(for: hero.icon)?.absoluteString,
            "https://cdn.zhuifen.cn/static/images/ranks/rank_gold.png"
        )
        XCTAssertEqual(
            CDNAsset.url(for: try XCTUnwrap(path.first { $0.level == 6 }).icon)?.absoluteString,
            "https://cdn.zhuifen.cn/static/images/ranks/rank_king.png"
        )
    }

    // MARK: - 辅助

    private func ruleText(_ rules: RankRules) -> String {
        (
            rules.summaryStats.map { "\($0.value) \($0.label)" }
                + [rules.note]
                + rules.detailRows.map { "\($0.title) \($0.description)" }
                + [rules.example.title, rules.example.formula]
        ).joined(separator: " ")
    }

    private func date(_ text: String) throws -> Date {
        try XCTUnwrap(ISO8601DateFormatter().date(from: text))
    }
}
