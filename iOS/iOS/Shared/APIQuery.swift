import Foundation

enum APIQuery {
    /// 生成带查询串的路径：nil 值跳过，键按字典序排列，值做百分号编码。
    static func path(_ path: String, _ items: [String: CustomStringConvertible?]) -> String {
        let pairs = items.compactMap { key, value -> (key: String, value: String)? in
            guard let value else { return nil }
            return (key, encode(String(describing: value)))
        }
        .sorted { $0.key < $1.key }
        guard !pairs.isEmpty else { return path }
        let query = pairs.map { "\(encode($0.key))=\($0.value)" }.joined(separator: "&")
        return "\(path)?\(query)"
    }

    /// 只保留 RFC 3986 的非保留字符；`URLComponents` 不会编码 `+`，必须自己处理。
    private static let unreserved = CharacterSet(
        charactersIn: "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-._~"
    )

    private static func encode(_ value: String) -> String {
        value.addingPercentEncoding(withAllowedCharacters: unreserved) ?? value
    }
}
