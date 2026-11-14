import CoreGraphics
import Foundation

struct SharePosterBadge: Equatable {
    let text: String
    let tier: String
}

struct SharePosterCard: Equatable {
    let label: String
    let value: String
    let desc: String?
    let tone: String?
}

/// 海报只实现标准模板所需的字段；会员模板依赖远程头像与头像框，本轮不迁移。
struct SharePosterPayload: Equatable {
    let gameTypeName: String
    let resultText: String
    let rankChangeLabel: String
    let myName: String
    let opponentName: String
    let myScore: Int
    let opponentScore: Int
    let matchTime: String
    let highlights: [SharePosterCard]
    let stats: [SharePosterCard]
    let opponentBadge: SharePosterBadge?
    let qrURL: String
    let qrCaption: String

    var statsTop: CGFloat { ShareResultRules.statsTop(highlightCount: highlights.count) }
    var height: CGFloat { ShareResultRules.posterHeight(highlightCount: highlights.count, statCount: stats.count) }
}

/// 战绩海报的布局与文案，逐条对应 `uniapp/utils/posterGenerator.js` 的 `drawMatchPoster` 标准模板。
enum ShareResultRules {
    static let headerTitle = "追分 · 对局战报"
    static let defaultResult = "已完赛"
    static let practiceLabel = "练习赛"
    static let myScoreLabel = "我的得分"
    static let opponentScoreLabel = "对手得分"
    static let summaryNote = "总结口径与战报页一致"
    static let highlightsTitle = "本场亮点"
    static let statsTitle = "本场数据"
    static let statsSubtitle = "海报展示项与对局总结页保持同一统计口径"
    static let emptyHighlights = "当前模式暂无可展示亮点，但战绩已完成记录"
    static let qrHint = "微信扫一扫进入小程序"
    static let brand = "追分"
    static let footnote = "真实对局数据生成，仅供复盘与分享"
    static let matchQrCaption = "微信扫码查看这场比赛"
    static let homeQrCaption = "微信扫码打开追分"

    static func highlightsSubtitle(gameTypeName: String) -> String {
        "按\(gameTypeName)模式提炼这场最值得记住的表现"
    }

    static func matchTimeText(_ matchTime: String) -> String {
        "对局时间 \(matchTime.isEmpty ? "--" : matchTime)"
    }

    static func rankChangeLabel(matchMode: String?, rankChange: Int) -> String {
        if matchMode == "practice" { return practiceLabel }
        if rankChange > 0 { return "排位 +\(rankChange)" }
        if rankChange < 0 { return "排位 \(rankChange)" }
        return "排位 ±0"
    }

    /// 文案里拼数字前先转字符串，避免系统区域给 Int 加千分位。
    static func scoreText(_ score: Int) -> String {
        String(score)
    }

    static func displayName(_ name: String, fallback: String) -> String {
        let value = name.isEmpty ? fallback : name
        return value.count > 7 ? "\(value.prefix(6))…" : value
    }

    /// 公开对局指向对局入口，私密或缺失（按私密处理）指向首页。
    static func resolvePosterQrTarget(visibility: String?, matchId: Int64) -> (url: String, caption: String) {
        guard visibility == "public", matchId > 0 else {
            return (EntryLink.buildLink(.home), homeQrCaption)
        }
        return (EntryLink.buildLink(.match(matchId: matchId)), matchQrCaption)
    }

    static func payload(from data: ShareResultData) -> SharePosterPayload {
        let gameTypeName = data.gameTypeName.isEmpty
            ? GameTypeCatalog.label(for: data.gameType)
            : data.gameTypeName
        let opponent = MemberIdentity.resolve(
            vipLevel: data.opponentVipLevel ?? 0,
            rankLevel: data.opponentRankLevel ?? 0
        )
        let qr = resolvePosterQrTarget(visibility: data.visibility, matchId: data.matchId)
        return SharePosterPayload(
            gameTypeName: gameTypeName,
            resultText: data.result.isEmpty ? defaultResult : data.result,
            rankChangeLabel: rankChangeLabel(matchMode: data.matchMode, rankChange: data.rankChange),
            myName: displayName(data.myName, fallback: "我"),
            opponentName: displayName(data.opponentName, fallback: "对手"),
            myScore: data.myScore,
            opponentScore: data.opponentScore,
            matchTime: data.matchTime,
            highlights: cards(from: data.summaryHighlights, limit: 4),
            stats: cards(from: data.summaryStats, limit: 6),
            opponentBadge: opponent.showBadge
                ? SharePosterBadge(text: opponent.badgeText, tier: opponent.tier)
                : nil,
            qrURL: qr.url,
            qrCaption: qr.caption
        )
    }

    /// 数据读取或解码失败时按小程序走降级海报，不把失败页展示给用户。
    static func fallbackPayload(date: Date = Date()) -> SharePosterPayload {
        let formatter = DateFormatter()
        formatter.locale = Locale(identifier: "zh_CN")
        formatter.dateFormat = "yyyy/M/d"
        return SharePosterPayload(
            gameTypeName: "中式八球",
            resultText: defaultResult,
            rankChangeLabel: rankChangeLabel(matchMode: nil, rankChange: 0),
            myName: "我",
            opponentName: "对手",
            myScore: 0,
            opponentScore: 0,
            matchTime: formatter.string(from: date),
            highlights: [],
            stats: [],
            opponentBadge: nil,
            qrURL: EntryLink.buildLink(.home),
            qrCaption: homeQrCaption
        )
    }

    static func statsTop(highlightCount: Int) -> CGFloat {
        let rows = max((highlightCount + 1) / 2, 1)
        return 572 + 56 + CGFloat(rows * 170) + 20
    }

    static func posterHeight(highlightCount: Int, statCount: Int) -> CGFloat {
        let statRows = (statCount + 1) / 2
        let statsEnd = statsTop(highlightCount: highlightCount) + 56 + CGFloat(statRows * 170)
        return max(1320, statsEnd + 240)
    }

    private static func cards(from items: [MatchSummaryItem]?, limit: Int) -> [SharePosterCard] {
        Array((items ?? []).prefix(limit)).map { item in
            SharePosterCard(
                label: item.label.isEmpty ? "--" : item.label,
                value: item.value.isEmpty ? "--" : item.value,
                desc: item.desc.flatMap { $0.isEmpty ? nil : $0 },
                tone: item.tone
            )
        }
    }
}
