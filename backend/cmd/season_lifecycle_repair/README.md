# 连续赛季修复命令

首次启用连续赛季前，先在目标环境配置以下变量：

```text
SEASON_LIFECYCLE_ENABLED=true
SEASON_LIFECYCLE_ANCHOR_DATE=2026-08-01
SEASON_LIFECYCLE_INITIAL_NUMBER=1
SEASON_LIFECYCLE_CYCLE_MONTHS=1
SEASON_LIFECYCLE_TIMEZONE=Asia/Shanghai
```

锚点必须是业务时区中的月初。锚点之前的比赛和赛事继续只属于生涯统计，不会被虚构到历史赛季。

先执行 dry-run，核对策略、逐个计划窗口、待结算赛季、覆盖事件和冲突：

```bash
cd backend
go run ./cmd/season_lifecycle_repair --dry-run -f etc/chasing_points-api.yaml
```

确认输出后再执行正式修复：

```bash
cd backend
go run ./cmd/season_lifecycle_repair -f etc/chasing_points-api.yaml
```

输出中的 `window` 行会逐项列出赛季名称、日期、是否已存在、是否由本次创建、持久化 status 和结算状态；结尾汇总会分别给出预估值及本次实际写入的挑战快照、赛季记录和称号数量。发生中途失败时，命令会先输出已经完成的窗口和失败原因，再以非零状态退出。

正式修复会幂等补齐缺失窗口，并静默结算已经结束的窗口；它不会复制或改写原始比赛、段位日志、成就进度事件，也不会为历史换季创建未读通知或发送 Push/WebSocket 提醒。建议立即再次执行一次，确认没有重复新增资产。
