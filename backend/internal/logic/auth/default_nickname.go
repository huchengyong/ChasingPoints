package auth

import "math/rand/v2"

// 默认昵称形如「会走位的柴犬」：特点 + 的 + 小动物，任意组合都要读得通顺。
var (
	defaultNicknameTraits = []string{
		"爱喝汽水", "爱吃橘子", "会走位", "会翻袋", "会清台", "会防守", "一杆清",
		"喜欢黑八", "喜欢斯诺克", "爱打九球", "刚学打球", "周末打球", "下班打球",
		"熬夜打球", "每天练球", "偷偷练球", "爱逛球房", "爱擦巧克粉", "认真瞄准",
		"眼神很准", "打球很稳", "手感火热", "偶尔进球", "总差一球", "想赢一局",
		"不服输", "很有耐心", "运气很好", "有点紧张", "慢吞吞", "元气满满",
		"爱笑", "晒太阳", "爱睡懒觉", "不想下班", "上班摸鱼",
	}
	defaultNicknameAnimals = []string{
		"猫", "橘猫", "柴犬", "柯基", "熊", "熊猫", "企鹅", "水獭", "狐狸", "海豹",
		"兔子", "松鼠", "考拉", "小鹿", "刺猬", "仓鼠", "浣熊", "鲸鱼", "树懒", "猫头鹰",
	}
)

func randomDefaultNickname() string {
	return randomDefaultNicknameWith(rand.IntN)
}

func randomDefaultNicknameWith(randomInt func(int) int) string {
	trait := defaultNicknameTraits[randomInt(len(defaultNicknameTraits))]
	animal := defaultNicknameAnimals[randomInt(len(defaultNicknameAnimals))]
	return trait + "的" + animal
}
