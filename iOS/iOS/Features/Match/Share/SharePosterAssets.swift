import UIKit

/// 海报渲染前已经下载好的远程图片；会员模板缺哪张就少画哪张，不阻塞整张海报。
struct SharePosterAssets {
    let myAvatar: UIImage?
    let opponentAvatar: UIImage?
    let frame: UIImage?

    static let empty = SharePosterAssets(myAvatar: nil, opponentAvatar: nil, frame: nil)
}

enum SharePosterImageDownloader {
    /// 单张 5 秒超时；失败返回 nil，由调用方退回默认头像或跳过头像框。
    static func load(_ url: URL) async -> UIImage? {
        var request = URLRequest(url: url)
        request.timeoutInterval = 5
        guard let (data, response) = try? await URLSession.shared.data(for: request),
              (response as? HTTPURLResponse).map({ (200..<300).contains($0.statusCode) }) ?? true else {
            return nil
        }
        return UIImage(data: data)
    }
}
