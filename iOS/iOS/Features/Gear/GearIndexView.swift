import SwiftUI

/// 我的装备，对应小程序 `subPages/gear/index.vue`。
struct GearIndexView: View {
    private struct PickerState: Identifiable {
        let id = UUID()
        let row: GearMainRow
        let slot: GearSlot
        let title: String
        let candidates: [GearItemInfo]
        let currentId: Int64
        let emptyText: String
    }

    @EnvironmentObject private var session: SessionStore
    @EnvironmentObject private var navigator: AppNavigator
    @EnvironmentObject private var toast: ToastCenter
    @Environment(\.colorScheme) private var colorScheme
    @Environment(\.dismiss) private var dismiss
    @StateObject private var model: GearIndexModel
    @State private var picker: PickerState?

    init(session: SessionStore, store: GearStore, model: GearIndexModel? = nil) {
        _model = StateObject(wrappedValue: model ?? GearIndexModel(session: session, store: store))
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
        .navigationTitle("我的装备")
        .navigationBarTitleDisplayMode(.inline)
        .onAppear { Task { await model.onAppear() } }
        .sheet(item: $picker) { state in
            pickerSheet(state)
        }
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
                } else {
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

            defaultsCard
            sectionHead

            if model.isEmpty {
                emptyCard
            } else {
                groupsContent
            }

            Text("好友可以看到你在用的装备；未收录型号的名称只有你自己能看到。")
                .font(.system(size: 11))
                .foregroundStyle(palette.textMuted)
                .multilineTextAlignment(.center)
                .padding(.horizontal, 4)
                .padding(.top, 2)
        }
    }

    // MARK: - 主力装备

    private var defaultsCard: some View {
        VStack(alignment: .leading, spacing: 0) {
            GearCardHead(title: "主力装备", caption: "对局完成时自动记下用杆")
            ForEach(Array(model.defaultsView.enumerated()), id: \.element.gameType) { index, row in
                if index > 0 {
                    Rectangle()
                        .fill(palette.border)
                        .frame(height: 1)
                        .padding(.top, 10)
                }
                HStack(alignment: .top, spacing: 10) {
                    Text(row.label)
                        .font(.system(size: 13, weight: .semibold))
                        .foregroundStyle(palette.textPrimary)
                        .frame(width: 74, alignment: .leading)
                        .padding(.top, 7)
                    VStack(spacing: 6) {
                        slotButton(row: row, slot: .cue, label: "球杆", item: row.cue)
                        if row.supportsBreak {
                            slotButton(row: row, slot: .breakSlot, label: "冲杆", item: row.breakCue)
                        }
                    }
                }
                .padding(.top, 10)
            }
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .appCard(padding: 14)
    }

    private func slotButton(row: GearMainRow, slot: GearSlot, label: String, item: GearItemInfo?) -> some View {
        Button {
            openPicker(row, slot)
        } label: {
            HStack(spacing: 8) {
                Text(label)
                    .font(.system(size: 11))
                    .foregroundStyle(palette.textMuted)
                Spacer(minLength: 4)
                Text(item.map(GearDisplayRules.itemTitle) ?? "未设置")
                    .font(.system(size: 13))
                    .foregroundStyle(item == nil ? palette.textMuted : palette.textPrimary)
                    .lineLimit(1)
            }
            .padding(.horizontal, 10)
            .padding(.vertical, 6)
            .background(palette.subtle, in: RoundedRectangle(cornerRadius: 8))
            .overlay { RoundedRectangle(cornerRadius: 8).stroke(palette.border) }
        }
        .buttonStyle(.plain)
        .disabled(model.isSaving)
    }

    private var sectionHead: some View {
        HStack(spacing: 8) {
            HStack(alignment: .firstTextBaseline, spacing: 6) {
                Text("我的装备")
                    .font(.system(size: 16, weight: .bold))
                    .foregroundStyle(palette.textPrimary)
                Text(model.countText)
                    .font(.system(size: 12))
                    .foregroundStyle(palette.textMuted)
            }
            Spacer(minLength: 8)
            Button("添加装备") {
                openAdd(category: .cue)
            }
            .font(.system(size: 13, weight: .semibold))
            .foregroundStyle(palette.brandStrong)
            .padding(.horizontal, 14)
            .frame(height: 32)
            .background(palette.subtle, in: Capsule())
            .overlay { Capsule().stroke(palette.border) }
            .buttonStyle(.plain)
            .disabled(model.atLimit)
            .opacity(model.atLimit ? 0.5 : 1)
        }
        .padding(.horizontal, 2)
    }

    private var emptyCard: some View {
        VStack(spacing: 6) {
            Text("还没有装备")
                .font(.system(size: 16, weight: .bold))
                .foregroundStyle(palette.textPrimary)
            Text("登记你的球杆后，每场对局完成时会自动记下这场用的杆、皮头和巧克。")
                .font(.system(size: 13))
                .foregroundStyle(palette.textSecondary)
                .multilineTextAlignment(.center)
            Button("添加球杆") {
                openAdd(category: .cue)
            }
            .buttonStyle(.appPill)
            .padding(.top, 6)
        }
        .frame(maxWidth: .infinity)
        .appCard(padding: 20)
    }

    private var groupsContent: some View {
        VStack(spacing: 8) {
            group(title: "球杆", items: model.groups.cues, showBadges: true)
            group(title: "冲杆 / 跳杆", items: model.groups.breakJump, showBadges: true)
            group(title: "已退役", items: model.groups.retired, showBadges: false)
        }
    }

    @ViewBuilder
    private func group(title: String, items: [GearItemInfo], showBadges: Bool) -> some View {
        if !items.isEmpty {
            VStack(alignment: .leading, spacing: 8) {
                Text(title)
                    .font(.system(size: 13, weight: .semibold))
                    .foregroundStyle(palette.textSecondary)
                LazyVGrid(
                    columns: [GridItem(.flexible(), spacing: 10), GridItem(.flexible(), spacing: 10)],
                    spacing: 10
                ) {
                    ForEach(items) { item in
                        Button {
                            navigator.open(.gearItem(id: item.id))
                        } label: {
                            GearItemCard(
                                item: item,
                                badges: showBadges ? model.mainLabels(for: item) : []
                            )
                        }
                        .buttonStyle(.plain)
                    }
                }
            }
            .frame(maxWidth: .infinity, alignment: .leading)
            .padding(.top, 4)
        }
    }

    // MARK: - 主力选择

    private func openPicker(_ row: GearMainRow, _ slot: GearSlot) {
        let isBreak = slot == .breakSlot
        picker = PickerState(
            row: row,
            slot: slot,
            title: "\(row.label) · 主力\(isBreak ? "冲杆" : "球杆")",
            candidates: GearDisplayRules.mainCandidates(model.items, slot: slot),
            currentId: (isBreak ? row.breakCue : row.cue)?.id ?? 0,
            emptyText: isBreak
                ? "还没有可选的冲杆，先添加一根带冲杆身份的装备。"
                : "还没有可选的球杆，先添加一根球杆。"
        )
    }

    private func pickerSheet(_ state: PickerState) -> some View {
        VStack(alignment: .leading, spacing: 0) {
            HStack {
                Text(state.title)
                    .font(.system(size: 17, weight: .bold))
                    .foregroundStyle(palette.textPrimary)
                Spacer(minLength: 8)
                Button("取消") { picker = nil }
                    .font(.system(size: 13))
                    .foregroundStyle(palette.textMuted)
            }
            .padding(.horizontal, 16)
            .padding(.top, 16)
            .padding(.bottom, 8)

            ScrollView {
                VStack(spacing: 0) {
                    if state.candidates.isEmpty {
                        VStack(spacing: 12) {
                            Text(state.emptyText)
                                .font(.system(size: 13))
                                .foregroundStyle(palette.textSecondary)
                                .multilineTextAlignment(.center)
                            Button("去添加") {
                                let category: GearCategory = state.slot == .breakSlot ? .breakJump : .cue
                                picker = nil
                                openAdd(category: category)
                            }
                            .buttonStyle(.appPill)
                        }
                        .padding(.vertical, 24)
                    } else {
                        ForEach(state.candidates) { item in
                            Button {
                                choose(state, itemId: item.id)
                            } label: {
                                pickerRow(item, isCurrent: item.id == state.currentId)
                            }
                            .buttonStyle(.plain)
                            .disabled(model.isSaving)
                        }
                        if state.currentId != 0 {
                            Button {
                                choose(state, itemId: 0)
                            } label: {
                                Text("不设主力")
                                    .font(.system(size: 14))
                                    .foregroundStyle(AppTheme.danger)
                                    .frame(maxWidth: .infinity)
                                    .padding(.vertical, 14)
                            }
                            .buttonStyle(.plain)
                            .disabled(model.isSaving)
                        }
                    }
                }
                .padding(.horizontal, 16)
            }
        }
        .presentationDetents([.medium, .large])
        .presentationDragIndicator(.visible)
    }

    private func pickerRow(_ item: GearItemInfo, isCurrent: Bool) -> some View {
        HStack(spacing: 10) {
            GearImageView(source: GearImageSource(item: item))
                .frame(width: 70, height: 70)
                .clipShape(RoundedRectangle(cornerRadius: 8))
            VStack(alignment: .leading, spacing: 3) {
                Text(GearDisplayRules.itemTitle(item))
                    .font(.system(size: 14, weight: .semibold))
                    .foregroundStyle(isCurrent ? palette.brandStrong : palette.textPrimary)
                    .multilineTextAlignment(.leading)
                Text(GearDisplayRules.itemSubtitle(item))
                    .font(.system(size: 11))
                    .foregroundStyle(palette.textMuted)
                    .multilineTextAlignment(.leading)
            }
            Spacer(minLength: 8)
            if isCurrent {
                Text("当前")
                    .font(.system(size: 11))
                    .foregroundStyle(palette.brandStrong)
                    .padding(.horizontal, 8)
                    .padding(.vertical, 2)
                    .background(palette.brandTint, in: Capsule())
            }
        }
        .padding(.vertical, 9)
        .contentShape(Rectangle())
        .overlay(alignment: .top) {
            Rectangle().fill(palette.border).frame(height: 1)
        }
    }

    private func choose(_ state: PickerState, itemId: Int64) {
        Task {
            if let message = await model.setDefault(row: state.row, slot: state.slot, itemId: itemId) {
                toast.show(message)
            } else {
                picker = nil
            }
        }
    }

    private func openAdd(category: GearCategory) {
        navigator.open(.gearAdd(category: category.rawValue, pickToken: nil))
    }
}
