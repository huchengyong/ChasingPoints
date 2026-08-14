package logic

import (
	"fmt"
	"time"

	"chasing_points/internal/model"
	seasonx "chasing_points/internal/season"
	"chasing_points/internal/svc"

	"gorm.io/gorm"
)

// CompetitiveProjectionInput 提供一场已完成比赛及结算前后的段位快照。
// 段位快照只用于保存结算时对手强度和当前赛季记录，不作为比赛事实真源。
type CompetitiveProjectionInput struct {
	Match         *model.Match
	Player1Before *model.UserRanking
	Player1After  *model.UserRanking
	Player2Before *model.UserRanking
	Player2After  *model.UserRanking
}

type CompetitiveProjectionResult struct {
	AppliedUsers         map[int64]bool
	CompetitiveRevisions map[int64]int64
	SeasonID             int64
}

type competitiveParticipant struct {
	result     model.MatchParticipantResult
	beforeRank *model.UserRanking
	afterRank  *model.UserRanking
	won        bool
}

type CompetitiveProjector struct {
	svcCtx *svc.ServiceContext
}

func NewCompetitiveProjector(svcCtx *svc.ServiceContext) *CompetitiveProjector {
	return &CompetitiveProjector{svcCtx: svcCtx}
}

func (p *CompetitiveProjector) ProjectWithTx(tx *gorm.DB, input CompetitiveProjectionInput) (CompetitiveProjectionResult, error) {
	out := CompetitiveProjectionResult{
		AppliedUsers:         map[int64]bool{},
		CompetitiveRevisions: map[int64]int64{},
	}
	if p == nil || p.svcCtx == nil || p.svcCtx.CompetitiveReadModel == nil || p.svcCtx.UserModel == nil {
		return out, fmt.Errorf("competitive read model infrastructure is unavailable")
	}
	match := input.Match
	if match == nil || match.Status != 2 || match.Result == nil {
		return out, nil
	}

	completedAt := resolveMatchCompletedAt(match)
	actions, err := p.svcCtx.MatchModel.ListActiveActionsWithTx(tx, match.Id)
	if err != nil {
		return out, err
	}
	player1, err := p.svcCtx.UserModel.FindByIdWithTx(tx, match.UserId)
	if err != nil {
		return out, err
	}
	if player1 == nil {
		return out, fmt.Errorf("match %d owner %d is missing", match.Id, match.UserId)
	}

	var player2 *model.User
	player2ID := int64(0)
	if match.OpponentId != nil && *match.OpponentId > 0 && *match.OpponentId != match.UserId {
		player2ID = *match.OpponentId
		player2, err = p.svcCtx.UserModel.FindByIdWithTx(tx, player2ID)
		if err != nil {
			return out, err
		}
	}

	participants := buildCompetitiveParticipants(match, completedAt, actions, player1, player2, input)
	eligibleForCompetitiveStats := model.NormalizeMatchMode(match.MatchMode) == model.MatchModeRanked && (*match.Result == 1 || *match.Result == 2)
	for _, participant := range participants {
		created, err := p.svcCtx.CompetitiveReadModel.InsertParticipantIfAbsentWithTx(tx, &participant.result)
		if err != nil {
			return out, err
		}
		if !created {
			continue
		}
		out.AppliedUsers[participant.result.UserId] = true
		if !eligibleForCompetitiveStats {
			continue
		}
		if err := p.applyParticipantStatsWithTx(tx, participant); err != nil {
			return out, err
		}
		seasonID, err := p.applySeasonRecordWithTx(tx, participant)
		if err != nil {
			return out, err
		}
		if seasonID > 0 {
			out.SeasonID = seasonID
		}
	}

	for userID := range out.AppliedUsers {
		if !eligibleForCompetitiveStats {
			continue
		}
		stats, err := p.svcCtx.CompetitiveReadModel.FindStatsWithTx(tx, userID, 0)
		if err != nil {
			return out, err
		}
		if stats != nil {
			out.CompetitiveRevisions[userID] = stats.Revision
		}
	}
	return out, nil
}

func buildCompetitiveParticipants(
	match *model.Match,
	completedAt time.Time,
	actions []model.MatchAction,
	player1, player2 *model.User,
	input CompetitiveProjectionInput,
) []competitiveParticipant {
	if match == nil || player1 == nil || match.Result == nil {
		return nil
	}
	duration := int64(completedAt.Sub(match.MatchTime).Seconds())
	if duration < 0 {
		duration = 0
	}
	player1Break, player2Break := match.MyScore, match.OpponentScore
	if match.GameType == 1 {
		player1Break, player2Break = CalculateSnookerHighestBreaks(actions, 1)
	}

	player2ID := int64(0)
	if match.OpponentId != nil {
		player2ID = *match.OpponentId
	}
	player2Identity := NormalizeOpponentIdentity(player2ID, match.OpponentName, match.Id)
	player2Avatar := ""
	player2Name := match.OpponentName
	if player2 != nil {
		player2Name = player2.Nickname
		player2Avatar = player2.Avatar
	}
	participants := []competitiveParticipant{{
		result: model.MatchParticipantResult{
			MatchId:            match.Id,
			UserId:             match.UserId,
			OpponentUserId:     player2Identity.UserId,
			OpponentNameKey:    player2Identity.NameKey,
			OpponentName:       player2Name,
			OpponentAvatar:     player2Avatar,
			GameType:           match.GameType,
			MatchMode:          model.NormalizeMatchMode(match.MatchMode),
			Result:             *match.Result,
			MyScore:            match.MyScore,
			OpponentScore:      match.OpponentScore,
			CompletedAt:        completedAt,
			DurationSeconds:    duration,
			MatchHighScore:     match.MyScore,
			BestBreak:          player1Break,
			OpponentRankBucket: rankBucket(input.Player2Before),
		},
		beforeRank: input.Player1Before,
		afterRank:  input.Player1After,
		won:        *match.Result == 1,
	}}
	if player2 == nil || player2ID <= 0 {
		return participants
	}

	player1Identity := NormalizeOpponentIdentity(match.UserId, player1.Nickname, match.Id)
	participants = append(participants, competitiveParticipant{
		result: model.MatchParticipantResult{
			MatchId:            match.Id,
			UserId:             player2ID,
			OpponentUserId:     player1Identity.UserId,
			OpponentNameKey:    player1Identity.NameKey,
			OpponentName:       player1.Nickname,
			OpponentAvatar:     player1.Avatar,
			GameType:           match.GameType,
			MatchMode:          model.NormalizeMatchMode(match.MatchMode),
			Result:             inverseMatchResult(*match.Result),
			MyScore:            match.OpponentScore,
			OpponentScore:      match.MyScore,
			CompletedAt:        completedAt,
			DurationSeconds:    duration,
			MatchHighScore:     match.OpponentScore,
			BestBreak:          player2Break,
			OpponentRankBucket: rankBucket(input.Player1Before),
		},
		beforeRank: input.Player2Before,
		afterRank:  input.Player2After,
		won:        *match.Result == 2,
	})
	return participants
}

func (p *CompetitiveProjector) applyParticipantStatsWithTx(tx *gorm.DB, participant competitiveParticipant) error {
	result := participant.result
	statsDeltas := []model.CompetitiveStatsDelta{
		{UserId: result.UserId, GameType: 0, Result: result.Result, MatchId: result.MatchId, CompletedAt: result.CompletedAt, Score: result.MatchHighScore, BestBreak: result.BestBreak, DurationSeconds: result.DurationSeconds},
		{UserId: result.UserId, GameType: result.GameType, Result: result.Result, MatchId: result.MatchId, CompletedAt: result.CompletedAt, Score: result.MatchHighScore, BestBreak: result.BestBreak, DurationSeconds: result.DurationSeconds},
	}
	for _, delta := range statsDeltas {
		if err := p.svcCtx.CompetitiveReadModel.ApplyCompetitiveStatsWithTx(tx, delta); err != nil {
			return err
		}
	}
	opponentDeltas := []model.OpponentStatsDelta{
		{UserId: result.UserId, OpponentUserId: result.OpponentUserId, OpponentNameKey: result.OpponentNameKey, GameType: 0, Result: result.Result, MatchId: result.MatchId, CompletedAt: result.CompletedAt, OpponentName: result.OpponentName, OpponentAvatar: result.OpponentAvatar, MyScore: result.MyScore, OpponentScore: result.OpponentScore},
		{UserId: result.UserId, OpponentUserId: result.OpponentUserId, OpponentNameKey: result.OpponentNameKey, GameType: result.GameType, Result: result.Result, MatchId: result.MatchId, CompletedAt: result.CompletedAt, OpponentName: result.OpponentName, OpponentAvatar: result.OpponentAvatar, MyScore: result.MyScore, OpponentScore: result.OpponentScore},
	}
	for _, delta := range opponentDeltas {
		if err := p.svcCtx.CompetitiveReadModel.ApplyOpponentStatsWithTx(tx, delta); err != nil {
			return err
		}
	}
	for _, gameType := range []int{0, result.GameType} {
		if err := p.svcCtx.CompetitiveReadModel.IncrementOpponentStrengthBucketWithTx(tx, model.OpponentStrengthBucketDelta{
			UserId:     result.UserId,
			GameType:   gameType,
			RankBucket: result.OpponentRankBucket,
			Result:     result.Result,
		}); err != nil {
			return err
		}
	}
	return nil
}

func (p *CompetitiveProjector) applySeasonRecordWithTx(tx *gorm.DB, participant competitiveParticipant) (int64, error) {
	if participant.beforeRank == nil || participant.afterRank == nil || p.svcCtx.SeasonModel == nil || p.svcCtx.SeasonRecordModel == nil {
		return 0, nil
	}
	location, err := seasonx.LocationForConfig(p.svcCtx.Config.SeasonLifecycle)
	if err != nil {
		return 0, err
	}
	season, err := p.svcCtx.SeasonModel.FindByEffectiveTimeInLocationWithTx(tx, participant.result.CompletedAt, location)
	if err != nil || season == nil {
		return 0, err
	}
	if err := p.svcCtx.SeasonRecordModel.ApplyCompetitiveMatchWithTx(tx, model.SeasonRecordMatchDelta{
		SeasonId:       season.Id,
		UserId:         participant.result.UserId,
		GameType:       participant.result.GameType,
		StartRankScore: participant.beforeRank.RankScore,
		EndRankScore:   participant.afterRank.RankScore,
		Won:            participant.won,
	}); err != nil {
		return 0, err
	}
	return season.Id, nil
}

func resolveMatchCompletedAt(match *model.Match) time.Time {
	if match != nil && match.CompletedAt != nil {
		return *match.CompletedAt
	}
	if match != nil && match.EndTime != nil {
		return *match.EndTime
	}
	if match != nil {
		return match.MatchTime
	}
	return time.Time{}
}

func inverseMatchResult(result int) int {
	if result == 1 {
		return 2
	}
	if result == 2 {
		return 1
	}
	return 3
}

func rankBucket(ranking *model.UserRanking) string {
	if ranking == nil {
		return "score_0_1000"
	}
	switch {
	case ranking.RankScore <= 1000:
		return "score_0_1000"
	case ranking.RankScore <= 2000:
		return "score_1001_2000"
	default:
		return "score_2001_plus"
	}
}
