import SwiftUI

/// 发起约球，对应小程序 `subPages/match/challengeCompose.vue`。
struct ChallengeComposeView: View {
    @EnvironmentObject private var session: SessionStore
    @EnvironmentObject private var navigator: AppNavigator
    @EnvironmentObject private var toast: ToastCenter
    @Environment(\.colorScheme) private var colorScheme

    @StateObject private var model: ChallengeComposeModel
    @State private var pendingConflict: PendingConflict?
    @State private var isSelectingOpponent = false

    private struct PendingConflict: Identifiable {
        let message: String
        let challengeId: Int64

        var id: String { "\(String(challengeId))-\(message)" }
    }

    init(session: SessionStore, activity: ActivityStore, target: ChallengeComposeTarget?, model: ChallengeComposeModel? = nil) {
        _model = StateObject(wrappedValue: model ?? ChallengeComposeModel(session: session, activity: activity, target: target))
    }

    private var palette: AppPalette { AppTheme.palette(for: colorScheme) }

    var body: some View {
        ScrollView {
            VStack(spacing: 12) {
                opponentCard
                formCard
                submitButton
            }
            .padding(.horizontal, 16)
            .padding(.vertical, 14)
        }
        .background(palette.page.ignoresSafeArea())
        .navigationTitle("发起约球")
        .navigationBarTitleDisplayMode(.inline)
        .task { await model.onAppear() }
        .navigationDestination(isPresented: $isSelectingOpponent) {
            OpponentSelectorView(session: session) { candidate in
                model.replaceOpponent(ChallengeComposeTarget(
                    userId: candidate.userId,
                    nickname: candidate.name,
                    avatar: candidate.avatar
                ))
                isSelectingOpponent = false
            }
        }
        .alert(item: $pendingConflict) { item in
            Alert(
                title: Text("约球未发送"),
                message: Text(item.message),
                primaryButton: .default(Text("查看并取消")) {
                    navigator.open(.challengeWaiting(challengeId: item.challengeId))
                },
                secondaryButton: .cancel(Text("稍后再说"))
            )
        }
    }

    // MARK: - 对手

    @ViewBuilder
    private var opponentCard: some View {
        if let opponent = model.opponent, opponent.userId > 0 {
            HStack(spacing: 12) {
                AvatarView(avatar: model.opponentAvatar, userId: opponent.userId, size: 46)
                VStack(alignment: .leading, spacing: 3) {
                    Text(model.opponentName)
                        .font(.system(size: 16, weight: .semibold))
                        .foregroundStyle(palette.textPrimary)
                        .lineLimit(1)
                    Text("约好后双方各点一次进入，直接开局")
                        .font(.system(size: 12))
                        .foregroundStyle(palette.textMuted)
                }
                Spacer(minLength: 8)
                Button("更换") {
                    isSelectingOpponent = true
                }
                .font(.system(size: 13, weight: .semibold))
                .foregroundStyle(palette.brandStrong)
                .buttonStyle(.plain)
            }
            .appCard()
        } else {
            Button {
                isSelectingOpponent = true
            } label: {
                VStack(alignment: .leading, spacing: 4) {
                    Text("选择球友")
                        .font(.system(size: 16, weight: .semibold))
                        .foregroundStyle(palette.textPrimary)
                    Text("好友和最近对手都可以约")
                        .font(.system(size: 12))
                        .foregroundStyle(palette.textMuted)
                }
                .frame(maxWidth: .infinity, alignment: .leading)
                .contentShape(Rectangle())
            }
            .buttonStyle(.plain)
            .appCard()
        }
    }

    // MARK: - 表单

    private var formCard: some View {
        VStack(alignment: .leading, spacing: 10) {
            label("打什么")
            chipRow {
                ForEach(GameTypeCatalog.ordered) { option in
                    chip(option.label, active: model.gameType == option.value) {
                        model.gameType = option.value
                    }
                }
            }

            label("比赛类型")
            chipRow {
                chip("排位", active: model.matchMode == "ranked") {
                    model.matchMode = "ranked"
                }
                chip("练习", active: model.matchMode == "practice") {
                    model.matchMode = "practice"
                }
            }
            hint(model.matchModeHint)

            label("预计什么时候")
            chipRow {
                ForEach(model.dayOptions) { option in
                    chip(option.label, active: model.activeDayOffset == option.value) {
                        model.chooseDay(option.value)
                    }
                }
            }
            HStack(spacing: 10) {
                hourMenu(model.startHourChoices, current: model.startHour) { model.setStartHour($0) }
                Text("到")
                    .font(.system(size: 13))
                    .foregroundStyle(palette.textSecondary)
                hourMenu(model.endHourChoices, current: model.endHour) { model.setEndHour($0) }
            }
            hint("预计时间仅作提醒，明天早晨7点失效")

            label("附言（选填）")
            TextField("给对方捎句话", text: $model.message, axis: .vertical)
                .font(.system(size: 14))
                .lineLimit(1...4)
                .padding(.horizontal, 12)
                .padding(.vertical, 10)
                .background(palette.subtle, in: RoundedRectangle(cornerRadius: 10))
                .overlay {
                    RoundedRectangle(cornerRadius: 10).stroke(palette.border)
                }
            Text("\(String(model.message.count))/\(String(ChallengeComposeModel.maxMessageLength))")
                .font(.system(size: 11))
                .foregroundStyle(palette.textMuted)
                .frame(maxWidth: .infinity, alignment: .trailing)
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .appCard()
    }

    private var submitButton: some View {
        Button {
            Task { await submit() }
        } label: {
            Text(model.isSubmitting ? "发送中…" : "发送约球")
        }
        .buttonStyle(.appPrimary)
        .disabled(!model.canSubmit)
    }

    private func label(_ text: String) -> some View {
        Text(text)
            .font(.system(size: 13, weight: .semibold))
            .foregroundStyle(palette.textPrimary)
            .padding(.top, 2)
    }

    private func hint(_ text: String) -> some View {
        Text(text)
            .font(.system(size: 11))
            .foregroundStyle(palette.textMuted)
            .fixedSize(horizontal: false, vertical: true)
    }

    private func chipRow<Content: View>(@ViewBuilder content: () -> Content) -> some View {
        HStack(spacing: 8) {
            content()
            Spacer(minLength: 0)
        }
    }

    private func chip(_ title: String, active: Bool, action: @escaping () -> Void) -> some View {
        Button(action: action) {
            Text(title)
                .font(.system(size: 13, weight: active ? .semibold : .regular))
                .foregroundStyle(active ? palette.brandStrong : palette.textSecondary)
                .padding(.horizontal, 12)
                .frame(height: 32)
                .background(active ? palette.brandTint : palette.subtle, in: Capsule())
                .overlay {
                    Capsule().stroke(active ? AppTheme.primary.opacity(0.5) : Color.clear)
                }
        }
        .buttonStyle(.plain)
    }

    private func hourMenu(_ choices: [Int], current: Int, select: @escaping (Int) -> Void) -> some View {
        Menu {
            ForEach(choices, id: \.self) { hour in
                Button(ChallengeRules.hourLabel(hour)) { select(hour) }
            }
        } label: {
            HStack(spacing: 4) {
                Text(ChallengeRules.hourLabel(current))
                    .font(.system(size: 14, weight: .medium))
                    .foregroundStyle(palette.textPrimary)
                Image(systemName: "chevron.down")
                    .font(.system(size: 10, weight: .semibold))
                    .foregroundStyle(palette.textMuted)
            }
            .padding(.horizontal, 14)
            .frame(height: 36)
            .background(palette.subtle, in: RoundedRectangle(cornerRadius: 9))
            .overlay {
                RoundedRectangle(cornerRadius: 9).stroke(palette.border)
            }
        }
    }

    // MARK: - 提交

    private func submit() async {
        switch await model.submit() {
        case .success:
            toast.show("约球已发送")
            navigator.select(.home, isSignedIn: session.identity != nil)
            navigator.popToRoot(.home)
        case .pendingConflict(let message, let challengeId):
            if let challengeId {
                pendingConflict = PendingConflict(message: message, challengeId: challengeId)
            } else {
                // 拿不到原约球 ID 时无法定位详情，按小程序的无 ID 分支只提示。
                toast.show(message)
            }
        case .failure(let message):
            toast.show(message)
        case .discarded:
            break
        }
    }
}
