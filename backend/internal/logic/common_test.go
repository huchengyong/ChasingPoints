package logic

import "testing"

func TestGetGameTypeName(t *testing.T) {
	cases := map[int]string{
		1: "斯诺克",
		2: "九球追分",
		3: "中式八球",
		4: "美式九球",
	}

	for gameType, expected := range cases {
		if got := GetGameTypeName(gameType); got != expected {
			t.Fatalf("GetGameTypeName(%d) = %q, want %q", gameType, got, expected)
		}
	}
}
