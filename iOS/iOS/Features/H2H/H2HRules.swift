import Foundation

// 迁移自 `uniapp/utils/h2h-copywriting.js`、`h2h-view-model.js` 与 `h2h-record.js` 的纯规则。

enum AdvantageLevel: String, Equatable {
    case noSample = "no_sample"
    case balanced
    case slightLead = "slight_lead"
    case slightTrail = "slight_trail"
    case clearLead = "clear_lead"
    case clearTrail = "clear_trail"
}

struct H2HViewModel: Equatable {
    let isTargetMode: Bool
    let navigationTitle: String
    let subjectName: String
    let winRateLabel: String
    let isOpponentMe: Bool
}

struct H2HLoadFailureAction: Equatable {
    let toastMessage: String
}

struct H2HHero: Equatable {
    let title: String
    let subtitle: String
    let scoreText: String
    let leftName: String
    let leftAvatar: String
    let rightName: String
    let rightAvatar: String
    let badges: [String]
    let advantageLevel: AdvantageLevel
    let primaryActionText: String
}

struct H2HCalendarCell: Equatable {
    let dateKey: String
    let day: Int
    let isCurrentMonth: Bool
    let winCount: Int
    let lossCount: Int
    let drawCount: Int
    let matchCount: Int
    let matchIds: [Int64]

    var hasMatches: Bool { matchCount > 0 }
}

struct H2HCalendar: Equatable {
    static let weekdays = ["日", "一", "二", "三", "四", "五", "六"]

    let monthKey: String
    let title: String
    let cells: [H2HCalendarCell]
}

struct H2HMatchCard: Identifiable, Equatable {
    let id: Int64
    let result: Int
    let dateText: String
    let gameTypeText: String
    let scoreText: String
    let resultText: String
    let diffText: String
    let tags: [String]
}

enum H2HPageStatus: Equatable {
    case loading
    case error
    case partial
    case ready
    case empty
}

enum H2HHistoryViewMode: Equatable {
    case calendar
    case list
}

enum H2HRules {
    static let hiddenTitle = "对方已隐藏战绩"
    static let hiddenDescription = "这位好友暂时没有公开自己的战绩"

    // MARK: - 关系与文案

    static func advantageLevel(totalMatches: Int, myWins: Int, opponentWins: Int) -> AdvantageLevel {
        guard totalMatches > 0 else { return .noSample }
        if myWins == opponentWins { return .balanced }

        let lead = myWins - opponentWins
        let isLead = lead > 0
        if totalMatches < 5 {
            return isLead ? .slightLead : .slightTrail
        }
        if abs(lead) >= 3 {
            return isLead ? .clearLead : .clearTrail
        }
        return isLead ? .slightLead : .slightTrail
    }

    static func title(
        totalMatches: Int,
        myWins: Int,
        opponentWins: Int,
        opponentName: String = "对手",
        subjectName: String = "你"
    ) -> String {
        switch advantageLevel(totalMatches: totalMatches, myWins: myWins, opponentWins: opponentWins) {
        case .noSample: "\(subjectName)和\(opponentName)还没形成交锋样本"
        case .balanced: "\(subjectName)和\(opponentName)目前势均力敌"
        case .slightLead: "\(subjectName)对\(opponentName)略占上风"
        case .slightTrail: "\(subjectName)对\(opponentName)略处下风"
        case .clearLead: "\(subjectName)对\(opponentName)明显占优"
        case .clearTrail: "\(subjectName)最近被\(opponentName)压制"
        }
    }

    static func subtitle(myWins: Int, opponentWins: Int, recentHistory: [MatchListItem]) -> String {
        let recent = resultCounts(recentHistory.prefix(5))
        var parts = ["总交锋 \(String(myWins)) 胜 \(String(opponentWins)) 负"]

        if !recentHistory.isEmpty {
            var recentPart = "近 \(String(min(recentHistory.count, 5))) 场 \(String(recent.wins)) 胜 \(String(recent.losses)) 负"
            if recent.draws > 0 {
                recentPart += " \(String(recent.draws)) 平"
            }
            parts.append(recentPart)
        }

        return parts.joined(separator: "，")
    }

    static func primaryActionText(_ level: AdvantageLevel) -> String {
        switch level {
        case .clearLead, .slightLead: "再约一场，扩大优势"
        case .clearTrail, .slightTrail: "发起复仇局"
        case .balanced: "约一场决胜局"
        case .noSample: "约一场见真章"
        }
    }

    static func viewModel(
        targetUserId: Int64,
        targetName: String,
        opponentId: Int64,
        currentUserId: Int64 = 0,
        opponentName: String
    ) -> H2HViewModel {
        let isTargetMode = targetUserId > 0
        let isOpponentMe = isTargetMode && currentUserId > 0 && opponentId == currentUserId
        let resolvedOpponentName = opponentName.isEmpty ? "对手" : opponentName

        guard isTargetMode else {
            return H2HViewModel(
                isTargetMode: false,
                navigationTitle: "交锋记录 - \(resolvedOpponentName)",
                subjectName: "你",
                winRateLabel: "你对\(resolvedOpponentName)的胜率是",
                isOpponentMe: false
            )
        }

        let resolvedTargetName = targetName.isEmpty ? "这位球友" : targetName
        return H2HViewModel(
            isTargetMode: true,
            navigationTitle: "\(resolvedTargetName)的交锋记录",
            subjectName: resolvedTargetName,
            winRateLabel: "\(resolvedTargetName) 对\(resolvedOpponentName)的胜率是",
            isOpponentMe: isOpponentMe
        )
    }

    /// 好友关系失效时小程序提示并返回上一页；其余错误按普通失败处理。
    static func loadFailureAction(targetUserId: Int64, message: String) -> H2HLoadFailureAction? {
        guard targetUserId > 0, message == "仅可查看好友的对方战绩" else { return nil }
        return H2HLoadFailureAction(toastMessage: message)
    }

    // MARK: - 首屏概览

    static func hero(
        stats: H2HStats?,
        opponent: H2HOpponent?,
        history: [MatchListItem],
        subjectName: String,
        subjectAvatar: String
    ) -> H2HHero {
        let totalMatches = stats?.totalMatches ?? 0
        let myWins = stats?.myWins ?? 0
        let opponentWins = stats?.opponentWins ?? 0
        let level = advantageLevel(totalMatches: totalMatches, myWins: myWins, opponentWins: opponentWins)

        return H2HHero(
            title: title(
                totalMatches: totalMatches,
                myWins: myWins,
                opponentWins: opponentWins,
                opponentName: opponentDisplayName(opponent),
                subjectName: subjectName
            ),
            subtitle: subtitle(myWins: myWins, opponentWins: opponentWins, recentHistory: history),
            scoreText: "\(String(myWins)) : \(String(opponentWins))",
            leftName: subjectName,
            leftAvatar: subjectAvatar,
            rightName: opponentDisplayName(opponent),
            rightAvatar: opponent?.avatar ?? "",
            badges: [
                momentumBadge(history),
                streakBadge(maxWinStreak: stats?.maxWinStreak ?? 0),
                diffBadge(avgScoreDiff: stats?.avgScoreDiff ?? 0),
            ],
            advantageLevel: level,
            primaryActionText: primaryActionText(level)
        )
    }

    static func opponentDisplayName(_ opponent: H2HOpponent?) -> String {
        guard let name = opponent?.name, !name.isEmpty else { return "对手" }
        return name
    }

    static func momentumBadge(_ history: [MatchListItem]) -> String {
        let recent = history.prefix(5)
        guard !recent.isEmpty else { return "等待首场交锋" }
        let net = recent.reduce(0) { total, item in
            switch item.result {
            case 1: total + 1
            case 2: total - 1
            default: total
            }
        }
        if net >= 2 { return "近期强势" }
        if net <= -2 { return "近期回落" }
        return "五五开"
    }

    static func streakBadge(maxWinStreak: Int) -> String {
        maxWinStreak > 0 ? "当前 \(String(maxWinStreak)) 连胜" : "等待连胜"
    }

    static func diffBadge(avgScoreDiff: Double) -> String {
        "平均分差 \(avgScoreDiff > 0 ? "+" : "")\(fixed1(avgScoreDiff))"
    }

    // MARK: - 请求参数

    static func historyParams(
        targetUserId: Int64 = 0,
        opponentId: Int64 = 0,
        fallbackOpponentId: Int64 = 0,
        opponentName: String = "",
        page: Int = 1,
        pageSize: Int = 20,
        result: Int = 0,
        startDate: String = "",
        endDate: String = ""
    ) -> [String: CustomStringConvertible?] {
        let resolvedOpponentId = opponentId > 0 ? opponentId : fallbackOpponentId
        var params: [String: CustomStringConvertible?] = [
            "page": page,
            "page_size": pageSize,
        ]

        if targetUserId > 0 {
            params["target_user_id"] = targetUserId
        }
        if resolvedOpponentId > 0 {
            params["opponent_id"] = resolvedOpponentId
        } else if !opponentName.isEmpty {
            params["opponent_name"] = opponentName
        }
        if result > 0 {
            params["result"] = result
        }
        if !startDate.isEmpty {
            params["start_date"] = startDate
        }
        if !endDate.isEmpty {
            params["end_date"] = endDate
        }
        return params
    }

    static func monthDateRange(_ monthKey: String) -> (startDate: String, endDate: String)? {
        guard let parts = monthComponents(monthKey) else { return nil }
        let calendar = AppFormat.beijingCalendar
        guard let first = calendar.date(from: DateComponents(year: parts.year, month: parts.month, day: 1)),
              let days = calendar.range(of: .day, in: .month, for: first) else { return nil }
        let prefix = String(format: "%04d-%02d", parts.year, parts.month)
        return ("\(prefix)-01", "\(prefix)-\(String(format: "%02d", days.count))")
    }

    static func currentMonthKey(_ date: Date = Date()) -> String {
        AppFormat.monthKey(date)
    }

    static func shiftMonthKey(_ monthKey: String, offset: Int) -> String {
        let calendar = AppFormat.beijingCalendar
        guard let parts = monthComponents(monthKey),
              let date = calendar.date(from: DateComponents(year: parts.year, month: parts.month, day: 1)),
              let shifted = calendar.date(byAdding: .month, value: offset, to: date) else {
            return currentMonthKey()
        }
        return AppFormat.monthKey(shifted)
    }

    // MARK: - 月历与对局卡

    static func calendar(history: [MatchListItem], monthKey: String) -> H2HCalendar {
        let (year, month) = resolveCalendarMonth(history: history, monthKey: monthKey)
        let currentKey = String(format: "%04d-%02d", year, month)
        let calendar = AppFormat.beijingCalendar

        var byDay: [String: (wins: Int, losses: Int, draws: Int, count: Int, ids: [Int64])] = [:]
        for item in history {
            guard let date = AppFormat.parseDateTime(item.matchTime),
                  AppFormat.monthKey(date) == currentKey else { continue }
            let key = AppFormat.dayKey(date)
            var stats = byDay[key] ?? (0, 0, 0, 0, [])
            switch item.result {
            case 1: stats.wins += 1
            case 2: stats.losses += 1
            case 3: stats.draws += 1
            default: break
            }
            stats.count += 1
            if item.id != 0 {
                stats.ids.append(item.id)
            }
            byDay[key] = stats
        }

        let firstDay = calendar.date(from: DateComponents(year: year, month: month, day: 1))!
        // 周首为周日（Calendar 的 weekday 1 = 周日），与小程序 `getDay()` 一致。
        let weekday = calendar.component(.weekday, from: firstDay)
        let startDate = calendar.date(byAdding: .day, value: -(weekday - 1), to: firstDay)!

        let cells = (0..<42).map { index -> H2HCalendarCell in
            let date = calendar.date(byAdding: .day, value: index, to: startDate)!
            let key = AppFormat.dayKey(date)
            let stats = byDay[key] ?? (0, 0, 0, 0, [])
            let day = calendar.component(.day, from: date)
            return H2HCalendarCell(
                dateKey: key,
                day: day,
                isCurrentMonth: AppFormat.monthKey(date) == currentKey,
                winCount: stats.wins,
                lossCount: stats.losses,
                drawCount: stats.draws,
                matchCount: stats.count,
                matchIds: stats.ids
            )
        }

        return H2HCalendar(
            monthKey: currentKey,
            title: "\(String(year))年\(String(month))月",
            cells: cells
        )
    }

    static func matchCards(_ history: [MatchListItem]) -> [H2HMatchCard] {
        history.map { item in
            let diff = item.myScore - item.opponentScore
            let tag: String
            if item.result == 1 {
                tag = diff >= 4 ? "大胜" : "险胜"
            } else if item.result == 2 {
                tag = diff <= -4 ? "被压制" : "惜败"
            } else {
                tag = "平局"
            }

            return H2HMatchCard(
                id: item.id,
                result: item.result,
                dateText: item.matchTime,
                gameTypeText: item.gameTypeName.isEmpty ? "台球" : item.gameTypeName,
                scoreText: "\(String(item.myScore)) : \(String(item.opponentScore))",
                resultText: resultLabel(item.result),
                diffText: "\(diff > 0 ? "+" : "")\(String(diff))",
                tags: [tag]
            )
        }
    }

    static func matchDateText(_ text: String) -> String {
        guard let monthDay = AppFormat.monthDay(text) else { return "" }
        return "\(monthDay) · \(AppFormat.relativeTime(text))"
    }

    // MARK: - 页面状态

    static func pageStatus(
        statsLoaded: Bool,
        historyLoaded: Bool,
        totalMatches: Int,
        historyLength: Int,
        hasError: Bool
    ) -> H2HPageStatus {
        if hasError && (statsLoaded || historyLoaded) { return .partial }
        if hasError { return .error }
        if statsLoaded && historyLoaded {
            return totalMatches <= 0 && historyLength <= 0 ? .empty : .ready
        }
        return .loading
    }

    static func shouldShowSummaryCard(pageStatus: H2HPageStatus, statsLoaded: Bool) -> Bool {
        statsLoaded && pageStatus != .loading && pageStatus != .error
    }

    // MARK: - 私有

    private static func monthComponents(_ monthKey: String) -> (year: Int, month: Int)? {
        guard monthKey.range(of: #"^\d{4}-\d{2}$"#, options: .regularExpression) != nil else { return nil }
        let parts = monthKey.split(separator: "-")
        guard parts.count == 2,
              let year = Int(parts[0]),
              let month = Int(parts[1]),
              (1...12).contains(month) else { return nil }
        return (year, month)
    }

    private static func resolveCalendarMonth(
        history: [MatchListItem],
        monthKey: String
    ) -> (year: Int, month: Int) {
        if let parts = monthComponents(monthKey) { return parts }

        let calendar = AppFormat.beijingCalendar
        let latest = history.first { AppFormat.parseDateTime($0.matchTime) != nil }
        let date = latest.flatMap { AppFormat.parseDateTime($0.matchTime) } ?? Date()
        let parts = calendar.dateComponents([.year, .month], from: date)
        return (parts.year ?? 0, parts.month ?? 0)
    }

    private static func resultCounts(_ items: ArraySlice<MatchListItem>) -> (wins: Int, losses: Int, draws: Int) {
        items.reduce(into: (wins: 0, losses: 0, draws: 0)) { result, item in
            switch item.result {
            case 1: result.wins += 1
            case 2: result.losses += 1
            case 3: result.draws += 1
            default: break
            }
        }
    }

    private static func resultLabel(_ result: Int) -> String {
        switch result {
        case 1: "胜"
        case 2: "负"
        default: "平"
        }
    }

    /// `toFixed(1)` 的等价实现；`String(format:)` 不带千分位。
    private static func fixed1(_ value: Double) -> String {
        String(format: "%.1f", value)
    }
}
