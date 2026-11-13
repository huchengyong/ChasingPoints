import SwiftUI

/// 设置页，对应小程序 `subPages/user/settings`。访客仍可使用主题与协议入口。
struct SettingsView: View {
    @EnvironmentObject private var session: SessionStore
    @EnvironmentObject private var toast: ToastCenter
    @Environment(\.colorScheme) private var colorScheme
    @Environment(\.dismiss) private var dismiss

    @AppStorage("theme_mode") private var themeMode = "system"
    @StateObject private var privacy: PrivacySettingsModel
    @StateObject private var deleteAccount: DeleteAccountModel

    @State private var showLogoutConfirm = false
    @State private var showDeleteAccount = false
    @State private var defaultGameType = 0

    private let preference = GameTypePreference()

    init(
        session: SessionStore,
        privacyModel: PrivacySettingsModel? = nil,
        deleteAccountModel: DeleteAccountModel? = nil
    ) {
        _privacy = StateObject(wrappedValue: privacyModel ?? PrivacySettingsModel(session: session))
        _deleteAccount = StateObject(wrappedValue: deleteAccountModel ?? DeleteAccountModel(session: session))
    }

    private var palette: AppPalette { AppTheme.palette(for: colorScheme) }
    private var isSignedIn: Bool { session.identity != nil }

    var body: some View {
        ScrollView {
            VStack(spacing: 16) {
                menuCard

                if isSignedIn {
                    Button(role: .destructive) {
                        showLogoutConfirm = true
                    } label: {
                        Text("退出登录")
                            .font(.system(size: 16, weight: .semibold))
                            .foregroundStyle(AppTheme.danger)
                            .frame(maxWidth: .infinity)
                            .frame(height: 52)
                            .background(AppTheme.danger.opacity(0.08), in: RoundedRectangle(cornerRadius: 14))
                    }
                }
            }
            .padding(16)
        }
        .background(palette.page.ignoresSafeArea())
        .navigationTitle("设置")
        .task {
            guard isSignedIn else { return }
            readDefaultGameType()
            await privacy.load()
        }
        .onChange(of: session.identity) { _, _ in
            readDefaultGameType()
            Task { await privacy.identityChanged() }
        }
        .sheet(isPresented: $showDeleteAccount) {
            DeleteAccountDialog(model: deleteAccount, onClose: { showDeleteAccount = false })
                .presentationDetents([.medium, .large])
                .presentationDragIndicator(.visible)
                .interactiveDismissDisabled(deleteAccount.isSubmitting)
        }
        .alert("确认退出登录？", isPresented: $showLogoutConfirm) {
            Button("退出登录", role: .destructive) {
                session.logout()
                dismiss()
                toast.show("已退出登录")
            }
            Button("取消", role: .cancel) {}
        } message: {
            Text("退出后需重新登录，主题偏好会保留")
        }
    }

    private var menuCard: some View {
        VStack(spacing: 0) {
            themeRow

            if isSignedIn {
                SettingsDivider()
                navigationRow("通知管理", symbol: "bell", route: .notificationSettings)
                SettingsDivider()
                privacyToggleRow(
                    title: "隐藏战绩",
                    subtitle: "控制是否在公开场景展示你的战绩",
                    symbol: "lock",
                    isOn: privacy.hideMatchRecord,
                    action: { await privacy.toggleHideMatch() }
                )
                SettingsDivider()
                navigationRow(
                    "黑名单",
                    subtitle: "查看并移出已拉黑的用户",
                    symbol: "person.crop.circle.badge.xmark",
                    route: .blacklist
                )
                SettingsDivider()
                privacyToggleRow(
                    title: "仅允许好友约球",
                    subtitle: "开启后，非好友无法向你发送约球邀请",
                    symbol: "person.2",
                    isOn: privacy.friendsOnlyChallenges,
                    action: { await privacy.toggleFriendsOnlyChallenges() }
                )
            }

            SettingsDivider()
            externalLinkRow(
                "隐私政策",
                symbol: "hand.raised",
                url: "https://www.zhuifen.cn/privacy"
            )
            SettingsDivider()
            externalLinkRow(
                "用户协议",
                symbol: "doc.text",
                url: "https://www.zhuifen.cn/agreement"
            )
            SettingsDivider()
            navigationRow(
                "帮助、投诉与举报",
                subtitle: "提交问题建议、投诉举报或获取帮助",
                symbol: "bubble.left.and.bubble.right",
                route: .feedback
            )

            if isSignedIn {
                SettingsDivider()
                gameTypeRow
                SettingsDivider()
                deleteAccountRow
            }
        }
        .appCard(padding: 0)
    }

    private var themeRow: some View {
        VStack(alignment: .leading, spacing: 12) {
            SettingsLabel(symbol: "circle.lefthalf.filled", title: "主题模式")

            Picker("主题模式", selection: $themeMode) {
                ForEach(ThemeMode.allCases, id: \.rawValue) { mode in
                    Text(mode.title).tag(mode.rawValue)
                }
            }
            .pickerStyle(.segmented)
        }
        .padding(12)
    }

    private func navigationRow(
        _ title: String,
        subtitle: String? = nil,
        symbol: String,
        route: AppRoute
    ) -> some View {
        NavigationLink(value: route) {
            HStack(spacing: 8) {
                SettingsLabel(symbol: symbol, title: title, subtitle: subtitle)
                Spacer(minLength: 8)
                SettingsChevron()
            }
            .padding(.horizontal, 12)
            .frame(minHeight: 56)
            .contentShape(Rectangle())
        }
        .buttonStyle(.plain)
    }

    private func externalLinkRow(_ title: String, symbol: String, url: String) -> some View {
        Link(destination: URL(string: url)!) {
            HStack(spacing: 8) {
                SettingsLabel(symbol: symbol, title: title)
                Spacer(minLength: 8)
                SettingsChevron()
            }
            .padding(.horizontal, 12)
            .frame(minHeight: 56)
            .contentShape(Rectangle())
        }
        .buttonStyle(.plain)
    }

    private func privacyToggleRow(
        title: String,
        subtitle: String,
        symbol: String,
        isOn: Bool,
        action: @escaping () async -> String?
    ) -> some View {
        Button {
            Task {
                if let message = await action() {
                    toast.show(message)
                }
            }
        } label: {
            HStack(spacing: 8) {
                SettingsLabel(symbol: symbol, title: title, subtitle: subtitle, tint: palette.brandStrong)
                Spacer(minLength: 8)
                privacyAccessory(isOn: isOn)
            }
            .padding(.horizontal, 12)
            .frame(minHeight: 56)
            .contentShape(Rectangle())
        }
        .buttonStyle(.plain)
        .accessibilityValue(isOn ? "已开启" : "已关闭")
    }

    @ViewBuilder
    private func privacyAccessory(isOn: Bool) -> some View {
        switch privacy.phase {
        case .ready:
            SettingsSwitch(isOn: isOn)
        case .idle, .loading:
            Text("读取中…")
                .font(.system(size: 12))
                .foregroundStyle(palette.textSecondary)
        case .failed:
            Text("读取失败，点击重试")
                .font(.system(size: 12))
                .foregroundStyle(AppTheme.danger)
        }
    }

    private var gameTypeRow: some View {
        VStack(alignment: .leading, spacing: 10) {
            SettingsLabel(
                symbol: "flag",
                title: "常用球种",
                subtitle: "「我的」段位卡优先展示，发起 PK 时自动预选"
            )

            LazyVGrid(columns: [GridItem(.flexible()), GridItem(.flexible())], spacing: 8) {
                gameTypeOption(value: 0, label: "每次询问")
                ForEach(GameTypeCatalog.ordered) { item in
                    gameTypeOption(value: item.value, label: item.label)
                }
            }
        }
        .padding(12)
    }

    private func gameTypeOption(value: Int, label: String) -> some View {
        let isSelected = defaultGameType == value

        return Button {
            setDefaultGameType(value)
        } label: {
            Text(label)
                .font(.system(size: 13, weight: isSelected ? .semibold : .regular))
                .lineLimit(1)
                .foregroundStyle(isSelected ? palette.brandStrong : palette.textSecondary)
                .frame(maxWidth: .infinity)
                .frame(height: 34)
                .background(
                    isSelected ? palette.brandTint : palette.subtle,
                    in: RoundedRectangle(cornerRadius: 17)
                )
                .overlay {
                    RoundedRectangle(cornerRadius: 17)
                        .stroke(isSelected ? AppTheme.primary.opacity(0.42) : palette.border)
                }
        }
        .buttonStyle(.plain)
    }

    private var deleteAccountRow: some View {
        Button {
            deleteAccount.open()
            showDeleteAccount = true
        } label: {
            HStack(spacing: 8) {
                SettingsLabel(
                    symbol: "trash",
                    title: "注销账号",
                    subtitle: "删除个人资料与关系，此操作不可恢复",
                    tint: AppTheme.danger,
                    titleColor: AppTheme.danger
                )
                Spacer(minLength: 8)
                Image(systemName: "chevron.right")
                    .font(.system(size: 12, weight: .semibold))
                    .foregroundStyle(AppTheme.danger)
            }
            .padding(.horizontal, 12)
            .frame(minHeight: 56)
            .contentShape(Rectangle())
        }
        .buttonStyle(.plain)
    }

    private func readDefaultGameType() {
        guard let userId = session.user?.id else {
            defaultGameType = 0
            return
        }
        defaultGameType = preference.read(userId: userId)
    }

    private func setDefaultGameType(_ value: Int) {
        guard let userId = session.user?.id else { return }
        defaultGameType = preference.save(userId: userId, gameType: value)
        toast.show(defaultGameType > 0 ? "常用球种已更新" : "已改为每次询问")
    }
}
