import Foundation

// 信誉明细的窄解码模型，字段与 `backend/chasing_points.api` 对齐。
// 后端业务失败时 Go 的 nil slice 会编码成 `null`，列表按可选解码后归一为空数组。

struct ReputationLog: Decodable, nonisolated Identifiable, Equatable {
    let id: Int64
    let matchId: Int64?
    let changeType: String
    let changeTypeText: String
    let reasonCode: String
    let reasonText: String
    let changeScore: Int
    let beforeScore: Int
    let afterScore: Int
    let createdAt: String

    var displayReason: String { reasonText.isEmpty ? "信誉变更" : reasonText }
    var displayType: String { changeTypeText.isEmpty ? "记录" : changeTypeText }
    /// 与小程序 `formatChangeScore` 一致：正数加 `+`，不补千分位。
    var changeScoreText: String { changeScore > 0 ? "+\(changeScore)" : String(changeScore) }
    var isMatchLog: Bool { (matchId ?? 0) > 0 }
}

struct ReputationLogsResponse: Decodable {
    let success: Bool
    let total: Int
    let list: [ReputationLog]?
}
