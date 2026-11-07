import Foundation
import XCTest
@testable import iOS

/// 认证与资料接口的虚构夹具，字段与 `backend/chasing_points.api` 及对应 logic 的实际响应一致。
/// 全部为假数据，不包含真实 token、验证码或完整手机号。
enum AuthFixtures {
    static let phone = "13800000000"
    static let maskedPhone = "138****0000"
    static let oldAccess = "old-access-token"
    static let oldRefresh = "old-refresh-token"
    static let newAccess = "new-access-token"
    static let newRefresh = "new-refresh-token"

    static func userInfo(id: Int64 = 1, nickname: String = "测试用户", status: Int = 1) -> [String: Any] {
        [
            "id": id,
            "phone": maskedPhone,
            "nickname": nickname,
            "avatar": "",
            "status": status,
            "created_at": "2026-01-01 00:00:00",
        ]
    }

    static func sendSMSResponse(success: Bool = true, message: String = "验证码已发送") -> [String: Any] {
        ["success": success, "message": message]
    }

    static func loginResponse(
        access: String = "login-access-token",
        refresh: String = "login-refresh-token",
        user: [String: Any]? = nil
    ) -> [String: Any] {
        [
            "success": true,
            "access_token": access,
            "refresh_token": refresh,
            "expires_in": 604800,
            "user_info": user ?? userInfo(),
        ]
    }

    static func refreshResponse(access: String = newAccess, refresh: String = newRefresh) -> [String: Any] {
        ["success": true, "access_token": access, "refresh_token": refresh, "expires_in": 604800]
    }

    static func userInfoResponse(_ user: [String: Any]? = nil) -> [String: Any] {
        ["success": true, "user_info": user ?? userInfo()]
    }

    /// `POST /api/user/profile` 成功：HTTP 200 + success + 完整 user_info。
    static func updateProfileResponse(nickname: String = "新昵称") -> [String: Any] {
        ["success": true, "message": "更新成功", "user_info": userInfo(nickname: nickname)]
    }

    /// `POST /api/user/profile` 业务失败：HTTP 200 + success:false。
    static func updateProfileFailure(message: String = "昵称长度需要在2-12个字符之间") -> [String: Any] {
        ["success": false, "message": message, "user_info": NSNull()]
    }

    /// 后端 `httperror.Payload`：HTTP 200/401 + success:false + reason。
    static func sessionInvalidPayload(status: Int = 401) -> [String: Any] {
        ["success": false, "reason": "SESSION_INVALID", "message": "登录状态已失效，请重新登录"]
    }

    static func credentials(access: String = oldAccess, refresh: String = oldRefresh) -> Credentials {
        Credentials(accessToken: access, refreshToken: refresh)
    }
}
