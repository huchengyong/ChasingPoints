import Foundation

// 对局记录的窄解码模型：只包含列表与战报用到的字段，实时计分字段不进入模型。
// 字段与 backend/chasing_points.api 对齐；服务端始终返回的数组按非可选解码，null 视为响应格式错误。

struct MatchListItem: Decodable, Identifiable, Equatable {
    let id: Int64
    let gameType: Int
    let gameTypeName: String
    let opponentName: String
    let opponentAvatar: String
    let myScore: Int
    let opponentScore: Int
    let result: Int
    let matchTime: String
}

struct RefereeHistoryItem: Decodable, Identifiable, Equatable {
    let id: Int64
    let player1Name: String
    let player1Avatar: String
    let player2Name: String
    let player2Avatar: String
    let player1Score: Int
    let player2Score: Int
    let gameType: Int
    let gameTypeName: String
    let status: Int
    let endTime: String?
}

struct MatchRankDetail: Decodable, Equatable {
    let label: String
    let value: Int
}

struct MatchSummaryItem: Decodable, Equatable {
    let label: String
    let value: String
    let desc: String?
    let tone: String?
}

/// `GET /api/match/detail` 的战报数据，视角由服务端按当前用户决定（`my_*` 为本人）。
struct MatchReport: Decodable, Equatable {
    let id: Int64
    let gameType: Int
    let status: Int
    let viewerRole: String?
    let myName: String
    let myAvatar: String
    let myScore: Int
    let myWinRate: Double
    let myRankChange: Int
    let myRankDetails: [MatchRankDetail]?
    let opponentName: String
    let opponentAvatar: String
    let opponentScore: Int
    let opponentWinRate: Double
    let opponentRankChange: Int
    let summaryHighlights: [MatchSummaryItem]?
    let summaryStats: [MatchSummaryItem]?
    let refereeBound: Bool?
    let refereeUserId: Int64?
    let refereeName: String?
    let refereeAvatar: String?
    let refereeDurationSeconds: Int64?
    let completionSource: String?
    let createdAt: String

    var isRefereeView: Bool { viewerRole == "referee" }
}

struct MatchRewardItem: Decodable, Identifiable, Equatable {
    let achievementId: Int64
    let achievementName: String
    let description: String
    let icon: String
    let rewardTitleName: String?

    var id: Int64 { achievementId }
}

/// `GET /api/match/reward-summary`：`pending` 表示仍在处理，`ready` 且列表为空表示本场无新解锁。
struct MatchRewardSummary: Decodable, Equatable {
    let status: String
    let list: [MatchRewardItem]
}

enum MatchRewardState: Equatable {
    case hidden
    case pending
    case ready([MatchRewardItem])
}

extension MatchRewardSummary {
    var state: MatchRewardState {
        switch status {
        case "pending": .pending
        case "ready" where !list.isEmpty: .ready(list)
        default: .hidden
        }
    }
}

enum MatchOutcome: Equatable {
    case win
    case loss
    case draw

    /// 列表使用服务端 `result`（1 胜、2 负、3 平）；未知值不猜测胜负。
    init?(result: Int) {
        switch result {
        case 1: self = .win
        case 2: self = .loss
        case 3: self = .draw
        default: return nil
        }
    }

    /// 战报没有 `result` 字段，按最终比分判定，规则与服务端结算一致。
    init(myScore: Int, opponentScore: Int) {
        if myScore > opponentScore {
            self = .win
        } else if myScore < opponentScore {
            self = .loss
        } else {
            self = .draw
        }
    }

    var title: String {
        switch self {
        case .win: "胜利"
        case .loss: "失败"
        case .draw: "平局"
        }
    }
}

enum MatchFormatting {
    static func gameTypeName(_ gameType: Int) -> String {
        switch gameType {
        case 1: "斯诺克"
        case 2: "九球追分"
        case 3: "中式八球"
        case 4: "美式九球"
        default: "台球"
        }
    }

    /// 九球追分按分计分，其余球种按局计分。
    static func scoreUnit(_ gameType: Int) -> String {
        gameType == 2 ? "分" : "局"
    }

    static func completionSourceLabel(_ source: String?) -> String {
        switch source {
        case "referee": "裁判结算"
        case "player_direct": "选手直接完成"
        case "player_confirmed": "双方确认完成"
        default: "未知"
        }
    }

    /// 完成方式为空或 `unknown` 时不展示归因，避免把缺失数据说成具体结算方。
    static func hasReliableCompletionSource(_ source: String?) -> Bool {
        guard let source else { return false }
        return !source.isEmpty && source != "unknown"
    }

    static func duration(seconds: Int64?) -> String? {
        guard let seconds, seconds > 0 else { return nil }
        if seconds < 60 { return "\(seconds)秒" }
        let minutes = seconds / 60
        let remainSeconds = seconds % 60
        if minutes < 60 {
            return remainSeconds > 0 ? "\(minutes)分\(remainSeconds)秒" : "\(minutes)分钟"
        }
        let hours = minutes / 60
        let remainMinutes = minutes % 60
        return remainMinutes > 0 ? "\(hours)小时\(remainMinutes)分" : "\(hours)小时"
    }
}

/// 时间字段形如 `2026-10-09T20:30:00+08:00`；解析失败统一显示“时间待同步”，不显示错误时间。
enum MatchTimeFormatting {
    static let unavailableText = "时间待同步"

    static func parse(_ text: String?) -> Date? {
        guard let text, !text.isEmpty else { return nil }
        let formatter = ISO8601DateFormatter()
        formatter.formatOptions = [.withInternetDateTime]
        return formatter.date(from: text)
    }

    /// 与 UniApp 列表的相对时间规则一致：一个月内用相对时间，之后显示日期。
    static func relative(_ text: String?, now: Date = Date()) -> String {
        guard let date = parse(text) else { return unavailableText }
        let seconds = Int(now.timeIntervalSince(date))
        if seconds < 60 { return "刚刚" }
        let minutes = seconds / 60
        if minutes < 60 { return "\(minutes)分钟前" }
        let hours = minutes / 60
        if hours < 24 { return "\(hours)小时前" }
        let days = hours / 24
        if days < 30 { return "\(days)天前" }
        return absolute(text)
    }

    static func absolute(_ text: String?) -> String {
        guard let date = parse(text) else { return unavailableText }
        let formatter = DateFormatter()
        formatter.locale = Locale(identifier: "en_US_POSIX")
        formatter.timeZone = TimeZone(identifier: "Asia/Shanghai")
        formatter.dateFormat = "yyyy-MM-dd HH:mm"
        return formatter.string(from: date)
    }
}
