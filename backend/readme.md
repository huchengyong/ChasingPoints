# Backend

后端当前新增了一个 WST 历史赛事同步命令：

- [/Users/wisesearch/Projects/ChasingPoints/backend/cmd/wst_sync/README.md](/Users/wisesearch/Projects/ChasingPoints/backend/cmd/wst_sync/README.md)

用于按赛季、自然年或自定义时间窗同步官方历史赛事，并自动投放到赛讯。

同步时会优先从 WST 官方赛事页或票务页抓取 `og:image` 作为赛事封面，失败时再回退默认图。

## 开发环境部署

开发接口地址：<https://api-zhuifen.kekemate.cn>

本机需要先在 `~/.ssh/config` 配置 `chasingpoints-dev`，对应 Oracle Ubuntu 服务器，并确保该用户可以执行免密 `sudo`。私钥内容和服务端环境变量不得写入仓库。

```sshconfig
Host chasingpoints-dev
    HostName <Oracle 服务器 IP>
    User ubuntu
    Port 22
    IdentityFile <私钥绝对路径>
    IdentitiesOnly yes
```

发布后端：

```bash
cd backend
./deploy-dev.sh
```

脚本会依次运行后端测试、构建 Linux ARM64 二进制、上传版本、执行 Goose 迁移、切换 `current` 软链接、重启 `chasing-points.service` 并完成健康检查。服务端目录如下：

```text
/www/server/chasing-points/releases/<版本号>
/www/server/chasing-points/current
/www/server/chasing-points/shared/.env
```

部署失败时会恢复上一版程序；已经执行的数据库迁移不会自动回滚。

## GitHub Actions 发布

[`.github/workflows/deploy-backend.yml`](../.github/workflows/deploy-backend.yml) 在 GitHub Actions 中调用 `deploy-dev.sh` / `deploy-prod.sh`：在 Actions 页面选择 **Deploy backend → Run workflow**，选分支和环境后运行。

- `dev`：直接发布开发服务器，行为与本机执行 `deploy-dev.sh` 相同（包括 Goose 迁移）。
- `production`：只允许 `devlop` 或 `main` 分支。默认只预演（测试、编译、上传并列出提交清单与待执行迁移，线上不变）；核对后勾选“生产正式发布”再运行一次。有待执行迁移时会自动取消，迁移发布在本机人工执行 `deploy-prod.sh`。
- workflow 文件需要先合入默认分支 `main`，Actions 页面才会出现运行按钮。

在仓库 **Settings → Environments** 建 `dev` 和 `production` 两个环境，各自配置以下 secrets（仓库公开，运行日志所有人可见，地址等信息不要写进 workflow 或仓库）：

| Secret | 内容 |
| --- | --- |
| `SSH_HOST` | 服务器 IP |
| `SSH_USER` | 登录用户（开发服务器需免密 `sudo`） |
| `SSH_PRIVATE_KEY` | CI 专用部署私钥，不要复用个人密钥 |
| `SSH_KNOWN_HOSTS` | 服务器主机公钥，可在本机执行 `ssh-keygen -F <服务器 IP>` 取得 |

建议给 `production` 环境设置部署分支限制（`devlop`、`main`）。
