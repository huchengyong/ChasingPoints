停下对「快照里页面内 `.task` 发网络请求为什么跑不完」的排查，不要再花时间在这上面，也不要改 `SnapshotSupport.swift`、`MockHTTPServer.swift`。

正确做法是另一条开发线已经在用的模式——**页面接受一个可选的、已经读好数据的模型**：

```swift
struct MyPage: View {
    @StateObject private var model: MyPageModel
    init(session: SessionStore, model: MyPageModel? = nil) {
        _model = StateObject(wrappedValue: model ?? MyPageModel(session: session))
    }
}
```

快照测试里先在测试函数中 `let model = MyPageModel(session: session, ...)`、`await model.onAppear()`（等两个资源都读完），再把这个 `model` 传给页面渲染。这样渲染时数据已经在模型里，不依赖视图里的 `.task`。`MatchQRCodeSheet` 的内容视图同理：接受一个已经处于 `ready` 状态的 `MatchQRCodeModel`。页面里的 `.task { await model.onAppear() }` 保留（因为有 60 秒新鲜度，渲染时不会重复请求）。

接下来请：

1. 删除探针文件 `iOS/iOSAuthTests/_ScratchSnapshotTaskProbe.swift` 以及排查过程中加的任何临时代码。
2. 按上面的模式完成 `MyPageSnapshotTests`（四组快照）并让 `SharedUISnapshotTests` 继续通过。
3. 对照原规格逐项检查还有哪些没做完（任务 0–4 和全部必须新增的测试），补齐。
4. 跑完整自检（Debug 构建、全部测试、Release 构建），按原规格的「交付」要求给出说明。

原规格的全部硬性约束继续有效。
