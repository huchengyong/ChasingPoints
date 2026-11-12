# 赛讯赛程中文化与状态筛选验收记录

日期：2026-11-12。范围：UniApp 赛讯列表页与赛事详情页（`pages/tournament/index.vue`、`subPages/tournament/detail.vue`、`utils/saixun.js`）。后端、接口与表结构无变更，无迁移。原生 iOS 没有赛讯模块，不涉及。

## 1. 改动

| # | 用户反馈 | 处理 |
| --- | --- | --- |
| 1 | 列表里两场即将开始的官方赛事没有封面 | 两场均为 WST 资格赛（德国大师赛 2027 资格赛、英国锦标赛 2026 资格赛）。后端同步把 WST 官网默认封面视为"无封面"，库里为空。客户端对 `source_type=official` 且无封面的赛事改用 WST 官网同款默认图，列表卡片与详情头图一致；图片放在 `cdn-assets/static/images/wst-default-event-cover.png`，由用户上传七牛 |
| 2 | 赛程赛果有英文，如 `Group Semi Finals` | 小组半决赛 / 小组决赛；并按生产全量轮次名补齐 `League Phase`、`Pre-Qualifier (Round)`、`Wildcard Round`、`Round N/Heldover`。占位对阵 `Winner of Match N`、`Winner Semi One` 译为"第N场胜者""半决赛1胜者"。真实球员名保持英文 |
| 3 | 详情页最前面是已结束的比赛，想看接下来的赛程 | 吸顶轮次栏下新增"全部 / 进行中 / 未开始 / 已结束"筛选，带场数；数量为 0 的状态不显示，轮次只有一种状态时隐藏；切换轮次保留所选状态，新轮次没有该状态时按"全部"；已取消场次归入"已结束" |

轮次名来源：在生产服务器上逐个请求 `/api/event-news/view`（event_id 1–59），汇总全部 `round_name` 与占位球员名后逐一核对。

## 2. 本次自动化执行

- `uniapp/`：`node --test tests/*.test.mjs` 1212 个，1211 通过。失败的是 `cdn-asset.test.mjs`：本机 `cdn-assets/static/images/` 下有 Finder 生成的 `.DS_Store`（已被 git 忽略），临时移走后该文件 3 个用例全部通过。

## 3. 渲染验证（微信开发者工具）

- 开发模式构建实际请求生产接口，用生产冠中冠 2026（event_id 46）核对：轮次标签为"小组半决赛 / 小组决赛 / 半决赛 / 决赛"；筛选显示"全部 8 / 未开始 2 / 已结束 6"；点"未开始"只剩 11 月 13 日两场；切到"小组决赛"后筛选保留，显示"第7场胜者 vs 第8场胜者"；列表卡片轮次标签为"小组半决赛"。浅色深色均截图检查。
- 两张无封面卡片的 `src` 为 `https://cdn.zhuifen.cn/static/images/wst-default-event-cover.png`。本机与生产服务器访问 `cdn.zhuifen.cn` 均返回 403（原有默认封面同样如此），模拟器内封面图片也未加载，封面显示效果未目测确认。

## 4. 未执行

- 真机与鸿蒙。
- 新封面上线后的实际显示（用户已上传七牛，待真机确认）。

## 5. 发布记录

- 小程序：版本号升为 1.0.6，生产构建（产物无开发域名，appid 正确）后上传开发版本，主包 1.48 MB、总包 3.03 MB；未设为体验版，未提审。
