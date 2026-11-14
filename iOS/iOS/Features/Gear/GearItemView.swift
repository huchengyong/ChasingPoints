import SwiftUI

/// 装备详情，对应小程序 `subPages/gear/item.vue`。
struct GearItemView: View {
    @EnvironmentObject private var session: SessionStore
    @EnvironmentObject private var navigator: AppNavigator
    @EnvironmentObject private var toast: ToastCenter
    @Environment(\.colorScheme) private var colorScheme
    @Environment(\.dismiss) private var dismiss
    @StateObject private var model: GearItemModel
    @State private var confirmingRetire = false
    @State private var confirmingDelete = false

    private let store: GearStore

    init(session: SessionStore, store: GearStore, itemId: Int64, model: GearItemModel? = nil) {
        self.store = store
        _model = StateObject(wrappedValue: model ?? GearItemModel(session: session, store: store, itemId: itemId))
    }

    private var palette: AppPalette { AppTheme.palette(for: colorScheme) }

    var body: some View {
        Group {
            if session.identity == nil {
                StateMessageView(
                    icon: "lock.fill",
                    title: "请登录后查看装备",
                    actionTitle: "去登录"
                ) {
                    navigator.openLogin()
                }
            } else {
                content
            }
        }
        .background(palette.page.ignoresSafeArea())
        .navigationTitle("装备详情")
        .navigationBarTitleDisplayMode(.inline)
        .onAppear { Task { await model.onAppear() } }
    }

    private var content: some View {
        ScrollView {
            VStack(spacing: 12) {
                if model.isInitialLoading {
                    VStack(spacing: 10) {
                        ProgressView()
                        Text("加载装备中...")
                            .font(.system(size: 15, weight: .semibold))
                            .foregroundStyle(palette.textPrimary)
                    }
                    .frame(maxWidth: .infinity)
                    .padding(.top, 140)
                } else if let failure = model.failure, model.data == nil {
                    StateMessageView(
                        icon: "info.circle",
                        title: failure.title,
                        detail: failure.description,
                        actionTitle: failure.actionText
                    ) {
                        if failure.retryable {
                            Task { await model.retry() }
                        } else {
                            dismiss()
                        }
                    }
                    .padding(.top, 100)
                } else if model.data != nil, model.item == nil {
                    StateMessageView(
                        icon: "info.circle",
                        title: "装备不存在",
                        detail: "这件装备可能已经被删除。",
                        actionTitle: "返回"
                    ) {
                        dismiss()
                    }
                    .padding(.top, 100)
                } else if model.item != nil {
                    loadedContent
                }
            }
            .padding(.horizontal, 12)
            .padding(.vertical, 12)
        }
    }

    private var loadedContent: some View {
        VStack(spacing: 12) {
            if let refreshFailure = model.refreshFailure {
                GearRefreshBanner(message: refreshFailure.description) {
                    Task { await model.retry() }
                }
            }

            if model.justAdded, model.isCue {
                Text("已添加。给这根杆设置现在装的皮头和巧克，对局会按它们记录。")
                    .font(.system(size: 13))
                    .foregroundStyle(palette.textPrimary)
                    .lineSpacing(2)
                    .frame(maxWidth: .infinity, alignment: .leading)
                    .padding(.horizontal, 12)
                    .padding(.vertical, 10)
                    .background(palette.brandTint, in: RoundedRectangle(cornerRadius: 8))
                    .overlay { RoundedRectangle(cornerRadius: 8).stroke(AppTheme.primary) }
            }

            heroCard

            if model.item?.unlisted == true {
                GearNoticeCard(
                    title: "未收录型号",
                    text: "你填写的名称只有你自己能看到，其他人会看到「未收录型号」。型号被收录后，这里会自动换成正式的名称、规格和图片。"
                )
            }

            if !model.specs.isEmpty {
                specsCard
            }

            if model.isCue {
                partsCard
            }

            mainSection

            actionCard
        }
    }

    // MARK: - 主图

    private var heroCard: some View {
        VStack(spacing: 0) {
            GearImageView(source: model.item.map(GearImageSource.init(item:)) ?? GearImageSource(category: GearCategory.cue.rawValue))
                .frame(maxWidth: .infinity)
                .frame(height: 260)
                .clipShape(RoundedRectangle(cornerRadius: 14))
            Text(model.title)
                .font(.system(size: 18, weight: .bold))
                .foregroundStyle(palette.textPrimary)
                .multilineTextAlignment(.center)
                .padding(.top, 14)
            HStack(spacing: 6) {
                if !model.subtitle.isEmpty {
                    Text(model.subtitle)
                        .font(.system(size: 11))
                        .foregroundStyle(palette.brandStrong)
                        .padding(.horizontal, 10)
                        .padding(.vertical, 3)
                        .background(palette.brandTint, in: Capsule())
                }
                if model.retired {
                    Text("已退役")
                        .font(.system(size: 11))
                        .foregroundStyle(palette.textSecondary)
                        .padding(.horizontal, 10)
                        .padding(.vertical, 3)
                        .background(palette.subtle, in: Capsule())
                        .overlay { Capsule().stroke(palette.border) }
                }
            }
            .padding(.top, 8)
        }
        .frame(maxWidth: .infinity)
        .appCard(padding: 14)
    }

    // MARK: - 官方规格

    private var specsCard: some View {
        VStack(alignment: .leading, spacing: 0) {
            GearCardHead(title: "官方规格", caption: "以品牌公布为准")
            ForEach(model.specs, id: \.key) { spec in
                HStack {
                    Text(spec.label)
                        .font(.system(size: 13))
                        .foregroundStyle(palette.textSecondary)
                    Spacer(minLength: 8)
                    Text(spec.value)
                        .font(.system(size: 14, weight: .semibold))
                        .foregroundStyle(palette.textPrimary)
                }
                .padding(.vertical, 9)
                .overlay(alignment: .top) {
                    Rectangle().fill(palette.border).frame(height: 1)
                }
            }
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .appCard(padding: 14)
    }

    // MARK: - 皮头与巧克

    private var partsCard: some View {
        VStack(alignment: .leading, spacing: 0) {
            GearCardHead(title: "皮头与巧克", caption: "对局会记录当时装的组合")
            ForEach(model.parts) { part in
                HStack(spacing: 10) {
                    GearImageView(source: part.subject)
                        .frame(width: 60, height: 60)
                        .clipShape(RoundedRectangle(cornerRadius: 8))
                    VStack(alignment: .leading, spacing: 3) {
                        Text(part.label)
                            .font(.system(size: 11))
                            .foregroundStyle(palette.textMuted)
                        Text(part.text)
                            .font(.system(size: 14, weight: part.product == nil ? .regular : .semibold))
                            .foregroundStyle(part.product == nil ? palette.textMuted : palette.textPrimary)
                            .multilineTextAlignment(.leading)
                    }
                    Spacer(minLength: 8)
                    VStack(alignment: .trailing, spacing: 5) {
                        Button(part.product == nil ? "选择" : "更换") {
                            pickPart(part.category)
                        }
                        .font(.system(size: 13))
                        .foregroundStyle(palette.brandStrong)
                        .buttonStyle(.plain)
                        .disabled(model.isBusy)
                        if part.product != nil {
                            Button("清除") {
                                clearPart(part.category)
                            }
                            .font(.system(size: 13))
                            .foregroundStyle(palette.textMuted)
                            .buttonStyle(.plain)
                            .disabled(model.isBusy)
                        }
                    }
                }
                .padding(.vertical, 9)
                .overlay(alignment: .top) {
                    Rectangle().fill(palette.border).frame(height: 1)
                }
            }

            VStack(alignment: .leading, spacing: 2) {
                Text("找不到你用的皮头或巧克？")
                Button("告诉我们，我们会尽快收录。") {
                    navigator.open(.feedback)
                }
                .foregroundStyle(palette.brandStrong)
                .buttonStyle(.plain)
            }
            .font(.system(size: 11))
            .foregroundStyle(palette.textMuted)
            .padding(.top, 10)
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .appCard(padding: 14)
    }

    // MARK: - 主力

    @ViewBuilder
    private var mainSection: some View {
        if !model.mainRows.isEmpty {
            VStack(alignment: .leading, spacing: 8) {
                GearCardHead(title: "设为主力", caption: "对局完成时自动记录这件装备")
                WrappingLayout(spacing: 6) {
                    ForEach(model.mainRows) { row in
                        Button {
                            toggleMain(row)
                        } label: {
                            GearChip(title: row.label, isActive: row.active)
                        }
                        .buttonStyle(.plain)
                        .disabled(model.isBusy)
                    }
                }
            }
            .frame(maxWidth: .infinity, alignment: .leading)
            .appCard(padding: 14)
        } else if model.retired, model.isCue || model.item?.category == GearCategory.breakJump.rawValue {
            GearNoticeCard(title: nil, text: "退役的装备不能设为主力。恢复使用后可以重新设置。")
        }
    }

    // MARK: - 操作

    private var actionCard: some View {
        VStack(spacing: 8) {
            Button(model.retired ? "恢复使用" : "退役") {
                if model.retired {
                    Task { await retire() }
                } else {
                    confirmingRetire = true
                }
            }
            .buttonStyle(.appSecondary)
            .disabled(model.isBusy)
            .alert("退役这件装备？", isPresented: $confirmingRetire) {
                Button("取消", role: .cancel) {}
                Button("退役") { Task { await retire() } }
            } message: {
                Text("退役后它会收进「已退役」，并取消它的主力设置；历史对局的记录不受影响。")
            }

            Button("删除装备") {
                confirmingDelete = true
            }
            .buttonStyle(.appDanger)
            .disabled(model.isBusy)
            .alert("删除这件装备？", isPresented: $confirmingDelete) {
                Button("取消", role: .cancel) {}
                Button("删除", role: .destructive) { Task { await delete() } }
            } message: {
                Text("删除后不再出现在你的装备列表里；已经记录的历史对局仍会显示它。")
            }

            Text("退役后装备会收进「已退役」，历史对局的记录不受影响；删除后不再出现在装备列表里。")
                .font(.system(size: 11))
                .foregroundStyle(palette.textMuted)
                .multilineTextAlignment(.center)
                .lineSpacing(2)
                .padding(.top, 2)
        }
    }

    // MARK: - 动作

    private func pickPart(_ category: GearCategory) {
        let token = UUID().uuidString
        store.beginPick(token: token) { product in
            Task {
                let outcome = await model.setPart(category, product: product)
                if let message = outcome.message { toast.show(message) }
            }
        }
        navigator.open(.gearAdd(category: category.rawValue, pickToken: token))
    }

    private func clearPart(_ category: GearCategory) {
        Task {
            let outcome = await model.setPart(category, product: nil)
            if let message = outcome.message { toast.show(message) }
        }
    }

    private func toggleMain(_ row: GearItemModel.MainToggle) {
        Task {
            let outcome = await model.toggleMain(row)
            if let message = outcome.message { toast.show(message) }
        }
    }

    private func retire() async {
        let outcome = await model.toggleRetire()
        if let message = outcome.message { toast.show(message) }
    }

    private func delete() async {
        let outcome = await model.delete()
        if let message = outcome.message { toast.show(message) }
        if outcome.succeeded { dismiss() }
    }
}
