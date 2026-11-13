import Foundation

// 迁移自 `uniapp/utils/member-performance-view.js` 的展示规则。

enum PerformanceTone: String, Equatable {
    case up
    case down
    case flat
}

struct PerformanceRatingView: Equatable {
    let available: Bool
    let value: String
    let qualifier: String
    let text: String
}

struct PerformanceSummaryView: Equatable {
    let available: Bool
    let sampleText: String
    let wins: Int
    let expectedText: String
    let diffText: String
    let tone: PerformanceTone
    let headline: String
    let rating: PerformanceRatingView
    let ratingHint: String
}

enum PerformanceMonthStatus: String, Equatable {
    case ready
    case incomplete
    case insufficient
}

struct PerformanceMonthBar: Equatable, Identifiable {
    let month: String
    let label: String
    let isCurrent: Bool
    let status: PerformanceMonthStatus
    let sampleCount: Int
    let ratingText: String
    let qualifier: String
    /// 当月表现分在共享刻度里的位置；不可用时为 0。
    let heightPercent: Double
    let ratingValue: Int?
    let endRankScore: Int
    /// 月末排位分在共享刻度里的位置；为 0 时没有点。
    let dotPercent: Double?
    let note: String

    var id: String { month }
}

struct PerformanceMonthChart: Equatable {
    let hasData: Bool
    let scaleMin: Int
    let scaleMax: Int
    let bars: [PerformanceMonthBar]
}

enum PerformanceNotableKind: String, Equatable {
    case bestWin = "best_win"
    case worstLoss = "worst_loss"
}

struct PerformanceNotableMatchView: Equatable, Identifiable {
    let kind: PerformanceNotableKind
    let title: String
    let matchId: Int64
    let opponentName: String
    let opponentAvatar: String
    let headline: String
    let detail: String
    let playedAt: String

    var id: PerformanceNotableKind { kind }
}

struct PerformancePreview: Equatable {
    let monthText: String
    let available: Bool
    let ratingText: String
    let qualifier: String
    let value: String
    let sampleCount: Int
    let hint: String
}

struct PerformanceLockedView: Equatable {
    let sampleText: String
    let hasSamples: Bool
    let features: [String]
    let preview: PerformancePreview
    let showPurchase: Bool
}

enum PerformanceRules {
    static let sampleCap = 200
    static let minSummarySamples = 5
    static let minMonthSamples = 3
    static let monthFootnote = "每月满 \(minMonthSamples) 场排位才显示表现分；读取上限内放不下的月份会标为数据不全。"
    static let note = "表现分只反映这段时间打出的水平，不改变排位分。"

    static let features = [
        "近 90 天实际胜场与期望胜场对比",
        "近 90 天表现分",
        "最佳胜利与爆冷失利",
        "最近 6 个月实力曲线",
    ]

    /// 失效版本：对局结算（rank）或会员状态变化（member）后，下一次进入页面回源。
    static func version(rankVersion: Int, memberVersion: Int) -> String {
        "\(String(rankVersion)):\(String(memberVersion))"
    }

    static func formatSampleCount(_ count: Int) -> String {
        let value = max(0, count)
        return value >= sampleCap ? "\(String(sampleCap))+" : String(value)
    }

    /// 表现分的展示：精确值直接显示；全胜/全负按求解边界显示「至少 / 至多」；样本不足为空。
    static func resolveRatingView(_ rating: PerformanceRating?) -> PerformanceRatingView {
        guard let rating else {
            return PerformanceRatingView(available: false, value: "", qualifier: "", text: "—")
        }
        let value = String(rating.rating)
        let qualifier: String
        switch rating.bound {
        case "at_least": qualifier = "至少"
        case "at_most": qualifier = "至多"
        default: qualifier = ""
        }
        return PerformanceRatingView(
            available: true,
            value: value,
            qualifier: qualifier,
            text: qualifier.isEmpty ? value : "\(qualifier)\(value)"
        )
    }

    /// 90 天摘要：实际胜场与期望胜场的差值，以及一句口径明确的解读。
    static func resolveSummaryView(_ summary: PerformanceSummary?, sampleCount: Int) -> PerformanceSummaryView {
        let samples = max(0, sampleCount)
        guard let summary else {
            return PerformanceSummaryView(
                available: false,
                sampleText: formatSampleCount(samples),
                wins: 0,
                expectedText: "",
                diffText: "",
                tone: .flat,
                headline: "",
                rating: resolveRatingView(nil),
                ratingHint: ""
            )
        }

        let diff = summary.winsDiff
        let tone: PerformanceTone = diff >= 0.5 ? .up : (diff <= -0.5 ? .down : .flat)
        let headline: String
        switch tone {
        case .up: headline = "近 90 天打得比对手实力预期更好"
        case .down: headline = "近 90 天赢的场次略少于对手实力预期"
        case .flat: headline = "近 90 天的战绩与对手实力预期基本一致"
        }
        let rating = resolveRatingView(summary.rating)

        return PerformanceSummaryView(
            available: true,
            sampleText: formatSampleCount(samples),
            wins: max(0, summary.wins),
            expectedText: toFixed1(summary.expectedWins),
            diffText: formatSigned(diff),
            tone: tone,
            headline: headline,
            rating: rating,
            ratingHint: rating.available ? "" : "近 90 天样本不足 \(String(minSummarySamples)) 场，打满后显示表现分"
        )
    }

    /// 6 个月实力曲线：柱是当月表现分，点是月末排位分。柱与点共用同一个刻度，刻度按数据范围向外取整到 100。
    /// 样本不足或被读取上限截断的月份不画柱，并说明原因，避免把残缺数据当成水平。
    static func buildMonthChart(_ months: [PerformanceMonthItem]?) -> PerformanceMonthChart {
        let list = months ?? []
        var values: [Int] = []
        for item in list {
            let rating = resolveRatingView(item.rating)
            if rating.available, item.incomplete != true {
                values.append(item.rating?.rating ?? 0)
            }
            if item.endRankScore > 0 {
                values.append(item.endRankScore)
            }
        }
        guard let minValue = values.min(), let maxValue = values.max() else {
            return PerformanceMonthChart(hasData: false, scaleMin: 0, scaleMax: 0, bars: [])
        }

        var scaleMin = Int(floor((Double(minValue) - 50) / 100)) * 100
        var scaleMax = Int(ceil((Double(maxValue) + 50) / 100)) * 100
        if scaleMax - scaleMin < 200 { scaleMax = scaleMin + 200 }
        scaleMin = max(0, scaleMin)

        let bars = list.enumerated().map { index, item -> PerformanceMonthBar in
            let rating = resolveRatingView(item.rating)
            let incomplete = item.incomplete == true
            let endRankScore = max(0, item.endRankScore)
            let sampleCount = max(0, item.sampleCount)
            let ready = rating.available && !incomplete
            let status: PerformanceMonthStatus = ready ? .ready : (incomplete ? .incomplete : .insufficient)
            let ratingValue = ready ? item.rating?.rating : nil

            return PerformanceMonthBar(
                month: item.month,
                label: chartMonthLabel(item.month),
                isCurrent: index == list.count - 1,
                status: status,
                sampleCount: sampleCount,
                ratingText: ready ? rating.text : "—",
                qualifier: ready ? rating.qualifier : "",
                heightPercent: ratingValue.map {
                    toPercent(Double($0), min: Double(scaleMin), max: Double(scaleMax))
                } ?? 0,
                ratingValue: ratingValue,
                endRankScore: endRankScore,
                dotPercent: endRankScore > 0
                    ? toPercent(Double(endRankScore), min: Double(scaleMin), max: Double(scaleMax))
                    : nil,
                note: ready ? "" : (incomplete ? "数据不全" : "样本不足")
            )
        }

        return PerformanceMonthChart(hasData: true, scaleMin: scaleMin, scaleMax: scaleMax, bars: bars)
    }

    /// 最佳胜利 / 爆冷失利：展示对手、赛前对手排位分与赛前整场胜率。
    static func resolveNotableMatchView(
        _ item: PerformanceMatchItem?,
        kind: PerformanceNotableKind
    ) -> PerformanceNotableMatchView? {
        guard let item else { return nil }
        let rateText = percentText(item.expectedWinRate)
        let name = item.opponentName.isEmpty ? "对手" : item.opponentName
        let isWin = kind == .bestWin

        return PerformanceNotableMatchView(
            kind: kind,
            title: isWin ? "最佳胜利" : "爆冷失利",
            matchId: item.matchId,
            opponentName: name,
            opponentAvatar: item.opponentAvatar,
            headline: isWin ? "击败 \(name)" : "负于 \(name)",
            detail: isWin
                ? "对手赛前排位分 \(String(item.opponentBeforeScore))，赛前你的胜率只有 \(rateText)"
                : "对手赛前排位分 \(String(item.opponentBeforeScore))，赛前你的胜率有 \(rateText)",
            playedAt: item.playedAt
        )
    }

    /// 锁定态（非有效会员）：只展示权益说明、服务端给出的样本数与本月表现分预览。
    /// 开通入口是否出现完全由调用方传入的合规开关决定，页面不自行判断。
    static func resolveLockedView(
        _ response: PerformanceResponse?,
        canShowPurchase: Bool
    ) -> PerformanceLockedView {
        let samples = max(0, response?.sampleCount ?? 0)
        let preview = response?.monthPreview
        let previewRating = resolveRatingView(preview?.rating)
        let previewSamples = max(0, preview?.sampleCount ?? 0)
        let monthText = preview?.month.isEmpty == false
            ? monthLabel(preview?.month ?? "")
            : "本月"

        return PerformanceLockedView(
            sampleText: formatSampleCount(samples),
            hasSamples: samples > 0,
            features: features,
            preview: PerformancePreview(
                monthText: monthText,
                available: previewRating.available,
                ratingText: previewRating.text,
                qualifier: previewRating.qualifier,
                value: previewRating.value,
                sampleCount: previewSamples,
                hint: previewRating.available
                    ? "基于\(monthText)已打的 \(String(previewSamples)) 场排位"
                    : "\(monthText)已打 \(String(previewSamples)) 场排位，满 \(String(minMonthSamples)) 场后显示表现分"
            ),
            showPurchase: canShowPurchase
        )
    }

    // MARK: - 私有

    private static func toPercent(_ value: Double, min: Double, max: Double) -> Double {
        guard max > min else { return 0 }
        return Swift.min(100, Swift.max(0, (value - min) / (max - min) * 100))
    }

    private static func formatSigned(_ value: Double) -> String {
        let rounded = Double(toFixed1(abs(value))) ?? 0
        if rounded == 0 { return toFixed1(0) }
        return "\(value > 0 ? "+" : "-")\(toFixed1(rounded))"
    }

    private static func toFixed1(_ value: Double) -> String {
        String(format: "%.1f", value)
    }

    private static func percentText(_ value: Double) -> String {
        let text = value == value.rounded() ? String(Int(value)) : toFixed1(value)
        return "\(text)%"
    }

    /// 曲线列标签：`YYYY-MM` 取月份数字，非法值原样展示。
    private static func chartMonthLabel(_ month: String) -> String {
        guard let number = monthNumber(month), (1...12).contains(number) else { return month }
        return "\(String(number))月"
    }

    /// 预览月份标签：无法解析时回落到「本月」。
    private static func monthLabel(_ month: String) -> String {
        guard let number = monthNumber(month), (1...12).contains(number) else { return "本月" }
        return "\(String(number))月"
    }

    private static func monthNumber(_ month: String) -> Int? {
        guard month.count >= 7 else { return nil }
        let start = month.index(month.startIndex, offsetBy: 5)
        let end = month.index(start, offsetBy: 2)
        return Int(month[start..<end])
    }
}
