package admin

import (
	"context"
	"time"

	logicx "chasing_points/internal/logic"
	"chasing_points/internal/model"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

var adminUserListNow = logicx.NowUTC8

type AdminGetUserListLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 获取用户列表（管理员）
func NewAdminGetUserListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AdminGetUserListLogic {
	return &AdminGetUserListLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx.WithContext(ctx),
	}
}

func (l *AdminGetUserListLogic) AdminGetUserList(req *types.AdminUserListReq) (resp *types.AdminUserListResp, err error) {
	users, total, err := l.svcCtx.UserModel.FindListForAdmin(req.Page, req.PageSize, req.Status)
	if err != nil {
		l.Logger.Errorf("获取用户列表失败: err=%v", err)
		return &types.AdminUserListResp{
			Code:    500,
			Success: false,
			Message: "获取用户列表失败",
		}, nil
	}

	reputationConfig, configErr := logicx.NewReputationConfigService(l.svcCtx).GetConfig()
	if configErr != nil {
		l.Logger.Errorf("获取信誉配置失败: err=%v", configErr)
		return &types.AdminUserListResp{
			Code:    500,
			Success: false,
			Message: "获取用户信誉信息失败",
		}, nil
	}

	ids := make([]int64, 0, len(users))
	for _, user := range users {
		ids = append(ids, user.Id)
	}

	profileByUserID := make(map[int64]*model.UserReputationProfile, len(ids))
	if len(ids) > 0 {
		var profiles []model.UserReputationProfile
		if err := l.svcCtx.DB.Where("user_id IN ?", ids).Find(&profiles).Error; err != nil {
			l.Logger.Errorf("批量获取用户信誉档案失败: err=%v", err)
			return &types.AdminUserListResp{
				Code:    500,
				Success: false,
				Message: "获取用户信誉信息失败",
			}, nil
		}
		for i := range profiles {
			profileByUserID[profiles[i].UserID] = &profiles[i]
		}
	}

	items := make([]types.AdminUserInfo, 0, len(users))
	now := logicx.InUTC8(adminUserListNow())
	for _, user := range users {
		// 脱敏手机号
		phone := ""
		if user.Phone != nil && len(*user.Phone) == 11 {
			phone = (*user.Phone)[:3] + "****" + (*user.Phone)[7:]
		}

		reputationScore := clampAdminDisplayReputationScore(reputationConfig.BaseRules.InitialScore, reputationConfig.BaseRules)
		banUntil := ""
		profile := profileByUserID[user.Id]
		if profile != nil {
			reputationScore = computeAdminDisplayReputationScore(profile, reputationConfig.BaseRules, reputationConfig.RecoveryRules, now)
			banUntil = logicx.FormatUTC8TimePtr(profile.BanUntil)
		}

		items = append(items, types.AdminUserInfo{
			Id:              user.Id,
			Phone:           phone,
			Nickname:        user.Nickname,
			Avatar:          user.Avatar,
			Status:          user.Status,
			ReputationScore: reputationScore,
			BanUntil:        banUntil,
			MemberStatus:    resolveAdminMemberStatus(&user),
			MemberExpiresAt: logicx.FormatUTC8TimePtr(user.MemberExpiresAt),
			CreatedAt:       user.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}

	return &types.AdminUserListResp{
		Code:    0,
		Success: true,
		Message: "success",
		Total:   total,
		List:    items,
	}, nil
}

func computeAdminDisplayReputationScore(profile *model.UserReputationProfile, baseRules model.ReputationBaseRules, recoveryRules model.ReputationRecoveryRules, now time.Time) int {
	if profile == nil {
		return clampAdminDisplayReputationScore(baseRules.InitialScore, baseRules)
	}

	score := clampAdminDisplayReputationScore(profile.ReputationScore, baseRules)
	if profile.LastRecoveredAt == nil || profile.LastRecoveredAt.IsZero() {
		return score
	}

	lastRecoveredAt := logicx.InUTC8(*profile.LastRecoveredAt)
	if !recoveryRules.Enabled || recoveryRules.RecoverPerHour <= 0 || !now.After(lastRecoveredAt) {
		return score
	}

	elapsedHours := int(now.Sub(lastRecoveredAt) / time.Hour)
	if elapsedHours <= 0 {
		return score
	}

	recoveryCap := recoveryRules.RecoverMaxScore
	if recoveryCap <= 0 {
		recoveryCap = baseRules.MaxScore
	}
	if recoveryCap > baseRules.MaxScore {
		recoveryCap = baseRules.MaxScore
	}
	if recoveryCap < baseRules.MinScore {
		recoveryCap = baseRules.MinScore
	}

	if score < recoveryCap {
		score += elapsedHours * recoveryRules.RecoverPerHour
		if score > recoveryCap {
			score = recoveryCap
		}
	}
	return clampAdminDisplayReputationScore(score, baseRules)
}

func clampAdminDisplayReputationScore(score int, baseRules model.ReputationBaseRules) int {
	if score < baseRules.MinScore {
		return baseRules.MinScore
	}
	if baseRules.MaxScore > 0 && score > baseRules.MaxScore {
		return baseRules.MaxScore
	}
	return score
}

func resolveAdminMemberStatus(user *model.User) string {
	if user == nil || user.MemberExpiresAt == nil {
		return "未开通"
	}
	if logicx.InUTC8(*user.MemberExpiresAt).After(logicx.NowUTC8()) {
		return "会员中"
	}
	return "已到期"
}
