import Foundation

// 迁移自 `uniapp/utils/friend-entry.js` 的好友规则。
enum FriendRules {
    static func displayName(_ nickname: String) -> String {
        nickname.isEmpty ? "球友" : nickname
    }

    /// 与 JS `resolveFriendUserId` 一致：优先 user_id，缺失或非法时退回关系记录 id。
    static func friendUserId(userId: Int64, id: Int64 = 0) -> Int64 {
        if userId > 0 { return userId }
        return id > 0 ? id : 0
    }

    static func applicantUserId(fromUserId: Int64, userId: Int64 = 0) -> Int64 {
        friendUserId(userId: fromUserId, id: userId)
    }

    static func searchStatus(isFriend: Bool, hasPendingRequest: Bool) -> FriendSearchStatus {
        if isFriend { return .friend }
        return hasPendingRequest ? .pending : .addable
    }

    /// 好友列表角标：超过 99 显示 99+，为 0 时不显示。
    static func pendingBadgeText(_ count: Int) -> String? {
        guard count > 0 else { return nil }
        return count > 99 ? "99+" : String(count)
    }

    static func deleteConfirmMessage(_ nickname: String) -> String {
        "确定删除好友「\(displayName(nickname))」吗？"
    }

    static func blacklistConfirmMessage(_ nickname: String) -> String {
        "确定将「\(displayName(nickname))」加入黑名单吗？加入后会自动删除好友，并阻止彼此再次搜索和添加。可在 设置 - 黑名单 中移出。"
    }

    static func rejectAndBlacklistConfirmMessage(_ nickname: String) -> String {
        "确定拒绝并拉黑「\(displayName(nickname))」吗？拉黑后对方将无法再搜索或添加你。"
    }

    static func unblacklistConfirmMessage(_ nickname: String) -> String {
        "确定将「\(displayName(nickname))」移出黑名单吗？移出后对方可以重新搜索并添加你。"
    }

    static func requestMessage(_ message: String) -> (text: String, isMuted: Bool) {
        message.isEmpty
            ? ("请求添加你为好友", true)
            : (message, false)
    }

    static func requestStatusText(_ status: Int) -> String {
        status == 1 ? "已接受" : "已拒绝"
    }

    /// 申请时间：无法解析时与 JS 一样回退成原始字符串。
    static func requestTime(_ createdAt: String, now: Date = Date()) -> String {
        guard !createdAt.isEmpty else { return "" }
        guard AppFormat.parseDateTime(createdAt) != nil else { return createdAt }
        return AppFormat.relativeTime(createdAt, now: now)
    }

    /// 拉黑时间：`formatRelativeTime` + 「拉黑」。
    static func blockedTime(_ createdAt: String, now: Date = Date()) -> String {
        guard !createdAt.isEmpty else { return "" }
        guard AppFormat.parseDateTime(createdAt) != nil else { return createdAt }
        return "\(AppFormat.relativeTime(createdAt, now: now))拉黑"
    }

    static func removeBlacklist(_ items: [FriendBlacklistItem], userId: Int64) -> [FriendBlacklistItem] {
        items.filter { $0.friendUserId != userId }
    }

    static func homepageTarget(_ friend: FriendInfo) -> FriendHomepageTarget {
        FriendHomepageTarget(
            userId: friend.friendUserId,
            name: friend.nickname,
            avatar: friend.avatar,
            rankName: friend.rankName,
            vipLevel: max(0, friend.vipLevel),
            rankLevel: max(0, friend.rankLevel)
        )
    }

    static func homepageRoute(_ friend: FriendInfo) -> AppRoute {
        .friendHomepage(target: homepageTarget(friend))
    }

    static func opponentRecordTarget(_ friend: FriendInfo) -> OpponentRecordTarget {
        OpponentRecordTarget(
            userId: friend.friendUserId,
            name: displayName(friend.nickname),
            avatar: friend.avatar
        )
    }

    static func opponentRecordRoute(_ friend: FriendInfo) -> AppRoute {
        .opponentRecord(target: opponentRecordTarget(friend))
    }
}
