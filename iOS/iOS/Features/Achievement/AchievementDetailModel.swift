import Combine
import Foundation

/// 成就详情：一次直读，只采纳最新请求，结果绑定发起时的身份。
@MainActor
final class AchievementDetailModel: ObservableObject {
    @Published private(set) var achievement: AchievementDef?
    @Published private(set) var isLoading = false
    @Published private(set) var failure: LoadFailure?

    let achievementId: Int64

    private let session: SessionStore
    private let api: AchievementAPI
    private var owner: SessionIdentity?
    private var requestToken = 0

    init(session: SessionStore, achievementId: Int64) {
        self.session = session
        self.api = AchievementAPI(session: session)
        self.achievementId = achievementId
        self.owner = session.identity
    }

    // MARK: - 展示值

    var hasLoaded: Bool { achievement != nil }
    var isInitialLoading: Bool { isLoading && achievement == nil }
    var isInitialError: Bool { achievement == nil && failure != nil && !isLoading && session.identity != nil }

    var failureFeedback: LoadFailure.Feedback? {
        failure?.feedback(resource: "成就详情")
    }

    /// 权限与不存在不重试，直接返回上一页（对应小程序 detail.vue 的错误动作）。
    var dismissesOnError: Bool {
        guard let failure else { return false }
        return failure.category == .permission || failure.category == .notFound
    }

    var progressPercent: Int {
        guard let achievement else { return 0 }
        return AchievementPageRules.detailProgressPercent(
            progress: achievement.progress,
            threshold: achievement.threshold,
            unlocked: achievement.unlocked
        )
    }

    var progressText: String {
        guard let achievement else { return "0 / 0" }
        return "\(String(max(0, achievement.progress))) / \(String(max(0, achievement.threshold)))"
    }

    var categoryLabel: String {
        AchievementPageRules.categoryLabel(achievement?.category)
    }

    var gameTypeLabel: String {
        AchievementPageRules.gameTypeLabel(achievement?.gameType ?? 0)
    }

    var fallbackEmoji: String {
        achievement.map(AchievementPageRules.fallbackEmoji) ?? "🎯"
    }

    // MARK: - 读取

    func loadIfNeeded() async {
        syncIdentity()
        guard session.identity != nil, achievement == nil, !isLoading else { return }
        await fetch()
    }

    func retry() async {
        syncIdentity()
        guard session.identity != nil, !isLoading else { return }
        await fetch()
    }

    /// 身份变化时作废在途请求并清空页面；相同身份不做任何事。
    func syncIdentity() {
        let current = session.identity
        guard current != owner else { return }
        owner = current
        requestToken += 1
        achievement = nil
        isLoading = false
        failure = nil
    }

    private func fetch() async {
        guard let requestOwner = session.identity else { return }
        requestToken += 1
        let token = requestToken
        isLoading = true
        failure = nil

        do {
            let response = try await api.detail(achievementId: achievementId)
            guard token == requestToken, session.identity == requestOwner else { return }
            achievement = response.achievement
            isLoading = false
        } catch {
            guard token == requestToken, session.identity == requestOwner else { return }
            isLoading = false
            if isCancellation(error) { return }
            let loadFailure = LoadFailure(error)
            // 全局会话失效流程已接管重新登录，页面不重复展示失败态。
            guard loadFailure.category != .session else { return }
            failure = loadFailure
        }
    }

    private func isCancellation(_ error: Error) -> Bool {
        if let apiError = error as? APIError, apiError.kind == .canceled { return true }
        return error is CancellationError
    }
}
