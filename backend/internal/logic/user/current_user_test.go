package user

import (
	"context"
	"testing"

	"chasing_points/internal/model"
	"chasing_points/internal/requestctx"
	"chasing_points/internal/svc"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestCurrentUserFromRequestReusesValidatedRequestUser(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	userModel := model.NewUserModel(db)
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get sql db: %v", err)
	}
	if err := sqlDB.Close(); err != nil {
		t.Fatalf("close sql db: %v", err)
	}
	validated := &model.User{Id: 7, Nickname: "已校验用户"}
	user, err := currentUserFromRequest(requestctx.WithActiveUser(context.Background(), validated), &svc.ServiceContext{UserModel: userModel}, 7)
	if err != nil || user != validated {
		t.Fatalf("request user must be reused without a database read: user=%#v err=%v", user, err)
	}
}
