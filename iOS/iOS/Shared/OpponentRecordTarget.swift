import Foundation

/// 好友主页（或他人战绩）的目标身份；`nil` 表示查看自己的过往对手。
struct OpponentRecordTarget: Hashable {
    let userId: Int64
    let name: String
    let avatar: String
}
