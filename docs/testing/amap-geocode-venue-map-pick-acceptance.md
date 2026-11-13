# 高德地址解析与小程序球馆选点验收记录

日期：2026-11-13。范围：后端球馆地理编码从 apihz 改为高德 Web 服务、提交球馆支持微信地图选点、用户提交的球馆一律人工审核；UniApp 只有微信小程序显示选点，鸿蒙与原生 iOS 不涉及（iOS 以后用 MapKit 选点，未做）。同日稍早的 apihz 密钥轮换与错误信息脱敏见提交 `c8f39f6`、`8d06079`。

## 1. 改动

- 新增 `internal/pkg/geocode/amap_client.go`：请求 `restapi.amap.com/v3/geocode/geo`，按高德数字签名规则加 `sig`（参数按名排序、原值拼接、末尾接私钥、MD5 小写）。key 与签名私钥只从环境变量 `AMAP_WEB_SERVICE_KEY`、`AMAP_WEB_SERVICE_SIGNING_KEY` 读取，`geocode_accounts` 里 `provider=amap` 的账号行 `provider_key` 留空，只承担限流、冷却和鉴权失败停用。错误码分类：key/IP 白名单/签名等鉴权类停用账号；`200xx` 参数类与 `3xxxx` 引擎数据错误（解析不了的地址返回 `30001`）只让任务退避重试、不冷却账号；其余冷却 1 分钟。请求失败的错误信息不含 URL、key 和签名。
- `GEOCODE_PROVIDER` 选择服务商（默认 `amap`）；缺 key 或服务商未知时记错误日志并不启动 worker，不影响服务启动。
- 精度门槛：高德只有门牌号、门址、兴趣点、单元号、楼层、房间算精确；其他级别只写坐标、不改球馆状态。`ApplyGeocodeResultWithTx` 不发布时不再写 `status=0`（原来 `AutoPublish=false` 会把球馆置为隐藏）。
- `CreateVenueReq` 增加可选 `latitude`、`longitude`（GCJ-02）。成对、非零、在中国大致范围内才接受；带坐标时直接写 `geo_source=user_pick`、`geo_level=用户选点`，不建解析任务。
- UniApp 提交页（仅 `MP-WEIXIN`）：可选「地图位置」，`uni.chooseLocation` 选点后回填空着的名称和地址（地址按地区名去前缀，直辖市省级名为「上海」「北京」时按长度优先匹配）；`manifest.json` 的 `requiredPrivateInfos` 增加 `chooseLocation`。小程序版本升为 1.0.9。
- 发布策略（用户决定）：常玩球馆 VIP 只在后台审核通过时发放，自动发布会让用户拿不到奖励，因此测试与生产服务器 env 均设 `GEOCODE_AUTO_PUBLISH=false`。现在解析成功或用户选点都只写坐标，球馆保持待审核。仓库 yaml 与代码默认值仍为 `true`，由 env 覆盖。

代码由 opencode 中的 DeepSeek V4.1 Flash 按规格编写，逐轮审查 diff 并复跑测试；审查中发现并返工两处：`30001` 原被当作临时错误会冷却唯一账号、直辖市地址去前缀得到「市浦东新区…」。

## 2. 本次自动化执行

- `backend/`：`go test ./...` 通过；`go vet` 涉及包通过；`gofmt -l` 对改动文件无输出（仓库另有 7 个历史未格式化文件，未改动）。新增覆盖：签名校验向量与中文地址签名、请求参数与回算签名、无私钥不带 `sig`、精度判定（含 `level` 为 `[]`）、空结果、错误码分类（含 `30001`、`32200`）、HTTP 500、传输错误不泄露 key/私钥/签名/域名、非法坐标；worker 仅在自动发布开启且精确时发布；服务商选择；创建球馆的选点校验（不完整、越界、NaN）与选点后不建任务。
- `uniapp/`：`node --test tests/*.test.mjs` 1219 个全部通过（worktree 中无 `.DS_Store`）。新增覆盖：payload 只在选点有效时带坐标且无 `undefined` 字段、`normalizePickedLocation`、`stripRegionPrefix`（含直辖市）、`requiredPrivateInfos` 含 `chooseLocation`。
- HBuilderX CLI 编译：微信小程序产物含选点区块与 `chooseLocation` 声明；鸿蒙产物编译通过，提交页存在且不含选点区块。

## 3. 发布与线上核对

- 高德请求实测（服务器上运行脚本，只打印状态与结果）：签名请求返回 `infocode=10000`；`世纪大道100号` 为门址级 `121.507572,31.234583`。同一地址 apihz 返回约偏 500 米（疑为 WGS-84），两边 `venues` 表当时都为 0 条，无旧数据需修正。测试机到高德 0.9–6.2 秒，偶发 10 秒超时，测试机 env 单独设 `GEOCODE_REQUEST_TIMEOUT_MS=10000`；生产 0.06 秒，保持默认 5000。
- 开发环境后端：发布号 `20261113160910-13e26629a8f0`。插入测试球馆实测：门址级自动发布（当时 `AutoPublish` 仍开启）、乡镇级写坐标保持待审核、不存在的地址返回 `30001` 后任务退避且账号冷却时间不变；测试数据已删。
- 生产后端：`devlop` 合入 `main`（`14dcdde`），先在生产库插入 `amap` 账号行（id 3，每分钟 60 次），预演提交清单为 3 个、待执行迁移 0 个，发布号 `20261113163449-14dcdde35c6a`；本机与公网健康检查 200，应用日志 error=0，二进制 `vcs.revision=14dcdde`。用乡镇级地址实测解析成功且保持待审核。
- 关闭自动发布：两台服务器 env 加 `GEOCODE_AUTO_PUBLISH=false` 后重启（生产 `restart-prod.sh`，健康检查 200、error=0）；生产用门址级地址实测坐标写入且保持待审核。所有生产测试数据已删，`venues` 为 0 条。
- 微信后台：`wx.chooseLocation` 接口申请已开通（用户操作）。申请材料截图用开发者工具开发构建（请求测试环境）拍摄，测试库临时示例球馆已删除。
- 小程序：版本 1.0.9 生产构建（产物无开发域名、appid 正确、只请求 `api.zhuifen.cn`）后上传开发版本，主包 1.42 MB、总包 2.89 MB；未设为体验版，未提审。

## 4. 未执行

- 真机上的地图选点与提交（需体验版）；鸿蒙真机。
- 通过接口实际提交带坐标的球馆（需登录态），只由单元测试覆盖。
- 原生 iOS 的 MapKit 选点（计划内，未做）。
