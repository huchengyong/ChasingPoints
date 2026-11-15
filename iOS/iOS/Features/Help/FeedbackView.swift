import SwiftUI

/// 帮助、投诉与举报，对应小程序 `subPages/help/feedback.vue`。
struct FeedbackView: View {
    @EnvironmentObject private var session: SessionStore
    @EnvironmentObject private var toast: ToastCenter
    @Environment(\.colorScheme) private var colorScheme
    @Environment(\.dismiss) private var dismiss

    @StateObject private var model: FeedbackModel

    init(session: SessionStore, model: FeedbackModel? = nil) {
        _model = StateObject(wrappedValue: model ?? FeedbackModel(session: session))
    }

    private var palette: AppPalette { AppTheme.palette(for: colorScheme) }

    var body: some View {
        ScrollView {
            VStack(spacing: 12) {
                intro
                formCard
            }
            .padding(.horizontal, 14)
            .padding(.vertical, 14)
        }
        .background(palette.page.ignoresSafeArea())
        .navigationTitle("帮助、投诉与举报")
        .navigationBarTitleDisplayMode(.inline)
        .safeAreaInset(edge: .bottom) {
            submitBar
        }
    }

    private var intro: some View {
        Text("此入口受理使用问题、投诉与举报。涉及账号、内容或线下纠纷时，请尽量补充对象、时间和证据线索。")
            .font(.system(size: 12))
            .foregroundStyle(palette.textSecondary)
            .lineSpacing(3)
            .frame(maxWidth: .infinity, alignment: .leading)
            .padding(13)
            .background(palette.subtle, in: RoundedRectangle(cornerRadius: 11))
    }

    private var formCard: some View {
        VStack(alignment: .leading, spacing: 10) {
            label("提交类型")
            Menu {
                ForEach(FeedbackCategory.allCases) { category in
                    Button(category.label) { model.category = category }
                }
            } label: {
                HStack {
                    Text(model.category.label)
                        .font(.system(size: 15, weight: .medium))
                        .foregroundStyle(palette.textPrimary)
                    Spacer(minLength: 0)
                    Image(systemName: "chevron.down")
                        .font(.system(size: 11, weight: .semibold))
                        .foregroundStyle(palette.textMuted)
                }
                .padding(.horizontal, 12)
                .frame(height: 42)
                .background(palette.subtle, in: RoundedRectangle(cornerRadius: 10))
                .overlay {
                    RoundedRectangle(cornerRadius: 10).stroke(palette.border)
                }
            }

            label("投诉举报或反馈内容")
            TextField("请详细描述您遇到的问题、投诉或举报事项...", text: $model.content, axis: .vertical)
                .font(.system(size: 14))
                .lineLimit(6...10)
                .padding(.horizontal, 12)
                .padding(.vertical, 10)
                .background(palette.subtle, in: RoundedRectangle(cornerRadius: 10))
                .overlay {
                    RoundedRectangle(cornerRadius: 10).stroke(palette.border)
                }
            Text("\(String(model.content.count))/\(String(FeedbackRules.maxContentLength))")
                .font(.system(size: 11))
                .foregroundStyle(palette.textMuted)
                .frame(maxWidth: .infinity, alignment: .trailing)

            label("联系方式 (选填)")
            TextField("手机号或邮箱，便于处理人员联系您", text: $model.contact)
                .font(.system(size: 14))
                .textInputAutocapitalization(.never)
                .autocorrectionDisabled()
                .padding(.horizontal, 12)
                .frame(height: 42)
                .background(palette.subtle, in: RoundedRectangle(cornerRadius: 10))
                .overlay {
                    RoundedRectangle(cornerRadius: 10).stroke(palette.border)
                }
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .appCard()
    }

    private var submitBar: some View {
        Button {
            Task { await submit() }
        } label: {
            Text(model.isSubmitting ? "提交中..." : "提交投诉举报与反馈")
        }
        .buttonStyle(.appPrimary)
        .disabled(!model.canSubmit)
        .padding(.horizontal, 16)
        .padding(.vertical, 10)
        .background(palette.page)
    }

    private func label(_ text: String) -> some View {
        Text(text)
            .font(.system(size: 13, weight: .semibold))
            .foregroundStyle(palette.textPrimary)
            .padding(.top, 2)
    }

    private func submit() async {
        switch await model.submit() {
        case .success:
            toast.show("提交成功")
            dismiss()
        case .validation(let message), .failure(let message):
            toast.show(message)
        case .discarded:
            break
        }
    }
}
