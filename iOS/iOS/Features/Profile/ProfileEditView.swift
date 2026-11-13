import PhotosUI
import SwiftUI

/// 编辑资料页，对应小程序 `subPages/user/editProfile`。
struct ProfileEditView: View {
    @EnvironmentObject private var session: SessionStore
    @EnvironmentObject private var toast: ToastCenter
    @Environment(\.colorScheme) private var colorScheme

    @StateObject private var avatarModel: AvatarUploadModel
    @State private var selectedPhoto: PhotosPickerItem?

    init(session: SessionStore, avatarModel: AvatarUploadModel? = nil) {
        _avatarModel = StateObject(wrappedValue: avatarModel ?? AvatarUploadModel(session: session))
    }

    private var palette: AppPalette { AppTheme.palette(for: colorScheme) }

    private var user: UserInfo? { session.user }
    private var nickname: String {
        let value = user?.nickname.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
        return value.isEmpty ? "用户" : value
    }
    private var phoneText: String {
        let value = SettingsProfileRules.formatSettingsPhone(user?.phone ?? "")
        return value.isEmpty ? "未绑定" : value
    }

    var body: some View {
        ScrollView {
            VStack(spacing: 24) {
                VStack(spacing: 10) {
                    PhotosPicker(selection: $selectedPhoto, matching: .images, photoLibrary: .shared()) {
                        ZStack(alignment: .bottomTrailing) {
                            AvatarView(avatar: user?.avatar ?? "", userId: user?.id ?? 0, size: 96)

                            Image(systemName: "camera.fill")
                                .font(.system(size: 13))
                                .foregroundStyle(.white)
                                .frame(width: 30, height: 30)
                                .background(AppTheme.primary, in: Circle())
                                .overlay { Circle().stroke(palette.card, lineWidth: 2) }
                                .offset(x: 2, y: 2)
                        }
                        .opacity(avatarModel.isUploading ? 0.6 : 1)
                    }
                    .disabled(avatarModel.isUploading)
                    .accessibilityLabel("更换头像")

                    Text(avatarModel.isUploading ? "上传中..." : "点击更换头像")
                        .font(.system(size: 12))
                        .foregroundStyle(palette.textSecondary)
                }
                .padding(.top, 8)

                VStack(spacing: 0) {
                    NavigationLink(value: AppRoute.editNickname) {
                        profileRow(label: "昵称", value: nickname, showsChevron: true)
                    }
                    .buttonStyle(.plain)

                    SettingsDivider()

                    profileRow(label: "手机号", value: phoneText, showsChevron: false)
                }
                .appCard(padding: 0)
            }
            .padding(16)
        }
        .background(palette.page.ignoresSafeArea())
        .navigationTitle("编辑资料")
        .navigationBarTitleDisplayMode(.inline)
        .onChange(of: selectedPhoto) { _, item in
            guard let item else { return }
            Task { await upload(item) }
        }
    }

    private func profileRow(label: String, value: String, showsChevron: Bool) -> some View {
        HStack(spacing: 10) {
            Text(label)
                .font(.system(size: 15))
                .foregroundStyle(palette.textPrimary)

            Spacer(minLength: 8)

            Text(value)
                .font(.system(size: 15))
                .lineLimit(1)
                .foregroundStyle(value == "未绑定" ? palette.textMuted : palette.textSecondary)

            if showsChevron {
                SettingsChevron()
            }
        }
        .padding(.horizontal, 14)
        .frame(minHeight: 54)
        .contentShape(Rectangle())
    }

    @MainActor
    private func upload(_ item: PhotosPickerItem) async {
        defer { selectedPhoto = nil }

        guard let data = try? await item.loadTransferable(type: Data.self) else {
            toast.show("头像上传失败")
            return
        }
        if let message = await avatarModel.upload(imageData: data) {
            toast.show(message)
        }
    }
}
