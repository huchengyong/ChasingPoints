import SwiftUI

/// 排行榜：球种 Tab、领奖台、第 4 名起的列表与底部「我的排名」。
struct RankingView: View {
    @EnvironmentObject private var session: SessionStore
    @EnvironmentObject private var navigator: AppNavigator
    @EnvironmentObject private var toast: ToastCenter
    @Environment(\.colorScheme) private var colorScheme
    @Environment(\.dismiss) private var dismiss
    @StateObject private var model: RankingModel

    init(session: SessionStore, model: RankingModel? = nil) {
        _model = StateObject(wrappedValue: model ?? RankingModel(session: session))
    }

    private var palette: AppPalette { AppTheme.palette(for: colorScheme) }

    var body: some View {
        Group {
            if model.phase == .loading || model.phase == .idle {
                loadingState
            } else if case .failed(let failure) = model.phase {
                failureState(failure)
            } else {
                VStack(spacing: 0) {
                    gameTypeTabs
                    if model.phase == .empty {
                        emptyState
                    } else {
                        leaderboard
                    }
                }
            }
        }
        .background(palette.page.ignoresSafeArea())
        .navigationTitle("排行榜")
        .navigationBarTitleDisplayMode(.inline)
        .task { await model.loadIfNeeded() }
        .onChange(of: session.identity) { _, _ in
            // 登录、退出或切号后清空并重读，登录后才会带回 my_ranking。
            model.syncIdentity()
            Task { await model.loadIfNeeded() }
        }
        .safeAreaInset(edge: .bottom, spacing: 0) {
            if let myRanking = model.myRanking {
                myRankingCard(myRanking)
            }
        }
    }

    private var loadingState: some View {
        VStack(spacing: 12) {
            ProgressView()
            Text("加载中...")
                .font(.system(size: 14))
                .foregroundStyle(palette.textMuted)
        }
        .frame(maxWidth: .infinity, maxHeight: .infinity)
    }

    private var gameTypeTabs: some View {
        ScrollView(.horizontal, showsIndicators: false) {
            HStack(spacing: 8) {
                ForEach(GameTypeCatalog.ordered) { option in
                    gameTypeTab(option)
                }
            }
            .padding(.horizontal, 12)
            .padding(.top, 12)
            .padding(.bottom, 4)
        }
    }

    private func gameTypeTab(_ option: GameTypeOption) -> some View {
        let isActive = model.currentGameType == option.value
        return Button {
            Task { await model.selectGameType(option.value) }
        } label: {
            Text(option.label)
                .font(.system(size: 13, weight: .semibold))
                .foregroundStyle(isActive ? palette.brandStrong : palette.textSecondary)
                .padding(.horizontal, 14)
                .frame(height: 36)
                .background(isActive ? palette.brandTint : palette.card, in: Capsule())
                .overlay {
                    Capsule().stroke(isActive ? palette.brandStrong.opacity(0.22) : palette.border)
                }
        }
        .buttonStyle(.plain)
        .accessibilityAddTraits(isActive ? .isSelected : [])
    }

    @ViewBuilder
    private var leaderboard: some View {
        ScrollView {
            LazyVStack(spacing: 4) {
                if let refreshFailure = model.refreshFailure {
                    refreshBanner(refreshFailure)
                }

                podium

                ForEach(model.list) { item in
                    rankingRow(item)
                }
            }
            .padding(.horizontal, 8)
            .padding(.bottom, 20)
        }
        .refreshable { await model.refresh() }
    }

    private var podium: some View {
        let slots = PodiumSlot.build(topThree: model.topThree)

        return HStack(alignment: .bottom, spacing: 0) {
            ForEach(slots) { slot in
                podiumColumn(slot)
                    .frame(maxWidth: .infinity)
            }
        }
        .frame(maxWidth: 340)
        .frame(maxWidth: .infinity)
        .padding(.top, 24)
        .padding(.bottom, 12)
    }

    private func podiumColumn(_ slot: PodiumSlot) -> some View {
        let isFirst = slot.placement == "first"

        return VStack(spacing: 2) {
            if let user = slot.user {
                if isFirst {
                    Image(systemName: "medal.fill")
                        .font(.system(size: 22))
                        .foregroundStyle(RankingPalette.medalGold)
                        .frame(height: 18)
                        .offset(y: 8)
                        .zIndex(1)
                }

                MemberAvatarView(
                    avatar: user.avatar,
                    userId: user.userId,
                    vipLevel: user.vipLevel ?? 0,
                    rankLevel: user.rankLevel,
                    size: isFirst ? 112 : 80,
                    fitFrame: true
                )
                .overlay(alignment: .bottom) {
                    Text(String(slot.rank))
                        .font(.system(size: isFirst ? 16 : 14, weight: .bold))
                        .foregroundStyle(.white)
                        .frame(width: isFirst ? 32 : 24, height: isFirst ? 32 : 24)
                        .background(Circle().fill(rankBadgeColor(slot.placement)))
                        .offset(y: isFirst ? 11 : 6)
                }
                .padding(.bottom, 8)
                .zIndex(2)

                Text(user.nickname.isEmpty ? "球手" : user.nickname)
                    .font(.system(size: isFirst ? 18 : 16, weight: isFirst ? .bold : .medium))
                    .foregroundStyle(palette.textPrimary)
                    .lineLimit(1)

                Text(verbatim: "\(String(user.rankScore))分")
                    .font(.system(size: isFirst ? 16 : 14))
                    .foregroundStyle(palette.textSecondary)
            }

            podiumBase(slot, isFirst: isFirst)
        }
    }

    private func podiumBase(_ slot: PodiumSlot, isFirst: Bool) -> some View {
        VStack(spacing: 4) {
            if let user = slot.user {
                Text(user.rankName)
                    .font(.system(size: isFirst ? 20 : 16, weight: isFirst ? .heavy : .semibold))
                    .foregroundStyle(.white.opacity(0.9))
                    .lineLimit(1)
                Text(verbatim: "胜率 \(String(user.winRate))%")
                    .font(.system(size: isFirst ? 12 : 11))
                    .foregroundStyle(.white.opacity(0.9))
                    .lineLimit(1)
            }
        }
        .frame(maxWidth: .infinity)
        .frame(height: isFirst ? 100 : (slot.placement == "third" ? 80 : 70))
        .background(
            rankBaseColor(slot.placement),
            in: UnevenRoundedRectangle(topLeadingRadius: 8, topTrailingRadius: 8)
        )
        .padding(.horizontal, slot.placement == "second" || slot.placement == "third" ? 4 : 0)
    }

    private func rankingRow(_ item: LeaderboardItem) -> some View {
        Button {
            toast.show("对方隐藏战绩，无法查看")
        } label: {
            HStack(spacing: 8) {
                Text(String(item.rank))
                    .font(.system(size: 16, weight: .medium))
                    .foregroundStyle(palette.textSecondary)
                    .frame(width: 24)

                HStack(spacing: 6) {
                    MemberAvatarView(
                        avatar: item.avatar,
                        userId: item.userId,
                        vipLevel: item.vipLevel ?? 0,
                        rankLevel: item.rankLevel,
                        size: 48,
                        reserveFrame: true
                    )
                    VStack(alignment: .leading, spacing: 2) {
                        Text(item.nickname.isEmpty ? "球手" : item.nickname)
                            .font(.system(size: 16, weight: .medium))
                            .foregroundStyle(palette.textPrimary)
                            .lineLimit(1)
                        Text(verbatim: "\(item.rankName) | \(String(item.rankScore))分")
                            .font(.system(size: 14))
                            .foregroundStyle(palette.textSecondary)
                            .lineLimit(1)
                    }
                    Spacer(minLength: 4)
                }
                .frame(maxWidth: .infinity, alignment: .leading)

                Text(verbatim: "胜率: \(String(item.winRate))%")
                    .font(.system(size: 14))
                    .foregroundStyle(palette.textSecondary)
                    .lineLimit(1)
            }
            .padding(.horizontal, 16)
            .padding(.vertical, 8)
            .background(palette.card, in: RoundedRectangle(cornerRadius: 8))
            .contentShape(RoundedRectangle(cornerRadius: 8))
        }
        .buttonStyle(.plain)
        .task {
            guard item.id == model.list.last?.id else { return }
            await model.loadMore()
        }
    }

    private var emptyState: some View {
        StateMessageView(
            icon: "medal",
            title: "暂无排行数据",
            detail: "快去对战提升排名吧！",
            actionTitle: "发起PK"
        ) {
            navigator.requestScan(isSignedIn: session.identity != nil)
        }
    }

    private func myRankingCard(_ item: LeaderboardItem) -> some View {
        HStack(spacing: 12) {
            Text(item.rank > 0 ? String(item.rank) : "-")
                .font(.system(size: 18, weight: .bold))
                .foregroundStyle(palette.textPrimary)
                .frame(width: 24)

            HStack(spacing: 8) {
                MemberAvatarView(
                    avatar: item.avatar,
                    userId: item.userId,
                    vipLevel: item.vipLevel ?? 0,
                    rankLevel: item.rankLevel,
                    size: 40,
                    reserveFrame: true
                )
                VStack(alignment: .leading, spacing: 2) {
                    Text(item.rank > 0 ? "我的排名" : "暂未上榜")
                        .font(.system(size: 16, weight: .medium))
                        .foregroundStyle(palette.textPrimary)
                    Text(item.rank > 0 ? item.rankName : "完成首场该模式对局后入榜")
                        .font(.system(size: 14))
                        .foregroundStyle(palette.textSecondary)
                        .lineLimit(1)
                }
                Spacer(minLength: 4)
            }
            .frame(maxWidth: .infinity, alignment: .leading)

            Text(item.rank > 0 ? "\(String(item.rankScore))分" : "未记录榜单分数")
                .font(.system(size: 16, weight: .medium))
                .foregroundStyle(palette.textPrimary)
                .lineLimit(1)
        }
        .padding(.horizontal, 24)
        .padding(.vertical, 16)
        .background(palette.card)
        .overlay(alignment: .top) {
            Rectangle().fill(palette.border).frame(height: 1)
        }
    }

    private func refreshBanner(_ failure: LoadFailure) -> some View {
        let feedback = failure.feedback(resource: "排行榜")
        return HStack(spacing: 10) {
            Text(feedback.description)
                .font(.system(size: 12))
                .foregroundStyle(palette.textSecondary)
            Spacer(minLength: 8)
            Button("重试") {
                Task { await model.refresh() }
            }
            .font(.system(size: 12, weight: .semibold))
            .foregroundStyle(palette.brandStrong)
        }
        .padding(.horizontal, 11)
        .padding(.vertical, 9)
        .background(palette.brandTint, in: RoundedRectangle(cornerRadius: 7))
    }

    private func failureState(_ failure: LoadFailure) -> some View {
        let feedback = failure.feedback(resource: "排行榜")
        return StateMessageView(
            icon: "info.circle",
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

    private func rankBadgeColor(_ placement: String) -> Color {
        switch placement {
        case "first": RankingPalette.medalGold
        case "third": RankingPalette.bronze
        default: RankingPalette.silver
        }
    }

    private func rankBaseColor(_ placement: String) -> Color {
        colorScheme == .dark
            ? RankingPalette.darkBase(placement)
            : RankingPalette.lightBase(placement)
    }
}

/// 领奖台名次色与台座色，取值照搬 `pages/ranking/index.scss`。
private enum RankingPalette {
    static let medalGold = Color(hex: 0xFBBF24)
    static let silver = Color(hex: 0x9A8C67)
    static let bronze = Color(hex: 0xA16207)

    static func lightBase(_ placement: String) -> Color {
        switch placement {
        case "first": Color(hex: 0xFCD34D)
        case "third": Color(hex: 0xA16207)
        default: Color(hex: 0xE9E2CF)
        }
    }

    static func darkBase(_ placement: String) -> Color {
        switch placement {
        case "first": Color(hex: 0xD97706)
        case "third": Color(hex: 0x78350F)
        default: Color(hex: 0x3A2E16)
        }
    }
}
