import SwiftUI

/// 好友主页，对应小程序 `subPages/social/friendHomepage.vue`。
struct FriendHomepageView: View {
    @EnvironmentObject private var session: SessionStore
    @Environment(\.colorScheme) private var colorScheme
    @Environment(\.dismiss) private var dismiss

    @StateObject private var model: FriendHomepageModel

    init(session: SessionStore, target: FriendHomepageTarget, model: FriendHomepageModel? = nil) {
        _model = StateObject(wrappedValue: model ?? FriendHomepageModel(session: session, target: target))
    }

    private var palette: AppPalette { AppTheme.palette(for: colorScheme) }

    var body: some View {
        Group {
            switch model.loader.phase {
            case .idle, .loading:
                loadingState
            case .failed:
                failureState
            case .loaded:
                content
            }
        }
        .background(palette.page.ignoresSafeArea())
        .navigationTitle("好友主页")
        .navigationBarTitleDisplayMode(.inline)
        .task { await model.onAppear() }
        .onChange(of: session.identity) { _, _ in
            Task { await model.identityChanged() }
        }
    }

    private var content: some View {
        ScrollView {
            VStack(spacing: 12) {
                if let refreshError = model.loader.refreshError {
                    refreshBanner(refreshError)
                }
                heroCard
                battleSection
            }
            .padding(.horizontal, 12)
            .padding(.vertical, 12)
        }
        .refreshable { await model.refresh() }
    }

    private var heroCard: some View {
        let summary = model.summary

        return VStack(alignment: .leading, spacing: 14) {
            Text("好友名片")
                .font(.system(size: 11, weight: .semibold))
                .foregroundStyle(palette.textMuted)

            HStack(spacing: 12) {
                profileAvatar
                VStack(alignment: .leading, spacing: 5) {
                    HStack(spacing: 6) {
                        Text(FriendRules.displayName(model.target.name))
                            .font(.system(size: 18, weight: .bold))
                            .foregroundStyle(palette.textPrimary)
                            .lineLimit(1)
                        if !model.target.rankName.isEmpty {
                            Text(model.target.rankName)
                                .font(.system(size: 11))
                                .foregroundStyle(AppTheme.warning)
                                .padding(.horizontal, 7)
                                .padding(.vertical, 2)
                                .background(AppTheme.warning.opacity(0.12), in: Capsule())
                        }
                    }
                    Text("ID: \(model.target.userId > 0 ? String(model.target.userId) : "--")")
                        .font(.system(size: 12))
                        .foregroundStyle(palette.textMuted)
                }
                Spacer(minLength: 0)
            }

            HStack(alignment: .top, spacing: 10) {
                Text("已是好友")
                    .font(.system(size: 11, weight: .semibold))
                    .foregroundStyle(palette.brandStrong)
                    .padding(.horizontal, 8)
                    .padding(.vertical, 3)
                    .background(palette.brandTint, in: Capsule())
                Text("这里直接看 TA 的公开战绩，想看你们之间的对比就去 PK 报表。")
                    .font(.system(size: 12))
                    .foregroundStyle(palette.textSecondary)
                    .fixedSize(horizontal: false, vertical: true)
            }

            VStack(alignment: .leading, spacing: 4) {
                Text("好友近况")
                    .font(.system(size: 11, weight: .semibold))
                    .foregroundStyle(palette.brandStrong)
                Text(summary.heroTitle)
                    .font(.system(size: 16, weight: .semibold))
                    .foregroundStyle(palette.textPrimary)
                Text(summary.heroDesc)
                    .font(.system(size: 12))
                    .foregroundStyle(palette.textSecondary)
            }

            ViewThatFits(in: .horizontal) {
                HStack(spacing: 6) {
                    factChip(summary.totalMatchesText)
                    factChip(summary.recordText)
                    factChip(summary.recentMatchText)
                }
                VStack(alignment: .leading, spacing: 6) {
                    factChip(summary.totalMatchesText)
                    factChip(summary.recordText)
                    factChip(summary.recentMatchText)
                }
            }

            VStack(spacing: 8) {
                NavigationLink(value: AppRoute.friendPkReport(userId: model.target.userId)) {
                    Text("PK 报表")
                }
                .buttonStyle(.appPrimary)

                NavigationLink(value: AppRoute.friendHonorWall(userId: model.target.userId)) {
                    Text("查看荣誉墙")
                }
                .buttonStyle(.appSecondary)

                NavigationLink(value: AppRoute.gearFriend(userId: model.target.userId, name: model.target.name)) {
                    Text("TA 的装备")
                }
                .buttonStyle(.appSecondary)
            }
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .appCard()
    }

    @ViewBuilder
    private var profileAvatar: some View {
        if !model.target.avatar.isEmpty || model.target.vipLevel > 0 {
            MemberAvatarView(
                avatar: model.target.avatar,
                userId: model.target.userId,
                vipLevel: model.target.vipLevel,
                rankLevel: model.target.rankLevel,
                size: 60,
                reserveFrame: true
            )
        } else {
            Text(avatarText)
                .font(.system(size: 22, weight: .bold))
                .foregroundStyle(palette.brandStrong)
                .frame(width: 60, height: 60)
                .background(palette.brandTint, in: Circle())
        }
    }

    private var avatarText: String {
        guard let first = model.target.name.first else { return "友" }
        return String(first).uppercased()
    }

    private func factChip(_ text: String) -> some View {
        Text(text)
            .font(.system(size: 11))
            .foregroundStyle(palette.textSecondary)
            .lineLimit(1)
            .padding(.horizontal, 8)
            .padding(.vertical, 4)
            .background(palette.subtle, in: Capsule())
    }

    private var battleSection: some View {
        let summary = model.summary

        return VStack(alignment: .leading, spacing: 12) {
            VStack(alignment: .leading, spacing: 3) {
                Text(summary.battleSectionTitle)
                    .font(.system(size: 16, weight: .semibold))
                    .foregroundStyle(palette.textPrimary)
                Text(summary.battleSectionTip)
                    .font(.system(size: 12))
                    .foregroundStyle(palette.textMuted)
            }

            if model.hidden {
                battleEmpty(title: summary.hiddenTitle, detail: summary.hiddenDesc)
            } else if model.cards.isEmpty {
                battleEmpty(title: summary.emptyTitle, detail: summary.emptyDesc)
            } else {
                VStack(spacing: 8) {
                    ForEach(model.cards) { card in
                        NavigationLink(value: h2hRoute(card)) {
                            battleRow(card)
                        }
                        .buttonStyle(.plain)
                    }
                }
            }
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .appCard()
    }

    private func battleEmpty(title: String, detail: String) -> some View {
        VStack(alignment: .leading, spacing: 4) {
            Text(title)
                .font(.system(size: 14, weight: .semibold))
                .foregroundStyle(palette.textPrimary)
            Text(detail)
                .font(.system(size: 12))
                .foregroundStyle(palette.textMuted)
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .padding(12)
        .background(palette.subtle, in: RoundedRectangle(cornerRadius: 10))
    }

    private func battleRow(_ card: OpponentCard) -> some View {
        HStack(alignment: .top, spacing: 10) {
            battleAvatar(card)

            VStack(alignment: .leading, spacing: 4) {
                HStack(spacing: 5) {
                    Text(card.opponent.name.isEmpty ? "对手" : card.opponent.name)
                        .font(.system(size: 14, weight: .semibold))
                        .foregroundStyle(palette.textPrimary)
                        .lineLimit(1)
                    if card.isMeOpponent {
                        Text("我")
                            .font(.system(size: 9, weight: .bold))
                            .foregroundStyle(.white)
                            .padding(.horizontal, 4)
                            .padding(.vertical, 1)
                            .background(AppTheme.primary, in: Capsule())
                    }
                    Text(card.relationshipBadge)
                        .font(.system(size: 10, weight: .semibold))
                        .foregroundStyle(toneColor(card.tone))
                        .padding(.horizontal, 6)
                        .padding(.vertical, 1)
                        .background(toneColor(card.tone).opacity(0.14), in: Capsule())
                }
                Text(card.relationshipText)
                    .font(.system(size: 11))
                    .foregroundStyle(palette.textSecondary)
                    .lineLimit(1)
                HStack(spacing: 6) {
                    metaText(card.recordText)
                    metaText(card.lastMatchText)
                }
            }

            Spacer(minLength: 0)

            VStack(alignment: .trailing, spacing: 4) {
                Text(card.winRateText)
                    .font(.system(size: 13, weight: .bold).monospacedDigit())
                    .foregroundStyle(toneColor(card.tone))
                Text(card.sampleText)
                    .font(.system(size: 10))
                    .foregroundStyle(palette.textMuted)
                Text("查看交锋")
                    .font(.system(size: 10, weight: .semibold))
                    .foregroundStyle(palette.brandStrong)
            }
        }
        .padding(10)
        .background(palette.subtle, in: RoundedRectangle(cornerRadius: 10))
        .overlay {
            RoundedRectangle(cornerRadius: 10).stroke(palette.border, lineWidth: 1)
        }
        .contentShape(Rectangle())
    }

    @ViewBuilder
    private func battleAvatar(_ card: OpponentCard) -> some View {
        if card.opponent.avatar.isEmpty {
            Text(avatarText(card.opponent.name))
                .font(.system(size: 13, weight: .bold))
                .foregroundStyle(palette.brandStrong)
                .frame(width: 36, height: 36)
                .background(palette.brandTint, in: Circle())
        } else {
            AvatarView(avatar: card.opponent.avatar, userId: card.opponent.id, size: 36)
        }
    }

    private func avatarText(_ name: String) -> String {
        guard let first = (name.isEmpty ? "友" : name).first else { return "友" }
        return String(first).uppercased()
    }

    private func metaText(_ text: String) -> some View {
        Text(text)
            .font(.system(size: 10))
            .foregroundStyle(palette.textSecondary)
            .lineLimit(1)
    }

    private func toneColor(_ tone: OpponentTone) -> Color {
        switch tone {
        case .lead: palette.success
        case .trail: AppTheme.danger
        case .balanced: palette.brandStrong
        case .waiting: palette.textSecondary
        }
    }

    private func h2hRoute(_ card: OpponentCard) -> AppRoute {
        OpponentRecordRules.h2hRoute(
            opponent: card.opponent,
            targetUserId: model.target.userId,
            targetName: model.target.name,
            targetAvatar: model.target.avatar
        )
    }

    private var loadingState: some View {
        VStack(spacing: 10) {
            ProgressView()
            Text("加载好友主页中...")
                .font(.system(size: 14))
                .foregroundStyle(palette.textMuted)
        }
        .frame(maxWidth: .infinity, maxHeight: .infinity)
    }

    private var failureState: some View {
        let feedback = model.loader.failure?.feedback(resource: "好友主页")
            ?? LoadFailure.Feedback(title: "好友主页加载失败", description: "请稍后重试", actionText: "重试", retryable: true)

        return StateMessageView(
            icon: "exclamationmark.circle",
            title: feedback.title,
            detail: feedback.description,
            actionTitle: feedback.actionText
        ) {
            if feedback.retryable {
                Task { await model.retry() }
            } else {
                dismiss()
            }
        }
    }

    private func refreshBanner(_ message: String) -> some View {
        HStack(spacing: 8) {
            Text(message)
                .font(.system(size: 12))
                .foregroundStyle(palette.textSecondary)
                .lineLimit(2)
            Spacer(minLength: 0)
            Button("重试") {
                Task { await model.refresh() }
            }
            .font(.system(size: 12, weight: .semibold))
            .foregroundStyle(palette.brandStrong)
        }
        .padding(.horizontal, 11)
        .padding(.vertical, 9)
        .background(palette.brandTint, in: RoundedRectangle(cornerRadius: 9))
    }
}
