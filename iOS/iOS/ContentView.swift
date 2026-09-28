import SwiftUI

struct ContentView: View {
    @AppStorage("theme_mode") private var themeMode = "system"

    var body: some View {
        MainTabs()
            .preferredColorScheme(ThemeMode(rawValue: themeMode)?.colorScheme)
    }
}

private enum ThemeMode: String, CaseIterable {
    case system
    case light
    case dark

    var title: String {
        switch self {
        case .system: "跟随系统"
        case .light: "浅色"
        case .dark: "深色"
        }
    }

    var colorScheme: ColorScheme? {
        switch self {
        case .system: nil
        case .light: .light
        case .dark: .dark
        }
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
    @AppStorage("theme_mode") private var themeMode = "system"

    private var palette: AppPalette { AppTheme.palette(for: colorScheme) }

    var body: some View {
        ScrollView {
            VStack(spacing: 20) {
                if session.isRestoring {
                    ProgressView("正在恢复登录状态")
                        .frame(maxWidth: .infinity)
                        .padding(24)
                } else if let user = session.user {
                    VStack(alignment: .leading, spacing: 16) {
                        HStack(spacing: 12) {
                            Image(systemName: "person.crop.circle.fill")
                                .font(.system(size: 40))
                                .foregroundStyle(AppTheme.primary)
                            VStack(alignment: .leading, spacing: 4) {
                                Text(user.nickname.isEmpty ? "追分用户" : user.nickname)
                                    .font(.title3.bold())
                                    .foregroundStyle(palette.textPrimary)
                                Text(user.phone)
                                    .font(.subheadline)
                                    .foregroundStyle(palette.textSecondary)
                            }
                        }
                        Button("退出登录", role: .destructive) {
                            session.logout()
                        }
                    }
                    .frame(maxWidth: .infinity, alignment: .leading)
                    .padding(20)
                    .background(palette.card, in: RoundedRectangle(cornerRadius: 16))
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

                VStack(alignment: .leading, spacing: 12) {
                    Text("主题模式")
                        .font(.headline)
                        .foregroundStyle(palette.textPrimary)
                    Picker("主题模式", selection: $themeMode) {
                        ForEach(ThemeMode.allCases, id: \.rawValue) { mode in
                            Text(mode.title).tag(mode.rawValue)
                        }
                    }
                    .pickerStyle(.segmented)
                }
                .frame(maxWidth: .infinity, alignment: .leading)
                .padding(20)
                .background(palette.card, in: RoundedRectangle(cornerRadius: 16))
            }
            .padding(20)
        }
        .background(palette.page.ignoresSafeArea())
        .navigationTitle("我的")
    }
}
