import SwiftUI

struct ContentView: View {
    @AppStorage("theme_mode") private var themeMode = "system"

    var body: some View {
        MainTabs()
            .preferredColorScheme(ThemeMode(rawValue: themeMode)?.colorScheme)
    }
}

private struct MainTabs: View {
    @Environment(\.colorScheme) private var colorScheme
    @State private var selection = 0

    private var palette: AppPalette { AppTheme.palette(for: colorScheme) }

    var body: some View {
        TabView(selection: $selection) {
            NavigationStack {
                FeaturePlaceholder(title: "首页", symbol: "house.fill")
                    .navigationTitle("追分")
            }
            .tabItem {
                Image(selection == 0 ? "TabHomeSelected" : "TabHome")
                    .renderingMode(.original)
                Text("首页")
            }
            .tag(0)

            NavigationStack {
                FeaturePlaceholder(title: "观赛", symbol: "play.rectangle.fill")
                    .navigationTitle("观赛")
            }
            .tabItem {
                Image(selection == 1 ? "TabWatchSelected" : "TabWatch")
                    .renderingMode(.original)
                Text("观赛")
            }
            .tag(1)

            NavigationStack {
                FeaturePlaceholder(title: "赛讯", symbol: "newspaper.fill")
                    .navigationTitle("赛讯")
            }
            .tabItem {
                Image(selection == 2 ? "TabNewsSelected" : "TabNews")
                    .renderingMode(.original)
                Text("赛讯")
            }
            .tag(2)

            NavigationStack {
                MyPage()
            }
            .tabItem {
                Image(selection == 3 ? "TabMeSelected" : "TabMe")
                    .renderingMode(.original)
                Text("我的")
            }
            .tag(3)
        }
        .tint(AppTheme.primary)
        .toolbarBackground(palette.card, for: .tabBar)
        .toolbarBackground(.visible, for: .tabBar)
    }
}

private struct FeaturePlaceholder: View {
    @Environment(\.colorScheme) private var colorScheme

    let title: String
    let symbol: String

    private var palette: AppPalette { AppTheme.palette(for: colorScheme) }

    var body: some View {
        VStack(spacing: 16) {
            Image(systemName: symbol)
                .font(.system(size: 44))
                .foregroundStyle(AppTheme.primary)
            Text("\(title)页面正在迁移")
                .font(.title3.bold())
                .foregroundStyle(palette.textPrimary)
            Text("此功能尚未在原生版开放")
                .font(.subheadline)
                .foregroundStyle(palette.textSecondary)
        }
        .frame(maxWidth: .infinity, maxHeight: .infinity)
        .background(palette.page.ignoresSafeArea())
    }
}

private struct MyPage: View {
    @EnvironmentObject private var session: SessionStore
    @Environment(\.colorScheme) private var colorScheme
    @State private var showLogin = false

    private var palette: AppPalette { AppTheme.palette(for: colorScheme) }

    var body: some View {
        ScrollView {
            VStack(spacing: 20) {
                if session.isRestoring {
                    ProgressView("正在恢复登录状态")
                        .frame(maxWidth: .infinity)
                        .padding(24)
                } else if let user = session.user {
                    ProfileCard(user: user)
                } else {
                    VStack(alignment: .leading, spacing: 16) {
                        Text("登录追分")
                            .font(.title3.bold())
                            .foregroundStyle(palette.textPrimary)
                        Text("登录后查看个人资料与对局记录")
                            .font(.subheadline)
                            .foregroundStyle(palette.textSecondary)
                        NavigationLink {
                            LoginView()
                        } label: {
                            Text("去登录")
                                .fontWeight(.semibold)
                                .frame(maxWidth: .infinity)
                                .padding(.vertical, 12)
                                .foregroundStyle(.white)
                                .background(AppTheme.primary, in: RoundedRectangle(cornerRadius: 10))
                        }
                    }
                    .frame(maxWidth: .infinity, alignment: .leading)
                    .padding(20)
                    .background(palette.card, in: RoundedRectangle(cornerRadius: 16))
                }

                if let restoreError = session.restoreError {
                    HStack {
                        Text(restoreError)
                            .font(.subheadline)
                            .foregroundStyle(palette.textSecondary)
                        Spacer()
                        Button("重试") {
                            Task { await session.retryRestore() }
                        }
                    }
                    .frame(maxWidth: .infinity, alignment: .leading)
                }

                VStack(spacing: 0) {
                    if session.user != nil {
                        NavigationLink {
                            NicknameEditView()
                        } label: {
                            SettingsRow(title: "编辑昵称", symbol: "pencil")
                        }
                        Divider().padding(.leading, 56)
                    }
                    NavigationLink {
                        SettingsView()
                    } label: {
                        SettingsRow(title: "设置", symbol: "gearshape")
                    }
                }
                .background(palette.card, in: RoundedRectangle(cornerRadius: 16))
            }
            .padding(20)
        }
        .background(palette.page.ignoresSafeArea())
        .navigationTitle("我的")
        .navigationDestination(isPresented: $showLogin) {
            LoginView()
        }
        .alert("登录状态已失效", isPresented: sessionExpiredBinding) {
            Button("重新登录") {
                session.acknowledgeSessionExpired()
                showLogin = true
            }
            Button("暂不登录", role: .cancel) {
                session.acknowledgeSessionExpired()
            }
        } message: {
            Text(session.sessionExpiredNotice ?? "请重新登录")
        }
    }

    private var sessionExpiredBinding: Binding<Bool> {
        Binding(
            get: { session.sessionExpiredNotice != nil },
            set: { presented in
                if !presented { session.acknowledgeSessionExpired() }
            }
        )
    }
}

private struct ProfileCard: View {
    @Environment(\.colorScheme) private var colorScheme
    let user: UserInfo

    private var palette: AppPalette { AppTheme.palette(for: colorScheme) }

    var body: some View {
        HStack(spacing: 14) {
            ProfileAvatar(urlString: user.avatar)
            VStack(alignment: .leading, spacing: 6) {
                Text(user.nickname.isEmpty ? "追分用户" : user.nickname)
                    .font(.title3.bold())
                    .foregroundStyle(palette.textPrimary)
                Text(user.phone.isEmpty ? "未绑定手机号" : user.phone)
                    .font(.subheadline)
                    .foregroundStyle(palette.textSecondary)
            }
            Spacer(minLength: 0)
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .padding(20)
        .background(palette.card, in: RoundedRectangle(cornerRadius: 16))
    }
}

/// 远程头像使用系统 AsyncImage：空值、无效地址与加载失败均保留底层占位图，不阻塞资料卡。
private struct ProfileAvatar: View {
    let urlString: String

    var body: some View {
        ZStack {
            Circle().fill(AppTheme.primary.opacity(0.14))
            Image(systemName: "person.crop.circle.fill")
                .resizable()
                .scaledToFit()
                .foregroundStyle(AppTheme.primary)
                .padding(8)
            if !urlString.isEmpty, let url = URL(string: urlString) {
                AsyncImage(url: url) { image in
                    image.resizable().scaledToFill()
                } placeholder: {
                    Color.clear
                }
            }
        }
        .frame(width: 64, height: 64)
        .clipShape(Circle())
        .accessibilityHidden(true)
    }
}
