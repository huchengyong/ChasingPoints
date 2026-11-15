import Combine
import Foundation

/// 设置页的隐私开关读取与保存，对应小程序 `loadUserPrivacy` / `toggleHideMatch` /
/// `toggleFriendsOnlyChallenges`。Toast 文案由方法返回，页面负责展示。
@MainActor
final class PrivacySettingsModel: ObservableObject {
    enum Phase: Equatable {
        case idle
        case loading
        case ready
        case failed
    }

    @Published private(set) var phase: Phase = .idle
    @Published private(set) var hideMatchRecord = false
    @Published private(set) var friendsOnlyChallenges = false
    @Published private(set) var isSavingHideMatch = false
    @Published private(set) var isSavingFriendsOnly = false

    private struct PrivacyResponse: Decodable {
        let success: Bool?
        let hideMatchRecord: Bool
        let friendsOnlyChallenges: Bool
    }

    private struct UpdatePrivacyRequest: Encodable {
        let hideMatchRecord: Bool?
        let friendsOnlyChallenges: Bool?
    }

    private struct UpdatePrivacyResponse: Decodable {
        let success: Bool
        let message: String?
        let hideMatchRecord: Bool
        let friendsOnlyChallenges: Bool
    }

    private let session: SessionStore

    init(session: SessionStore) {
        self.session = session
    }

    func load(force: Bool = false) async {
        guard phase != .loading else { return }
        if !force, phase == .ready { return }
        guard let identity = session.identity else {
            reset()
            return
        }

        phase = .loading
        do {
            let response: PrivacyResponse = try await session.authorizedRequest("/api/user/privacy")
            guard session.identity == identity else { return }
            guard response.success != false else {
                throw APIError(message: "读取隐私设置失败", kind: .business)
            }
            hideMatchRecord = response.hideMatchRecord
            friendsOnlyChallenges = response.friendsOnlyChallenges
            phase = .ready
        } catch {
            guard session.identity == identity else { return }
            phase = .failed
        }
    }

    /// 退出、切号或会话失效后清空旧状态；新身份登录后重新读取。
    func identityChanged() async {
        reset()
        if session.identity != nil {
            await load(force: true)
        }
    }

    func toggleHideMatch() async -> String? {
        guard !isSavingHideMatch else { return nil }
        guard phase == .ready else {
            await load(force: true)
            return nil
        }
        guard let identity = session.identity else { return nil }

        let previousValue = hideMatchRecord
        let nextValue = !previousValue
        hideMatchRecord = nextValue
        isSavingHideMatch = true
        defer { isSavingHideMatch = false }

        do {
            let response: UpdatePrivacyResponse = try await session.authorizedPost(
                "/api/user/privacy",
                body: UpdatePrivacyRequest(hideMatchRecord: nextValue, friendsOnlyChallenges: nil)
            )
            guard session.identity == identity else { return nil }
            guard response.success else {
                throw APIError(message: response.message ?? "更新失败", kind: .business)
            }
            hideMatchRecord = response.hideMatchRecord
            return response.hideMatchRecord ? "已隐藏战绩" : "已公开战绩"
        } catch {
            guard session.identity == identity else { return nil }
            hideMatchRecord = previousValue
            return message(for: error, fallback: "更新失败")
        }
    }

    func toggleFriendsOnlyChallenges() async -> String? {
        guard !isSavingFriendsOnly else { return nil }
        guard phase == .ready else {
            await load(force: true)
            return nil
        }
        guard let identity = session.identity else { return nil }

        isSavingFriendsOnly = true
        defer { isSavingFriendsOnly = false }

        do {
            let response: UpdatePrivacyResponse = try await session.authorizedPost(
                "/api/user/privacy",
                body: UpdatePrivacyRequest(hideMatchRecord: nil, friendsOnlyChallenges: !friendsOnlyChallenges)
            )
            guard session.identity == identity else { return nil }
            guard response.success else {
                throw APIError(message: response.message ?? "保存失败，请重试", kind: .business)
            }
            friendsOnlyChallenges = response.friendsOnlyChallenges
            return response.friendsOnlyChallenges ? "已开启仅好友约球" : "已允许非好友约球"
        } catch {
            guard session.identity == identity else { return nil }
            return "保存失败，请重试"
        }
    }

    private func reset() {
        phase = .idle
        hideMatchRecord = false
        friendsOnlyChallenges = false
        isSavingHideMatch = false
        isSavingFriendsOnly = false
    }

    private func message(for error: Error, fallback: String) -> String {
        let text = error.localizedDescription.trimmingCharacters(in: .whitespacesAndNewlines)
        return text.isEmpty ? fallback : text
    }
}
