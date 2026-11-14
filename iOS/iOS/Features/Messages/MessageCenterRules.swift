import Foundation

/// 消息 Tab 的纯逻辑：通知列表何时回源、顶部入口条目，以及通知到原生路由的映射。
enum MessageCenterRules {
    /// load：需要重新加载首屏；wait：该身份的加载已在进行；reuse：沿用已加载的列表。
    static func resolveNotificationReload(
        identityKey: String,
        loadedIdentityKey: String,
        requestedIdentityKey: String,
        loadedScopeVersion: Int,
        scopeVersion: Int
    ) -> NotificationReloadDecision {
        if loadedIdentityKey != identityKey {
            return requestedIdentityKey == identityKey ? .wait : .load
        }
        return loadedScopeVersion == scopeVersion ? .reuse : .load
    }

    static func formatEntryCount(_ value: Int) -> String {
        let count = max(0, value)
        if count == 0 { return "" }
        return count > 99 ? "99+" : String(count)
    }

    static func buildMessageEntries(friendRequestCount: Int = 0, challengeCount: Int = 0) -> [MessageEntry] {
        let friends = max(0, friendRequestCount)
        let invites = max(0, challengeCount)
        return [
            MessageEntry(
                id: "friend-requests",
                title: "好友申请",
                symbol: "person.badge.plus",
                route: .friendRequests,
                countText: formatEntryCount(friends),
                desc: friends > 0 ? "\(friends) 条待处理" : "暂无待处理申请"
            ),
            MessageEntry(
                id: "challenge-invites",
                title: "约球邀请",
                symbol: "envelope",
                route: .challenges(tab: .received),
                countText: formatEntryCount(invites),
                desc: invites > 0 ? "\(invites) 条待回应" : "暂无待回应的邀请"
            ),
        ]
    }

    /// 通知点击后的跳转，逐条对应 `pages/message/index.vue` 的 `navigateByNotification`。
    /// `data` 解析失败时返回 nil（不跳转）；目标页未迁移的落到已有占位路由。
    static func notificationRoute(type: String, data rawData: String) -> AppRoute? {
        guard let data = parseNotificationData(rawData) else { return nil }
        let target = (data["target"] as? String) ?? (data["url"] as? String) ?? ""
        let matchId = int64(data["match_id"])
        let challengeId = int64(data["challenge_id"])

        if !target.isEmpty {
            let path = String(target.split(separator: "?", maxSplits: 1).first ?? "")
            switch path {
            case "/subPages/match/matchResult":
                return matchId > 0 ? .matchReport(matchId: matchId) : nil
            case "/subPages/match/playing":
                return matchId > 0 ? .liveMatch(matchId: matchId) : nil
            case "/subPages/social/friendRequests":
                return .friendRequests
            case "/subPages/social/friendList":
                return .friendList
            case "/subPages/social/challenges":
                return .challenges(tab: .received)
            case "/subPages/achievement/index":
                return .achievementWall(gameType: queryInt(target, key: "game_type") ?? 0)
            case "/subPages/season/report":
                return .seasonIndex
            default:
                return nil
            }
        }

        // 约球通知：只打开权威详情，不替用户进入或开局。
        if type == "challenge", challengeId > 0 {
            return .challengeWaiting(challengeId: challengeId)
        }
        if type == "friend_request" {
            return .friendRequests
        }
        return nil
    }

    /// 与小程序一致：空数据视为 `{}`，非空但解析失败返回 nil。
    private static func parseNotificationData(_ raw: String) -> [String: Any]? {
        let trimmed = raw.trimmingCharacters(in: .whitespacesAndNewlines)
        if trimmed.isEmpty { return [:] }
        guard let data = trimmed.data(using: .utf8),
              let object = try? JSONSerialization.jsonObject(with: data),
              let dictionary = object as? [String: Any] else {
            return nil
        }
        return dictionary
    }

    private static func int64(_ value: Any?) -> Int64 {
        switch value {
        case let number as Int64: number
        case let number as Int: Int64(number)
        case let number as Double: Int64(number)
        case let text as String: Int64(text) ?? 0
        default: 0
        }
    }

    private static func queryInt(_ target: String, key: String) -> Int? {
        guard let components = URLComponents(string: "https://example.test\(target)"),
              let value = components.queryItems?.first(where: { $0.name == key })?.value else {
            return nil
        }
        return Int(value)
    }
}

enum NotificationReloadDecision: Equatable {
    case load
    case wait
    case reuse
}

struct MessageEntry: Identifiable, Equatable {
    let id: String
    let title: String
    let symbol: String
    let route: AppRoute
    let countText: String
    let desc: String
}
