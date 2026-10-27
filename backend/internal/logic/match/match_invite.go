package match

import (
	logic "chasing_points/internal/logic"
	"chasing_points/internal/pkg/matchinvite"
	"chasing_points/internal/svc"
)

func newMatchInviteSigner(svcCtx *svc.ServiceContext) (*matchinvite.Signer, error) {
	return logic.NewMatchInviteSigner(svcCtx)
}

func verifyMatchInvite(svcCtx *svc.ServiceContext, token string) (*matchinvite.Claims, string) {
	return logic.VerifyMatchInvite(svcCtx, token)
}
