import SwiftUI

/// 扫码确认页：开局选择球种 / 对局类型，或预览裁判码后确认执裁。
struct ScanConfirmView: View {
    @EnvironmentObject private var session: SessionStore
    @EnvironmentObject private var navigator: AppNavigator
    @EnvironmentObject private var toast: ToastCenter
    @Environment(\.colorScheme) private var colorScheme
    @Environment(\.dismiss) private var dismiss
    @StateObject private var model: ScanConfirmModel
    @State private var isRescanPresented = false

    init(session: SessionStore, model: ScanConfirmModel? = nil) {
        _model = StateObject(wrappedValue: model ?? ScanConfirmModel(
            session: session,
            handoff: QuickScanFlow.shared.takeHandoff(identity: session.identity)
        ))
    }

    private var palette: AppPalette { AppTheme.palette(for: colorScheme) }

    var body: some View {
        ScrollView {
            content
                .padding(.horizontal, 12)
                .padding(.vertical, 12)
        }
        .background(palette.page.ignoresSafeArea())
        .navigationTitle(model.navigationTitle)
        .navigationBarTitleDisplayMode(.inline)
        .task {
            QuickScanFlow.shared.bind(session: session)
            await model.loadIfNeeded()
        }
        .onChange(of: session.identity) { _, _ in
            model.handleIdentityChange()
        }
        .onChange(of: model.liveMatchId) { _, matchId in
            guard let matchId else { return }
            model.consumeNavigation()
            navigator.open(.liveMatch(matchId: matchId))
        }
        .onChange(of: model.toastMessage) { _, message in
            guard let message else { return }
            model.consumeToast()
            toast.show(message)
        }
        .alert("你有未结束的对局", isPresented: selfOngoingBinding) {
            Button("进入该对局") { model.confirmSelfOngoing() }
            Button("稍后处理", role: .cancel) { model.dismissSelfOngoing() }
        } message: {
            Text(model.selfOngoingPrompt?.message ?? "")
        }
        .fullScreenCover(isPresented: $isRescanPresented) {
            QRScannerView { outcome in
                isRescanPresented = false
                Task { await handleRescan(outcome) }
            }
        }
    }

    @ViewBuilder
    private var content: some View {
        switch model.stage {
        case .loading:
            loadingBlock
        case .expired, .error:
            expiredBlock
        case .ready:
            if model.mode == .start {
                startForm
            } else {
                refereeForm
            }
        }
    }

    private var loadingBlock: some View {
        VStack(spacing: 10) {
            ProgressView()
            Text("正在核验裁判码...")
                .font(.system(size: 14))
                .foregroundStyle(palette.textSecondary)
        }
        .frame(maxWidth: .infinity)
        .padding(.top, 100)
    }

    private var expiredBlock: some View {
        VStack(spacing: 16) {
            VStack(spacing: 8) {
                Text("无法继续")
                    .font(.system(size: 17, weight: .bold))
                    .foregroundStyle(palette.textPrimary)
                Text(model.errorMessage)
                    .font(.system(size: 14))
                    .foregroundStyle(AppTheme.danger)
                    .multilineTextAlignment(.center)
            }
            Button("重新扫码") { isRescanPresented = true }
                .buttonStyle(.appPrimary)
            Button("返回") { dismiss() }
                .buttonStyle(.appSecondary)
        }
        .padding(.top, 80)
    }

    private var startForm: some View {
        VStack(spacing: 12) {
            opponentCard

            VStack(alignment: .leading, spacing: 9) {
                Text("球种")
                    .font(.system(size: 13, weight: .semibold))
                    .foregroundStyle(palette.textPrimary)

                ScrollView(.horizontal, showsIndicators: false) {
                    HStack(spacing: 8) {
                        ForEach(GameTypeCatalog.ordered) { item in
                            gameTypeChip(item)
                        }
                    }
                }

                if model.canSetDefault {
                    Button { model.toggleSetAsDefault() } label: {
                        HStack(spacing: 6) {
                            Image(systemName: model.setAsDefault ? "checkmark.square.fill" : "square")
                                .font(.system(size: 16))
                                .foregroundStyle(model.setAsDefault ? AppTheme.primary : palette.textMuted)
                            Text("将当前选择设为默认对局类型")
                                .font(.system(size: 12))
                                .foregroundStyle(palette.textSecondary)
                            Spacer(minLength: 0)
                        }
                    }
                    .buttonStyle(.plain)
                }

                Text("对局类型")
                    .font(.system(size: 13, weight: .semibold))
                    .foregroundStyle(palette.textPrimary)
                    .padding(.top, 3)

                VStack(spacing: 6) {
                    ForEach(Array(StartMatch.modeOptions.enumerated()), id: \.offset) { index, item in
                        modeOption(index: index, item: item)
                    }
                }
            }
            .appCard(padding: 14)

            if !model.errorMessage.isEmpty {
                Text(model.errorMessage)
                    .font(.system(size: 12))
                    .foregroundStyle(AppTheme.danger)
                    .multilineTextAlignment(.center)
                    .frame(maxWidth: .infinity)
            }

            Button {
                Task { await model.start() }
            } label: {
                Text(model.submitting ? "创建中..." : "开始对局")
            }
            .buttonStyle(.appPrimary)
            .disabled(model.gameType <= 0 || model.submitting)

            Button("重新扫码") { isRescanPresented = true }
                .buttonStyle(.appSecondary)
                .disabled(model.submitting)
        }
    }

    private var opponentCard: some View {
        HStack(spacing: 10) {
            AvatarView(avatar: model.opponent.avatar, userId: model.opponent.id, size: 48)
            VStack(alignment: .leading, spacing: 3) {
                Text(model.opponent.nickname.isEmpty ? "对手" : model.opponent.nickname)
                    .font(.system(size: 16, weight: .bold))
                    .foregroundStyle(palette.textPrimary)
                    .lineLimit(1)
                Text("已识别对手，选好球种和类型后开始对局")
                    .font(.system(size: 12))
                    .foregroundStyle(palette.textMuted)
            }
            Spacer(minLength: 0)
        }
        .appCard(padding: 14)
    }

    private func gameTypeChip(_ item: GameTypeOption) -> some View {
        let isActive = model.gameType == item.value
        return Button { model.selectGameType(item.value) } label: {
            HStack(spacing: 4) {
                Text(item.label)
                    .font(.system(size: 13, weight: isActive ? .semibold : .regular))
                if model.defaultGameType == item.value {
                    Text("默认")
                        .font(.system(size: 10))
                        .padding(.horizontal, 5)
                        .padding(.vertical, 1)
                        .background(
                            isActive ? Color.white.opacity(0.28) : palette.brandTint,
                            in: Capsule()
                        )
                        .foregroundStyle(isActive ? .white : palette.brandStrong)
                }
            }
            .foregroundStyle(isActive ? .white : palette.textSecondary)
            .padding(.horizontal, 15)
            .padding(.vertical, 7)
            .background(isActive ? AppTheme.primary : palette.subtle, in: Capsule())
        }
        .buttonStyle(.plain)
    }

    private func modeOption(index: Int, item: StartMatchModeOption) -> some View {
        let isActive = model.modeIndex == index
        return Button { model.selectMode(index) } label: {
            HStack(spacing: 8) {
                Circle()
                    .strokeBorder(isActive ? AppTheme.primary : palette.textMuted, lineWidth: isActive ? 5 : 1)
                    .frame(width: 16, height: 16)
                Text(item.label)
                    .font(.system(size: 13))
                    .foregroundStyle(palette.textPrimary)
                    .multilineTextAlignment(.leading)
                Spacer(minLength: 0)
            }
            .padding(.horizontal, 10)
            .padding(.vertical, 11)
            .background(isActive ? palette.brandTint : palette.subtle, in: RoundedRectangle(cornerRadius: 9))
            .overlay {
                RoundedRectangle(cornerRadius: 9)
                    .stroke(isActive ? AppTheme.primary : palette.border, lineWidth: 1)
            }
        }
        .buttonStyle(.plain)
    }

    private var refereeForm: some View {
        VStack(spacing: 12) {
            if let preview = model.refereePreview {
                VStack(alignment: .leading, spacing: 10) {
                    Text("确认担任本场裁判")
                        .font(.system(size: 17, weight: .bold))
                        .foregroundStyle(palette.textPrimary)
                    Text("确认后，你将负责本场记分和结算，双方选手会切换为只读。")
                        .font(.system(size: 12))
                        .foregroundStyle(palette.textSecondary)
                        .lineSpacing(3)

                    VStack(spacing: 6) {
                        HStack(spacing: 14) {
                            refereePlayer(
                                avatar: preview.player1Avatar,
                                id: preview.player1Id,
                                name: preview.player1Name.isEmpty ? "选手1" : preview.player1Name
                            )
                            Text("VS")
                                .font(.system(size: 11, weight: .bold))
                                .foregroundStyle(palette.textMuted)
                            refereePlayer(
                                avatar: preview.player2Avatar,
                                id: preview.player2Id,
                                name: preview.player2Name.isEmpty ? "选手2" : preview.player2Name
                            )
                        }
                        .padding(.vertical, 14)

                        Text("\(String(preview.player1Score)) : \(String(preview.player2Score))")
                            .font(.system(size: 26, weight: .heavy))
                            .foregroundStyle(palette.brandStrong)
                        Text("\(preview.gameTypeName.isEmpty ? "对局" : preview.gameTypeName) · \(model.refereeModeText)")
                            .font(.system(size: 12))
                            .foregroundStyle(palette.textSecondary)
                    }
                    .frame(maxWidth: .infinity)
                    .background(palette.subtle, in: RoundedRectangle(cornerRadius: 12))
                }
                .appCard()
            }

            Button {
                Task { await model.join() }
            } label: {
                Text(model.joining ? "确认中..." : "确认担任裁判")
            }
            .buttonStyle(.appPrimary)
            .disabled(model.joining)

            Button("暂不担任") { dismiss() }
                .buttonStyle(.appSecondary)
                .disabled(model.joining)
        }
    }

    private func refereePlayer(avatar: String, id: Int64, name: String) -> some View {
        VStack(spacing: 5) {
            AvatarView(avatar: avatar, userId: id, size: 43)
            Text(name)
                .font(.system(size: 12, weight: .semibold))
                .foregroundStyle(palette.textPrimary)
                .lineLimit(1)
        }
        .frame(maxWidth: .infinity)
    }

    private var selfOngoingBinding: Binding<Bool> {
        Binding(
            get: { model.selfOngoingPrompt != nil },
            set: { if !$0 { model.dismissSelfOngoing() } }
        )
    }

    /// 重新扫码拿到新的交接后就地重载，避免重复压栈。
    private func handleRescan(_ outcome: ScanOutcome) async {
        guard let result = try? await QuickScanFlow.shared.start(rematch: model.rematch, scan: { outcome }) else {
            return
        }
        switch result {
        case .confirm:
            model.apply(handoff: QuickScanFlow.shared.takeHandoff(identity: session.identity))
            await model.loadIfNeeded()
        case .login:
            navigator.openLogin()
        case .error(let message):
            toast.show(message ?? "扫码失败")
        case .mismatch:
            toast.show("请扫描指定对手的匹配码")
        case .busy, .cancelled, .stale, .spectate:
            break
        }
    }
}
