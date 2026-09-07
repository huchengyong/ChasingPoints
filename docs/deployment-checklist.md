# 追分部署上线清单

> 每次生产发布复制本清单并逐项确认。不适用项标记为 `N/A` 并说明原因；任何阻断项未通过时停止上线。

## 发布信息

- [ ] 发布负责人、时间窗口和受影响子系统已确认：`app` / `backend` / `admin` / `website`
- [ ] 发布分支、提交 SHA、版本号和回滚负责人已记录
- [ ] 工作树干净，发布内容只包含已评审变更
- [ ] 上一个可用后端版本、前端产物和数据库备份可恢复

## 阻断项

- [ ] 所有相关测试与构建通过
- [ ] MySQL 已备份，待执行迁移已评审
- [ ] Redis 已安装、运行并设置开机启动，后端运行环境执行 `PING` 返回 `PONG`
- [ ] 生产密钥不为空、不使用样例值，且未进入仓库、构建产物或日志
- [ ] HTTPS/WSS 证书、DNS、反向代理和微信合法域名已生效
- [ ] 回滚目标和操作步骤已在上线前验证

## 自动化验证

```bash
cd app && node --test tests/*.test.mjs
cd backend && go test ./...
cd admin && npm ci && npm run build
cd website && npm ci && npm run test && npm run build
```

- [ ] 命令全部以退出码 `0` 完成，警告已确认不会影响生产
- [ ] 若修改 `backend/chasing_points.api`，已立即运行 goctl，并确认没有新增 `todo` 空壳 logic
- [ ] 若修改数据库字段，迁移、Gorm 模型、API 字段和前端字段已一起核对

## 基础设施与环境变量

- [ ] `APP_ENV=production`，服务端口和时区正确
- [ ] `MYSQL_DSN` 指向生产库，账号权限遵循最小权限
- [ ] `REDIS_HOST`、`REDIS_PORT`、`REDIS_PASSWORD` 与实际 Redis 一致
- [ ] Redis 未暴露到公网；如仅供同机后端使用，只监听 `127.0.0.1` / `::1` 并开启 protected mode
- [ ] JWT、微信小程序、短信、七牛及已启用支付/OAuth 服务的密钥已配置
- [ ] `MATCH_INVITE_SIGNING_SECRET` 已配置；生产启动安全校验通过
- [ ] HTTP/WebSocket allowed origins 只包含正式 HTTPS 域名，不包含 `*`
- [ ] 合规开关已明确：未获准功能继续保持 restricted/disabled
- [ ] `CompetitiveReadModel.ReadMode` 仅在回建审计 `differences=0` 后设为 `enabled`

Redis 验证示例：

```bash
systemctl is-active redis-server
systemctl is-enabled redis-server
REDISCLI_AUTH="$REDIS_PASSWORD" redis-cli -h "$REDIS_HOST" -p "$REDIS_PORT" PING
```

## 数据库迁移

- [ ] 已检查当前版本和全部待执行迁移
- [ ] 已确认迁移兼容目标 MySQL；已有表补字段不依赖 `ADD COLUMN IF NOT EXISTS`
- [ ] 大表 DDL 已评估锁表时间并避开业务高峰
- [ ] 已在兼容环境演练迁移，确认可重复执行或有明确失败恢复方式
- [ ] 应用切换前执行迁移并核对最终版本
- [ ] 数据库迁移不会随应用回滚自动撤销；如需回滚字段，先回滚到不读取该字段的应用版本

## 后端发布

- [ ] 构建目标与生产服务器架构一致，发布包包含二进制、配置和完整 migrations
- [ ] 使用独立 release 目录并保留上一版本；切换通过 `current` 软链接或等价原子方式完成
- [ ] 服务以非 root 用户运行，环境文件和密钥权限正确
- [ ] `chasing-points.service` 重启成功且保持 `active`
- [ ] `GET /api/public/rank/configs` 返回 `200`
- [ ] 无 token 请求受保护接口返回 `401`，不会返回业务数据
- [ ] 有效 access token 请求 `POST /api/user/ws-ticket` 返回 `200`，而非 `503`
- [ ] `/api/user/ws`、`/api/match/ws` 均能通过 WSS 完成握手和消息收发
- [ ] 发布日志无持续数据库、Redis、panic 或 5xx 错误

> `backend/deploy-dev.sh` 只用于开发服务器，不直接作为生产部署命令。

## 微信小程序与 App

- [ ] 生产构建使用 `https://api.zhuifen.cn` 和 `wss://ws.zhuifen.cn`，产物中不含开发 API 域名
- [ ] 微信公众平台已配置 request、socket、uploadFile、downloadFile 合法域名
- [ ] 微信开发者工具开启合法域名校验后完成验收，不能以“关闭校验”作为通过依据
- [ ] 小程序 AppID、隐私保护指引、位置权限声明和版本号正确
- [ ] 登录/绑定手机、“我的”页 UserWS、创建对局、计分和结束对局已冒烟验证
- [ ] iOS/Android 版本号、签名、Universal Links 和商店资料已核对（如本次发布原生 App）

## 管理后台

- [ ] `admin/.env.production` 的 `VITE_API_BASE_URL` 指向正式 API
- [ ] 生产构建产物已部署，HTTPS 和 SPA history fallback 正常
- [ ] 管理员登录、首页统计和至少一个核心管理列表可用
- [ ] 首次管理员初始化完成后，已轮换或移除 `ADMIN_SETUP_TOKEN`

## 官网

- [ ] `NUXT_PUBLIC_SITE_URL=https://www.zhuifen.cn`
- [ ] `NUXT_PUBLIC_API_BASE_URL` 指向正式 API
- [ ] iOS/Android 下载链接和二维码已替换，不再指向 coming-soon（正式开放下载时）
- [ ] 联系邮箱、电话、ICP备案和公安备案信息已确认
- [ ] 隐私政策、用户协议内容及更新日期已完成上线审核
- [ ] canonical、OG URL、sitemap、公开页面和反馈提交均正常

## 上线后验收

- [ ] 官网、管理后台、公开 API 和小程序首屏可访问
- [ ] 微信登录/手机号登录、刷新 token 和退出登录正常
- [ ] UserWS、MatchWS 连接稳定，无持续重连或 ticket 503
- [ ] 图片上传和 CDN 访问正常
- [ ] 通知、好友申请和比赛开始消息正常
- [ ] 已启用的支付、OAuth、短信和 Push 各完成一次受控验收
- [ ] 观察窗口内 5xx、慢 SQL、Redis error、WebSocket 断连和资源使用无异常

## 回滚

- [ ] 停止继续发布并记录异常开始时间、请求 ID 和受影响功能
- [ ] 后端切回上一 release，重启服务并重跑核心健康检查
- [ ] 管理后台、官网和客户端配置按各自上一产物恢复
- [ ] 不自动回滚已执行迁移；确认旧应用与当前 schema 兼容后再恢复流量
- [ ] 回滚后再次验证登录、WS ticket、UserWS、MatchWS 和核心对局链路
- [ ] 保存脱敏日志与复盘结论，密钥、token 和用户数据不得进入工单

## 完成确认

- [ ] 发布负责人确认上线完成
- [ ] 监控负责人确认指标稳定
- [ ] 回滚负责人确认回滚窗口结束
