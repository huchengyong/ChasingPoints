import Foundation

/// 记分动作选项：type 为 normal / break_clear / continue_clear / small_gold / big_gold。
struct ScoringActionOption: Equatable {
    let type: String
    let score: Int
    let title: String
    let desc: String
}

/// 逐条对应小程序 `utils/match-scoring-target.js`：界面上的 1 / 2 号位与接口的选手 1 / 2 之间的换算和文案。
enum MatchScoringTarget {
    /// UI 的 1 / 2 号位只有 1 和 2，其他值一律当 2。
    private static func normalizedActor(_ actor: Int) -> Int {
        actor == 1 ? 1 : 2
    }

    /// 选手视角的 2 号位是接口的选手 1；裁判视角不换边。
    static func resolveMatchApiActor(viewerRole: String = "player1", isPlayer1: Bool = true, uiActor: Int = 2) -> Int {
        let actor = normalizedActor(uiActor)
        if viewerRole == "referee" || isPlayer1 { return actor }
        return actor == 1 ? 2 : 1
    }

    static func resolveScoringParticipantLabel(viewerRole: String = "player1", uiActor: Int = 2) -> String {
        let actor = normalizedActor(uiActor)
        if viewerRole == "referee" {
            return actor == 1 ? "选手1" : "选手2"
        }
        return actor == 1 ? "我方" : "对手"
    }

    static func buildRoundWinActionOptions(
        gameType: Int,
        viewerRole: String = "player1",
        uiActor: Int = 2
    ) -> [ScoringActionOption] {
        let label = resolveScoringParticipantLabel(viewerRole: viewerRole, uiActor: uiActor)
        if gameType == 4 {
            return [
                ScoringActionOption(type: "normal", score: 1, title: "\(label)普胜", desc: "本局计给\(label)"),
                ScoringActionOption(type: "small_gold", score: 1, title: "\(label)小金", desc: "\(label)金球直接制胜"),
                ScoringActionOption(type: "big_gold", score: 1, title: "\(label)大金", desc: "\(label)开球直接制胜"),
            ]
        }
        return [
            ScoringActionOption(type: "normal", score: 1, title: "\(label)普胜", desc: "本局计给\(label)"),
            ScoringActionOption(type: "break_clear", score: 1, title: "\(label)炸清", desc: "\(label)直接清台获胜"),
            ScoringActionOption(type: "continue_clear", score: 1, title: "\(label)接清", desc: "\(label)连续清台获胜"),
        ]
    }

    static func buildJiuqiuWinOptions(viewerRole: String = "player1", uiActor: Int = 2) -> [ScoringActionOption] {
        let label = resolveScoringParticipantLabel(viewerRole: viewerRole, uiActor: uiActor)
        return [
            ScoringActionOption(type: "normal", score: 4, title: "\(label)普胜", desc: "\(label) +4 分"),
            ScoringActionOption(type: "small_gold", score: 7, title: "\(label)小金", desc: "\(label) +7 分"),
            ScoringActionOption(type: "big_gold", score: 10, title: "\(label)大金", desc: "\(label) +10 分"),
        ]
    }

    static func resolveScoreFeedback(viewerRole: String = "player1", uiActor: Int = 2, score: Int = 0) -> String {
        "已给\(resolveScoringParticipantLabel(viewerRole: viewerRole, uiActor: uiActor)) +\(score) 分"
    }

    static func resolveWinFeedback(viewerRole: String = "player1", uiActor: Int = 2) -> String {
        "已判给\(resolveScoringParticipantLabel(viewerRole: viewerRole, uiActor: uiActor))"
    }

    static func resolveFoulFeedback(viewerRole: String = "player1", foulingActor: Int = 2, score: Int = 1) -> String {
        let foulActor = normalizedActor(foulingActor)
        let scoringActor = foulActor == 1 ? 2 : 1
        let foulLabel = resolveScoringParticipantLabel(viewerRole: viewerRole, uiActor: foulActor)
        let scoringLabel = resolveScoringParticipantLabel(viewerRole: viewerRole, uiActor: scoringActor)
        return "\(foulLabel)犯规，\(scoringLabel) +\(score) 分"
    }
}
