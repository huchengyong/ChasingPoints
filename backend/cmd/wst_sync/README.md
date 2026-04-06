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

常用附加参数：

- `--dry-run`
  只抓取和统计，不写数据库
- `--publish=false`
  写入事实表和赛讯壳，但赛讯不发布
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
