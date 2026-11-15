import Combine
import Foundation

/// 通知管理页的偏好读取与保存，对应小程序 `notification.vue`。
@MainActor
final class NotificationPreferencesModel: ObservableObject {
    enum Phase: Equatable {
        case idle
        case loading
        case ready
        case failed
    }

    enum Preference: CaseIterable, Hashable {
        case matchResult
        case friendRequest
        case challenge
        case tournament
        case follow

        var title: String {
            switch self {
            case .matchResult: "对局结果"
            case .friendRequest: "好友申请"
            case .challenge: "约球提醒"
            case .tournament: "赛事报名提醒"
            case .follow: "新关注提醒"
            }
        }

        var key: String {
            switch self {
            case .matchResult: "match_result_enabled"
            case .friendRequest: "friend_request_enabled"
            case .challenge: "challenge_enabled"
            case .tournament: "tournament_enabled"
            case .follow: "follow_enabled"
            }
        }
    }

    struct Preferences: Equatable {
        var matchResult = true
        var friendRequest = true
        var challenge = true
        var tournament = true
        var follow = true

        func value(for preference: Preference) -> Bool {
            switch preference {
            case .matchResult: matchResult
            case .friendRequest: friendRequest
            case .challenge: challenge
            case .tournament: tournament
            case .follow: follow
            }
        }

        mutating func set(_ value: Bool, for preference: Preference) {
            switch preference {
            case .matchResult: matchResult = value
            case .friendRequest: friendRequest = value
            case .challenge: challenge = value
            case .tournament: tournament = value
            case .follow: follow = value
            }
        }
    }

    private struct Payload: Codable {
        let matchResultEnabled: Bool
        let friendRequestEnabled: Bool
        let challengeEnabled: Bool
        let tournamentEnabled: Bool
        let followEnabled: Bool
    }

    private struct Response: Decodable {
        let success: Bool?
        let message: String?
        let matchResultEnabled: Bool
        let friendRequestEnabled: Bool
        let challengeEnabled: Bool
        let tournamentEnabled: Bool
        let followEnabled: Bool

        var preferences: Preferences {
            Preferences(
                matchResult: matchResultEnabled,
                friendRequest: friendRequestEnabled,
                challenge: challengeEnabled,
                tournament: tournamentEnabled,
                follow: followEnabled
            )
        }
    }

    @Published private(set) var phase: Phase = .idle
    @Published private(set) var preferences = Preferences()
    @Published private(set) var errorMessage: String?
    @Published private(set) var isSaving = false

    let showFollowPreference = AppFlags.canShowSocialPublishing

    private let session: SessionStore

    init(session: SessionStore) {
        self.session = session
    }

    func load() async {
        guard phase != .loading else { return }
        guard let identity = session.identity else {
            reset()
            return
        }

        phase = .loading
        errorMessage = nil
        do {
            let response: Response = try await session.authorizedRequest("/api/notification/preferences")
            guard session.identity == identity else { return }
            guard response.success != false else {
                throw APIError(message: response.message ?? "加载通知设置失败", kind: .business)
            }
            preferences = response.preferences
            phase = .ready
        } catch {
            guard session.identity == identity else { return }
            errorMessage = error.localizedDescription
            phase = .failed
        }
    }

    func retry() async {
        await load()
    }

    func identityChanged() async {
        reset()
        if session.identity != nil {
            await load()
        }
    }

    /// 保存整份偏好；失败回滚到保存前的值并返回 Toast 文案。
    func toggle(_ preference: Preference, value: Bool) async -> String? {
        guard !isSaving, phase == .ready else { return nil }
        guard let identity = session.identity else { return nil }

        let previous = preferences
        preferences.set(value, for: preference)
        isSaving = true
        defer { isSaving = false }

        do {
            let response: Response = try await session.authorizedPost(
                "/api/notification/preferences",
                body: payload(from: preferences)
            )
            guard session.identity == identity else { return nil }
            guard response.success != false else {
                throw APIError(message: response.message ?? "保存失败，请重试", kind: .business)
            }
            preferences = response.preferences
            return nil
        } catch {
            guard session.identity == identity else { return nil }
            preferences = previous
            return "保存失败，请重试"
        }
    }

    private func payload(from preferences: Preferences) -> Payload {
        Payload(
            matchResultEnabled: preferences.matchResult,
            friendRequestEnabled: preferences.friendRequest,
            challengeEnabled: preferences.challenge,
            tournamentEnabled: preferences.tournament,
            followEnabled: preferences.follow
        )
    }

    private func reset() {
        phase = .idle
        preferences = Preferences()
        errorMessage = nil
        isSaving = false
    }
}
