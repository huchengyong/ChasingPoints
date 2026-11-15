import SwiftUI
import UIKit
import XCTest

/// 把视图渲染成 PNG。设置了环境变量 SNAPSHOT_DIR 时写入 `<SNAPSHOT_DIR>/<name>-<light|dark>.png`，否则只渲染不落盘。
/// 返回渲染出的图片，调用方至少断言尺寸正确。
@MainActor
@discardableResult
func renderSnapshot<Content: View>(
    _ name: String,
    colorScheme: ColorScheme,
    size: CGSize = CGSize(width: 393, height: 852),
    @ViewBuilder content: () -> Content
) throws -> UIImage {
    let scenes = UIApplication.shared.connectedScenes.compactMap { $0 as? UIWindowScene }
    guard let scene = scenes.first(where: { $0.activationState == .foregroundActive }) ?? scenes.first else {
        throw XCTSkip("找不到可用的 UIWindowScene，跳过快照")
    }

    let window = UIWindow(windowScene: scene)
    window.frame = CGRect(origin: .zero, size: size)
    window.overrideUserInterfaceStyle = colorScheme == .dark ? .dark : .light
    window.rootViewController = UIHostingController(
        rootView: content().environment(\.colorScheme, colorScheme)
    )
    window.makeKeyAndVisible()
    window.layoutIfNeeded()
    // 等 SwiftUI 完成布局、导航栏与异步图片等首帧状态。
    RunLoop.main.run(until: Date().addingTimeInterval(0.3))
    window.layoutIfNeeded()

    let image = UIGraphicsImageRenderer(size: size).image { _ in
        window.drawHierarchy(in: window.bounds, afterScreenUpdates: true)
    }

    window.isHidden = true
    window.rootViewController = nil

    if let directory = ProcessInfo.processInfo.environment["SNAPSHOT_DIR"] {
        let safeName = name.filter { $0.isASCII && ($0.isLetter || $0.isNumber || $0 == "-" || $0 == "_") }
        let directoryURL = URL(fileURLWithPath: directory, isDirectory: true)
        try FileManager.default.createDirectory(at: directoryURL, withIntermediateDirectories: true)
        let suffix = colorScheme == .dark ? "dark" : "light"
        guard let data = image.pngData() else {
            throw NSError(domain: "renderSnapshot", code: 1, userInfo: [NSLocalizedDescriptionKey: "无法生成 PNG"])
        }
        try data.write(to: directoryURL.appendingPathComponent("\(safeName)-\(suffix).png"))
    }

    return image
}
