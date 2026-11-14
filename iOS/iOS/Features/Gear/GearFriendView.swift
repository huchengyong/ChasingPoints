import SwiftUI

/// 好友装备，对应小程序 `subPages/gear/friend.vue`。
struct GearFriendView: View {
    @EnvironmentObject private var session: SessionStore
    @EnvironmentObject private var navigator: AppNavigator
    @Environment(\.colorScheme) private var colorScheme
    @Environment(\.dismiss) private var dismiss
    @StateObject private var model: GearFriendModel

    init(session: SessionStore, userId: Int64, name: String, model: GearFriendModel? = nil) {
        let ownerName = name.trimmingCharacters(in: .whitespacesAndNewlines)
        let resolvedName = ownerName.isEmpty ? "TA" : ownerName
        _model = StateObject(wrappedValue: model ?? GearFriendModel(
            session: session,
            userId: userId,
            ownerName: resolvedName
        ))
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
        .navigationTitle("TA 的装备")
        .navigationBarTitleDisplayMode(.inline)
        .task { await model.load() }
    }

    private var content: some View {
        ScrollView {
            VStack(spacing: 12) {
                if model.loading {
                    VStack(spacing: 10) {
                        ProgressView()
                        Text("加载装备中...")
                            .font(.system(size: 15, weight: .semibold))
                            .foregroundStyle(palette.textPrimary)
                    }
                    .frame(maxWidth: .infinity)
                    .padding(.top, 140)
                } else if let failure = model.failure {
                    StateMessageView(
                        icon: "info.circle",
                        title: failure.title,
                        detail: failure.description,
                        actionTitle: failure.retryable ? "重试" : "返回"
                    ) {
                        if failure.retryable {
                            Task { await model.load() }
                        } else {
                            dismiss()
                        }
                    }
                    .padding(.top, 100)
                } else {
                    ownerLine
                    if !model.mainRows.isEmpty {
                        mainCard
                    }
                    if model.items.isEmpty {
                        emptyCard
                    } else {
                        groupsContent
                    }
                }
            }
            .padding(.horizontal, 12)
            .padding(.vertical, 12)
        }
    }

    private var ownerLine: some View {
        HStack(alignment: .firstTextBaseline, spacing: 8) {
            Text(model.ownerTitle)
                .font(.system(size: 17, weight: .bold))
                .foregroundStyle(palette.textPrimary)
            Spacer(minLength: 8)
            Text("只展示在用的装备")
                .font(.system(size: 11))
                .foregroundStyle(palette.textMuted)
        }
        .padding(.horizontal, 2)
    }

    private var mainCard: some View {
        VStack(alignment: .leading, spacing: 0) {
            GearCardHead(title: "主力装备")
            ForEach(Array(model.mainRows.enumerated()), id: \.element.gameType) { index, row in
                if index > 0 {
                    Rectangle()
                        .fill(palette.border)
                        .frame(height: 1)
                        .padding(.top, 9)
                }
                HStack(alignment: .top, spacing: 10) {
                    Text(row.label)
                        .font(.system(size: 13, weight: .semibold))
                        .foregroundStyle(palette.textPrimary)
                        .frame(width: 74, alignment: .leading)
                    VStack(alignment: .leading, spacing: 4) {
                        Text("球杆：\(row.cue.map(GearDisplayRules.itemTitle) ?? "未设置")")
                        if row.supportsBreak {
                            Text("冲杆：\(row.breakCue.map(GearDisplayRules.itemTitle) ?? "未设置")")
                        }
                    }
                    .font(.system(size: 13))
                    .foregroundStyle(palette.textSecondary)
                    .frame(maxWidth: .infinity, alignment: .leading)
                }
                .padding(.vertical, 9)
            }
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .appCard(padding: 14)
    }

    private var emptyCard: some View {
        VStack(spacing: 6) {
            Text("TA 还没有展示装备")
                .font(.system(size: 15, weight: .bold))
                .foregroundStyle(palette.textPrimary)
            Text("TA 登记球杆后，会显示在这里。")
                .font(.system(size: 13))
                .foregroundStyle(palette.textSecondary)
        }
        .frame(maxWidth: .infinity)
        .appCard(padding: 20)
    }

    private var groupsContent: some View {
        VStack(spacing: 8) {
            group(title: "球杆", items: model.groups.cues)
            group(title: "冲杆 / 跳杆", items: model.groups.breakJump)
        }
    }

    @ViewBuilder
    private func group(title: String, items: [GearItemInfo]) -> some View {
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
                        GearItemCard(item: item, badges: model.mainLabels(for: item))
                    }
                }
            }
            .frame(maxWidth: .infinity, alignment: .leading)
            .padding(.top, 4)
        }
    }
}
