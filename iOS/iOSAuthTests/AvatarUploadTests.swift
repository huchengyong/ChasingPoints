import Foundation
import UIKit
import XCTest
@testable import iOS

@MainActor
final class AvatarUploadTests: XCTestCase {
    func testCompressionDimensionsKeepLongestEdgeWithinLimit() {
        XCTAssertEqual(
            SettingsProfileRules.getAvatarCompressionDimensions(width: 2400, height: 1200),
            SettingsProfileRules.AvatarDimensions(width: 512, height: 256)
        )
        XCTAssertEqual(
            SettingsProfileRules.getAvatarCompressionDimensions(width: 300, height: 200),
            SettingsProfileRules.AvatarDimensions(width: 300, height: 200)
        )
    }

    func testObjectKeyUsesFixedTimestamp() {
        let timestamp = Date(timeIntervalSince1970: 1_775_722_800)
        XCTAssertTrue(
            SettingsProfileRules.buildQiniuAvatarObjectKey(
                userId: 18, filePath: "avatar.jpg", timestamp: timestamp
            ).hasPrefix("avatars/18/20260409/1775722800000")
        )
    }

    func testQiniuMultipartContainsTokenKeyAndFileWithoutAuthorization() async throws {
        let server = MockHTTPServer()
        server.stub("POST", "/upload", .json(["key": "avatars/1/a.jpg"]))
        let uploader = QiniuAvatarUploader(session: server.makeSession())

        let key = try await uploader.upload(
            Data("jpeg-bytes".utf8),
            to: URL(string: "https://upload.test/upload")!,
            token: "upload-token",
            key: "avatars/1/a.jpg"
        )

        XCTAssertEqual(key, "avatars/1/a.jpg")
        let request = try XCTUnwrap(server.requests("/upload").first)
        XCTAssertNil(request.bearerToken)
        let body = String(decoding: request.body, as: UTF8.self)
        XCTAssertTrue(body.contains("name=\"token\"\r\n\r\nupload-token"))
        XCTAssertTrue(body.contains("name=\"key\"\r\n\r\navatars/1/a.jpg"))
        XCTAssertTrue(body.contains("name=\"file\"; filename=\"a.jpg\""))
        XCTAssertTrue(body.contains("Content-Type: image/jpeg"))
    }

    func testEmptyUploadURLFallsBackToDefaultQiniuHost() async {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        stubToken(server, uploadURL: "")
        // 域名根路径在 URL.path 里是空串。
        server.stub("POST", "", .json(["key": "avatars/1/a.jpg"]))
        server.stub(
            "POST",
            "/api/user/profile",
            .json(AuthFixtures.updateProfileResponse(nickname: "测试用户", avatar: "https://cdn.example.com/avatars/1/a.jpg"))
        )
        let model = AvatarUploadModel(
            session: session,
            uploader: QiniuAvatarUploader(session: server.makeSession())
        )

        let message = await model.upload(preparedJPEG: Data("jpeg".utf8))

        XCTAssertEqual(message, "头像更新成功")
        XCTAssertEqual(server.requestCount(""), 1)
        XCTAssertEqual(SettingsProfileRules.defaultQiniuUploadURL, "https://up-z2.qiniup.com")
    }

    func testEmptyTokenFailsWithoutUpdatingProfile() async {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        stubToken(server, token: "")
        let model = AvatarUploadModel(
            session: session,
            uploader: QiniuAvatarUploader(session: server.makeSession())
        )

        let message = await model.upload(preparedJPEG: Data("jpeg".utf8))

        XCTAssertEqual(message, "头像上传失败")
        XCTAssertEqual(server.requestCount("/api/user/profile"), 0)
    }

    func testUploadServerErrorFailsWithoutUpdatingProfile() async {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        stubToken(server)
        server.stub("POST", "/", .json(["error": "failed"], status: 500))
        let model = AvatarUploadModel(
            session: session,
            uploader: QiniuAvatarUploader(session: server.makeSession())
        )

        let message = await model.upload(preparedJPEG: Data("jpeg".utf8))

        XCTAssertEqual(message, "上传失败")
        XCTAssertEqual(server.requestCount("/api/user/profile"), 0)
        XCTAssertEqual(server.requestCount("/"), 1)
    }

    func testEmptyDomainFailsWithoutUpdatingProfile() async {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        stubToken(server, domain: "")
        server.stub("POST", "/", .json(["key": "avatars/1/a.jpg"]))
        let model = AvatarUploadModel(
            session: session,
            uploader: QiniuAvatarUploader(session: server.makeSession())
        )

        let message = await model.upload(preparedJPEG: Data("jpeg".utf8))

        XCTAssertEqual(message, "头像地址生成失败")
        XCTAssertEqual(server.requestCount("/api/user/profile"), 0)
    }

    func testFullSuccessWritesAvatarBackToSession() async {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        let avatar = "https://cdn.example.com/avatars/1/a.jpg"
        stubToken(server, key: "avatars/1/a.jpg")
        server.stub("POST", "/", .json(["key": "avatars/1/a.jpg"]))
        server.stub(
            "POST",
            "/api/user/profile",
            .json(AuthFixtures.updateProfileResponse(nickname: "测试用户", avatar: avatar))
        )
        let model = AvatarUploadModel(
            session: session,
            uploader: QiniuAvatarUploader(session: server.makeSession())
        )

        let message = await model.upload(preparedJPEG: Data("jpeg".utf8))

        XCTAssertEqual(message, "头像更新成功")
        XCTAssertEqual(session.user?.avatar, avatar)
        let update = server.requests("/api/user/profile").first
        XCTAssertEqual(update?.bodyJSON["nickname"] as? String, "测试用户")
        XCTAssertEqual(update?.bodyJSON["avatar"] as? String, avatar)
    }

    func testLogoutDuringUploadSkipsProfileUpdate() async {
        let server = MockHTTPServer()
        let session = await makeRestoredAccountSession(server)
        stubToken(server)
        let gate = DispatchSemaphore(value: 0)
        server.stub("POST", "/") { _ in
            _ = gate.wait(timeout: .now() + 5)
            return .json(["key": "avatars/1/a.jpg"])
        }
        let model = AvatarUploadModel(
            session: session,
            uploader: QiniuAvatarUploader(session: server.makeSession())
        )

        let upload = Task { await model.upload(preparedJPEG: Data("jpeg".utf8)) }
        await waitUntil { server.requestCount("/") == 1 }
        session.logout()
        gate.signal()

        let message = await upload.value

        XCTAssertNil(message)
        XCTAssertEqual(server.requestCount("/api/user/profile"), 0)
        XCTAssertNil(session.user)
    }

    private func stubToken(
        _ server: MockHTTPServer,
        uploadURL: String = "https://upload.test/",
        token: String = "upload-token",
        key: String = "",
        domain: String = "https://cdn.example.com/"
    ) {
        server.stub("GET", "/api/user/upload-token", .json([
            "success": true,
            "message": "",
            "upload_token": token,
            "key": key,
            "upload_url": uploadURL,
            "domain": domain,
        ]))
    }
}
