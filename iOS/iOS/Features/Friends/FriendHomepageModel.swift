import Combine
import Foundation

/// 好友主页：一次读取目标好友的战绩列表，概览与战斗卡都从同一份数据推导。
@MainActor
final class FriendHomepageModel: ObservableObject {
    let target: FriendHomepageTarget
    let loader: ResourceLoader<FriendHomepageData>

    private let session: SessionStore
    private let now: () -> Date

    init(
        session: SessionStore,
        target: FriendHomepageTarget,
        api: FriendAPI? = nil,
        now: @escaping () -> Date = Date.init
    ) {
        self.session = session
        self.target = target
        self.now = now
        let api = api ?? FriendAPI(session: session)
        loader = ResourceLoader(session: session, now: now) {
            try await api.homepage(userId: target.userId)
        }
    }

    var data: FriendHomepageData? { loader.value }

    var subjectName: String {
        target.name.isEmpty ? "TA" : target.name
    }

    var summary: FriendHomepageSummary {
        FriendHomepageRules.summary(
            totalMatches: data?.totalMatches ?? 0,
            totalWins: data?.totalWins ?? 0,
            lastMatchAt: data?.lastMatchAt ?? "",
            now: now()
        )
    }

    var cards: [OpponentCard] {
        OpponentRecordRules.cards(
            opponents: data?.opponents ?? [],
            subjectName: subjectName,
            currentUserId: session.user?.id ?? 0,
            showWinRateLabel: true
        )
    }

    var hidden: Bool { data?.hidden ?? false }

    func onAppear() async {
        await loader.loadIfNeeded()
    }

    func identityChanged() async {
        loader.syncIdentity()
        await loader.loadIfNeeded()
    }

    func retry() async {
        await loader.retry()
    }

    func refresh() async {
        await loader.refresh()
    }
}
