# Backend

后端当前新增了一个 WST 历史赛事同步命令：

- [/Users/wisesearch/Projects/ChasingPoints/backend/cmd/wst_sync/README.md](/Users/wisesearch/Projects/ChasingPoints/backend/cmd/wst_sync/README.md)

用于按赛季、自然年或自定义时间窗同步官方历史赛事，并自动投放到赛讯。

同步时会优先从 WST 官方赛事页或票务页抓取 `og:image` 作为赛事封面，失败时再回退默认图。

## 开发环境部署

开发接口地址：<https://dev-api.kekemate.cn>

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
