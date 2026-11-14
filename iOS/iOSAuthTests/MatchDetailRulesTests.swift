import Foundation
import XCTest
@testable import iOS

/// 逐条对应小程序 `uniapp/tests/match-detail.test.mjs` 的 19 条行为断言。
final class MatchDetailRulesTests: XCTestCase {
    private let chasingBreakdown: [String: Any] = [
        "available": true,
        "player1": [
            "score": 16, "normal_wins": 2, "small_golds": 1, "big_golds": 0,
            "foul_count": 3, "foul_points_given": 3, "other_points": 0,
        ],
        "player2": [
            "score": 13, "normal_wins": 0, "small_golds": 0, "big_golds": 1,
            "foul_count": 1, "foul_points_given": 1, "other_points": 0,
        ],
        "events": [
            [
                "seq": 1, "kind": "big_gold", "scorer": 2, "points": 10,
                "player1_score": 5, "player2_score": 10, "created_at": "2026-11-10 12:00:03",
            ],
            [
                "seq": 2, "kind": "foul", "scorer": 1, "points": 1,
                "player1_score": 11, "player2_score": 10, "created_at": "2026-11-10 12:00:04",
            ],
        ],
        "events_truncated": false,
    ]

    private var chasingMatch: [String: Any] {
        [
            "player1_id": 1,
            "player2_id": 2,
            "player1_name": "创建者",
            "player2_name": "当前用户",
            "player1_score": 16,
            "player2_score": 13,
            "game_type": 2,
            "status": 2,
            "score_breakdown": chasingBreakdown,
        ]
    }

    func testHistoryMatchDetailUsesPublicDetailDataWithoutShowingSpectateBadge() {
        XCTAssertTrue(MatchDetailRules.shouldUsePublicMatchDetail(source: "history"))
        XCTAssertFalse(MatchDetailRules.shouldShowSpectateBadge(mode: ""))
    }

    func testSpectateMatchDetailUsesPublicDetailDataAndShowsSpectateBadge() {
        XCTAssertTrue(MatchDetailRules.shouldUsePublicMatchDetail(mode: "spectate"))
        XCTAssertTrue(MatchDetailRules.shouldShowSpectateBadge(mode: "spectate"))
    }

    func testNormalizeMapsPrivateDetailFieldsIntoDisplayFields() throws {
        let data = MatchDetailRules.normalize(try snapshot([
            "my_score": 7,
            "opponent_score": 5,
            "my_name": "张三",
            "opponent_name": "李四",
            "my_avatar": "a.png",
            "opponent_avatar": "b.png",
            "game_type": 2,
            "status": 2,
        ]))
        let match = data.matchData

        XCTAssertEqual(match.player1Score, 7)
        XCTAssertEqual(match.player2Score, 5)
        XCTAssertEqual(match.gameType, 2)
        XCTAssertEqual(match.status, 2)
        XCTAssertEqual(match.durationSeconds, 0)
        XCTAssertEqual(match.currentRound, 1)
        XCTAssertEqual(match.totalRounds, 0)
        XCTAssertEqual(match.viewerRole, "")
        XCTAssertFalse(match.refereeBound)
        XCTAssertEqual(match.refereeUserId, 0)
        XCTAssertEqual(match.refereeName, "")
        XCTAssertEqual(match.refereeAvatar, "")
        XCTAssertEqual(match.refereeJoinedAt, "")
        XCTAssertEqual(match.refereeDurationSeconds, 0)
        XCTAssertEqual(match.completedByUserId, 0)
        XCTAssertEqual(match.completionSource, "unknown")
        XCTAssertEqual(data.player1Info, MatchDetailSide(name: "张三", avatar: "a.png", vipLevel: 0, rankLevel: 0))
        XCTAssertEqual(data.player2Info, MatchDetailSide(name: "李四", avatar: "b.png", vipLevel: 0, rankLevel: 0))
    }

    func testNormalizeKeepsEachSideMemberIdentityNextToItsOwnAvatar() throws {
        let data = MatchDetailRules.normalize(try snapshot([
            "my_name": "张三",
            "opponent_name": "李四",
            "my_avatar": "a.png",
            "opponent_avatar": "b.png",
            "my_vip_level": 0,
            "my_rank_level": 0,
            "opponent_vip_level": 5,
            "opponent_rank_level": 4,
        ]))

        // 免费用户看到对手会员身份，自己一侧没有。
        XCTAssertEqual([data.player1Info.vipLevel, data.player1Info.rankLevel], [0, 0])
        XCTAssertEqual([data.player2Info.vipLevel, data.player2Info.rankLevel], [5, 4])
    }

    func testNormalizeSwapsMemberIdentityWithTheAvatarForTheOtherPerspective() throws {
        let data = MatchDetailRules.normalize(try snapshot([
            "player1_id": 1,
            "player2_id": 2,
            "player1_name": "创建者",
            "player2_name": "当前用户",
            "player1_avatar": "p1.png",
            "player2_avatar": "p2.png",
            "player1_vip_level": 3,
            "player1_rank_level": 2,
            "player2_vip_level": 0,
            "player2_rank_level": 0,
        ]), perspectiveUserId: 2)

        XCTAssertEqual(data.player1Info.avatar, "p2.png")
        XCTAssertEqual(data.player1Info.vipLevel, 0)
        XCTAssertEqual(data.player2Info.avatar, "p1.png")
        XCTAssertEqual([data.player2Info.vipLevel, data.player2Info.rankLevel], [3, 2])
    }

    func testNormalizeTreatsMissingMemberFieldsAsNonMembers() throws {
        let data = MatchDetailRules.normalize(try snapshot([
            "player1_id": 1, "player2_id": 2, "player1_name": "A", "player2_name": "B",
        ]))

        XCTAssertEqual([data.player1Info.vipLevel, data.player2Info.vipLevel], [0, 0])
    }

    func testNormalizeKeepsRefereeIdentityAndCompletionAttribution() throws {
        let match = MatchDetailRules.normalize(try snapshot([
            "player1_score": 5,
            "player2_score": 3,
            "game_type": 3,
            "status": 2,
            "viewer_role": "referee",
            "referee_bound": true,
            "referee_user_id": 3,
            "referee_name": "裁判丙",
            "referee_avatar": "referee.png",
            "referee_joined_at": "2026-07-22T14:00:00+08:00",
            "referee_duration_seconds": 600,
            "completed_by_user_id": 3,
            "completion_source": "referee",
        ])).matchData

        XCTAssertEqual(match.viewerRole, "referee")
        XCTAssertEqual(match.refereeName, "裁判丙")
        XCTAssertEqual(match.refereeAvatar, "referee.png")
        XCTAssertEqual(match.completionSource, "referee")
    }

    func testNormalizeConvertsPublicRoundsToTheRequestedUserPerspective() throws {
        let data = MatchDetailRules.normalize(try snapshot([
            "player1_id": 1,
            "player1_name": "创建者",
            "player1_score": 66,
            "player2_id": 2,
            "player2_name": "当前用户",
            "player2_score": 52,
            "rounds": [
                ["round_number": 1, "player1_score": 21, "player2_score": 0, "winner": 1],
                ["round_number": 2, "player1_score": 21, "player2_score": 10, "winner": 2],
            ],
        ]), perspectiveUserId: 2)

        XCTAssertEqual(data.player1Info.name, "当前用户")
        XCTAssertEqual(data.player2Info.name, "创建者")
        XCTAssertEqual(data.matchData.player1Score, 52)
        XCTAssertEqual(data.matchData.player2Score, 66)
        XCTAssertEqual(data.roundRecords.map { "\($0.player1Score)-\($0.player2Score)" }, ["0-21", "10-21"])
        XCTAssertEqual(data.roundRecords.map(\.result), ["loss", "win"])
        XCTAssertEqual(data.roundRecords.map(\.resultText), ["负", "胜"])
    }

    func testNormalizeSwapsParticipantRoundRecordsOnceForPlayerTwo() throws {
        let rounds: [[String: Any]] = [
            ["round_number": 1, "player1_score": 21, "player2_score": 10, "winner": 1],
            ["round_number": 2, "player1_score": 33, "player2_score": 21, "winner": 2],
            ["round_number": 3, "player1_score": 8, "player2_score": 60, "winner": 2],
        ]
        let asPlayer2 = MatchDetailRules.normalize(try snapshot([
            "is_player1": false,
            "viewer_role": "player2",
            "rounds": rounds,
        ]))
        XCTAssertEqual(asPlayer2.roundRecords.map { "\($0.player1Score)-\($0.player2Score)" }, ["10-21", "21-33", "60-8"])
        XCTAssertEqual(asPlayer2.roundRecords.map(\.winner), [2, 1, 1])
        XCTAssertEqual(asPlayer2.roundRecords.map(\.result), ["loss", "win", "win"])
        XCTAssertEqual(asPlayer2.roundRecords.map(\.resultText), ["负", "胜", "胜"])
        XCTAssertTrue(asPlayer2.roundRecordsSwapped)

        // 参与者详情同时带 perspective_user_id 时也只换一次边。
        let withPerspective = MatchDetailRules.normalize(try snapshot([
            "is_player1": false,
            "viewer_role": "player2",
            "player1_id": 1,
            "player2_id": 2,
            "rounds": rounds,
        ]), perspectiveUserId: 2)
        XCTAssertEqual(
            withPerspective.roundRecords.map { "\($0.player1Score)-\($0.player2Score)" },
            ["10-21", "21-33", "60-8"]
        )
    }

    func testNormalizeKeepsRoundRecordsForPlayerOneAndReferees() throws {
        let rounds: [[String: Any]] = [
            ["round_number": 1, "player1_score": 21, "player2_score": 10, "winner": 1],
        ]
        let asPlayer1 = MatchDetailRules.normalize(try snapshot([
            "is_player1": true, "viewer_role": "player1", "rounds": rounds,
        ]))
        XCTAssertEqual(asPlayer1.roundRecords.map { [$0.player1Score, $0.player2Score] }, [[21, 10]])
        XCTAssertEqual(asPlayer1.roundRecords.map(\.resultText), ["胜"])
        XCTAssertFalse(asPlayer1.roundRecordsSwapped)

        let asReferee = MatchDetailRules.normalize(try snapshot([
            "is_player1": false, "viewer_role": "referee", "rounds": rounds,
        ]))
        XCTAssertEqual(asReferee.roundRecords.map { [$0.player1Score, $0.player2Score] }, [[21, 10]])
        XCTAssertEqual(asReferee.roundRecords.map(\.resultText), ["胜"])
        XCTAssertFalse(asReferee.roundRecordsSwapped)
    }

    func testNormalizeRoundRecordsLabelsOnlyTheFourNineBallWinTypes() {
        let records = MatchDetailRules.normalizeRoundRecords([
            MatchRoundRecord(roundNumber: 1, winner: 1, winType: "break_clear"),
            MatchRoundRecord(roundNumber: 2, winner: 1, winType: "continue_clear"),
            MatchRoundRecord(roundNumber: 3, winner: 2, winType: "small_gold"),
            MatchRoundRecord(roundNumber: 4, winner: 2, winType: "big_gold"),
            MatchRoundRecord(roundNumber: 5, winner: 1, winType: "normal"),
            MatchRoundRecord(roundNumber: 6, winner: 1, winType: nil),
            MatchRoundRecord(roundNumber: 7, winner: 1, winType: ""),
            MatchRoundRecord(roundNumber: 8, winner: 1, winType: "snooker_century"),
        ])

        XCTAssertEqual(records.map(\.winTypeLabel), ["炸清", "接清", "小金", "大金", "", "", "", ""])
        XCTAssertEqual(
            records.map { $0.winType.isEmpty ? "" : $0.winType },
            ["break_clear", "continue_clear", "small_gold", "big_gold", "normal", "", "", "snooker_century"]
        )
    }

    func testNormalizeKeepsWinTypeAttachedToTheSwappedRound() throws {
        let data = MatchDetailRules.normalize(try snapshot([
            "is_player1": false,
            "viewer_role": "player2",
            "rounds": [
                ["round_number": 1, "player1_score": 21, "player2_score": 10, "winner": 1, "win_type": "break_clear"],
                ["round_number": 2, "player1_score": 10, "player2_score": 21, "winner": 2, "win_type": "big_gold"],
            ],
        ]))

        XCTAssertEqual(data.roundRecords.map(\.winner), [2, 1])
        XCTAssertEqual(data.roundRecords.map(\.winType), ["break_clear", "big_gold"])
        XCTAssertEqual(data.roundRecords.map(\.winTypeLabel), ["炸清", "大金"])
    }

    func testNormalizeReportsNeutralRoundRecordsForRefereesSpectatorsAndPublicDetail() throws {
        let asPlayer1 = MatchDetailRules.normalize(try snapshot(["is_player1": true, "viewer_role": "player1"]))
        XCTAssertFalse(asPlayer1.roundRecordsNeutral)

        let asReferee = MatchDetailRules.normalize(try snapshot(["is_player1": false, "viewer_role": "referee"]))
        XCTAssertTrue(asReferee.roundRecordsNeutral)

        let publicDetail = MatchDetailRules.normalize(try snapshot(["player1_id": 1, "player2_id": 2]))
        XCTAssertTrue(publicDetail.roundRecordsNeutral)
    }

    func testNormalizeMapsChasingBreakdownForPlayerTwoWithSwappedSides() throws {
        let view = try XCTUnwrap(MatchDetailRules.normalize(try snapshot(chasingMatch), perspectiveUserId: 2).scoreBreakdown)

        XCTAssertEqual([view.leftLabel, view.rightLabel], ["我", "对手"])
        XCTAssertEqual([view.leftScore, view.rightScore], [13, 16])
        XCTAssertEqual(view.rows.prefix(4).map { [$0.label, $0.left, $0.right] }, [
            ["总分", "13", "16"],
            ["普胜", "0", "2"],
            ["小金", "0", "1"],
            ["大金", "1", "0"],
        ])
        XCTAssertEqual([view.events[0].side, view.events[0].desc], ["right", "我犯规"])
        XCTAssertEqual(view.events[0].scoreText, "10 : 11")
        XCTAssertEqual([view.events[1].side, view.events[1].scoreText], ["left", "10 : 5"])
    }

    func testNormalizeKeepsPlayerOneAndTwoLabelsForSpectators() throws {
        let view = try XCTUnwrap(MatchDetailRules.normalize(try snapshot(chasingMatch)).scoreBreakdown)

        XCTAssertEqual([view.leftLabel, view.rightLabel], ["选手1", "选手2"])
        XCTAssertEqual([view.leftScore, view.rightScore], [16, 13])
        XCTAssertEqual([view.events[0].side, view.events[0].sideLabel], ["left", "选手1"])
        XCTAssertEqual(view.events[0].desc, "选手2犯规")
        XCTAssertEqual([view.events[1].side, view.events[1].sideLabel], ["right", "选手2"])
    }

    func testNormalizeKeepsTheRefereePerspectiveNeutralEvenForAPlayer() throws {
        var json = chasingMatch
        json["viewer_role"] = "referee"
        let view = try XCTUnwrap(MatchDetailRules.normalize(try snapshot(json), perspectiveUserId: 2).scoreBreakdown)

        XCTAssertEqual([view.leftLabel, view.rightLabel], ["选手1", "选手2"])
        XCTAssertEqual([view.leftScore, view.rightScore], [16, 13])
    }

    func testNormalizeHasNoScoreBreakdownForOtherGameTypesOrMissingData() throws {
        var otherGameType = chasingMatch
        otherGameType["game_type"] = 3
        XCTAssertNil(MatchDetailRules.normalize(try snapshot(otherGameType)).scoreBreakdown)

        var missingBreakdown = chasingMatch
        missingBreakdown.removeValue(forKey: "score_breakdown")
        XCTAssertNil(MatchDetailRules.normalize(try snapshot(missingBreakdown)).scoreBreakdown)

        XCTAssertNil(MatchDetailRules.normalize(try snapshot([
            "game_type": 3, "player1_score": 5, "player2_score": 3,
        ])).scoreBreakdown)
    }

    func testNormalizeFollowsTheViewerOnParticipantDetailWithoutPerspectiveUser() throws {
        let asPlayer2 = MatchDetailRules.normalize(try snapshot([
            "player1_id": 1, "player2_id": 2,
            "my_name": "我", "opponent_name": "对手",
            "game_type": 2, "status": 2, "score_breakdown": chasingBreakdown,
            "is_player1": false, "viewer_role": "player2", "my_score": 13, "opponent_score": 16,
        ]))
        XCTAssertEqual([asPlayer2.matchData.player1Score, asPlayer2.matchData.player2Score], [13, 16])
        XCTAssertEqual([asPlayer2.scoreBreakdown?.leftLabel, asPlayer2.scoreBreakdown?.rightLabel], ["我", "对手"])
        XCTAssertEqual([asPlayer2.scoreBreakdown?.leftScore, asPlayer2.scoreBreakdown?.rightScore], [13, 16])

        let asPlayer1 = MatchDetailRules.normalize(try snapshot([
            "player1_id": 1, "player2_id": 2,
            "my_name": "我", "opponent_name": "对手",
            "game_type": 2, "status": 2, "score_breakdown": chasingBreakdown,
            "is_player1": true, "viewer_role": "player1", "my_score": 16, "opponent_score": 13,
        ]))
        let scoreView = try XCTUnwrap(asPlayer1.scoreBreakdown)
        XCTAssertEqual(scoreView.leftLabel, "我")
        XCTAssertEqual(scoreView.leftScore, 16)
        XCTAssertEqual(scoreView.rightScore, 13)
    }

    func testNormalizeKeepsTheRefereeNeutralOnParticipantDetail() throws {
        let view = try XCTUnwrap(MatchDetailRules.normalize(try snapshot([
            "player1_id": 1, "player2_id": 2,
            "my_name": "我", "opponent_name": "对手",
            "game_type": 2, "status": 2, "score_breakdown": chasingBreakdown,
            "is_player1": false, "viewer_role": "referee", "my_score": 16, "opponent_score": 13,
        ])).scoreBreakdown)

        XCTAssertEqual([view.leftLabel, view.rightLabel], ["选手1", "选手2"])
        XCTAssertEqual([view.leftScore, view.rightScore], [16, 13])
    }
}

private func snapshot(_ json: [String: Any]) throws -> LiveMatchSnapshot {
    let data = try JSONSerialization.data(withJSONObject: json)
    let decoder = JSONDecoder()
    decoder.keyDecodingStrategy = .convertFromSnakeCase
    return try decoder.decode(LiveMatchSnapshot.self, from: data)
}
