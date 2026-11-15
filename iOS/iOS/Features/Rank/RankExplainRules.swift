import Foundation

// 段位列表项，对应 `/api/public/rank/configs` 的段位配置。
struct RankItem: Decodable, Equatable, Identifiable {
    let level: Int
    let name: String
    let icon: String
    let minScore: Int

    var id: Int { level }
}

enum RankGrowthState: String, Equatable {
    case current
    case reached
    case pending
}

struct RankGrowthStep: Equatable, Identifiable {
    let level: Int
    let name: String
    let icon: String
    let minScore: Int
    let rangeText: String
    let state: RankGrowthState
    let stateText: String

    var id: Int { level }
}

struct RankHeroModel: Equatable {
    let level: Int
    let name: String
    let icon: String
    let rankScore: Int
    let placementRemaining: Int
    let isMax: Bool
    let progressPercent: Int
    let nextName: String?
    let nextScore: Int?
    let earned: Int?
    let span: Int?
    let remaining: Int?
}

struct RankRuleStat: Equatable {
    let value: String
    let label: String
}

struct RankRuleRow: Equatable {
    let title: String
    let description: String
}

struct RankRuleExample: Equatable {
    let title: String
    let formula: String
}

struct RankRules: Equatable {
    let version: Int
    let summaryStats: [RankRuleStat]
    let note: String
    let detailRows: [RankRuleRow]
    let example: RankRuleExample
    let notice: String

    func withNotice(_ notice: String) -> RankRules {
        RankRules(
            version: version,
            summaryStats: summaryStats,
            note: note,
            detailRows: detailRows,
            example: example,
            notice: notice
        )
    }
}

struct RankRulesState: Equatable {
    let active: Bool
    let scheduled: Bool
    let effectiveDateText: String
}

/// 段位说明的纯规则，逐条迁移自 `uniapp/utils/rank-explain.js`。
enum RankExplainRules {
    static let maxRankLevel = 6

    private static let rulesVersionLegacy = 1
    private static let rulesVersionV2 = 2

    /// 旧规则：只在排位规则 V2 生效前展示。
    static let legacyRules = RankRules(
        version: rulesVersionLegacy,
        summaryStats: [
            RankRuleStat(value: "+8～40", label: "胜方基础分"),
            RankRuleStat(value: "约一半", label: "败方基础扣分"),
            RankRuleStat(value: "+500", label: "每日上涨上限"),
        ],
        note: "实际结算还会受到完成局数、同一对手场次、对手剩余排位分和会员特殊战绩影响。",
        detailRows: [
            RankRuleRow(title: "完成局数", description: "胜方基础分随完成局数递增，最低 +8，最高 +40。"),
            RankRuleRow(title: "败方扣分", description: "败方扣胜方基础分的一半；排位分不会低于 0。"),
            RankRuleRow(title: "同一对手", description: "当天同一对手前 2 场按 100% 结算，第 3 场按 80%，第 4～6 场按 30%（只缩减胜方加分），第 7 场起不计排位分和胜率。"),
            RankRuleRow(title: "对手剩余分", description: "对手剩余排位分不足时，胜方实际基础分会相应降低，最低 +2。"),
            RankRuleRow(title: "会员特殊战绩", description: "会员赢球时，特殊战绩按会员等级倍率计入排位分，单日最多 200 分；输球不计，普通用户只记录不计分。"),
            RankRuleRow(title: "每日封顶", description: "每个球种每天最多上涨 500 排位分。"),
        ],
        example: RankRuleExample(
            title: "算分示例 · 5 局制胜利",
            formula: "基础分 +20；若这是当天与同一对手的第 3 场，则按 80% 结算为 +16。"
        ),
        notice: ""
    )

    /// 排位规则 V2：示例分值来自服务端同一公式。
    static let ratingV2Rules = RankRules(
        version: rulesVersionV2,
        summaryStats: [
            RankRuleStat(value: "±45", label: "抢 5 同分胜负"),
            RankRuleStat(value: "+68", label: "赢高 500 分的对手"),
            RankRuleStat(value: "10 场", label: "定级赛涨跌加倍"),
        ],
        note: "排位分只看胜负和对手实力：赢比你强的人加得多，输给比你强的人扣得少，赛制越长分值越大。会员身份和特殊战绩都不影响排位分。",
        detailRows: [
            RankRuleRow(title: "赛前胜率", description: "按双方排位分算出赛前胜率：分差 500 分时，高分方单局胜率约 61%，抢 5 约 76%。结果越出乎意料，分数变化越大。"),
            RankRuleRow(title: "赛制分值", description: "同分对手胜负：抢 1 为 ±20，抢 3 为 ±35，抢 5 为 ±45，抢 9 为 ±60，更长赛制按抢 9 计。5:0 与 5:4 分值相同；自由局按赢局较多一方的局数计。"),
            RankRuleRow(title: "定级赛", description: "每个球种前 10 场为定级赛，涨跌加倍；已定级的球手遇到定级中的对手，涨跌减半。"),
            RankRuleRow(title: "同一对手", description: "当天同一对手前 2 场按 100% 结算，第 3 场按 80%，第 4～6 场按 30%，双方同比例；第 7 场起不计排位分和胜率。"),
            RankRuleRow(title: "每日封顶", description: "每个球种每天最多上涨 500 排位分；排位分不会低于 0。"),
            RankRuleRow(title: "高光不计分", description: "炸清、接清、单杆等高光照常记录在战报和成就里，但不改变排位分，会员与普通用户规则相同。"),
            RankRuleRow(title: "赛季排名", description: "赛季内该球种打满 5 场、遇到至少 3 位不同对手才参与赛季排名与称号；排位分跨赛季延续，不做重置。"),
        ],
        example: RankRuleExample(
            title: "算分示例 · 抢 5",
            formula: "两人同分时胜方 +45、负方 −45；你比对手低 500 分并获胜得 +68；输给比你高 500 分的对手只扣 21。"
        ),
        notice: ""
    )

    /// 成长路径：按王者到青铜的顺序返回六级段位，每一级带分数区间与状态。
    static func growthPath(rankList: [RankItem], currentLevel: Int, rankScore: Int) -> [RankGrowthStep] {
        let ascending = normalized(rankList)
        let steps = ascending.enumerated().map { index, item in
            let rangeText: String
            if index + 1 < ascending.count {
                let upper = ascending[index + 1]
                rangeText = "\(item.minScore)–\(upper.minScore - 1) 分"
            } else {
                rangeText = "\(item.minScore) 分以上"
            }
            let state: RankGrowthState
            if item.level == currentLevel {
                state = .current
            } else if rankScore >= item.minScore {
                state = .reached
            } else {
                state = .pending
            }
            return RankGrowthStep(
                level: item.level,
                name: item.name,
                icon: item.icon,
                minScore: item.minScore,
                rangeText: rangeText,
                state: state,
                stateText: stateText(for: state)
            )
        }
        return Array(steps.reversed())
    }

    /// 英雄卡展示模型：等级 1–5 计算级内进度，等级 6 返回满级状态。
    static func heroModel(rankInfo: RankInfo?, rankList: [RankItem]) -> RankHeroModel? {
        guard let rankInfo else { return nil }

        let level = rankInfo.level
        let isMax = level >= maxRankLevel
        let placementRemaining = max(0, rankInfo.placementRemaining ?? 0)
        if isMax {
            return RankHeroModel(
                level: level,
                name: rankInfo.name,
                icon: rankInfo.icon,
                rankScore: rankInfo.rankScore,
                placementRemaining: placementRemaining,
                isMax: true,
                progressPercent: 100,
                nextName: nil,
                nextScore: nil,
                earned: nil,
                span: nil,
                remaining: nil
            )
        }

        let currentConfig = normalized(rankList).first { $0.level == level }
        let minScore = currentConfig?.minScore ?? 0
        let nextScore = rankInfo.nextScore
        let earned = max(0, rankInfo.rankScore - minScore)
        let span = max(0, nextScore - minScore)
        let progressPercent = span > 0
            ? clampPercent(Double(earned * 100) / Double(span))
            : clampPercent(Double(rankInfo.progress))

        return RankHeroModel(
            level: level,
            name: rankInfo.name,
            icon: rankInfo.icon,
            rankScore: rankInfo.rankScore,
            placementRemaining: placementRemaining,
            isMax: false,
            progressPercent: progressPercent,
            nextName: rankInfo.nextName,
            nextScore: nextScore,
            earned: earned,
            span: span,
            remaining: max(0, nextScore - rankInfo.rankScore)
        )
    }

    /// 优先按生效时刻判断 V2 是否已生效或已排期。
    static func rulesState(rulesVersion: Int, rulesEffectiveAt: String, now: Date) -> RankRulesState {
        guard let effectiveAt = parseEffectiveAt(rulesEffectiveAt) else {
            return RankRulesState(active: rulesVersion == rulesVersionV2, scheduled: false, effectiveDateText: "")
        }
        let active = now >= effectiveAt
        return RankRulesState(
            active: active,
            scheduled: !active,
            effectiveDateText: formatEffectiveDate(effectiveAt)
        )
    }

    /// 选择要展示的计分规则与规则切换提示。
    static func rules(rulesVersion: Int, rulesEffectiveAt: String, now: Date) -> RankRules {
        let state = rulesState(rulesVersion: rulesVersion, rulesEffectiveAt: rulesEffectiveAt, now: now)
        if state.active {
            let notice = state.effectiveDateText.isEmpty
                ? ""
                : "新排位规则已于\(state.effectiveDateText)生效，此前的赛季成绩与称号保持不变。"
            return ratingV2Rules.withNotice(notice)
        }
        let notice = state.scheduled
            ? "新排位规则将于\(state.effectiveDateText)起生效：排位分改为按对手实力计算，生效时用新规则重算你的历史排位确定初始分，前 10 场为定级赛。"
            : ""
        return legacyRules.withNotice(notice)
    }

    // MARK: - 私有

    private static func normalized(_ rankList: [RankItem]) -> [RankItem] {
        rankList.filter { $0.level > 0 }.sorted { $0.level < $1.level }
    }

    private static func stateText(for state: RankGrowthState) -> String {
        switch state {
        case .current: "当前"
        case .reached: "已达成"
        case .pending: "待晋级"
        }
    }

    private static func clampPercent(_ value: Double) -> Int {
        guard value.isFinite else { return 0 }
        return max(0, min(100, Int(value.rounded())))
    }

    private static func parseEffectiveAt(_ value: String) -> Date? {
        let text = value.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !text.isEmpty else { return nil }
        let formatter = ISO8601DateFormatter()
        formatter.formatOptions = [.withInternetDateTime]
        if let date = formatter.date(from: text) { return date }
        formatter.formatOptions = [.withInternetDateTime, .withFractionalSeconds]
        return formatter.date(from: text)
    }

    /// 生效时刻按北京时间零点下发，换算成北京时间日期展示。
    private static func formatEffectiveDate(_ date: Date) -> String {
        var calendar = Calendar(identifier: .gregorian)
        calendar.timeZone = TimeZone(secondsFromGMT: 8 * 60 * 60) ?? .gmt
        let parts = calendar.dateComponents([.year, .month, .day], from: date)
        return "\(parts.year ?? 0)年\(parts.month ?? 0)月\(parts.day ?? 0)日"
    }
}
