import Foundation
import XCTest
@testable import iOS

/// 逐条对应小程序 `uniapp/tests/saixun.test.mjs` 的 38 条用例。
@MainActor
final class SaiXunRulesTests: XCTestCase {
    private func time(_ iso: String) -> Date { EventNewsFixtures.time(iso) }

    // MARK: - 封面常量

    func testDefaultEventCoverIsServedFromCdnInsteadOfMiniProgramMainPackage() {
        XCTAssertEqual(
            SaiXunRules.defaultEventCover.absoluteString,
            "https://cdn.zhuifen.cn/static/images/default-event-cover.png"
        )
    }

    // MARK: - 日期与时间

    func testFormatEventDateRangePrefersDateOnlyOutput() {
        XCTAssertEqual(SaiXunRules.formatEventDateRange("2026-03-30", "2026-04-05"), "3月30日 - 4月5日")
        XCTAssertEqual(SaiXunRules.formatEventDateRange("2026-03-30", "2026-03-30"), "3月30日")
        XCTAssertEqual(SaiXunRules.formatEventDateRange("", ""), "日期待定")
    }

    func testFormatEventTimeRangeReturnsEmptyStringWhenEventHasNoExplicitTime() {
        XCTAssertEqual(
            SaiXunRules.formatEventTimeRange("", "", now: time("2026-04-01T12:00:00+08:00")),
            ""
        )
        XCTAssertEqual(
            SaiXunRules.formatEventTimeRange(
                "2026-04-01T20:00:00+08:00",
                "2026-04-05T20:00:00+08:00",
                now: time("2026-04-01T12:00:00+08:00")
            ),
            "今天 20:00 - 4月5日 20:00"
        )
    }

    // MARK: - 列表卡片

    func testNormalizeSaiXunCardMapsSummaryAndFallsBackToDefaultCover() {
        let card = SaiXunRules.normalizeSaiXunCard(
            EventNewsFixtures.eventNewsModel(
                id: 7,
                title: "斯诺克公开赛总决赛",
                sourceName: "WST",
                tournamentName: "Sportsbet.io Tour Championship 2026",
                gameType: 1,
                city: "曼彻斯特",
                venue: "Manchester Central",
                startDate: "2026-03-30",
                endDate: "2026-04-05",
                status: 1,
                currentRoundText: "Semi Finals",
                latestResultText: "今晚决出冠军",
                matchCount: 11
            ),
            now: time("2026-04-01T12:00:00+08:00")
        )

        XCTAssertEqual(card.id, 7)
        XCTAssertEqual(card.title, "斯诺克巡回锦标赛 2026")
        XCTAssertEqual(card.summary, "今晚决出冠军")
        XCTAssertEqual(card.statusText, "进行中")
        XCTAssertEqual(card.dateText, "3月30日 - 4月5日")
        XCTAssertEqual(card.timeText, "")
        XCTAssertEqual(card.showTime, false)
        XCTAssertEqual(card.locationText, "曼彻斯特 · Manchester Central")
        XCTAssertEqual(card.currentRoundText, "半决赛")
        XCTAssertEqual(card.matchCountText, "11 场比赛")
        XCTAssertEqual(card.sourceText, "WST")
        XCTAssertEqual(card.coverImage, .remote(SaiXunRules.defaultEventCover))
    }

    func testNormalizeSaiXunCardKeepsMirroredCdnCoverURL() {
        let cover = "https://cdn.example.com/wst/tournaments/tour-cover.png"
        let card = SaiXunRules.normalizeSaiXunCard(
            EventNewsFixtures.eventNewsModel(id: 8, title: "Tour Championship 2026", coverImage: cover),
            now: time("2026-04-01T12:00:00+08:00")
        )

        XCTAssertEqual(card.coverImage, .remote(URL(string: cover)!))
    }

    // MARK: - 赛事名

    func testLocalizeTournamentTitleStripsSponsorPrefixAndKeepsYear() {
        XCTAssertEqual(SaiXunRules.localizeTournamentTitle("Sportsbet.io Tour Championship 2026"), "斯诺克巡回锦标赛 2026")
        XCTAssertEqual(SaiXunRules.localizeTournamentTitle("BetVictor Scottish Open 2025"), "苏格兰公开赛 2025")
        XCTAssertEqual(SaiXunRules.localizeTournamentTitle("Victorian Plumbing UK Championship 2025"), "英国锦标赛 2025")
        XCTAssertEqual(SaiXunRules.localizeTournamentTitle("Sportsbet.io Champion of Champions"), "冠中冠")
    }

    func testLocalizeTournamentTitleTranslatesChampionshipLeaguePhasesAndFallsBack() {
        XCTAssertEqual(
            SaiXunRules.localizeTournamentTitle("BetVictor Championship League Snooker 2025 (Stage One/WK1)"),
            "冠军联赛 2025（第一阶段 / 第1周）"
        )
        XCTAssertEqual(
            SaiXunRules.localizeTournamentTitle("Championship League Group Three"),
            "冠军联赛第3组"
        )
        XCTAssertEqual(SaiXunRules.localizeTournamentTitle("Unknown Cup 2025"), "Unknown Cup 2025")
    }

    func testLocalizeTournamentTitleKeepsQualifierPhasesApartFromTheMainEvent() {
        XCTAssertEqual(SaiXunRules.localizeTournamentTitle("BetVictor Northern Ireland Open 2026"), "北爱尔兰公开赛 2026")
        XCTAssertEqual(
            SaiXunRules.localizeTournamentTitle("BetVictor Northern Ireland Open 2026 Qualifiers"),
            "北爱尔兰公开赛 2026 资格赛"
        )
        XCTAssertEqual(
            SaiXunRules.localizeTournamentTitle("Halo World Championship 2026 Qualifiers"),
            "世界锦标赛 2026 资格赛"
        )
        XCTAssertEqual(
            SaiXunRules.localizeTournamentTitle("Saudi Arabia Snooker Masters 2024 Final Stages"),
            "沙特阿拉伯斯诺克大师赛 2024 决赛阶段"
        )
        XCTAssertEqual(SaiXunRules.localizeTournamentTitle("Shenzhen Open 2026"), "深圳公开赛 2026")
        XCTAssertEqual(
            SaiXunRules.localizeTournamentTitle("Riyadh Season World Masters of Snooker 2024"),
            "世界斯诺克大师赛 2024"
        )
        XCTAssertEqual(SaiXunRules.localizeTournamentTitle("World Mixed Doubles 2024"), "世界混双赛 2024")
        XCTAssertEqual(
            SaiXunRules.localizeTournamentTitle("Asia-Oceania Q School 2026"),
            "亚洲及大洋洲 Q School 2026"
        )
    }

    func testLocalizeTournamentTitleCoversSponsoredChampionshipLeagueGroups() {
        XCTAssertEqual(
            SaiXunRules.localizeTournamentTitle("BetVictor Championship League Snooker Invitational 2026 (Group 7)"),
            "冠军联赛 2026（第7组）"
        )
        XCTAssertEqual(
            SaiXunRules.localizeTournamentTitle("BetVictor Championship League Snooker Invitational 2026 (Winners Group)"),
            "冠军联赛 2026（胜者组）"
        )
        XCTAssertEqual(
            SaiXunRules.localizeTournamentTitle("BetVictor Championship League Group 7"),
            "冠军联赛第7组"
        )
        XCTAssertEqual(
            SaiXunRules.localizeTournamentTitle("BetVictor Championship League Winners Group"),
            "冠军联赛胜者组"
        )
    }

    // MARK: - 轮次

    func testStandardizeRoundTextLocalizesWstRoundNames() {
        XCTAssertEqual(SaiXunRules.standardizeRoundText("Round 1"), "第1轮")
        XCTAssertEqual(SaiXunRules.standardizeRoundText("Round 2 (Held Over)"), "第2轮（保留赛）")
        XCTAssertEqual(SaiXunRules.standardizeRoundText("Quarter Finals"), "四分之一决赛")
        XCTAssertEqual(SaiXunRules.standardizeRoundText("Last 16"), "16强")
        XCTAssertEqual(
            SaiXunRules.standardizeRoundText("League Phase (STAGE TWO / WEEK 2)"),
            "联赛阶段（第二阶段 / 第2周）"
        )
        XCTAssertEqual(SaiXunRules.standardizeRoundText("Stage One/WK2"), "第一阶段 / 第2周")
        XCTAssertEqual(SaiXunRules.standardizeRoundText("小组赛"), "小组赛")
    }

    func testStandardizeRoundTextLocalizesGroupKnockoutLeagueAndPreQualifierRounds() {
        XCTAssertEqual(SaiXunRules.standardizeRoundText("Group Semi Finals"), "小组半决赛")
        XCTAssertEqual(SaiXunRules.standardizeRoundText("Group Final"), "小组决赛")
        XCTAssertEqual(SaiXunRules.standardizeRoundText("Semi-Finals"), "半决赛")
        XCTAssertEqual(SaiXunRules.standardizeRoundText("Semi Final"), "半决赛")
        XCTAssertEqual(SaiXunRules.standardizeRoundText("Quarter Final"), "四分之一决赛")
        XCTAssertEqual(SaiXunRules.standardizeRoundText("League Phase"), "联赛阶段")
        XCTAssertEqual(SaiXunRules.standardizeRoundText("League Phase (STAGE ONE)"), "联赛阶段（第一阶段）")
        XCTAssertEqual(SaiXunRules.standardizeRoundText("Pre-Qualifier"), "预资格赛")
        XCTAssertEqual(SaiXunRules.standardizeRoundText("Pre-Qualifier Round"), "预资格赛")
        XCTAssertEqual(SaiXunRules.standardizeRoundText("Wildcard Round"), "外卡赛")
        XCTAssertEqual(SaiXunRules.standardizeRoundText("Round 1 (held over)"), "第1轮（保留赛）")
        XCTAssertEqual(SaiXunRules.standardizeRoundText("Round 1/Heldover"), "第1轮（保留赛）")
        XCTAssertEqual(SaiXunRules.standardizeRoundText("Group 3"), "第3组")
    }

    // MARK: - 球员名

    func testLocalizeMatchPlayerNameTranslatesBracketPlaceholdersAndKeepsRealNames() {
        XCTAssertEqual(SaiXunRules.localizeMatchPlayerName("Winner of Match 7"), "第7场胜者")
        XCTAssertEqual(SaiXunRules.localizeMatchPlayerName("Loser of Match 3"), "第3场负者")
        XCTAssertEqual(SaiXunRules.localizeMatchPlayerName("Winner Semi One"), "半决赛1胜者")
        XCTAssertEqual(SaiXunRules.localizeMatchPlayerName("Winner Semi Two"), "半决赛2胜者")
        XCTAssertEqual(SaiXunRules.localizeMatchPlayerName("TBC"), "待定")
        XCTAssertEqual(SaiXunRules.localizeMatchPlayerName("Mark Selby"), "Mark Selby")
        XCTAssertEqual(SaiXunRules.localizeMatchPlayerName(""), "")
    }

    // MARK: - 详情轮次

    func testBuildSaiXunDetailRoundsOrdersRoundsAndHidesAllPlaceholderRounds() {
        let rounds = SaiXunRules.buildSaiXunDetailRounds([
            EventNewsFixtures.matchModel(
                id: 101,
                roundName: "Quarter Finals",
                roundOrder: 2,
                matchOrder: 2,
                startTime: "2026-04-01T20:00:00+08:00",
                status: 2,
                bestOf: 19,
                homePlayerName: "Neil Robertson",
                homePlayerFirstName: "Neil",
                homePlayerLastName: "Robertson",
                homePlayerFlagEmoji: "🇦🇺",
                homePlayerCountryCode: "au",
                awayPlayerName: "Barry Hawkins",
                awayPlayerFirstName: "Barry",
                awayPlayerLastName: "Hawkins",
                awayPlayerFlagEmoji: "🏴",
                awayPlayerCountryCode: "gb-eng",
                homeScore: 10,
                awayScore: 8,
                winnerSide: 1
            ),
            EventNewsFixtures.matchModel(
                id: 102,
                sourceType: "official",
                roundName: "Semi Finals",
                roundOrder: 3,
                matchOrder: 1,
                startTime: "2026-04-03T20:00:00+08:00",
                status: 0,
                bestOf: 19,
                homePlayerName: "Judd Trump",
                homePlayerAvatar: "https://cdn.example.com/wst/players/trump.png",
                awayPlayerName: "待定"
            ),
            EventNewsFixtures.matchModel(
                id: 103,
                roundName: "Final",
                roundOrder: 4,
                matchOrder: 1,
                startTime: "2026-04-05T20:00:00+08:00",
                isPlaceholder: true
            ),
        ], now: time("2026-04-03T12:00:00+08:00"))

        XCTAssertEqual(rounds.count, 2)
        XCTAssertEqual(rounds[0].roundName, "四分之一决赛")
        XCTAssertEqual(rounds[1].roundName, "半决赛")
        XCTAssertEqual(
            rounds[1].matches[0].homePlayerAvatar,
            .remote(URL(string: "https://cdn.example.com/wst/players/trump.png")!)
        )
        XCTAssertEqual(rounds[1].matches[0].awayPlayerAvatar, SaiXunRules.officialPlayerPlaceholder)

        let quarterFinal = rounds[0].matches[0]
        XCTAssertEqual(quarterFinal.scoreText, "10 : 8")
        XCTAssertTrue(quarterFinal.showScore)
        XCTAssertEqual(quarterFinal.homeScore, 10)
        XCTAssertEqual(quarterFinal.awayScore, 8)
        XCTAssertEqual(quarterFinal.metaText, "19局10胜")
        XCTAssertEqual(quarterFinal.homeResultText, "胜")
        XCTAssertEqual(quarterFinal.awayResultText, "败")
        XCTAssertEqual(quarterFinal.homePlayerFirstName, "Neil")
        XCTAssertEqual(quarterFinal.homePlayerLastName, "Robertson")
        XCTAssertEqual(quarterFinal.homePlayerFlagEmoji, "🇦🇺")
        XCTAssertEqual(quarterFinal.homePlayerCountryCode, "au")
        XCTAssertEqual(quarterFinal.homePlayerFlag, SaiXunFlag(type: .emoji, value: "🇦🇺"))
        XCTAssertEqual(quarterFinal.awayPlayerFirstName, "Barry")
        XCTAssertEqual(quarterFinal.awayPlayerLastName, "Hawkins")
        XCTAssertEqual(quarterFinal.awayPlayerFlagEmoji, "")
        XCTAssertEqual(quarterFinal.awayPlayerCountryCode, "gb-eng")
        XCTAssertEqual(quarterFinal.awayPlayerFlag, SaiXunFlag(type: .image, value: "/static/flags/eng.png"))

        XCTAssertFalse(rounds.contains { $0.roundName == "决赛" })
        XCTAssertNil(SaiXunRules.resolveSaiXunChampion(rounds))
    }

    func testOfficialMatchesUseOfficialPlaceholderWhilePlatformOnesKeepAppAvatar() {
        func rounds(sourceType: String) -> [SaiXunRound] {
            SaiXunRules.buildSaiXunDetailRounds([
                EventNewsFixtures.matchModel(
                    id: 1,
                    sourceType: sourceType,
                    roundName: "Round 1",
                    roundOrder: 1,
                    matchOrder: 1,
                    status: 2
                ),
            ], now: Date())
        }

        let official = rounds(sourceType: "official")
        let platform = rounds(sourceType: "")

        XCTAssertEqual(official[0].matches[0].homePlayerAvatar, SaiXunRules.officialPlayerPlaceholder)
        XCTAssertEqual(official[0].matches[0].awayPlayerAvatar, SaiXunRules.officialPlayerPlaceholder)
        XCTAssertEqual(platform[0].matches[0].homePlayerAvatar, SaiXunRules.defaultPlayerAvatar)
        XCTAssertEqual(platform[0].matches[0].awayPlayerAvatar, SaiXunRules.defaultPlayerAvatar)
    }

    func testBuildSaiXunDetailRoundsPromotesOverdueScheduledMatchesToLive() {
        let rounds = SaiXunRules.buildSaiXunDetailRounds([
            EventNewsFixtures.matchModel(
                id: 201,
                roundName: "Semi Finals",
                roundOrder: 3,
                matchOrder: 1,
                startTime: "2026-04-04T20:00:00+08:00",
                homePlayerName: "John Higgins",
                awayPlayerName: "Zhao Xintong"
            ),
        ], now: time("2026-04-04T20:53:00+08:00"))

        XCTAssertEqual(rounds[0].matches[0].status, 1)
        XCTAssertEqual(rounds[0].matches[0].statusText, "进行中")
    }

    func testBuildSaiXunDetailRoundsPromotesOverdueScoredMatchesWithWinnerToCompleted() {
        let rounds = SaiXunRules.buildSaiXunDetailRounds([
            EventNewsFixtures.matchModel(
                id: 202,
                roundName: "Quarter Finals",
                roundOrder: 2,
                matchOrder: 1,
                startTime: "2026-04-04T18:00:00+08:00",
                homePlayerName: "Neil Robertson",
                awayPlayerName: "Barry Hawkins",
                homeScore: 10,
                awayScore: 8,
                winnerSide: 1
            ),
        ], now: time("2026-04-04T20:53:00+08:00"))

        XCTAssertEqual(rounds[0].matches[0].status, 2)
        XCTAssertEqual(rounds[0].matches[0].statusText, "已结束")
        XCTAssertEqual(rounds[0].matches[0].homeResultText, "胜")
        XCTAssertEqual(rounds[0].matches[0].awayResultText, "败")
    }

    func testBuildSaiXunDetailRoundsMarksStaleOverdueMatchesAsCompleted() {
        let rounds = SaiXunRules.buildSaiXunDetailRounds([
            EventNewsFixtures.matchModel(
                id: 203,
                roundName: "Semi Finals",
                roundOrder: 3,
                matchOrder: 1,
                startTime: "2026-04-03T20:00:00+08:00",
                homePlayerName: "Neil Robertson",
                awayPlayerName: "Judd Trump"
            ),
            EventNewsFixtures.matchModel(
                id: 204,
                roundName: "Semi Finals",
                roundOrder: 3,
                matchOrder: 2,
                startTime: "2026-04-04T20:00:00+08:00",
                homePlayerName: "John Higgins",
                awayPlayerName: "Zhao Xintong"
            ),
        ], now: time("2026-04-04T21:03:00+08:00"))

        let staleMatch = rounds[0].matches.first { $0.id == 203 }
        let liveMatch = rounds[0].matches.first { $0.id == 204 }
        XCTAssertEqual(rounds[0].matches.map(\.id), [203, 204])
        XCTAssertEqual(liveMatch?.status, 1)
        XCTAssertEqual(liveMatch?.statusText, "进行中")
        XCTAssertEqual(liveMatch?.isLive, true)
        XCTAssertEqual(liveMatch?.showScore, false)
        XCTAssertEqual(staleMatch?.status, 2)
        XCTAssertEqual(staleMatch?.statusText, "已结束")
        XCTAssertEqual(staleMatch?.showScore, false)
        XCTAssertEqual(liveMatch?.homeResultText, "")
        XCTAssertEqual(liveMatch?.awayResultText, "")
        XCTAssertEqual(staleMatch?.homeResultText, "")
        XCTAssertEqual(staleMatch?.awayResultText, "")
        XCTAssertEqual(
            SaiXunRules.buildSaiXunScheduleDays(rounds[0].matches).map { [$0.label, String($0.matches.count)] },
            [["4月3日 周五 · 昨天", "1"], ["4月4日 周六 · 今天", "1"]]
        )
    }

    func testBuildSaiXunDetailRoundsNormalizesSubdivisionFlagsToImageAndFiltersBadEmoji() {
        let rounds = SaiXunRules.buildSaiXunDetailRounds([
            EventNewsFixtures.matchModel(
                id: 301,
                roundName: "Final",
                roundOrder: 5,
                matchOrder: 1,
                status: 2,
                homePlayerName: "Judd Trump",
                homePlayerFlagEmoji: "🇬🇧",
                homePlayerCountryCode: "gb-eng",
                awayPlayerName: "Mark Williams",
                awayPlayerFlagEmoji: "🏴",
                awayPlayerCountryCode: "gb-wls",
                winnerSide: 1
            ),
        ], now: time("2026-04-06T20:00:00+08:00"))

        XCTAssertEqual(rounds[0].matches[0].homePlayerFlagEmoji, "🇬🇧")
        XCTAssertEqual(rounds[0].matches[0].homePlayerFlag, SaiXunFlag(type: .image, value: "/static/flags/eng.png"))
        XCTAssertEqual(rounds[0].matches[0].awayPlayerFlagEmoji, "")
        XCTAssertEqual(rounds[0].matches[0].awayPlayerFlag, SaiXunFlag(type: .image, value: "/static/flags/wls.png"))
    }

    func testBuildSaiXunDetailRoundsShowsChinesePlaceholderNamesInScheduleAndHighlight() {
        let rounds = SaiXunRules.buildSaiXunDetailRounds([
            EventNewsFixtures.matchModel(
                id: 1,
                roundName: "Group Final",
                roundOrder: 0,
                startTime: "2026-11-13T11:00:00+08:00",
                homePlayerName: "Winner of Match 7",
                awayPlayerName: "Winner of Match 8"
            ),
        ], now: time("2026-11-12T12:00:00+08:00"))

        XCTAssertEqual(rounds[0].roundName, "小组决赛")
        XCTAssertEqual(rounds[0].matches[0].homePlayerName, "第7场胜者")
        XCTAssertEqual(rounds[0].matches[0].awayPlayerName, "第8场胜者")

        let card = SaiXunRules.normalizeSaiXunCard(
            EventNewsFixtures.eventNewsModel(
                status: 0,
                currentRoundText: "Final",
                latestResultText: "Winner Semi One - Winner Semi Two"
            ),
            now: time("2026-11-12T12:00:00+08:00")
        )
        XCTAssertEqual(card.highlight?.primary, "半决赛1胜者 vs 半决赛2胜者")
    }

    // MARK: - 国旗

    func testSanitizeFlagEmojiFiltersBlackFlagBaseAndAllSubdivisionFlags() {
        XCTAssertEqual(SaiXunRules.sanitizeFlagEmoji(""), "")
        XCTAssertEqual(SaiXunRules.sanitizeFlagEmoji("🇦🇺"), "🇦🇺")
        XCTAssertEqual(SaiXunRules.sanitizeFlagEmoji("🇨🇳"), "🇨🇳")
        XCTAssertEqual(SaiXunRules.sanitizeFlagEmoji("🏴"), "")

        let englandFlag = "\u{1F3F4}\u{E0067}\u{E0062}\u{E0065}\u{E006E}\u{E0067}\u{E007F}"
        XCTAssertEqual(englandFlag.unicodeScalars.first?.value, 0x1F3F4)
        XCTAssertEqual(SaiXunRules.sanitizeFlagEmoji(englandFlag), "")

        let wrongEngFlag = "\u{1F3F4}\u{E0065}\u{E006E}\u{E0067}\u{E007F}"
        XCTAssertEqual(SaiXunRules.sanitizeFlagEmoji(wrongEngFlag), "")
    }

    func testResolvePlayerFlagReturnsLocalImageForEnglandScotlandWales() {
        XCTAssertEqual(SaiXunRules.resolvePlayerFlag("", "gb-eng"), SaiXunFlag(type: .image, value: "/static/flags/eng.png"))
        XCTAssertEqual(SaiXunRules.resolvePlayerFlag("", "gb-sct"), SaiXunFlag(type: .image, value: "/static/flags/sco.png"))
        XCTAssertEqual(SaiXunRules.resolvePlayerFlag("", "gb-wls"), SaiXunFlag(type: .image, value: "/static/flags/wls.png"))
        XCTAssertEqual(SaiXunRules.resolvePlayerFlag("", "eng"), SaiXunFlag(type: .image, value: "/static/flags/eng.png"))
        XCTAssertEqual(SaiXunRules.resolvePlayerFlag("", "sct"), SaiXunFlag(type: .image, value: "/static/flags/sco.png"))
        XCTAssertEqual(SaiXunRules.resolvePlayerFlag("", "wls"), SaiXunFlag(type: .image, value: "/static/flags/wls.png"))
    }

    func testResolvePlayerFlagReturnsEmojiForNormalCountryCodes() {
        XCTAssertEqual(SaiXunRules.resolvePlayerFlag("🇦🇺", "au"), SaiXunFlag(type: .emoji, value: "🇦🇺"))
        XCTAssertEqual(SaiXunRules.resolvePlayerFlag("🇨🇳", "cn"), SaiXunFlag(type: .emoji, value: "🇨🇳"))
        XCTAssertEqual(SaiXunRules.resolvePlayerFlag("🇧🇪", "be"), SaiXunFlag(type: .emoji, value: "🇧🇪"))
    }

    func testResolvePlayerFlagReturnsImageForNorthernIreland() {
        XCTAssertEqual(SaiXunRules.resolvePlayerFlag("", "gb-nir"), SaiXunFlag(type: .image, value: "/static/flags/nir.png"))
        XCTAssertEqual(SaiXunRules.resolvePlayerFlag("", "nir"), SaiXunFlag(type: .image, value: "/static/flags/nir.png"))
    }

    func testResolvePlayerFlagReturnsNoneForUnknownOrEmptyCodes() {
        XCTAssertEqual(SaiXunRules.resolvePlayerFlag("", "gb-xxx"), SaiXunFlag(type: .none, value: ""))
        XCTAssertEqual(SaiXunRules.resolvePlayerFlag("", ""), SaiXunFlag(type: .none, value: ""))
        XCTAssertEqual(SaiXunRules.resolvePlayerFlag("", "xx"), SaiXunFlag(type: .none, value: ""))
    }

    func testResolvePlayerFlagPrefersImageOverEmojiForSubdivisionCodes() {
        XCTAssertEqual(SaiXunRules.resolvePlayerFlag("🏴", "gb-eng"), SaiXunFlag(type: .image, value: "/static/flags/eng.png"))
        XCTAssertEqual(SaiXunRules.resolvePlayerFlag("🏴", "gb-sct"), SaiXunFlag(type: .image, value: "/static/flags/sco.png"))
    }

    // MARK: - 官方封面

    func testOfficialEventsWithoutCoverFallBackToWstDefaultCover() {
        XCTAssertEqual(
            SaiXunRules.officialDefaultEventCover.absoluteString,
            "https://cdn.zhuifen.cn/static/images/wst-default-event-cover.png"
        )

        let official = SaiXunRules.normalizeSaiXunCard(
            EventNewsFixtures.eventNewsModel(id: 47, sourceType: "official"),
            now: time("2026-04-01T12:00:00+08:00")
        )
        XCTAssertTrue(official.hasCover)
        XCTAssertEqual(official.coverImage, .remote(SaiXunRules.officialDefaultEventCover))

        let mirrored = "https://cdn.zhuifen.cn/wst/tournaments/cover.jpg"
        let mirroredCard = SaiXunRules.normalizeSaiXunCard(
            EventNewsFixtures.eventNewsModel(sourceType: "official", coverImage: mirrored),
            now: time("2026-04-01T12:00:00+08:00")
        )
        XCTAssertEqual(mirroredCard.coverImage, .remote(URL(string: mirrored)!))

        let platform = SaiXunRules.normalizeSaiXunCard(
            EventNewsFixtures.eventNewsModel(id: 9, sourceType: "platform"),
            now: time("2026-04-01T12:00:00+08:00")
        )
        XCTAssertFalse(platform.hasCover)
        XCTAssertEqual(platform.coverImage, .remote(SaiXunRules.defaultEventCover))
    }

    // MARK: - 赛程

    func testScheduleSplitsAllRoundsIntoEndedLatestFirstAndUpcomingSoonestFirst() {
        let now = time("2026-11-12T12:00:00+08:00")
        let rounds = SaiXunRules.buildSaiXunDetailRounds([
            EventNewsFixtures.matchModel(id: 1, roundName: "Group Semi Finals", startTime: "2026-11-10T05:00:00+08:00", status: 2, homeScore: 4, awayScore: 3, winnerSide: 1),
            EventNewsFixtures.matchModel(id: 2, roundName: "Group Final", startTime: "2026-11-10T11:00:00+08:00", status: 2, homeScore: 2, awayScore: 6, winnerSide: 2),
            EventNewsFixtures.matchModel(id: 3, roundName: "Group Semi Finals", startTime: "2026-11-12T11:00:00+08:00", status: 1, homeScore: 2, awayScore: 1),
            EventNewsFixtures.matchModel(id: 4, roundName: "Group Semi Finals", startTime: "2026-11-13T05:00:00+08:00"),
            EventNewsFixtures.matchModel(id: 5, roundName: "Group Semi Finals", startTime: "2026-11-11T06:00:00+08:00", status: 3),
            EventNewsFixtures.matchModel(id: 6, roundName: "Final", startTime: "2026-11-16T05:00:00+08:00", homePlayerName: "Winner Semi One", awayPlayerName: "Winner Semi Two"),
            EventNewsFixtures.matchModel(id: 7, roundName: "Semi-Finals"),
            EventNewsFixtures.matchModel(id: 8, roundName: "Group Final", status: 2, homeScore: 6, awayScore: 0, winnerSide: 1),
        ], now: now)

        let tabs = SaiXunRules.buildSaiXunScheduleTabs(rounds)
        XCTAssertEqual(tabs.map(\.key), ["ended", "upcoming"])
        XCTAssertEqual(tabs.map(\.label), ["已结束", "即将开始"])
        XCTAssertEqual(tabs[0].matches.map(\.id), [5, 2, 1, 8])
        XCTAssertEqual(tabs[1].matches.map(\.id), [3, 4, 6, 7])
        XCTAssertEqual(tabs[1].matches.map(\.roundName), ["小组半决赛", "小组半决赛", "决赛", "半决赛"])
        XCTAssertEqual(
            SaiXunRules.buildSaiXunScheduleDays(tabs[0].matches).map { [$0.label, $0.matches.map(\.id).map(String.init).joined(separator: ",")] },
            [
                ["11月11日 周三 · 昨天", "5"],
                ["11月10日 周二", "2,1"],
                ["时间待定", "8"],
            ]
        )
    }

    func testScheduleTabDefaultsToUpcomingWhileMatchesRemainAndKeepsManualChoice() {
        let now = time("2026-11-12T12:00:00+08:00")
        let mixed = SaiXunRules.buildSaiXunScheduleTabs(SaiXunRules.buildSaiXunDetailRounds([
            EventNewsFixtures.matchModel(id: 1, roundName: "Final", startTime: "2026-11-10T05:00:00+08:00", status: 2, homeScore: 10, awayScore: 8, winnerSide: 1),
            EventNewsFixtures.matchModel(id: 2, roundName: "Final", startTime: "2026-11-13T05:00:00+08:00"),
        ], now: now))
        XCTAssertEqual(SaiXunRules.resolveSaiXunScheduleTabKey(mixed, selectedKey: ""), "upcoming")
        XCTAssertEqual(SaiXunRules.resolveSaiXunScheduleTabKey(mixed, selectedKey: "ended"), "ended")

        let finished = SaiXunRules.buildSaiXunScheduleTabs(SaiXunRules.buildSaiXunDetailRounds([
            EventNewsFixtures.matchModel(id: 1, roundName: "Final", startTime: "2026-11-10T05:00:00+08:00", status: 2, homeScore: 10, awayScore: 8, winnerSide: 1),
        ], now: now))
        XCTAssertEqual(SaiXunRules.resolveSaiXunScheduleTabKey(finished, selectedKey: ""), "ended")
        XCTAssertEqual(SaiXunRules.resolveSaiXunScheduleTabKey(finished, selectedKey: "upcoming"), "upcoming")
        XCTAssertEqual(SaiXunRules.resolveSaiXunScheduleTabKey([], selectedKey: ""), "ended")
    }

    func testScheduleDaysRenderInBatchesForLargeTournaments() {
        let matches = SaiXunRules.buildSaiXunDetailRounds(
            (0..<70).map { index in
                EventNewsFixtures.matchModel(
                    id: Int64(index + 1),
                    roundName: "Round 1",
                    startTime: "2026-11-\(13 + index / 20)T10:00:00+08:00"
                )
            },
            now: time("2026-11-12T12:00:00+08:00")
        )[0].matches

        func count(_ days: [SaiXunScheduleDay]) -> Int {
            days.reduce(0) { $0 + $1.matches.count }
        }
        XCTAssertEqual(SaiXunRules.schedulePageSize, 30)
        XCTAssertEqual(count(SaiXunRules.buildSaiXunScheduleDays(matches)), 30)
        XCTAssertEqual(count(SaiXunRules.buildSaiXunScheduleDays(matches, limit: 60)), 60)
        XCTAssertEqual(count(SaiXunRules.buildSaiXunScheduleDays(matches, limit: 90)), 70)
    }

    func testHeldOverRoundsSharingRoundOrderStayInScheduleOrder() {
        let rounds = SaiXunRules.buildSaiXunDetailRounds([
            EventNewsFixtures.matchModel(id: 3, roundName: "Round 2 (Held Over)", roundOrder: 0, matchOrder: 1, startTime: "2026-09-28T19:00:00+08:00", status: 2),
            EventNewsFixtures.matchModel(id: 1, roundName: "Round 1 (Held Over)", roundOrder: 0, matchOrder: 1, startTime: "2026-09-28T09:30:00+08:00", status: 2),
            EventNewsFixtures.matchModel(id: 4, roundName: "Round 3", roundOrder: 30, matchOrder: 1, startTime: "2026-09-29T14:00:00+08:00", status: 2),
        ], now: time("2026-10-17T12:00:00+08:00"))

        XCTAssertEqual(rounds.map(\.roundName), ["第1轮（保留赛）", "第2轮（保留赛）", "第3轮"])
        XCTAssertEqual(rounds[0].matches[0].clockText, "09:30")
    }

    // MARK: - 赛制与倒计时

    func testFormatBestOfTextFollowsChineseFrameFormatConvention() {
        XCTAssertEqual(SaiXunRules.formatBestOfText(9), "9局5胜")
        XCTAssertEqual(SaiXunRules.formatBestOfText(35), "35局18胜")
        XCTAssertEqual(SaiXunRules.formatBestOfText(4), "共4局")
        XCTAssertEqual(SaiXunRules.formatBestOfText(0), "")
    }

    func testFormatEventCountdownTextCountsCalendarDaysToEventStart() {
        let now = time("2026-10-17T19:30:00+08:00")
        XCTAssertEqual(SaiXunRules.formatEventCountdownText("2026-10-17", now: now), "今天开赛")
        XCTAssertEqual(SaiXunRules.formatEventCountdownText("2026-10-18", now: now), "明天开赛")
        XCTAssertEqual(SaiXunRules.formatEventCountdownText("2026-10-24", now: now), "7天后开赛")
        XCTAssertEqual(SaiXunRules.formatEventCountdownText("2026-12-01", now: now), "12月1日开赛")
        XCTAssertEqual(SaiXunRules.formatEventCountdownText("2026-10-10", now: now), "即将开始")
        XCTAssertEqual(SaiXunRules.formatEventCountdownText("", now: now), "即将开始")
    }

    func testNormalizeSaiXunCardDerivesChampionHighlightOnlyFromFinishedFinal() {
        let finished = SaiXunRules.normalizeSaiXunCard(
            EventNewsFixtures.eventNewsModel(
                id: 164,
                tournamentName: "Shenzhen Open 2026",
                country: "China",
                city: "Shenzhen",
                startDate: "2026-09-28",
                endDate: "2026-10-04",
                status: 2,
                currentRoundText: "Final",
                latestResultText: "Yuan Sijun 5 - 10 Wu Yize"
            ),
            now: time("2026-10-17T12:00:00+08:00")
        )

        XCTAssertEqual(finished.title, "深圳公开赛 2026")
        XCTAssertEqual(finished.originalTitle, "Shenzhen Open 2026")
        XCTAssertEqual(finished.statusTone, .ended)
        XCTAssertFalse(finished.isFeatured)
        XCTAssertEqual(finished.metaText, "9月28日 - 10月4日 · 中国 · Shenzhen")
        XCTAssertEqual(
            finished.highlight,
            SaiXunCardHighlight(type: .champion, label: "冠军", primary: "Wu Yize", secondary: "决赛 10-5 胜 Yuan Sijun")
        )

        let qualifier = SaiXunRules.normalizeSaiXunCard(
            EventNewsFixtures.eventNewsModel(
                tournamentName: "BetVictor Scottish Open 2026 Qualifiers",
                status: 2,
                currentRoundText: "Round 2",
                latestResultText: "Liu Hongyu 4 - 3 Jimmy White"
            ),
            now: time("2026-10-17T12:00:00+08:00")
        )
        XCTAssertNil(qualifier.highlight)

        let upcoming = SaiXunRules.normalizeSaiXunCard(
            EventNewsFixtures.eventNewsModel(
                tournamentName: "BetVictor Northern Ireland Open 2026",
                startDate: "2026-10-18",
                status: 0,
                currentRoundText: "Round 1",
                latestResultText: "Jack Lisowski - Huang Jiahao"
            ),
            now: time("2026-10-17T12:00:00+08:00")
        )
        XCTAssertTrue(upcoming.isFeatured)
        XCTAssertEqual(upcoming.statusTone, .upcoming)
        XCTAssertEqual(upcoming.statusLabel, "明天开赛")
        XCTAssertEqual(
            upcoming.highlight,
            SaiXunCardHighlight(type: .upcoming, label: "第1轮", primary: "Jack Lisowski vs Huang Jiahao", secondary: "")
        )

        let live = SaiXunRules.normalizeSaiXunCard(
            EventNewsFixtures.eventNewsModel(
                status: 1,
                currentRoundText: "Semi Finals",
                latestResultText: "Judd Trump 3 - 2 Mark Selby"
            ),
            now: time("2026-10-17T12:00:00+08:00")
        )
        XCTAssertEqual(live.statusLabel, "进行中")
        XCTAssertEqual(live.highlight?.primary, "Judd Trump 3 - 2 Mark Selby")
        XCTAssertEqual(live.highlight?.label, "半决赛")
    }

    // MARK: - 地区与分组

    func testBuildEventRegionTextLocalizesCountryAndDropsDuplicatedParts() {
        XCTAssertEqual(
            SaiXunRules.buildEventRegionText(EventNewsFixtures.eventNewsModel(country: "Northern Ireland", city: "Belfast")),
            "北爱尔兰 · Belfast"
        )
        XCTAssertEqual(
            SaiXunRules.buildEventRegionText(EventNewsFixtures.eventNewsModel(country: "中国", city: "中国")),
            "中国"
        )
        XCTAssertEqual(SaiXunRules.buildEventRegionText(EventNewsFixtures.eventNewsModel()), "")
    }

    func testBuildSaiXunListSectionsGroupsAdjacentItemsByStatusAndFinishedMonth() {
        let sections = SaiXunRules.buildSaiXunListSections([
            EventNewsFixtures.eventNewsModel(id: 1, startDate: "2026-10-12", status: 1),
            EventNewsFixtures.eventNewsModel(id: 2, startDate: "2026-10-18", status: 0),
            EventNewsFixtures.eventNewsModel(id: 3, startDate: "2026-11-02", status: 0),
            EventNewsFixtures.eventNewsModel(id: 4, startDate: "2026-09-28", status: 2),
            EventNewsFixtures.eventNewsModel(id: 5, startDate: "2026-09-07", status: 2),
            EventNewsFixtures.eventNewsModel(id: 6, startDate: "2026-08-31", status: 2),
            EventNewsFixtures.eventNewsModel(id: 7, startDate: "2026-08-01", status: 3),
        ])

        XCTAssertEqual(sections.map(\.title), ["进行中", "即将开始", "9月", "8月", "已取消"])
        XCTAssertEqual(sections.map(\.caption), ["", "", "已结束", "已结束", ""])
        XCTAssertEqual(sections.map { $0.items.map(\.id) }, [[1], [2, 3], [4, 5], [6], [7]])
        XCTAssertEqual(Set(sections.map(\.key)).count, sections.count)
    }

    func testBuildSaiXunHeroStatsCountsTotalLiveAndUpcomingItems() {
        let stats = SaiXunRules.buildSaiXunHeroStats([
            EventNewsFixtures.eventNewsModel(status: 1),
            EventNewsFixtures.eventNewsModel(status: 0),
            EventNewsFixtures.eventNewsModel(status: 1),
            EventNewsFixtures.eventNewsModel(status: 2),
        ])

        XCTAssertEqual(stats, SaiXunHeroStats(totalCount: 4, liveCount: 2, upcomingCount: 1))
    }

    // MARK: - 冠军与中文名

    func testResolveSaiXunChampionReadsFinishedFinalFromDetailRounds() {
        let rounds = SaiXunRules.buildSaiXunDetailRounds([
            EventNewsFixtures.matchModel(
                id: 1,
                sourceType: "official",
                roundName: "Final",
                roundOrder: 70,
                matchOrder: 1,
                status: 2,
                homePlayerName: "Yuan Sijun",
                awayPlayerName: "Wu Yize",
                awayPlayerFlagEmoji: "🇨🇳",
                awayPlayerCountryCode: "CN",
                awayPlayerAvatar: "https://cdn.example.com/wst/players/wu.png",
                homeScore: 5,
                awayScore: 10,
                winnerSide: 2
            ),
        ], now: time("2026-10-17T12:00:00+08:00"))

        XCTAssertEqual(
            SaiXunRules.resolveSaiXunChampion(rounds),
            SaiXunChampion(
                name: "Wu Yize",
                subName: "",
                avatar: .remote(URL(string: "https://cdn.example.com/wst/players/wu.png")!),
                flag: SaiXunFlag(type: .emoji, value: "🇨🇳"),
                opponentName: "Yuan Sijun",
                scoreText: "10-5"
            )
        )
    }

    func testPlayersWithChineseNameShowItFirstAndKeepEnglishNameAsSubLine() {
        let rounds = SaiXunRules.buildSaiXunDetailRounds([
            EventNewsFixtures.matchModel(
                id: 1,
                roundName: "Final",
                startTime: "2026-11-16T05:00:00+08:00",
                status: 2,
                homePlayerName: "Zhao Xintong",
                homePlayerNameZh: " 赵心童 ",
                awayPlayerName: "Mark Selby",
                awayPlayerNameZh: "",
                homeScore: 10,
                awayScore: 8,
                winnerSide: 1
            ),
            EventNewsFixtures.matchModel(
                id: 2,
                roundName: "Semi-Finals",
                startTime: "2026-11-15T05:00:00+08:00",
                homePlayerName: "Winner of Match 12",
                awayPlayerName: "Neil Robertson",
                awayPlayerNameZh: "Neil Robertson"
            ),
        ], now: time("2026-11-17T12:00:00+08:00"))

        let final = rounds.first { $0.roundName == "决赛" }?.matches[0]
        XCTAssertEqual(final?.homePlayerName, "赵心童")
        XCTAssertEqual(final?.homePlayerSubName, "Zhao Xintong")
        XCTAssertEqual(final?.awayPlayerName, "Mark Selby")
        XCTAssertEqual(final?.awayPlayerSubName, "")

        let semi = rounds.first { $0.roundName == "半决赛" }?.matches[0]
        XCTAssertEqual(semi?.homePlayerName, "第12场胜者")
        XCTAssertEqual(semi?.homePlayerSubName, "")
        XCTAssertEqual(semi?.awayPlayerSubName, "")

        XCTAssertEqual(
            SaiXunRules.resolveSaiXunChampion(rounds),
            SaiXunChampion(
                name: "赵心童",
                subName: "Zhao Xintong",
                avatar: final?.homePlayerAvatar ?? .defaultAvatar,
                flag: final?.homePlayerFlag ?? SaiXunFlag(type: .none, value: ""),
                opponentName: "Mark Selby",
                scoreText: "10-8"
            )
        )
    }

    // MARK: - 对照表完整性

    func testRoundNameAliasTableMatchesMiniProgram() {
        XCTAssertEqual(SaiXunTables.roundNameAliases.count, 15)
    }

    func testOrdinalWordNumberTableMatchesMiniProgram() {
        XCTAssertEqual(SaiXunTables.ordinalWordNumbers.count, 10)
    }

    func testCountryNameTableMatchesMiniProgram() {
        XCTAssertEqual(SaiXunTables.countryNameMap.count, 36)
    }

    func testTournamentSeriesTranslationTableMatchesMiniProgram() {
        XCTAssertEqual(SaiXunTables.tournamentSeriesTranslations.count, 27)
    }

    func testTournamentPhaseSuffixTableMatchesMiniProgram() {
        XCTAssertEqual(SaiXunTables.tournamentPhaseSuffixes.count, 2)
    }

    func testStatusToneTableMatchesMiniProgram() {
        XCTAssertEqual(SaiXunTables.statusTones.count, 4)
    }

    func testWeekdayLabelTableMatchesMiniProgram() {
        XCTAssertEqual(SaiXunTables.weekdayLabels.count, 7)
    }

    func testSubdivisionFlagImageTableMatchesMiniProgram() {
        XCTAssertEqual(SaiXunTables.subdivisionFlagImages.count, 3)
    }

    func testSubdivisionLabelTableMatchesMiniProgram() {
        XCTAssertEqual(SaiXunTables.subdivisionLabels.count, 3)
    }
}
