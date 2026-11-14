import Combine
import Foundation
import UIKit

/// 战绩海报页，对应小程序 `subPages/match/shareResult.vue`。
@MainActor
final class ShareResultModel: ObservableObject {
    enum Phase: Equatable {
        case loading
        case ready
        case failed
    }

    @Published private(set) var phase: Phase = .loading
    @Published private(set) var payload: SharePosterPayload?
    @Published private(set) var posterImage: UIImage?

    let matchId: Int64

    private let session: SessionStore
    private let api: ShareResultAPI
    private let render: (SharePosterPayload) -> UIImage?
    private var requestToken = 0

    convenience init(session: SessionStore, matchId: Int64) {
        self.init(
            session: session,
            matchId: matchId,
            api: ShareResultAPI(session: session),
            render: { payload in
                let qr = QRCodeImage.make(content: payload.qrURL, size: 120)
                return ShareResultPoster.render(payload, qrImage: qr)
            }
        )
    }

    init(
        session: SessionStore,
        matchId: Int64,
        api: ShareResultAPI,
        render: @escaping (SharePosterPayload) -> UIImage?
    ) {
        self.session = session
        self.matchId = matchId
        self.api = api
        self.render = render
    }

    /// 无论接口成功还是失败都渲染一张海报：失败走降级数据，只有渲染失败才展示「海报生成失败」。
    func load() async {
        requestToken += 1
        let token = requestToken
        let requestOwner = session.identity
        phase = .loading
        posterImage = nil
        payload = nil

        do {
            let data = try await api.matchShareData(matchId: matchId)
            guard token == requestToken, session.identity == requestOwner else { return }
            apply(ShareResultRules.payload(from: data))
        } catch {
            guard token == requestToken, session.identity == requestOwner else { return }
            apply(ShareResultRules.fallbackPayload())
        }
    }

    private func apply(_ payload: SharePosterPayload) {
        let image = render(payload)
        self.payload = payload
        posterImage = image
        phase = image == nil ? .failed : .ready
    }
}
