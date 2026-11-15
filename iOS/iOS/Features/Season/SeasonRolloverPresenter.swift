import Combine
import Foundation
import SwiftUI

/// 换季提醒弹窗内容；报告入口只能来自服务端给出的合法回顾路径。
struct SeasonRolloverAlert: Equatable {
    let title: String
    let content: String
    let confirmText: String
    let cancelText: String
    let showCancel: Bool
    let reportRoute: AppRoute?
}

/// 展示最近一条未读换季通知：同一时刻只弹一次，关闭后才标已读。
///
/// 对应小程序 `utils/honor-wall.js` 的 `presentLatestSeasonRollover`。
@MainActor
final class SeasonRolloverPresenter: ObservableObject {
    @Published private(set) var alert: SeasonRolloverAlert?

    private let session: SessionStore
    private let api: AchievementAPI
    private var isPresenting = false
    private var resume: CheckedContinuation<Bool, Never>?

    init(session: SessionStore) {
        self.session = session
        self.api = AchievementAPI(session: session)
    }

    /// 优先使用调用方已缓存的未读通知；没有则自己读通知列表。
    func present(
        notification: NotificationInfo? = nil,
        onRead: (() -> Void)? = nil,
        onNavigate: ((AppRoute) -> Void)? = nil
    ) async {
        guard !isPresenting, let requestOwner = session.identity else { return }
        isPresenting = true
        defer { isPresenting = false }

        let target: NotificationInfo?
        if let notification {
            target = notification
        } else {
            target = try? await loadLatestUnread()
        }
        // 读取期间退出或切号时，旧账号的提醒不能弹给新身份。
        guard let target, session.identity == requestOwner else { return }

        var rollover = HonorWallRules.seasonRollover(target, allowReport: onNavigate != nil)
        let route = reportRoute(rollover.reportURL)
        // 回顾地址解析不出报告页时不能给「查看回顾」，否则点了没有去处。
        if route == nil, onNavigate != nil {
            rollover = HonorWallRules.seasonRollover(target, allowReport: false)
        }

        let confirmed = await withCheckedContinuation { continuation in
            resume = continuation
            alert = SeasonRolloverAlert(
                title: rollover.title,
                content: rollover.content,
                confirmText: rollover.confirmText,
                cancelText: rollover.cancelText,
                showCancel: rollover.showCancel,
                reportRoute: route
            )
        }
        alert = nil
        resume = nil

        guard session.identity == requestOwner else { return }
        try? await api.markNotificationRead(notificationId: target.id)
        guard session.identity == requestOwner else { return }
        onRead?()
        if confirmed, let route {
            onNavigate?(route)
        }
    }

    /// 关闭弹窗，携带用户是否点了确认；重复调用无效果。
    func respond(confirmed: Bool) {
        guard let continuation = resume else { return }
        resume = nil
        alert = nil
        continuation.resume(returning: confirmed)
    }

    private func loadLatestUnread() async throws -> NotificationInfo? {
        let list = try await api.latestSeasonRolloverNotifications()
        return HonorWallRules.latestUnreadSeasonRollover(list)
    }

    /// `/subPages/season/report?id=..&game_type=..` → 原生赛季报告路由。
    private func reportRoute(_ url: String) -> AppRoute? {
        guard let components = URLComponents(string: url),
              components.path == HonorWallRules.seasonReportPath,
              let idValue = components.queryItems?.first(where: { $0.name == "id" })?.value,
              let seasonId = Int64(idValue), seasonId > 0 else { return nil }
        let gameType = components.queryItems?
            .first(where: { $0.name == "game_type" })?
            .value
            .flatMap(Int.init) ?? 3
        return .seasonReport(
            seasonId: seasonId,
            gameType: GameTypeCatalog.isSupported(gameType) ? gameType : 3
        )
    }
}

extension View {
    /// 换季提醒弹窗：关闭后才标已读，确认且有回顾入口时跳转。
    func seasonRolloverAlert(_ presenter: SeasonRolloverPresenter) -> some View {
        modifier(SeasonRolloverAlertModifier(presenter: presenter))
    }
}

private struct SeasonRolloverAlertModifier: ViewModifier {
    @ObservedObject var presenter: SeasonRolloverPresenter

    func body(content: Content) -> some View {
        content.alert(
            presenter.alert?.title ?? "",
            isPresented: Binding(
                get: { presenter.alert != nil },
                set: { presented in
                    if !presented { presenter.respond(confirmed: false) }
                }
            )
        ) {
            if presenter.alert?.showCancel == true {
                Button(presenter.alert?.cancelText ?? "知道了", role: .cancel) {
                    presenter.respond(confirmed: false)
                }
            }
            Button(presenter.alert?.confirmText ?? "知道了") {
                presenter.respond(confirmed: true)
            }
        } message: {
            Text(presenter.alert?.content ?? "")
        }
    }
}
