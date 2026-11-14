监工复核后的补充任务（仍遵守本会话开头的全部硬性约束与工程须知；不要做 git 写操作；只改 `iOS/iOS/**`、`iOS/iOSAuthTests/**`；并行线正在改的文件仍然不要碰）。

1. **对局链接的去向不对**：`uniapp/subPages/match/linkEntry.vue` 约 140 行对对局链接是 `uni.redirectTo({ url: buildSpectateRoute(parsed.matchId) })`，而 `entry-link.js` 的 `buildSpectateRoute` 是 `matchDetail?match_id=…&mode=spectate`（观战详情）。原生 `LinkEntryView.start()` 现在打开的是 `.liveMatch(matchId:)`，改成 `.publicMatchDetail(matchId:)`。把这段去向判断抽成可测的纯函数并加测试。

2. **会员海报模板**：`uniapp/utils/posterGenerator.js` 约 160–200 行与 350–470 行：分享者是有效会员（`resolveMemberIdentity({ vipLevel: my_vip_level, rankLevel: my_rank_level }).showBadge`）时用会员模板——`MEMBER_POSTER_THEMES[tier]` 的配色、分享者头像 + 段位头像框、VIP 标识、`MEMBER_HEADER_EXTRA` 的额外高度；对手是会员时两套模板都在对手一侧显示 VIP 标识。原生已有 `MemberIdentity.resolve(vipLevel:rankLevel:)`（`Shared/UI/MemberAvatarView.swift` 一带），直接复用其档位、徽标文案与头像框地址。
   - `ImageRenderer` 不能等远程图片：渲染前先**异步下载**分享者头像与头像框（`URLSession.shared.data(from:)`，单张超时 5 秒；失败就用默认头像 / 不画头像框，海报照样生成），把得到的 `UIImage` 传进海报视图。下载函数做成可注入的闭包，测试里不联网。
   - 会员模板的配色常量逐个照搬 `MEMBER_POSTER_THEMES`（允许在海报文件里用十六进制颜色，这是海报专用色，和 PK 报表海报的做法一致）。
   - 测试：会员 / 非会员选模板、三档配色、对手会员标识、头像下载失败仍能生成海报。快照补 `share-result-poster-member`（亮暗各一张，注入本地生成的头像图片）。

3. **删除无用代码**：`EntryFunnelRules.postLoginNavigation(pageCount:)` 与 `PostLoginNavigation` 没有任何调用方（你在交付说明里也提到原生登录页 `dismiss()` 已经等价），连同对应测试删掉，在交付说明里写明原因。

完成后照常跑 Debug / Release 构建与全量测试；交付说明只写本次补充的部分，并贴出 `Executed N tests` 那一行。快照导出放到 `iOS/build/` 下。
