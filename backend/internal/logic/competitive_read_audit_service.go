package logic

import (
	"context"
	"fmt"

	"chasing_points/internal/model"
	"chasing_points/internal/svc"
)

type CompetitiveReadAuditDifference struct {
	Kind    string
	MatchId int64
	UserId  int64
	Detail  string
}

type CompetitiveReadAuditSummary struct {
	MatchesSampled int
	UsersSampled   int
	Differences    []CompetitiveReadAuditDifference
}

type CompetitiveReadAuditService struct {
	svcCtx *svc.ServiceContext
}

func NewCompetitiveReadAuditService(svcCtx *svc.ServiceContext) *CompetitiveReadAuditService {
	return &CompetitiveReadAuditService{svcCtx: svcCtx}
}

// Audit 只比较事实表与读模型，不修复也不写入任何业务表。
func (s *CompetitiveReadAuditService) Audit(ctx context.Context, sampleSize int) (*CompetitiveReadAuditSummary, error) {
	if s == nil || s.svcCtx == nil || s.svcCtx.MatchModel == nil || s.svcCtx.CompetitiveReadModel == nil {
		return nil, fmt.Errorf("competitive read model audit infrastructure is unavailable")
	}
	if sampleSize <= 0 {
		sampleSize = 100
	}
	if sampleSize > 1000 {
		sampleSize = 1000
	}
	matches, err := s.svcCtx.MatchModel.ListCompletedByIDAfterWithTx(s.svcCtx.DB.WithContext(ctx), 0, sampleSize)
	if err != nil {
		return nil, err
	}
	summary := &CompetitiveReadAuditSummary{MatchesSampled: len(matches)}
	users := map[int64]struct{}{}
	for _, match := range matches {
		for _, userID := range auditMatchParticipants(&match) {
			users[userID] = struct{}{}
			projection, err := s.svcCtx.CompetitiveReadModel.FindParticipantByMatchAndUser(match.Id, userID)
			if err != nil {
				return nil, err
			}
			if projection == nil {
				summary.Differences = append(summary.Differences, CompetitiveReadAuditDifference{
					Kind: "missing_projection", MatchId: match.Id, UserId: userID,
					Detail: "completed match has no participant projection",
				})
			}
		}
	}
	for userID := range users {
		if err := ctx.Err(); err != nil {
			return summary, err
		}
		legacy, err := s.svcCtx.MatchModel.GetUserStats(userID)
		if err != nil {
			return nil, err
		}
		snapshot, err := s.svcCtx.CompetitiveReadModel.FindStats(userID, 0)
		if err != nil {
			return nil, err
		}
		if snapshot == nil {
			summary.Differences = append(summary.Differences, CompetitiveReadAuditDifference{
				Kind: "missing_stats_snapshot", UserId: userID,
				Detail: "legacy stats exist but competitive snapshot is absent",
			})
			continue
		}
		if legacy.TotalMatches != snapshot.TotalMatches || legacy.Wins != snapshot.Wins || legacy.Losses != snapshot.Losses || legacy.MaxWinStreak != snapshot.MaxWinStreak {
			summary.Differences = append(summary.Differences, CompetitiveReadAuditDifference{
				Kind: "stats_mismatch", UserId: userID,
				Detail: fmt.Sprintf("legacy total/wins/losses/streak=%d/%d/%d/%d snapshot=%d/%d/%d/%d",
					legacy.TotalMatches, legacy.Wins, legacy.Losses, legacy.MaxWinStreak,
					snapshot.TotalMatches, snapshot.Wins, snapshot.Losses, snapshot.MaxWinStreak),
			})
		}
	}
	summary.UsersSampled = len(users)
	return summary, nil
}

func auditMatchParticipants(match *model.Match) []int64 {
	if match == nil || match.UserId <= 0 {
		return nil
	}
	users := []int64{match.UserId}
	if match.OpponentId != nil && *match.OpponentId > 0 && *match.OpponentId != match.UserId {
		users = append(users, *match.OpponentId)
	}
	return users
}
