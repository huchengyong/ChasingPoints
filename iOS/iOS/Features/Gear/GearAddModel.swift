import Combine
import Foundation

/// 添加装备 / 选择皮头与巧克：目录筛选、分页与登记。
@MainActor
final class GearAddModel: ObservableObject {
    enum CreateOutcome: Equatable {
        case created(Int64?)
        case failed(String)
    }

    static let searchDebounce: UInt64 = 350_000_000

    let isPick: Bool
    let pickToken: String?

    @Published private(set) var category: GearCategory
    @Published private(set) var brands: [GearBrandInfo] = []
    @Published private(set) var brandId: Int64 = 0
    @Published private(set) var keyword = ""
    @Published private(set) var houseProduct: GearProductInfo?
    @Published private(set) var products: [GearProductInfo] = []
    @Published private(set) var listLoading = false
    @Published private(set) var listError: String?
    @Published private(set) var hasMore = false
    @Published private(set) var showUnlisted = false
    @Published private(set) var submitting = false
    @Published var form = GearUnlistedForm()

    private let session: SessionStore
    private let store: GearStore
    private let api: GearAPI
    private var queryVersion = 0
    private var page = 1
    private var searchTask: Task<Void, Never>?

    init(
        session: SessionStore,
        store: GearStore,
        requestedCategory: String?,
        isPick: Bool,
        pickToken: String?,
        api: GearAPI? = nil
    ) {
        self.session = session
        self.store = store
        self.api = api ?? GearAPI(session: session)
        self.isPick = isPick
        self.pickToken = pickToken

        // 选择模式只允许皮头与巧克；非法类别取允许的第一个。
        let allowed: [GearCategory] = isPick ? [.tip, .chalk] : [.cue, .breakJump]
        let requested = requestedCategory.flatMap(GearCategory.init(rawValue:))
        let initial = requested.flatMap { allowed.contains($0) ? $0 : nil } ?? allowed[0]
        category = initial
        form.category = initial.rawValue
    }

    var categoryLabel: String { category.label }
    var houseHint: String { category == .chalk ? "用球房的巧克也能记录" : "用球房的公杆打球也能记录" }
    var worksWithHouse: Bool { category == .cue || category == .chalk }

    // MARK: - 首屏与筛选

    func start() async {
        queryVersion += 1
        let version = queryVersion
        brandId = 0
        keyword = ""
        products = []
        brands = []
        page = 1
        hasMore = false
        showUnlisted = false
        listError = nil
        form = GearUnlistedForm(category: category.rawValue)

        async let brandsLoad: Void = loadBrands(version: version)
        async let houseLoad: Void = loadHouse(version: version)
        async let productsLoad: Void = loadProducts(append: false, version: version)
        _ = await (brandsLoad, houseLoad, productsLoad)
    }

    func switchCategory(_ value: GearCategory) {
        guard value != category else { return }
        category = value
        Task { await start() }
    }

    func selectBrand(_ id: Int64) {
        guard id != brandId else { return }
        brandId = id
        reloadList()
    }

    func updateKeyword(_ text: String) {
        guard text != keyword else { return }
        keyword = text
        searchTask?.cancel()
        searchTask = Task { [weak self] in
            try? await Task.sleep(nanoseconds: Self.searchDebounce)
            guard !Task.isCancelled else { return }
            self?.reloadList()
        }
    }

    func applyKeyword() {
        searchTask?.cancel()
        reloadList()
    }

    func clearKeyword() {
        guard !keyword.isEmpty else { return }
        keyword = ""
        applyKeyword()
    }

    func toggleUnlisted() {
        showUnlisted.toggle()
    }

    func chooseFormBrand(_ id: Int64) {
        form.brandId = id
        if id != 0 { form.brandName = "" }
    }

    func chooseCueStyle(_ value: String) {
        form.cueStyle = value
    }

    func chooseRole(_ value: String) {
        form.roles = GearFormRules.toggleRole(form.roles, value)
    }

    func loadMore() {
        guard !listLoading, hasMore || listError != nil else { return }
        let version = queryVersion
        Task { await loadProducts(append: true, version: version) }
    }

    private func reloadList() {
        queryVersion += 1
        page = 1
        hasMore = false
        products = []
        listError = nil
        let version = queryVersion
        Task { await loadProducts(append: false, version: version) }
    }

    // MARK: - 读取

    private var currentIdentity: SessionIdentity? { session.identity }

    private func loadBrands(version: Int) async {
        let identity = currentIdentity
        do {
            let list = try await api.brands(category: category)
            guard version == queryVersion, identity == session.identity else { return }
            brands = list
        } catch {
            guard version == queryVersion, identity == session.identity else { return }
            // 品牌筛选只是辅助：读取失败不影响直接搜索与浏览型号。
            brands = []
        }
    }

    private func loadHouse(version: Int) async {
        houseProduct = nil
        guard worksWithHouse else { return }
        let identity = currentIdentity
        do {
            let response = try await api.products(category: category, house: true, page: 1, pageSize: 1)
            guard version == queryVersion, identity == session.identity else { return }
            houseProduct = response.list.first
        } catch {
            guard version == queryVersion, identity == session.identity else { return }
            houseProduct = nil
        }
    }

    private func loadProducts(append: Bool, version: Int) async {
        let identity = currentIdentity
        let nextPage = append ? page + 1 : 1
        listLoading = true
        listError = nil
        defer {
            if version == queryVersion { listLoading = false }
        }
        do {
            let response = try await api.products(
                category: category,
                brandId: brandId,
                keyword: keyword,
                page: nextPage
            )
            guard version == queryVersion, identity == session.identity else { return }
            // 没有关键词或品牌筛选时不重复展示球房公用条目。
            let list = response.list.filter {
                !$0.house || !keyword.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty || brandId > 0
            }
            products = append ? products + list : list
            page = nextPage
            hasMore = response.total > Int64(nextPage * GearAPI.pageSize)
        } catch {
            guard version == queryVersion, identity == session.identity else { return }
            listError = error.localizedDescription
            if !append { products = [] }
        }
    }

    // MARK: - 登记

    func create(from product: GearProductInfo) async -> CreateOutcome? {
        guard let identity = session.identity, !submitting else { return nil }
        submitting = true
        defer { submitting = false }
        do {
            let item = try await api.create(GearCreateRequest(category: category.rawValue, productId: product.id))
            guard session.identity == identity else { return nil }
            return finishCreate(item, fallback: "添加失败，请重试")
        } catch {
            guard session.identity == identity else { return nil }
            return .failed(error.localizedDescription)
        }
    }

    func submitUnlisted() async -> CreateOutcome? {
        guard let identity = session.identity, !submitting else { return nil }
        let payload = GearFormRules.buildUnlistedPayload(form)
        let request: GearCreateRequest
        switch payload {
        case .failure(let message):
            return .failed(message)
        case .success(let value):
            request = GearCreateRequest(
                category: value.category,
                brandId: value.brandId,
                brandName: value.brandName,
                modelName: value.modelName,
                cueStyle: value.cueStyle,
                roles: value.roles
            )
        }

        submitting = true
        defer { submitting = false }
        do {
            let item = try await api.create(request)
            guard session.identity == identity else { return nil }
            return finishCreate(item, fallback: "登记失败，请重试")
        } catch {
            guard session.identity == identity else { return nil }
            return .failed(error.localizedDescription)
        }
    }

    private func finishCreate(_ item: GearItemInfo?, fallback: String) -> CreateOutcome {
        guard let item else { return .failed(fallback) }
        store.applyCreated(item)
        store.markJustAdded(item.id)
        return .created(item.id)
    }
}
