import Foundation

struct GearOption: Equatable, Identifiable {
    let value: String
    let label: String

    var id: String { value }
}

/// 登记未收录型号的请求体，只包含用户实际填写的字段。
struct GearUnlistedPayload: Equatable {
    let category: String
    let modelName: String
    var brandId: Int64?
    var brandName: String?
    var cueStyle: String?
    var roles: [String]?
}

struct GearUnlistedForm: Equatable {
    var category: String = GearCategory.cue.rawValue
    var brandId: Int64 = 0
    var brandName = ""
    var modelName = ""
    var cueStyle = ""
    var roles: [String] = []
}

/// 校验结果直接用文案表示失败原因，页面上原样展示。
enum GearFormResult: Equatable {
    case success(GearUnlistedPayload)
    case failure(String)
}

/// 登记未收录型号的表单校验与请求体构造（纯函数），对应小程序 `utils/gear-form.js`。
/// 长度按 Unicode scalar 计，与后端按 rune 计数一致。
enum GearFormRules {
    static let brandNameMax = 32
    static let modelNameMax = 64

    static let cueStyleOptions = [
        GearOption(value: "snooker", label: "斯诺克 / 中式杆"),
        GearOption(value: "pool", label: "美式杆"),
    ]

    static let roleOptions = [
        GearOption(value: "break", label: "冲杆"),
        GearOption(value: "jump", label: "跳杆"),
    ]

    private static func charCount(_ text: String) -> Int {
        text.unicodeScalars.count
    }

    static func toggleRole(_ roles: [String], _ role: String) -> [String] {
        let next = roles.contains(role) ? roles.filter { $0 != role } : roles + [role]
        // 固定顺序，和后端存储一致
        return roleOptions.map(\.value).filter { next.contains($0) }
    }

    /// brandId > 0 表示选了目录品牌；否则必须填写自填品牌名。
    static func buildUnlistedPayload(_ form: GearUnlistedForm) -> GearFormResult {
        guard form.category == GearCategory.cue.rawValue || form.category == GearCategory.breakJump.rawValue else {
            return .failure("装备类别无效")
        }

        let modelName = form.modelName.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !modelName.isEmpty else { return .failure("请填写型号名称") }
        guard charCount(modelName) <= modelNameMax else {
            return .failure("型号名称不能超过 \(String(modelNameMax)) 个字")
        }

        let brandId = form.brandId > 0 ? form.brandId : 0
        let brandName = brandId > 0 ? "" : form.brandName.trimmingCharacters(in: .whitespacesAndNewlines)
        if brandId == 0 {
            guard !brandName.isEmpty else { return .failure("请选择品牌或填写品牌名称") }
            guard charCount(brandName) <= brandNameMax else {
                return .failure("品牌名称不能超过 \(String(brandNameMax)) 个字")
            }
        }

        var payload = GearUnlistedPayload(
            category: form.category,
            modelName: modelName,
            brandId: brandId > 0 ? brandId : nil,
            brandName: brandId > 0 ? nil : brandName
        )

        if form.category == GearCategory.cue.rawValue {
            guard cueStyleOptions.contains(where: { $0.value == form.cueStyle }) else {
                return .failure("请选择杆型")
            }
            payload.cueStyle = form.cueStyle
        } else {
            let roles = roleOptions.map(\.value).filter { form.roles.contains($0) }
            guard !roles.isEmpty else { return .failure("请选择冲杆或跳杆") }
            payload.roles = roles
        }
        return .success(payload)
    }
}
