package auth

import "chasing_points/internal/svc"

func ensureUserRankingProfiles(svcCtx *svc.ServiceContext, userID int64) error {
	if svcCtx == nil || svcCtx.RankingModel == nil || userID <= 0 {
		return nil
	}
	_, err := svcCtx.RankingModel.FindOrCreateByGameTypes(userID)
	return err
}
