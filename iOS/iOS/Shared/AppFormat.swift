import Foundation

/// 小程序 `utils/format.js` 的时间格式化规则。
///
/// 服务端时间统一按北京时间展示；小程序依赖设备本地时区，原生端固定 `Asia/Shanghai`。
enum AppFormat {
    private static let timeZone = TimeZone(identifier: "Asia/Shanghai")!

    /// 固定北京时区的公历，供同模块的日历规则复用。
    static var beijingCalendar: Calendar {
        var calendar = Calendar(identifier: .gregorian)
        calendar.timeZone = timeZone
        calendar.locale = Locale(identifier: "en_US_POSIX")
        return calendar
    }

    private static var calendar: Calendar { beijingCalendar }

    /// 解析 ISO 8601（带偏移或 Z）和 `YYYY-MM-DD HH:mm:ss`；空格分隔按北京时间解释。
    static func parseDateTime(_ text: String?) -> Date? {
        guard let text else { return nil }
        let trimmed = text.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !trimmed.isEmpty else { return nil }

        if trimmed.contains("T") {
            let formatter = ISO8601DateFormatter()
            formatter.formatOptions = [.withInternetDateTime]
            if let date = formatter.date(from: trimmed) { return date }
            // 带毫秒的 ISO 串需要额外选项，小程序 test 里出现过 `.000Z`。
            formatter.formatOptions = [.withInternetDateTime, .withFractionalSeconds]
            return formatter.date(from: trimmed)
        }

        let formatter = DateFormatter()
        formatter.locale = Locale(identifier: "en_US_POSIX")
        formatter.timeZone = timeZone
        for pattern in ["yyyy-MM-dd HH:mm:ss", "yyyy-MM-dd HH:mm", "yyyy-MM-dd"] {
            formatter.dateFormat = pattern
            if let date = formatter.date(from: trimmed) { return date }
        }
        return nil
    }

    /// 与 JS 一致的相对时间：一分钟内“刚刚”，之后逐级放大到年。
    static func relativeTime(_ text: String?, now: Date = Date()) -> String {
        guard let date = parseDateTime(text) else { return "" }
        let seconds = Int(now.timeIntervalSince(date).rounded(.down))
        if seconds < 60 { return "刚刚" }

        let minutes = seconds / 60
        if minutes < 60 { return "\(String(minutes))分钟前" }

        let hours = minutes / 60
        if hours < 24 { return "\(String(hours))小时前" }

        let days = hours / 24
        if days < 30 { return "\(String(days))天前" }

        let months = days / 30
        return months < 12 ? "\(String(months))个月前" : "\(String(days / 365))年前"
    }

    /// 默认模板与 `formatDate` 相同：`YYYY年MM月DD日`。
    static func date(_ text: String?, format: String = "YYYY年MM月DD日") -> String {
        guard let date = parseDateTime(text) else { return "" }
        return string(from: date, format: format)
    }

    /// `formatDateTime`：`YYYY年MM月DD日 HH:mm`。
    static func dateTime(_ text: String?) -> String {
        date(text, format: "YYYY年MM月DD日 HH:mm")
    }

    /// 与 JS `String.replace` 相同：每个占位符只替换第一次出现。
    private static func string(from date: Date, format: String) -> String {
        let parts = calendar.dateComponents([.year, .month, .day, .hour, .minute, .second], from: date)
        let replacements = [
            ("YYYY", String(format: "%04d", parts.year ?? 0)),
            ("MM", String(format: "%02d", parts.month ?? 0)),
            ("DD", String(format: "%02d", parts.day ?? 0)),
            ("HH", String(format: "%02d", parts.hour ?? 0)),
            ("mm", String(format: "%02d", parts.minute ?? 0)),
            ("ss", String(format: "%02d", parts.second ?? 0)),
        ]
        var result = format
        for (token, value) in replacements {
            if let range = result.range(of: token) {
                result.replaceSubrange(range, with: value)
            }
        }
        return result
    }

    /// 北京时间 `YYYY-MM` 月份键。
    static func monthKey(_ date: Date) -> String {
        let parts = calendar.dateComponents([.year, .month], from: date)
        return String(format: "%04d-%02d", parts.year ?? 0, parts.month ?? 0)
    }

    /// 字符串输入先解析再取北京时间月份；无法解析返回空串。
    static func monthKey(_ text: String?) -> String {
        guard let date = parseDateTime(text) else { return "" }
        return monthKey(date)
    }

    /// `M月D日`，用于交锋列表的日期前缀。
    static func monthDay(_ text: String?) -> String? {
        guard let date = parseDateTime(text) else { return nil }
        let parts = calendar.dateComponents([.month, .day], from: date)
        return "\(String(parts.month ?? 0))月\(String(parts.day ?? 0))日"
    }

    static func dayKey(_ date: Date) -> String {
        let parts = calendar.dateComponents([.year, .month, .day], from: date)
        return String(format: "%04d-%02d-%02d", parts.year ?? 0, parts.month ?? 0, parts.day ?? 0)
    }
}
