import SwiftUI

/// 首页：开局卡、约球待办、正在进行、赛事、排行榜与附近球房，对应 `pages/index/index.vue`。
///
/// 各区块独立读取与报错：整页错误只在排行榜与赛事同时失败且没有内容时出现。
struct HomeView: View {
    @EnvironmentObject private var session: SessionStore
    @EnvironmentObject private var navigator: AppNavigator
    @EnvironmentObject private var activity: ActivityStore
    @Environment(\.colorScheme) private var colorScheme
    @Environment(\.dismiss) private var dismiss
    @StateObject private var model: HomeModel
    @State private var greetingHour = Calendar.current.component(.hour, from: Date())

    init(session: SessionStore, model: HomeModel? = nil) {
        _model = StateObject(wrappedValue: model ?? HomeModel(session: session))
    }

    private var palette: AppPalette { AppTheme.palette(for: colorScheme) }
    private var isLoggedIn: Bool { session.identity != nil }

    private var ongoingCard: HomeMatchCard? {
        HomeRules.ongoingMatchCard(isLoggedIn ? activity.currentMatch : nil, currentUser: session.user)
    }

    private var challengeCard: HomeChallengeCard? {
        guard let card = HomeRules.challengeQuickCard(
            isLoggedIn ? activity.currentChallenge : nil,
            userId: session.identity?.userId ?? 0,
            serverTime: activity.challengeServerTime
        ) else { return nil }
        guard let ongoingCard else { return card }
        // 约球已开局时由「我的对局」卡承接，不重复展示。
        let sameMatch = card.matchId > 0 && card.matchId == ongoingCard.id
        let sameChallenge = card.challengeId > 0 && card.challengeId == (activity.currentMatch?.challengeId ?? 0)
        return sameMatch || sameChallenge ? nil : card
    }

    private var hasContent: Bool {
        model.hasPublicContent || ongoingCard != nil || challengeCard != nil
    }

    private var greetingText: String {
        guard isLoggedIn else { return "记录每一局真实比分" }
        let name = (session.user?.nickname ?? "").trimmingCharacters(in: .whitespacesAndNewlines)
        return "\(HomeRules.greeting(hour: greetingHour))，\(name.isEmpty ? "球友" : name)"
    }

    private var liveCards: [SpectatorMatchCard] {
        SpectatorLobbyRules.buildHomeLiveMatches(
            model.liveMatches,
            userId: session.identity?.userId ?? 0,
            currentMatchId: ongoingCard?.id ?? 0
        )
    }

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 0) {
                Text(greetingText)
                    .font(.system(size: 13))
                    .foregroundStyle(palette.textSecondary)
                    .padding(.horizontal, 4)
                    .padding(.top, 2)
                    .padding(.bottom, 12)

                if let failure = model.pageFailure, !hasContent {
                    pageFailure(failure)
                } else {
                    playSection
                    if let challengeCard {
                        challengeRow(challengeCard)
                    }
                    if isLoggedIn && !activity.challengeAvailable {
                        challengeRetryRow
                    }
                    if let failure = model.pageFailure {
                        refreshBanner(failure)
                    }
                    liveSection
                    eventSection
                    rankingSection
                    nearbySection
                    Color.clear.frame(height: 24)
                }
            }
            .padding(.horizontal, 12)
        }
        .background(palette.page.ignoresSafeArea())
        .navigationTitle("追分")
        .navigationBarTitleDisplayMode(.large)
        .task {
            async let activityLoad: Void = activity.refreshIfNeeded()
            async let homeLoad: Void = model.onAppear()
            _ = await (activityLoad, homeLoad)
        }
        .refreshable {
            greetingHour = Calendar.current.component(.hour, from: Date())
            async let homeLoad: Void = model.refresh()
            async let activityLoad: Void = activity.refresh()
            _ = await (homeLoad, activityLoad)
        }
        .onAppear {
            greetingHour = Calendar.current.component(.hour, from: Date())
        }
        .onChange(of: session.identity) { _, _ in
            Task {
                await model.onIdentityChanged()
                await activity.refreshIfNeeded()
            }
        }
    }

    // MARK: - 开局卡

    @ViewBuilder
    private var playSection: some View {
        // 活动状态首次返回前不确定是否有进行中的对局，先占位，避免「来一局」闪成「继续对局」。
        if isLoggedIn && !activity.isLoaded && model.isPageLoading {
            skeleton(height: 128)
        } else if let card = ongoingCard {
            ongoingMatchCard(card)
        } else if isLoggedIn {
            startCard
        } else {
            guestCard
        }
    }

    private func ongoingMatchCard(_ card: HomeMatchCard) -> some View {
        VStack(spacing: 14) {
            Button {
                navigator.open(.liveMatch(matchId: card.id))
            } label: {
                VStack(spacing: 14) {
                    HStack(spacing: 6) {
                        HStack(spacing: 4) {
                            Circle()
                                .fill(card.isPending ? AppTheme.warning : palette.success)
                                .frame(width: 5, height: 5)
                            Text(card.statusText)
                                .font(.system(size: 10, weight: .semibold))
                        }
                        .foregroundStyle(card.isPending ? AppTheme.warning : palette.success)
                        .padding(.horizontal, 7)
                        .frame(height: 20)
                        .background(
                            (card.isPending ? AppTheme.warning : palette.success).opacity(0.12),
                            in: Capsule()
                        )

                        Text(card.metaText)
                            .font(.system(size: 11))
                            .foregroundStyle(palette.textSecondary)
                            .lineLimit(1)
                        Spacer(minLength: 6)
                        Text(card.durationText)
                            .font(.system(size: 11).monospacedDigit())
                            .foregroundStyle(palette.textMuted)
                    }

                    HStack(spacing: 10) {
                        versusPlayer(card.players[0], palette: palette)
                        Text(verbatim: "\(String(card.players[0].score)) : \(String(card.players[1].score))")
                            .font(.system(size: 30, weight: .heavy).monospacedDigit())
                            .foregroundStyle(palette.brandStrong)
                            .lineLimit(1)
                        versusPlayer(card.players[1], palette: palette)
                    }
                }
                .contentShape(Rectangle())
            }
            .buttonStyle(.plain)

            Button {
                navigator.open(.liveMatch(matchId: card.id))
            } label: {
                Text(card.actionLabel)
                    .font(.system(size: 14, weight: .semibold))
                    .foregroundStyle(.white)
                    .frame(maxWidth: .infinity)
                    .frame(height: 38)
                    .background(AppTheme.primary, in: Capsule())
            }
            .buttonStyle(.plain)
        }
        .padding(16)
        .background(palette.card, in: RoundedRectangle(cornerRadius: 16))
        .overlay {
            RoundedRectangle(cornerRadius: 16).stroke(AppTheme.primary.opacity(0.4))
        }
    }

    private func versusPlayer(_ player: HomeMatchPlayer, palette: AppPalette) -> some View {
        VStack(spacing: 6) {
            AvatarView(avatar: player.avatar, userId: player.id, size: 52)
            HStack(spacing: 4) {
                Text(player.name)
                    .font(.system(size: 13, weight: .semibold))
                    .foregroundStyle(palette.textPrimary)
                    .lineLimit(1)
                if player.isMe {
                    meTag(palette: palette)
                }
            }
        }
        .frame(maxWidth: .infinity)
    }

    private var startCard: some View {
        VStack(alignment: .leading, spacing: 8) {
            Text("来一局")
                .font(.system(size: 18, weight: .bold))
                .foregroundStyle(palette.textPrimary)
            Text("扫码或选择球友开局，比分实时同步给双方")
                .font(.system(size: 12))
                .foregroundStyle(palette.textSecondary)

            HStack(spacing: 8) {
                Button {
                    navigator.requestScan(isSignedIn: true)
                } label: {
                    playButtonLabel("发起对局", ghost: false)
                }
                .buttonStyle(.plain)

                NavigationLink(value: AppRoute.challengeCompose(target: nil)) {
                    playButtonLabel("约球", ghost: true)
                        .frame(width: 88)
                }
                .buttonStyle(.plain)
            }
            .padding(.top, 6)
        }
        .appCard(padding: 16)
    }

    private var guestCard: some View {
        VStack(alignment: .leading, spacing: 8) {
            Text("登录后开始记分")
                .font(.system(size: 18, weight: .bold))
                .foregroundStyle(palette.textPrimary)
            Text("发起对局、约球，排位赛成绩自动计入排行榜")
                .font(.system(size: 12))
                .foregroundStyle(palette.textSecondary)

            Button {
                navigator.openLogin()
            } label: {
                Text("登录 / 注册")
                    .font(.system(size: 14, weight: .semibold))
                    .foregroundStyle(.white)
                    .frame(maxWidth: .infinity)
                    .frame(height: 38)
                    .background(AppTheme.primary, in: Capsule())
            }
            .buttonStyle(.plain)
            .padding(.top, 6)
        }
        .appCard(padding: 16)
    }

    private func playButtonLabel(_ title: String, ghost: Bool) -> some View {
        Text(title)
            .font(.system(size: 14, weight: .semibold))
            .foregroundStyle(ghost ? palette.brandStrong : .white)
            .frame(maxWidth: .infinity)
            .frame(height: 38)
            .background(ghost ? palette.subtle : AppTheme.primary, in: Capsule())
            .overlay {
                if ghost {
                    Capsule().stroke(palette.border)
                }
            }
    }

    // MARK: - 约球

    private func challengeRow(_ card: HomeChallengeCard) -> some View {
        let route: AppRoute = card.matchId > 0
            ? .liveMatch(matchId: card.matchId)
            : .challengeWaiting(challengeId: card.challengeId)

        return NavigationLink(value: route) {
            HStack(spacing: 10) {
                VStack(alignment: .leading, spacing: 3) {
                    Text(card.title)
                        .font(.system(size: 14, weight: .semibold))
                        .foregroundStyle(palette.textPrimary)
                    if !card.subText.isEmpty {
                        Text(card.subText)
                            .font(.system(size: 11))
                            .foregroundStyle(palette.textSecondary)
                            .lineLimit(1)
                    }
                }
                Spacer(minLength: 8)
                Text(card.actionLabel)
                    .font(.system(size: 12, weight: .semibold))
                    .foregroundStyle(.white)
                    .padding(.horizontal, 14)
                    .frame(height: 28)
                    .background(AppTheme.primary, in: Capsule())
            }
            .padding(.horizontal, 14)
            .padding(.vertical, 12)
            .background(
                card.needsAction ? palette.brandTint : palette.card,
                in: RoundedRectangle(cornerRadius: 12)
            )
            .overlay {
                RoundedRectangle(cornerRadius: 12)
                    .stroke(card.needsAction ? AppTheme.primary.opacity(0.4) : palette.border)
            }
            .contentShape(RoundedRectangle(cornerRadius: 12))
        }
        .buttonStyle(.plain)
        .padding(.top, 10)
    }

    private var challengeRetryRow: some View {
        Button {
            Task { await activity.refresh() }
        } label: {
            Text("约球状态读取失败，点击重试")
                .font(.system(size: 12))
                .foregroundStyle(palette.textSecondary)
                .frame(maxWidth: .infinity, alignment: .leading)
                .padding(.horizontal, 14)
                .padding(.vertical, 10)
                .background(palette.card, in: RoundedRectangle(cornerRadius: 10))
                .overlay {
                    RoundedRectangle(cornerRadius: 10).stroke(palette.border)
                }
        }
        .buttonStyle(.plain)
        .padding(.top, 10)
    }

    // MARK: - 正在进行

    private var liveSection: some View {
        VStack(alignment: .leading, spacing: 8) {
            SectionHeader(title: "正在进行", moreTitle: "观赛大厅") {
                navigator.open(.spectatorLobby)
            }

            if case .failed(let failure) = model.livePhase {
                sectionError(failure.feedback(resource: "正在进行的对局")) {
                    Task { await model.retryLiveMatches() }
                }
            } else if !liveCards.isEmpty {
                VStack(spacing: 0) {
                    ForEach(Array(liveCards.enumerated()), id: \.element.id) { index, card in
                        if index > 0 {
                            divider
                        }
                        liveRowLink(card)
                    }
                }
                .padding(.horizontal, 14)
                .background(palette.card, in: RoundedRectangle(cornerRadius: 12))
                .overlay {
                    RoundedRectangle(cornerRadius: 12).stroke(palette.border)
                }
            } else if model.isPageLoading {
                skeleton(height: 112)
            } else {
                emptyCard(title: "暂无正在进行的对局", desc: "公开对局开打后会出现在这里。")
            }
        }
        .padding(.top, 20)
    }

    @ViewBuilder
    private func liveRowLink(_ card: SpectatorMatchCard) -> some View {
        if let route = liveDestination(card) {
            NavigationLink(value: route) {
                liveRow(card)
            }
            .buttonStyle(.plain)
        } else {
            liveRow(card)
        }
    }

    private func liveRow(_ card: SpectatorMatchCard) -> some View {
        VStack(alignment: .leading, spacing: 6) {
            HStack(spacing: 6) {
                Text(card.metaText)
                    .font(.system(size: 11))
                    .foregroundStyle(palette.textMuted)
                    .lineLimit(1)
                Spacer(minLength: 6)
                HStack(spacing: 4) {
                    if card.isLive {
                        Circle()
                            .fill(palette.success)
                            .frame(width: 5, height: 5)
                    }
                    Text(card.statusText)
                        .font(.system(size: 10, weight: .semibold))
                }
                .foregroundStyle(card.isLive ? palette.success : palette.textMuted)
                .padding(.horizontal, 7)
                .frame(height: 20)
                .background(card.isLive ? palette.success.opacity(0.12) : palette.subtle, in: Capsule())
            }

            ForEach(Array(card.players.enumerated()), id: \.offset) { _, player in
                HStack(spacing: 8) {
                    AvatarView(avatar: player.avatar, userId: player.id, size: 26)
                    Text(player.name)
                        .font(.system(size: 14, weight: .medium))
                        .foregroundStyle(palette.textPrimary)
                        .lineLimit(1)
                    if player.isMe {
                        meTag(palette: palette)
                    }
                    Spacer(minLength: 6)
                    Text(String(player.score))
                        .font(.system(size: 17, weight: .bold).monospacedDigit())
                        .foregroundStyle(palette.textPrimary)
                }
                .frame(minHeight: 30)
            }
        }
        .padding(.vertical, 10)
        .contentShape(Rectangle())
        .accessibilityElement(children: .combine)
    }

    private func liveDestination(_ card: SpectatorMatchCard) -> AppRoute? {
        let identity = session.identity
        switch SpectatorLobbyRules.destination(
            for: card,
            userId: identity?.userId ?? 0,
            isLoggedIn: identity != nil
        ) {
        case .report: return .matchDetail(matchId: card.id, source: nil, perspectiveUserId: nil)
        case .publicDetail: return .publicMatchDetail(matchId: card.id)
        case .live: return .liveMatch(matchId: card.id)
        case nil: return nil
        }
    }

    // MARK: - 赛事

    private var eventSection: some View {
        VStack(alignment: .leading, spacing: 8) {
            SectionHeader(title: "赛事", moreTitle: "全部赛讯") {
                navigator.select(.news, isSignedIn: isLoggedIn)
            }

            if case .failed(let failure) = model.eventPhase {
                sectionError(failure.feedback(resource: "赛事情报")) {
                    Task { await model.retryEventNews() }
                }
            } else if let card = model.topEventNews {
                eventCard(card)
            } else if model.isPageLoading {
                skeleton(height: 190)
            } else {
                emptyCard(title: "暂时没有赛讯", desc: "新的职业赛事公布后会出现在这里。")
            }
        }
        .padding(.top, 20)
    }

    private func eventCard(_ card: SaiXunCard) -> some View {
        NavigationLink(value: AppRoute.tournamentDetail(id: card.id)) {
            VStack(spacing: 0) {
                ZStack(alignment: .bottomLeading) {
                    LinearGradient(
                        colors: palette.heroColors,
                        startPoint: .topLeading,
                        endPoint: .bottomTrailing
                    )

                    if card.hasCover, case .remote(let url) = card.coverImage {
                        RemoteImage(url: url) {
                            Color.clear
                        }
                    }

                    LinearGradient(
                        colors: [.clear, Color.black.opacity(0.8)],
                        startPoint: UnitPoint(x: 0.5, y: 0.28),
                        endPoint: .bottom
                    )

                    VStack(alignment: .leading, spacing: 4) {
                        Text(card.title)
                            .font(.system(size: 17, weight: .bold))
                            .foregroundStyle(.white)
                            .lineLimit(2)
                        Text(card.metaText)
                            .font(.system(size: 11))
                            .foregroundStyle(.white.opacity(0.82))
                            .lineLimit(1)
                    }
                    .padding(.horizontal, 12)
                    .padding(.bottom, 11)
                    .frame(maxWidth: .infinity, alignment: .leading)
                }
                .frame(height: 150)
                .overlay(alignment: .topLeading) {
                    HStack(spacing: 4) {
                        if card.statusTone == .live {
                            Circle()
                                .fill(palette.success)
                                .frame(width: 5, height: 5)
                        }
                        Text(card.statusLabel)
                            .font(.system(size: 11, weight: .semibold))
                            .foregroundStyle(.white)
                    }
                    .padding(.horizontal, 8)
                    .frame(height: 22)
                    .background(Color.black.opacity(0.6), in: Capsule())
                    .padding(10)
                }
                .clipped()

                if let highlight = card.highlight {
                    HStack(spacing: 7) {
                        if !highlight.label.isEmpty {
                            Text(highlight.label)
                                .font(.system(size: 10, weight: .semibold))
                                .foregroundStyle(palette.brandStrong)
                                .padding(.horizontal, 6)
                                .padding(.vertical, 2)
                                .background(palette.brandTint, in: RoundedRectangle(cornerRadius: 6))
                        }
                        Text(highlight.primary)
                            .font(.system(size: 13, weight: .semibold))
                            .foregroundStyle(palette.textPrimary)
                            .lineLimit(1)
                        if !highlight.secondary.isEmpty {
                            Text(highlight.secondary)
                                .font(.system(size: 11))
                                .foregroundStyle(palette.textSecondary)
                                .lineLimit(1)
                        }
                        Spacer(minLength: 0)
                    }
                    .padding(.horizontal, 12)
                    .padding(.vertical, 11)
                }
            }
            .background(palette.card)
            .clipShape(RoundedRectangle(cornerRadius: 12))
            .overlay {
                RoundedRectangle(cornerRadius: 12).stroke(palette.border)
            }
        }
        .buttonStyle(.plain)
    }

    // MARK: - 排行榜

    private var rankingSection: some View {
        VStack(alignment: .leading, spacing: 8) {
            SectionHeader(title: "排行榜", caption: "中式八球", moreTitle: "完整榜单") {
                navigator.open(.ranking)
            }

            if case .failed(let failure) = model.leaderboardPhase {
                sectionError(failure.feedback(resource: "排行榜")) {
                    Task { await model.retryLeaderboard() }
                }
            } else if !model.leaderboardTopThree.isEmpty {
                rankingCard
            } else if model.isPageLoading {
                skeleton(height: 170)
            } else {
                emptyCard(title: "排行榜还在准备中", desc: "打一场排位赛，你的成绩也会出现在这里。")
            }
        }
        .padding(.top, 20)
    }

    private var rankingCard: some View {
        VStack(spacing: 0) {
            ForEach(Array(model.leaderboardTopThree.prefix(3).enumerated()), id: \.element.id) { index, item in
                if index > 0 {
                    divider
                }
                HStack(spacing: 10) {
                    Text(String(item.rank))
                        .font(.system(size: 12, weight: .heavy))
                        .foregroundStyle(medalForeground(item.rank))
                        .frame(width: 22, height: 22)
                        .background(medalBackground(item.rank), in: Circle())

                    MemberAvatarView(
                        avatar: item.avatar,
                        userId: item.userId,
                        vipLevel: item.vipLevel ?? 0,
                        rankLevel: item.rankLevel,
                        size: 34,
                        reserveFrame: true
                    )

                    VStack(alignment: .leading, spacing: 2) {
                        Text(item.nickname.isEmpty ? "球手" : item.nickname)
                            .font(.system(size: 14, weight: .semibold))
                            .foregroundStyle(palette.textPrimary)
                            .lineLimit(1)
                        Text(item.rankName.isEmpty ? "冲榜中" : item.rankName)
                            .font(.system(size: 11))
                            .foregroundStyle(palette.textMuted)
                            .lineLimit(1)
                    }

                    Spacer(minLength: 6)

                    HStack(alignment: .firstTextBaseline, spacing: 2) {
                        Text(String(item.rankScore))
                            .font(.system(size: 15, weight: .bold).monospacedDigit())
                            .foregroundStyle(palette.textPrimary)
                        Text("分")
                            .font(.system(size: 11))
                            .foregroundStyle(palette.textMuted)
                    }
                }
                .frame(minHeight: 50)
            }

            if isLoggedIn {
                divider
                NavigationLink(value: AppRoute.ranking) {
                    HStack(spacing: 10) {
                        Text("我")
                            .font(.system(size: 11, weight: .bold))
                            .foregroundStyle(palette.brandStrong)
                            .frame(width: 22, height: 22)
                            .background(palette.brandTint, in: Circle())
                        Text(HomeRules.myRankingText(model.myRanking))
                            .font(.system(size: 13))
                            .foregroundStyle(palette.textSecondary)
                            .lineLimit(1)
                        Spacer(minLength: 6)
                        Image(systemName: "chevron.right")
                            .font(.system(size: 10, weight: .semibold))
                            .foregroundStyle(palette.textMuted)
                    }
                    .frame(minHeight: 40)
                    .contentShape(Rectangle())
                }
                .buttonStyle(.plain)
            }
        }
        .padding(.horizontal, 14)
        .padding(.vertical, 6)
        .background(palette.card, in: RoundedRectangle(cornerRadius: 12))
        .overlay {
            RoundedRectangle(cornerRadius: 12).stroke(palette.border)
        }
    }

    private func medalBackground(_ rank: Int) -> Color {
        switch rank {
        case 1: AppTheme.primary
        case 2: palette.textMuted
        case 3: AppTheme.warning
        default: palette.subtle
        }
    }

    private func medalForeground(_ rank: Int) -> Color {
        (1...3).contains(rank) ? .white : palette.textSecondary
    }

    // MARK: - 附近球房

    private var nearbySection: some View {
        VStack(alignment: .leading, spacing: 8) {
            SectionHeader(title: "附近球房", moreTitle: "查看球房") {
                navigator.open(.venueList)
            }

            if let feedback = model.nearbyFeedback {
                sectionError(feedback) {
                    Task { await model.retryNearby() }
                }
            } else if model.nearbyPhase == .loading, model.nearbyVenues.isEmpty {
                skeleton(height: 96)
            } else if !model.nearbyVenues.isEmpty {
                VStack(spacing: 0) {
                    ForEach(Array(model.nearbyVenues.enumerated()), id: \.element.id) { index, venue in
                        if index > 0 {
                            divider
                        }
                        NavigationLink(value: AppRoute.venueDetail(id: venue.id)) {
                            venueRow(venue)
                        }
                        .buttonStyle(.plain)
                    }
                }
                .padding(.horizontal, 14)
                .background(palette.card, in: RoundedRectangle(cornerRadius: 12))
                .overlay {
                    RoundedRectangle(cornerRadius: 12).stroke(palette.border)
                }
            } else {
                venueEmptyCard
            }
        }
        .padding(.top, 20)
    }

    private func venueRow(_ venue: VenueInfo) -> some View {
        HStack(alignment: .center, spacing: 10) {
            VStack(alignment: .leading, spacing: 4) {
                Text(venue.name.isEmpty ? "球房" : venue.name)
                    .font(.system(size: 14, weight: .semibold))
                    .foregroundStyle(palette.textPrimary)
                    .lineLimit(1)
                Text(venueAddress(venue))
                    .font(.system(size: 11))
                    .foregroundStyle(palette.textSecondary)
                    .lineLimit(1)

                let tags = venueTags(venue)
                if !tags.isEmpty {
                    HStack(spacing: 8) {
                        ForEach(tags, id: \.self) { tag in
                            Text(tag)
                                .font(.system(size: 11))
                                .foregroundStyle(palette.textMuted)
                        }
                    }
                }
            }
            Spacer(minLength: 6)
            if VenueDistance.isVisible(venue.distance), let distance = venue.distance {
                Text(HomeRules.venueDistance(distance))
                    .font(.system(size: 12, weight: .semibold).monospacedDigit())
                    .foregroundStyle(palette.brandStrong)
            }
        }
        .padding(.vertical, 11)
        .contentShape(Rectangle())
    }

    private func venueAddress(_ venue: VenueInfo) -> String {
        if !venue.address.isEmpty { return venue.address }
        return venue.city.isEmpty ? "地址待补充" : venue.city
    }

    private func venueTags(_ venue: VenueInfo) -> [String] {
        var tags: [String] = []
        if venue.tableCount > 0 { tags.append("\(String(venue.tableCount))台") }
        if !venue.priceRange.isEmpty { tags.append(venue.priceRange) }
        if venue.checkinCount > 0 { tags.append("\(String(venue.checkinCount))人签到") }
        return tags
    }

    private var venueEmptyCard: some View {
        let action = model.venueEmptyAction
        return VStack(spacing: 8) {
            Text("附近暂无球房")
                .font(.system(size: 14, weight: .semibold))
                .foregroundStyle(palette.textPrimary)
            Text(action.desc)
                .font(.system(size: 12))
                .foregroundStyle(palette.textSecondary)
                .multilineTextAlignment(.center)
            Button {
                navigator.open(.venueSubmit)
            } label: {
                Text(action.text)
                    .font(.system(size: 13, weight: .semibold))
                    .foregroundStyle(.white)
                    .padding(.horizontal, 18)
                    .frame(height: 34)
                    .background(AppTheme.primary, in: Capsule())
            }
            .buttonStyle(.plain)
            .padding(.top, 4)
        }
        .frame(maxWidth: .infinity)
        .padding(.vertical, 18)
        .padding(.horizontal, 14)
        .background(palette.card, in: RoundedRectangle(cornerRadius: 12))
        .overlay {
            RoundedRectangle(cornerRadius: 12).stroke(palette.border)
        }
    }

    // MARK: - 状态块

    private func pageFailure(_ feedback: LoadFailure.Feedback) -> some View {
        VStack(spacing: 10) {
            Image(systemName: "info.circle")
                .font(.system(size: 34))
                .foregroundStyle(palette.textMuted)
                .accessibilityHidden(true)
            Text(feedback.title)
                .font(.system(size: 16, weight: .bold))
                .foregroundStyle(palette.textPrimary)
            Text(feedback.description)
                .font(.system(size: 13))
                .foregroundStyle(palette.textSecondary)
                .multilineTextAlignment(.center)
            Button(feedback.actionText) {
                if feedback.retryable {
                    Task { await model.refresh() }
                } else {
                    dismiss()
                }
            }
            .buttonStyle(.appPill)
            .padding(.top, 4)
        }
        .frame(maxWidth: .infinity)
        .padding(.vertical, 48)
        .padding(.horizontal, 20)
        .background(palette.card, in: RoundedRectangle(cornerRadius: 16))
        .overlay {
            RoundedRectangle(cornerRadius: 16).stroke(palette.border)
        }
    }

    private func sectionError(_ feedback: LoadFailure.Feedback, retry: @escaping () -> Void) -> some View {
        VStack(alignment: .leading, spacing: 5) {
            Text(feedback.title)
                .font(.system(size: 14, weight: .semibold))
                .foregroundStyle(palette.textPrimary)
            Text(feedback.description)
                .font(.system(size: 12))
                .foregroundStyle(palette.textSecondary)
            if feedback.retryable {
                Button("重试", action: retry)
                    .font(.system(size: 12, weight: .semibold))
                    .foregroundStyle(palette.brandStrong)
                    .buttonStyle(.plain)
                    .padding(.top, 4)
            }
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .padding(.horizontal, 14)
        .padding(.vertical, 12)
        .background(palette.card, in: RoundedRectangle(cornerRadius: 12))
        .overlay {
            RoundedRectangle(cornerRadius: 12).stroke(palette.border)
        }
    }

    private func refreshBanner(_ feedback: LoadFailure.Feedback) -> some View {
        HStack(spacing: 10) {
            Text(feedback.description)
                .font(.system(size: 12))
                .foregroundStyle(palette.textSecondary)
                .lineLimit(2)
            Spacer(minLength: 8)
            Button("重试") {
                Task { await model.refresh() }
            }
            .font(.system(size: 12, weight: .semibold))
            .foregroundStyle(palette.brandStrong)
            .buttonStyle(.plain)
        }
        .padding(.horizontal, 12)
        .padding(.vertical, 10)
        .background(palette.card, in: RoundedRectangle(cornerRadius: 10))
        .overlay {
            RoundedRectangle(cornerRadius: 10).stroke(palette.border)
        }
        .padding(.top, 10)
    }

    private func emptyCard(title: String, desc: String) -> some View {
        VStack(alignment: .leading, spacing: 4) {
            Text(title)
                .font(.system(size: 14, weight: .semibold))
                .foregroundStyle(palette.textPrimary)
            Text(desc)
                .font(.system(size: 12))
                .foregroundStyle(palette.textSecondary)
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .padding(.horizontal, 14)
        .padding(.vertical, 14)
        .background(palette.card, in: RoundedRectangle(cornerRadius: 12))
        .overlay {
            RoundedRectangle(cornerRadius: 12).stroke(palette.border)
        }
    }

    private func skeleton(height: CGFloat) -> some View {
        RoundedRectangle(cornerRadius: 12)
            .fill(palette.border)
            .frame(height: height)
            .opacity(0.6)
            .accessibilityHidden(true)
    }

    private var divider: some View {
        Rectangle()
            .fill(palette.border)
            .frame(height: 1)
    }

    private func meTag(palette: AppPalette) -> some View {
        Text("我")
            .font(.system(size: 9, weight: .bold))
            .foregroundStyle(palette.brandStrong)
            .padding(.horizontal, 5)
            .frame(height: 15)
            .background(palette.brandTint, in: RoundedRectangle(cornerRadius: 4))
    }
}
