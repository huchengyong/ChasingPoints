import Foundation

enum CDNAsset {
    /// 这些资源存放在 CDN，后端仍返回 `/static/images/...` 形式的路径。
    private static let pattern = #"^/static/images/(?:(?:achievements|ranks|avatar-frames|gear)/[^/]+|(?:wst-)?default-event-cover\.png)$"#

    private static let regex = try! NSRegularExpression(pattern: pattern)

    /// 后端返回的 `/static/images/...` 路径是否属于 CDN 资源（规则与小程序一致）。
    static func isCDNPath(_ path: String) -> Bool {
        guard !path.isEmpty else { return false }
        let range = NSRange(path.startIndex..., in: path)
        return regex.firstMatch(in: path, range: range) != nil
    }

    /// CDN 路径 → CDN 地址；http(s) 绝对地址原样返回；其他（空串、小程序本地路径）返回 nil。
    static func url(for path: String) -> URL? {
        if isCDNPath(path) {
            return URL(string: AppConfiguration.cdnBaseURL + path)
        }
        if path.hasPrefix("http://") || path.hasPrefix("https://") {
            return URL(string: path)
        }
        return nil
    }
}
