import SwiftUI

struct MatchReportRoute: Hashable {
    let matchId: Int64
    var fromHistory = false
}

enum HistorySection: Hashable {
    case participant
    case referee
}

/// 对局记录：参赛与执裁两个分栏，各自持有独立的分页状态；战报为只读页面。
struct MatchHistoryView: View {
    @EnvironmentObject private var session: SessionStore
    @EnvironmentObject private var navigator: AppNavigator
    @Environment(\.colorScheme) private var colorScheme
    @StateObject private var participant: PagedHistoryList<MatchListItem>
    @StateObject private var referee: PagedHistoryList<RefereeHistoryItem>
    @State private var section: HistorySection
    @State private var ongoingMatch: OngoingRefereeMatch?
    @State private var ongoingLoaded: Bool
    private let api: MatchHistoryAPI

    init(
        session: SessionStore,
        section: HistorySection = .participant,
        ongoingMatch: OngoingRefereeMatch? = nil,
        participantList: PagedHistoryList<MatchListItem>? = nil,
        refereeList: PagedHistoryList<RefereeHistoryItem>? = nil
    ) {
        let api = MatchHistoryAPI(session: session)
        self.api = api
        _participant = StateObject(wrappedValue: participantList ?? PagedHistoryList(session: session) { page in
            try await api.participantPage(page: page)
        })
        _referee = StateObject(wrappedValue: refereeList ?? PagedHistoryList(session: session) { page in
            try await api.refereePage(page: page)
        })
        _section = State(initialValue: section)
        _ongoingMatch = State(initialValue: ongoingMatch)
        _ongoingLoaded = State(initialValue: ongoingMatch != nil)
    }

    private var palette: AppPalette { AppTheme.palette(for: colorScheme) }

    var body: some View {
        Group {
            if session.identity == nil {
                MatchHistoryStateMessage(icon: "lock.fill", title: "请登录后查看对局记录", detail: nil)
            } else {
                VStack(spacing: 0) {
                    Picker("记录类型", selection: $section) {
                        Text("参赛").tag(HistorySection.participant)
                        Text("执裁").tag(HistorySection.referee)
                    }
                    .pickerStyle(.segmented)
                    .padding(.horizontal, 20)
                    .padding(.vertical, 12)

                    switch section {
                    case .participant:
                        PagedHistoryListContent(
                            list: participant,
                            emptyTitle: "暂无对局记录",
                            emptyHint: "快去发起一场PK吧！"
                        ) { item in
                            ParticipantHistoryRow(item: item)
                                .background(HiddenRowLink(value: MatchReportRoute(matchId: item.id, fromHistory: true)))
                        }
                    case .referee:
                        VStack(spacing: 0) {
                            if let ongoingMatch {
                                NavigationLink(value: AppRoute.liveMatch(matchId: ongoingMatch.matchId)) {
                                    OngoingRefereeCard(match: ongoingMatch)
                                }
                                .buttonStyle(.plain)
                                .padding(.horizontal, 16)
                                .padding(.bottom, 8)
                            }
                            PagedHistoryListContent(
                                list: referee,
                                emptyTitle: "暂无执裁记录",
                                emptyHint: "去对局首页扫码担任裁判",
                                emptyActionTitle: "前往对局大厅",
                                emptyAction: { navigator.select(.home, isSignedIn: session.identity != nil) }
                            ) { item in
                                // 已取消的执裁记录没有可读战报，只展示状态。
                                if item.status == 2 {
                                    RefereeHistoryRow(item: item, isNavigable: true)
                                        .background(HiddenRowLink(value: MatchReportRoute(matchId: item.id, fromHistory: true)))
                                } else {
                                    RefereeHistoryRow(item: item, isNavigable: false)
                                }
                            }
                        }
                    }
                }
            }
        }
        .background(palette.page.ignoresSafeArea())
        .navigationTitle("对局记录")
        .navigationBarTitleDisplayMode(.inline)
        .navigationDestination(for: MatchReportRoute.self) { route in
            MatchReportView(session: session, matchId: route.matchId, fromHistory: route.fromHistory)
        }
        .task { await loadOngoingMatch() }
        .onChange(of: session.identity) { _, _ in
            // 退出、切号或会话失效后清空两个分栏与进行中入口，旧账号的在途结果不再采纳。
            participant.syncIdentity()
            referee.syncIdentity()
            ongoingMatch = nil
            ongoingLoaded = false
            Task { await loadOngoingMatch() }
        }
    }

    /// 小程序进入对局记录时总会检查一次当前对局；失败静默，不影响历史列表。
    private func loadOngoingMatch() async {
        guard !ongoingLoaded, let identity = session.identity else { return }
        ongoingLoaded = true
        do {
            let ref = try await api.currentMatch()
            guard session.identity == identity else { return }
            ongoingMatch = OngoingRefereeMatch(ref: ref)
        } catch {
            // 没有进行中的对局或读取失败都按无卡片处理。
        }
    }
}

private struct PagedHistoryListContent<Item: Identifiable, Row: View>: View where Item.ID: Hashable {
    @ObservedObject var list: PagedHistoryList<Item>
    let emptyTitle: String
    let emptyHint: String
    var emptyActionTitle: String?
    var emptyAction: (() -> Void)?
    @ViewBuilder let row: (Item) -> Row

    @Environment(\.colorScheme) private var colorScheme

    private var palette: AppPalette { AppTheme.palette(for: colorScheme) }

    var body: some View {
        content
            .task { await list.loadIfNeeded() }
    }

    @ViewBuilder
    private var content: some View {
        if list.items.isEmpty {
            switch list.phase {
            case .failed(let message):
                MatchHistoryStateMessage(
                    icon: "exclamationmark.triangle",
                    title: "加载失败",
                    detail: message,
                    actionTitle: "重试",
                    action: { Task { await list.retry() } }
                )
            case .empty:
                MatchHistoryStateMessage(
                    icon: "list.bullet",
                    title: emptyTitle,
                    detail: emptyHint,
                    actionTitle: emptyActionTitle,
                    action: emptyAction
                )
            case .idle, .loading, .loaded:
                ProgressView()
                    .frame(maxWidth: .infinity, maxHeight: .infinity)
            }
        } else {
            List {
                if let refreshError = list.refreshError {
                    HStack {
                        Text(refreshError)
                            .font(.system(size: 13))
                            .foregroundStyle(palette.textSecondary)
                        Spacer()
                        Button("重试") { Task { await list.refresh() } }
                            .font(.system(size: 13, weight: .semibold))
                    }
                    .listRowBackground(Color.clear)
                    .listRowSeparator(.hidden)
                }

                ForEach(list.items) { item in
                    row(item)
                        .listRowBackground(Color.clear)
                        .listRowSeparator(.hidden)
                        .onAppear {
                            // 滚动到最后一条时请求下一页；失败与“没有更多”由页脚表达。
                            if item.id == list.items.last?.id {
                                Task { await list.loadMore() }
                            }
                        }
                }

                footer
                    .listRowBackground(Color.clear)
                    .listRowSeparator(.hidden)
            }
            .listStyle(.plain)
            .scrollContentBackground(.hidden)
            .refreshable { await list.refresh() }
        }
    }

    @ViewBuilder
    private var footer: some View {
        if list.isLoadingMore {
            ProgressView()
                .frame(maxWidth: .infinity)
        } else if let loadMoreError = list.loadMoreError {
            VStack(spacing: 8) {
                Text(loadMoreError)
                    .font(.system(size: 13))
                    .foregroundStyle(palette.textSecondary)
                Button("重试") { Task { await list.loadMore() } }
                    .font(.system(size: 13, weight: .semibold))
            }
            .frame(maxWidth: .infinity)
        } else if list.loadedPage > 0, !list.hasMore {
            Text("没有更多了")
                .font(.system(size: 13))
                .foregroundStyle(palette.textSecondary)
                .frame(maxWidth: .infinity)
        }
    }
}

private struct ParticipantHistoryRow: View {
    @Environment(\.colorScheme) private var colorScheme
    let item: MatchListItem

    var body: some View {
        let palette = AppTheme.palette(for: colorScheme)
        let outcome = MatchOutcome(result: item.result)
        let timeText = MatchTimeFormatting.relative(item.matchTime)

        HStack(spacing: 12) {
            AvatarView(avatar: item.opponentAvatar, userId: item.opponentId, size: 44)
            VStack(alignment: .leading, spacing: 6) {
                HStack(spacing: 8) {
                    Text(item.opponentName)
                        .font(.system(size: 16, weight: .semibold))
                        .foregroundStyle(palette.textPrimary)
                        .lineLimit(1)
                    GameTypeTag(text: item.gameTypeName)
                }
                Text(timeText)
                    .font(.system(size: 13))
                    .foregroundStyle(palette.textSecondary)
            }
            Spacer(minLength: 8)
            VStack(alignment: .trailing, spacing: 6) {
                OutcomeBadge(outcome: outcome)
                Text("\(String(item.myScore)) - \(String(item.opponentScore))")
                    .font(.system(size: 15, weight: .semibold).monospacedDigit())
                    .foregroundStyle(palette.textPrimary)
            }
            Image(systemName: "chevron.right")
                .font(.system(size: 13, weight: .semibold))
                .foregroundStyle(palette.textSecondary)
        }
        .padding(16)
        .background(palette.card, in: RoundedRectangle(cornerRadius: 16))
        .accessibilityElement(children: .combine)
        .accessibilityLabel(
            "对手\(item.opponentName)，\(item.gameTypeName)，\(timeText)，\(outcome?.title ?? "结果未知")，比分\(item.myScore)比\(item.opponentScore)"
        )
    }
}

private struct RefereeHistoryRow: View {
    @Environment(\.colorScheme) private var colorScheme
    let item: RefereeHistoryItem
    let isNavigable: Bool

    var body: some View {
        let palette = AppTheme.palette(for: colorScheme)
        let isCancelled = item.status == 3
        let durationText = MatchFormatting.duration(seconds: item.refereeDurationSeconds)
        let statusText = item.statusText.isEmpty ? (isCancelled ? "已取消" : "已完成") : item.statusText
        let matchup = "\(item.player1Name) VS \(item.player2Name)"

        VStack(spacing: 10) {
            HStack(alignment: .center, spacing: 10) {
                RefereePlayer(
                    userId: item.player1Id,
                    name: item.player1Name,
                    avatarUrl: item.player1Avatar
                )
                VStack(spacing: 2) {
                    Text("\(String(item.player1Score)) - \(String(item.player2Score))")
                        .font(.system(size: 18, weight: .bold).monospacedDigit())
                        .foregroundStyle(palette.textPrimary)
                    Text("VS")
                        .font(.system(size: 10, weight: .semibold))
                        .foregroundStyle(palette.textMuted)
                }
                .frame(minWidth: 72)
                RefereePlayer(
                    userId: item.player2Id,
                    name: item.player2Name,
                    avatarUrl: item.player2Avatar
                )
            }

            HStack(spacing: 8) {
                GameTypeTag(text: item.gameTypeName)
                Text(statusText)
                    .font(.system(size: 11, weight: .semibold))
                    .foregroundStyle(isCancelled ? palette.textSecondary : palette.success)
                    .padding(.horizontal, 8)
                    .padding(.vertical, 2)
                    .background(palette.subtle, in: RoundedRectangle(cornerRadius: 6))
                Spacer(minLength: 8)
                if let durationText {
                    Text("执裁 \(durationText)")
                        .font(.system(size: 11))
                        .foregroundStyle(palette.textMuted)
                }
                if isNavigable {
                    Image(systemName: "chevron.right")
                        .font(.system(size: 12, weight: .semibold))
                        .foregroundStyle(palette.textSecondary)
                }
            }
        }
        .padding(16)
        .background(palette.card, in: RoundedRectangle(cornerRadius: 16))
        .opacity(isCancelled ? 0.6 : 1)
        .accessibilityElement(children: .combine)
        .accessibilityLabel(
            "\(matchup)，\(item.gameTypeName)，\(statusText)，比分\(item.player1Score)比\(item.player2Score)"
                + (durationText.map { "，执裁\($0)" } ?? "")
        )
    }
}

private struct RefereePlayer: View {
    @Environment(\.colorScheme) private var colorScheme
    let userId: Int64
    let name: String
    let avatarUrl: String

    var body: some View {
        let palette = AppTheme.palette(for: colorScheme)
        VStack(spacing: 4) {
            AvatarView(avatar: avatarUrl, userId: userId, size: 40)
            Text(name)
                .font(.system(size: 12, weight: .semibold))
                .foregroundStyle(palette.textPrimary)
                .lineLimit(1)
                .minimumScaleFactor(0.8)
        }
        .frame(maxWidth: .infinity)
    }
}

/// 执裁记录页顶部的「继续执裁」入口。
private struct OngoingRefereeCard: View {
    @Environment(\.colorScheme) private var colorScheme
    let match: OngoingRefereeMatch

    var body: some View {
        let palette = AppTheme.palette(for: colorScheme)
        HStack(spacing: 12) {
            HStack(spacing: 8) {
                Image(systemName: "arrow.triangle.2.circlepath")
                    .font(.system(size: 26))
                    .foregroundStyle(AppTheme.primary)
                Text("继续执裁")
                    .font(.system(size: 16, weight: .semibold))
                    .foregroundStyle(palette.textPrimary)
            }
            Spacer(minLength: 8)
            VStack(alignment: .trailing, spacing: 3) {
                Text("\(match.player1Name) VS \(match.player2Name)")
                    .font(.system(size: 12, weight: .semibold))
                    .foregroundStyle(palette.textSecondary)
                    .lineLimit(1)
                Text("\(String(match.player1Score)) - \(String(match.player2Score))")
                    .font(.system(size: 15, weight: .bold).monospacedDigit())
                    .foregroundStyle(palette.textPrimary)
            }
            Image(systemName: "chevron.right")
                .font(.system(size: 12, weight: .semibold))
                .foregroundStyle(palette.textMuted)
        }
        .padding(14)
        .background(palette.brandTint, in: RoundedRectangle(cornerRadius: 16))
        .overlay { RoundedRectangle(cornerRadius: 16).stroke(AppTheme.primary.opacity(0.4)) }
        .accessibilityElement(children: .combine)
        .accessibilityLabel(
            "继续执裁，\(match.player1Name)对\(match.player2Name)，比分\(match.player1Score)比\(match.player2Score)"
        )
    }
}

private struct OutcomeBadge: View {
    @Environment(\.colorScheme) private var colorScheme
    let outcome: MatchOutcome?

    var body: some View {
        let palette = AppTheme.palette(for: colorScheme)
        let isWin = outcome == .win
        Text(outcome?.title ?? "结果未知")
            .font(.system(size: 12, weight: .semibold))
            .foregroundStyle(isWin ? palette.textPrimary : palette.textSecondary)
            .padding(.horizontal, 10)
            .padding(.vertical, 4)
            .background(
                isWin ? AppTheme.primary.opacity(0.22) : palette.subtle,
                in: Capsule()
            )
    }
}

private struct GameTypeTag: View {
    @Environment(\.colorScheme) private var colorScheme
    let text: String

    var body: some View {
        Text(text)
            .font(.system(size: 12))
            .foregroundStyle(AppTheme.palette(for: colorScheme).textSecondary)
            .padding(.horizontal, 8)
            .padding(.vertical, 2)
            .background(AppTheme.palette(for: colorScheme).subtle, in: RoundedRectangle(cornerRadius: 6))
    }
}

/// 头像加载失败、地址为空或无效时，底层占位图保持可见。
struct MatchHistoryAvatar: View {
    let urlString: String
    let size: CGFloat

    var body: some View {
        ZStack {
            Circle().fill(AppTheme.primary.opacity(0.14))
            Image(systemName: "person.crop.circle.fill")
                .resizable()
                .scaledToFit()
                .foregroundStyle(AppTheme.primary)
                .padding(size * 0.12)
            if !urlString.isEmpty, let url = URL(string: urlString) {
                AsyncImage(url: url) { image in
                    image.resizable().scaledToFill()
                } placeholder: {
                    Color.clear
                }
            }
        }
        .frame(width: size, height: size)
        .clipShape(Circle())
        .accessibilityHidden(true)
    }
}

struct MatchHistoryStateMessage: View {
    @Environment(\.colorScheme) private var colorScheme
    let icon: String
    let title: String
    let detail: String?
    var actionTitle: String?
    var action: (() -> Void)?

    var body: some View {
        let palette = AppTheme.palette(for: colorScheme)
        VStack(spacing: 12) {
            Image(systemName: icon)
                .font(.system(size: 40))
                .foregroundStyle(palette.textSecondary)
                .accessibilityHidden(true)
            Text(title)
                .font(.system(size: 17, weight: .semibold))
                .foregroundStyle(palette.textPrimary)
            if let detail {
                Text(detail)
                    .font(.system(size: 14))
                    .foregroundStyle(palette.textSecondary)
                    .multilineTextAlignment(.center)
            }
            if let actionTitle, let action {
                Button(action: action) {
                    Text(actionTitle)
                        .font(.system(size: 15, weight: .semibold))
                        .padding(.horizontal, 24)
                        .padding(.vertical, 10)
                        .foregroundStyle(.white)
                        .background(AppTheme.primary, in: RoundedRectangle(cornerRadius: 10))
                }
                .padding(.top, 4)
            }
        }
        .padding(24)
        .frame(maxWidth: .infinity, maxHeight: .infinity)
    }
}

/// List 里的 NavigationLink 会自带系统箭头；把链接垫在行下面并隐藏，只保留卡片内自绘的箭头，整行仍可点。
private struct HiddenRowLink<Value: Hashable>: View {
    let value: Value

    var body: some View {
        NavigationLink(value: value) { EmptyView() }
            .opacity(0)
    }
}
