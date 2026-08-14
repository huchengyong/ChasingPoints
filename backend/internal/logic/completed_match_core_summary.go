package logic

import (
	"context"
	"fmt"
	"time"

	"chasing_points/internal/model"
	"chasing_points/internal/observability"
	"chasing_points/internal/readcache"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"

	"github.com/redis/go-redis/v9"
	"golang.org/x/sync/singleflight"
)

const completedMatchCoreCacheTTL = 24 * time.Hour

var (
	completedMatchCoreCacheMetrics observability.CacheMetrics
	completedMatchCoreCacheGroup   singleflight.Group
)

// CompletedMatchCoreSummary contains only immutable, match-scoped result data.
// Current user aggregates intentionally stay outside this structure.
type CompletedMatchCoreSummary struct {
	Match        *model.Match
	Player1      *model.User
	Player2      *model.User
	Referee      *model.User
	Rounds       []model.MatchRound
	Actions      []model.MatchAction
	Achievements types.MatchAchievement
	RankChanges  []model.RankChangeLog
}

type completedMatchCoreCachePayload struct {
	Match        model.Match            `json:"match"`
	Rounds       []model.MatchRound     `json:"rounds"`
	Actions      []model.MatchAction    `json:"actions"`
	Achievements types.MatchAchievement `json:"achievements"`
	RankChanges  []model.RankChangeLog  `json:"rank_changes"`
}

type CompletedMatchViewerPerspective struct {
	IsPlayer1      bool
	MyUserId       int64
	OpponentUserId int64
	MyScore        int
	OpponentScore  int
	MyName         string
	OpponentName   string
	MyAvatar       string
	OpponentAvatar string
}

func (s *CompletedMatchCoreSummary) ViewerPerspective(viewerUserId int64) CompletedMatchViewerPerspective {
	if s == nil || s.Match == nil {
		return CompletedMatchViewerPerspective{}
	}
	player1Name, player1Avatar := "玩家1", ""
	if s.Player1 != nil {
		player1Name, player1Avatar = s.Player1.Nickname, s.Player1.Avatar
	}
	player2Id := int64(0)
	player2Name, player2Avatar := s.Match.OpponentName, ""
	if s.Match.OpponentId != nil {
		player2Id = *s.Match.OpponentId
	}
	if s.Player2 != nil {
		player2Name, player2Avatar = s.Player2.Nickname, s.Player2.Avatar
	}
	if player2Id == viewerUserId {
		return CompletedMatchViewerPerspective{
			MyUserId: player2Id, OpponentUserId: s.Match.UserId,
			MyScore: s.Match.OpponentScore, OpponentScore: s.Match.MyScore,
			MyName: player2Name, OpponentName: player1Name,
			MyAvatar: player2Avatar, OpponentAvatar: player1Avatar,
		}
	}
	return CompletedMatchViewerPerspective{
		IsPlayer1: true, MyUserId: s.Match.UserId, OpponentUserId: player2Id,
		MyScore: s.Match.MyScore, OpponentScore: s.Match.OpponentScore,
		MyName: player1Name, OpponentName: player2Name,
		MyAvatar: player1Avatar, OpponentAvatar: player2Avatar,
	}
}

func BuildCompletedMatchCoreSummary(ctx context.Context, svcCtx *svc.ServiceContext, match *model.Match) (*CompletedMatchCoreSummary, error) {
	if svcCtx == nil || svcCtx.MatchModel == nil || match == nil || match.Status != 2 {
		return nil, fmt.Errorf("completed match core is unavailable")
	}
	var payload completedMatchCoreCachePayload
	if match.AchievementSyncedAt != nil {
		var client *redis.Client
		if svcCtx != nil {
			client = svcCtx.Redis
		}
		cached, err := readcache.LoadJSON(ctx, client, fmt.Sprintf("completed-match-core:v1:%d", match.Id), completedMatchCoreCacheTTL, &completedMatchCoreCacheMetrics, &completedMatchCoreCacheGroup, func() (completedMatchCoreCachePayload, error) {
			return loadCompletedMatchCorePayload(svcCtx, match)
		})
		if err == nil {
			payload = cached
		} else {
			payload = completedMatchCoreBestEffort(svcCtx, match)
		}
	} else {
		payload = completedMatchCoreBestEffort(svcCtx, match)
	}
	core := &CompletedMatchCoreSummary{
		Match: &payload.Match, Rounds: payload.Rounds, Actions: payload.Actions,
		Achievements: payload.Achievements, RankChanges: payload.RankChanges,
	}
	hydrateCompletedMatchUsers(svcCtx, core)
	return core, nil
}

func loadCompletedMatchCorePayload(svcCtx *svc.ServiceContext, match *model.Match) (completedMatchCoreCachePayload, error) {
	payload := completedMatchCoreCachePayload{Match: *match, Rounds: []model.MatchRound{}, Actions: []model.MatchAction{}, RankChanges: []model.RankChangeLog{}}
	rounds, err := svcCtx.MatchModel.ListCompletedRounds(match.Id)
	if err != nil {
		return payload, err
	}
	payload.Rounds = rounds
	actions, err := svcCtx.MatchModel.ListActiveActions(match.Id)
	if err != nil {
		return payload, err
	}
	payload.Actions = actions
	achievements, err := svcCtx.MatchModel.GetAchievements(match.Id)
	if err != nil {
		return payload, err
	}
	payload.Achievements = BuildMatchAchievementPayload(achievements)
	if svcCtx.RankingModel != nil {
		logs, err := svcCtx.RankingModel.ListRankChangesByMatchAndGameType(match.Id, match.GameType)
		if err != nil {
			return payload, err
		}
		payload.RankChanges = logs
	}
	return payload, nil
}

func completedMatchCoreBestEffort(svcCtx *svc.ServiceContext, match *model.Match) completedMatchCoreCachePayload {
	payload := completedMatchCoreCachePayload{Match: *match, Rounds: []model.MatchRound{}, Actions: []model.MatchAction{}, RankChanges: []model.RankChangeLog{}}
	if rounds, err := svcCtx.MatchModel.ListCompletedRounds(match.Id); err == nil {
		payload.Rounds = rounds
	}
	if actions, err := svcCtx.MatchModel.ListActiveActions(match.Id); err == nil {
		payload.Actions = actions
	}
	if achievements, err := svcCtx.MatchModel.GetAchievements(match.Id); err == nil {
		payload.Achievements = BuildMatchAchievementPayload(achievements)
	}
	if svcCtx.RankingModel != nil {
		if logs, err := svcCtx.RankingModel.ListRankChangesByMatchAndGameType(match.Id, match.GameType); err == nil {
			payload.RankChanges = logs
		}
	}
	return payload
}

func hydrateCompletedMatchUsers(svcCtx *svc.ServiceContext, core *CompletedMatchCoreSummary) {
	if svcCtx == nil || svcCtx.UserModel == nil || core == nil || core.Match == nil {
		return
	}
	ids := []int64{core.Match.UserId}
	if core.Match.OpponentId != nil && *core.Match.OpponentId > 0 {
		ids = append(ids, *core.Match.OpponentId)
	}
	if core.Match.RefereeUserId != nil && *core.Match.RefereeUserId > 0 {
		ids = append(ids, *core.Match.RefereeUserId)
	}
	users, err := svcCtx.UserModel.FindByIds(ids)
	if err != nil {
		return
	}
	if user, ok := users[core.Match.UserId]; ok {
		copy := user
		core.Player1 = &copy
	}
	if core.Match.OpponentId != nil {
		if user, ok := users[*core.Match.OpponentId]; ok {
			copy := user
			core.Player2 = &copy
		}
	}
	if core.Match.RefereeUserId != nil {
		if user, ok := users[*core.Match.RefereeUserId]; ok {
			copy := user
			core.Referee = &copy
		}
	}
}

func CompletedMatchCoreCacheMetrics() observability.CacheSnapshot {
	return completedMatchCoreCacheMetrics.Snapshot()
}

func ResetCompletedMatchCoreCacheForTest() {
	completedMatchCoreCacheMetrics.Reset()
	completedMatchCoreCacheGroup = singleflight.Group{}
}
