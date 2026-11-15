import Photos
import SwiftUI
import UIKit

/// 战绩海报页，对应小程序 `subPages/match/shareResult.vue`。
struct ShareResultView: View {
    @EnvironmentObject private var session: SessionStore
    @EnvironmentObject private var toast: ToastCenter
    @Environment(\.colorScheme) private var colorScheme

    @StateObject private var model: ShareResultModel
    @State private var isPermissionAlertPresented = false

    init(session: SessionStore, matchId: Int64, model: ShareResultModel? = nil) {
        _model = StateObject(wrappedValue: model ?? ShareResultModel(session: session, matchId: matchId))
    }

    private var palette: AppPalette { AppTheme.palette(for: colorScheme) }

    var body: some View {
        Group {
            switch model.phase {
            case .loading:
                loadingState
            case .ready:
                previewState
            case .failed:
                failureState
            }
        }
        .frame(maxWidth: .infinity, maxHeight: .infinity)
        .background(palette.page.ignoresSafeArea())
        .navigationTitle("战绩海报")
        .navigationBarTitleDisplayMode(.inline)
        .safeAreaInset(edge: .bottom) { actionBar }
        .task { await model.load() }
        .onChange(of: session.identity) { _, _ in
            Task { await model.load() }
        }
        .alert("提示", isPresented: $isPermissionAlertPresented) {
            Button("去设置") {
                guard let url = URL(string: UIApplication.openSettingsURLString) else { return }
                UIApplication.shared.open(url)
            }
            Button("取消", role: .cancel) {}
        } message: {
            Text("需要您授权保存图片到相册")
        }
    }

    private var loadingState: some View {
        VStack(spacing: 10) {
            ProgressView()
            Text("生成海报中...")
                .font(.system(size: 14))
                .foregroundStyle(palette.textSecondary)
        }
    }

    private var previewState: some View {
        ScrollView {
            if let image = model.posterImage {
                Image(uiImage: image)
                    .resizable()
                    .scaledToFit()
                    .clipShape(RoundedRectangle(cornerRadius: 16))
                    .shadow(color: .black.opacity(0.12), radius: 16, y: 4)
                    .padding(.horizontal, 24)
                    .padding(.vertical, 20)
            }
        }
    }

    private var failureState: some View {
        Text("海报生成失败")
            .font(.system(size: 14))
            .foregroundStyle(palette.textSecondary)
    }

    private var actionBar: some View {
        HStack(spacing: 12) {
            Button {
                Task { await saveToAlbum() }
            } label: {
                Label("保存到相册", systemImage: "arrow.down.to.line")
            }
            .buttonStyle(.appPrimary)

            if let image = model.posterImage {
                ShareLink(
                    item: Image(uiImage: image),
                    preview: SharePreview("战绩海报", image: Image(uiImage: image))
                ) {
                    Label("分享", systemImage: "square.and.arrow.up")
                }
                .buttonStyle(.appSecondary)
            } else {
                Button {} label: {
                    Label("分享", systemImage: "square.and.arrow.up")
                }
                .buttonStyle(.appSecondary)
                .disabled(true)
            }
        }
        .padding(.horizontal, 16)
        .padding(.vertical, 10)
        .background(palette.card)
        .overlay(alignment: .top) {
            Divider()
        }
    }

    private func saveToAlbum() async {
        guard let image = model.posterImage else {
            toast.show("海报未生成")
            return
        }

        let status = await PHPhotoLibrary.requestAuthorization(for: .addOnly)
        switch status {
        case .authorized, .limited:
            do {
                try await PHPhotoLibrary.shared().performChanges {
                    PHAssetChangeRequest.creationRequestForAsset(from: image)
                }
                toast.show("保存成功")
            } catch {
                toast.show("保存失败")
            }
        case .denied, .restricted:
            isPermissionAlertPresented = true
        default:
            toast.show("保存失败")
        }
    }
}
