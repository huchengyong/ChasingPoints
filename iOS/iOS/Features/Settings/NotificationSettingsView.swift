import SwiftUI

/// 通知管理页，对应小程序 `subPages/user/notification`。
struct NotificationSettingsView: View {
    @EnvironmentObject private var session: SessionStore
    @EnvironmentObject private var toast: ToastCenter
    @Environment(\.colorScheme) private var colorScheme

    @StateObject private var model: NotificationPreferencesModel

    init(session: SessionStore, model: NotificationPreferencesModel? = nil) {
        _model = StateObject(wrappedValue: model ?? NotificationPreferencesModel(session: session))
    }

    private var palette: AppPalette { AppTheme.palette(for: colorScheme) }

    private var visiblePreferences: [NotificationPreferencesModel.Preference] {
        NotificationPreferencesModel.Preference.allCases.filter {
            $0 != .follow || model.showFollowPreference
        }
    }

    var body: some View {
        Group {
            switch model.phase {
            case .idle, .loading:
                VStack(spacing: 12) {
                    ProgressView()
                    Text("正在加载通知偏好...")
                        .font(.system(size: 14))
                        .foregroundStyle(palette.textSecondary)
                }
                .frame(maxWidth: .infinity, maxHeight: .infinity)
            case .failed:
                StateMessageView(
                    icon: "exclamationmark.triangle",
                    title: "通知偏好加载失败",
                    detail: model.errorMessage,
                    actionTitle: "重试",
                    action: { Task { await model.retry() } }
                )
            case .ready:
                ScrollView {
                    VStack(alignment: .leading, spacing: 12) {
                        VStack(spacing: 0) {
                            ForEach(Array(visiblePreferences.enumerated()), id: \.element) { index, preference in
                                if index > 0 {
                                    SettingsDivider()
                                }
                                preferenceRow(preference)
                            }
                        }
                        .appCard(padding: 0)

                        Text("管理消息中心和推送共同使用的通知类型。关闭后，仅影响后续新消息，不清理历史消息。")
                            .font(.system(size: 12))
                            .foregroundStyle(palette.textSecondary)
                            .fixedSize(horizontal: false, vertical: true)
                            .padding(.horizontal, 4)
                    }
                    .padding(16)
                }
            }
        }
        .background(palette.page.ignoresSafeArea())
        .navigationTitle("通知管理")
        .task {
            if model.phase == .idle {
                await model.load()
            }
        }
        .onChange(of: session.identity) { _, _ in
            Task { await model.identityChanged() }
        }
    }

    private func preferenceRow(_ preference: NotificationPreferencesModel.Preference) -> some View {
        let isOn = model.preferences.value(for: preference)

        return Button {
            guard !model.isSaving else { return }
            Task {
                if let message = await model.toggle(preference, value: !isOn) {
                    toast.show(message)
                }
            }
        } label: {
            HStack(spacing: 8) {
                SettingsLabel(symbol: symbol(for: preference), title: preference.title)
                Spacer(minLength: 8)
                SettingsSwitch(isOn: isOn)
            }
            .padding(.horizontal, 12)
            .frame(minHeight: 58)
            .contentShape(Rectangle())
        }
        .buttonStyle(.plain)
        .disabled(model.isSaving)
        .accessibilityValue(isOn ? "已开启" : "已关闭")
    }

    private func symbol(for preference: NotificationPreferencesModel.Preference) -> String {
        switch preference {
        case .matchResult: "trophy"
        case .friendRequest: "person.2"
        case .challenge: "bell"
        case .tournament: "calendar"
        case .follow: "star"
        }
    }
}
