import Combine
import Foundation

/// 注销账号两步弹层，对应小程序 settings 页的 deletion-dialog。
/// 原生端只有手机号登录，不做第三方授权复核。
@MainActor
final class DeleteAccountModel: ObservableObject {
    enum Step {
        case warning
        case verify
    }

    enum SubmitResult: Equatable {
        case success
        case failure(String)
        case ignored
    }

    @Published private(set) var step: Step = .warning
    @Published var phone = ""
    @Published var smsCode = ""
    @Published private(set) var isSendingSMS = false
    @Published private(set) var isSubmitting = false

    private let session: SessionStore

    init(session: SessionStore) {
        self.session = session
    }

    var hasBoundPhone: Bool {
        !(session.user?.phone ?? "").isEmpty
    }

    func open() {
        guard !isSubmitting else { return }
        step = .warning
        phone = ""
        smsCode = ""
    }

    func continueToVerify() {
        guard !isSubmitting else { return }
        step = .verify
    }

    func close() -> Bool {
        guard !isSubmitting else { return false }
        step = .warning
        return true
    }

    func sendSMS() async -> String? {
        guard !isSendingSMS else { return nil }
        let value = phone.trimmingCharacters(in: .whitespacesAndNewlines)
        guard value.range(of: #"^1[3-9]\d{9}$"#, options: .regularExpression) != nil else {
            return "请输入已绑定的完整手机号"
        }
        guard let identity = session.identity else { return nil }

        isSendingSMS = true
        defer { isSendingSMS = false }

        do {
            try await session.sendSMS(phone: value, scene: "delete_account")
            guard session.identity == identity else { return nil }
            return "注销验证码已发送"
        } catch {
            guard session.identity == identity else { return nil }
            return message(for: error, fallback: "验证码发送失败")
        }
    }

    func submit() async -> SubmitResult {
        guard !isSubmitting else { return .ignored }
        let phoneValue = phone.trimmingCharacters(in: .whitespacesAndNewlines)
        let codeValue = smsCode.trimmingCharacters(in: .whitespacesAndNewlines)
        guard phoneValue.range(of: #"^1[3-9]\d{9}$"#, options: .regularExpression) != nil,
              codeValue.range(of: #"^\d{6}$"#, options: .regularExpression) != nil else {
            return .failure("请填写完整手机号和注销验证码")
        }
        guard let identity = session.identity else { return .ignored }

        isSubmitting = true
        defer { isSubmitting = false }

        do {
            try await session.deleteAccount(smsCode: codeValue)
            // 会话已由 deleteAccount 清理；迟到结果不属于当前身份时不再提示。
            guard session.identity == nil else { return .ignored }
            _ = identity
            return .success
        } catch {
            guard session.identity == identity else { return .ignored }
            return .failure(message(for: error, fallback: "注销失败，请重试"))
        }
    }

    private func message(for error: Error, fallback: String) -> String {
        let text = error.localizedDescription.trimmingCharacters(in: .whitespacesAndNewlines)
        return text.isEmpty ? fallback : text
    }
}
