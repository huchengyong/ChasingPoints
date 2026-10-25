# UniApp 主导航重排：验收记录

对应变更：`openspec/changes/restructure-uniapp-main-tabs`（任务 1.6、9.1–9.4）。
范围：UniApp 底部导航改为「首页 | 赛讯 | 扫码 | 消息 | 我的」，观赛并入首页，统一扫码流程，消息 Tab。UniApp 只编译微信小程序与鸿蒙；iOS / Android 改为原生开发，不在本次范围。

## 1. 本次自动化执行

环境：本机 macOS；Node v22.19.0；HBuilderX 4.76（`/Applications/HBuilderX.app`，CLI 构建）。

| 项目 | 命令（工作目录 `uniapp/`） | 结果 |
| --- | --- | --- |
| UniApp Node 测试 | `node --test tests/*.test.mjs` | 991 个用例全部通过 |
| 指南镜像 | `cmp AGENTS.md GEMINI.md` | 一致 |
| 空白检查 | `git diff --check`（仓库根） | 无问题 |
| OpenSpec 校验 | `openspec validate restructure-uniapp-main-tabs --strict` | 通过 |
| 微信小程序生产构建 | HBuilderX CLI，`UNI_PLATFORM=mp-weixin`、`NODE_ENV=production`，输出到临时目录 | 构建成功，见第 2 节 |
| 鸿蒙编译 | 同上，`UNI_PLATFORM=app-harmony`、`NODE_ENV=production` | 构建成功，见第 2 节 |

新增测试覆盖：主导航纯逻辑与两端同步分支、原生自定义 TabBar 组件（桩 `Component` / `wx`）、`useMainTabBar` 的身份绑定与重试、统一扫码流程（重复触发、取消、切号丢弃、凭证只交接一次、再来一局锁对手）、扫码确认页挂载、消息 Tab 挂载与刷新规则、首页「正在进行」缓存与读取预算、导航契约（所有 `switchTab` 目标都是 Tab、无指向已移除页面的链接）。

## 2. 构建产物核对

微信小程序生产构建：

- `app.json` 的 `tabBar.custom` 为 `true`，没有 `midButton`；Tab 页依次为 `pages/index/index`、`pages/tournament/index`、`pages/message/index`、`pages/user/index`。
- `custom-tab-bar/` 的四个文件原样拷入产物；页面 JSON 含 `usingComponents`，`app.json` 含全局 `usingComponents`。
- `subPages/match/scanConfirm` 已编译，`subPages/notification` 分包已不存在。
- 主包约 976 KB（上限 2 MB，构建前约 978 KB）；产物中 `kekemate` 出现 0 次。
- `app.wxss` 含 `--ui-tab-bar-space`（小程序为 `calc(50px + env(safe-area-inset-bottom))`），「我的」页的 `max(env(...), var(--ui-tab-bar-space))` 在 Sass 编译后保留。

鸿蒙编译产物：`app-config-service.js` 中 `tabBar.midButton` 为 `{"width":"50px","height":"50px","backgroundImage":"/static/tabbar/scan-button.png"}`，无 `custom`；`app-service.js` 含 `onTabBarMidButtonTap` 注册与 `scanConfirm` 页。运行时源码（`@dcloudio/uni-app-runtime` 4.73）中 `uni-tabview` 原生模块提供 `onMidButtonClick`，中间按钮插在 `list.length / 2` 项之后。

## 3. 视觉自检（仅布局参考，不是验收）

用 H5 构建 + 无头 Chrome 配合**本地 mock 接口**截图，检查了首页（头部、「正在进行」区块、观赛大厅入口）、消息页（已登录 / 未登录）、观赛大厅、「我的」页和确认页失效态的布局；另把小程序自定义 TabBar 的真实 `wxml` / `wxss` 渲染成静态页，检查亮 / 暗、选中、数字角标、99+、红点各状态。

局限：H5 不是发布目标，且 H5 构建会摇掉未被静态引用的 `uni.*`，因此 H5 里看不到原生 TabBar 角标，暗色主题也不生效（`uni.getStorageSync` 等在 H5 下报错，属已有行为）；mock 数据不代表真实后端。这些截图不能代替下面第 4 节的人工验收。

## 4. 未执行 / 需要人工确认

以下项目没有在本次自动化中执行（没有微信开发者工具、鸿蒙模拟器和真机可用），任务 1.6、9.2（设备部分）、9.3 保持未勾选：

微信小程序（开发者工具与真机）：

- [ ] 四个 Tab 切换、选中态、首次切到某个 Tab 时是否闪烁（每页各有一个 TabBar 实例，实例在首帧按用户偏好和系统主题取值）。
- [ ] 中间扫码按钮点击后直接打开相机；取消 / 无效码 / 对手码 / 裁判码四种结果；再来一局只接受原对手。
- [ ] 消息角标：未读数、99+、仅有待处理事项时的红点；退出登录、切换账号后立即消失。
- [ ] 亮 / 暗主题切换（含手动浅色而系统暗色的组合）。
- [ ] 「出示我的二维码」弹层打开时 TabBar 隐藏，关闭后恢复；弹层没有被 TabBar 遮挡。
- [ ] 各 Tab 页底部内容没有被 TabBar 遮挡（含 iPhone 底部安全区）；「我的」页浮动奖励条位置。
- [ ] 消息 Tab：首次进入请求次数、切回不重复请求、收到新通知后刷新。

鸿蒙（模拟器或真机）：

- [ ] `midButton` 的位置（赛讯与消息之间）、点击回调、样式和高度。这是本次最大的不确定项：HBuilderX 运行时源码已确认支持，但没有在设备上跑过。若不可用，按 design 的风险预案在首页头部放扫码入口，复用 `startQuickScan`。
- [ ] `setTabBarBadge` / `showTabBarRedDot` 在消息 Tab 上的显示；从非 Tab 页（例如计分页）返回后角标是否补齐。
- [ ] 暗色主题下 TabBar 与中间按钮；二维码弹层打开时隐藏 TabBar。
- [ ] 扫码相机权限与 `uni.scanCode` 在鸿蒙上的表现。

## 5. 其他备注

- 排行榜页的「发起PK」是实施中发现的第四个扫码入口（原先固定按排位开局），已一并改走统一流程，并同步了 OpenSpec 文档。
- `components/gameTypeModal.vue` 随最后一个使用方移除而删除。
- `scripts/check-notification-payloads.mjs` 在本次改动之前就已失效（引用了不存在的 `uniapp/backend/...` 路径），本次只把通知页路径改为 `pages/message/index.vue`，其余未处理。
- 小程序自定义 TabBar 需要基础库 2.5.0 以上（`getTabBar` 为 2.6.2 以上）；`manifest.json` 没有设置最低基础库，发布前请在小程序后台确认最低基础库版本不低于 2.6.2。
- 原生 iOS 目前仍是「首页 / 观赛 / 赛讯 / 我的」四个占位 Tab（`iOS/iOS/ContentView.swift`），不在本次范围；iOS 迁移时需要按新的主导航与消息 Tab 重新规划。
