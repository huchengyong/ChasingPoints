import SwiftUI

/// 昵称编辑页：只保存草稿，成功回写共享会话后返回；取消或失败不改动共享资料。
struct NicknameEditView: View {
    @EnvironmentObject private var session: SessionStore
    @Environment(\.colorScheme) private var colorScheme
    @Environment(\.dismiss) private var dismiss
    @State private var draft = ""
    @State private var ownerId: Int64?
    @State private var isSaving = false
    @State private var errorMessage: String?
    @FocusState private var isFocused: Bool

    private var palette: AppPalette { AppTheme.palette(for: colorScheme) }
    private var validationMessage: String? { SessionStore.nicknameValidationMessage(draft) }
    private var isUnchanged: Bool { SessionStore.normalizedNickname(draft) == session.user?.nickname }
    private var canSave: Bool { validationMessage == nil && !isUnchanged && !isSaving && ownerId != nil }

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 0) {
                Text("昵称")
                    .font(.system(size: 14, weight: .medium))
                    .foregroundStyle(palette.textPrimary)

                TextField("请输入昵称", text: $draft)
                    .font(.system(size: 16))
                    .foregroundStyle(palette.textPrimary)
                    .focused($isFocused)
                    .disabled(isSaving)
                    .padding(.horizontal, 16)
                    .frame(height: 54)
                    .background(palette.subtle, in: RoundedRectangle(cornerRadius: 12))
                    .overlay {
                        RoundedRectangle(cornerRadius: 12)
                            .stroke(validationMessage == nil ? palette.border : AppTheme.danger)
                    }
                    .padding(.top, 10)
                    .onChange(of: draft) { _, _ in errorMessage = nil }

                if let validationMessage {
                    Text(validationMessage)
                        .font(.system(size: 12))
                        .foregroundStyle(AppTheme.danger)
                        .padding(.top, 6)
                }

                Button {
                    isFocused = false
                    Task { await save() }
                } label: {
                    Text(isSaving ? "保存中…" : "保存")
                        .font(.system(size: 16, weight: .semibold))
                        .frame(maxWidth: .infinity)
                        .frame(height: 52)
                }
                .foregroundStyle(.white)
                .background(AppTheme.primary, in: RoundedRectangle(cornerRadius: 12))
                .disabled(!canSave)
                .opacity(canSave ? 1 : 0.55)
                .padding(.top, 28)

                if let errorMessage {
                    Text(errorMessage)
                        .font(.system(size: 13))
                        .foregroundStyle(AppTheme.danger)
                        .padding(.top, 14)
                        .accessibilityAddTraits(.updatesFrequently)
                }
            }
            .padding(20)
        }
        .scrollDismissesKeyboard(.interactively)
        .background(palette.page.ignoresSafeArea())
        .navigationTitle("修改昵称")
        .navigationBarTitleDisplayMode(.inline)
        .onAppear {
            draft = session.user?.nickname ?? ""
            ownerId = session.user?.id
        }
        .onChange(of: session.user?.id) { _, newValue in
            // 退出、切号或会话失效后关闭旧身份的编辑状态。
            guard let ownerId else { return }
            if newValue != ownerId { dismiss() }
        }
    }

    @MainActor
    private func save() async {
        guard canSave, let ownerId else { return }
        isSaving = true
        errorMessage = nil
        defer { isSaving = false }

        do {
            try await session.updateNickname(draft)
            dismiss()
        } catch {
            // 迟到结果属于旧身份时不向当前页面展示。
            guard session.user?.id == ownerId else { return }
            errorMessage = error.localizedDescription
        }
    }
}
