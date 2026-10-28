package auth

import (
	"regexp"
	"strings"
	"testing"
	"unicode/utf8"
)

var defaultNicknamePattern = regexp.MustCompile(
	`^(?:` + strings.Join(defaultNicknameTraits, "|") + `)的(?:` + strings.Join(defaultNicknameAnimals, "|") + `)$`,
)

func TestRandomDefaultNicknameCombinesTraitAndAnimal(t *testing.T) {
	tests := []struct {
		name   string
		values []int
		want   string
	}{
		{name: "首个组合", values: []int{0, 0}, want: "爱喝汽水的猫"},
		{name: "末个组合", values: []int{len(defaultNicknameTraits) - 1, len(defaultNicknameAnimals) - 1}, want: "上班摸鱼的猫头鹰"},
		{name: "中间组合", values: []int{2, 2}, want: "会走位的柴犬"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			index := 0
			got := randomDefaultNicknameWith(func(int) int {
				value := tt.values[index]
				index++
				return value
			})
			if got != tt.want {
				t.Fatalf("random nickname = %q, want %q", got, tt.want)
			}
		})
	}
}

// 默认昵称须满足改昵称接口的 2-12 字符限制。
func TestDefaultNicknameWordsFitNicknameLength(t *testing.T) {
	for _, trait := range defaultNicknameTraits {
		for _, animal := range defaultNicknameAnimals {
			nickname := trait + "的" + animal
			if n := utf8.RuneCountInString(nickname); n < 2 || n > 12 {
				t.Fatalf("nickname %q has %d runes", nickname, n)
			}
		}
	}
}
