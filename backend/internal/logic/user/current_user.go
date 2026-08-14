package user

import (
	"context"

	"chasing_points/internal/model"
	"chasing_points/internal/requestctx"
	"chasing_points/internal/svc"
)

func currentUserFromRequest(ctx context.Context, svcCtx *svc.ServiceContext, userID int64) (*model.User, error) {
	if activeUser := requestctx.ActiveUser(ctx); activeUser != nil && activeUser.Id == userID {
		return activeUser, nil
	}
	return svcCtx.UserModel.FindById(userID)
}
