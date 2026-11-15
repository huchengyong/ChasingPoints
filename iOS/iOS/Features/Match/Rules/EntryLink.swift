import Foundation

/// 入口链接的解析结果；`home` 是无参数首页，`other` 表示根本不认识（不是入口链接）。
enum EntryLinkResult: Equatable {
    case home
    case invite(inviteCode: String)
    case match(matchId: Int64)
    case other
}

/// 逐条对应小程序 `utils/entry-link.js`。入口链接、扫码和分享卡片共用同一套识别规则。
enum EntryLink {
    static let base = "https://www.zhuifen.cn/go/"

    private static let entryLinkPattern = try! NSRegularExpression(
        pattern: #"^https:\/\/www\.zhuifen\.cn\/go\/?(?:\?([^#]*))?(?:#.*)?$"#,
        options: [.caseInsensitive]
    )
    private static let inviteCodePattern = try! NSRegularExpression(pattern: #"^[0-9A-Za-z]{12}$"#)
    private static let positiveIntegerPattern = try! NSRegularExpression(pattern: #"^[1-9][0-9]*$"#)
    /// JS 的 Number.isSafeInteger 上限。
    private static let maxSafeInteger: Int64 = 9_007_199_254_740_991

    static func buildLink(_ target: EntryLinkTarget) -> String {
        switch target {
        case .invite(let code): "\(base)?j=\(code)"
        case .match(let matchId): "\(base)?m=\(matchId)"
        case .home: base
        }
    }

    /// 入口链接、端内扫码和分享卡片打开对局时共用的观战详情目的地。
    static func spectateDestination(matchId: Int64) -> MatchDestination {
        .spectate(matchId: matchId)
    }

    static func parse(_ raw: String?) -> EntryLinkResult {
        let value = (raw ?? "").trimmingCharacters(in: .whitespacesAndNewlines)
        let range = NSRange(value.startIndex..., in: value)
        guard let matched = entryLinkPattern.firstMatch(in: value, range: range) else {
            return .other
        }

        let queryGroup = matched.range(at: 1)
        let query = queryGroup.location == NSNotFound
            ? ""
            : String(value[Range(queryGroup, in: value)!])
        let params = readQuery(query)
        let hasCode = params["j"] != nil
        let hasMatchId = params["m"] != nil
        if hasCode && hasMatchId { return .home }

        if let code = params["j"] {
            return inviteCodePattern.matches(code) ? .invite(inviteCode: code) : .home
        }
        if let rawMatchId = params["m"] {
            guard positiveIntegerPattern.matches(rawMatchId),
                  let matchId = Int64(rawMatchId),
                  matchId <= maxSafeInteger else {
                return .home
            }
            return .match(matchId: matchId)
        }
        return .home
    }

    /// 微信启动参数 q 是原链接的 URL 编码，少数场景会编码两次；缺失或解码失败时原样返回。
    static func decodeLaunchQuery(_ q: String?) -> String {
        var value = q ?? ""
        for _ in 0..<2 {
            guard let decoded = value.removingPercentEncoding, decoded != value else { break }
            value = decoded
        }
        return value
    }

    /// 同名参数只取第一个；没有 `=` 的参数值为空串，与小程序的 URL 解析一致。
    private static func readQuery(_ query: String) -> [String: String] {
        var params: [String: String] = [:]
        for pair in query.split(separator: "&", omittingEmptySubsequences: true) {
            let text = String(pair)
            let index = text.firstIndex(of: "=")
            let key = index.map { String(text[..<$0]) } ?? text
            if params[key] != nil { continue }
            params[key] = index.map { String(text[text.index(after: $0)...]) } ?? ""
        }
        return params
    }
}

enum EntryLinkTarget: Equatable {
    case home
    case invite(code: String)
    case match(matchId: Int64)
}

private extension NSRegularExpression {
    func matches(_ value: String) -> Bool {
        firstMatch(in: value, range: NSRange(value.startIndex..., in: value)) != nil
    }
}
