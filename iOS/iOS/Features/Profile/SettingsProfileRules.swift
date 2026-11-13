import Foundation

/// 设置/编辑资料页的纯规则，对应小程序 `utils/settings-profile.js`。
enum SettingsProfileRules {
    static let avatarImageMaxEdge = 512
    static let avatarImageQuality = 80
    static let defaultQiniuUploadURL = "https://up-z2.qiniup.com"

    struct AvatarDimensions: Equatable {
        let width: Int
        let height: Int
    }

    static func formatSettingsPhone(_ phone: String = "") -> String {
        let value = phone.trimmingCharacters(in: .whitespacesAndNewlines)
        if value.isEmpty { return "" }

        if value.range(of: #"^\d{11}$"#, options: .regularExpression) != nil {
            return "+86 \(value.prefix(3))****\(value.suffix(4))"
        }
        if value.range(of: #"^\d{3}\*{4}\d{4}$"#, options: .regularExpression) != nil {
            return "+86 \(value)"
        }
        if value.hasPrefix("+86 ") || value.hasPrefix("+") { return value }
        return "+86 \(value)"
    }

    static func getFileExtensionFromPath(_ filePath: String = "") -> String {
        let value = filePath.split(separator: "?", maxSplits: 1, omittingEmptySubsequences: false)
            .first.map(String.init) ?? ""
        guard let dotIndex = value.lastIndex(of: ".") else { return ".jpg" }
        let ext = value[dotIndex...].lowercased()
        let body = ext.dropFirst()
        guard !body.isEmpty, body.allSatisfy({ $0.isASCII && ($0.isLetter || $0.isNumber) }) else {
            return ".jpg"
        }
        return ext
    }

    static func buildQiniuAvatarObjectKey(
        userId: Int64 = 0,
        filePath: String = "",
        timestamp: Date = Date()
    ) -> String {
        let formatter = DateFormatter()
        formatter.calendar = Calendar(identifier: .gregorian)
        formatter.locale = Locale(identifier: "en_US_POSIX")
        formatter.timeZone = .current
        formatter.dateFormat = "yyyyMMdd"

        let dateSegment = formatter.string(from: timestamp)
        let milliseconds = Int64((timestamp.timeIntervalSince1970 * 1000).rounded(.down))
        let ext = getFileExtensionFromPath(filePath)
        return "avatars/\(userId)/\(dateSegment)/\(milliseconds)\(ext)"
    }

    static func normalizeQiniuUploadResult(domain: String = "", key: String = "") -> String {
        var normalizedDomain = domain.trimmingCharacters(in: .whitespacesAndNewlines)
        var normalizedKey = key.trimmingCharacters(in: .whitespacesAndNewlines)
        while normalizedDomain.hasSuffix("/") { normalizedDomain.removeLast() }
        while normalizedKey.hasPrefix("/") { normalizedKey.removeFirst() }
        guard !normalizedDomain.isEmpty, !normalizedKey.isEmpty else { return "" }
        return "\(normalizedDomain)/\(normalizedKey)"
    }

    static func getAvatarCompressionDimensions(
        width: Int = 0,
        height: Int = 0,
        maxEdge: Int = avatarImageMaxEdge
    ) -> AvatarDimensions {
        let normalizedMaxEdge = maxEdge > 0 ? maxEdge : avatarImageMaxEdge
        guard width > 0, height > 0 else {
            return AvatarDimensions(width: normalizedMaxEdge, height: normalizedMaxEdge)
        }

        let longestEdge = max(width, height)
        guard longestEdge > normalizedMaxEdge else {
            return AvatarDimensions(width: width, height: height)
        }

        let scale = Double(normalizedMaxEdge) / Double(longestEdge)
        return AvatarDimensions(
            width: max(1, Int((Double(width) * scale).rounded())),
            height: max(1, Int((Double(height) * scale).rounded()))
        )
    }
}
