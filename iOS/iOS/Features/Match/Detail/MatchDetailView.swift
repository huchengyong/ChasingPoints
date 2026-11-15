import SwiftUI

/// 对局详情（小程序 `matchDetail`）：固定选手1 / 选手2 的中性视角；观战入口读公开接口并显示徽标。
struct MatchDetailView: View {
    @EnvironmentObject private var session: SessionStore
    @EnvironmentObject private var navigator: AppNavigator
    @Environment(\.colorScheme) private var colorScheme
    @Environment(\.dismiss) private var dismiss
    @StateObject private var model: MatchDetailModel
    private let gearModel: MatchGearCardModel?

    init(
        session: SessionStore,
        matchId: Int64,
        mode: String = "",
        source: String? = nil,
        perspectiveUserId: Int64? = nil,
        model: MatchDetailModel? = nil,
        gearModel: MatchGearCardModel? = nil
    ) {
        _model = StateObject(wrappedValue: model ?? MatchDetailModel(
            session: session,
            matchId: matchId,
            mode: mode,
            source: source ?? "",
            perspectiveUserId: perspectiveUserId ?? 0
        ))
        self.gearModel = gearModel
    }

    private var palette: AppPalette { AppTheme.palette(for: colorScheme) }

    var body: some View {
        Group {
            switch model.phase {
            case .loading:
                VStack(spacing: 10) {
                    ProgressView()
                    Text("加载中...")
                        .font(.system(size: 13))
                        .foregroundStyle(palette.textSecondary)
                }
                .frame(maxWidth: .infinity, maxHeight: .infinity)
            case .failed(let failure):
                failureView(failure)
            case .loaded(let data):
                content(data)
            }
        }
        .background(palette.page.ignoresSafeArea())
        .navigationTitle(navigationTitle)
        .navigationBarTitleDisplayMode(.inline)
        .safeAreaInset(edge: .bottom) {
            if model.showsLoginHint {
                loginHint
            }
        }
        .task { await model.start() }
        .onDisappear { model.stop() }
        .onChange(of: session.identity) { _, _ in
            // 退出或切号后按新身份重新读取，公开详情与参与者详情不能混用。
            Task { await model.load() }
        }
    }

    private func failureView(_ failure: LoadFailure) -> some View {
        let feedback = failure.feedback(resource: "对局")
        return StateMessageView(
            icon: "exclamationmark.triangle",
            title: feedback.title,
            detail: feedback.description,
            actionTitle: feedback.actionText
        ) {
            if feedback.retryable {
                Task { await model.load() }
            } else {
                dismiss()
            }
        }
    }

    private var navigationTitle: String {
        if let data = model.viewData, data.matchData.viewerRole == "referee" {
            return "执裁详情"
        }
        return model.isSpectate ? "观战" : "对局详情"
    }

    private func content(_ data: MatchDetailViewData) -> some View {
        ScrollView {
            VStack(spacing: 14) {
                if model.isSpectate {
                    HStack(spacing: 6) {
                        Image(systemName: "eye.fill")
                        Text("观战模式")
                    }
                    .font(.system(size: 12, weight: .semibold))
                    .foregroundStyle(.white)
                    .padding(.horizontal, 12)
                    .padding(.vertical, 5)
                    .background(palette.info, in: Capsule())
                    .frame(maxWidth: .infinity, alignment: .leading)
                }

                playersCard(data)

                if refereeCard(data).hasReferee {
                    refereeSection(data)
                }

                let canReplay = MatchReplayEntry.canOpenMatchReplay(
                    status: data.matchData.status,
                    viewerRole: data.matchData.viewerRole,
                    source: model.source,
                    perspectiveUserId: model.perspectiveUserId,
                    currentUserId: session.identity?.userId ?? 0
                )
                let showH2H = data.matchData.status == 2 && data.matchData.viewerRole != "referee"
                let rematch = MatchDetailRules.rematchTarget(data)
                if canReplay || showH2H || rematch != nil {
                    entryRow(canReplay: canReplay, showH2H: showH2H, rematch: rematch)
                }

                MatchGearCardView(
                    session: session,
                    matchId: model.matchId,
                    enabled: MatchDetailRules.shouldShowMatchGear(data, isSpectate: model.isSpectate),
                    model: gearModel
                )

                if data.matchData.gameType == 2, let score = data.scoreBreakdown {
                    ChasingScoreCard(view: score, showEvents: true)
                } else {
                    roundHistory(data)
                }
            }
            .padding(16)
        }
    }

    // MARK: - 双方与比分

    private func playersCard(_ data: MatchDetailViewData) -> some View {
        let match = data.matchData
        let chasing = match.gameType == 2
        return CardBox {
            VStack(spacing: 12) {
                HStack(alignment: .top) {
                    playerColumn(data.player1Info, userId: data.player1UserId, fallback: "玩家1")
                    Text("VS")
                        .font(.system(size: 13, weight: .semibold))
                        .foregroundStyle(palette.textMuted)
                        .padding(.top, 36)
                    playerColumn(data.player2Info, userId: data.player2UserId, fallback: "玩家2")
                }

                Text("\(String(match.player1Score)) : \(String(match.player2Score))")
                    .font(.system(size: 30, weight: .bold).monospacedDigit())
                    .foregroundStyle(palette.textPrimary)

                Text(matchInfoText(match))
                    .font(.system(size: 12))
                    .foregroundStyle(palette.textSecondary)

                HStack(spacing: 8) {
                    if !chasing {
                        statItem(label: "总局数", value: String(match.totalRounds))
                    }
                    if !chasing && match.status != 2 {
                        statItem(label: "当前局", value: String(match.currentRound))
                    }
                    statItem(
                        label: "状态",
                        value: statusText(match.status),
                        tint: match.status == 2 ? palette.success : AppTheme.primary
                    )
                }
            }
        }
    }

    private func playerColumn(_ info: MatchDetailSide, userId: Int64, fallback: String) -> some View {
        VStack(spacing: 6) {
            MemberAvatarView(
                avatar: info.avatar,
                userId: userId,
                vipLevel: info.vipLevel,
                rankLevel: info.rankLevel,
                size: 50
            )
            Text(info.name.isEmpty ? fallback : info.name)
                .font(.system(size: 14, weight: .semibold))
                .foregroundStyle(palette.textPrimary)
                .lineLimit(1)
        }
        .frame(maxWidth: .infinity)
    }

    private func statItem(label: String, value: String, tint: Color? = nil) -> some View {
        VStack(spacing: 3) {
            Text(label)
                .font(.system(size: 10))
                .foregroundStyle(palette.textMuted)
            Text(value)
                .font(.system(size: 13, weight: .semibold))
                .foregroundStyle(tint ?? palette.textPrimary)
        }
        .frame(maxWidth: .infinity)
        .padding(.vertical, 8)
        .background(palette.subtle, in: RoundedRectangle(cornerRadius: 8))
    }

    private func matchInfoText(_ match: MatchDetailMatch) -> String {
        let name = LiveMatchText.gameTitle(match.gameType)
        let duration = LiveMatchText.duration(match.durationSeconds)
        return duration.isEmpty ? name : "\(name) · 用时 \(duration)"
    }

    private func statusText(_ status: Int) -> String {
        switch status {
        case 2: "已结束"
        case 3: "已取消"
        default: "进行中"
        }
    }

    // MARK: - 裁判

    private func refereeCard(_ data: MatchDetailViewData) -> RefereeIdentityCard {
        let match = data.matchData
        return MatchRefereeView.identityCard(
            refereeBound: match.refereeBound,
            refereeUserId: match.refereeUserId,
            refereeName: match.refereeName,
            refereeAvatar: match.refereeAvatar,
            refereeJoinedAt: match.refereeJoinedAt,
            refereeDurationSeconds: match.refereeDurationSeconds,
            completedByUserId: match.completedByUserId,
            completionSource: match.completionSource,
            status: match.status
        )
    }

    private func refereeSection(_ data: MatchDetailViewData) -> some View {
        let card = refereeCard(data)
        return CardBox {
            HStack(alignment: .top, spacing: 12) {
                MatchHistoryAvatar(urlString: card.refereeAvatar, size: 48)
                VStack(alignment: .leading, spacing: 3) {
                    Text(card.neutralLabel)
                        .font(.system(size: 11))
                        .foregroundStyle(palette.textMuted)
                    Text(card.refereeName)
                        .font(.system(size: 15, weight: .semibold))
                        .foregroundStyle(palette.textPrimary)
                    if !card.refereeJoinedAtText.isEmpty {
                        Text("加入时间 \(card.refereeJoinedAtText)")
                            .font(.system(size: 12))
                            .foregroundStyle(palette.textSecondary)
                    }
                    if !card.refereeDurationText.isEmpty {
                        Text("执裁时长 \(card.refereeDurationText)")
                            .font(.system(size: 12))
                            .foregroundStyle(palette.textSecondary)
                    }
                    if card.hasReliableAttribution {
                        Text(card.completionLabel)
                            .font(.system(size: 12))
                            .foregroundStyle(palette.textSecondary)
                    } else if data.matchData.status == 2 {
                        Text("本场曾绑定裁判")
                            .font(.system(size: 12))
                            .foregroundStyle(palette.textMuted)
                    }
                }
                Spacer(minLength: 0)
            }
        }
    }

    // MARK: - 入口

    private func entryRow(
        canReplay: Bool,
        showH2H: Bool,
        rematch: ChallengeComposeTarget?
    ) -> some View {
        VStack(spacing: 8) {
            if canReplay {
                Button("单场复盘") {
                    navigator.open(.matchReplay(matchId: model.matchId))
                }
                .buttonStyle(.appSecondary)
            }
            if showH2H {
                Button("查看双方交锋记录") { openH2H() }
                    .buttonStyle(.appSecondary)
            }
            if let rematch {
                Button("再约一场") { navigator.open(.challengeCompose(target: rematch)) }
                    .buttonStyle(.appSecondary)
            }
        }
    }

    private func openH2H() {
        guard let data = model.viewData else { return }
        let player1Id = data.player1UserId
        let player2Id = data.player2UserId
        guard player1Id > 0, player2Id > 0 else {
            return
        }
        navigator.open(.h2hRecord(
            opponentId: player2Id,
            targetUserId: player1Id,
            targetName: data.player1Info.name,
            targetAvatar: data.player1Info.avatar,
            opponentName: data.player2Info.name
        ))
    }

    // MARK: - 局记录

    private func roundHistory(_ data: MatchDetailViewData) -> some View {
        CardBox {
            VStack(alignment: .leading, spacing: 10) {
                Text("局记录")
                    .font(.system(size: 16, weight: .semibold))
                    .foregroundStyle(palette.textPrimary)

                if data.roundRecords.isEmpty {
                    Text("暂无已完成的局")
                        .font(.system(size: 13))
                        .foregroundStyle(palette.textMuted)
                } else {
                    ForEach(Array(data.roundRecords.enumerated()), id: \.offset) { _, round in
                        roundRow(round, data: data)
                    }
                }
            }
        }
    }

    private func roundRow(_ round: MatchRoundRecordView, data: MatchDetailViewData) -> some View {
        let tone: Color = data.roundRecordsNeutral
            ? palette.textPrimary
            : (round.winner == 1 ? palette.brandStrong : palette.textSecondary)
        let resultText: String = {
            if !data.roundRecordsNeutral {
                return round.resultText
            }
            let isRight = round.winner == 2
            let info = isRight ? data.player2Info : data.player1Info
            let fallback = isRight ? "选手2" : "选手1"
            return "\(info.name.isEmpty ? fallback : info.name) 胜"
        }()
        return HStack(alignment: .top) {
            VStack(alignment: .leading, spacing: 3) {
                Text("第 \(String(round.roundNumber)) 局")
                    .font(.system(size: 13, weight: .semibold))
                    .foregroundStyle(palette.textPrimary)
                HStack(spacing: 6) {
                    Text(LiveMatchText.gameTitle(data.matchData.gameType))
                        .font(.system(size: 11))
                        .foregroundStyle(palette.textMuted)
                    if !round.winTypeLabel.isEmpty {
                        Text(round.winTypeLabel)
                            .font(.system(size: 11))
                            .foregroundStyle(palette.brandStrong)
                    }
                }
            }
            Spacer()
            VStack(alignment: .trailing, spacing: 3) {
                Text("\(String(round.player1Score)) - \(String(round.player2Score))")
                    .font(.system(size: 15, weight: .semibold).monospacedDigit())
                    .foregroundStyle(palette.textPrimary)
                Text(resultText)
                    .font(.system(size: 12, weight: .semibold))
                    .foregroundStyle(tone)
            }
        }
        .padding(10)
        .background(palette.subtle, in: RoundedRectangle(cornerRadius: 9))
    }

    // MARK: - 访客引导

    private var loginHint: some View {
        HStack(spacing: 12) {
            Text("用追分记录你的比赛")
                .font(.system(size: 13))
                .foregroundStyle(palette.textSecondary)
            Spacer()
            Button("登录") { navigator.openLogin() }
                .buttonStyle(.appPill)
        }
        .padding(.horizontal, 16)
        .padding(.vertical, 10)
        .background(palette.card)
    }
}

private struct CardBox<Content: View>: View {
    @Environment(\.colorScheme) private var colorScheme
    @ViewBuilder let content: () -> Content

    var body: some View {
        content()
            .frame(maxWidth: .infinity, alignment: .leading)
            .padding(14)
            .background(AppTheme.palette(for: colorScheme).card, in: RoundedRectangle(cornerRadius: 16))
    }
}
