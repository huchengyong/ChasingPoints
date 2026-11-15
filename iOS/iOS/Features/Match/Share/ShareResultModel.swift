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
    private let loadImage: (URL) async -> UIImage?
    private let render: (SharePosterPayload, SharePosterAssets) -> UIImage?
    private var requestToken = 0

    convenience init(session: SessionStore, matchId: Int64) {
        self.init(
            session: session,
            matchId: matchId,
            api: ShareResultAPI(session: session),
            loadImage: { url in await SharePosterImageDownloader.load(url) },
            render: { payload, assets in
                let qr = QRCodeImage.make(content: payload.qrURL, size: 120)
                return ShareResultPoster.render(payload, qrImage: qr, assets: assets)
            }
        )
    }

    init(
        session: SessionStore,
        matchId: Int64,
        api: ShareResultAPI,
        loadImage: @escaping (URL) async -> UIImage?,
        render: @escaping (SharePosterPayload, SharePosterAssets) -> UIImage?
    ) {
        self.session = session
        self.matchId = matchId
        self.api = api
        self.loadImage = loadImage
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

        let payload: SharePosterPayload
        do {
            let data = try await api.matchShareData(matchId: matchId)
            guard token == requestToken, session.identity == requestOwner else { return }
            payload = ShareResultRules.payload(from: data)
        } catch {
            guard token == requestToken, session.identity == requestOwner else { return }
            payload = ShareResultRules.fallbackPayload()
        }

        // 会员模板需要远程头像与头像框；下载期间身份变化就丢弃这次结果。
        let assets = await loadAssets(for: payload)
        guard token == requestToken, session.identity == requestOwner else { return }

        let image = render(payload, assets)
        self.payload = payload
        posterImage = image
        phase = image == nil ? .failed : .ready
    }

    private func loadAssets(for payload: SharePosterPayload) async -> SharePosterAssets {
        guard payload.isMember else { return .empty }
        // 下载失败退回本地默认头像；头像框失败就不画，海报照常生成。
        let fallbackAvatar = UIImage(named: "DefaultAvatar01")
        // 三张图并行下载，弱网下最多等一张的超时，而不是三张累加。
        async let myAvatar = download(payload.myAvatar)
        async let opponentAvatar = download(payload.opponentAvatar)
        async let frame = download(payload.myIdentity.frameURL?.absoluteString ?? "")
        return SharePosterAssets(
            myAvatar: await myAvatar ?? fallbackAvatar,
            opponentAvatar: await opponentAvatar ?? fallbackAvatar,
            frame: await frame
        )
    }

    private func download(_ value: String) async -> UIImage? {
        // 与 AvatarView 一致：只有 http(s) 地址才下载，本地资源名和相对路径直接跳过。
        guard value.hasPrefix("http://") || value.hasPrefix("https://"),
              let url = URL(string: value) else { return nil }
        return await loadImage(url)
    }
}
