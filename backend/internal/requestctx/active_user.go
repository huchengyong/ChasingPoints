package requestctx

import (
	"context"

	"chasing_points/internal/model"
)

type activeUserContextKey struct{}

func WithActiveUser(ctx context.Context, user *model.User) context.Context {
	return context.WithValue(ctx, activeUserContextKey{}, user)
}

func ActiveUser(ctx context.Context) *model.User {
	if ctx == nil {
		return nil
	}
	user, _ := ctx.Value(activeUserContextKey{}).(*model.User)
	return user
}
