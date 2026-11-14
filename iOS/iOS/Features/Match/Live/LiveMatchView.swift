import SwiftUI
import UIKit

/// 实时计分页：对应小程序 `subPages/match/playing.vue`（本轮含中式八球、美式九球、观赛与结束流程）。
struct LiveMatchView: View {
    @EnvironmentObject private var session: SessionStore
    @EnvironmentObject private var navigator: AppNavigator
    @EnvironmentObject private var activity: ActivityStore
    @EnvironmentObject private var toast: ToastCenter
    @Environment(\.dismiss) private var dismiss
    @Environment(\.colorScheme) private var colorScheme
    @StateObject private var model: LiveMatchModel

    init(session: SessionStore, matchId: Int64, entry: LiveMatchEntry = .automatic, model: LiveMatchModel? = nil) {
        _model = StateObject(wrappedValue: model ?? LiveMatchModel(session: session, matchId: matchId, entry: entry))
    }

    private var palette: AppPalette { AppTheme.palette(for: colorScheme) }

    var body: some View {
        ZStack {
            palette.page.ignoresSafeArea()
            VStack(spacing: 0) {
                header
                content
                footer
            }
            if model.showFinishConfirmDialog {
                finishConfirmDialog
            }
            if model.isRefereeQRCodePresented {
                refereeQRCodeDialog
            }
        }
        .toolbar(.hidden, for: .navigationBar)
        .toolbar(.hidden, for: .tabBar)
        .task {
            model.setSettlementHandlers(
                onMatchSettled: { activity.markDirty() },
                onRankInvalidated: { activity.bumpUserDataRevision() }
            )
            UIApplication.shared.isIdleTimerDisabled = true
            await model.start()
            while !Task.isCancelled {
                model.tick()
                try? await Task.sleep(for: .seconds(1))
            }
        }
        .onAppear { model.setVisible(true) }
        .onDisappear {
            model.stop()
            UIApplication.shared.isIdleTimerDisabled = false
        }
        .onChange(of: session.identity) { _, _ in model.handleIdentityChange() }
        .onChange(of: model.toastMessage) { _, message in
            guard let message else { return }
            model.consumeToast()
            toast.show(message)
        }
        .onChange(of: model.shouldDismiss) { _, dismissRequested in
            guard dismissRequested else { return }
            model.consumeDismiss()
            dismiss()
        }
        .onChange(of: model.destination) { _, destination in
            guard let destination else { return }
            model.consumeDestination()
            dismiss()
            // 等 dismiss 完成后再压入目的地，避免同一次导航更新互相覆盖。
            DispatchQueue.main.async {
                switch destination {
                case .result(let matchId):
                    navigator.open(.matchReport(matchId: matchId))
                case .matchHistory:
                    navigator.open(.matchHistory)
                }
            }
        }
        .alert(
            model.confirm?.title ?? "",
            isPresented: Binding(
                get: { model.confirm != nil },
                set: { if !$0 { model.cancelConfirm() } }
            ),
            presenting: model.confirm
        ) { confirm in
            Button(confirm.cancelTitle, role: .cancel) { model.cancelConfirm() }
            Button(confirm.confirmTitle, role: .destructive) { model.acceptConfirm() }
        } message: { confirm in
            Text(confirm.message)
        }
    }

    // MARK: - 顶部

    private var header: some View {
        HStack(spacing: 8) {
            Button {
                model.requestBack()
            } label: {
                Image(systemName: "chevron.left")
                    .font(.system(size: 17, weight: .semibold))
                    .foregroundStyle(palette.textPrimary)
                    .frame(width: 40, height: 40)
            }
            .buttonStyle(.plain)
            .accessibilityLabel("返回")

            Spacer(minLength: 4)

            VStack(spacing: 2) {
                Text(model.gameTypeName)
                    .font(.system(size: 16, weight: .bold))
                    .foregroundStyle(palette.textPrimary)
                Text(model.scoreboardSubtitle)
                    .font(.system(size: 11))
                    .foregroundStyle(palette.textSecondary)
                    .lineLimit(1)
            }

            Spacer(minLength: 4)

            Menu {
                ForEach(model.menuOptions) { option in
                    Button(option.title) { model.selectMenu(option) }
                }
            } label: {
                Image(systemName: "ellipsis")
                    .font(.system(size: 17, weight: .semibold))
                    .foregroundStyle(palette.textPrimary)
                    .frame(width: 40, height: 40)
            }
            .accessibilityLabel("对局菜单")
        }
        .padding(.horizontal, 4)
        .padding(.top, 4)
    }

    // MARK: - 主体

    @ViewBuilder
    private var content: some View {
        switch model.phase {
        case .loading:
            VStack(spacing: 10) {
                ProgressView()
                Text("正在加载对局...")
                    .font(.system(size: 14))
                    .foregroundStyle(palette.textSecondary)
            }
            .frame(maxWidth: .infinity, maxHeight: .infinity)
        case .failed(let message):
            StateMessageView(
                icon: "exclamationmark.triangle",
                title: "无法加载对局",
                detail: message,
                actionTitle: "重试"
            ) {
                Task { await model.retry() }
            }
        case .ready:
            ScrollView {
                VStack(spacing: 12) {
                    scoreboard
                    if model.isPoolFormatVisible {
                        poolFormatOverview
                    }
                    actionArea
                }
                .padding(.horizontal, 12)
                .padding(.vertical, 8)
            }
        }
    }

    private var scoreboard: some View {
        VStack(spacing: 10) {
            HStack(alignment: .top, spacing: 8) {
                playerColumn(
                    info: model.myInfo,
                    identity: model.viewerUi.leftIdentity,
                    score: model.myScore,
                    isLeft: true
                )
                Text(":")
                    .font(.system(size: 22, weight: .bold))
                    .foregroundStyle(palette.textMuted)
                    .padding(.top, 24)
                playerColumn(
                    info: model.opponentInfo,
                    identity: model.viewerUi.rightIdentity,
                    score: model.opponentScore,
                    isLeft: false
                )
            }

            HStack(spacing: 6) {
                if model.gameType != 2 {
                    Text(model.currentRoundText)
                    Text("·")
                }
                Text(model.syncStatusText)
                    .foregroundStyle(model.isSyncing ? AppTheme.warning : palette.textSecondary)
            }
            .font(.system(size: 12))
            .foregroundStyle(palette.textSecondary)

            viewerBanner

            if let description = model.lastAction?.description, !description.isEmpty {
                HStack(spacing: 6) {
                    Text("最近操作")
                        .font(.system(size: 11, weight: .semibold))
                        .foregroundStyle(palette.textMuted)
                    Text(description)
                        .font(.system(size: 12))
                        .foregroundStyle(palette.textSecondary)
                        .lineLimit(2)
                    Spacer(minLength: 0)
                }
            }

            if model.showFinishConfirmPanel {
                finishConfirmPanel
            }

            if model.gameType == 1 {
                HStack {
                    Text("当前局比分")
                        .font(.system(size: 12, weight: .semibold))
                        .foregroundStyle(palette.textSecondary)
                    Text(model.currentFrameStarted ? "进行中" : "待开始")
                        .font(.system(size: 11))
                        .foregroundStyle(palette.textMuted)
                    Spacer()
                    Text("\(String(model.currentFrameMyScore)) : \(String(model.currentFrameOpponentScore))")
                        .font(.system(size: 15, weight: .bold))
                        .foregroundStyle(palette.textPrimary)
                }
            }
        }
        .padding(14)
        .background(palette.card, in: RoundedRectangle(cornerRadius: 16))
        .overlay { RoundedRectangle(cornerRadius: 16).stroke(palette.border) }
    }

    private func playerColumn(
        info: LiveMatchPlayerInfo,
        identity: String,
        score: Int,
        isLeft: Bool
    ) -> some View {
        VStack(alignment: isLeft ? .leading : .trailing, spacing: 6) {
            HStack(spacing: 8) {
                if isLeft {
                    MemberAvatarView(
                        avatar: info.avatar,
                        userId: info.userId,
                        vipLevel: info.vipLevel,
                        rankLevel: info.rankLevel,
                        size: 52,
                        reserveFrame: true
                    )
                }
                VStack(alignment: isLeft ? .leading : .trailing, spacing: 2) {
                    Text(identity)
                        .font(.system(size: 10, weight: .semibold))
                        .foregroundStyle(palette.brandStrong)
                    Text(info.nickname.isEmpty ? (isLeft ? "我" : "对手") : info.nickname)
                        .font(.system(size: 13, weight: .semibold))
                        .foregroundStyle(palette.textPrimary)
                        .lineLimit(1)
                    Text("胜率 \(String(info.winRate))%")
                        .font(.system(size: 11))
                        .foregroundStyle(palette.textMuted)
                    if model.gameType == 1 {
                        Text("单杆 \(String(info.maxScore))")
                            .font(.system(size: 11))
                            .foregroundStyle(palette.textMuted)
                    }
                }
                if !isLeft {
                    MemberAvatarView(
                        avatar: info.avatar,
                        userId: info.userId,
                        vipLevel: info.vipLevel,
                        rankLevel: info.rankLevel,
                        size: 52,
                        reserveFrame: true
                    )
                }
            }
            Text(String(score))
                .font(.system(size: 30, weight: .heavy, design: .rounded))
                .foregroundStyle(palette.textPrimary)
        }
        .frame(maxWidth: .infinity, alignment: isLeft ? .leading : .trailing)
    }

    private var viewerBanner: some View {
        VStack(alignment: .leading, spacing: 6) {
            HStack(spacing: 8) {
                Text(model.viewerUi.roleLabel)
                    .font(.system(size: 11, weight: .semibold))
                    .foregroundStyle(palette.brandStrong)
                if model.refereeBound {
                    MemberAvatarView(
                        avatar: model.refereeAvatar,
                        userId: model.refereeUserId,
                        size: 20,
                        showBadge: false
                    )
                    Text("本场裁判：\(model.refereeName.isEmpty ? "裁判" : model.refereeName)")
                        .font(.system(size: 11))
                        .foregroundStyle(palette.textSecondary)
                }
                Spacer(minLength: 0)
            }
            if !model.viewerUi.readonlyHint.isEmpty {
                Text(model.viewerUi.readonlyHint)
                    .font(.system(size: 11))
                    .foregroundStyle(palette.textSecondary)
            }
            if model.showInviteRefereeAction {
                Button("邀请裁判扫码") {
                    Task { await model.openRefereeQRCode() }
                }
                .buttonStyle(.appPill)
            }
        }
        .frame(maxWidth: .infinity, alignment: .leading)
    }

    private var finishConfirmPanel: some View {
        let view = model.finishConfirmView
        return VStack(alignment: .leading, spacing: 6) {
            HStack(spacing: 4) {
                Text(view.title)
                    .font(.system(size: 13, weight: .semibold))
                    .foregroundStyle(palette.textPrimary)
                if !view.finishing {
                    Text("· \(String(view.secondsLeft)) 秒")
                        .font(.system(size: 12))
                        .foregroundStyle(AppTheme.warning)
                }
            }
            Text(view.desc)
                .font(.system(size: 12))
                .foregroundStyle(palette.textSecondary)
            if view.showWithdraw {
                Button("撤销结束") {
                    Task { await model.withdrawFinish() }
                }
                .buttonStyle(.appSecondary)
            }
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .padding(10)
        .background(palette.subtle, in: RoundedRectangle(cornerRadius: 12))
    }

    private var poolFormatOverview: some View {
        VStack(alignment: .leading, spacing: 8) {
            HStack {
                Text("对局赛制")
                    .font(.system(size: 12, weight: .semibold))
                    .foregroundStyle(palette.textSecondary)
                Spacer()
                Button {
                    model.toggleFormatEditor()
                } label: {
                    HStack(spacing: 2) {
                        Text(model.matchFormatLabel)
                            .font(.system(size: 13, weight: .semibold))
                        if model.canChangeMatchFormat {
                            Image(systemName: "chevron.right")
                                .font(.system(size: 11, weight: .semibold))
                        }
                    }
                    .foregroundStyle(palette.brandStrong)
                }
                .buttonStyle(.plain)
                .disabled(!model.canChangeMatchFormat)
            }
            Text(model.matchFormatHint)
                .font(.system(size: 11))
                .foregroundStyle(palette.textMuted)
            if model.isFormatEditorShown {
                formatEditor
            }
        }
        .padding(14)
        .background(palette.card, in: RoundedRectangle(cornerRadius: 16))
        .overlay { RoundedRectangle(cornerRadius: 16).stroke(palette.border) }
    }

    private var formatEditor: some View {
        VStack(alignment: .leading, spacing: 8) {
            HStack(spacing: 8) {
                formatModeButton("自由局数", format: PoolMatchFormat.free)
                formatModeButton("抢 N 局制", format: PoolMatchFormat.raceTo)
            }
            if model.draftMatchFormat == PoolMatchFormat.raceTo {
                Stepper(
                    "抢 \(String(model.draftTargetWins)) 局制",
                    value: Binding(
                        get: { model.draftTargetWins },
                        set: { model.setDraftTargetWins($0) }
                    ),
                    in: 1...PoolMatchFormat.maxTargetWins
                )
                .font(.system(size: 13))
                .foregroundStyle(palette.textPrimary)
            }
            Text(model.draftMatchFormatHint)
                .font(.system(size: 11))
                .foregroundStyle(palette.textMuted)
            if !model.formatError.isEmpty {
                Text(model.formatError)
                    .font(.system(size: 11))
                    .foregroundStyle(AppTheme.danger)
            }
            Button(model.isFormatSaving ? "保存中..." : "保存赛制") {
                Task { await model.saveMatchFormat() }
            }
            .buttonStyle(.appPrimary)
            .disabled(model.isFormatSaving)
        }
    }

    private func formatModeButton(_ title: String, format: String) -> some View {
        let active = model.draftMatchFormat == format
        return Button(title) {
            model.selectFormatMode(format)
        }
        .font(.system(size: 13, weight: .semibold))
        .foregroundStyle(active ? .white : palette.textSecondary)
        .frame(maxWidth: .infinity)
        .frame(height: 36)
        .background(active ? AppTheme.primary : palette.subtle, in: RoundedRectangle(cornerRadius: 10))
        .buttonStyle(.plain)
    }

    @ViewBuilder
    private var actionArea: some View {
        if model.viewerUi.showActionPanel {
            VStack(alignment: .leading, spacing: 10) {
                VStack(alignment: .leading, spacing: 3) {
                    Text("记分操作")
                        .font(.system(size: 14, weight: .bold))
                        .foregroundStyle(palette.textPrimary)
                    Text(model.actionPanelDescription)
                        .font(.system(size: 11))
                        .foregroundStyle(palette.textSecondary)
                }

                if model.isRoundWinMode {
                    if model.isReferee {
                        ForEach(model.refereeRoundWinGroups) { group in
                            VStack(alignment: .leading, spacing: 6) {
                                Text("\(group.label)获胜")
                                    .font(.system(size: 12, weight: .semibold))
                                    .foregroundStyle(palette.textSecondary)
                                actionGrid(group.options, winnerActor: group.actor)
                            }
                        }
                    } else {
                        actionGrid(model.roundWinActionOptions, winnerActor: 2)
                    }
                } else {
                    Text("该球种的计分面板正在迁移")
                        .font(.system(size: 13))
                        .foregroundStyle(palette.textMuted)
                        .frame(maxWidth: .infinity)
                        .padding(.vertical, 18)
                }
            }
            .padding(14)
            .frame(maxWidth: .infinity, alignment: .leading)
            .background(palette.card, in: RoundedRectangle(cornerRadius: 16))
            .overlay { RoundedRectangle(cornerRadius: 16).stroke(palette.border) }
        } else {
            VStack(alignment: .leading, spacing: 6) {
                Text(model.readonlyTitle)
                    .font(.system(size: 14, weight: .bold))
                    .foregroundStyle(palette.textPrimary)
                Text(model.readonlyDescription)
                    .font(.system(size: 12))
                    .foregroundStyle(palette.textSecondary)
            }
            .frame(maxWidth: .infinity, alignment: .leading)
            .padding(14)
            .background(palette.card, in: RoundedRectangle(cornerRadius: 16))
            .overlay { RoundedRectangle(cornerRadius: 16).stroke(palette.border) }
        }
    }

    private func actionGrid(_ options: [ScoringActionOption], winnerActor: Int) -> some View {
        LazyVGrid(columns: [GridItem(.flexible()), GridItem(.flexible()), GridItem(.flexible())], spacing: 8) {
            ForEach(options, id: \.type) { option in
                Button {
                    Task { await model.addRoundWin(winType: option.type, score: option.score, winnerActor: winnerActor) }
                } label: {
                    VStack(spacing: 3) {
                        Text(option.title)
                            .font(.system(size: 13, weight: .semibold))
                            .foregroundStyle(palette.textPrimary)
                        Text(option.desc)
                            .font(.system(size: 10))
                            .foregroundStyle(palette.textMuted)
                            .multilineTextAlignment(.center)
                    }
                    .frame(maxWidth: .infinity)
                    .padding(.vertical, 10)
                    .background(palette.subtle, in: RoundedRectangle(cornerRadius: 10))
                    .overlay { RoundedRectangle(cornerRadius: 10).stroke(palette.border) }
                }
                .buttonStyle(.plain)
            }
        }
    }

    // MARK: - 底部

    private var footer: some View {
        VStack(spacing: 8) {
            if model.gameType == 1 && model.viewerUi.showActionPanel && (!model.isSnookerV2 || !model.currentFrameStarted) {
                Button("开始下一局") { model.requestNextRound() }
                    .buttonStyle(.appSecondary)
            }

            if model.isChallengeMatchFinishSlider {
                SlideToConfirmView(
                    label: "向右滑动结束对局 ››",
                    accessibilityLabelText: "向右滑动结束对局"
                ) {
                    Task { await model.commitSlide() }
                }
                Text(model.finishSlideNote)
                    .font(.system(size: 11))
                    .foregroundStyle(palette.textMuted)
                    .multilineTextAlignment(.center)
            } else {
                if model.viewerUi.showFinishButton {
                    Button(model.finishButtonLabel) { model.finishMatch() }
                        .buttonStyle(.appPrimary)
                }
                if model.viewerUi.showFinishRequestButton == true {
                    Button(model.finishRequestButtonLabel) { model.requestFinish() }
                        .buttonStyle(.appPrimary)
                }
            }

            if model.viewerUi.showUndoButton {
                Button("撤销") {
                    Task { await model.undo() }
                }
                .buttonStyle(.plain)
                .font(.system(size: 13, weight: .semibold))
                .foregroundStyle(palette.textSecondary)
            }
        }
        .padding(.horizontal, 12)
        .padding(.top, 8)
        .padding(.bottom, 8)
    }

    // MARK: - 弹层

    private var finishConfirmDialog: some View {
        let view = model.finishConfirmView
        return ZStack {
            Color.black.opacity(0.45).ignoresSafeArea()
                .onTapGesture {}
            VStack(spacing: 12) {
                Text(view.title)
                    .font(.system(size: 16, weight: .bold))
                    .foregroundStyle(palette.textPrimary)
                Text(view.desc)
                    .font(.system(size: 13))
                    .foregroundStyle(palette.textSecondary)
                    .multilineTextAlignment(.center)
                if view.showConfirm {
                    SlideToConfirmView(
                        label: "向右滑动确认结束 ››",
                        accessibilityLabelText: "滑动确认结束对局"
                    ) {
                        Task { await model.commitSlide() }
                    }
                }
                if view.showDispute {
                    Button("有异议，继续比赛") {
                        Task { await model.disputeFinish() }
                    }
                    .font(.system(size: 13, weight: .semibold))
                    .foregroundStyle(AppTheme.danger)
                    .buttonStyle(.plain)
                }
            }
            .padding(18)
            .frame(maxWidth: 320)
            .background(palette.card, in: RoundedRectangle(cornerRadius: 16))
        }
        .accessibilityAddTraits(.isModal)
    }

    private var refereeQRCodeDialog: some View {
        ZStack {
            Color.black.opacity(0.45).ignoresSafeArea()
                .onTapGesture { model.closeRefereeQRCode() }
            VStack(spacing: 12) {
                Text("邀请裁判扫码")
                    .font(.system(size: 16, weight: .bold))
                    .foregroundStyle(palette.textPrimary)
                Text("让球童、助教或第三方扫码后接管本场记分")
                    .font(.system(size: 12))
                    .foregroundStyle(palette.textSecondary)
                    .multilineTextAlignment(.center)

                if model.isRefereeQRCodeLoading {
                    ProgressView()
                    Text("生成中...")
                        .font(.system(size: 13))
                        .foregroundStyle(palette.textSecondary)
                } else if let image = model.refereeQRCodeImage {
                    // 二维码正文是凭证，只用于渲染，不进日志。
                    Image(uiImage: image)
                        .interpolation(.none)
                        .resizable()
                        .frame(width: 220, height: 220)
                        .padding(8)
                        .background(Color.white, in: RoundedRectangle(cornerRadius: 8))
                } else if !model.refereeQRCodeError.isEmpty {
                    Text(model.refereeQRCodeError)
                        .font(.system(size: 13))
                        .foregroundStyle(AppTheme.danger)
                    Button("刷新二维码") {
                        Task { await model.refreshRefereeQRCode() }
                    }
                    .buttonStyle(.appSecondary)
                }

                Text("裁判加入后，选手端会自动切换为只读比分视图。")
                    .font(.system(size: 11))
                    .foregroundStyle(palette.textMuted)
                    .multilineTextAlignment(.center)
                Button("关闭") { model.closeRefereeQRCode() }
                    .font(.system(size: 13, weight: .semibold))
                    .foregroundStyle(palette.brandStrong)
                    .buttonStyle(.plain)
            }
            .padding(18)
            .frame(maxWidth: 320)
            .background(palette.card, in: RoundedRectangle(cornerRadius: 16))
        }
        .accessibilityAddTraits(.isModal)
    }
}
