import Foundation

struct LinkEntryInviter: Equatable {
    let name: String
    let avatar: String
}

/// 承接页的状态结果；`ignore` 表示已有请求在途、已结束或身份不符，页面保持原状。
enum LinkEntryAction: Equatable {
    case awaitLogin(inviter: LinkEntryInviter, expiresAt: String)
    case confirm
    case expired(message: String)
    case error(message: String)
    case ignore
}

/// 逐条对应小程序 `utils/link-entry-flow.js`：先匿名预览，登录后兑换一次并交接给确认页。
@MainActor
final class LinkEntryFlow {
    typealias PreviewInvite = (String) async throws -> LinkEntryPreviewResponse
    typealias ResolveInvite = (String) async throws -> MatchInviteResolveResponse
    typealias HandoffResolved = (SessionIdentity, String, MatchScanOpponent) -> Bool

    static let expiredMessage = "匹配二维码已失效，请让对方刷新二维码"
    static let unavailableMessage = "匹配二维码服务暂不可用，请稍后重试"

    private let currentIdentity: () -> SessionIdentity?
    private let previewInvite: PreviewInvite
    private let resolveInvite: ResolveInvite
    private let handoffResolved: HandoffResolved

    private var inviteCode = ""
    private var started = false
    private var resolving = false
    private var settled = false

    /// 生产入口：预览走公开接口，兑换复用扫码同一接口，成功凭证交给扫码流程持有。
    convenience init(session: SessionStore, quickScan: QuickScanFlow, api: LinkEntryAPI? = nil) {
        let previewAPI = api ?? LinkEntryAPI(session: session)
        let startAPI = MatchStartAPI(session: session)
        self.init(
            currentIdentity: { session.identity },
            previewInvite: { try await previewAPI.preview($0) },
            resolveInvite: { try await startAPI.resolveMatchInviteCode($0) },
            handoffResolved: { identity, token, opponent in
                quickScan.handoffResolved(identity: identity, inviteToken: token, opponent: opponent)
            }
        )
    }

    init(
        currentIdentity: @escaping () -> SessionIdentity?,
        previewInvite: @escaping PreviewInvite,
        resolveInvite: @escaping ResolveInvite,
        handoffResolved: @escaping HandoffResolved
    ) {
        self.currentIdentity = currentIdentity
        self.previewInvite = previewInvite
        self.resolveInvite = resolveInvite
        self.handoffResolved = handoffResolved
    }

    /// 进入承接页时调用：已登录直接兑换，未登录先展示邀请人。
    func begin(_ code: String) async -> LinkEntryAction {
        inviteCode = code
        started = true
        if let identity = currentIdentity() {
            return await exchange(identity)
        }
        return await loadPreview()
    }

    /// 登录页返回或页面重新显示时调用；每次只兑换一次。
    func resume() async -> LinkEntryAction {
        guard started, !settled, !resolving else { return .ignore }
        guard let identity = currentIdentity() else { return .ignore }
        return await exchange(identity)
    }

    /// 网络失败后由页面「重试」触发：短码已确认失效或已交接时不再请求。
    func retry() async -> LinkEntryAction {
        guard started, !settled, !resolving else { return .ignore }
        return await begin(inviteCode)
    }

    private func loadPreview() async -> LinkEntryAction {
        do {
            let response = try await previewInvite(inviteCode)
            guard response.success, let preview = response.preview else {
                settled = true
                return .expired(message: Self.failureMessage(response.message))
            }
            return .awaitLogin(
                inviter: LinkEntryInviter(
                    name: preview.inviterName.isEmpty ? "球友" : preview.inviterName,
                    avatar: preview.inviterAvatar
                ),
                expiresAt: preview.expiresAt
            )
        } catch {
            if let message = Self.businessMessage(error) {
                settled = true
                return .expired(message: message)
            }
            // 网络或服务异常不代表短码失效：不结束流程，页面可以重试。
            return .error(message: Self.unavailableMessage)
        }
    }

    private func exchange(_ identity: SessionIdentity) async -> LinkEntryAction {
        guard !resolving, !settled else { return .ignore }
        resolving = true
        defer { resolving = false }

        do {
            let response = try await resolveInvite(inviteCode)
            // 迟到响应：登录期间换了账号或登出，直接丢弃，不交接凭证。
            guard isCurrent(identity) else { return .ignore }
            guard response.success,
                  let token = response.inviteToken, !token.isEmpty,
                  let preview = response.preview, preview.opponentId != 0 else {
                settled = true
                return .expired(message: Self.failureMessage(response.message))
            }
            let handed = handoffResolved(
                identity,
                token,
                MatchScanOpponent(
                    id: preview.opponentId,
                    nickname: preview.opponentName.isEmpty ? "对手" : preview.opponentName,
                    avatar: preview.opponentAvatar
                )
            )
            guard handed else { return .ignore }
            settled = true
            return .confirm
        } catch {
            guard isCurrent(identity) else { return .ignore }
            if let message = Self.businessMessage(error) {
                settled = true
                return .expired(message: message)
            }
            return .error(message: Self.unavailableMessage)
        }
    }

    private func isCurrent(_ identity: SessionIdentity) -> Bool {
        currentIdentity() == identity
    }

    private static func failureMessage(_ message: String?) -> String {
        guard let message, !message.isEmpty else { return expiredMessage }
        return message
    }

    /// 业务失败（HTTP 200 + success:false）沿用服务端文案；网络/服务异常才允许重试。
    private static func businessMessage(_ error: Error) -> String? {
        guard let apiError = error as? APIError, apiError.kind == .business else { return nil }
        return failureMessage(apiError.message)
    }
}

/// 入口链接在原生端的落点：认识的链接都先进入承接页，由承接页按类型分流。
enum LinkEntryRouting {
    static func route(for url: String) -> AppRoute? {
        switch EntryLink.parse(url) {
        case .other: nil
        case .home, .invite, .match: .linkEntry(url: url)
        }
    }

    /// 小程序对局链接是 `redirectTo(buildSpectateRoute(...))`，即观战详情而非实时对局页。
    static func landing(for result: EntryLinkResult) -> LinkEntryLanding {
        switch result {
        case .invite(let code): .invite(code: code)
        case .match(let matchId): .publicMatch(matchId: matchId)
        case .home, .other: .home
        }
    }
}

enum LinkEntryLanding: Equatable {
    case invite(code: String)
    case publicMatch(matchId: Int64)
    case home
}

/// 承接页的展示规则，对应 `linkEntry.vue` 里的剩余有效期计算。
enum LinkEntryRules {
    static func remainingText(expiresAt: String, now: Date) -> String {
        guard let date = parse(expiresAt) else { return "" }
        let remaining = date.timeIntervalSince(now) * 1000
        guard remaining.isFinite, remaining > 0 else { return "" }
        let minutes = Int((remaining / 60000).rounded(.up))
        return "\(minutes) 分钟内有效"
    }

    private static func parse(_ value: String) -> Date? {
        guard !value.isEmpty else { return nil }
        let formatter = ISO8601DateFormatter()
        formatter.formatOptions = [.withInternetDateTime, .withFractionalSeconds]
        if let date = formatter.date(from: value) { return date }
        formatter.formatOptions = [.withInternetDateTime]
        return formatter.date(from: value)
    }
}
