package gear

import (
	"strconv"
	"testing"

	"chasing_points/internal/model"
	"chasing_points/internal/types"
)

func (e *gearEnv) viewUser(viewerId, targetId int64) (*types.GetGearUserResp, int) {
	e.t.Helper()
	ctx, svcCtx, metrics := e.request(viewerId)
	resp, err := NewGetGearUserLogic(ctx, svcCtx).GetGearUser(&types.GetGearUserReq{UserId: targetId})
	if err != nil {
		e.t.Fatalf("get gear user returned an error: %v", err)
	}
	return resp, int(metrics.Snapshot().SQLCount)
}

func TestGetGearUserShowsFriendsOnlyActiveItemsAndHidesCustomNames(t *testing.T) {
	env := newGearEnv(t, "")
	env.makeFriends(1, 2)

	listed := env.mustCreate(2, types.CreateGearItemReq{Category: model.GearCategoryCue, ProductId: env.c.cuePool, TipProductId: env.c.tipAlpha})
	unlisted := env.mustCreate(2, types.CreateGearItemReq{Category: model.GearCategoryCue, BrandName: "私人品牌", ModelName: "私人型号", CueStyle: model.GearCueStylePool})
	retired := env.mustCreate(2, types.CreateGearItemReq{Category: model.GearCategoryCue, ProductId: env.c.cueSnooker})
	ctx, svcCtx, _ := env.request(2)
	if resp, _ := NewUpdateGearItemLogic(ctx, svcCtx).UpdateGearItem(&types.UpdateGearItemReq{ItemId: retired, Status: model.GearItemStatusRetired}); !resp.Success {
		t.Fatalf("retire: %+v", resp)
	}
	ctx, svcCtx, _ = env.request(2)
	if resp, _ := NewSetGearDefaultLogic(ctx, svcCtx).SetGearDefault(&types.SetGearDefaultReq{GameType: 3, Slot: model.GearSlotCue, ItemId: listed}); !resp.Success {
		t.Fatalf("set default: %+v", resp)
	}

	friendView, _ := env.viewUser(1, 2)
	if !friendView.Success || len(friendView.Items) != 2 {
		t.Fatalf("a friend sees the two active items only: %+v", friendView)
	}
	if findItem(friendView.Items, retired) != nil {
		t.Fatal("retired items are private to the owner")
	}
	if len(friendView.Defaults) != 1 || friendView.Defaults[0].ItemId != listed {
		t.Fatalf("a friend sees the main-gear settings: %+v", friendView.Defaults)
	}
	hidden := findItem(friendView.Items, unlisted)
	if hidden == nil || hidden.BrandName != "其他品牌" || hidden.ModelName != "未收录型号" {
		t.Fatalf("a friend must not see typed names: %+v", hidden)
	}
	if shown := findItem(friendView.Items, listed); shown == nil || shown.Product == nil || shown.TipProduct == nil {
		t.Fatalf("catalog items carry their catalog data: %+v", shown)
	}

	selfView, _ := env.viewUser(2, 2)
	if !selfView.Success || findItem(selfView.Items, unlisted).ModelName != "私人型号" {
		t.Fatalf("the owner asking by their own id sees their typed names: %+v", selfView)
	}
}

func TestGetGearUserDeniesNonFriendsAndBlockedUsers(t *testing.T) {
	env := newGearEnv(t, "")
	env.mustCreate(2, types.CreateGearItemReq{Category: model.GearCategoryCue, ProductId: env.c.cuePool})

	stranger, _ := env.viewUser(3, 2)
	if stranger.Success || stranger.Reason != reasonNotFriend || len(stranger.Items) != 0 || stranger.Items == nil {
		t.Fatalf("a non-friend must get nothing: %+v", stranger)
	}

	env.makeFriends(3, 2)
	if allowed, _ := env.viewUser(3, 2); !allowed.Success {
		t.Fatalf("befriended viewer should be allowed: %+v", allowed)
	}
	// 双向拉黑都要拒绝
	if err := env.db.Create(&model.FriendBlacklist{UserId: 2, BlockedUserId: 3}).Error; err != nil {
		t.Fatalf("blacklist: %v", err)
	}
	if blocked, _ := env.viewUser(3, 2); blocked.Success || blocked.Reason != reasonNotFriend || len(blocked.Items) != 0 {
		t.Fatalf("a blocked relation must hide the gear even between friends: %+v", blocked)
	}

	if invalid, _ := env.viewUser(3, 0); invalid.Success {
		t.Fatalf("a missing user id is rejected: %+v", invalid)
	}
	admin, _ := env.viewUser(-1, 2)
	if admin.Success {
		t.Fatalf("an admin token must not read user gear: %+v", admin)
	}
}

func TestGetGearUserUsesConstantQueries(t *testing.T) {
	count := func(items int) int {
		env := newGearEnv(t, strconv.Itoa(items))
		env.makeFriends(1, 2)
		m := model.NewGearModel(env.db)
		for i := 0; i < items; i++ {
			item := model.UserGearItem{
				UserId: 2, Category: model.GearCategoryCue, ProductId: env.c.cuePool,
				TipProductId: env.c.tipAlpha, Status: model.GearItemStatusActive,
			}
			if err := m.CreateItemWithTx(nil, &item); err != nil {
				t.Fatalf("seed item: %v", err)
			}
		}
		resp, queries := env.viewUser(1, 2)
		if !resp.Success || len(resp.Items) != items {
			t.Fatalf("view user: %+v", resp)
		}
		return queries
	}
	one, thirty := count(1), count(30)
	if one != thirty {
		t.Fatalf("the friend view must not issue more queries for more items: one=%d thirty=%d", one, thirty)
	}
}
