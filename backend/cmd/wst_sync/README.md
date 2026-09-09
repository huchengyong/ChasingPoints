# WST 历史赛事同步

这个命令用于把 WST 官方历史赛事同步到本地数据库，并自动投放到赛讯。

同步链路：

- `players`
- `tournaments`
- `tournament_matches`
- `event_news_events`

## 命令入口

```bash
cd /Users/wisesearch/Projects/ChasingPoints/backend
set -a && source .env >/dev/null 2>&1
GOCACHE=/tmp/chasingpoints-gocache go run ./cmd/wst_sync --year 2025 --dry-run
```

## 支持的模式

三种模式互斥，只能传一组：

- `--season 2025`
- `--year 2025`
- `--from 2026-01-01 --to 2026-04-30`

### 历史补采模式（`--backfill`）

一次性补采历史数据，默认从 2023-01-01 到命令执行当天（UTC）：

```bash
# 预览补采（不写数据库）
go run ./cmd/wst_sync --backfill --dry-run

# 正式补采
go run ./cmd/wst_sync --backfill

# 指定范围
go run ./cmd/wst_sync --backfill --from 2023-01-01 --to 2023-12-31

# 只补采不发布赛讯
go run ./cmd/wst_sync --backfill --publish=false
```

补采模式特点：
- 与 `--season`、`--year` 互斥
- 仅支持斯诺克（`game-type=1`），其他球种会报错
- `--from` 和 `--to` 可独立覆盖，未指定的端点使用默认值
- 不会删除源站未返回的历史对阵（保留已有数据）
- 缺失比分的对阵会被跳过并报告
- 命中软删除记录时会明确失败，需人工处理后重跑
- dry-run 预览会输出拟发布/下架数量

### 退出码（仅补采模式）

补采模式使用独立的退出码供脚本识别：

| 退出码 | 状态 | 含义 |
|--------|------|------|
| 0 | completed | 无已发现异常；dry-run 仅完成预览，正式运行已成功提交 |
| 1 | failed | 请求/解析/分页/写入失败 |
| 2 | needs_review | 存在待核对项（缺失比分、无对阵赛事、提交后缓存刷新失败等）；是否已提交以摘要为准 |

摘要中的 `commit_state` 会区分 `not_started`、`rolled_back`、`committed` 和 `unknown`，并分别输出四类 `*_committed` 数量。`unknown` 表示 COMMIT 结果无法确认，数量也会显示为 `unknown`；应先按官方来源 ID 核对，再幂等重跑。

> **注意**：`go run` 可能将程序的非零退出码包装为退出码 1。需要区分 1/2 的脚本应先构建二进制：
> ```bash
> go build -o wst_sync ./cmd/wst_sync
> ./wst_sync --backfill
> echo "exit code: $?"
> ```

### 维护窗口

正式补采前应暂停所有实例的 WST 自动任务，使用单个补采进程，完成后恢复：

1. 暂停所有实例的 WST 自动任务（通过配置或部署）
2. 等待已运行的采集任务结束
3. 执行补采命令
4. 核对四类数据关联，抽查官方赛事/对阵/比分
5. 重跑同范围验证幂等
6. 恢复自动任务

> **重叠窗口提示**：恢复自动任务后，其查询窗口内的近期赛事仍会按原有自动同步清理语义处理，补采的保留策略不构成重叠窗口的永久保留承诺。

### 日志保存

```bash
# 保存日志并保留补采程序的真实退出码
go build -o wst_sync ./cmd/wst_sync
set -o pipefail
./wst_sync --backfill 2>&1 | tee backfill_$(date +%Y%m%d_%H%M%S).log
exit_code=${PIPESTATUS[0]}
echo "exit code: $exit_code"
```

### 失败重跑

补采失败后可直接重跑相同范围，不会重复增长数据（按来源 ID 幂等 upsert）。分页失败或请求限流时可稍后重试。

### 回退

代码回退不删除已导入的历史数据。若确需撤销，按运行前备份恢复。

常用附加参数：

- `--dry-run`
  只抓取和统计，不写数据库
- `--publish=false`
  写入事实表和赛讯壳，但赛讯不发布；命中已有发布赛讯会将其下架
- `--include-qualifiers=false`
  排除名字里带 qualifiers / qualifier / qualification / preliminary 的赛事
- `--game-type 1`
  指定写入赛事的球种，当前默认是 `1`，也就是斯诺克
- `-f etc/chasing_points-api.yaml`
  指定 go-zero 配置文件

## 示例

同步自然年 2025，并直接发布到赛讯：

```bash
cd /Users/wisesearch/Projects/ChasingPoints/backend
set -a && source .env >/dev/null 2>&1
GOCACHE=/tmp/chasingpoints-gocache go run ./cmd/wst_sync --year 2025
```

同步自然年 2024：

```bash
cd /Users/wisesearch/Projects/ChasingPoints/backend
set -a && source .env >/dev/null 2>&1
GOCACHE=/tmp/chasingpoints-gocache go run ./cmd/wst_sync --year 2024
```

同步 2026 年 1 月到 4 月：

```bash
cd /Users/wisesearch/Projects/ChasingPoints/backend
set -a && source .env >/dev/null 2>&1
GOCACHE=/tmp/chasingpoints-gocache go run ./cmd/wst_sync --from 2026-01-01 --to 2026-04-30
```

同步整个 2025/26 赛季，但先 dry run：

```bash
cd /Users/wisesearch/Projects/ChasingPoints/backend
set -a && source .env >/dev/null 2>&1
GOCACHE=/tmp/chasingpoints-gocache go run ./cmd/wst_sync --season 2025 --dry-run
```

## 行为说明

- `matches` 接口不依赖服务端过滤，而是分页全扫后本地按 `tournamentID` 过滤。
- 赛事、球员、比赛都按官方 source id 幂等 upsert。
- 同步器会优先从 WST 赛事页或票务页抓取 `og:image` 作为赛事封面，抓不到时才回退默认图。
- 历史同步默认会自动生成或更新 `event_news_events`，因此同步完成后会直接出现在赛讯里。
- 如果前置球员或赛事没有成功同步，比赛 upsert 会直接报错，不会静默写脏数据。
- 补采模式不会删除源站未返回的历史对阵，保留的旧对阵会列在摘要中。
- 赛事列表和赛季目录会遍历全部页，检测分页停滞和响应结构异常。
- 请求遇到 429/5xx 会自动重试（最多 3 次），遵守 Retry-After 头。
- 进程接收 SIGINT/SIGTERM 会取消进行中的 HTTP 请求和数据库事务。
