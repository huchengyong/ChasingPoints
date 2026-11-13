import SwiftUI

/// 注销账号两步弹层内容，可直接嵌入 sheet，也可在测试中单独渲染。
struct DeleteAccountDialog: View {
    @EnvironmentObject private var toast: ToastCenter
    @Environment(\.colorScheme) private var colorScheme
    @ObservedObject var model: DeleteAccountModel
    let onClose: () -> Void

    private var palette: AppPalette { AppTheme.palette(for: colorScheme) }

    var body: some View {
        VStack(alignment: .leading, spacing: 0) {
            Text(model.step == .warning ? "注销账号" : "身份复核")
                .font(.system(size: 19, weight: .bold))
                .foregroundStyle(palette.textPrimary)

            if model.step == .warning {
                Text("注销后，账号资料、认证关联和社交数据会被删除；必要的历史比赛会匿名保留。此操作不可恢复。")
                    .font(.system(size: 14))
                    .foregroundStyle(palette.textSecondary)
                    .fixedSize(horizontal: false, vertical: true)
                    .padding(.top, 10)

                actions {
                    Button("取消", action: onClose)
                        .buttonStyle(.appSecondary)
                    Button("我已知晓，继续", action: model.continueToVerify)
                        .buttonStyle(DangerFilledButtonStyle())
                }
            } else {
                Text(model.hasBoundPhone
                    ? "请输入已绑定的完整手机号并完成注销验证码验证。"
                    : "当前账号未绑定手机号，请在微信小程序中完成注销。")
                    .font(.system(size: 14))
                    .foregroundStyle(palette.textSecondary)
                    .fixedSize(horizontal: false, vertical: true)
                    .padding(.top, 10)

                if model.hasBoundPhone {
                    VStack(spacing: 10) {
                        inputField {
                            TextField("已绑定的完整手机号", text: $model.phone)
                                .keyboardType(.numberPad)
                                .textContentType(.telephoneNumber)
                                .disabled(model.isSubmitting)
                                .onChange(of: model.phone) { _, value in
                                    model.phone = String(value.filter { "0123456789".contains($0) }.prefix(11))
                                }
                        }

                        inputField {
                            HStack(spacing: 8) {
                                TextField("注销验证码", text: $model.smsCode)
                                    .keyboardType(.numberPad)
                                    .textContentType(.oneTimeCode)
                                    .disabled(model.isSubmitting)
                                    .onChange(of: model.smsCode) { _, value in
                                        model.smsCode = String(value.filter { "0123456789".contains($0) }.prefix(6))
                                    }

                                Button {
                                    Task {
                                        if let message = await model.sendSMS() {
                                            toast.show(message)
                                        }
                                    }
                                } label: {
                                    Text(model.isSendingSMS ? "发送中…" : "获取验证码")
                                        .font(.system(size: 13, weight: .semibold))
                                        .fixedSize()
                                }
                                .foregroundStyle(palette.brandStrong)
                                .disabled(model.isSendingSMS || model.isSubmitting)
                            }
                        }
                    }
                    .padding(.top, 14)
                }

                actions {
                    Button("取消", action: onClose)
                        .buttonStyle(.appSecondary)
                        .disabled(model.isSubmitting)
                    if model.hasBoundPhone {
                        Button {
                            Task {
                                switch await model.submit() {
                                case .success:
                                    onClose()
                                    toast.show("账号已注销")
                                case .failure(let message):
                                    toast.show(message)
                                case .ignored:
                                    break
                                }
                            }
                        } label: {
                            Text(model.isSubmitting ? "处理中…" : "确认注销")
                        }
                        .buttonStyle(DangerFilledButtonStyle())
                        .disabled(model.isSubmitting)
                    }
                }
            }
        }
        .padding(22)
        .frame(maxWidth: .infinity, alignment: .leading)
        .background(palette.card, in: RoundedRectangle(cornerRadius: 20))
        .padding(20)
        .background(palette.page.ignoresSafeArea())
    }

    private func inputField<Content: View>(@ViewBuilder content: () -> Content) -> some View {
        content()
            .font(.system(size: 15))
            .foregroundStyle(palette.textPrimary)
            .padding(.horizontal, 14)
            .frame(height: 48)
            .background(palette.subtle, in: RoundedRectangle(cornerRadius: 12))
            .overlay {
                RoundedRectangle(cornerRadius: 12).stroke(palette.border)
            }
    }

    @ViewBuilder
    private func actions<Content: View>(@ViewBuilder content: () -> Content) -> some View {
        HStack(spacing: 10) {
            content()
        }
        .padding(.top, 20)
    }
}

private struct DangerFilledButtonStyle: ButtonStyle {
    @Environment(\.isEnabled) private var isEnabled

    func makeBody(configuration: Configuration) -> some View {
        configuration.label
            .font(.system(size: 15, weight: .semibold))
            .foregroundStyle(.white)
            .frame(maxWidth: .infinity)
            .frame(height: 48)
            .background(AppTheme.danger, in: RoundedRectangle(cornerRadius: 12))
            .opacity(isEnabled ? (configuration.isPressed ? 0.85 : 1) : 0.5)
    }
}
