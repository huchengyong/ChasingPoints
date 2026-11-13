# RB4：球馆（附近球馆、球馆详情、上传球馆）

本轮迁移球馆模块三个页面。

## 本轮要读的小程序源码

- `uniapp/subPages/venue/index.vue`、`detail.vue`、`submit.vue`（含各自样式）、`uniapp/api/venue.js`
- `uniapp/utils/venue-submit.js` + `uniapp/tests/venue-submit.test.mjs`；`uniapp/utils/permission-helper.js` 里的 `getCurrentLocation`（了解定位失败 / 拒绝授权时页面怎么降级）
- 上传球馆成功后与「常玩球馆奖励」的联动：`submit.vue` 里对 `useUserOverviewStore` 的处理
- 契约：`backend/chasing_points.api` 的 `/api/venue/*`（`GetVenueListReq/Resp`、`GetNearbyVenuesReq`、`GetVenueDetailReq/Resp`、`GetVenueAreaOptionsReq/Resp`、`CreateVenueReq/Resp`、`VenueIdReq`、`VenueInfo` 等）。地址解析、坐标字段的含义读 `backend/internal/logic/venue/` 确认（坐标系是 GCJ-02）。

## 任务

1. **定位能力**（新建 `iOS/iOS/Shared/LocationProvider.swift`）：

```swift
struct Coordinate: Equatable { let latitude: Double; let longitude: Double }

enum LocationError: Error, Equatable { case denied, unavailable }

/// 一次性定位。页面通过它取当前位置，测试里注入替身。
@MainActor
protocol LocationProviding {
    func currentLocation() async throws -> Coordinate
}

/// 系统实现：CLLocationManager，使用期间授权；未授权时发起授权请求，被拒绝抛 .denied。
@MainActor
final class SystemLocationProvider: NSObject, LocationProviding { ... }
```

   - 大陆地区 `CLLocationManager` 返回的是 WGS-84，后端与小程序用 GCJ-02：在这个文件里实现标准的 WGS-84 → GCJ-02 转换（公开算法，纯函数，境外坐标不转换），`SystemLocationProvider` 返回转换后的坐标，并为转换函数写测试（用几组已知对照值，误差 < 1e-4 度即可；境外坐标原样返回）。
   - 定位权限说明文案（Info.plist）由监工另行配置，本轮不要改工程文件；没有配置时系统会直接拒绝，按 `.denied` 处理即可。
2. **附近球馆**（`iOS/iOS/Features/Venue/`）：`.venueList` → `VenueListView`，标题「附近球馆」。列表 / 附近两种读取方式、筛选、分页、定位失败或拒绝授权时的降级与提示、空态、「上传球馆」入口，全部以 `index.vue` 为准。访客可看；需要登录的操作未登录时走登录页。
3. **球馆详情**：`.venueDetail(id:)` → `VenueDetailView`，标题「球馆详情」。内容与操作以 `detail.vue` 为准：
   - 拨打电话 → `UIApplication.shared.open` 打开 `tel:` 链接；
   - 导航 / 查看位置（小程序的 `uni.openLocation`）→ 用 MapKit 的 `MKMapItem.openInMaps` 打开系统地图（坐标按 GCJ-02 直接使用）；详情页里可以放一张不可交互的 `Map` 小地图显示位置；
   - 打卡（`POST /api/venue/checkin`）及其成功 / 失败提示。
4. **上传球馆**：`.venueSubmit` → `VenueSubmitView`，标题「上传球馆」。需要登录。表单字段、校验规则、提交参数、地区选项读取、成功后的提示与返回，全部以 `submit.vue` 和 `venue-submit.js` 为准（规则迁移并搬测试）。
   - 地图选点（小程序的 `uni.chooseLocation`）：原生端做一个 `VenueLocationPicker`（sheet）——`Map` + 屏幕中心固定图钉，拖动地图选位置，初始位置取当前定位（失败时用表单里已有坐标或默认城市中心），「确定」后把中心坐标回填表单。地址文字不做逆地理编码，由用户在表单里填写（与小程序选点后可手动修改地址的行为兼容）。
   - 提交成功后与常玩球馆奖励的联动：原生端没有跨页面的概览缓存，提交成功即可；在交付说明里写明小程序这里做了什么。
5. 图片：球馆封面等后端返回的地址按现有 `RemoteImage` 显示。如果 `submit.vue` 有上传图片的步骤，本轮先不做上传（在交付说明里列出），表单其余部分照常迁移。

## 测试

搬运 `venue-submit.test.mjs`；坐标转换；接口门面的路径 / 参数（含 `nearby` 的经纬度参数、访客不带 Authorization）；列表模型：定位成功走附近、定位被拒走降级路径并给出提示、分页、空、失败重试；详情模型：读取、打卡成功 / 失败 / 未登录；提交模型：各项校验文案、提交参数、失败保留表单、提交中不可重复提交。快照：`venue-list`、`venue-list-location-denied`、`venue-list-empty`、`venue-detail`、`venue-submit`（已填部分字段）、`venue-submit-errors`（校验失败态）。快照里不要渲染真实地图瓦片：给地图区域留一个可注入的开关或占位（测试时显示占位色块）。

## 并行开发提醒

其他开发线正在修改 `iOS/iOS/Features/Profile/**`、`Features/Settings/**`、`Features/Reputation/**`、`Features/Opponents/**`、`Features/Stats/**`、`Features/News/**`、`iOS/iOS/SettingsView.swift`、`iOS/iOS/SessionStore.swift` 及其测试，本轮不要碰这些文件。
