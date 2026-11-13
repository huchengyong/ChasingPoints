import Foundation

// 迁移自 `uniapp/utils/opponent-record.js` 的纯规则。

struct OpponentRecordViewModel: Equatable {
    let isTargetMode: Bool
    let subjectName: String
    let navigationTitle: String
    let loginHint: String
    let searchPlaceholder: String
    let emptyText: String
    let emptyHint: String
}

struct OpponentStatsSummary: Equatable {
    let title: String
    let subtitle: String
}

enum OpponentTone: Equatable {
    case lead
    case trail
    case balanced
    case waiting
}

struct OpponentCard: Identifiable, Equatable {
    let opponent: OpponentRecord
    let totalMatches: Int
    let isMeOpponent: Bool
    let winRateText: String
    let recordText: String
    let sampleText: String
    let lastMatchText: String
    let relationshipText: String
    let relationshipBadge: String
    let tone: OpponentTone

    var id: Int64 { opponent.id }
}

enum OpponentRecordRules {
    /// 查询参数与 JS 一致：`keyword` 始终携带（可为空），`target_user_id` 仅在目标模式追加。
    static func requestParams(
        page: Int,
        pageSize: Int = 20,
        keyword: String,
        targetUserId: Int64
    ) -> [String: CustomStringConvertible?] {
        var params: [String: CustomStringConvertible?] = [
            "page": page,
            "page_size": pageSize,
            "keyword": keyword,
        ]
        if targetUserId > 0 {
            params["target_user_id"] = targetUserId
        }
        return params
    }

    static func viewModel(targetName: String, targetUserId: Int64) -> OpponentRecordViewModel {
        guard targetUserId > 0 else {
            return OpponentRecordViewModel(
                isTargetMode: false,
                subjectName: "你",
                navigationTitle: "过往对手",
                loginHint: "请登录后查看过往对手",
                searchPlaceholder: "搜索对手",
                emptyText: "暂无过往对手",
                emptyHint: "快去发起一场PK吧！"
            )
        }

        let resolvedName = targetName.isEmpty ? "这位球友" : targetName
        return OpponentRecordViewModel(
            isTargetMode: true,
            subjectName: resolvedName,
            navigationTitle: "对方战绩",
            loginHint: "请登录后查看对方战绩",
            searchPlaceholder: "搜索 TA 的对手",
            emptyText: "暂无对方战绩",
            emptyHint: "还没看到\(resolvedName) 的历史过往对手"
        )
    }

    static func statsSummary(
        totalOpponents: Int,
        totalWins: Int,
        targetName: String,
        targetUserId: Int64
    ) -> OpponentStatsSummary {
        let isTargetMode = targetUserId > 0
        let title = isTargetMode
            ? "\(targetName.isEmpty ? "这位球友" : targetName)的对手档案"
            : "你的对手档案"
        let subtitle = totalOpponents > 0
            ? "已经形成 \(String(totalOpponents)) 个对手样本，累计拿下 \(String(totalWins)) 场胜利"
            : "先找到最值得点进复盘的那个对手"
        return OpponentStatsSummary(title: title, subtitle: subtitle)
    }

    static func cards(
        opponents: [OpponentRecord],
        subjectName: String,
        currentUserId: Int64 = 0,
        showWinRateLabel: Bool = false
    ) -> [OpponentCard] {
        opponents.map { card(
            for: $0,
            subjectName: subjectName,
            currentUserId: currentUserId,
            showWinRateLabel: showWinRateLabel
        ) }
    }

    static func card(
        for opponent: OpponentRecord,
        subjectName: String,
        currentUserId: Int64 = 0,
        showWinRateLabel: Bool = false
    ) -> OpponentCard {
        let wins = opponent.wins
        let losses = opponent.losses
        let totalMatches = opponent.totalMatches > 0 ? opponent.totalMatches : wins + losses
        let isMeOpponent = currentUserId > 0 && opponent.id == currentUserId
        let level = H2HRules.advantageLevel(
            totalMatches: totalMatches,
            myWins: wins,
            opponentWins: losses
        )
        // JS 用 Math.round，正数下 Swift 的 rounded() 结果相同。
        let winRate = Int(opponent.winRate.rounded())

        return OpponentCard(
            opponent: opponent,
            totalMatches: totalMatches,
            isMeOpponent: isMeOpponent,
            winRateText: "\(String(winRate))%\(showWinRateLabel ? " 胜率" : "")",
            recordText: "总交锋 \(String(wins)) 胜 \(String(losses)) 负",
            sampleText: totalMatches > 0 ? "\(String(totalMatches)) 场交锋" : "等待首场交锋",
            lastMatchText: opponent.lastMatchAt.isEmpty
                ? "还没交过手"
                : "上次对局 \(AppFormat.relativeTime(opponent.lastMatchAt))",
            relationshipText: relationshipText(level: level, subjectName: subjectName),
            relationshipBadge: relationshipBadge(level: level),
            tone: tone(level: level)
        )
    }

    /// 小程序通过 `navigateTo` 拼查询串；原生直接返回结构化路由。
    /// `decodeURIComponent` 不迁移：路由传结构化值，双解码会破坏含 `%` 的名字。
    static func h2hRoute(
        opponent: OpponentRecord,
        targetUserId: Int64,
        targetName: String,
        targetAvatar: String
    ) -> AppRoute {
        let isTargetMode = targetUserId > 0
        return .h2hRecord(
            opponentId: opponent.id > 0 ? opponent.id : 0,
            targetUserId: isTargetMode ? targetUserId : 0,
            targetName: isTargetMode ? (targetName.isEmpty ? "球友" : targetName) : "",
            targetAvatar: isTargetMode ? targetAvatar : "",
            opponentName: opponent.name.isEmpty ? "对手" : opponent.name
        )
    }

    // MARK: - 关系文案

    private static func relationshipText(level: AdvantageLevel, subjectName: String) -> String {
        switch level {
        case .noSample: "\(subjectName)和他还没形成样本"
        case .balanced: "\(subjectName)和他势均力敌"
        case .slightLead: "\(subjectName)略占上风"
        case .slightTrail: "\(subjectName)暂时落后"
        case .clearLead: "\(subjectName)明显占优"
        case .clearTrail: "\(subjectName)最近被压制"
        }
    }

    private static func relationshipBadge(level: AdvantageLevel) -> String {
        switch level {
        case .noSample: "待交锋"
        case .balanced: "均势"
        case .slightLead: "领先"
        case .slightTrail: "落后"
        case .clearLead: "压制"
        case .clearTrail: "复仇"
        }
    }

    private static func tone(level: AdvantageLevel) -> OpponentTone {
        switch level {
        case .slightLead, .clearLead: .lead
        case .slightTrail, .clearTrail: .trail
        case .balanced: .balanced
        case .noSample: .waiting
        }
    }
}
