import Foundation

struct MatchScanOpponent: Equatable {
    let id: Int64
    let nickname: String
    let avatar: String
}

/// 裁判入场的凭据；`joinToken` 是服务端签发的一次性凭证，不进日志与打印。
struct RefereeJoin: Encodable, Equatable, CustomStringConvertible {
    let matchId: Int64
    let joinToken: String

    var description: String { "RefereeJoin(matchId: \(matchId), joinToken: <redacted>)" }
}

/// 扫码文本的解析与分流结果。
enum MatchScanResult: Equatable, CustomStringConvertible {
    case error(message: String)
    case resolveInviteCode(inviteCode: String)
    case openMatch(matchId: Int64)
    case joinReferee(refereeJoin: RefereeJoin)
    case previewMatchInvite(inviteToken: String)
    case startMatch(inviteToken: String, opponent: MatchScanOpponent)

    /// invite 短码与 invite token 都是凭证，打印时遮住。
    var description: String {
        switch self {
        case .error(let message): "error(\(message))"
        case .resolveInviteCode: "resolveInviteCode(<redacted>)"
        case .openMatch(let matchId): "openMatch(matchId: \(matchId))"
        case .joinReferee(let refereeJoin): "joinReferee(\(refereeJoin))"
        case .previewMatchInvite: "previewMatchInvite(<redacted>)"
        case .startMatch(_, let opponent): "startMatch(<redacted>, opponentId: \(opponent.id))"
        }
    }
}

/// 逐条对应小程序 `utils/match-scan.js` 的扫码文本规则。
/// 调用相机、相册与权限的部分不迁移；需要请求接口的地方由调用方注入闭包。
enum MatchScan {
    private static let inviteTokenPattern = try! NSRegularExpression(
        pattern: #"^[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+$"#
    )
    private static let urlPattern = try! NSRegularExpression(
        pattern: #"^https?:\/\/"#,
        options: [.caseInsensitive]
    )

    /// 只有入口链接的匹配码 / 对局两种能进业务，其他网址一律无效；不认识的内容按「无法识别」处理。
    static func parsePayload(_ rawValue: String) -> MatchScanResult {
        let value = rawValue.trimmingCharacters(in: .whitespacesAndNewlines)
        if value.isEmpty {
            return .error(message: "无效的二维码")
        }

        if matches(urlPattern, value) {
            switch EntryLink.parse(value) {
            case .invite(let inviteCode):
                return .resolveInviteCode(inviteCode: inviteCode)
            case .match(let matchId):
                return .openMatch(matchId: matchId)
            default:
                return .error(message: "无效的二维码")
            }
        }

        var parsedAsJSON = false
        var object: [String: Any]?
        if let data = value.data(using: .utf8),
           let json = try? JSONSerialization.jsonObject(with: data, options: .fragmentsAllowed) {
            parsedAsJSON = true
            object = json as? [String: Any]
        }

        if parsedAsJSON {
            if let object,
               object["type"] as? String == "match_referee",
               let rawMatchId = object["match_id"],
               let matchId = number(rawMatchId), matchId != 0,
               let joinToken = object["join_token"] as? String, !joinToken.isEmpty {
                return .joinReferee(refereeJoin: RefereeJoin(matchId: matchId, joinToken: joinToken))
            }
            return .error(message: "该二维码已不再支持，请让对方刷新匹配二维码")
        }

        guard matches(inviteTokenPattern, value) else {
            return .error(message: "无效的二维码")
        }
        return .previewMatchInvite(inviteToken: value)
    }

    /// 拿到扫码文本之后的分流：短码换成签名 token、校验 token 并归一化对手预览。
    static func resolvePayload(
        rawValue: String,
        previewMatchInvite: ((String) async throws -> MatchInvitePreviewResponse)? = nil,
        resolveMatchInviteCode: ((String) async throws -> MatchInviteResolveResponse)? = nil
    ) async -> MatchScanResult {
        let parsed = parsePayload(rawValue)

        if case .resolveInviteCode(let inviteCode) = parsed {
            guard let resolveMatchInviteCode else {
                return .error(message: "匹配二维码服务暂不可用")
            }
            do {
                let response = try await resolveMatchInviteCode(inviteCode)
                guard response.success,
                      let token = response.inviteToken, !token.isEmpty,
                      let preview = response.preview, preview.opponentId != 0 else {
                    return .error(message: failureMessage(response.message))
                }
                return buildStartMatch(inviteToken: token, preview: preview)
            } catch {
                return .error(message: "匹配二维码服务暂不可用，请稍后重试")
            }
        }

        guard case .previewMatchInvite(let inviteToken) = parsed else { return parsed }
        guard let previewMatchInvite else {
            return .error(message: "匹配二维码服务暂不可用")
        }
        do {
            let response = try await previewMatchInvite(inviteToken)
            guard response.success, let preview = response.preview, preview.opponentId != 0 else {
                return .error(message: failureMessage(response.message))
            }
            return buildStartMatch(inviteToken: inviteToken, preview: preview)
        } catch {
            return .error(message: "匹配二维码服务暂不可用，请稍后重试")
        }
    }

    private static func buildStartMatch(inviteToken: String, preview: MatchInvitePreviewInfo) -> MatchScanResult {
        .startMatch(
            inviteToken: inviteToken,
            opponent: MatchScanOpponent(
                id: preview.opponentId,
                nickname: preview.opponentName.isEmpty ? "对手" : preview.opponentName,
                avatar: preview.opponentAvatar
            )
        )
    }

    private static func failureMessage(_ message: String?) -> String {
        let text = message ?? ""
        return text.isEmpty ? "匹配二维码已失效，请让对方刷新二维码" : text
    }

    private static func matches(_ regex: NSRegularExpression, _ value: String) -> Bool {
        regex.firstMatch(in: value, range: NSRange(value.startIndex..., in: value)) != nil
    }

    /// JS 的 `Number(value)`：数字直取，数字字符串可解析；其余视为没有值。
    private static func number(_ value: Any) -> Int64? {
        if let number = value as? NSNumber { return number.int64Value }
        if let text = value as? String {
            return Int64(text.trimmingCharacters(in: .whitespacesAndNewlines))
        }
        return nil
    }
}
