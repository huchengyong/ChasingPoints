import Foundation

/// 扫码 / 选对手时的对手摘要；`id` 系列与名称系列各自按小程序顺序兜底。
struct StartMatchOpponent: Equatable {
    var id: Int64? = nil
    var userId: Int64? = nil
    var nickname: String? = nil
    var name: String? = nil
    var avatar: String? = nil
    var opponentId: Int64? = nil
    var opponentName: String? = nil
    var opponentAvatar: String? = nil
}

/// `POST /api/match/start` 的请求体；nil 字段不编码。
struct StartMatchRequest: Equatable, Encodable, CustomStringConvertible {
    var gameType: Int? = nil
    var inviteToken: String? = nil
    var opponentId: Int64? = nil
    var opponentName: String? = nil
    var opponentAvatar: String? = nil
    var matchMode: String? = nil
    var visibility: String? = nil
    var snookerRulesVersion: Int? = nil
    var snookerFormat: String? = nil
    var snookerTargetWins: Int? = nil
    var bestOfFrames: Int? = nil
    var startingActor: Int? = nil
    var matchFormat: String? = nil
    var targetWins: Int? = nil

    enum CodingKeys: String, CodingKey {
        case gameType = "game_type"
        case inviteToken = "invite_token"
        case opponentId = "opponent_id"
        case opponentName = "opponent_name"
        case opponentAvatar = "opponent_avatar"
        case matchMode = "match_mode"
        case visibility
        case snookerRulesVersion = "snooker_rules_version"
        case snookerFormat = "snooker_format"
        case snookerTargetWins = "snooker_target_wins"
        case bestOfFrames = "best_of_frames"
        case startingActor = "starting_actor"
        case matchFormat = "match_format"
        case targetWins = "target_wins"
    }

    /// 邀请 token 是凭证，任何打印都要遮住。
    var description: String {
        let token = inviteToken.map { $0.isEmpty ? "" : "<redacted>" } ?? ""
        return "StartMatchRequest(gameType: \(gameType.map(String.init) ?? "nil"), inviteToken: \(token))"
    }
}

struct StartMatchModeOption: Equatable {
    let label: String
    let matchMode: String
    let visibility: String
}

struct NormalizedStartMatchOptions: Equatable {
    let matchMode: String
    let visibility: String
}

/// 逐条对应小程序 `utils/start-match.js`。
enum StartMatch {
    /// 扫码确认页的对局类型选项：练习赛可私密，排位赛必须公开。
    static let modeOptions: [StartMatchModeOption] = [
        StartMatchModeOption(label: "练习赛（默认私密，不影响竞技权益）", matchMode: "practice", visibility: "private"),
        StartMatchModeOption(label: "练习赛（公开展示，不影响竞技权益）", matchMode: "practice", visibility: "public"),
        StartMatchModeOption(label: "排位赛（公开展示，需结束确认）", matchMode: "ranked", visibility: "public"),
    ]

    static func validate(_ payload: StartMatchRequest = StartMatchRequest()) -> String {
        let gameType = payload.gameType ?? 0
        let inviteToken = (payload.inviteToken ?? "").trimmingCharacters(in: .whitespacesAndNewlines)
        let opponentId = payload.opponentId ?? 0
        if inviteToken.isEmpty && opponentId <= 0 {
            return "请选择有效的平台对手"
        }
        let mode = payload.matchMode ?? ""
        let visibility = payload.visibility ?? ""
        if !mode.isEmpty && !["practice", "ranked"].contains(mode) { return "请选择有效的对局模式" }
        if !visibility.isEmpty && !["private", "public"].contains(visibility) { return "请选择有效的公开范围" }
        if mode == "ranked" && visibility == "private" { return "排位赛必须公开展示" }
        if gameType == 1 {
            // JS 的 `snooker_rules_version || 2` 把 0 也当成未设置。
            let rawRulesVersion = payload.snookerRulesVersion ?? 0
            let rulesVersion = rawRulesVersion == 0 ? 2 : rawRulesVersion
            if ![1, 2].contains(rulesVersion) { return "不支持的斯诺克规则版本" }
            if rulesVersion == 2 {
                let format = payload.snookerFormat ?? ""
                let targetWins = payload.snookerTargetWins ?? 0
                if format == "free" {
                    if targetWins != 0 { return "自由局数不能设置目标胜局" }
                } else if format == "race_to" {
                    if targetWins < 1 || targetWins > 25 { return "抢N局目标必须在1至25之间" }
                } else {
                    let bestOfFrames = payload.bestOfFrames ?? 0
                    let startingActor = payload.startingActor ?? 0
                    if bestOfFrames <= 0 || bestOfFrames % 2 == 0 { return "斯诺克总局数必须为正奇数" }
                    if ![1, 2].contains(startingActor) { return "请选择首局开球方" }
                }
            }
        }
        if [3, 4].contains(gameType) {
            let format = payload.matchFormat ?? ""
            let targetWins = payload.targetWins ?? 0
            if format.isEmpty {
                if targetWins != 0 { return "请选择有效的赛制" }
            } else if format == "free" {
                if targetWins != 0 { return "自由局数不能设置目标胜局" }
            } else if format == "race_to" {
                if targetWins < 1 || targetWins > 65 { return "抢N局目标必须在1至65之间" }
            } else if format != "legacy" {
                return "请选择有效的赛制"
            }
        }
        return ""
    }

    static func normalizeOptions(matchMode: String? = nil, visibility: String? = nil) -> NormalizedStartMatchOptions {
        let rawMode = matchMode ?? ""
        let mode = rawMode.isEmpty ? "ranked" : rawMode
        let normalizedMode = (mode == "practice" || mode == "ranked") ? mode : "ranked"
        let fallbackVisibility = normalizedMode == "practice" ? "private" : "public"
        let rawVisibility = visibility ?? ""
        let resolvedVisibility = rawVisibility.isEmpty ? fallbackVisibility : rawVisibility
        return NormalizedStartMatchOptions(
            matchMode: normalizedMode,
            visibility: normalizedMode == "ranked" ? "public" : (resolvedVisibility == "public" ? "public" : "private")
        )
    }

    static func buildPayload(
        gameType: Int? = nil,
        opponent: StartMatchOpponent = StartMatchOpponent(),
        matchMode: String? = nil,
        visibility: String? = nil,
        inviteToken: String? = nil
    ) -> StartMatchRequest {
        let options = normalizeOptions(matchMode: matchMode, visibility: visibility)
        let normalizedToken = (inviteToken ?? "").trimmingCharacters(in: .whitespacesAndNewlines)
        var payload = StartMatchRequest(
            gameType: gameType ?? 0,
            matchMode: options.matchMode,
            visibility: options.visibility
        )
        if !normalizedToken.isEmpty {
            payload.inviteToken = normalizedToken
        } else {
            payload.opponentId = [opponent.id, opponent.userId, opponent.opponentId]
                .compactMap { $0 }
                .first { $0 != 0 } ?? 0
            payload.opponentName = [opponent.nickname, opponent.name, opponent.opponentName]
                .compactMap { $0 }
                .first { !$0.isEmpty } ?? "对手"
            payload.opponentAvatar = [opponent.avatar, opponent.opponentAvatar]
                .compactMap { $0 }
                .first { !$0.isEmpty } ?? ""
        }
        if payload.gameType == 1 {
            payload.snookerRulesVersion = 2
            payload.snookerFormat = "free"
            payload.snookerTargetWins = 0
        }
        if [3, 4].contains(payload.gameType ?? 0) {
            payload.matchFormat = "free"
            payload.targetWins = 0
        }
        return payload
    }
}
