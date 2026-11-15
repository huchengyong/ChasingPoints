import Combine
import Foundation

/// 附近 / 全部球馆列表状态。定位失败时降级到全部球馆并给出提示。
@MainActor
final class VenueListModel: ObservableObject {
    enum Mode: CaseIterable, Equatable {
        case nearby
        case list

        var title: String {
            switch self {
            case .nearby: "附近球馆"
            case .list: "全部球馆"
            }
        }
    }

    enum Phase: Equatable {
        case idle
        case loading
        case loaded
        case empty
        case failed(LoadFailure)
    }

    @Published private(set) var mode: Mode = .nearby
    @Published private(set) var items: [VenueInfo] = []
    @Published private(set) var phase: Phase = .idle
    /// 定位栏文案；空字符串时界面显示「定位中...」。
    @Published private(set) var locationText = ""
    @Published private(set) var isRefreshing = false
    @Published private(set) var isLoadingMore = false
    @Published private(set) var refreshError: String?
    @Published private(set) var loadMoreError: String?
    @Published private(set) var hasMore = false
    /// 需要以轻提示告知用户的定位结果。
    @Published private(set) var notice: String?

    private enum FetchKind {
        case first
        case refresh
        case more
    }

    private let session: SessionStore
    private let location: LocationProviding
    private let api: VenueAPI
    private var coordinate: Coordinate?
    private var page = 1
    private var total = 0
    private var owner: SessionIdentity?
    private var requestToken = 0
    private var isLocating = false

    init(session: SessionStore, location: LocationProviding? = nil, api: VenueAPI? = nil) {
        self.session = session
        self.location = location ?? SystemLocationProvider()
        self.api = api ?? VenueAPI(session: session)
    }

    var emptyTitle: String {
        mode == .nearby ? "附近暂无球馆" : "暂无球馆数据"
    }

    /// 身份变化时清空列表；定位结果与访问者无关，保留。
    func syncIdentity() {
        let current = session.identity
        guard current != owner else { return }
        owner = current
        requestToken += 1
        items = []
        total = 0
        page = 1
        hasMore = false
        phase = .idle
        isRefreshing = false
        isLoadingMore = false
        refreshError = nil
        loadMoreError = nil
    }

    func loadIfNeeded() async {
        syncIdentity()
        guard phase == .idle else { return }
        await fetch(.refresh)
    }

    func refresh() async {
        syncIdentity()
        guard !isBusy else { return }
        await fetch(.refresh)
    }

    func retry() async {
        syncIdentity()
        guard !isBusy else { return }
        await fetch(.refresh)
    }

    func loadMore() async {
        syncIdentity()
        guard mode == .list, hasMore, !isBusy else { return }
        await fetch(.more)
    }

    func switchMode(_ newMode: Mode) async {
        guard mode != newMode else { return }
        mode = newMode
        requestToken += 1
        items = []
        total = 0
        page = 1
        hasMore = false
        phase = .idle
        isRefreshing = false
        isLoadingMore = false
        refreshError = nil
        loadMoreError = nil
        await fetch(.refresh)
    }

    /// 点击定位栏：重新取位置并更新文案，不主动重读列表。
    func requestCurrentLocation() async {
        await resolveLocation()
    }

    func clearNotice() {
        notice = nil
    }

    private var isBusy: Bool {
        phase == .loading || isRefreshing || isLoadingMore
    }

    private func fetch(_ kind: FetchKind) async {
        requestToken += 1
        let token = requestToken
        let requestOwner = session.identity
        let requestedPage = kind == .more ? page + 1 : 1

        switch kind {
        case .first, .refresh:
            if items.isEmpty {
                phase = .loading
            } else {
                isRefreshing = true
            }
        case .more:
            isLoadingMore = true
        }
        refreshError = nil
        loadMoreError = nil

        do {
            if mode == .nearby, coordinate == nil {
                await resolveLocation()
                guard token == requestToken else { return }
            }

            let result: VenuePage
            if mode == .nearby, let coordinate {
                result = try await api.nearby(coordinate)
            } else {
                result = try await api.list(page: requestedPage, coordinate: coordinate)
            }
            guard token == requestToken, session.identity == requestOwner else { return }

            switch kind {
            case .first, .refresh:
                items = result.items
            case .more:
                items += result.items
            }
            page = requestedPage
            total = result.total
            // 附近接口一次返回全部结果，不再分页。
            hasMore = mode == .list && items.count < total
            phase = items.isEmpty ? .empty : .loaded
            isRefreshing = false
            isLoadingMore = false
        } catch {
            guard token == requestToken, session.identity == requestOwner else { return }
            isRefreshing = false
            isLoadingMore = false
            if let apiError = error as? APIError, apiError.kind == .canceled {
                if kind != .more, items.isEmpty { phase = .idle }
                return
            }
            switch kind {
            case .first, .refresh:
                if items.isEmpty {
                    phase = .failed(LoadFailure(error))
                } else {
                    refreshError = error.localizedDescription
                }
            case .more:
                loadMoreError = error.localizedDescription
            }
        }
    }

    /// 取当前位置；失败时切换为全部球馆并提示，与小程序 getLocation 一致。
    private func resolveLocation() async {
        guard !isLocating else { return }
        isLocating = true
        defer { isLocating = false }

        do {
            coordinate = try await location.currentLocation()
            locationText = "已获取位置"
        } catch {
            let denied = (error as? LocationError) == .denied
            locationText = denied ? "未授权定位" : "定位失败"
            notice = denied ? "定位权限未开启，将仅展示全部球馆" : "定位失败，将仅展示全部球馆"
            if mode == .nearby {
                mode = .list
            }
        }
    }
}
