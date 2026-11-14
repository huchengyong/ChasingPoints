import SwiftUI

/// 本场用杆：双方在这场对局里用的球杆、皮头、巧克与冲杆，可补记或修改。
struct MatchGearCardView: View {
    @Environment(\.colorScheme) private var colorScheme
    @EnvironmentObject private var toast: ToastCenter
    @StateObject private var model: MatchGearCardModel

    init(session: SessionStore, matchId: Int64, enabled: Bool, model: MatchGearCardModel? = nil) {
        _model = StateObject(wrappedValue: model ?? MatchGearCardModel(
            session: session,
            matchId: matchId,
            enabled: enabled
        ))
    }

    private var palette: AppPalette { AppTheme.palette(for: colorScheme) }

    var body: some View {
        Group {
            switch model.phase {
            case .hidden:
                EmptyView()
            case .loading:
                card {
                    errorLine("加载中...")
                }
            case .failed(let message):
                card {
                    HStack(spacing: 8) {
                        Text(message)
                            .font(.system(size: 13))
                            .foregroundStyle(palette.textMuted)
                        Spacer()
                        Button("重试") { Task { await model.load() } }
                            .font(.system(size: 13))
                            .foregroundStyle(palette.brandStrong)
                    }
                }
            case .ready(let data):
                card {
                    HStack(alignment: .top, spacing: 10) {
                        sideColumn(label: "我", side: data.me, mine: true)
                        if data.hasOpponent {
                            sideColumn(label: "对手", side: data.opponent, mine: false)
                        }
                    }
                    if model.canEdit, !model.deadlineText.isEmpty {
                        Text(model.deadlineText)
                            .font(.system(size: 11))
                            .foregroundStyle(palette.textMuted)
                    }
                }
            }
        }
        .task(id: model.matchId) { await model.load() }
        .onDisappear { model.cancel() }
        .sheet(isPresented: editorBinding) {
            editorSheet
        }
    }

    private func card<Content: View>(@ViewBuilder content: () -> Content) -> some View {
        VStack(alignment: .leading, spacing: 8) {
            HStack {
                Text("本场用杆")
                    .font(.system(size: 15, weight: .bold))
                    .foregroundStyle(palette.textPrimary)
                Spacer()
                if model.canEdit {
                    Button("修改") { model.openEditor() }
                        .font(.system(size: 13))
                        .foregroundStyle(palette.brandStrong)
                }
            }
            content()
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .padding(14)
        .background(palette.card, in: RoundedRectangle(cornerRadius: 16))
    }

    private func errorLine(_ text: String) -> some View {
        HStack {
            Text(text)
                .font(.system(size: 13))
                .foregroundStyle(palette.textMuted)
            Spacer()
        }
    }

    private func sideColumn(label: String, side: GearMatchSide?, mine: Bool) -> some View {
        let view = GearMatchRules.describeSide(side)
        return VStack(alignment: .leading, spacing: 7) {
            Text(label)
                .font(.system(size: 12, weight: .semibold))
                .foregroundStyle(palette.textSecondary)
            if view.recorded {
                ForEach(view.rows, id: \.key) { row in
                    HStack(spacing: 7) {
                        GearThumb(subject: row.subject)
                            .frame(width: 42, height: 42)
                        VStack(alignment: .leading, spacing: 1) {
                            Text(row.label)
                                .font(.system(size: 10))
                                .foregroundStyle(palette.textMuted)
                            Text(row.title)
                                .font(.system(size: 12))
                                .foregroundStyle(palette.textPrimary)
                                .fixedSize(horizontal: false, vertical: true)
                        }
                    }
                }
            } else {
                HStack {
                    Text("未记录")
                        .font(.system(size: 13))
                        .foregroundStyle(palette.textMuted)
                    Spacer()
                    if mine && model.canEdit {
                        Button("补记") { model.openEditor() }
                            .font(.system(size: 13))
                            .foregroundStyle(palette.brandStrong)
                    }
                }
            }
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .padding(9)
        .background(palette.subtle, in: RoundedRectangle(cornerRadius: 9))
    }

    private var editorBinding: Binding<Bool> {
        Binding(
            get: { model.editor.visible },
            set: { presented in
                if !presented { model.closeEditor() }
            }
        )
    }

    private var editorSheet: some View {
        NavigationStack {
            ScrollView {
                VStack(alignment: .leading, spacing: 10) {
                    if model.editor.itemsLoading {
                        errorLine("加载我的装备...")
                    } else if !model.editor.itemsError.isEmpty {
                        HStack(spacing: 8) {
                            Text(model.editor.itemsError)
                                .font(.system(size: 12))
                                .foregroundStyle(palette.textMuted)
                            Spacer()
                            Button("重试") { Task { await model.loadItems() } }
                                .font(.system(size: 12))
                                .foregroundStyle(palette.brandStrong)
                        }
                    } else {
                        section("球杆")
                        option(title: "不记录", active: model.editor.draft.cueItemId == 0) {
                            model.chooseCue(nil)
                        }
                        ForEach(model.cueOptions, id: \.id) { item in
                            itemOption(item, active: model.editor.draft.cueItemId == item.id) {
                                model.chooseCue(item)
                            }
                        }
                        if model.cueOptions.isEmpty {
                            errorLine("还没有球杆，先到「我的 → 装备」添加。")
                        }

                        section("皮头")
                        partRow(title: "皮头", product: model.editor.draft.tipProduct, tip: true)

                        section("巧克")
                        partRow(title: "巧克", product: model.editor.draft.chalkProduct, tip: false)

                        if model.responseSupportsBreak {
                            section("冲杆")
                            option(title: "不记录", active: model.editor.draft.breakItemId == 0) {
                                model.chooseBreak(nil)
                            }
                            ForEach(model.breakOptions, id: \.id) { item in
                                itemOption(item, active: model.editor.draft.breakItemId == item.id) {
                                    model.chooseBreak(item)
                                }
                            }
                        }
                    }
                }
                .padding(16)
            }
            .background(palette.page)
            .navigationTitle("修改本场用杆")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button("取消") { model.closeEditor() }
                        .foregroundStyle(palette.textMuted)
                }
            }
            .safeAreaInset(edge: .bottom) {
                Button {
                    Task { await save() }
                } label: {
                    Text(model.editor.saving ? "保存中..." : "保存")
                        .font(.system(size: 15, weight: .semibold))
                        .frame(maxWidth: .infinity)
                }
                .buttonStyle(.borderedProminent)
                .tint(AppTheme.primary)
                .disabled(model.editor.saving || model.editor.itemsLoading)
                .padding(16)
                .background(palette.card)
            }
        }
        .presentationDetents([.large])
    }

    private func section(_ title: String) -> some View {
        Text(title)
            .font(.system(size: 12, weight: .semibold))
            .foregroundStyle(palette.textSecondary)
            .padding(.top, 4)
    }

    private func option(title: String, active: Bool, action: @escaping () -> Void) -> some View {
        Button(action: action) {
            Text(title)
                .font(.system(size: 13))
                .foregroundStyle(palette.textPrimary)
                .frame(maxWidth: .infinity, alignment: .leading)
                .padding(.horizontal, 8)
                .padding(.vertical, 10)
                .background(optionBackground(active), in: RoundedRectangle(cornerRadius: 9))
        }
        .buttonStyle(.plain)
    }

    private func itemOption(_ item: GearItemInfo, active: Bool, action: @escaping () -> Void) -> some View {
        Button(action: action) {
            HStack(spacing: 8) {
                GearThumb(subject: .item(item))
                    .frame(width: 44, height: 44)
                Text(GearDisplay.itemTitle(item))
                    .font(.system(size: 13))
                    .foregroundStyle(palette.textPrimary)
                    .frame(maxWidth: .infinity, alignment: .leading)
                if item.status == GearItemStatus.retired {
                    Text("已退役")
                        .font(.system(size: 10))
                        .foregroundStyle(palette.textMuted)
                }
            }
            .padding(.horizontal, 8)
            .padding(.vertical, 7)
            .background(optionBackground(active), in: RoundedRectangle(cornerRadius: 9))
        }
        .buttonStyle(.plain)
    }

    private func optionBackground(_ active: Bool) -> Color {
        active ? palette.brandTint : palette.subtle
    }

    private func partRow(title: String, product: GearProductInfo?, tip: Bool) -> some View {
        HStack(spacing: 6) {
            Text(product == nil ? "未设置" : GearDisplay.productTitle(product))
                .font(.system(size: 13))
                .foregroundStyle(product == nil ? palette.textMuted : palette.textPrimary)
                .frame(maxWidth: .infinity, alignment: .leading)
            if product != nil {
                Button("清除") { model.clearPart(tip: tip) }
                    .font(.system(size: 13))
                    .foregroundStyle(palette.textMuted)
            }
        }
        .padding(.horizontal, 8)
        .padding(.vertical, 10)
        .background(palette.subtle, in: RoundedRectangle(cornerRadius: 9))
        .accessibilityElement(children: .combine)
        .accessibilityLabel("\(title)\(product == nil ? "未设置" : GearDisplay.productTitle(product))")
    }

    private func save() async {
        if let message = await model.save() {
            toast.show(message)
        } else {
            toast.show("已保存")
        }
    }
}

/// 装备缩略图：目录图缺失或加载失败时回落到按类别选出的默认图。
private struct GearThumb: View {
    @Environment(\.colorScheme) private var colorScheme
    let subject: GearMatchRow.Subject

    var body: some View {
        let image: GearImage = switch subject {
        case .item(let item): GearDisplay.image(item)
        case .product(let product): GearDisplay.image(product)
        }
        RemoteImage(url: image.src, contentMode: .fit) {
            RemoteImage(url: image.fallback, contentMode: .fit) {
                Color.clear
            }
        }
        .background(AppTheme.palette(for: colorScheme).subtle, in: RoundedRectangle(cornerRadius: 6))
        .accessibilityHidden(true)
    }
}
