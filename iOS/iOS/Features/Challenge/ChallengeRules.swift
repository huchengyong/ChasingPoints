import Foundation

/// 约球规则，逐条对应小程序 `utils/challenge-view.js`、`challenge-time.js`、`challenge-entry.js`。
/// 时间只按服务端北京时间字符串计算，不依赖设备时区。
enum ChallengeRules {
    // MARK: - 类型

    struct DayParts: Equatable {
        let year: Int
        let month: Int
        let day: Int
        let hour: Int
        let minute: Int
    }

    struct DayOption: Equatable, Identifiable {
        let value: Int
        let label: String

        var id: Int { value }
    }

    struct Slot: Equatable {
        let dayOffset: Int
        let startHour: Int
        let endHour: Int
    }

    static let dayOptionsBase: [DayOption] = [
        DayOption(value: 0, label: "今天"),
        DayOption(value: 1, label: "明天"),
        DayOption(value: 2, label: "后天"),
    ]

    private static let serverTimePattern = #"^(\d{4})-(\d{2})-(\d{2})[ T](\d{2}):(\d{2})(?::(\d{2}))?$"#
    private static let datePattern = #"^(\d{4})-(\d{2})-(\d{2})$"#

    // MARK: - 时间

    static func parseServerTime(_ value: String) -> DayParts? {
        guard let match = groups(pattern: serverTimePattern, in: value),
              let year = Int(match[0]),
              let month = Int(match[1]),
              let day = Int(match[2]),
              let hour = Int(match[3]),
              let minute = Int(match[4]) else { return nil }
        return DayParts(year: year, month: month, day: day, hour: hour, minute: minute)
    }

    /// 按服务端日期 + 相对天数计算实际预约日。
    static func computeScheduledDate(serverTime: String, dayOffset: Int) -> String {
        guard let parts = parseServerTime(serverTime),
              let base = utcDate(year: parts.year, month: parts.month, day: parts.day),
              let shifted = utcCalendar.date(byAdding: .day, value: dayOffset, to: base) else { return "" }
        return dateKey(shifted)
    }

    /// 当天开始小时需满足区间未完全过去；服务端时间缺失时返回全部小时。
    static func availableStartHours(serverTime: String, dayOffset: Int) -> [Int] {
        let all = Array(0...23)
        guard let parts = parseServerTime(serverTime) else { return all }
        if dayOffset > 0 { return all }
        return all.filter { $0 >= parts.hour }
    }

    /// 默认选择下一个可用整点；当天没有可用区间则明天 0—1 点。
    static func defaultSlot(serverTime: String) -> Slot {
        let parts = parseServerTime(serverTime)
        for dayOffset in [0, 1, 2] {
            let hours = availableStartHours(serverTime: serverTime, dayOffset: dayOffset)
            let candidates: [Int]
            if let parts, dayOffset == 0 {
                candidates = hours.filter { $0 > parts.hour || ($0 == parts.hour && parts.minute == 0) }
            } else {
                candidates = hours
            }
            guard let startHour = (candidates.isEmpty ? hours : candidates).first else { continue }
            if dayOffset == 0, let parts, startHour == parts.hour, parts.minute > 0,
               !candidates.contains(startHour) {
                continue
            }
            if !hours.isEmpty {
                return Slot(dayOffset: dayOffset, startHour: startHour, endHour: min(startHour + 1, 24))
            }
        }
        return Slot(dayOffset: 1, startHour: 0, endHour: 1)
    }

    /// 实际预约日相对服务端今天的天数差（可为负）；无法解析时返回 nil。
    static func dayOffsetOfScheduledDate(serverTime: String, scheduledDate: String) -> Int? {
        guard let parts = parseServerTime(serverTime),
              let match = groups(pattern: datePattern, in: scheduledDate),
              let year = Int(match[0]), let month = Int(match[1]), let day = Int(match[2]),
              let today = utcDate(year: parts.year, month: parts.month, day: parts.day),
              let scheduled = utcDate(year: year, month: month, day: day) else { return nil }
        return utcCalendar.dateComponents([.day], from: today, to: scheduled).day
    }

    /// 结束小时必须晚于开始，范围 1—24。
    static func endHourOptions(startHour: Int) -> [Int] {
        guard (0...23).contains(startHour) else { return [] }
        return Array((startHour + 1)...24)
    }

    static func hourLabel(_ hour: Int) -> String { "\(hour)点" }

    /// 服务端时间缺失时的本地兜底：按北京时间格式化当前时刻。
    static func localServerTime(now: Date) -> String {
        var calendar = Calendar(identifier: .gregorian)
        calendar.timeZone = TimeZone(secondsFromGMT: 8 * 3600) ?? .current
        let parts = calendar.dateComponents([.year, .month, .day, .hour, .minute], from: now)
        return String(
            format: "%04d-%02d-%02d %02d:%02d:00",
            parts.year ?? 0,
            parts.month ?? 0,
            parts.day ?? 0,
            parts.hour ?? 0,
            parts.minute ?? 0
        )
    }

    /// 发起页可选的相对日期；当天没有剩余时段时不展示今天。
    static func dayOptions(serverTime: String) -> [DayOption] {
        guard !serverTime.isEmpty else { return dayOptionsBase }
        return dayOptionsBase.filter {
            !availableStartHours(serverTime: serverTime, dayOffset: $0.value).isEmpty || $0.value > 0
        }
    }

    // MARK: - 展示

    /// 今天 / 明天 / 昨天 / 月日（跨年带年份）。
    static func dayLabel(_ scheduledDate: String, serverTime: String) -> String {
        let trimmed = scheduledDate.trimmingCharacters(in: .whitespacesAndNewlines)
        guard let match = groups(pattern: datePattern, in: trimmed),
              let targetYear = Int(match[0]), let targetMonth = Int(match[1]), let targetDay = Int(match[2]),
              let now = parseServerTime(serverTime),
              let target = utcDate(year: targetYear, month: targetMonth, day: targetDay),
              let today = utcDate(year: now.year, month: now.month, day: now.day) else {
            return String(trimmed.prefix(10))
        }
        let dayDiff = utcCalendar.dateComponents([.day], from: today, to: target).day ?? 0
        if dayDiff == 0 { return "今天" }
        if dayDiff == 1 { return "明天" }
        if dayDiff == -1 { return "昨天" }
        if targetYear == now.year { return "\(targetMonth)月\(targetDay)日" }
        return "\(targetYear)年\(targetMonth)月\(targetDay)日"
    }

    /// 卡片/详情的预约时间行：`今天16点—17点`。
    static func schedule(_ challenge: ChallengeInfo?, serverTime: String) -> String {
        guard let challenge,
              let scheduledDate = challenge.scheduledDate,
              !scheduledDate.isEmpty else { return "" }
        let day = dayLabel(scheduledDate, serverTime: serverTime)
        guard let startHour = challenge.startHour, let endHour = challenge.endHour else { return day }
        return "\(day)\(startHour)点—\(endHour)点"
    }

    /// 失效提示：今天/明天早晨 7 点失效；已过期或非次晨 7 点显示已失效。
    static func expiry(_ challenge: ChallengeInfo?, serverTime: String) -> String {
        guard let challenge, let expiresAt = challenge.expiresAt, !expiresAt.isEmpty else { return "" }
        guard let now = parseServerTime(serverTime), let expiry = parseServerTime(expiresAt) else {
            return "已失效"
        }
        guard let expiryDate = utcDate(year: expiry.year, month: expiry.month, day: expiry.day),
              let nowDate = utcDate(year: now.year, month: now.month, day: now.day) else { return "已失效" }
        let dayDiff = utcCalendar.dateComponents([.day], from: nowDate, to: expiryDate).day ?? 0
        let nowMinutes = now.hour * 60 + now.minute
        let expiryMinutes = expiry.hour * 60 + expiry.minute
        if dayDiff < 0 || (dayDiff == 0 && nowMinutes >= expiryMinutes) { return "已失效" }
        if dayDiff == 0 && expiryMinutes == 420 { return "今天早晨7点失效" }
        if dayDiff == 1 && expiryMinutes == 420 { return "明天早晨7点失效" }
        return "已失效"
    }

    /// 首页/活动卡使用的有效状态判断：过期由服务端时间派生。
    static func isExpired(status: Int, expiresAt: String?, serverTime: String) -> Bool {
        if status == 3 { return true }
        guard status == 0 || status == 1 else { return false }
        guard let expiresAt, !expiresAt.isEmpty, !serverTime.isEmpty else { return false }
        return expiresAt <= serverTime
    }

    /// 约球待办卡；不认识的状态返回 nil。
    static func quickCard(_ challenge: ChallengeInfo?, userId: Int64, serverTime: String) -> ChallengeQuickCard? {
        guard let challenge else { return nil }
        let status = challenge.status
        let viewerId = max(userId, 0)
        let waitingUserId = max(challenge.waitingUserId ?? 0, 0)
        let isSender = challenge.fromUserId == viewerId

        let title: String
        var needsAction = false
        switch status {
        case 6:
            title = "对局进行中"
        case 1:
            if waitingUserId == viewerId {
                title = "你已进入，等待对方"
            } else if waitingUserId > 0 {
                title = "对方已进入，等你开始"
                needsAction = true
            } else {
                title = "已约好，待进入"
            }
        case 0:
            title = isSender ? "等待对方回应" : "收到约球邀请"
            needsAction = !isSender
        default:
            return nil
        }

        let matchId = status == 6 ? max(challenge.matchId ?? 0, 0) : 0
        let opponentName = trimmed(isSender ? challenge.toNickname : challenge.fromNickname)
        let modeText = challenge.matchMode == "practice" ? "练习" : "排位"
        let subText = [
            opponentName.isEmpty ? "球友" : opponentName,
            GameTypeCatalog.label(for: challenge.gameType),
            modeText,
            schedule(challenge, serverTime: serverTime),
        ]
        .filter { !$0.isEmpty }
        .joined(separator: " · ")

        return ChallengeQuickCard(
            challengeId: challenge.id,
            matchId: matchId,
            title: title,
            needsAction: needsAction,
            subText: subText,
            actionLabel: matchId > 0 ? "继续比赛" : "进入"
        )
    }

    static func opponent(_ item: ChallengeInfo, userId: Int64) -> ChallengeListItemView {
        let isSent = userId > 0 ? item.fromUserId == userId : false
        return isSent
            ? ChallengeListItemView(
                direction: .sent,
                opponentId: item.toUserId,
                nickname: item.toNickname,
                avatar: item.toAvatar
            )
            : ChallengeListItemView(
                direction: .received,
                opponentId: item.fromUserId,
                nickname: item.fromNickname,
                avatar: item.fromAvatar
            )
    }

    static func isMineReceivedPending(_ item: ChallengeInfo, userId: Int64) -> Bool {
        item.toUserId == userId && item.status == 0
    }

    static func isMineSentPending(_ item: ChallengeInfo, userId: Int64) -> Bool {
        item.fromUserId == userId && item.status == 0
    }

    static func isTerminal(_ item: ChallengeInfo) -> Bool {
        [2, 3, 4, 5, 7, 8].contains(item.status)
    }

    static func statusLabel(_ item: ChallengeInfo, userId: Int64) -> String {
        switch item.status {
        case 0:
            return item.fromUserId == userId ? "等待回应" : "待你回应"
        case 1:
            let waitingUserId = max(item.waitingUserId ?? 0, 0)
            if waitingUserId == userId { return "等待对方进入" }
            if waitingUserId > 0 { return "对方已进入" }
            return "已接受"
        case 6:
            return "对局进行中"
        case 2: return "已拒绝"
        case 3: return "已失效"
        case 4: return "已取消"
        case 5: return "已放弃"
        case 7: return "已完成"
        case 8: return "对局已取消"
        default: return ""
        }
    }

    static func emptyText(tab: ChallengeTab) -> String {
        switch tab {
        case .received: return "暂无待回应的约球"
        case .sent: return "暂无进行中的约球"
        case .history: return "暂无约球记录"
        }
    }

    static func matchModeLabel(_ value: String?) -> String {
        value == "practice" ? "练习" : "排位"
    }

    /// 昵称为空时显示「球友」。
    static func displayName(_ value: String) -> String {
        let name = trimmed(value)
        return name.isEmpty ? "球友" : name
    }

    static func matchModeHint(_ value: String) -> String {
        value == "ranked"
            ? "排位赛公开展示并计入段位，双方进入后单方滑动即可结束"
            : "练习赛不计段位，仅记录战绩"
    }

    // MARK: - 入口

    static func buildPayload(
        targetUserId: Int64,
        recordId: Int64 = 0,
        gameType: Int,
        message: String
    ) -> ChallengeEntryPayload {
        let resolved = targetUserId > 0 ? targetUserId : (recordId > 0 ? recordId : 0)
        return ChallengeEntryPayload(toUserId: resolved, gameType: gameType, message: message)
    }

    static func resolveAction(_ item: ChallengeInfo) -> ChallengeEntryAction {
        let matchId = max(item.matchId ?? 0, 0)
        if item.status == 1 && matchId > 0 {
            return ChallengeEntryAction(kind: .viewMatch, label: "查看对局", matchId: matchId)
        }
        if item.status == 1 && matchId == 0 {
            return ChallengeEntryAction(kind: .offlineStart, label: "线下扫码开局", matchId: 0)
        }
        return ChallengeEntryAction(kind: .report, label: "查看PK报表", matchId: matchId)
    }

    static func buildStartContext(_ item: ChallengeInfo, opponent: ChallengeListItemView) -> ChallengeStartContext? {
        guard resolveAction(item).kind == .offlineStart else { return nil }
        let name = trimmed(opponent.nickname)
        let gameType = item.gameType
        return ChallengeStartContext(
            challengeId: item.id,
            opponentId: opponent.opponentId,
            opponentName: name.isEmpty ? "球友" : name,
            opponentAvatar: opponent.avatar,
            gameType: gameType > 0 ? gameType : 3
        )
    }

    // MARK: - 私有

    private static let utcCalendar: Calendar = {
        var calendar = Calendar(identifier: .gregorian)
        calendar.timeZone = TimeZone(secondsFromGMT: 0) ?? .current
        return calendar
    }()

    private static func utcDate(year: Int, month: Int, day: Int) -> Date? {
        utcCalendar.date(from: DateComponents(year: year, month: month, day: day))
    }

    private static func dateKey(_ date: Date) -> String {
        let parts = utcCalendar.dateComponents([.year, .month, .day], from: date)
        return String(format: "%04d-%02d-%02d", parts.year ?? 0, parts.month ?? 0, parts.day ?? 0)
    }

    /// 返回正则的捕获组；整串必须匹配。
    private static func groups(pattern: String, in value: String) -> [String]? {
        let text = value.trimmingCharacters(in: .whitespacesAndNewlines)
        guard let regex = try? NSRegularExpression(pattern: pattern) else { return nil }
        let range = NSRange(text.startIndex..<text.endIndex, in: text)
        guard let match = regex.firstMatch(in: text, range: range), match.range == range else { return nil }
        return (1..<match.numberOfRanges).compactMap { index in
            Range(match.range(at: index), in: text).map { String(text[$0]) }
        }
    }

    private static func trimmed(_ value: String) -> String {
        value.trimmingCharacters(in: .whitespacesAndNewlines)
    }
}
