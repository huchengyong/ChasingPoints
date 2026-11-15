import Foundation

/// 小程序按本地时区取年月日；原生端固定北京时区，结果不随运行机器的时区变化。
enum SaiXunCalendar {
    static let shanghai: Calendar = {
        var calendar = Calendar(identifier: .gregorian)
        calendar.timeZone = TimeZone(identifier: "Asia/Shanghai") ?? .current
        return calendar
    }()
}

/// 封面 / 头像来源：CDN 或接口返回的 http(s) 地址转成 `URL`；
/// 小程序本地占位图保留为标记，不拼出一个不存在的 URL。
enum SaiXunImageSource: Equatable {
    case remote(URL)
    /// `/static/images/wst-player-placeholder.png`
    case officialPlayerPlaceholder
    /// `/static/images/default-avatars/ball-01.png`
    case defaultAvatar
}

struct SaiXunFlag: Equatable {
    enum Kind: String {
        case image
        case emoji
        case none
    }

    let type: Kind
    let value: String
}

enum SaiXunStatusTone: String, Equatable {
    case upcoming
    case live
    case ended
    case canceled
}

struct SaiXunCardHighlight: Equatable {
    enum Kind: String {
        case champion
        case live
        case upcoming
    }

    let type: Kind
    let label: String
    let primary: String
    let secondary: String
}

struct SaiXunCard: Equatable, Identifiable {
    let item: EventNewsInfo
    let title: String
    let originalTitle: String
    let summary: String
    let statusText: String
    let statusTone: SaiXunStatusTone
    let statusLabel: String
    let isFeatured: Bool
    let gameTypeText: String
    let dateText: String
    let timeText: String
    let showTime: Bool
    let locationText: String
    let regionText: String
    let metaText: String
    let currentRoundText: String
    let resultText: String
    let highlight: SaiXunCardHighlight?
    let matchCount: Int
    let matchCountText: String
    let sourceText: String
    let hasCover: Bool
    let coverImage: SaiXunImageSource

    var id: Int64 { item.id }
}

struct SaiXunListSection: Equatable {
    let key: String
    let groupKey: String
    let title: String
    let caption: String
    let items: [EventNewsInfo]
}

struct SaiXunHeroStats: Equatable {
    let totalCount: Int
    let liveCount: Int
    let upcomingCount: Int
}

struct SaiXunMatch: Equatable, Identifiable {
    let id: Int64
    let roundName: String
    let roundOrder: Int
    let matchOrder: Int
    let status: Int
    let statusText: String
    let isLive: Bool
    let startAt: Date?
    let startTimeText: String
    let clockText: String
    let dayKey: String
    let dayLabel: String
    let homePlayerName: String
    let homePlayerSubName: String
    let homePlayerFirstName: String
    let homePlayerLastName: String
    let homePlayerFlagEmoji: String
    let homePlayerCountryCode: String
    let homePlayerFlag: SaiXunFlag
    let homePlayerAvatar: SaiXunImageSource
    let homeResultText: String
    let awayPlayerName: String
    let awayPlayerSubName: String
    let awayPlayerFirstName: String
    let awayPlayerLastName: String
    let awayPlayerFlagEmoji: String
    let awayPlayerCountryCode: String
    let awayPlayerFlag: SaiXunFlag
    let awayPlayerAvatar: SaiXunImageSource
    let awayResultText: String
    let showScore: Bool
    let homeScore: Int
    let awayScore: Int
    let scoreText: String
    let winnerSide: Int
    let metaText: String
    let isPlaceholder: Bool
}

struct SaiXunRound: Equatable {
    let key: String
    let roundName: String
    let roundOrder: Int
    let matches: [SaiXunMatch]
}

struct SaiXunScheduleTab: Equatable {
    let key: String
    let label: String
    let matches: [SaiXunMatch]
}

struct SaiXunScheduleDay: Equatable {
    let key: String
    let label: String
    let matches: [SaiXunMatch]
}

struct SaiXunChampion: Equatable {
    let name: String
    let subName: String
    let avatar: SaiXunImageSource
    let flag: SaiXunFlag
    let opponentName: String
    let scoreText: String
}

/// 赛讯展示规则，逐条对应小程序 `utils/saixun.js`。
enum SaiXunRules {
    /// 小程序主包里的默认封面已迁到 CDN，与 `DEFAULT_EVENT_COVER` 一致。
    static let defaultEventCover = CDNAsset.url(for: "/static/images/default-event-cover.png")!
    /// 官方（WST）赛事没有专属封面时（常见于资格赛），沿用 WST 官网自己的默认封面。
    static let officialDefaultEventCover = CDNAsset.url(for: "/static/images/wst-default-event-cover.png")!
    static let defaultPlayerAvatar: SaiXunImageSource = .defaultAvatar
    /// 官方赛讯球员无头像时的中性占位图（WST 官方风格剪影），不用平台自定义台球头像。
    static let officialPlayerPlaceholder: SaiXunImageSource = .officialPlayerPlaceholder

    static let schedulePageSize = 30

    private static let matchLiveFallbackWindow: TimeInterval = 12 * 60 * 60
    private static let defaultStatusFallback = "赛讯更新中"

    // MARK: - 列表卡片

    static func normalizeSaiXunCard(
        _ item: EventNewsInfo,
        now: Date,
        calendar: Calendar = SaiXunCalendar.shanghai
    ) -> SaiXunCard {
        let status = item.status
        let currentRoundText = standardizeRoundText(item.currentRoundText)
        let matchCount = item.matchCount

        let timeText = formatEventTimeRange(item.startTime, item.endTime, now: now, calendar: calendar)
        let rawTitle: String
        if !item.tournamentName.isEmpty {
            rawTitle = item.tournamentName
        } else if !item.title.isEmpty {
            rawTitle = item.title
        } else {
            rawTitle = "赛事情报"
        }
        let localizedTitle = localizeTournamentTitle(rawTitle)
        let title = localizedTitle.isEmpty ? rawTitle : localizedTitle
        let dateText = formatEventDateRange(item.startDate, item.endDate, calendar: calendar)
        let regionText = buildEventRegionText(item)

        let coverURL = resolveEventCover(item)
        let hasCover = !SaiXunText.trimmed(item.coverImage).isEmpty || item.sourceType == "official"
        let coverImage = SaiXunImageSource.remote(coverURL ?? defaultEventCover)

        let statusText = eventNewsStatusText(status, fallback: defaultStatusFallback)
        let summary: String
        if !item.summary.isEmpty {
            summary = item.summary
        } else if !item.latestResultText.isEmpty {
            summary = item.latestResultText
        } else {
            summary = "官方赛讯持续更新中"
        }

        return SaiXunCard(
            item: item,
            title: title,
            originalTitle: title == rawTitle ? "" : rawTitle,
            summary: summary,
            statusText: statusText,
            statusTone: statusTone(status),
            statusLabel: status == 0
                ? formatEventCountdownText(item.startDate, now: now, calendar: calendar)
                : statusText,
            isFeatured: status == 0 || status == 1,
            gameTypeText: GameTypeCatalog.label(for: item.gameType, fallback: "台球"),
            dateText: dateText,
            timeText: timeText,
            showTime: !timeText.isEmpty,
            locationText: buildEventLocationText(item),
            regionText: regionText,
            metaText: [dateText, regionText].filter { !$0.isEmpty }.joined(separator: " · "),
            currentRoundText: currentRoundText.isEmpty ? "轮次待更新" : currentRoundText,
            resultText: item.latestResultText,
            highlight: buildCardHighlight(
                status: status,
                roundText: currentRoundText,
                latestResultText: item.latestResultText
            ),
            matchCount: matchCount,
            matchCountText: matchCount > 0 ? "\(matchCount) 场比赛" : "比赛待更新",
            sourceText: SaiXunText.trimmed(item.sourceName),
            hasCover: hasCover,
            coverImage: coverImage
        )
    }

    /// 列表按后端顺序（进行中 → 即将开始 → 已结束按时间倒序）切分分组，只合并相邻同组项。
    static func buildSaiXunListSections(
        _ list: [EventNewsInfo],
        calendar: Calendar = SaiXunCalendar.shanghai
    ) -> [SaiXunListSection] {
        var sections: [SaiXunListSection] = []
        for item in list {
            let section = resolveListSection(item, calendar: calendar)
            if let last = sections.last, last.groupKey == section.key {
                sections[sections.count - 1] = SaiXunListSection(
                    key: last.key,
                    groupKey: last.groupKey,
                    title: last.title,
                    caption: last.caption,
                    items: last.items + [item]
                )
                continue
            }
            sections.append(
                SaiXunListSection(
                    key: "\(section.key)-\(sections.count)",
                    groupKey: section.key,
                    title: section.title,
                    caption: section.caption,
                    items: [item]
                )
            )
        }
        return sections
    }

    static func buildSaiXunHeroStats(_ list: [EventNewsInfo]) -> SaiXunHeroStats {
        SaiXunHeroStats(
            totalCount: list.count,
            liveCount: list.filter { $0.status == 1 }.count,
            upcomingCount: list.filter { $0.status == 0 }.count
        )
    }

    // MARK: - 赛事详情

    /// 按轮次先后（含 round_order 相同的保留赛）返回全部轮次；全部为占位对阵的轮次没有可看信息，不展示。
    static func buildSaiXunDetailRounds(
        _ matches: [EventNewsMatchInfo],
        now: Date,
        calendar: Calendar = SaiXunCalendar.shanghai
    ) -> [SaiXunRound] {
        var builders: [(key: String, roundName: String, roundOrder: Int, matches: [SaiXunMatch])] = []

        for item in matches {
            let normalized = normalizeMatch(item, now: now, calendar: calendar)
            let key = "\(normalized.roundOrder)-\(normalized.roundName)"
            if let index = builders.firstIndex(where: { $0.key == key }) {
                builders[index].matches.append(normalized)
            } else {
                builders.append((key, normalized.roundName, normalized.roundOrder, [normalized]))
            }
        }

        let rounds = builders.compactMap { builder -> SaiXunRound? in
            guard builder.matches.contains(where: { !$0.isPlaceholder }) else { return nil }
            return SaiXunRound(
                key: builder.key,
                roundName: builder.roundName,
                roundOrder: builder.roundOrder,
                matches: builder.matches.sorted(by: compareMatches)
            )
        }
        return rounds.sorted(by: compareRounds)
    }

    /// 赛程按状态分两栏，每场自带轮次名：进行中的场次还没打完，与未开始的一起放在「即将开始」并按时间正序；
    /// 「已结束」按时间倒序，最近结束的在前。
    static func buildSaiXunScheduleTabs(_ rounds: [SaiXunRound]) -> [SaiXunScheduleTab] {
        let matches = rounds.flatMap(\.matches)
        return [
            SaiXunScheduleTab(
                key: "ended",
                label: "已结束",
                matches: matches.filter(isEndedMatch).sorted(by: compareEndedMatches)
            ),
            SaiXunScheduleTab(
                key: "upcoming",
                label: "即将开始",
                matches: matches.filter { !isEndedMatch($0) }.sorted(by: compareMatches)
            ),
        ]
    }

    /// 尊重用户手动选择；没选过时有待进行的场次就看「即将开始」，否则看「已结束」。
    static func resolveSaiXunScheduleTabKey(
        _ tabs: [SaiXunScheduleTab],
        selectedKey: String = ""
    ) -> String {
        if tabs.contains(where: { $0.key == selectedKey }) { return selectedKey }
        let upcomingTab = tabs.first { $0.key == "upcoming" }
        return upcomingTab?.matches.isEmpty == false ? "upcoming" : "ended"
    }

    /// 大型赛事一栏可达数百场，按 limit 分批渲染，避免一次渲染过大。
    static func buildSaiXunScheduleDays(
        _ matches: [SaiXunMatch],
        limit: Int = schedulePageSize
    ) -> [SaiXunScheduleDay] {
        var days: [SaiXunScheduleDay] = []
        for match in matches.prefix(max(0, limit)) {
            if let last = days.last, last.key == match.dayKey {
                days[days.count - 1] = SaiXunScheduleDay(
                    key: last.key,
                    label: last.label,
                    matches: last.matches + [match]
                )
                continue
            }
            days.append(SaiXunScheduleDay(key: match.dayKey, label: match.dayLabel, matches: [match]))
        }
        return days
    }

    static func resolveSaiXunChampion(_ rounds: [SaiXunRound]) -> SaiXunChampion? {
        let finalRound = rounds.first { $0.roundName == "决赛" }
        let finalMatch = finalRound?.matches.first {
            $0.status == 2 && ($0.winnerSide == 1 || $0.winnerSide == 2)
        }
        guard let finalMatch else { return nil }

        let homeWon = finalMatch.winnerSide == 1
        return SaiXunChampion(
            name: homeWon ? finalMatch.homePlayerName : finalMatch.awayPlayerName,
            subName: homeWon ? finalMatch.homePlayerSubName : finalMatch.awayPlayerSubName,
            avatar: homeWon ? finalMatch.homePlayerAvatar : finalMatch.awayPlayerAvatar,
            flag: homeWon ? finalMatch.homePlayerFlag : finalMatch.awayPlayerFlag,
            opponentName: homeWon ? finalMatch.awayPlayerName : finalMatch.homePlayerName,
            scoreText: "\(max(finalMatch.homeScore, finalMatch.awayScore))-\(min(finalMatch.homeScore, finalMatch.awayScore))"
        )
    }

    // MARK: - 时间与地点

    static func formatEventDateRange(
        _ startDate: String?,
        _ endDate: String?,
        calendar: Calendar = SaiXunCalendar.shanghai
    ) -> String {
        let start = parseDateOnly(startDate)
        let end = parseDateOnly(endDate ?? startDate)

        switch (start, end) {
        case (nil, nil):
            return "日期待定"
        case let (start?, end?) where start == end:
            return formatMonthDay(start, calendar: calendar)
        case let (nil, end?):
            return formatMonthDay(end, calendar: calendar)
        case let (start?, nil):
            return formatMonthDay(start, calendar: calendar)
        case let (start?, end?):
            return "\(formatMonthDay(start, calendar: calendar)) - \(formatMonthDay(end, calendar: calendar))"
        }
    }

    static func formatEventTimeRange(
        _ startTime: String?,
        _ endTime: String?,
        now: Date,
        calendar: Calendar = SaiXunCalendar.shanghai
    ) -> String {
        let start = parseDateTime(startTime)
        let end = parseDateTime(endTime)

        if start == nil && end == nil { return "" }
        if let start, let end {
            return "\(formatEventNewsTime(start, now: now, calendar: calendar)) - \(formatEventNewsTime(end, now: now, calendar: calendar))"
        }
        if let start { return formatEventNewsTime(start, now: now, calendar: calendar) }
        return formatEventNewsTime(end!, now: now, calendar: calendar)
    }

    static func formatEventCountdownText(
        _ startDate: String?,
        now: Date,
        calendar: Calendar = SaiXunCalendar.shanghai
    ) -> String {
        guard let start = parseDateOnly(startDate) else { return "即将开始" }

        let diffDays = diffLocalDays(start, now: now, calendar: calendar)
        if diffDays < 0 { return "即将开始" }
        if diffDays == 0 { return "今天开赛" }
        if diffDays == 1 { return "明天开赛" }
        if diffDays <= 30 { return "\(diffDays)天后开赛" }
        return "\(formatMonthDay(start, calendar: calendar))开赛"
    }

    static func buildEventLocationText(_ item: EventNewsInfo) -> String {
        let city = SaiXunText.trimmed(item.city)
        let venue = SaiXunText.trimmed(item.venue)
        if !city.isEmpty && !venue.isEmpty { return "\(city) · \(venue)" }
        return city.isEmpty ? venue : city
    }

    static func buildEventRegionText(_ item: EventNewsInfo) -> String {
        var parts: [String] = []
        for value in [localizeCountryName(item.country), item.city, item.venue] {
            let text = SaiXunText.trimmed(value)
            guard !text.isEmpty, !parts.contains(text) else { continue }
            parts.append(text)
        }
        return parts.joined(separator: " · ")
    }

    static func formatBestOfText(_ bestOf: Int?) -> String {
        let frames = bestOf ?? 0
        guard frames > 0 else { return "" }
        if frames % 2 == 0 { return "共\(frames)局" }
        return "\(frames)局\((frames + 1) / 2)胜"
    }

    // MARK: - 私有规则

    private struct MatchSummary {
        let home: String
        let away: String
        let homeScore: Int?
        let awayScore: Int?

        var hasScore: Bool { homeScore != nil && awayScore != nil }
    }

    private static func eventNewsStatusText(_ status: Int, fallback: String) -> String {
        switch status {
        case 0: "即将开始"
        case 1: "进行中"
        case 2: "已结束"
        case 3: "已取消"
        default: fallback
        }
    }

    private static func statusTone(_ status: Int) -> SaiXunStatusTone {
        SaiXunTables.statusTones[status].flatMap(SaiXunStatusTone.init(rawValue:)) ?? .ended
    }

    private static func resolveEventCover(_ item: EventNewsInfo) -> URL? {
        let cover = SaiXunText.trimmed(item.coverImage)
        if !cover.isEmpty { return CDNAsset.url(for: cover) }
        return item.sourceType == "official" ? officialDefaultEventCover : nil
    }

    private static func resolveListSection(
        _ item: EventNewsInfo,
        calendar: Calendar
    ) -> (key: String, title: String, caption: String) {
        switch item.status {
        case 1: return ("live", "进行中", "")
        case 0: return ("upcoming", "即将开始", "")
        case 3: return ("canceled", "已取消", "")
        default: break
        }

        guard let start = parseDateOnly(item.startDate) else {
            return ("ended", "已结束", "")
        }
        let components = calendar.dateComponents([.year, .month], from: start)
        let month = components.month ?? 1
        return ("ended-\(components.year ?? 0)-\(month)", "\(month)月", "已结束")
    }

    private static func parseMatchSummaryText(_ value: String?) -> MatchSummary? {
        let text = SaiXunText.trimmed(value)

        if let groups = SaiXunText.captures(#"^(.+?)\s+(\d+)\s+-\s+(\d+)\s+(.+)$"#, text, options: []) {
            return MatchSummary(
                home: localizeMatchPlayerName(groups[1]),
                away: localizeMatchPlayerName(groups[4]),
                homeScore: Int(groups[2]),
                awayScore: Int(groups[3])
            )
        }
        if let groups = SaiXunText.captures(#"^(.+?)\s+-\s+(.+)$"#, text, options: []) {
            return MatchSummary(
                home: localizeMatchPlayerName(groups[1]),
                away: localizeMatchPlayerName(groups[2]),
                homeScore: nil,
                awayScore: nil
            )
        }
        return nil
    }

    private static func buildCardHighlight(
        status: Int,
        roundText: String,
        latestResultText: String
    ) -> SaiXunCardHighlight? {
        guard let summary = parseMatchSummaryText(latestResultText) else { return nil }

        if status == 2 {
            guard roundText == "决赛",
                  let homeScore = summary.homeScore,
                  let awayScore = summary.awayScore,
                  homeScore != awayScore else {
                return nil
            }
            let homeWon = homeScore > awayScore
            let winner = homeWon ? summary.home : summary.away
            let loser = homeWon ? summary.away : summary.home
            return SaiXunCardHighlight(
                type: .champion,
                label: "冠军",
                primary: winner,
                secondary: "决赛 \(max(homeScore, awayScore))-\(min(homeScore, awayScore)) 胜 \(loser)"
            )
        }

        guard status == 0 || status == 1 else { return nil }
        let primary: String
        if let homeScore = summary.homeScore, let awayScore = summary.awayScore {
            primary = "\(summary.home) \(homeScore) - \(awayScore) \(summary.away)"
        } else {
            primary = "\(summary.home) vs \(summary.away)"
        }
        return SaiXunCardHighlight(
            type: status == 1 ? .live : .upcoming,
            label: roundText,
            primary: primary,
            secondary: ""
        )
    }

    private static func formatMonthDay(_ date: Date, calendar: Calendar) -> String {
        let components = calendar.dateComponents([.month, .day], from: date)
        return "\(components.month ?? 1)月\(components.day ?? 1)日"
    }

    private static func formatEventNewsTime(_ date: Date, now: Date, calendar: Calendar) -> String {
        let components = calendar.dateComponents([.month, .day, .hour, .minute], from: date)
        let clock = "\(SaiXunText.pad2(components.hour ?? 0)):\(SaiXunText.pad2(components.minute ?? 0))"

        switch diffLocalDays(date, now: now, calendar: calendar) {
        case 0: return "今天 \(clock)"
        case 1: return "明天 \(clock)"
        case 2: return "后天 \(clock)"
        default: return "\(components.month ?? 1)月\(components.day ?? 1)日 \(clock)"
        }
    }

    private static func diffLocalDays(_ date: Date, now: Date, calendar: Calendar) -> Int {
        let target = calendar.startOfDay(for: date)
        let today = calendar.startOfDay(for: now)
        return calendar.dateComponents([.day], from: today, to: target).day ?? 0
    }

    /// 小程序按本地时区取年月日；这里固定北京时区。
    private static func parseDateOnly(_ value: String?) -> Date? {
        guard let raw = value, !raw.isEmpty else { return nil }
        guard let groups = SaiXunText.captures(#"^(\d{4})-(\d{1,2})-(\d{1,2})$"#, raw, options: []),
              let year = Int(groups[1]), let month = Int(groups[2]), let day = Int(groups[3]) else {
            return nil
        }

        let calendar = SaiXunCalendar.shanghai
        guard let date = calendar.date(from: DateComponents(year: year, month: month, day: day)) else {
            return nil
        }
        let check = calendar.dateComponents([.year, .month, .day], from: date)
        guard check.year == year, check.month == month, check.day == day else { return nil }
        return date
    }

    /// 含 `T` 的文本按 ISO 8601（无时区时按本地）；否则把空格换成 `T` 并补 `+08:00`。
    private static func parseDateTime(_ value: String?) -> Date? {
        guard let raw = value, !raw.isEmpty else { return nil }

        if raw.contains("T") {
            if let date = isoFormatter().date(from: raw) { return date }
            return localDateTimeFormatter().date(from: raw)
        }

        var normalized = raw
        if let space = normalized.firstIndex(of: " ") {
            normalized.replaceSubrange(space...space, with: "T")
        }
        normalized += "+08:00"
        return isoFormatter().date(from: normalized)
    }

    private static func isoFormatter() -> ISO8601DateFormatter {
        let formatter = ISO8601DateFormatter()
        formatter.formatOptions = [.withInternetDateTime]
        return formatter
    }

    private static func localDateTimeFormatter() -> DateFormatter {
        let formatter = DateFormatter()
        formatter.locale = Locale(identifier: "en_US_POSIX")
        formatter.timeZone = TimeZone(identifier: "Asia/Shanghai")
        formatter.dateFormat = "yyyy-MM-dd'T'HH:mm:ss"
        return formatter
    }

    private static func normalizeMatch(
        _ match: EventNewsMatchInfo,
        now: Date,
        calendar: Calendar
    ) -> SaiXunMatch {
        let startAt = parseDateTime(match.startTime)
        let effectiveStatus = effectiveMatchStatus(match, startAt: startAt, now: now)

        let homeLocalizedName = localizeMatchPlayerName(match.homePlayerName)
        let awayLocalizedName = localizeMatchPlayerName(match.awayPlayerName)
        let homeEnglishName = homeLocalizedName.isEmpty ? "待定" : homeLocalizedName
        let awayEnglishName = awayLocalizedName.isEmpty ? "待定" : awayLocalizedName
        let homeNames = resolveMatchPlayerNames(match.homePlayerNameZh, englishName: homeEnglishName)
        let awayNames = resolveMatchPlayerNames(match.awayPlayerNameZh, englishName: awayEnglishName)
        let homeParts = resolvePlayerTextParts(
            firstName: match.homePlayerFirstName,
            lastName: match.homePlayerLastName,
            fullName: homeEnglishName
        )
        let awayParts = resolvePlayerTextParts(
            firstName: match.awayPlayerFirstName,
            lastName: match.awayPlayerLastName,
            fullName: awayEnglishName
        )

        let homeFlagEmoji = sanitizeFlagEmoji(match.homePlayerFlagEmoji)
        let awayFlagEmoji = sanitizeFlagEmoji(match.awayPlayerFlagEmoji)
        let homeCountryCode = SaiXunText.trimmed(match.homePlayerCountryCode).lowercased()
        let awayCountryCode = SaiXunText.trimmed(match.awayPlayerCountryCode).lowercased()
        let official = match.sourceType == "official"

        let showScore = isMatchScoreReady(match)
        let startComponents = startAt.map {
            calendar.dateComponents([.year, .month, .day, .hour, .minute, .weekday], from: $0)
        }
        let roundName = standardizeRoundText(match.roundName)

        return SaiXunMatch(
            id: match.id,
            roundName: roundName.isEmpty ? "轮次待更新" : roundName,
            roundOrder: match.roundOrder,
            matchOrder: match.matchOrder,
            status: effectiveStatus,
            statusText: eventNewsStatusText(effectiveStatus, fallback: "待更新"),
            isLive: effectiveStatus == 1,
            startAt: startAt,
            startTimeText: startAt.map { formatEventNewsTime($0, now: now, calendar: calendar) } ?? "时间待定",
            clockText: startComponents.map {
                "\(SaiXunText.pad2($0.hour ?? 0)):\(SaiXunText.pad2($0.minute ?? 0))"
            } ?? "时间待定",
            dayKey: startComponents.map {
                "\($0.year ?? 0)-\(SaiXunText.pad2($0.month ?? 1))-\(SaiXunText.pad2($0.day ?? 1))"
            } ?? "unscheduled",
            dayLabel: startAt.map { formatMatchDayLabel($0, now: now, calendar: calendar) } ?? "时间待定",
            homePlayerName: homeNames.name,
            homePlayerSubName: homeNames.subName,
            homePlayerFirstName: homeParts.firstName,
            homePlayerLastName: homeParts.lastName,
            homePlayerFlagEmoji: homeFlagEmoji,
            homePlayerCountryCode: homeCountryCode,
            homePlayerFlag: resolvePlayerFlag(homeFlagEmoji, homeCountryCode),
            homePlayerAvatar: resolvePlayerAvatar(match.homePlayerAvatar, official: official),
            homeResultText: buildPlayerResultText(match.winnerSide, side: 1),
            awayPlayerName: awayNames.name,
            awayPlayerSubName: awayNames.subName,
            awayPlayerFirstName: awayParts.firstName,
            awayPlayerLastName: awayParts.lastName,
            awayPlayerFlagEmoji: awayFlagEmoji,
            awayPlayerCountryCode: awayCountryCode,
            awayPlayerFlag: resolvePlayerFlag(awayFlagEmoji, awayCountryCode),
            awayPlayerAvatar: resolvePlayerAvatar(match.awayPlayerAvatar, official: official),
            awayResultText: buildPlayerResultText(match.winnerSide, side: 2),
            showScore: showScore,
            homeScore: match.homeScore,
            awayScore: match.awayScore,
            scoreText: showScore ? "\(match.homeScore) : \(match.awayScore)" : "-",
            winnerSide: match.winnerSide,
            metaText: formatBestOfText(match.bestOf),
            isPlaceholder: match.isPlaceholder
        )
    }

    private static func effectiveMatchStatus(
        _ match: EventNewsMatchInfo,
        startAt: Date?,
        now: Date
    ) -> Int {
        if match.status != 0 { return match.status }

        let hasWinner = match.winnerSide > 0
        let hasScore = match.homeScore > 0 || match.awayScore > 0
        if hasWinner && hasScore { return 2 }
        guard let startAt, startAt <= now else { return 0 }
        return now.timeIntervalSince(startAt) <= matchLiveFallbackWindow ? 1 : 2
    }

    /// 只按源数据状态判断比分是否可信，按开赛时间推断出的进行中不展示 0:0。
    private static func isMatchScoreReady(_ match: EventNewsMatchInfo) -> Bool {
        match.status == 1 || match.status == 2 || match.homeScore > 0 || match.awayScore > 0
    }

    private static func buildPlayerResultText(_ winnerSide: Int, side: Int) -> String {
        guard winnerSide == 1 || winnerSide == 2 else { return "" }
        guard side == 1 || side == 2 else { return "" }
        return winnerSide == side ? "胜" : "败"
    }

    /// 有后台维护的中文名时主名用中文、英文名作为副名；没有中文名时只显示英文名（占位名已在上一步译成中文）。
    private static func resolveMatchPlayerNames(
        _ nameZh: String?,
        englishName: String
    ) -> (name: String, subName: String) {
        let zh = SaiXunText.trimmed(nameZh)
        if zh.isEmpty || zh == englishName { return (englishName, "") }
        return (zh, englishName)
    }

    private static func splitDisplayName(_ name: String?) -> (firstName: String, lastName: String) {
        let text = SaiXunText.trimmed(name)
        if text.isEmpty || text == "待定" { return ("", text.isEmpty ? "待定" : text) }

        let parts = text.split(whereSeparator: { $0.isWhitespace }).map(String.init).filter { !$0.isEmpty }
        if parts.count <= 1 { return ("", text) }

        return (parts.dropLast().joined(separator: " "), parts[parts.count - 1])
    }

    private static func resolvePlayerTextParts(
        firstName: String?,
        lastName: String?,
        fullName: String
    ) -> (firstName: String, lastName: String) {
        let normalizedFirstName = SaiXunText.trimmed(firstName)
        let normalizedLastName = SaiXunText.trimmed(lastName)

        if !normalizedFirstName.isEmpty || !normalizedLastName.isEmpty {
            return (
                normalizedFirstName,
                normalizedLastName.isEmpty ? (fullName.isEmpty ? "待定" : fullName) : normalizedLastName
            )
        }
        return splitDisplayName(fullName)
    }

    private static func resolvePlayerAvatar(_ raw: String?, official: Bool) -> SaiXunImageSource {
        if let url = CDNAsset.url(for: SaiXunText.trimmed(raw)) {
            return .remote(url)
        }
        return official ? officialPlayerPlaceholder : defaultPlayerAvatar
    }

    private static func formatMatchDayLabel(_ date: Date, now: Date, calendar: Calendar) -> String {
        let components = calendar.dateComponents([.month, .day, .weekday], from: date)
        let weekdayIndex = (components.weekday ?? 1) - 1
        let weekday = SaiXunTables.weekdayLabels.indices.contains(weekdayIndex)
            ? SaiXunTables.weekdayLabels[weekdayIndex]
            : ""
        let base = "\(components.month ?? 1)月\(components.day ?? 1)日 \(weekday)"

        switch diffLocalDays(date, now: now, calendar: calendar) {
        case 0: return "\(base) · 今天"
        case 1: return "\(base) · 明天"
        case -1: return "\(base) · 昨天"
        default: return base
        }
    }

    private static func matchStartValue(_ match: SaiXunMatch) -> TimeInterval {
        match.startAt?.timeIntervalSince1970 ?? .infinity
    }

    /// 按开赛时间正序；没有开赛时间的排在最后。
    private static func compareMatches(_ left: SaiXunMatch, _ right: SaiXunMatch) -> Bool {
        let leftTime = matchStartValue(left)
        let rightTime = matchStartValue(right)
        if leftTime != rightTime { return leftTime < rightTime }
        if left.matchOrder != right.matchOrder { return left.matchOrder < right.matchOrder }
        return left.id < right.id
    }

    /// 按开赛时间倒序（最近结束的在前）；没有开赛时间的仍排在最后。
    private static func compareEndedMatches(_ left: SaiXunMatch, _ right: SaiXunMatch) -> Bool {
        let leftTime = matchStartValue(left)
        let rightTime = matchStartValue(right)
        if leftTime != rightTime {
            if !leftTime.isFinite { return false }
            if !rightTime.isFinite { return true }
            return leftTime > rightTime
        }
        return compareMatches(right, left)
    }

    private static func compareRounds(_ left: SaiXunRound, _ right: SaiXunRound) -> Bool {
        if left.roundOrder != right.roundOrder { return left.roundOrder < right.roundOrder }
        let leftTime = left.matches.first.map(matchStartValue) ?? .infinity
        let rightTime = right.matches.first.map(matchStartValue) ?? .infinity
        return leftTime < rightTime
    }

    /// 已取消的场次不会再打，与已结束放在一起。
    private static func isEndedMatch(_ match: SaiXunMatch) -> Bool {
        match.status == 2 || match.status == 3
    }
}
