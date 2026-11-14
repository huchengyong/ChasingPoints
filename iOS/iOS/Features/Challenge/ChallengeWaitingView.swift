import SwiftUI

/// 等待对手，对应小程序 `subPages/match/challengeWaiting.vue`。
struct ChallengeWaitingView: View {
    @EnvironmentObject private var session: SessionStore
    @EnvironmentObject private var navigator: AppNavigator
    @EnvironmentObject private var toast: ToastCenter
    @Environment(\.colorScheme) private var colorScheme

    @StateObject private var model: ChallengeWaitingModel

    init(
        session: SessionStore,
        activity: ActivityStore,
        challengeId: Int64,
        model: ChallengeWaitingModel? = nil
    ) {
        _model = StateObject(wrappedValue: model ?? ChallengeWaitingModel(
            session: session,
            activity: activity,
            challengeId: challengeId
        ))
    }

    private var palette: AppPalette { AppTheme.palette(for: colorScheme) }

    var body: some View {
        Group {
            if session.identity == nil {
                StateMessageView(
                    icon: "lock.fill",
                    title: "登录后查看约球",
                    actionTitle: "去登录"
                ) {
                    navigator.openLogin()
                }
            } else {
                content
            }
        }
        .background(palette.page.ignoresSafeArea())
        .navigationTitle("等待对手")
        .navigationBarTitleDisplayMode(.inline)
        .task { await model.start() }
        .onDisappear { model.stop() }
        .onChange(of: session.identity) { _, _ in
            Task { await model.identityChanged() }
        }
        .onChange(of: model.feedback) { _, value in
            guard let value else { return }
            toast.show(value)
            model.feedback = nil
        }
        .onChange(of: model.navigation) { _, value in
            guard let value else { return }
            model.navigation = nil
            switch value {
            case .match(let matchId):
                navigator.open(.liveMatch(matchId: matchId))
            }
        }
        .alert(item: $model.prompt) { prompt in
            alert(for: prompt)
        }
    }

    @ViewBuilder
    private var content: some View {
        switch model.phase {
        case .loading:
            VStack(spacing: 10) {
                ProgressView()
                Text("加载中…")
                    .font(.system(size: 13))
                    .foregroundStyle(palette.textMuted)
            }
            .frame(maxWidth: .infinity, maxHeight: .infinity)
        case .failed:
            StateMessageView(
                icon: "exclamationmark.circle",
                title: "暂时无法确认约球状态",
                actionTitle: "重新加载"
            ) {
                Task { await model.load() }
            }
        case .loaded:
            if let challenge = model.challenge {
                loaded(challenge)
            }
        }
    }

    private func loaded(_ challenge: ChallengeInfo) -> some View {
        ScrollView {
            VStack(spacing: 14) {
                vsRow(challenge)
                Text("\(model.gameTypeLabel) · \(model.matchModeLabel)")
                    .font(.system(size: 13))
                    .foregroundStyle(palette.textSecondary)
                Text(model.scheduleLabel)
                    .font(.system(size: 13))
                    .foregroundStyle(palette.textSecondary)
                if !model.metaHint.isEmpty {
                    Text(model.metaHint)
                        .font(.system(size: 12))
                        .foregroundStyle(model.expiredNow ? AppTheme.danger : palette.textMuted)
                        .multilineTextAlignment(.center)
                }
                actions(challenge)
                    .padding(.top, 6)
            }
            .padding(.horizontal, 20)
            .padding(.vertical, 24)
        }
    }

    private func vsRow(_ challenge: ChallengeInfo) -> some View {
        HStack(spacing: 36) {
            player(
                avatar: challenge.fromAvatar,
                userId: challenge.fromUserId,
                vipLevel: challenge.fromVipLevel ?? 0,
                waiting: model.isWaiting(challenge.fromUserId)
            )
            Text("VS")
                .font(.system(size: 14, weight: .heavy))
                .foregroundStyle(palette.textMuted)
            player(
                avatar: challenge.toAvatar,
                userId: challenge.toUserId,
                vipLevel: challenge.toVipLevel ?? 0,
                waiting: model.isWaiting(challenge.toUserId)
            )
        }
        .frame(maxWidth: .infinity)
        .padding(.vertical, 18)
        .appCard()
    }

    private func player(avatar: String, userId: Int64, vipLevel: Int, waiting: Bool) -> some View {
        VStack(spacing: 8) {
            MemberAvatarView(avatar: avatar, userId: userId, vipLevel: vipLevel, size: 70, reserveFrame: true)
            Text(waiting ? "已进入" : "待进入")
                .font(.system(size: 12, weight: waiting ? .semibold : .regular))
                .foregroundStyle(waiting ? palette.brandStrong : palette.textMuted)
        }
    }

    @ViewBuilder
    private func actions(_ challenge: ChallengeInfo) -> some View {
        VStack(spacing: 10) {
            if challenge.status == 0 && !model.expiredNow {
                if challenge.toUserId == model.myUserId {
                    Button("接受约球") {
                        Task { await model.accept() }
                    }
                    .buttonStyle(.appPrimary)
                    .disabled(model.isActing)
                    Button("这次不了") {
                        Task { await model.reject() }
                    }
                    .buttonStyle(.appSecondary)
                    .disabled(model.isActing)
                } else {
                    Button("取消邀请") {
                        model.prompt = .cancel
                    }
                    .buttonStyle(.appSecondary)
                    .disabled(model.isActing)
                }
            } else if challenge.status == 1 && !model.expiredNow {
                if model.isWaiting(model.myUserId) {
                    Button("退出等待，保留约球") {
                        Task { await model.leave() }
                    }
                    .buttonStyle(.appPrimary)
                    .disabled(model.isActing)
                    Button("取消本次约球") {
                        model.prompt = .cancel
                    }
                    .buttonStyle(.appSecondary)
                    .disabled(model.isActing)
                } else if model.isWaiting(model.opponentUserId) {
                    Button("进入对局") {
                        Task { await model.enter(confirmEarly: true) }
                    }
                    .buttonStyle(.appPrimary)
                    .disabled(model.isActing)
                    Button("放弃本次约球") {
                        model.prompt = .abandon
                    }
                    .buttonStyle(.appSecondary)
                    .disabled(model.isActing)
                } else {
                    Button("进入对局") {
                        Task { await model.enter() }
                    }
                    .buttonStyle(.appPrimary)
                    .disabled(model.isActing)
                    Button("取消本次约球") {
                        model.prompt = .cancel
                    }
                    .buttonStyle(.appSecondary)
                    .disabled(model.isActing)
                }
            } else if challenge.status == 6, let matchId = challenge.matchId, matchId > 0 {
                Button("继续比赛") {
                    navigator.open(.liveMatch(matchId: matchId))
                }
                .buttonStyle(.appPrimary)
            } else {
                Text(model.expiredNow ? "已失效" : model.statusLabel)
                    .font(.system(size: 15, weight: .medium))
                    .foregroundStyle(palette.textSecondary)
                    .padding(.vertical, 10)
            }
        }
    }

    private func alert(for prompt: ChallengeWaitingModel.Prompt) -> Alert {
        switch prompt {
        case .closeOwn:
            return Alert(
                title: Text("接受这个约球？"),
                message: Text("接受后会自动关闭你已发出的邀请。"),
                primaryButton: .default(Text("确认接受")) {
                    Task { await model.accept(confirmCloseOwn: true) }
                },
                secondaryButton: .cancel(Text("取消"))
            )
        case .typeConflict(let message):
            return Alert(
                title: Text("约球类型不同"),
                message: Text(message),
                primaryButton: .default(Text("查看已发出的邀请")) {
                    navigator.open(.challenges(tab: .sent))
                },
                secondaryButton: .cancel(Text("知道了"))
            )
        case .confirmEarly(let schedule):
            return Alert(
                title: Text("现在进入吗？"),
                message: Text("预计\(schedule)开始。双方进入后才会正式开局。"),
                primaryButton: .default(Text("现在进入")) {
                    Task { await model.enter(confirmEarly: true) }
                },
                secondaryButton: .cancel(Text("稍后再打"))
            )
        case .cancel:
            return Alert(
                title: Text("取消本次约球？"),
                message: Text("取消后需要重新邀约才能再打。"),
                primaryButton: .destructive(Text("确定")) {
                    Task { await model.cancel() }
                },
                secondaryButton: .cancel(Text("取消"))
            )
        case .abandon:
            return Alert(
                title: Text("放弃本次约球？"),
                message: Text("对方将退出等待，约球结束。不会产生比赛和战绩。"),
                primaryButton: .destructive(Text("确认放弃")) {
                    Task { await model.abandon() }
                },
                secondaryButton: .cancel(Text("继续约球"))
            )
        case .ongoingMatch(let message, let matchId):
            return Alert(
                title: Text("你有未结束的对局"),
                message: Text(message),
                primaryButton: .default(Text("继续比赛")) {
                    navigator.open(.liveMatch(matchId: matchId))
                },
                secondaryButton: .cancel(Text("稍后处理"))
            )
        }
    }
}
