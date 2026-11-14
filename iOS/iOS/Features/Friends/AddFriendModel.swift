import Combine
import Foundation

/// 添加好友：搜索昵称/手机号，结果状态本地覆盖，发送成功后置为已发送。
@MainActor
final class AddFriendModel: ObservableObject {
    enum Phase: Equatable {
        case idle
        case searching
        case loaded
        case empty
        case failed(String)
    }

    @Published var keyword = ""
    @Published private(set) var results: [SearchUserItem] = []
    @Published private(set) var phase: Phase = .idle
    @Published private(set) var hasSearched = false
    @Published private(set) var failure: LoadFailure?
    @Published private(set) var busyIDs: Set<Int64> = []
    @Published private(set) var statuses: [Int64: FriendSearchStatus] = [:]
    @Published private(set) var notice: String?

    private let session: SessionStore
    private let api: FriendAPI
    private var requestToken = 0

    init(session: SessionStore, api: FriendAPI? = nil) {
        self.session = session
        self.api = api ?? FriendAPI(session: session)
    }

    func status(of item: SearchUserItem) -> FriendSearchStatus {
        statuses[item.userId] ?? FriendRules.searchStatus(
            isFriend: item.isFriend,
            hasPendingRequest: item.hasPendingRequest
        )
    }

    func isBusy(_ item: SearchUserItem) -> Bool {
        busyIDs.contains(item.userId)
    }

    func search() async {
        let trimmed = keyword.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !trimmed.isEmpty else {
            notice = "请输入搜索内容"
            return
        }
        guard let identity = session.identity else {
            phase = .failed("登录状态已失效")
            failure = LoadFailure(category: .session, message: "登录状态已失效")
            return
        }
        requestToken += 1
        let token = requestToken
        hasSearched = true
        phase = .searching
        failure = nil
        do {
            let response = try await api.search(keyword: trimmed)
            guard token == requestToken, session.identity == identity else { return }
            results = response.list ?? []
            statuses = [:]
            phase = results.isEmpty ? .empty : .loaded
        } catch {
            guard token == requestToken, session.identity == identity else { return }
            guard !isCancellation(error) else {
                phase = .idle
                return
            }
            phase = .failed(error.localizedDescription)
            failure = LoadFailure(error)
        }
    }

    func retry() async {
        await search()
    }

    func clear() {
        keyword = ""
        hasSearched = false
        results = []
        statuses = [:]
        failure = nil
        phase = .idle
        // 让在途搜索的响应失效，避免清空后又被写回。
        requestToken += 1
    }

    func sendRequest(to item: SearchUserItem) async -> FriendActionOutcome {
        let userId = item.userId
        guard userId > 0, !busyIDs.contains(userId), let identity = session.identity else {
            return .failure("发送失败")
        }
        busyIDs.insert(userId)
        defer { busyIDs.remove(userId) }
        do {
            _ = try await api.sendRequest(toUserId: userId)
            guard session.identity == identity else { return .failure("发送失败") }
            statuses[userId] = .pending
            return .success
        } catch {
            return .failure(error.localizedDescription)
        }
    }

    func clearNotice() {
        notice = nil
    }

    private func isCancellation(_ error: Error) -> Bool {
        if let apiError = error as? APIError, apiError.kind == .canceled { return true }
        return error is CancellationError
    }
}
