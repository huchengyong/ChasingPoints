import SwiftUI

/// 基础设置：主题、协议与退出。访客同样可以打开并使用主题与协议入口。
struct SettingsView: View {
    @EnvironmentObject private var session: SessionStore
    @Environment(\.colorScheme) private var colorScheme
    @Environment(\.dismiss) private var dismiss
    @AppStorage("theme_mode") private var themeMode = "system"
    @State private var showLogoutConfirm = false

    private var palette: AppPalette { AppTheme.palette(for: colorScheme) }

    var body: some View {
        ScrollView {
            VStack(spacing: 20) {
                VStack(alignment: .leading, spacing: 12) {
                    Text("主题模式")
                        .font(.headline)
                        .foregroundStyle(palette.textPrimary)
                    Picker("主题模式", selection: $themeMode) {
                        ForEach(ThemeMode.allCases, id: \.rawValue) { mode in
                            Text(mode.title).tag(mode.rawValue)
                        }
                    }
                    .pickerStyle(.segmented)
                }
                .frame(maxWidth: .infinity, alignment: .leading)
                .padding(20)
                .background(palette.card, in: RoundedRectangle(cornerRadius: 16))

                VStack(spacing: 0) {
                    Link(destination: URL(string: "https://www.zhuifen.cn/agreement")!) {
                        SettingsRow(title: "用户协议", symbol: "doc.text")
                    }
                    Divider().padding(.leading, 56)
                    Link(destination: URL(string: "https://www.zhuifen.cn/privacy")!) {
                        SettingsRow(title: "隐私政策", symbol: "hand.raised")
                    }
                }
                .background(palette.card, in: RoundedRectangle(cornerRadius: 16))

                if session.user != nil {
                    Button(role: .destructive) {
                        showLogoutConfirm = true
                    } label: {
                        Text("退出登录")
                            .font(.system(size: 16, weight: .semibold))
                            .frame(maxWidth: .infinity)
                            .frame(height: 52)
                    }
                    .foregroundStyle(AppTheme.danger)
                    .background(palette.card, in: RoundedRectangle(cornerRadius: 16))
                }
            }
            .padding(20)
        }
        .background(palette.page.ignoresSafeArea())
        .navigationTitle("设置")
        .alert("确认退出登录？", isPresented: $showLogoutConfirm) {
            Button("退出登录", role: .destructive) {
                session.logout()
                dismiss()
            }
            Button("取消", role: .cancel) {}
        } message: {
            Text("退出后需重新登录，主题偏好会保留")
        }
    }
}

/// “我的”与设置共用的入口行样式。
struct SettingsRow: View {
    @Environment(\.colorScheme) private var colorScheme

    let title: String
    let symbol: String

    var body: some View {
        HStack(spacing: 12) {
            Image(systemName: symbol)
                .font(.system(size: 17))
                .foregroundStyle(AppTheme.primary)
                .frame(width: 28)
            Text(title)
                .font(.system(size: 16))
                .foregroundStyle(AppTheme.palette(for: colorScheme).textPrimary)
            Spacer()
            Image(systemName: "chevron.right")
                .font(.system(size: 13, weight: .semibold))
                .foregroundStyle(AppTheme.palette(for: colorScheme).textSecondary)
        }
        .padding(.horizontal, 16)
        .frame(height: 52)
        .contentShape(Rectangle())
    }
}
