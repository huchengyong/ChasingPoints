import Foundation

struct SnookerBall: Equatable, Identifiable {
    let value: Int
    /// red / yellow / ... / black。
    let key: String
    let label: String
    let color: String

    var id: String { key }
}

/// 服务端权威的斯诺克 V2 局面；字段与 `normalizeSnookerV2State` 的输出一致。
struct SnookerV2State: Equatable {
    var rulesVersion: Int = 0
    var bestOfFrames: Int = 0
    var startingActor: Int = 0
    var phase: String = ""
    var ballOn: String = ""
    var striker: Int = 0
    var visitNo: Int = 0
    var currentBreak: Int = 0
    var redsRemaining: Int = 0
    var freeBallAvailable: Bool = false
    var cueBallInHand: Bool = false
    var missWarningActive: Bool = false
    var respottedBlackPending: Bool = false
    var pendingConcessionActor: Int = 0
    var pendingConcessionScope: String = ""
    var frameEndReason: String = ""
    var currentFrameStarted: Bool = true
    var status: Int = 1
}

struct SnookerV2PotPayload: Equatable, Encodable {
    let actor: Int
    let outcome = "pot"
    let ballOnValue: Int
    let pottedReds: Int
    let ballOnPotted: Bool
    let freeBallValue: Int
    let freeBallPotted: Bool

    enum CodingKeys: String, CodingKey {
        case actor
        case outcome
        case ballOnValue = "ball_on_value"
        case pottedReds = "potted_reds"
        case ballOnPotted = "ball_on_potted"
        case freeBallValue = "free_ball_value"
        case freeBallPotted = "free_ball_potted"
    }
}

struct SnookerV2NoScorePayload: Equatable, Encodable {
    let actor: Int
    let outcome = "no_score"
    let ballOnValue: Int
    /// 只有大于 0 时才下发。
    let freeBallValue: Int?

    enum CodingKeys: String, CodingKey {
        case actor
        case outcome
        case ballOnValue = "ball_on_value"
        case freeBallValue = "free_ball_value"
    }
}

struct SnookerV2FoulPayload: Equatable, Encodable {
    let actor: Int
    let outcome = "foul"
    let ballOnValue: Int
    let penalty: Int
    let redsRemoved: Int
    let foulResolution: String
    let freeBallAwarded: Bool
    let foulAndMiss: Bool
    let missSequenceEligible: Bool
    let cueBallInHand: Bool

    enum CodingKeys: String, CodingKey {
        case actor
        case outcome
        case ballOnValue = "ball_on_value"
        case penalty
        case redsRemoved = "reds_removed"
        case foulResolution = "foul_resolution"
        case freeBallAwarded = "free_ball_awarded"
        case foulAndMiss = "foul_and_miss"
        case missSequenceEligible = "miss_sequence_eligible"
        case cueBallInHand = "cue_ball_in_hand"
    }
}

/// 逐条对应小程序 `utils/snooker-v2.js`。
enum SnookerV2 {
    static let balls: [SnookerBall] = [
        SnookerBall(value: 1, key: "red", label: "红球", color: "red"),
        SnookerBall(value: 2, key: "yellow", label: "黄球", color: "yellow"),
        SnookerBall(value: 3, key: "green", label: "绿球", color: "green"),
        SnookerBall(value: 4, key: "brown", label: "咖啡球", color: "brown"),
        SnookerBall(value: 5, key: "blue", label: "蓝球", color: "blue"),
        SnookerBall(value: 6, key: "pink", label: "粉球", color: "pink"),
        SnookerBall(value: 7, key: "black", label: "黑球", color: "black"),
    ]

    private static let ballValueByKey: [String: Int] = Dictionary(
        uniqueKeysWithValues: balls.map { ($0.key, $0.value) }
    )

    static func isRulesV2(_ payload: SnookerV2SnapshotInput) -> Bool {
        (payload.snookerRulesVersion ?? 0) == 2
    }

    static func normalizeState(_ payload: SnookerV2SnapshotInput = SnookerV2SnapshotInput()) -> SnookerV2State {
        SnookerV2State(
            rulesVersion: payload.snookerRulesVersion ?? 0,
            bestOfFrames: payload.bestOfFrames ?? 0,
            startingActor: payload.startingActor ?? 0,
            phase: payload.snookerPhase ?? "",
            ballOn: payload.snookerBallOn ?? "",
            striker: payload.snookerStriker ?? 0,
            visitNo: payload.snookerVisitNo ?? 0,
            currentBreak: payload.snookerCurrentBreak ?? 0,
            redsRemaining: max(0, payload.snookerRedsRemaining ?? 0),
            freeBallAvailable: payload.snookerFreeBallAvailable == true,
            cueBallInHand: payload.snookerCueBallInHand == true,
            missWarningActive: payload.snookerMissWarningActive == true,
            respottedBlackPending: payload.snookerRespottedBlackPending == true,
            pendingConcessionActor: payload.snookerPendingConcessionActor ?? 0,
            pendingConcessionScope: payload.snookerPendingConcessionScope ?? "",
            frameEndReason: payload.snookerFrameEndReason ?? "",
            // 服务端只在明确下发 false 时才表示当前局已结束。
            currentFrameStarted: payload.currentFrameStarted != false,
            status: payload.status ?? 1
        )
    }

    static func ballOnValue(_ state: SnookerV2State, selectedColor: Int = 0) -> Int {
        if state.ballOn == "red" { return 1 }
        if state.ballOn == "color_choice" {
            return (2...7).contains(selectedColor) ? selectedColor : 0
        }
        return ballValueByKey[state.ballOn] ?? 0
    }

    static func potOptions(_ state: SnookerV2State) -> [SnookerBall] {
        if !state.currentFrameStarted || state.status == 2 || ["ended", "respotted_black_pending"].contains(state.phase) {
            return []
        }
        if state.ballOn == "red" {
            return state.redsRemaining > 0 ? [balls[0]] : []
        }
        if state.ballOn == "color_choice" {
            return Array(balls.dropFirst())
        }
        let value = ballOnValue(state)
        return value != 0 ? balls.filter { $0.value == value } : []
    }

    /// 自由球只能把任意彩球当作目标球，不能选真正的目标球。
    static func freeBallOptions(_ state: SnookerV2State, ballOnValue: Int = 0) -> [SnookerBall] {
        if !state.freeBallAvailable { return [] }
        let target = SnookerV2.ballOnValue(state, selectedColor: ballOnValue)
        if target == 0 { return [] }
        return balls.dropFirst().filter { $0.value != target }
    }

    static func phaseLabel(_ state: SnookerV2State) -> String {
        switch state.phase {
        case "reds": "红球阶段"
        case "color_after_red": "红球后的彩球"
        case "colors": "顺序清彩"
        case "respotted_black_pending": "等待重置黑球"
        case "respotted_black": "重置黑球决胜"
        case "ended": "本局已结束"
        default: "等待同步局面"
        }
    }

    static func ballOnLabel(_ state: SnookerV2State) -> String {
        if state.ballOn == "color_choice" { return "自选彩球" }
        return balls.first { $0.key == state.ballOn }?.label ?? "等待裁判确认"
    }

    static func potPayload(
        state: SnookerV2State = SnookerV2State(),
        actor: Int = 0,
        ballOnValue: Int = 0,
        pottedReds: Int = 0,
        freeBallValue: Int = 0,
        freeBallPotted: Bool = false,
        ballOnPotted: Bool = false
    ) -> SnookerV2PotPayload {
        SnookerV2PotPayload(
            actor: actor,
            ballOnValue: SnookerV2.ballOnValue(state, selectedColor: ballOnValue),
            pottedReds: max(0, pottedReds),
            ballOnPotted: ballOnPotted,
            freeBallValue: freeBallValue,
            freeBallPotted: freeBallPotted
        )
    }

    static func noScorePayload(
        state: SnookerV2State = SnookerV2State(),
        actor: Int = 0,
        ballOnValue: Int = 0,
        freeBallValue: Int = 0
    ) -> SnookerV2NoScorePayload {
        let resolvedBallOn = state.ballOn == "color_choice"
            ? ballOnValue
            : SnookerV2.ballOnValue(state)
        return SnookerV2NoScorePayload(
            actor: actor,
            ballOnValue: resolvedBallOn,
            freeBallValue: freeBallValue > 0 ? freeBallValue : nil
        )
    }

    static func foulPayload(
        state: SnookerV2State = SnookerV2State(),
        actor: Int = 0,
        penalty: Int = 4,
        ballOnValue: Int = 0,
        redsRemoved: Int = 0,
        resolution: String = "incoming_plays",
        freeBallAwarded: Bool = false,
        foulAndMiss: Bool = false,
        missSequenceEligible: Bool = false,
        cueBallInHand: Bool = false
    ) -> SnookerV2FoulPayload {
        SnookerV2FoulPayload(
            actor: actor,
            ballOnValue: SnookerV2.ballOnValue(state, selectedColor: ballOnValue),
            penalty: penalty,
            redsRemoved: max(0, redsRemoved),
            foulResolution: resolution,
            freeBallAwarded: resolution == "incoming_plays" && freeBallAwarded,
            foulAndMiss: foulAndMiss,
            missSequenceEligible: resolution == "offender_replays_original" && foulAndMiss && missSequenceEligible,
            cueBallInHand: resolution != "offender_replays_original" && cueBallInHand
        )
    }

    static func minimumPenalty(_ state: SnookerV2State, ballOnValue: Int = 0) -> Int {
        max(4, SnookerV2.ballOnValue(state, selectedColor: ballOnValue))
    }

    /// 数量选择器分页：跨端动作面板一页最多 6 项。
    static func countPages(minimum: Int?, maximum: Int?, pageSize: Int = 6) -> [[Int]] {
        let minValue = max(0, minimum ?? 0)
        let maxValue = max(minValue, maximum ?? 0)
        let size = min(6, max(1, pageSize))
        let values = Array(minValue...maxValue)
        return stride(from: 0, to: values.count, by: size).map {
            Array(values[$0..<min($0 + size, values.count)])
        }
    }
}
