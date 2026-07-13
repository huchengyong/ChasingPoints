# Backend

后端当前新增了一个 WST 历史赛事同步命令：

- [/Users/wisesearch/Projects/ChasingPoints/backend/cmd/wst_sync/README.md](/Users/wisesearch/Projects/ChasingPoints/backend/cmd/wst_sync/README.md)

用于按赛季、自然年或自定义时间窗同步官方历史赛事，并自动投放到赛讯。

同步时会优先从 WST 官方赛事页或票务页抓取 `og:image` 作为赛事封面，失败时再回退默认图。
