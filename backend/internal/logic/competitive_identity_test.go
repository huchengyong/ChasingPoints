package logic

import "testing"

func TestNormalizeOpponentIdentityPrefersStableRegisteredUserID(t *testing.T) {
	first := NormalizeOpponentIdentity(42, "旧昵称", 1)
	second := NormalizeOpponentIdentity(42, "新昵称", 2)
	if first != second || first.NameKey != "user:42" || first.Guest {
		t.Fatalf("registered identity must ignore nickname changes: first=%+v second=%+v", first, second)
	}
}

func TestNormalizeOpponentIdentityDefinesGuestNameAndBlankBoundaries(t *testing.T) {
	first := NormalizeOpponentIdentity(0, "  Guest\tPlayer ", 1)
	second := NormalizeOpponentIdentity(0, "guest player", 2)
	if first.NameKey != second.NameKey || !first.Guest {
		t.Fatalf("guest names should normalize case and whitespace: first=%+v second=%+v", first, second)
	}
	blankFirst := NormalizeOpponentIdentity(0, "", 10)
	blankSecond := NormalizeOpponentIdentity(0, " ", 11)
	if blankFirst.NameKey == blankSecond.NameKey {
		t.Fatalf("blank guest names must not merge across matches: first=%+v second=%+v", blankFirst, blankSecond)
	}
}
