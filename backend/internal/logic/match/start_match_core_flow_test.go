package match

import (
	"testing"

	"chasing_points/internal/model"
	"chasing_points/internal/types"
)

func TestNormalizeStartMatchOptionsKeepsLegacyRankedPublicDefaults(t *testing.T) {
	req := &types.StartMatchReq{}
	if message := normalizeStartMatchOptions(req); message != "" {
		t.Fatalf("normalize legacy request: %s", message)
	}
	if req.MatchMode != model.MatchModeRanked || req.Visibility != model.MatchVisibilityPublic {
		t.Fatalf("expected ranked/public defaults, got mode=%q visibility=%q", req.MatchMode, req.Visibility)
	}
}

func TestNormalizeStartMatchOptionsRejectsInvalidModeVisibilityPairs(t *testing.T) {
	practicePublic := &types.StartMatchReq{MatchMode: model.MatchModePractice, Visibility: model.MatchVisibilityPublic}
	if message := normalizeStartMatchOptions(practicePublic); message != "" {
		t.Fatalf("practice public should be allowed: %s", message)
	}

	rankedPrivate := &types.StartMatchReq{MatchMode: model.MatchModeRanked, Visibility: model.MatchVisibilityPrivate}
	if message := normalizeStartMatchOptions(rankedPrivate); message == "" {
		t.Fatal("expected ranked private to be rejected")
	}

	invalidMode := &types.StartMatchReq{MatchMode: "friendly"}
	if message := normalizeStartMatchOptions(invalidMode); message == "" {
		t.Fatal("expected invalid mode to be rejected")
	}
}

func TestChallengeMatchesUsersRequiresBothParticipants(t *testing.T) {
	challenge := &model.Challenge{FromUserId: 1001, ToUserId: 2002}
	if !challengeMatchesUsers(challenge, 1001, 2002) || !challengeMatchesUsers(challenge, 2002, 1001) {
		t.Fatal("expected challenge to match both directions")
	}
	if challengeMatchesUsers(challenge, 1001, 3003) {
		t.Fatal("unexpected challenge match for unrelated opponent")
	}
}
