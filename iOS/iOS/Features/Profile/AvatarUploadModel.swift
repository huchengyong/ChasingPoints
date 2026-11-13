import Combine
import Foundation
import UIKit

protocol AvatarUploading {
    func upload(_ data: Data, to url: URL, token: String, key: String) async throws -> String
}

/// 七牛第三方上传：multipart/form-data，不带 Authorization。
struct QiniuAvatarUploader: AvatarUploading {
    private let session: URLSession

    init(session: URLSession = .shared) {
        self.session = session
    }

    func upload(_ data: Data, to url: URL, token: String, key: String) async throws -> String {
        let boundary = "Boundary-\(UUID().uuidString)"
        var request = URLRequest(url: url, cachePolicy: .reloadIgnoringLocalCacheData, timeoutInterval: 60)
        request.httpMethod = "POST"
        request.setValue("multipart/form-data; boundary=\(boundary)", forHTTPHeaderField: "Content-Type")
        request.httpBody = Self.multipartBody(data: data, token: token, key: key, boundary: boundary)

        let (responseData, rawResponse) = try await session.data(for: request)
        guard let response = rawResponse as? HTTPURLResponse,
              (200..<300).contains(response.statusCode) else {
            throw APIError(message: "上传失败", kind: .server)
        }
        guard let payload = try? JSONSerialization.jsonObject(with: responseData) as? [String: Any] else {
            throw APIError(message: "上传结果解析失败", kind: .invalidResponse)
        }
        return payload["key"] as? String ?? ""
    }

    private static func multipartBody(data: Data, token: String, key: String, boundary: String) -> Data {
        var body = Data()
        let filename = key.split(separator: "/").last.map(String.init) ?? "avatar.jpg"

        func append(_ text: String) {
            body.append(Data(text.utf8))
        }

        append("--\(boundary)\r\n")
        append("Content-Disposition: form-data; name=\"token\"\r\n\r\n")
        append("\(token)\r\n")
        append("--\(boundary)\r\n")
        append("Content-Disposition: form-data; name=\"key\"\r\n\r\n")
        append("\(key)\r\n")
        append("--\(boundary)\r\n")
        append("Content-Disposition: form-data; name=\"file\"; filename=\"\(filename)\"\r\n")
        append("Content-Type: image/jpeg\r\n\r\n")
        body.append(data)
        append("\r\n--\(boundary)--\r\n")
        return body
    }
}

enum AvatarImageProcessing {
    /// 最长边缩到不超过 512px（不放大），再按 0.8 质量编码 JPEG。
    static func jpegData(
        from data: Data,
        maxEdge: Int = SettingsProfileRules.avatarImageMaxEdge,
        quality: Double = Double(SettingsProfileRules.avatarImageQuality) / 100
    ) -> Data? {
        guard let image = UIImage(data: data) else { return nil }

        let pixelWidth = Int(image.size.width * image.scale)
        let pixelHeight = Int(image.size.height * image.scale)
        let target = SettingsProfileRules.getAvatarCompressionDimensions(
            width: pixelWidth, height: pixelHeight, maxEdge: maxEdge
        )

        if target.width == pixelWidth, target.height == pixelHeight {
            return image.jpegData(compressionQuality: quality)
        }

        let renderer = UIGraphicsImageRenderer(size: CGSize(width: target.width, height: target.height))
        let resized = renderer.image { _ in
            image.draw(in: CGRect(x: 0, y: 0, width: target.width, height: target.height))
        }
        return resized.jpegData(compressionQuality: quality)
    }
}

/// 编辑资料页的头像上传，对应小程序 `uploadAvatar`。
@MainActor
final class AvatarUploadModel: ObservableObject {
    @Published private(set) var isUploading = false

    private struct UploadTokenResponse: Decodable {
        let success: Bool?
        let message: String?
        let uploadToken: String?
        let key: String?
        let uploadUrl: String?
        let domain: String?
    }

    private let session: SessionStore
    private let uploader: AvatarUploading
    private let now: () -> Date

    init(
        session: SessionStore,
        uploader: AvatarUploading = QiniuAvatarUploader(),
        now: @escaping () -> Date = Date.init
    ) {
        self.session = session
        self.uploader = uploader
        self.now = now
    }

    /// 处理图片并上传；返回要展示的 Toast 文案，身份变化时返回 nil。
    func upload(imageData: Data) async -> String? {
        guard !isUploading else { return nil }
        guard let jpeg = AvatarImageProcessing.jpegData(from: imageData) else {
            return "头像上传失败"
        }
        return await upload(preparedJPEG: jpeg)
    }

    func upload(preparedJPEG: Data) async -> String? {
        guard !isUploading else { return nil }
        guard let identity = session.identity, let user = session.user else { return nil }

        isUploading = true
        defer { isUploading = false }

        do {
            let fileExt = SettingsProfileRules.getFileExtensionFromPath("avatar.jpg")
            let tokenPath = APIQuery.path("/api/user/upload-token", ["file_ext": fileExt])
            let tokenResponse: UploadTokenResponse = try await session.authorizedRequest(tokenPath)
            guard session.identity == identity else { return nil }
            guard let uploadToken = tokenResponse.uploadToken, !uploadToken.isEmpty else {
                throw APIError(message: "头像上传失败", kind: .invalidResponse)
            }

            let uploadKey = (tokenResponse.key?.isEmpty == false)
                ? tokenResponse.key!
                : SettingsProfileRules.buildQiniuAvatarObjectKey(
                    userId: user.id,
                    filePath: "avatar.jpg",
                    timestamp: now()
                )

            let uploadURL = URL(string: tokenResponse.uploadUrl ?? "")
                ?? URL(string: SettingsProfileRules.defaultQiniuUploadURL)!
            let uploadedKey = try await uploader.upload(
                preparedJPEG, to: uploadURL, token: uploadToken, key: uploadKey
            )
            guard session.identity == identity else { return nil }

            let avatarURL = SettingsProfileRules.normalizeQiniuUploadResult(
                domain: tokenResponse.domain ?? "",
                key: uploadedKey.isEmpty ? uploadKey : uploadedKey
            )
            guard !avatarURL.isEmpty else {
                throw APIError(message: "头像地址生成失败", kind: .invalidResponse)
            }

            try await session.updateAvatar(avatarURL)
            guard session.identity == identity else { return nil }
            return "头像更新成功"
        } catch {
            guard session.identity == identity else { return nil }
            let text = error.localizedDescription.trimmingCharacters(in: .whitespacesAndNewlines)
            return text.isEmpty ? "头像上传失败" : text
        }
    }
}
