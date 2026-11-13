import CoreImage
import CoreImage.CIFilterBuiltins
import UIKit

enum QRCodeImage {
    /// 用系统 CoreImage 生成二维码位图（纠错级别 M、黑白）。
    /// 按整数倍放大到不小于 `size * UIScreen.main.scale` 的像素尺寸，最近邻插值保证边缘不糊。
    static func make(content: String, size: CGFloat) -> UIImage? {
        guard !content.isEmpty, size > 0 else { return nil }

        let filter = CIFilter.qrCodeGenerator()
        filter.message = Data(content.utf8)
        filter.correctionLevel = "M"
        guard let output = filter.outputImage else { return nil }

        let context = CIContext()
        guard let base = context.createCGImage(output, from: output.extent) else { return nil }

        let targetPixels = size * UIScreen.main.scale
        let factor = max(1, Int((targetPixels / CGFloat(base.width)).rounded(.up)))
        let dimension = base.width * factor

        guard let canvas = CGContext(
            data: nil,
            width: dimension,
            height: dimension,
            bitsPerComponent: 8,
            bytesPerRow: 0,
            space: CGColorSpaceCreateDeviceRGB(),
            bitmapInfo: CGImageAlphaInfo.noneSkipLast.rawValue
        ) else { return nil }
        canvas.interpolationQuality = .none
        canvas.draw(base, in: CGRect(x: 0, y: 0, width: dimension, height: dimension))
        guard let image = canvas.makeImage() else { return nil }
        return UIImage(cgImage: image)
    }
}
