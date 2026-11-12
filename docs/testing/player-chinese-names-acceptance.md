# 赛讯球员中文名验收记录

日期：2026-11-12。范围：后端 `players.name_zh` 与赛讯读取、WST 同步保留中文名、管理后台「赛讯球员」页、UniApp 赛事详情的中英文名展示。原生 iOS 没有赛讯模块，不涉及。上一轮见 [赛讯赛程中文化与状态筛选验收记录](saixun-schedule-localization-acceptance.md)。

## 1. 改动

- 迁移 `20261112170000_add_player_name_zh`：`players` 增加 `name_zh VARCHAR(64) NOT NULL DEFAULT ''`，先查 `information_schema` 再加列，不指定字符集。
- 迁移 `20261112170100_seed_player_name_zh`：首批 135 名近期赛事球员（生产 event_id 40 起）。中国内地与港澳球员用本名，其他球员用「名·姓」音译（用户确认用全名，避免 Wilson、Robertson、Jones 等同姓混淆）。只写 `name_zh = ''` 的行，按 `display_name` 或 `first_name + last_name` 匹配；Down 不回滚数据。未能核实汉字的 Gao Yang、Zhao Hanyang、Liu Yang、Li Jun、Deng Haohui 留空，需在后台补填。
- 赛讯详情返回 `home/away_player_name_zh`；列表与详情的 `latest_result_text` 有中文名时用中文名。列表只为当前页的摘要比赛批量查一次球员。
- WST 同步整行保存球员时保留 `name_zh`（修复前每次同步都会清空）。
- 管理端 `GET /api/admin/event-news/players`（关键词匹配中英文名并转义通配符、只看未填写、`page_size` 上限 100）与 `POST /api/admin/event-news/player/update`（去首尾空白、32 字上限、拒绝控制字符、可清空；成功后刷新赛讯缓存版本）。后台新增菜单「赛讯球员」，表格内直接编辑、回车保存。
- UniApp：比赛卡片与冠军卡片主名用中文名、英文名小字副行；没有中文名时只显示英文名。

## 2. 本次自动化执行

- `backend/`：`go test ./...` 通过。新增：赛讯详情返回中文名、列表摘要用中文名且 1 场与 5 场赛讯都只查 1 次 players、同步保留中文名、管理端列表筛选 / 通配符转义 / 校验 / 404 / 清空 / 缓存版本 / 未登录、迁移内容检查（无 `IF NOT EXISTS`、无字符集）。
- 本机临时 MySQL 8.0（:33099，结束后已关闭并删除）：全量 `goose up` 后插入样例球员再执行回填，结果为按 `display_name` 命中、`display_name` 为空时按姓名拼接命中、已有中文名不覆盖、未核实球员留空、非官方球员不动；两个迁移的 Up 段可重复执行；`down` 两次后列被删除，再 `up` 正常。
- `admin/`：`node --experimental-strip-types --test tests/*.test.mjs` 19 个全部通过；`npm run build`（含 vue-tsc）通过。
- `uniapp/`：`node --test tests/*.test.mjs` 1213 个，1212 通过；失败的 `cdn-asset.test.mjs` 是本机 `cdn-assets/static/images/.DS_Store`（git 已忽略），与本次无关。

## 3. 发布与线上核对

- 开发环境后端：发布号 `20261112170257-891bdc317100`，两个迁移执行成功。`/api/event-news/view?event_id=167`（冠中冠）15 场中 25/30 个球员位有中文名，其余 5 个是「Winner of Match N」「Winner Semi One」等占位；列表摘要为中文名；管理端球员接口未登录返回 401。
- 开发环境管理后台：版本 `20261112170353-891bdc317100`。
- 生产后端：`devlop` 合入 `main`（`dcd5755`）后发布，发布号 `20261112171729-dcd5755c3d19`。首次预演在上传发布包时卡住 10 分钟（生产 SSH 不稳定），停止后改用 SSH 复用连接重新预演，提交清单只有 `891bdc3`、待执行迁移 2 个；正式发布前备份数据库，迁移成功，本机与公网健康检查 200，启动后应用日志 error=0。
- 生产核对（在服务器上请求）：`event_id=46`（冠中冠）25/30 个球员位有中文名，其余 5 个为占位；列表摘要「赵心童 - 特普查亚·乌诺」「米哈尔·舒巴尔契克 - 菲尔·奥凯恩」；管理端球员接口未登录返回 401。
- 生产管理后台：版本 `20261112171942-dcd5755c3d19`。
- 微信开发者工具（开发构建实际请求生产接口）：冠中冠详情显示「赵心童 / Zhao Xintong」「特普查亚·乌诺 / Thepchaiya Un-Nooh」，占位对阵为「第7场胜者」且无副名；国际锦标赛 2026 冠军卡片为「凯伦·威尔逊 / Kyren Wilson / 决赛 10-8 胜 约翰·希金斯」；列表卡片为「赵心童 vs 特普查亚·乌诺」。浅色深色均截图检查。
- 小程序：版本号升为 1.0.8，生产构建（产物无开发域名，appid 正确）后上传开发版本，主包 1.48 MB、总包 3.03 MB；未设为体验版，未提审。

## 4. 未执行

- 管理后台「赛讯球员」页没有登录后实际操作（没有使用管理员账号），只核对了构建产物与接口鉴权。
- 真机与鸿蒙。
- 135 个音译名中较冷门的外国球员与业余选手为自拟音译，需在后台按需修正；5 名未核实汉字的中国球员待补填。
