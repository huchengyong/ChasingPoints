## MODIFIED Requirements

### Requirement: App 前台恢复必须通过一次 Bootstrap 校准共享资源
登录后的冷启动、登录完成和 App 从后台恢复 SHALL 至多发起一次用户 Bootstrap，并以其结果更新用户资料、当前对局、通知角标、好友申请角标、换季提醒和竞技 revision。

#### Scenario: 从后台恢复后进入多个 Tab
- **WHEN** Bootstrap 已成功完成，随后用户依次进入首页、消息和“我的”
- **THEN** 三个页面 SHALL 复用 Bootstrap 中的当前对局、收到邀请数和角标
- **AND** 不得各自立即重复请求相同资源

#### Scenario: 后台期间错过 WebSocket
- **WHEN** 客户端断线期间服务端竞技 revision 已变化
- **THEN** 前台 Bootstrap SHALL 返回新 revision
- **AND** 对应竞技资源 Store SHALL 被标记 dirty 或重新加载

#### Scenario: Bootstrap 预约遇到 logout 或身份切换
- **WHEN** Bootstrap 预约的 in-flight Promise 尚未被响应，认证代次已变化或用户退出登录
- **THEN** Store SHALL 释放并 resolve 原预约等待者
- **AND** 新身份不得永久等待旧预约或接收旧 Bootstrap 回写
