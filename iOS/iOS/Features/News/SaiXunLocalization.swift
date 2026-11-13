import Foundation

// 赛讯文案对照表与本地化规则：逐条对应小程序 `utils/saixun.js`。
// 表内条目、顺序与文案必须与 JS 完全一致，测试会断言每张表的条目数。

/// 共享文本工具：逐条对应 JS 的 `trim` / 正则 / 大小写行为。
enum SaiXunText {
    static func trimmed(_ value: String?) -> String {
        (value ?? "").trimmingCharacters(in: .whitespacesAndNewlines)
    }

    static func containsCJK(_ text: String) -> Bool {
        text.unicodeScalars.contains { (0x3400...0x9FFF).contains($0.value) }
    }

    static func isAllDigits(_ text: String) -> Bool {
        !text.isEmpty && text.unicodeScalars.allSatisfy { $0.value >= 0x30 && $0.value <= 0x39 }
    }

    static func pad2(_ value: Int) -> String {
        value < 10 ? "0\(value)" : "\(value)"
    }

    /// 返回完整匹配与各捕获组；未参与匹配的组按空串返回（对应 JS 的 `undefined`，在判空处等价）。
    static func captures(
        _ pattern: String,
        _ text: String,
        options: NSRegularExpression.Options = [.caseInsensitive]
    ) -> [String]? {
        guard let regex = try? NSRegularExpression(pattern: pattern, options: options),
              let match = regex.firstMatch(in: text, range: NSRange(text.startIndex..., in: text)) else {
            return nil
        }
        return (0..<match.numberOfRanges).map { index in
            guard let range = Range(match.range(at: index), in: text) else { return "" }
            return String(text[range])
        }
    }

    static func hasMatch(
        _ pattern: String,
        _ text: String,
        options: NSRegularExpression.Options = [.caseInsensitive]
    ) -> Bool {
        captures(pattern, text, options: options) != nil
    }

    /// 与 JS `String.replace(/.../g)` 一致：替换全部匹配。
    static func replacing(
        _ pattern: String,
        in text: String,
        with template: String,
        options: NSRegularExpression.Options = [.caseInsensitive]
    ) -> String {
        guard let regex = try? NSRegularExpression(pattern: pattern, options: options) else { return text }
        let range = NSRange(text.startIndex..., in: text)
        return regex.stringByReplacingMatches(in: text, range: range, withTemplate: template)
    }
}

struct SaiXunPatternTranslation: Equatable {
    let pattern: String
    let zh: String
}

enum SaiXunTables {
    static let roundNameAliases: [String: String] = [
        "quarter_final": "四分之一决赛",
        "quarter_finals": "四分之一决赛",
        "quarterfinal": "四分之一决赛",
        "quarterfinals": "四分之一决赛",
        "semi_final": "半决赛",
        "semi_finals": "半决赛",
        "semifinal": "半决赛",
        "semifinals": "半决赛",
        "final": "决赛",
        "finals": "决赛",
        "league_phase": "联赛阶段",
        "pre_qualifier": "预资格赛",
        "pre_qualifier_round": "预资格赛",
        "wildcard_round": "外卡赛",
        "wild_card_round": "外卡赛",
    ]

    static let ordinalWordNumbers: [String: String] = [
        "one": "1",
        "two": "2",
        "three": "3",
        "four": "4",
        "five": "5",
        "six": "6",
        "seven": "7",
        "eight": "8",
        "nine": "9",
        "ten": "10",
    ]

    static let countryNameMap: [String: String] = [
        "england": "英格兰",
        "scotland": "苏格兰",
        "wales": "威尔士",
        "northern ireland": "北爱尔兰",
        "china": "中国",
        "germany": "德国",
        "saudi arabia": "沙特阿拉伯",
        "thailand": "泰国",
        "belgium": "比利时",
        "australia": "澳大利亚",
        "iran": "伊朗",
        "india": "印度",
        "pakistan": "巴基斯坦",
        "malta": "马耳他",
        "brazil": "巴西",
        "canada": "加拿大",
        "usa": "美国",
        "united states": "美国",
        "japan": "日本",
        "south korea": "韩国",
        "france": "法国",
        "italy": "意大利",
        "spain": "西班牙",
        "netherlands": "荷兰",
        "russia": "俄罗斯",
        "switzerland": "瑞士",
        "austria": "奥地利",
        "poland": "波兰",
        "norway": "挪威",
        "sweden": "瑞典",
        "denmark": "丹麦",
        "finland": "芬兰",
        "ireland": "爱尔兰",
        "hong kong": "中国香港",
        "taiwan": "中国台湾",
        "macau": "中国澳门",
    ]

    static let tournamentSeriesTranslations: [SaiXunPatternTranslation] = [
        SaiXunPatternTranslation(pattern: #"\bRiyadh Season Snooker Championship\b"#, zh: "利雅得狂欢季斯诺克锦标赛"),
        SaiXunPatternTranslation(pattern: #"\bSaudi Arabia Snooker Masters\b"#, zh: "沙特阿拉伯斯诺克大师赛"),
        SaiXunPatternTranslation(pattern: #"\bShanghai Masters\b"#, zh: "上海大师赛"),
        SaiXunPatternTranslation(pattern: #"\bGerman Masters\b"#, zh: "德国大师赛"),
        SaiXunPatternTranslation(pattern: #"\bTour Championship\b"#, zh: "斯诺克巡回锦标赛"),
        SaiXunPatternTranslation(pattern: #"\bPlayers Championship\b"#, zh: "球员锦标赛"),
        SaiXunPatternTranslation(pattern: #"\bChampion of Champions\b"#, zh: "冠中冠"),
        SaiXunPatternTranslation(pattern: #"\bWorld Championship\b"#, zh: "世界锦标赛"),
        SaiXunPatternTranslation(pattern: #"\bWorld Grand Prix\b"#, zh: "世界大奖赛"),
        SaiXunPatternTranslation(pattern: #"\bInternational Championship\b"#, zh: "国际锦标赛"),
        SaiXunPatternTranslation(pattern: #"\bBritish Open\b"#, zh: "英国公开赛"),
        SaiXunPatternTranslation(pattern: #"\bUK Championship\b"#, zh: "英国锦标赛"),
        SaiXunPatternTranslation(pattern: #"\bEnglish Open\b"#, zh: "英格兰公开赛"),
        SaiXunPatternTranslation(pattern: #"\bNorthern Ireland Open\b"#, zh: "北爱尔兰公开赛"),
        SaiXunPatternTranslation(pattern: #"\bScottish Open\b"#, zh: "苏格兰公开赛"),
        SaiXunPatternTranslation(pattern: #"\bWelsh Open\b"#, zh: "威尔士公开赛"),
        SaiXunPatternTranslation(pattern: #"\bWorld Open\b"#, zh: "世界公开赛"),
        SaiXunPatternTranslation(pattern: #"\bWuhan Open\b"#, zh: "武汉公开赛"),
        SaiXunPatternTranslation(pattern: #"\bShenzhen Open\b"#, zh: "深圳公开赛"),
        SaiXunPatternTranslation(pattern: #"\bChina Open\b"#, zh: "中国公开赛"),
        SaiXunPatternTranslation(pattern: #"\bXi'an Grand Prix\b"#, zh: "西安大奖赛"),
        SaiXunPatternTranslation(pattern: #"\bWorld Mixed Doubles\b"#, zh: "世界混双赛"),
        SaiXunPatternTranslation(pattern: #"\bWorld Masters of Snooker\b"#, zh: "世界斯诺克大师赛"),
        SaiXunPatternTranslation(pattern: #"\bAsia\s*[&-]\s*Oceania Q School\b"#, zh: "亚洲及大洋洲 Q School"),
        SaiXunPatternTranslation(pattern: #"\bQ School\b"#, zh: "Q School"),
        SaiXunPatternTranslation(pattern: #"\bShoot Out\b"#, zh: "单局限时赛"),
        SaiXunPatternTranslation(pattern: #"\bMasters\b"#, zh: "大师赛"),
    ]

    /// 资格赛 / 正赛阶段与主赛事同名，翻译时必须保留后缀，避免列表出现两张同名卡片。
    static let tournamentPhaseSuffixes: [SaiXunPatternTranslation] = [
        SaiXunPatternTranslation(pattern: #"\s+Qualifiers?$"#, zh: "资格赛"),
        SaiXunPatternTranslation(pattern: #"\s+Final Stages?$"#, zh: "决赛阶段"),
    ]

    /// 0:即将开始 1:进行中 2:已结束 3:已取消。
    static let statusTones: [Int: String] = [
        0: "upcoming",
        1: "live",
        2: "ended",
        3: "canceled",
    ]

    /// `Date.getDay()`：0 为周日。
    static let weekdayLabels: [String] = ["周日", "周一", "周二", "周三", "周四", "周五", "周六"]

    static let subdivisionFlagImages: [String: String] = [
        "eng": "/static/flags/eng.png",
        "sct": "/static/flags/sco.png",
        "wls": "/static/flags/wls.png",
    ]

    /// 与国家码一起返回的三字母标识；当前版本未展示，保留以便与小程序表一致。
    static let subdivisionLabels: [String: String] = [
        "eng": "ENG",
        "sct": "SCO",
        "wls": "WLS",
    ]
}

extension SaiXunRules {
    static func localizeTournamentTitle(_ value: String?) -> String {
        let title = SaiXunText.trimmed(value)
        if title.isEmpty || SaiXunText.containsCJK(title) { return title }

        if let groups = SaiXunText.captures(
            #"^(?:.+?\s+)?Championship League Snooker(?:\s+Invitational)?\s+(\d{4})\s+\(([^)]+)\)$"#,
            title
        ) {
            return titleWithYearAndSuffix("冠军联赛", year: groups[1], suffix: localizeLeagueStageSuffix(groups[2]))
        }

        if let groups = SaiXunText.captures(#"^(?:.+?\s+)?Championship League Group\s+(.+)$"#, title) {
            let groupNumber = normalizeOrdinalToken(groups[1])
            return groupNumber.isEmpty ? "冠军联赛小组赛" : "冠军联赛第\(groupNumber)组"
        }

        if SaiXunText.hasMatch(#"^(?:.+?\s+)?Championship League Winners Group$"#, title) {
            return "冠军联赛胜者组"
        }

        if let groups = SaiXunText.captures(#"^Q School\s+(\d{4})\s*-\s*Event\s+(\d+)$"#, title) {
            return "Q School \(groups[1]) - 第\(groups[2])站"
        }

        let phase = SaiXunTables.tournamentPhaseSuffixes.first {
            SaiXunText.hasMatch($0.pattern, title)
        }
        let titleWithoutPhase = phase.map {
            SaiXunText.replacing($0.pattern, in: title, with: "")
        } ?? title
        let year = extractTournamentYear(titleWithoutPhase)
        let titleWithoutYear = stripYearFromTitle(titleWithoutPhase, year: year)
        let seriesTitle = localizeTournamentSeries(titleWithoutYear)

        if !seriesTitle.isEmpty {
            let localized = titleWithYearAndSuffix(seriesTitle, year: year, suffix: "")
            return phase.map { "\(localized) \($0.zh)" } ?? localized
        }

        return title
    }

    static func standardizeRoundText(_ value: String?) -> String {
        let text = SaiXunText.trimmed(value)
        if text.isEmpty { return "" }

        if let alias = SaiXunTables.roundNameAliases[normalizeRoundKey(text)] { return alias }

        // 冠中冠等赛事的小组内淘汰轮次，如 "Group Semi Finals" / "Group Final"。
        if let groups = SaiXunText.captures(#"^Group\s+(.+)$"#, text),
           let alias = SaiXunTables.roundNameAliases[normalizeRoundKey(groups[1])] {
            return "小组\(alias)"
        }

        // 保留赛有 "Round 1 (Held Over)" 与 "Round 1/Heldover" 两种写法。
        if let groups = SaiXunText.captures(
            #"^Round\s+(\d+)(?:\s*\(([^)]+)\)|\s*\/\s*(.+))?$"#,
            text
        ) {
            let qualifier = groups[2].isEmpty ? groups[3] : groups[2]
            return qualifier.isEmpty
                ? "第\(groups[1])轮"
                : "第\(groups[1])轮（\(localizeRoundQualifier(qualifier))）"
        }

        if let groups = SaiXunText.captures(#"^Last\s+(\d+)$"#, text) {
            return "\(groups[1])强"
        }

        if let groups = SaiXunText.captures(#"^League Phase\s*\(([^)]+)\)$"#, text) {
            return "联赛阶段（\(localizeLeagueStageSuffix(groups[1]))）"
        }

        if SaiXunText.containsCJK(text) { return text }
        return localizeLeagueStageSuffix(text)
    }

    /// 对阵未定时 WST 用 "Winner of Match 7" / "Winner Semi One" 等占位名，球员真名保持原样。
    static func localizeMatchPlayerName(_ value: String?) -> String {
        let text = SaiXunText.trimmed(value)
        if text.isEmpty { return "" }
        if SaiXunText.hasMatch(#"^(?:TBC|TBA|TBD)$"#, text) { return "待定" }

        if let groups = SaiXunText.captures(#"^(Winner|Loser)\s+of\s+Match\s+(\d+)$"#, text) {
            let side = groups[1].lowercased() == "winner" ? "胜者" : "负者"
            return "第\(groups[2])场\(side)"
        }

        if let groups = SaiXunText.captures(#"^Winner\s+(?:of\s+)?Semi(?:[\s-]*Finals?)?\s+(\S+)$"#, text) {
            let semiNo = normalizeOrdinalToken(groups[1])
            if SaiXunText.isAllDigits(semiNo) { return "半决赛\(semiNo)胜者" }
        }

        return text
    }

    static func localizeCountryName(_ value: String?) -> String {
        let text = SaiXunText.trimmed(value)
        if text.isEmpty { return "" }
        if SaiXunText.containsCJK(text) { return text }
        return SaiXunTables.countryNameMap[text.lowercased()] ?? text
    }

    static func sanitizeFlagEmoji(_ value: String?) -> String {
        let text = SaiXunText.trimmed(value)
        if text.isEmpty { return "" }
        // 黑色旗帜基座（U+1F3F4）：所有细分地区旗帜改走本地图片。
        if text.unicodeScalars.first?.value == 0x1F3F4 { return "" }
        return text
    }

    static func resolvePlayerFlag(_ flagEmoji: String?, _ countryCode: String?) -> SaiXunFlag {
        let code = SaiXunText.trimmed(countryCode).lowercased()
        switch code {
        case "gb-eng", "eng":
            return SaiXunFlag(type: .image, value: SaiXunTables.subdivisionFlagImages["eng"] ?? "")
        case "gb-sct", "sct":
            return SaiXunFlag(type: .image, value: SaiXunTables.subdivisionFlagImages["sct"] ?? "")
        case "gb-wls", "wls":
            return SaiXunFlag(type: .image, value: SaiXunTables.subdivisionFlagImages["wls"] ?? "")
        case "gb-nir", "nir":
            return SaiXunFlag(type: .image, value: "/static/flags/nir.png")
        default:
            let emoji = SaiXunText.trimmed(flagEmoji)
            return emoji.isEmpty ? SaiXunFlag(type: .none, value: "") : SaiXunFlag(type: .emoji, value: emoji)
        }
    }

    // MARK: - 私有规则

    private static func titleWithYearAndSuffix(_ label: String, year: String, suffix: String) -> String {
        if label.isEmpty { return "" }

        var result = label
        if !year.isEmpty { result += " \(year)" }
        if !suffix.isEmpty { result += "（\(suffix)）" }
        return result
    }

    private static func extractTournamentYear(_ text: String) -> String {
        guard let regex = try? NSRegularExpression(pattern: #"\b(19|20)\d{2}\b"#) else { return "" }
        let matches = regex.matches(in: text, range: NSRange(text.startIndex..., in: text))
        guard let last = matches.last, let range = Range(last.range, in: text) else { return "" }
        return String(text[range])
    }

    private static func stripYearFromTitle(_ text: String, year: String) -> String {
        if year.isEmpty { return SaiXunText.trimmed(text) }
        guard let range = text.range(of: year, options: .backwards) else {
            return SaiXunText.trimmed(text)
        }

        var result = text.replacingCharacters(in: range, with: "")
        result = SaiXunText.replacing(#"\s{2,}"#, in: result, with: " ", options: [])
        result = SaiXunText.replacing(#"\s+([)\]-])"#, in: result, with: "$1", options: [])
        result = SaiXunText.replacing(#"([([\-])\s+"#, in: result, with: "$1", options: [])
        return SaiXunText.trimmed(result)
    }

    private static func localizeTournamentSeries(_ text: String) -> String {
        guard !text.isEmpty else { return "" }
        return SaiXunTables.tournamentSeriesTranslations.first {
            SaiXunText.hasMatch($0.pattern, text)
        }?.zh ?? ""
    }

    private static func normalizeOrdinalToken(_ value: String?) -> String {
        let text = SaiXunText.trimmed(value)
        if text.isEmpty { return "" }
        if SaiXunText.isAllDigits(text) { return text }

        let normalized = SaiXunText.replacing(#"[^a-z0-9]+"#, in: text.lowercased(), with: "", options: [])
        return SaiXunTables.ordinalWordNumbers[normalized] ?? text
    }

    private static func normalizeRoundKey(_ value: String?) -> String {
        let lowered = SaiXunText.trimmed(value).lowercased()
        let underscored = SaiXunText.replacing(#"[^a-z]+"#, in: lowered, with: "_", options: [])
        return underscored.trimmingCharacters(in: CharacterSet(charactersIn: "_"))
    }

    private static func localizeRoundQualifier(_ value: String?) -> String {
        let text = SaiXunText.trimmed(value)
        if SaiXunText.hasMatch(#"^held\s*over$"#, text) { return "保留赛" }
        return localizeLeagueStageSuffix(text)
    }

    private static func localizeLeagueStageSuffix(_ value: String?) -> String {
        let text = SaiXunText.trimmed(value)
        if text.isEmpty { return "" }

        var result = text
        result = SaiXunText.replacing(#"\bStage One\b"#, in: result, with: "第一阶段")
        result = SaiXunText.replacing(#"\bStage Two\b"#, in: result, with: "第二阶段")
        result = SaiXunText.replacing(#"\bStage Three\b"#, in: result, with: "第三阶段")
        result = SaiXunText.replacing(#"\bW(?:EE)?K\s*(\d+)\b"#, in: result, with: "第$1周")
        result = SaiXunText.replacing(#"\bWinners Group\b"#, in: result, with: "胜者组")
        result = SaiXunText.replacing(#"\bGroup\s+(\d+)\b"#, in: result, with: "第$1组")
        result = SaiXunText.replacing(#"\s*&\s*Final\b"#, in: result, with: "及决赛")
        result = SaiXunText.replacing(#"\s*\/\s*"#, in: result, with: " / ", options: [])
        result = SaiXunText.replacing(#"\s{2,}"#, in: result, with: " ", options: [])
        return SaiXunText.trimmed(result)
    }
}
