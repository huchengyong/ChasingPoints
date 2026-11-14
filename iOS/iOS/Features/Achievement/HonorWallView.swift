import SwiftUI

/// 荣誉墙，对应小程序 `subPages/achievement/index.vue`。
struct HonorWallView: View {
    @EnvironmentObject private var session: SessionStore
    @EnvironmentObject private var navigator: AppNavigator
    @EnvironmentObject private var toast: ToastCenter
    @EnvironmentObject private var activity: ActivityStore
    @Environment(\.colorScheme) private var colorScheme
    @Environment(\.dismiss) private var dismiss
    @StateObject private var model: HonorWallModel
    @StateObject private var rolloverPresenter: SeasonRolloverPresenter
    @State private var isTitleSelectorPresented = false

    init(session: SessionStore, gameType: Int, model: HonorWallModel? = nil) {
        _model = StateObject(wrappedValue: model ?? HonorWallModel(session: session, gameType: gameType))
        _rolloverPresenter = StateObject(wrappedValue: SeasonRolloverPresenter(session: session))
    }

    private var palette: AppPalette { AppTheme.palette(for: colorScheme) }

    var body: some View {
        Group {
            if session.identity == nil {
                StateMessageView(
                    icon: "lock.fill",
                    title: "登录后查看荣誉墙",
                    actionTitle: "去登录"
                ) {
                    navigator.openLogin()
                }
            } else if model.isInitialLoading {
                loadingState
            } else if model.isInitialFailure {
                failureState
            } else if model.hasLoaded {
                content
            } else {
                loadingState
            }
        }
        .background(palette.page.ignoresSafeArea())
        .navigationTitle("荣誉墙")
        .navigationBarTitleDisplayMode(.inline)
        .task {
            await model.loadIfNeeded()
            // 换季提醒只在本人荣誉墙弹出；关闭弹窗后才会标已读。
            guard model.hasLoaded, model.isSelf else { return }
            await rolloverPresenter.present {
                activity.decrementUnread()
                activity.clearLatestSeasonRollover()
            }
        }
        .onChange(of: session.identity) { _, _ in
            Task { await model.loadIfNeeded() }
        }
        .onChange(of: model.notice) { _, notice in
            guard let notice else { return }
            toast.show(notice)
            model.clearNotice()
        }
        .seasonRolloverAlert(rolloverPresenter)
        .onDisappear { rolloverPresenter.respond(confirmed: false) }
        .sheet(isPresented: $isTitleSelectorPresented) {
            titleSelector
        }
    }

    // MARK: - 状态

    private var loadingState: some View {
        VStack(spacing: 10) {
            ProgressView()
            Text("正在整理荣誉墙")
                .font(.system(size: 15, weight: .semibold))
                .foregroundStyle(palette.textPrimary)
            Text("生涯成就、赛季挑战与历届荣誉即将呈现")
                .font(.system(size: 13))
                .foregroundStyle(palette.textSecondary)
                .multilineTextAlignment(.center)
        }
        .frame(maxWidth: .infinity, maxHeight: .infinity)
        .padding(24)
    }

    @ViewBuilder
    private var failureState: some View {
        let error = model.loadError
        StateMessageView(
            icon: "exclamationmark.triangle",
            title: error?.title ?? "荣誉墙暂时没加载出来",
            detail: error?.description ?? "请稍后重试，已有荣誉不会受到影响",
            actionTitle: error?.showRetry == true ? "重新加载" : "返回"
        ) {
            if error?.showRetry == true {
                Task { await model.retry() }
            } else {
                dismiss()
            }
        }
    }

    // MARK: - 内容

    private var content: some View {
        ScrollView {
            VStack(spacing: 12) {
                if model.isRefreshing {
                    refreshIndicator
                }
                heroCard
                tabs
                switch model.activeTab {
                case .career:
                    careerPanel
                case .season:
                    seasonPanel
                case .history:
                    historyPanel
                }
            }
            .padding(.horizontal, 12)
            .padding(.top, 10)
            .padding(.bottom, 28)
        }
        .refreshable { await model.refresh() }
    }

    private var refreshIndicator: some View {
        HStack(spacing: 6) {
            ProgressView()
                .controlSize(.small)
            Text("正在更新")
                .font(.system(size: 12))
                .foregroundStyle(palette.brandStrong)
        }
        .frame(maxWidth: .infinity)
    }

    private var heroCard: some View {
        VStack(alignment: .leading, spacing: 14) {
            HStack(spacing: 12) {
                AvatarView(
                    avatar: model.wall?.profile.avatar ?? "",
                    userId: model.wall?.profile.userId ?? 0,
                    size: 54
                )
                VStack(alignment: .leading, spacing: 3) {
                    Text(model.profileKicker)
                        .font(.system(size: 11, weight: .semibold))
                        .tracking(1)
                        .foregroundStyle(palette.brandStrong)
                    Text(model.profileName)
                        .font(.system(size: 18, weight: .bold))
                        .foregroundStyle(palette.textPrimary)
                        .lineLimit(1)
                    Text(model.profileNote)
                        .font(.system(size: 12))
                        .foregroundStyle(palette.textSecondary)
                        .fixedSize(horizontal: false, vertical: true)
                }
                Spacer(minLength: 0)
            }

            Button {
                guard model.isSelf else { return }
                isTitleSelectorPresented = true
                Task { await model.prepareTitleSelector() }
            } label: {
                HStack(spacing: 10) {
                    Image(systemName: "medal.fill")
                        .font(.system(size: 20))
                        .foregroundStyle(palette.brandStrong)
                        .frame(width: 38, height: 38)
                        .background(palette.brandTint, in: RoundedRectangle(cornerRadius: 10))
                    VStack(alignment: .leading, spacing: 2) {
                        Text("当前称号")
                            .font(.system(size: 11))
                            .foregroundStyle(palette.textMuted)
                        Text(model.equippedTitleName)
                            .font(.system(size: 14, weight: .semibold))
                            .foregroundStyle(palette.textPrimary)
                            .lineLimit(1)
                        Text(model.equippedTitleSource)
                            .font(.system(size: 11))
                            .foregroundStyle(palette.textSecondary)
                            .lineLimit(1)
                    }
                    Spacer(minLength: 4)
                    if model.isSelf {
                        Text(model.activeTitleActionText)
                            .font(.system(size: 12, weight: .semibold))
                            .foregroundStyle(palette.brandStrong)
                    }
                }
                .padding(10)
                .background(palette.subtle, in: RoundedRectangle(cornerRadius: 12))
            }
            .buttonStyle(.plain)
            .disabled(!model.isSelf)

            HStack(spacing: 0) {
                ForEach(model.summaryItems) { item in
                    VStack(spacing: 2) {
                        Text(item.value)
                            .font(.system(size: 16, weight: .bold).monospacedDigit())
                            .foregroundStyle(palette.textPrimary)
                            .lineLimit(1)
                            .minimumScaleFactor(0.7)
                        Text(item.label)
                            .font(.system(size: 10))
                            .foregroundStyle(palette.textMuted)
                            .lineLimit(1)
                            .minimumScaleFactor(0.8)
                    }
                    .frame(maxWidth: .infinity)
                    .accessibilityElement(children: .combine)
                }
            }

            recentSection
        }
        .appCard(padding: 14)
    }

    private var recentSection: some View {
        VStack(alignment: .leading, spacing: 8) {
            HStack(alignment: .firstTextBaseline) {
                Text("最近荣誉")
                    .font(.system(size: 14, weight: .bold))
                    .foregroundStyle(palette.textPrimary)
                Spacer(minLength: 8)
                Text("自动取最近 3 项")
                    .font(.system(size: 11))
                    .foregroundStyle(palette.textMuted)
            }
            if model.recentHonors.isEmpty {
                Text("完成第一项成就后，最近荣誉会出现在这里")
                    .font(.system(size: 12))
                    .foregroundStyle(palette.textSecondary)
                    .frame(maxWidth: .infinity, alignment: .leading)
                    .padding(.vertical, 8)
            } else {
                VStack(spacing: 10) {
                    ForEach(model.recentHonors) { item in
                        HStack(alignment: .top, spacing: 10) {
                            Text(AchievementPageRules.honorEmoji(item.sourceType))
                                .font(.system(size: 20))
                                .frame(width: 36, height: 36)
                                .background(palette.subtle, in: RoundedRectangle(cornerRadius: 9))
                            VStack(alignment: .leading, spacing: 3) {
                                HStack(alignment: .firstTextBaseline, spacing: 8) {
                                    Text(item.name)
                                        .font(.system(size: 13, weight: .semibold))
                                        .foregroundStyle(palette.textPrimary)
                                        .lineLimit(1)
                                    Spacer(minLength: 4)
                                    Text(item.sourceLabel)
                                        .font(.system(size: 10))
                                        .foregroundStyle(palette.brandStrong)
                                }
                                Text(item.detailText)
                                    .font(.system(size: 12))
                                    .foregroundStyle(palette.textSecondary)
                                    .fixedSize(horizontal: false, vertical: true)
                                if !item.earnedAt.isEmpty {
                                    Text(item.earnedAt)
                                        .font(.system(size: 10))
                                        .foregroundStyle(palette.textMuted)
                                }
                            }
                        }
                    }
                }
            }
        }
    }

    private var tabs: some View {
        HStack(spacing: 6) {
            ForEach(model.tabs) { tab in
                let isActive = model.activeTab == tab
                Button {
                    model.switchTab(tab)
                } label: {
                    Text(tab.label)
                        .font(.system(size: 13, weight: isActive ? .semibold : .regular))
                        .foregroundStyle(isActive ? .white : palette.textSecondary)
                        .frame(maxWidth: .infinity)
                        .frame(height: 34)
                        .background(isActive ? AppTheme.primary : palette.card, in: Capsule())
                        .overlay {
                            if !isActive {
                                Capsule().stroke(palette.border)
                            }
                        }
                }
                .buttonStyle(.plain)
                .accessibilityAddTraits(isActive ? .isSelected : [])
            }
        }
    }

    // MARK: - 生涯成就

    private var careerPanel: some View {
        VStack(spacing: 12) {
            sectionHeading("生涯成就", description: "永久累计，不随赛季清零；球种绝技独立展示")

            if model.isSelf && !model.upcomingSection.hidden {
                upcomingCard
            }

            sectionHeading(
                "通用里程碑",
                description: "对局、胜场、连胜与赛事历程跨球种累计",
                count: "\(String(model.wall?.summary.universalUnlocked ?? 0))/\(String(model.wall?.summary.universalTotal ?? 0))"
            )
            if model.universalGroups.isEmpty {
                emptyCard(title: "暂无通用里程碑", detail: "完成有效比赛后，通用进度会自动更新")
            } else {
                ForEach(model.universalGroups) { group in
                    groupCard(group)
                }
            }

            sectionHeading(
                "球种绝技",
                description: "只计算当前球种，不要求跨球种集齐",
                count: "\(String(model.wall?.summary.specialtyUnlocked ?? 0))/\(String(model.wall?.summary.specialtyTotal ?? 0))"
            )
            gameTypeChips
            specialtyCard
        }
    }

    private var upcomingCard: some View {
        VStack(alignment: .leading, spacing: 10) {
            compactHeading("即将达成", hint: "通用 + \(model.currentGameTypeLabel)")
            if model.upcomingSection.items.isEmpty {
                Text(model.upcomingEmptyText)
                    .font(.system(size: 12))
                    .foregroundStyle(palette.textSecondary)
                    .frame(maxWidth: .infinity, alignment: .leading)
                    .padding(.vertical, 8)
            } else {
                VStack(spacing: 10) {
                    ForEach(model.upcomingSection.items) { item in
                        upcomingRow(item)
                    }
                }
            }
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .appCard(padding: 14)
    }

    private func groupCard(_ group: AchievementGroup) -> some View {
        VStack(alignment: .leading, spacing: 10) {
            HStack(spacing: 6) {
                Text(AchievementPageRules.categoryEmoji(group.key))
                    .font(.system(size: 16))
                Text(group.label)
                    .font(.system(size: 14, weight: .bold))
                    .foregroundStyle(palette.textPrimary)
                Spacer(minLength: 8)
                Text("\(String(group.items.count)) 项")
                    .font(.system(size: 11))
                    .foregroundStyle(palette.textMuted)
            }
            VStack(spacing: 10) {
                ForEach(group.items) { item in
                    achievementRow(item, showUnlockedTag: true, interactive: model.isSelf, showsChevron: model.isSelf)
                }
            }
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .appCard(padding: 14)
    }

    private var specialtyCard: some View {
        Group {
            if model.specialtyAchievements.isEmpty {
                emptyCard(
                    title: "暂无\(model.currentGameTypeLabel)绝技",
                    detail: model.isSelf ? "完成该球种的特殊战绩后会在这里点亮" : "TA 尚未解锁该球种绝技"
                )
            } else {
                VStack(alignment: .leading, spacing: 10) {
                    HStack(spacing: 6) {
                        Text(AchievementPageRules.fallbackEmoji(model.specialtyAchievements[0]))
                            .font(.system(size: 16))
                        Text("\(model.currentGameTypeLabel)绝技")
                            .font(.system(size: 14, weight: .bold))
                            .foregroundStyle(palette.textPrimary)
                        Spacer(minLength: 8)
                        Text("\(String(model.specialtyAchievements.count)) 项")
                            .font(.system(size: 11))
                            .foregroundStyle(palette.textMuted)
                    }
                    VStack(spacing: 10) {
                        ForEach(model.specialtyAchievements) { item in
                            achievementRow(item, showUnlockedTag: true, interactive: model.isSelf, showsChevron: model.isSelf)
                        }
                    }
                }
                .frame(maxWidth: .infinity, alignment: .leading)
                .appCard(padding: 14)
            }
        }
    }

    private func upcomingRow(_ item: HonorWallUpcomingItem) -> some View {
        let achievement = item.achievement
        let row = HStack(alignment: .top, spacing: 10) {
            achievementIcon(achievement)
            VStack(alignment: .leading, spacing: 4) {
                HStack(alignment: .firstTextBaseline, spacing: 6) {
                    Text(achievement.name)
                        .font(.system(size: 13, weight: .semibold))
                        .foregroundStyle(palette.textPrimary)
                        .lineLimit(1)
                    Spacer(minLength: 4)
                    Text("\(String(item.progressPercent))%")
                        .font(.system(size: 11, weight: .semibold))
                        .foregroundStyle(palette.brandStrong)
                }
                progressTrack(percent: item.progressPercent)
                Text("\(item.progressText) · \(item.remainingText)")
                    .font(.system(size: 10))
                    .foregroundStyle(palette.textMuted)
            }
            Image(systemName: "chevron.right")
                .font(.system(size: 12, weight: .semibold))
                .foregroundStyle(palette.textMuted)
        }
        .contentShape(Rectangle())

        return Group {
            if model.isSelf && achievement.id > 0 {
                NavigationLink(value: AppRoute.achievementDetail(achievementId: achievement.id)) {
                    row
                }
                .buttonStyle(.plain)
            } else {
                row
            }
        }
    }

    private var gameTypeChips: some View {
        HStack(spacing: 6) {
            ForEach(GameTypeCatalog.ordered) { option in
                let isActive = model.currentGameType == option.value
                Button {
                    Task { await model.selectGameType(option.value) }
                } label: {
                    Text(option.label)
                        .font(.system(size: 12, weight: isActive ? .semibold : .regular))
                        .foregroundStyle(isActive ? .white : palette.textSecondary)
                        .lineLimit(1)
                        .minimumScaleFactor(0.8)
                        .frame(maxWidth: .infinity)
                        .frame(height: 32)
                        .background(isActive ? AppTheme.primary : palette.card, in: Capsule())
                        .overlay {
                            if !isActive {
                                Capsule().stroke(palette.border)
                            }
                        }
                }
                .buttonStyle(.plain)
                .accessibilityAddTraits(isActive ? .isSelected : [])
            }
        }
    }

    private func achievementRow(
        _ item: AchievementDef,
        showUnlockedTag: Bool,
        interactive: Bool,
        showsChevron: Bool
    ) -> some View {
        let content = achievementRowContent(item, showUnlockedTag: showUnlockedTag, showsChevron: showsChevron)
        return Group {
            if interactive && item.id > 0 {
                NavigationLink(value: AppRoute.achievementDetail(achievementId: item.id)) {
                    content
                }
                .buttonStyle(.plain)
            } else {
                content
            }
        }
    }

    private func achievementRowContent(
        _ item: AchievementDef,
        showUnlockedTag: Bool,
        showsChevron: Bool
    ) -> some View {
        HStack(alignment: .top, spacing: 10) {
            achievementIcon(item)
            VStack(alignment: .leading, spacing: 4) {
                HStack(alignment: .firstTextBaseline, spacing: 6) {
                    Text(item.name)
                        .font(.system(size: 13, weight: .semibold))
                        .foregroundStyle(palette.textPrimary)
                        .lineLimit(1)
                    if showUnlockedTag && item.unlocked {
                        Text("已解锁")
                            .font(.system(size: 10, weight: .semibold))
                            .foregroundStyle(.white)
                            .padding(.horizontal, 6)
                            .padding(.vertical, 1)
                            .background(palette.success, in: Capsule())
                    }
                    Spacer(minLength: 4)
                }
                Text(item.description.isEmpty ? "完成目标即可永久点亮" : item.description)
                    .font(.system(size: 11))
                    .foregroundStyle(palette.textSecondary)
                    .lineLimit(2)
                progressTrack(
                    percent: AchievementPageRules.detailProgressPercent(
                        progress: item.progress,
                        threshold: item.threshold,
                        unlocked: item.unlocked
                    )
                )
                Text(careerProgressText(item))
                    .font(.system(size: 10))
                    .foregroundStyle(palette.textMuted)
            }
            if showsChevron {
                Image(systemName: "chevron.right")
                    .font(.system(size: 12, weight: .semibold))
                    .foregroundStyle(palette.textMuted)
            }
        }
        .contentShape(Rectangle())
    }

    private func careerProgressText(_ item: AchievementDef) -> String {
        if item.unlocked {
            guard let unlockedAt = item.unlockedAt, !unlockedAt.isEmpty else { return "已解锁" }
            return "已解锁 · \(unlockedAt)"
        }
        return "\(String(max(0, item.progress)))/\(String(max(0, item.threshold)))"
    }

    // MARK: - 当前赛季

    private var seasonPanel: some View {
        VStack(spacing: 12) {
            sectionHeading("当前赛季", description: "每个赛季、每种球类独立累计")
            gameTypeChips

            if model.currentSeasonState.mode != .active {
                intermissionCard
            } else {
                seasonBanner
                challengeCard
            }
        }
    }

    private var intermissionCard: some View {
        VStack(spacing: 8) {
            Image(systemName: "calendar")
                .font(.system(size: 28))
                .foregroundStyle(palette.brandStrong)
            Text(model.currentSeasonState.title)
                .font(.system(size: 15, weight: .semibold))
                .foregroundStyle(palette.textPrimary)
            Text(model.currentSeasonState.description)
                .font(.system(size: 12))
                .foregroundStyle(palette.textSecondary)
                .multilineTextAlignment(.center)
                .fixedSize(horizontal: false, vertical: true)
        }
        .frame(maxWidth: .infinity)
        .appCard(padding: 18)
    }

    private var seasonBanner: some View {
        HStack(alignment: .center, spacing: 10) {
            VStack(alignment: .leading, spacing: 3) {
                Text("进行中的赛季")
                    .font(.system(size: 11, weight: .semibold))
                    .tracking(1)
                    .foregroundStyle(palette.brandStrong)
                Text("\(model.currentSeasonName) · \(model.currentGameTypeLabel)")
                    .font(.system(size: 16, weight: .bold))
                    .foregroundStyle(palette.textPrimary)
                Text(model.seasonDateText)
                    .font(.system(size: 11))
                    .foregroundStyle(palette.textSecondary)
            }
            Spacer(minLength: 8)
            Text("独立累计")
                .font(.system(size: 10, weight: .semibold))
                .foregroundStyle(.white)
                .padding(.horizontal, 9)
                .padding(.vertical, 3)
                .background(AppTheme.primary, in: Capsule())
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .appCard(padding: 14)
    }

    private var challengeCard: some View {
        VStack(alignment: .leading, spacing: 10) {
            compactHeading("本赛季挑战", hint: "固定 3 项")
            VStack(spacing: 10) {
                ForEach(model.currentChallenges) { challenge in
                    challengeRow(challenge)
                }
            }
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .appCard(padding: 14)
    }

    private func challengeRow(_ item: HonorWallChallengeItem) -> some View {
        HStack(alignment: .top, spacing: 10) {
            challengeIcon(item)
            VStack(alignment: .leading, spacing: 4) {
                HStack(alignment: .firstTextBaseline, spacing: 6) {
                    Text(item.name)
                        .font(.system(size: 13, weight: .semibold))
                        .foregroundStyle(palette.textPrimary)
                        .lineLimit(1)
                    Spacer(minLength: 4)
                    Text(item.progressText)
                        .font(.system(size: 11, weight: .semibold))
                        .foregroundStyle(item.completed ? palette.success : palette.textSecondary)
                }
                Text(item.description)
                    .font(.system(size: 11))
                    .foregroundStyle(palette.textSecondary)
                    .lineLimit(2)
                progressTrack(percent: item.progressPercent)
                Text(item.remainingText)
                    .font(.system(size: 10))
                    .foregroundStyle(palette.textMuted)
            }
        }
    }

    // MARK: - 历届荣誉

    private var historyPanel: some View {
        VStack(spacing: 12) {
            sectionHeading(
                "历届荣誉",
                description: "赛季排名与赛事名次永久保留",
                count: "\(model.historyTotalText) 项"
            )

            if model.isSelf && !model.historySeasonOptions.isEmpty {
                archiveCard
            }

            if model.historyHonors.isEmpty {
                emptyCard(title: "暂无历届荣誉", detail: "赛季排名或赛事名次产生后，会永久陈列在这里")
            } else {
                historyCard
            }
        }
    }

    private var archiveCard: some View {
        VStack(alignment: .leading, spacing: 10) {
            compactHeading("往届挑战记录", hint: "只读快照")
            HStack(spacing: 6) {
                ForEach(model.historySeasonOptions) { season in
                    let isActive = model.selectedHistorySeasonId == season.id
                    Button {
                        Task { await model.selectHistorySeason(season.id) }
                    } label: {
                        Text(season.label)
                            .font(.system(size: 12, weight: isActive ? .semibold : .regular))
                            .foregroundStyle(isActive ? .white : palette.textSecondary)
                            .lineLimit(1)
                            .padding(.horizontal, 10)
                            .frame(height: 30)
                            .background(isActive ? AppTheme.primary : palette.subtle, in: Capsule())
                    }
                    .buttonStyle(.plain)
                }
                Spacer(minLength: 0)
            }

            if let season = model.wall?.history.challengeSeason {
                VStack(alignment: .leading, spacing: 8) {
                    Text("\(season.seasonName) · \(model.currentGameTypeLabel)")
                        .font(.system(size: 13, weight: .semibold))
                        .foregroundStyle(palette.textPrimary)
                    if model.archivedChallenges.isEmpty {
                        Text("该赛季没有可回看的挑战记录")
                            .font(.system(size: 11))
                            .foregroundStyle(palette.textSecondary)
                    } else {
                        ForEach(model.archivedChallenges) { item in
                            HStack(alignment: .top, spacing: 10) {
                                challengeIcon(item, size: 30)
                                VStack(alignment: .leading, spacing: 2) {
                                    Text(item.name)
                                        .font(.system(size: 12, weight: .semibold))
                                        .foregroundStyle(palette.textPrimary)
                                    Text(item.description)
                                        .font(.system(size: 11))
                                        .foregroundStyle(palette.textSecondary)
                                        .lineLimit(2)
                                }
                                Spacer(minLength: 4)
                                Text("\(String(item.progress))/\(String(item.threshold))")
                                    .font(.system(size: 11, weight: .semibold))
                                    .foregroundStyle(item.completed ? palette.success : palette.textSecondary)
                            }
                        }
                    }
                }
                .padding(.top, 2)
            }
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .appCard(padding: 14)
    }

    private var historyCard: some View {
        VStack(spacing: 0) {
            VStack(spacing: 10) {
                ForEach(model.historyHonors) { item in
                    HStack(alignment: .top, spacing: 10) {
                        Text(AchievementPageRules.honorEmoji(item.sourceType))
                            .font(.system(size: 20))
                            .frame(width: 36, height: 36)
                            .background(palette.subtle, in: RoundedRectangle(cornerRadius: 9))
                        VStack(alignment: .leading, spacing: 3) {
                            Text(item.name)
                                .font(.system(size: 13, weight: .semibold))
                                .foregroundStyle(palette.textPrimary)
                                .lineLimit(1)
                            Text(historyDescription(item))
                                .font(.system(size: 11))
                                .foregroundStyle(palette.textSecondary)
                                .lineLimit(2)
                            if !item.earnedAt.isEmpty {
                                Text(item.earnedAt)
                                    .font(.system(size: 10))
                                    .foregroundStyle(palette.textMuted)
                            }
                        }
                        Spacer(minLength: 4)
                        Text(item.sourceType == "season" ? "赛季" : "赛事")
                            .font(.system(size: 10, weight: .semibold))
                            .foregroundStyle(palette.brandStrong)
                    }
                }
            }

            if model.hasMoreHistory {
                Button {
                    Task { await model.loadMoreHistory() }
                } label: {
                    HStack(spacing: 6) {
                        if model.isHistoryLoading {
                            ProgressView()
                                .controlSize(.small)
                        }
                        Text(model.isHistoryLoading ? "加载中" : "加载更多荣誉")
                            .font(.system(size: 13, weight: .semibold))
                    }
                    .foregroundStyle(palette.brandStrong)
                    .frame(maxWidth: .infinity)
                    .padding(.vertical, 12)
                }
                .buttonStyle(.plain)
                .disabled(model.isHistoryLoading)
            }
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .appCard(padding: 14)
    }

    private func historyDescription(_ item: HonorItem) -> String {
        if let description = item.description, !description.isEmpty { return description }
        if let name = item.sourceRefName, !name.isEmpty { return name }
        return "永久荣誉"
    }

    // MARK: - 通用组件

    private func sectionHeading(_ title: String, description: String, count: String? = nil) -> some View {
        HStack(alignment: .firstTextBaseline, spacing: 8) {
            VStack(alignment: .leading, spacing: 2) {
                Text(title)
                    .font(.system(size: 16, weight: .bold))
                    .foregroundStyle(palette.textPrimary)
                Text(description)
                    .font(.system(size: 11))
                    .foregroundStyle(palette.textSecondary)
                    .fixedSize(horizontal: false, vertical: true)
            }
            Spacer(minLength: 4)
            if let count {
                Text(count)
                    .font(.system(size: 12, weight: .semibold))
                    .foregroundStyle(palette.textMuted)
            }
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .padding(.top, 2)
    }

    private func compactHeading(_ title: String, hint: String) -> some View {
        HStack(alignment: .firstTextBaseline) {
            Text(title)
                .font(.system(size: 14, weight: .bold))
                .foregroundStyle(palette.textPrimary)
            Spacer(minLength: 8)
            Text(hint)
                .font(.system(size: 11))
                .foregroundStyle(palette.textMuted)
        }
    }

    private func emptyCard(title: String, detail: String) -> some View {
        VStack(spacing: 5) {
            Text(title)
                .font(.system(size: 14, weight: .semibold))
                .foregroundStyle(palette.textPrimary)
            Text(detail)
                .font(.system(size: 12))
                .foregroundStyle(palette.textSecondary)
                .multilineTextAlignment(.center)
        }
        .frame(maxWidth: .infinity)
        .padding(.vertical, 8)
        .appCard(padding: 16)
    }

    private func achievementIcon(_ item: AchievementDef) -> some View {
        RemoteImage(url: CDNAsset.url(for: item.icon), contentMode: .fit) {
            Text(AchievementPageRules.fallbackEmoji(item))
                .font(.system(size: 20))
        }
        .frame(width: 38, height: 38)
        .background(palette.subtle, in: RoundedRectangle(cornerRadius: 9))
        .opacity(item.unlocked ? 1 : 0.65)
    }

    private func challengeIcon(_ item: HonorWallChallengeItem, size: CGFloat = 36) -> some View {
        RemoteImage(url: CDNAsset.url(for: item.iconPath), contentMode: .fit) {
            Text(AchievementPageRules.challengeEmoji(item.key))
                .font(.system(size: 18))
        }
        .frame(width: size, height: size)
        .background(palette.subtle, in: RoundedRectangle(cornerRadius: 9))
    }

    private func progressTrack(percent: Int) -> some View {
        GeometryReader { proxy in
            ZStack(alignment: .leading) {
                Capsule().fill(palette.border)
                Capsule()
                    .fill(AppTheme.primary)
                    .frame(width: proxy.size.width * CGFloat(min(100, max(0, percent))) / 100)
            }
        }
        .frame(height: 6)
        .accessibilityHidden(true)
    }

    // MARK: - 称号选择

    private var titleSelector: some View {
        NavigationStack {
            ScrollView {
                VStack(alignment: .leading, spacing: 14) {
                    VStack(alignment: .leading, spacing: 3) {
                        Text("选择佩戴称号")
                            .font(.system(size: 18, weight: .bold))
                            .foregroundStyle(palette.textPrimary)
                        Text("选择后立即生效，仅改变对外展示")
                            .font(.system(size: 12))
                            .foregroundStyle(palette.textSecondary)
                    }

                    if model.isTitleListLoading {
                        titleSelectorState(title: "正在加载已获称号", detail: nil, showsSpinner: true)
                    } else if model.isTitleListFailed {
                        titleSelectorState(
                            title: "称号列表暂时没加载出来",
                            detail: "当前佩戴状态不会改变，可以重新加载后再选择",
                            showsSpinner: false
                        )
                        Button("重新加载称号") {
                            Task { await model.loadTitles() }
                        }
                        .buttonStyle(.appSecondary)
                    } else if model.isTitleListLoaded && model.titleList.isEmpty {
                        titleSelectorState(
                            title: "暂无可佩戴称号",
                            detail: "解锁生涯成就或获得赛季、赛事荣誉后，称号会出现在这里",
                            showsSpinner: false
                        )
                        noneOptionText("已获称号为空，当前没有对外展示的称号")
                    } else {
                        VStack(spacing: 8) {
                            ForEach(model.titleOptions) { title in
                                titleOption(title)
                            }
                            noneOptionText("隐藏对外展示，已获称号仍永久保留")
                        }
                    }
                }
                .padding(16)
            }
            .background(palette.page.ignoresSafeArea())
            .navigationTitle("选择佩戴称号")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .topBarTrailing) {
                    Button("关闭") { isTitleSelectorPresented = false }
                        .foregroundStyle(palette.brandStrong)
                }
            }
        }
        .presentationDetents([.medium, .large])
    }

    private func titleSelectorState(title: String, detail: String?, showsSpinner: Bool) -> some View {
        VStack(spacing: 8) {
            if showsSpinner {
                ProgressView()
            }
            Text(title)
                .font(.system(size: 14, weight: .semibold))
                .foregroundStyle(palette.textPrimary)
            if let detail {
                Text(detail)
                    .font(.system(size: 12))
                    .foregroundStyle(palette.textSecondary)
                    .multilineTextAlignment(.center)
            }
        }
        .frame(maxWidth: .infinity)
        .padding(.vertical, 18)
    }

    private func titleOption(_ title: TitleInfo) -> some View {
        Button {
            Task {
                if await model.selectTitle(title) { isTitleSelectorPresented = false }
            }
        } label: {
            HStack(spacing: 10) {
                Image(systemName: "medal.fill")
                    .font(.system(size: 17))
                    .foregroundStyle(palette.brandStrong)
                    .frame(width: 36, height: 36)
                    .background(palette.brandTint, in: RoundedRectangle(cornerRadius: 10))
                VStack(alignment: .leading, spacing: 2) {
                    Text(title.titleName)
                        .font(.system(size: 14, weight: .semibold))
                        .foregroundStyle(palette.textPrimary)
                        .lineLimit(1)
                    Text(AchievementPageRules.titleSourceDescription(title))
                        .font(.system(size: 11))
                        .foregroundStyle(palette.textSecondary)
                        .lineLimit(1)
                }
                Spacer(minLength: 4)
                if title.equipped {
                    Image(systemName: "checkmark.circle.fill")
                        .font(.system(size: 18))
                        .foregroundStyle(palette.brandStrong)
                }
            }
            .padding(10)
            .background(title.equipped ? palette.brandTint : palette.card, in: RoundedRectangle(cornerRadius: 12))
            .overlay {
                RoundedRectangle(cornerRadius: 12).stroke(title.equipped ? AppTheme.primary.opacity(0.5) : palette.border)
            }
        }
        .buttonStyle(.plain)
        .disabled(model.isTitleSubmitting)
    }

    private func noneOptionText(_ detail: String) -> some View {
        Button {
            Task {
                if await model.selectNoTitle() { isTitleSelectorPresented = false }
            }
        } label: {
            HStack(spacing: 10) {
                Text("—")
                    .font(.system(size: 18, weight: .semibold))
                    .foregroundStyle(palette.textMuted)
                    .frame(width: 36, height: 36)
                    .background(palette.subtle, in: RoundedRectangle(cornerRadius: 10))
                VStack(alignment: .leading, spacing: 2) {
                    Text("不佩戴称号")
                        .font(.system(size: 14, weight: .semibold))
                        .foregroundStyle(palette.textPrimary)
                    Text(detail)
                        .font(.system(size: 11))
                        .foregroundStyle(palette.textSecondary)
                        .lineLimit(2)
                }
                Spacer(minLength: 4)
                if model.currentTitleId == 0 {
                    Image(systemName: "checkmark.circle.fill")
                        .font(.system(size: 18))
                        .foregroundStyle(palette.brandStrong)
                }
            }
            .padding(10)
            .background(model.currentTitleId == 0 ? palette.brandTint : palette.card, in: RoundedRectangle(cornerRadius: 12))
            .overlay {
                RoundedRectangle(cornerRadius: 12)
                    .stroke(model.currentTitleId == 0 ? AppTheme.primary.opacity(0.5) : palette.border)
            }
        }
        .buttonStyle(.plain)
        .disabled(model.isTitleSubmitting)
    }
}
