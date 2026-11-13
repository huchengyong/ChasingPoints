import SwiftUI

/// 上传球馆：表单、省市区选择、地图选点与提交后的返回。
struct VenueSubmitView: View {
    @EnvironmentObject private var session: SessionStore
    @EnvironmentObject private var navigator: AppNavigator
    @EnvironmentObject private var toast: ToastCenter
    @Environment(\.colorScheme) private var colorScheme
    @Environment(\.dismiss) private var dismiss
    @StateObject private var model: VenueSubmitModel
    @State private var isAreaPickerPresented = false
    @State private var isLocationPickerPresented = false

    init(session: SessionStore, model: VenueSubmitModel? = nil) {
        _model = StateObject(wrappedValue: model ?? VenueSubmitModel(session: session))
    }

    private var palette: AppPalette { AppTheme.palette(for: colorScheme) }

    var body: some View {
        Group {
            if session.identity == nil {
                StateMessageView(icon: "lock.fill", title: "登录后上传球馆", actionTitle: "去登录") {
                    navigator.openLogin()
                }
            } else {
                form
            }
        }
        .background(palette.page.ignoresSafeArea())
        .navigationTitle("上传球馆")
        .navigationBarTitleDisplayMode(.inline)
        .task { await model.loadIfNeeded() }
        .onChange(of: session.identity) { _, _ in
            model.syncIdentity()
            Task { await model.loadIfNeeded() }
        }
        .onChange(of: model.notice) { _, notice in
            guard let notice else { return }
            toast.show(notice)
            model.clearNotice()
        }
        .onChange(of: model.didSubmit) { _, didSubmit in
            guard didSubmit else { return }
            Task {
                try? await Task.sleep(nanoseconds: 1_500_000_000)
                dismiss()
            }
        }
        .sheet(isPresented: $isAreaPickerPresented) {
            VenueAreaPicker(model: model)
        }
        .sheet(isPresented: $isLocationPickerPresented) {
            VenueLocationPicker(initial: model.form.location.map { Coordinate(latitude: $0.latitude, longitude: $0.longitude) }) { coordinate in
                model.form.location = PickedLocation(
                    latitude: coordinate.latitude,
                    longitude: coordinate.longitude,
                    name: "",
                    address: ""
                )
            }
        }
    }

    private var form: some View {
        ScrollView {
            VStack(spacing: 12) {
                intro
                formSection
            }
            .padding(.horizontal, 12)
            .padding(.vertical, 12)
        }
        .safeAreaInset(edge: .bottom) {
            submitBar
        }
    }

    private var intro: some View {
        VStack(alignment: .leading, spacing: 6) {
            Text("绑定常玩球馆")
                .font(.system(size: 12))
                .foregroundStyle(.white.opacity(0.72))
            Text(model.intro.title)
                .font(.system(size: 22, weight: .bold))
                .foregroundStyle(.white)
            Text(model.intro.desc)
                .font(.system(size: 13))
                .foregroundStyle(.white.opacity(0.82))
                .lineSpacing(4)
        }
        .padding(16)
        .frame(maxWidth: .infinity, alignment: .leading)
        .background(
            LinearGradient(colors: palette.heroColors, startPoint: .topLeading, endPoint: .bottomTrailing),
            in: RoundedRectangle(cornerRadius: 16)
        )
    }

    private var formSection: some View {
        VStack(alignment: .leading, spacing: 16) {
            VStack(alignment: .leading, spacing: 4) {
                Text("常玩球馆资料")
                    .font(.system(size: 15, weight: .semibold))
                    .foregroundStyle(palette.textPrimary)
                Text(model.intro.sectionTip)
                    .font(.system(size: 12))
                    .foregroundStyle(palette.textMuted)
            }

            nameField
            regionField
            locationField
            addressField
        }
        .appCard()
    }

    private var nameField: some View {
        VStack(alignment: .leading, spacing: 6) {
            fieldLabel("球馆名称", isComplete: model.isNameValid)
            TextField("例如：星轨台球俱乐部", text: $model.form.name)
                .font(.system(size: 14))
                .foregroundStyle(palette.textPrimary)
                .padding(.horizontal, 12)
                .frame(height: 40)
                .background(palette.subtle, in: RoundedRectangle(cornerRadius: 8))
                .overlay {
                    RoundedRectangle(cornerRadius: 8)
                        .stroke(missingBorder(isMissing: !model.isNameValid), lineWidth: 1)
                }
                .onChange(of: model.form.name) { _, value in
                    if value.count > 50 {
                        model.form.name = String(value.prefix(50))
                    }
                }
            fieldMeta(hint: "建议填写门店常用名称，方便大家识别。", count: "\(model.form.name.count)/50")
        }
    }

    private var regionField: some View {
        VStack(alignment: .leading, spacing: 6) {
            fieldLabel("地区", isComplete: model.isRegionValid)
            Button {
                isAreaPickerPresented = true
            } label: {
                HStack(spacing: 8) {
                    Text(model.form.regionText.isEmpty ? model.regionPlaceholder : model.form.regionText)
                        .font(.system(size: 14))
                        .foregroundStyle(model.form.regionText.isEmpty ? palette.textMuted : palette.textPrimary)
                        .lineLimit(1)
                    Spacer(minLength: 0)
                    Text("›")
                        .font(.system(size: 16))
                        .foregroundStyle(palette.textMuted)
                }
                .padding(.horizontal, 12)
                .frame(height: 40)
                .background(palette.subtle, in: RoundedRectangle(cornerRadius: 8))
                .overlay {
                    RoundedRectangle(cornerRadius: 8)
                        .stroke(missingBorder(isMissing: !model.isRegionValid), lineWidth: 1)
                }
            }
            .buttonStyle(.plain)
            .disabled(model.isSubmitting || (model.isLoadingAreas && !model.isAreaReady))

            Text("选择省、市、区后，审核和定位都会更准确。")
                .font(.system(size: 11))
                .foregroundStyle(palette.textMuted)
        }
    }

    private var locationField: some View {
        VStack(alignment: .leading, spacing: 6) {
            HStack(spacing: 6) {
                Text("地图位置")
                    .font(.system(size: 13, weight: .medium))
                    .foregroundStyle(palette.textPrimary)
                Text("可选")
                    .font(.system(size: 11))
                    .foregroundStyle(palette.textMuted)
                Spacer(minLength: 0)
            }

            if let location = model.form.location {
                HStack(spacing: 8) {
                    VStack(alignment: .leading, spacing: 2) {
                        Text(location.name.isEmpty ? "已选位置" : location.name)
                            .font(.system(size: 13))
                            .foregroundStyle(palette.textPrimary)
                        if !location.address.isEmpty {
                            Text(location.address)
                                .font(.system(size: 11))
                                .foregroundStyle(palette.textMuted)
                                .lineLimit(1)
                        }
                    }
                    Spacer(minLength: 0)
                    Button("重新选择") { isLocationPickerPresented = true }
                        .font(.system(size: 12))
                        .foregroundStyle(palette.brandStrong)
                    Button("清除") { model.form.location = nil }
                        .font(.system(size: 12))
                        .foregroundStyle(palette.textMuted)
                }
                .padding(.horizontal, 12)
                .padding(.vertical, 10)
                .background(palette.subtle, in: RoundedRectangle(cornerRadius: 8))
            } else {
                Button {
                    isLocationPickerPresented = true
                } label: {
                    HStack(spacing: 8) {
                        Text("在地图上选点")
                            .font(.system(size: 14))
                            .foregroundStyle(palette.textMuted)
                        Spacer(minLength: 0)
                        Text("›")
                            .font(.system(size: 16))
                            .foregroundStyle(palette.textMuted)
                    }
                    .padding(.horizontal, 12)
                    .frame(height: 40)
                    .background(palette.subtle, in: RoundedRectangle(cornerRadius: 8))
                }
                .buttonStyle(.plain)
            }

            Text("选点后位置更准确，附近球友能更快找到。")
                .font(.system(size: 11))
                .foregroundStyle(palette.textMuted)
        }
    }

    private var addressField: some View {
        VStack(alignment: .leading, spacing: 6) {
            fieldLabel("详细地址", isComplete: model.isAddressValid)
            TextField("请输入门牌号、楼层等详细地址", text: $model.form.address)
                .font(.system(size: 14))
                .foregroundStyle(palette.textPrimary)
                .padding(.horizontal, 12)
                .frame(height: 40)
                .background(palette.subtle, in: RoundedRectangle(cornerRadius: 8))
                .overlay {
                    RoundedRectangle(cornerRadius: 8)
                        .stroke(missingBorder(isMissing: !model.isAddressValid), lineWidth: 1)
                }
                .onChange(of: model.form.address) { _, value in
                    if value.count > 100 {
                        model.form.address = String(value.prefix(100))
                    }
                }
            fieldMeta(hint: "建议包含楼层或门牌，首次到店更容易找到。", count: "\(model.form.address.count)/100")
        }
    }

    private var submitBar: some View {
        HStack(spacing: 12) {
            VStack(alignment: .leading, spacing: 2) {
                Text(model.barCopy.title)
                    .font(.system(size: 13, weight: .semibold))
                    .foregroundStyle(palette.textPrimary)
                Text(model.barCopy.tip)
                    .font(.system(size: 11))
                    .foregroundStyle(palette.textMuted)
            }
            Spacer(minLength: 8)
            Button {
                Task { await model.submit() }
            } label: {
                Text(model.isSubmitting ? "提交中..." : "提交常玩球馆")
                    .font(.system(size: 14, weight: .semibold))
                    .foregroundStyle(.white)
                    .padding(.horizontal, 16)
                    .frame(height: 40)
                    .background(AppTheme.primary, in: RoundedRectangle(cornerRadius: 10))
                    .opacity(model.isSubmitting ? 0.5 : 1)
            }
            .buttonStyle(.plain)
            .disabled(model.isSubmitting)
        }
        .padding(.horizontal, 12)
        .padding(.vertical, 10)
        .background(palette.card)
        .overlay(alignment: .top) {
            Rectangle()
                .fill(palette.border)
                .frame(height: 1)
        }
    }

    private func fieldLabel(_ title: String, isComplete: Bool) -> some View {
        HStack(spacing: 6) {
            Text(title)
                .font(.system(size: 13, weight: .medium))
                .foregroundStyle(palette.textPrimary)
            Text(isComplete ? "已完成" : "必填")
                .font(.system(size: 11))
                .foregroundStyle(isComplete ? palette.success : palette.brandStrong)
            Spacer(minLength: 0)
        }
    }

    private func fieldMeta(hint: String, count: String) -> some View {
        HStack(alignment: .firstTextBaseline, spacing: 8) {
            Text(hint)
                .font(.system(size: 11))
                .foregroundStyle(palette.textMuted)
            Spacer(minLength: 0)
            Text(count)
                .font(.system(size: 11))
                .foregroundStyle(palette.textMuted)
        }
    }

    private func missingBorder(isMissing: Bool) -> Color {
        model.showValidation && isMissing ? AppTheme.danger : palette.border
    }
}

/// 省 / 市 / 区三列滚轮，切换上级后重新加载下级。
private struct VenueAreaPicker: View {
    @ObservedObject var model: VenueSubmitModel
    @Environment(\.dismiss) private var dismiss

    var body: some View {
        NavigationStack {
            HStack(spacing: 0) {
                ForEach(0..<3, id: \.self) { column in
                    Picker("", selection: selection(column)) {
                        ForEach(Array(options(column).enumerated()), id: \.offset) { index, option in
                            Text(option.name).tag(index)
                        }
                    }
                    .pickerStyle(.wheel)
                    .clipped()
                }
            }
            .navigationTitle("选择地区")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button("取消") { dismiss() }
                }
                ToolbarItem(placement: .confirmationAction) {
                    Button("确定") {
                        Task {
                            await model.confirmAreaSelection()
                            dismiss()
                        }
                    }
                    .fontWeight(.semibold)
                }
            }
        }
    }

    private func options(_ column: Int) -> [VenueAreaOption] {
        guard column < model.columns.count else { return [] }
        return model.columns[column]
    }

    private func selection(_ column: Int) -> Binding<Int> {
        Binding(
            get: {
                let count = options(column).count
                guard count > 0, column < model.columnIndexes.count else { return 0 }
                return min(max(model.columnIndexes[column], 0), count - 1)
            },
            set: { index in
                Task { await model.selectAreaColumn(column, index: index) }
            }
        )
    }
}
