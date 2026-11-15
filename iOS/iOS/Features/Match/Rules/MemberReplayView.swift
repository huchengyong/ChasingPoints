import Foundation

struct ReplaySummaryRow: Equatable {
    let label: String
    let mine: String
    let opponent: String
}

struct ReplayTag: Equatable {
    let key: String
    let label: String
}

struct ReplayDetailRow: Equatable {
    let label: String
    let mine: String
    let opponent: String
}

struct ReplayTrendColumn: Equatable {
    let step: Int
    let diff: Int
    /// up / down / flat。
    let direction: String
    let heightPercent: Double
}

struct ReplayTrend: Equatable {
    let hasData: Bool
    let maxAbs: Int
    let columns: [ReplayTrendColumn]
}

struct ReplayFrameView: Equatable {
    let frameNo: Int
    let title: String
    /// me / opponent / none。
    let winner: String
    let winnerText: String
    let scoreText: String
    let winTypeText: String
    let durationText: String
    let deficitText: String
    let tags: [ReplayTag]
    let isKey: Bool
    let details: [ReplayDetailRow]
    let trend: ReplayTrend
}

struct ReplaySequenceItem: Equatable {
    let frameNo: Int
    /// me / opponent / none。
    let winner: String
}

/// 复盘页展示模型；locked 时只有局数与功能清单，明细字段为 nil。
struct ReplayView: Equatable {
    let gameType: Int
    let gameTypeLabel: String
    let frameCount: Int
    let wholeMatch: Bool
    let locked: Bool
    let showPurchase: Bool
    let features: [String]?
    let title: String?
    let scoreText: String?
    let summaryRows: [ReplaySummaryRow]?
    let winTypeRows: [ReplaySummaryRow]?
    let sequence: [ReplaySequenceItem]?
    let frames: [ReplayFrameView]?
    let keyFrames: [ReplayFrameView]?
    let notice: String?
    let hasDetail: Bool?
    let empty: Bool?

    init(
        gameType: Int,
        gameTypeLabel: String,
        frameCount: Int,
        wholeMatch: Bool,
        locked: Bool,
        showPurchase: Bool = false,
        features: [String]? = nil,
        title: String? = nil,
        scoreText: String? = nil,
        summaryRows: [ReplaySummaryRow]? = nil,
        winTypeRows: [ReplaySummaryRow]? = nil,
        sequence: [ReplaySequenceItem]? = nil,
        frames: [ReplayFrameView]? = nil,
        keyFrames: [ReplayFrameView]? = nil,
        notice: String? = nil,
        hasDetail: Bool? = nil,
        empty: Bool? = nil
    ) {
        self.gameType = gameType
        self.gameTypeLabel = gameTypeLabel
        self.frameCount = frameCount
        self.wholeMatch = wholeMatch
        self.locked = locked
        self.showPurchase = showPurchase
        self.features = features
        self.title = title
        self.scoreText = scoreText
        self.summaryRows = summaryRows
        self.winTypeRows = winTypeRows
        self.sequence = sequence
        self.frames = frames
        self.keyFrames = keyFrames
        self.notice = notice
        self.hasDetail = hasDetail
        self.empty = empty
    }
}

/// 逐条对应小程序 `utils/member-replay-view.js`：单场复盘页的文案与图表比例。
/// 数据来自 GET /api/member/match-replay，统一为「我 / 对手」视角。
enum MemberReplayView {
    static let gameTypeSnooker = 1
    static let gameTypeNineBall = 2
    static let gameTypeChineseEight = 3
    static let gameTypeAmericanNine = 4

    static let winTypeLabels = [
        "normal": "普胜",
        "break_clear": "炸清",
        "continue_clear": "接清",
        "small_gold": "小金",
        "big_gold": "大金",
    ]

    private static let keyTagLabels = [
        "decider": "决胜局",
        "comeback": "逆转局",
    ]

    static let maxTrendColumns = 60

    private static func isPoolGame(_ gameType: Int) -> Bool {
        gameType == gameTypeChineseEight || gameType == gameTypeAmericanNine
    }

    private static func toCount(_ value: Int?) -> Int {
        max(0, value ?? 0)
    }

    static func formatFrameDuration(_ seconds: Int?) -> String {
        let total = toCount(seconds)
        if total == 0 { return "" }
        let minutes = total / 60
        let rest = total % 60
        if minutes == 0 { return "\(rest)秒" }
        return rest == 0 ? "\(minutes)分" : "\(minutes)分\(rest)秒"
    }

    /// 以 0 为轴的走势柱：正数（我方领先）向上，负数向下。高度按本局最大绝对分差归一。
    static func buildTrendColumns(_ trend: [ReplayTrendPoint]?) -> ReplayTrend {
        let points = Array((trend ?? []).prefix(maxTrendColumns))
        let maxAbs = points.reduce(0) { max($0, abs($1.diff)) }
        return ReplayTrend(
            hasData: points.count > 1 && maxAbs > 0,
            maxAbs: maxAbs,
            columns: points.map { point in
                let diff = point.diff
                return ReplayTrendColumn(
                    step: point.step,
                    diff: diff,
                    direction: diff > 0 ? "up" : diff < 0 ? "down" : "flat",
                    heightPercent: maxAbs > 0 ? Double(abs(diff)) / Double(maxAbs) * 100 : 0
                )
            }
        )
    }

    /// 中式八球、美式九球的逐局胜负序列（我赢 / 对手赢）。
    static func buildFrameSequence(_ frames: [ReplayFrame]?) -> [ReplaySequenceItem] {
        (frames ?? []).map {
            ReplaySequenceItem(frameNo: toCount($0.frameNo), winner: frameWinner($0.winner))
        }
    }

    static func resolveReplayView(_ response: MatchReplayResponse, canShowPurchase: Bool = false) -> ReplayView {
        let gameType = toCount(response.gameType)
        let frameCount = toCount(response.frameCount)
        let wholeMatch = gameType == gameTypeNineBall
        let label = GameTypeCatalog.label(for: gameType)

        if response.locked {
            return ReplayView(
                gameType: gameType,
                gameTypeLabel: label,
                frameCount: frameCount,
                wholeMatch: wholeMatch,
                locked: true,
                showPurchase: canShowPurchase,
                features: wholeMatch
                    ? [
                        "整场走势：谁领先、什么时候被追上",
                        "双方普胜、小金、大金次数对照",
                        "犯规次数与送分",
                        "最大连续得分",
                    ]
                    : [
                        "每局走势：谁领先、什么时候被追上",
                        "单杆与连续得分、犯规统计",
                        "逐局胜负序列与胜局方式",
                        "决胜局与逆转局标注",
                    ]
            )
        }

        if wholeMatch {
            return resolveChasingReplayView(response, label: label)
        }

        let frames = response.frames ?? []
        var winTypeLabels: [String: String] = [:]
        for item in response.summary?.winTypeStats ?? [] where !item.winType.isEmpty && !item.label.isEmpty {
            winTypeLabels[item.winType] = item.label
        }
        let frameViews = frames.map { frameView($0, gameType: gameType, winTypeLabels: winTypeLabels) }
        let truncated = response.detailTruncated == true
        let frameOnly = response.frameOnly == true
        return ReplayView(
            gameType: gameType,
            gameTypeLabel: label,
            frameCount: frameCount,
            wholeMatch: false,
            locked: false,
            title: "\(label)复盘",
            scoreText: "\(toCount(response.summary?.myFrameWins)) : \(toCount(response.summary?.opponentFrameWins))",
            summaryRows: buildSummaryRows(response.summary, gameType: gameType),
            winTypeRows: (response.summary?.winTypeStats ?? [])
                .map { ReplaySummaryRow(label: $0.label, mine: String(toCount($0.myCount)), opponent: String(toCount($0.opponentCount))) }
                .filter { !$0.label.isEmpty },
            sequence: isPoolGame(gameType) ? buildFrameSequence(frames) : [],
            frames: frameViews,
            keyFrames: frameViews.filter(\.isKey),
            notice: truncated
                ? "这场对局的记录太长，只显示每局结果。"
                : frameOnly
                    ? "这场对局没有逐杆记录，只显示每局结果。"
                    : "",
            hasDetail: !truncated && !frameOnly,
            empty: frameViews.isEmpty
        )
    }

    /// 九球追分整场口径：单片段代表整场，得分构成来自 score_breakdown。
    private static func resolveChasingReplayView(_ response: MatchReplayResponse, label: String) -> ReplayView {
        let frame = response.frames?.first
        let myScore = toCount(frame?.myScore)
        let opponentScore = toCount(frame?.opponentScore)
        let scoreView = ChasingScore.resolveView(
            response.summary?.scoreBreakdown,
            swap: toCount(response.viewerActor) == 2
        )
        var summaryRows = scoreView?.rows.map {
            ReplaySummaryRow(label: $0.label, mine: $0.left, opponent: $0.right)
        } ?? [ReplaySummaryRow(label: "总分", mine: String(myScore), opponent: String(opponentScore))]
        let myRun = toCount(response.summary?.myMaxRun)
        let opponentRun = toCount(response.summary?.opponentMaxRun)
        if myRun > 0 || opponentRun > 0 {
            summaryRows.append(ReplaySummaryRow(label: "最大连续得分", mine: String(myRun), opponent: String(opponentRun)))
        }

        var frames: [ReplayFrameView] = []
        if let frame {
            let base = frameView(frame, gameType: gameTypeNineBall, winTypeLabels: [:])
            frames = [ReplayFrameView(
                frameNo: base.frameNo,
                title: "整场走势",
                winner: base.winner,
                winnerText: base.winnerText,
                scoreText: base.scoreText,
                // 整场只有一个片段：胜局方式与连续得分已在上方整场数据里。
                winTypeText: "",
                durationText: base.durationText,
                deficitText: base.deficitText,
                tags: [],
                isKey: false,
                details: [],
                trend: base.trend
            )]
        }

        let truncated = response.detailTruncated == true
        let frameOnly = response.frameOnly == true
        return ReplayView(
            gameType: gameTypeNineBall,
            gameTypeLabel: label,
            frameCount: toCount(response.frameCount),
            wholeMatch: true,
            locked: false,
            title: "\(label)复盘",
            scoreText: "\(myScore) : \(opponentScore)",
            summaryRows: summaryRows,
            winTypeRows: [],
            sequence: [],
            frames: frames,
            keyFrames: [],
            notice: truncated
                ? "这场对局的记录太长，只显示总分。"
                : frameOnly
                    ? "这场对局的记分记录不完整，只显示总分。"
                    : "",
            hasDetail: !truncated && !frameOnly,
            empty: frames.isEmpty
        )
    }

    private static func frameView(
        _ frame: ReplayFrame,
        gameType: Int,
        winTypeLabels: [String: String]
    ) -> ReplayFrameView {
        let winner = frameWinner(frame.winner)
        let tags = (frame.keyTags ?? []).compactMap { tag -> ReplayTag? in
            let label = keyTagLabels[tag] ?? ""
            return label.isEmpty ? nil : ReplayTag(key: tag, label: label)
        }
        let pool = isPoolGame(gameType)
        var details: [ReplayDetailRow] = []
        let myRun = toCount(frame.myMaxRun)
        let opponentRun = toCount(frame.opponentMaxRun)
        if myRun > 0 || opponentRun > 0 {
            details.append(ReplayDetailRow(
                label: gameType == gameTypeSnooker ? "单杆最高" : "最大连续得分",
                mine: String(myRun),
                opponent: String(opponentRun)
            ))
        }
        let myFouls = toCount(frame.myFoulCount)
        let opponentFouls = toCount(frame.opponentFoulCount)
        if gameType == gameTypeSnooker && (myFouls > 0 || opponentFouls > 0) {
            details.append(ReplayDetailRow(
                label: "犯规（次 / 罚分）",
                mine: "\(myFouls) / \(toCount(frame.myFoulPoints))",
                opponent: "\(opponentFouls) / \(toCount(frame.opponentFoulPoints))"
            ))
        }
        let maxDeficit = toCount(frame.maxDeficit)
        let frameNo = toCount(frame.frameNo)
        let winType = frame.winType ?? ""
        return ReplayFrameView(
            frameNo: frameNo,
            title: "第 \(frameNo) 局",
            winner: winner,
            winnerText: winner == "me" ? "我赢" : winner == "opponent" ? "对手赢" : "未分胜负",
            // 中式八球、美式九球每局只记胜负，没有比分可言。
            scoreText: pool ? "" : "\(toCount(frame.myScore)) : \(toCount(frame.opponentScore))",
            winTypeText: winType.isEmpty
                ? ""
                : (winTypeLabels[winType] ?? MemberReplayView.winTypeLabels[winType] ?? ""),
            durationText: formatFrameDuration(frame.durationSeconds),
            deficitText: maxDeficit > 0 && winner != "none"
                ? "\(winner == "me" ? "我" : "对手")曾落后 \(maxDeficit) 分后赢下"
                : "",
            tags: tags,
            isKey: !tags.isEmpty,
            details: details,
            trend: buildTrendColumns(frame.trend)
        )
    }

    private static func buildSummaryRows(_ summary: ReplaySummary?, gameType: Int) -> [ReplaySummaryRow] {
        guard let summary else { return [] }
        var rows = [ReplaySummaryRow(
            label: "赢下局数",
            mine: String(toCount(summary.myFrameWins)),
            opponent: String(toCount(summary.opponentFrameWins))
        )]
        let myRun = toCount(summary.myMaxRun)
        let opponentRun = toCount(summary.opponentMaxRun)
        if myRun > 0 || opponentRun > 0 {
            rows.append(ReplaySummaryRow(
                label: gameType == gameTypeSnooker ? "单杆最高" : "最大连续得分",
                mine: String(myRun),
                opponent: String(opponentRun)
            ))
        }
        let myStreak = toCount(summary.myLongestStreak)
        let opponentStreak = toCount(summary.opponentLongestStreak)
        if isPoolGame(gameType) && (myStreak > 0 || opponentStreak > 0) {
            rows.append(ReplaySummaryRow(
                label: "最长连胜（局）",
                mine: String(myStreak),
                opponent: String(opponentStreak)
            ))
        }
        return rows
    }

    private static func frameWinner(_ winner: Int) -> String {
        winner == 1 ? "me" : winner == 2 ? "opponent" : "none"
    }
}
