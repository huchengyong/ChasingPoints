import SwiftUI

/// 添加装备 / 选择皮头与巧克，对应小程序 `subPages/gear/add.vue`。
struct GearAddView: View {
    @EnvironmentObject private var session: SessionStore
    @EnvironmentObject private var navigator: AppNavigator
    @EnvironmentObject private var toast: ToastCenter
    @Environment(\.colorScheme) private var colorScheme
    @Environment(\.dismiss) private var dismiss
    @StateObject private var model: GearAddModel
    @State private var keywordText = ""
    @State private var pendingProduct: GearProductInfo?
    @State private var started = false

    private let store: GearStore

    init(
        session: SessionStore,
        store: GearStore,
        category: String,
        pickToken: String?,
        model: GearAddModel? = nil
    ) {
        self.store = store
        _model = StateObject(wrappedValue: model ?? GearAddModel(
            session: session,
            store: store,
            requestedCategory: category,
            isPick: pickToken != nil,
            pickToken: pickToken
        ))
    }

    private var palette: AppPalette { AppTheme.palette(for: colorScheme) }

    var body: some View {
        Group {
            if session.identity == nil {
                StateMessageView(
                    icon: "lock.fill",
                    title: "请登录后添加装备",
                    actionTitle: "去登录"
                ) {
                    navigator.openLogin()
                }
            } else {
                content
            }
        }
        .background(palette.page.ignoresSafeArea())
        .navigationTitle(model.isPick ? "选择\(model.categoryLabel)" : "添加装备")
        .navigationBarTitleDisplayMode(.inline)
        .task {
            guard !started else { return }
            started = true
            await model.start()
        }
        .onDisappear {
            if let token = model.pickToken { store.cancelPick(token: token) }
        }
        .alert("添加装备", isPresented: pendingBinding, presenting: pendingProduct) { product in
            Button("取消", role: .cancel) { pendingProduct = nil }
            Button("添加") { create(from: product) }
        } message: { product in
            Text("添加「\(GearDisplayRules.productTitle(product))」到我的装备？")
        }
    }

    private var pendingBinding: Binding<Bool> {
        Binding(
            get: { pendingProduct != nil },
            set: { if !$0 { pendingProduct = nil } }
        )
    }

    private var content: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 0) {
                if model.isPick {
                    pickBanner
                } else {
                    categoryTabs
                }
                searchBox
                brandScroll
                if let house = model.houseProduct, model.keyword.isEmpty, model.brandId == 0 {
                    houseRow(house)
                }
                listContent
                if !model.isPick {
                    unlistedSection
                }
            }
            .padding(.horizontal, 12)
            .padding(.bottom, 40)
        }
    }

    // MARK: - 顶部

    private var categoryTabs: some View {
        HStack(spacing: 6) {
            ForEach([GearCategory.cue, .breakJump], id: \.rawValue) { tab in
                Button {
                    model.switchCategory(tab)
                } label: {
                    Text(tab.label)
                        .font(.system(size: 13, weight: model.category == tab ? .semibold : .regular))
                        .foregroundStyle(model.category == tab ? .white : palette.textSecondary)
                        .frame(maxWidth: .infinity)
                        .frame(height: 36)
                        .background(model.category == tab ? AppTheme.primary : palette.card, in: Capsule())
                        .overlay { Capsule().stroke(model.category == tab ? AppTheme.primary : palette.border) }
                }
                .buttonStyle(.plain)
            }
        }
        .padding(.top, 10)
        .padding(.bottom, 10)
    }

    private var pickBanner: some View {
        HStack {
            Text("选择\(model.categoryLabel)")
                .font(.system(size: 16, weight: .bold))
                .foregroundStyle(palette.textPrimary)
            Spacer(minLength: 8)
            Button("不使用") { finishPick(nil) }
                .font(.system(size: 13))
                .foregroundStyle(palette.brandStrong)
        }
        .padding(.horizontal, 2)
        .padding(.top, 12)
        .padding(.bottom, 10)
    }

    private var searchBox: some View {
        HStack(spacing: 6) {
            Image(systemName: "magnifyingglass")
                .font(.system(size: 15))
                .foregroundStyle(palette.textMuted)
            TextField("搜索品牌或型号", text: $keywordText)
                .font(.system(size: 14))
                .foregroundStyle(palette.textPrimary)
                .submitLabel(.search)
                .onSubmit { model.applyKeyword() }
                .onChange(of: keywordText) { _, text in model.updateKeyword(text) }
            if !keywordText.isEmpty {
                Button("清除") {
                    keywordText = ""
                    model.clearKeyword()
                }
                .font(.system(size: 12))
                .foregroundStyle(palette.textMuted)
            }
        }
        .padding(.horizontal, 12)
        .frame(height: 40)
        .background(palette.card, in: Capsule())
        .overlay { Capsule().stroke(palette.border) }
    }

    @ViewBuilder
    private var brandScroll: some View {
        if !model.brands.isEmpty {
            ScrollView(.horizontal, showsIndicators: false) {
                HStack(spacing: 6) {
                    brandChip(title: "全部品牌", active: model.brandId == 0) {
                        model.selectBrand(0)
                    }
                    ForEach(model.brands) { brand in
                        brandChip(title: brand.name, active: model.brandId == brand.id) {
                            model.selectBrand(brand.id)
                        }
                    }
                }
                .padding(.vertical, 10)
            }
        }
    }

    private func brandChip(title: String, active: Bool, action: @escaping () -> Void) -> some View {
        Button(action: action) {
            Text(title)
                .font(.system(size: 12, weight: active ? .semibold : .regular))
                .foregroundStyle(active ? palette.brandStrong : palette.textSecondary)
                .padding(.horizontal, 13)
                .padding(.vertical, 5)
                .background(active ? palette.brandTint : palette.card, in: Capsule())
                .overlay { Capsule().stroke(active ? AppTheme.primary : palette.border) }
        }
        .buttonStyle(.plain)
    }

    private func houseRow(_ house: GearProductInfo) -> some View {
        Button {
            select(house)
        } label: {
            HStack(spacing: 10) {
                GearImageView(source: GearImageSource(product: house))
                    .frame(width: 70, height: 70)
                    .clipShape(RoundedRectangle(cornerRadius: 8))
                VStack(alignment: .leading, spacing: 3) {
                    Text(house.name)
                        .font(.system(size: 15, weight: .bold))
                        .foregroundStyle(palette.textPrimary)
                        .multilineTextAlignment(.leading)
                    Text(model.houseHint)
                        .font(.system(size: 12))
                        .foregroundStyle(palette.textSecondary)
                }
                Spacer(minLength: 8)
                Image(systemName: "chevron.right")
                    .font(.system(size: 13))
                    .foregroundStyle(palette.textMuted)
            }
            .padding(.horizontal, 12)
            .padding(.vertical, 9)
            .background(palette.brandTint, in: RoundedRectangle(cornerRadius: 14))
            .overlay { RoundedRectangle(cornerRadius: 14).stroke(AppTheme.primary) }
        }
        .buttonStyle(.plain)
        .padding(.top, 2)
    }

    // MARK: - 型号列表

    @ViewBuilder
    private var listContent: some View {
        if model.listLoading && model.products.isEmpty {
            VStack(spacing: 8) {
                ProgressView()
                Text("加载型号中...")
                    .font(.system(size: 15, weight: .semibold))
                    .foregroundStyle(palette.textPrimary)
            }
            .frame(maxWidth: .infinity)
            .padding(.vertical, 60)
        } else if let message = model.listError, model.products.isEmpty {
            VStack(spacing: 8) {
                Image(systemName: "info.circle")
                    .font(.system(size: 36))
                    .foregroundStyle(palette.textMuted)
                Text("型号加载失败")
                    .font(.system(size: 15, weight: .semibold))
                    .foregroundStyle(palette.textPrimary)
                Text(message)
                    .font(.system(size: 13))
                    .foregroundStyle(palette.textSecondary)
                    .multilineTextAlignment(.center)
                Button("重试") { model.applyKeyword() }
                    .buttonStyle(.appPill)
                    .padding(.top, 2)
            }
            .frame(maxWidth: .infinity)
            .padding(.vertical, 48)
        } else if model.products.isEmpty {
            VStack(spacing: 4) {
                Text("没有找到匹配的型号")
                    .font(.system(size: 15, weight: .semibold))
                    .foregroundStyle(palette.textPrimary)
                if !model.isPick {
                    Text("可以换个关键词，或在下方登记未收录型号。")
                        .font(.system(size: 13))
                        .foregroundStyle(palette.textSecondary)
                }
            }
            .frame(maxWidth: .infinity)
            .padding(.vertical, 50)
        } else {
            VStack(spacing: 0) {
                ForEach(model.products) { product in
                    Button {
                        select(product)
                    } label: {
                        productRow(product)
                    }
                    .buttonStyle(.plain)
                    .onAppear {
                        if product.id == model.products.last?.id { model.loadMore() }
                    }
                }
                footer
            }
            .background(palette.card, in: RoundedRectangle(cornerRadius: 14))
            .overlay { RoundedRectangle(cornerRadius: 14).stroke(palette.border) }
            .clipShape(RoundedRectangle(cornerRadius: 14))
            .padding(.top, 10)
        }
    }

    private func productRow(_ product: GearProductInfo) -> some View {
        HStack(spacing: 10) {
            GearImageView(source: GearImageSource(product: product))
                .frame(width: 80, height: 80)
                .clipShape(RoundedRectangle(cornerRadius: 8))
            VStack(alignment: .leading, spacing: 4) {
                Text(GearDisplayRules.productTitle(product))
                    .font(.system(size: 14, weight: .semibold))
                    .foregroundStyle(palette.textPrimary)
                    .multilineTextAlignment(.leading)
                let subtitle = GearDisplayRules.productSubtitle(product)
                if !subtitle.isEmpty {
                    Text(subtitle)
                        .font(.system(size: 11))
                        .foregroundStyle(palette.textMuted)
                        .multilineTextAlignment(.leading)
                }
            }
            Spacer(minLength: 8)
            Image(systemName: "chevron.right")
                .font(.system(size: 13))
                .foregroundStyle(palette.textMuted)
        }
        .padding(.horizontal, 12)
        .padding(.vertical, 10)
        .overlay(alignment: .top) {
            Rectangle().fill(palette.border).frame(height: 1)
        }
        .contentShape(Rectangle())
    }

    @ViewBuilder
    private var footer: some View {
        if model.listLoading {
            Text("加载中...")
                .font(.system(size: 12))
                .foregroundStyle(palette.textMuted)
                .frame(maxWidth: .infinity)
                .padding(.vertical, 12)
        } else if model.listError != nil {
            Button("加载失败，点击重试") { model.loadMore() }
                .font(.system(size: 12))
                .foregroundStyle(palette.brandStrong)
                .frame(maxWidth: .infinity)
                .padding(.vertical, 12)
                .buttonStyle(.plain)
        } else if !model.hasMore {
            Text("已经到底了")
                .font(.system(size: 12))
                .foregroundStyle(palette.textMuted)
                .frame(maxWidth: .infinity)
                .padding(.vertical, 12)
        }
    }

    // MARK: - 未收录型号

    private var unlistedSection: some View {
        VStack(spacing: 0) {
            Button {
                model.toggleUnlisted()
            } label: {
                HStack(spacing: 8) {
                    VStack(alignment: .leading, spacing: 3) {
                        Text("找不到我的型号？")
                            .font(.system(size: 14, weight: .semibold))
                            .foregroundStyle(palette.textPrimary)
                        Text("先登记为「未收录型号」，我们收录后会自动更新")
                            .font(.system(size: 11))
                            .foregroundStyle(palette.textMuted)
                    }
                    Spacer(minLength: 8)
                    Image(systemName: model.showUnlisted ? "chevron.up" : "chevron.down")
                        .font(.system(size: 13))
                        .foregroundStyle(palette.textMuted)
                }
                .padding(.horizontal, 14)
                .padding(.vertical, 12)
                .contentShape(Rectangle())
            }
            .buttonStyle(.plain)
            .background(palette.card, in: RoundedRectangle(cornerRadius: 14))
            .overlay {
                RoundedRectangle(cornerRadius: 14)
                    .stroke(palette.border, style: StrokeStyle(dash: [4, 3]))
            }
            .padding(.top, 14)

            if model.showUnlisted {
                unlistedForm
            }
        }
    }

    private var unlistedForm: some View {
        VStack(alignment: .leading, spacing: 0) {
            formLabel("品牌").padding(.top, 0)
            WrappingLayout(spacing: 6) {
                ForEach(model.brands) { brand in
                    Button {
                        model.chooseFormBrand(brand.id)
                    } label: {
                        GearChip(title: brand.name, isActive: model.form.brandId == brand.id)
                    }
                    .buttonStyle(.plain)
                }
                Button {
                    model.chooseFormBrand(0)
                } label: {
                    GearChip(title: "其他品牌", isActive: model.form.brandId == 0)
                }
                .buttonStyle(.plain)
            }

            if model.form.brandId == 0 {
                TextField("填写品牌名称", text: $model.form.brandName)
                    .font(.system(size: 14))
                    .padding(.horizontal, 12)
                    .frame(height: 40)
                    .background(palette.subtle, in: RoundedRectangle(cornerRadius: 8))
                    .overlay { RoundedRectangle(cornerRadius: 8).stroke(palette.border) }
                    .padding(.top, 8)
            }

            formLabel("型号名称")
            TextField("例如：杆身系列 + 型号", text: $model.form.modelName)
                .font(.system(size: 14))
                .padding(.horizontal, 12)
                .frame(height: 40)
                .background(palette.subtle, in: RoundedRectangle(cornerRadius: 8))
                .overlay { RoundedRectangle(cornerRadius: 8).stroke(palette.border) }

            if model.category == .cue {
                formLabel("杆型")
                WrappingLayout(spacing: 6) {
                    ForEach(GearFormRules.cueStyleOptions) { option in
                        Button {
                            model.chooseCueStyle(option.value)
                        } label: {
                            GearChip(title: option.label, isActive: model.form.cueStyle == option.value)
                        }
                        .buttonStyle(.plain)
                    }
                }
            } else {
                formLabel("身份（可多选）")
                WrappingLayout(spacing: 6) {
                    ForEach(GearFormRules.roleOptions) { option in
                        Button {
                            model.chooseRole(option.value)
                        } label: {
                            GearChip(title: option.label, isActive: model.form.roles.contains(option.value))
                        }
                        .buttonStyle(.plain)
                    }
                }
            }

            Text("你填写的品牌和型号名称只有你自己能看到，其他人会看到「未收录型号」。")
                .font(.system(size: 11))
                .foregroundStyle(palette.textMuted)
                .lineSpacing(2)
                .padding(.top, 12)

            Button(model.submitting ? "登记中..." : "登记这件装备") {
                submitUnlisted()
            }
            .buttonStyle(.appPrimary)
            .disabled(model.submitting)
            .padding(.top, 12)
        }
        .padding(14)
        .frame(maxWidth: .infinity, alignment: .leading)
        .background(palette.card, in: RoundedRectangle(cornerRadius: 14))
        .overlay { RoundedRectangle(cornerRadius: 14).stroke(palette.border) }
        .padding(.top, 8)
    }

    private func formLabel(_ text: String) -> some View {
        Text(text)
            .font(.system(size: 12, weight: .semibold))
            .foregroundStyle(palette.textSecondary)
            .padding(.top, 14)
            .padding(.bottom, 6)
    }

    // MARK: - 动作

    private func select(_ product: GearProductInfo) {
        if model.isPick {
            finishPick(product)
        } else {
            pendingProduct = product
        }
    }

    private func finishPick(_ product: GearProductInfo?) {
        guard let token = model.pickToken else { return }
        store.finishPick(token: token, product: product)
        dismiss()
    }

    private func create(from product: GearProductInfo) {
        Task {
            guard let outcome = await model.create(from: product) else { return }
            handle(outcome)
        }
    }

    private func submitUnlisted() {
        Task {
            guard let outcome = await model.submitUnlisted() else { return }
            handle(outcome)
        }
    }

    private func handle(_ outcome: GearAddModel.CreateOutcome) {
        switch outcome {
        case .failed(let message):
            toast.show(message)
        case .created(let itemId):
            // 原生没有 redirectTo：先 push 详情页，返回时会回到本页。
            if let itemId {
                navigator.open(.gearItem(id: itemId))
            } else {
                navigator.open(.gearIndex)
            }
        }
    }
}
