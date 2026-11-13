import Combine
import Foundation

/// 球馆详情：公开读取 + 需登录的签到。
@MainActor
final class VenueDetailModel: ObservableObject {
    enum CheckinResult: Equatable {
        case success
        case failed
        case requiresLogin
        case ignored
    }

    let venueId: Int64
    let loader: ResourceLoader<VenueInfo>

    @Published private(set) var hasCheckedIn = false
    @Published private(set) var isCheckingIn = false
    @Published private(set) var notice: String?

    private let session: SessionStore
    private let api: VenueAPI
    private var owner: SessionIdentity?

    init(session: SessionStore, venueId: Int64, api: VenueAPI? = nil) {
        self.venueId = venueId
        let api = api ?? VenueAPI(session: session)
        self.api = api
        self.session = session
        loader = ResourceLoader(session: session, access: .anyone) {
            try await api.detail(id: venueId)
        }
    }

    var venue: VenueInfo? { loader.value }

    func loadIfNeeded() async {
        syncIdentity()
        await loader.loadIfNeeded()
    }

    func retry() async {
        await loader.retry()
    }

    /// 退出或切号后丢弃签到状态。
    func syncIdentity() {
        let current = session.identity
        guard current != owner else { return }
        owner = current
        hasCheckedIn = false
        loader.syncIdentity()
    }

    func checkin() async -> CheckinResult {
        guard !isCheckingIn else { return .ignored }
        guard session.identity != nil else { return .requiresLogin }
        if hasCheckedIn {
            notice = "今日已签到"
            return .ignored
        }

        let requestOwner = session.identity
        isCheckingIn = true
        defer { isCheckingIn = false }

        do {
            _ = try await api.checkin(venueId: venueId)
            guard session.identity == requestOwner else { return .ignored }
            hasCheckedIn = true
            notice = "签到成功！"
            await loader.refresh()
            return .success
        } catch {
            guard session.identity == requestOwner else { return .ignored }
            notice = "签到失败"
            return .failed
        }
    }

    func clearNotice() {
        notice = nil
    }
}
