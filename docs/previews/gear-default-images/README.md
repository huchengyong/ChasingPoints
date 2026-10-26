# 装备默认图与风格说明

> 2026-10-25 讨论「装备」功能时产出，对应 OpenSpec 变更 [add-gear-management](../../../openspec/changes/add-gear-management/design.md)。第一期所有装备图片都用这套默认图；以后每个型号的目录图按本文风格生成，保证整个目录看起来像同一套游戏道具。

预览：在本目录用浏览器打开 [index.html](index.html)，会把每张图放在浅色、深色和透明棋盘格上对比。

## 文件

| 文件 | 用途 |
| --- | --- |
| `gear-cue-snooker-default.png` | 球杆，斯诺克 / 中八风格：白蜡木杆身、乌木后把、四尖接木、铜接头、铭牌 |
| `gear-cue-pool-default.png` | 球杆，美式风格：枫木杆身、白箍、麻绳缠把、前把尖角 |
| `gear-break-cue-default.png` | 冲杆：酚醛硬杆头、运动握把、铜色前把 |
| `gear-jump-cue-default.png` | 跳杆：短杆、酚醛硬杆头 |
| `gear-house-cue-default.png` | 球房公杆：一体杆、素色后把、重量贴纸 |
| `gear-tip-default.png` | 皮头 |
| `gear-chalk-default.png` | 巧克 |

每个 PNG 都有同名 SVG 源文件。

选图规则（建议）：

- 球杆按目录里该型号的杆型选：斯诺克杆、中式杆用 `snooker`，美式杆用 `pool`。
- 同时是冲杆和跳杆的开跳杆，用冲杆图。

## 构图

- 画布 1024×1024 正方形，PNG 透明背景。只有装备本身：不要背景、投影、倒影、地面、手、球台、文字或水印。
- 球杆从左下到右上斜放，杆头朝右上，与竖直方向成 25°（与水平成 65°）。
- 杆身中心对准画布中心。全长杆两端离画布边缘约 50px。跳杆等短杆按真实长度比例缩短，不要放大到满幅。
- 皮头、巧克等小件正放，3/4 俯视，居中，约占画布宽度的 55%。

## 比例

- 杆身风格化加粗到真实比例的 2 倍左右，后把直径约为杆长的 1/18。按真实比例画，杆在列表里只剩一条线。
- 即使加粗，缩到 120px 以下也只能看出轮廓和配色。列表里要用小图时，另裁一张后把特写，或者只显示类别图标加文字。

## 光照与上色

- 光源在左上方。
- 硬边分层上色（cel shading），不用写实渐变：底色、一条宽亮带、一条细高光线、下沿一条暗带。
- 描边：外轮廓用暖深棕 `#2A1A0C`，约 5px（按 1024 画布）；部件之间的分段线约 2.4px。
- 外缘：描边外再加一圈约 4px 的浅暖色（`#FFF4D8`，不透明度 42%）。浅色背景上几乎看不见，深色模式下能托住乌木、黑色后把这类深色杆身。
- 配色以真实材质为准：白蜡木浅金、枫木奶白、乌木深棕、黄铜金。金属件靠近品牌金 `#E0AE12`。巧克保留传统蓝色。
- 不要照片质感、金属反射贴图或发光特效。

## 型号图必须保留的识别细节

- 木纹：白蜡木的箭头纹、枫木的细直纹、乌木的暗纹。
- 接木：尖角数量和长度，镶线的颜色与层数。
- 铭牌：位置、形状、材质色。字样不需要清晰可读，生成的文字通常会乱码，宁可留空。
- 接头环、套环、缠把（麻绳纹、皮革或无缠）、后把镶嵌（菱形、圆点、图案）。
- 杆头：皮头颜色（棕或黑）、箍的颜色（白或铜）。冲杆、跳杆的酚醛杆头画成浅灰白硬头。

## 生成提示词模板

把方括号换成具体型号的信息。

中文：

```text
一根台球杆的游戏道具插画，[品牌 型号]，[斯诺克杆 / 美式杆 / 冲杆 / 跳杆]。
透明背景，画面里只有球杆，没有背景和投影。球杆从左下到右上斜放，与竖直方向成 25°，
杆头朝右上，居中，几乎占满画面对角线。卡通渲染，硬边明暗分层，光源在左上方，暖深棕色描边。
保留这款杆的特征：[木纹]，[接木尖角数量与镶线颜色]，[铭牌位置与形状]，[接头与缠把]。
杆身比真实比例略粗。不要照片质感，不要文字和水印，不要手和球台。
```

English:

```text
Game item illustration of a single billiard cue, [brand model], [snooker cue / pool cue / break cue / jump cue].
Transparent background, the cue only, no backdrop and no shadow. The cue runs diagonally from bottom-left
to top-right, tilted 25° from vertical, tip at the top right, centred, spanning almost the full diagonal.
Cel-shaded cartoon style with hard-edged light and shadow bands, light from the upper left, warm dark-brown outline.
Keep the model's signature details: [grain], [number of splice points and veneer colours],
[nameplate position and shape], [joint and wrap]. Slightly thicker than real proportions.
No photorealism, no text or watermark, no hands, no table.
```

很多生成工具给不出真正的透明背景。可以先生成纯色背景再抠图，抠完放到深色背景上检查边缘有没有残留的白边。

## 修改与重新导出

- 角度：SVG 外层 `<g>` 的 `rotate(-65)` 是与水平夹角取负，对应与竖直方向成 25°。
- 导出 PNG（在本目录执行，`<chrome>` 换成本机 Chrome 可执行文件）：

```bash
<chrome> --headless=new --hide-scrollbars --default-background-color=00000000 \
  --window-size=1024,1024 --screenshot=gear-cue-pool-default.png "file://$PWD/gear-cue-pool-default.svg"
```
