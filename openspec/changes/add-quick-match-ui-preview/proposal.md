## Why

熟人约球 PRD V1.3 与低保真 W1.3 已收口，需要在改动应用前验证视觉层级与双人操作流程。静态可点击 HTML 能独立评审，不依赖 API 或 UniApp 构建。

## What Changes

- 在 `docs/previews/quick-match-ui/` 提供亮色高保真预览，遵循根级 DESIGN.md。
- 桌面并排展示两位用户，窄屏切换用户；场景控制放在手机界面外。
- 用明确标注的模拟数据演示发起、回应、等待、开始、结束、再约、设置、互邀和过期。
- 加入可执行检查与打开说明，不新增运行时依赖。

## Capabilities

### New Capabilities
- `quick-match-ui-preview`: 独立可打开、双人状态联动的约球视觉与交互评审原型。

### Modified Capabilities
无。现有应用业务要求不在本次实施范围。

## Impact

仅涉及静态预览、配套文档与本变更材料。不修改 app/backend/admin/website 的业务实现、接口、数据库或用户数据；不将模拟比赛结算视为正式功能实现。
