package logic

// GetGameTypeName 获取游戏类型名称（公共方法，供各logic使用）
func GetGameTypeName(gameType int) string {
	switch gameType {
	case 1:
		return "斯诺克"
	case 2:
		return "九球追分"
	case 3:
		return "中式八球"
	case 4:
		return "美式九球"
	default:
		return "未知"
	}
}
