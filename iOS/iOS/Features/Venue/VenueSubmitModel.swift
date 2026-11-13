import Combine
import Foundation

/// 上传球馆表单状态。地区选项按 parent_id 懒加载并缓存，规则来自 VenueSubmitRules。
@MainActor
final class VenueSubmitModel: ObservableObject {
    @Published var form = VenueSubmitForm()
    @Published private(set) var columns: [[VenueAreaOption]] = VenueSubmitModel.initialColumns
    @Published private(set) var columnIndexes = [0, 0, 0]
    @Published private(set) var isLoadingAreas = false
    @Published private(set) var isAreaReady = false
    @Published private(set) var showValidation = false
    @Published private(set) var isSubmitting = false
    @Published private(set) var notice: String?
    @Published private(set) var didSubmit = false
    @Published private(set) var rewardStatus: FavoriteVenueRewardStatus?

    private static let initialColumns = [
        [VenueAreaOption.placeholder("地区加载中...")],
        [VenueAreaOption.placeholder("请选择城市")],
        [VenueAreaOption.placeholder("请选择区域")],
    ]

    private let session: SessionStore
    private let api: VenueAPI
    private let rewardStatusLoader: @MainActor () async -> FavoriteVenueRewardStatus?
    private var areaCache: [Int64: [VenueAreaOption]] = [:]
    private var hydrateToken = 0
    private var didLoad = false
    private var owner: SessionIdentity?

    init(
        session: SessionStore,
        api: VenueAPI? = nil,
        rewardStatusLoader: (@MainActor () async -> FavoriteVenueRewardStatus?)? = nil
    ) {
        self.session = session
        self.api = api ?? VenueAPI(session: session)
        self.rewardStatusLoader = rewardStatusLoader ?? {
            try? await ProfileAPI(session: session).overview().availableFavoriteVenueReward
        }
        owner = session.identity
    }

    var intro: VenueSubmitIntro {
        VenueSubmitRules.resolveIntro(rewardStatus)
    }

    var barCopy: VenueSubmitBarCopy {
        VenueSubmitRules.resolveBarCopy(missingRequiredLabels: missingRequiredLabels, rewardStatus: rewardStatus)
    }

    var regionPlaceholder: String {
        isLoadingAreas && !isAreaReady ? "地区加载中..." : "请选择省 / 市 / 区"
    }

    var isNameValid: Bool { !trimmed(form.name).isEmpty }
    var isRegionValid: Bool { !trimmed(form.regionText).isEmpty }
    var isAddressValid: Bool { !trimmed(form.address).isEmpty }

    var missingRequiredLabels: [String] {
        var labels: [String] = []
        if !isNameValid { labels.append("球馆名称") }
        if !isRegionValid { labels.append("地区") }
        if !isAddressValid { labels.append("详细地址") }
        return labels
    }

    func loadIfNeeded() async {
        syncIdentity()
        guard !didLoad else { return }
        didLoad = true
        await hydrateAreaColumns(indexes: [0, 0, 0])
        rewardStatus = await rewardStatusLoader()
    }

    func syncIdentity() {
        let current = session.identity
        guard current != owner else { return }
        owner = current
        didLoad = false
        // 切号或退出后丢弃上一位用户的草稿与结果。
        form = VenueSubmitForm()
        showValidation = false
        notice = nil
        areaCache = [:]
        columns = Self.initialColumns
        columnIndexes = [0, 0, 0]
        isAreaReady = false
        rewardStatus = nil
        didSubmit = false
    }

    /// 逐列选择后重新加载下级地区；与小程序 columnchange 一致。
    func selectAreaColumn(_ column: Int, index: Int) async {
        guard (0..<3).contains(column) else { return }
        var next = columnIndexes
        next[column] = index
        if column == 0 {
            next[1] = 0
            next[2] = 0
        }
        if column == 1 {
            next[2] = 0
        }
        // 先更新滚轮位置，避免网络加载期间选择看起来没反应。
        columnIndexes = next
        await hydrateAreaColumns(indexes: next)
    }

    /// 确认地区选择；缺少城市时提示并保持原选择。
    @discardableResult
    func confirmAreaSelection() async -> Bool {
        await hydrateAreaColumns(indexes: columnIndexes)
        let selection = VenueSubmitRules.regionSelection(currentAreaPath)
        guard !selection.city.isEmpty else {
            notice = "请选择完整地区"
            return false
        }
        form.regionText = selection.regionText
        form.areaIds = selection.areaIds
        form.city = selection.city
        form.district = selection.district
        return true
    }

    /// 按小程序顺序返回第一条校验错误；全部通过返回 nil。
    @discardableResult
    func validate() -> String? {
        showValidation = true
        if !isNameValid { return "请输入球馆名称" }
        if !isRegionValid { return "请选择地区" }
        if !isAddressValid { return "请输入详细地址" }
        return nil
    }

    func submit() async {
        guard !isSubmitting else { return }
        if let message = validate() {
            notice = message
            return
        }
        guard session.identity != nil else {
            notice = "请先登录"
            return
        }

        let requestOwner = session.identity
        isSubmitting = true
        defer { isSubmitting = false }

        do {
            let result = try await api.create(VenueSubmitRules.payload(form))
            guard session.identity == requestOwner else { return }
            didSubmit = true
            notice = result.message.isEmpty ? "已提交，审核通过后会展示给附近球友" : result.message
        } catch {
            guard session.identity == requestOwner else { return }
            notice = "提交失败"
        }
    }

    func clearNotice() {
        notice = nil
    }

    private var currentAreaPath: [VenueAreaOption] {
        columnIndexes.enumerated().compactMap { column, index in
            guard column < columns.count, index < columns[column].count else { return nil }
            return columns[column][index]
        }
        .filter { $0.areaId > 0 }
    }

    private func hydrateAreaColumns(indexes: [Int]) async {
        hydrateToken += 1
        let token = hydrateToken
        isLoadingAreas = true
        defer {
            if token == hydrateToken { isLoadingAreas = false }
        }

        do {
            let provinces = try await areaOptions(parentId: 0)
            let provinceIndex = clamp(indexes.first ?? 0, provinces)
            let province = provinces[provinceIndex]

            let cities: [VenueAreaOption]
            if province.areaId > 0 {
                cities = try await areaOptions(parentId: province.areaId)
            } else {
                cities = [.placeholder("请选择城市")]
            }
            let cityIndex = clamp(indexes.count > 1 ? indexes[1] : 0, cities)
            let city = cities[cityIndex]

            let districts: [VenueAreaOption]
            if city.areaId > 0 {
                districts = try await areaOptions(parentId: city.areaId)
            } else {
                districts = [.placeholder("请选择区域")]
            }
            let districtIndex = clamp(indexes.count > 2 ? indexes[2] : 0, districts)

            guard token == hydrateToken else { return }
            columns = [provinces, cities, districts]
            columnIndexes = [provinceIndex, cityIndex, districtIndex]
            isAreaReady = provinces.contains { $0.areaId > 0 }
        } catch {
            guard token == hydrateToken else { return }
            isAreaReady = false
            notice = "地区数据加载失败"
        }
    }

    private func areaOptions(parentId: Int64) async throws -> [VenueAreaOption] {
        if let cached = areaCache[parentId] { return cached }
        let response = try await api.areas(parentId: parentId)
        let list = normalize(response, emptyLabel: parentId == 0 ? "暂无地区数据" : "暂无下级地区")
        areaCache[parentId] = list
        return list
    }

    private func normalize(_ list: [VenueAreaOption], emptyLabel: String) -> [VenueAreaOption] {
        let normalized = list
            .map {
                VenueAreaOption(
                    areaId: $0.areaId > 0 ? $0.areaId : 0,
                    parentId: $0.parentId > 0 ? $0.parentId : 0,
                    name: trimmed($0.name)
                )
            }
            .filter { $0.areaId > 0 && !$0.name.isEmpty }
        return normalized.isEmpty ? [.placeholder(emptyLabel)] : normalized
    }

    private func clamp(_ index: Int, _ options: [VenueAreaOption]) -> Int {
        guard !options.isEmpty else { return 0 }
        return min(max(index, 0), options.count - 1)
    }

    private func trimmed(_ value: String) -> String {
        value.trimmingCharacters(in: .whitespacesAndNewlines)
    }
}
