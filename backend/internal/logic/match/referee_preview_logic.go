package match

import (
	"context"

	"chasing_points/internal/model"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"
	"chasing_points/internal/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

type RefereePreviewLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewRefereePreviewLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RefereePreviewLogic {
	return &RefereePreviewLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx.WithContext(ctx),
	}
}

func (l *RefereePreviewLogic) RefereePreview(req *types.RefereePreviewReq) (resp *types.RefereePreviewResp, err error) {
	userId, err := utils.GetUserIDFromCtx(l.ctx)
	if err != nil {
		return &types.RefereePreviewResp{Success: false, Message: "请先登录"}, nil
	}
	if req.MatchId <= 0 || req.JoinToken == "" {
		return &types.RefereePreviewResp{Success: false, Message: "无效的裁判二维码"}, nil
	}

	match, err := l.svcCtx.MatchModel.FindById(req.MatchId)
	if err != nil || match == nil {
		return &types.RefereePreviewResp{Success: false, Message: "对局不存在"}, nil
	}
	if match.Status != 1 {
		return &types.RefereePreviewResp{Success: false, Message: "对局已结束或已取消"}, nil
	}
	if match.RefereeUserId != nil && *match.RefereeUserId > 0 {
		return &types.RefereePreviewResp{Success: false, Message: "本场已有裁判，请重新扫码其他对局"}, nil
	}
	if match.UserId == userId || (match.OpponentId != nil && *match.OpponentId == userId) {
		return &types.RefereePreviewResp{Success: false, Message: "不能担任自己对局的裁判"}, nil
	}

	token, redisErr := l.svcCtx.Redis.Get(l.ctx, buildMatchRefereeJoinTokenKey(req.MatchId)).Result()
	if redisErr != nil || token != req.JoinToken {
		return &types.RefereePreviewResp{Success: false, Message: "裁判码已过期或无效，请重新扫码"}, nil
	}

	player1Name := "玩家1"
	player1Avatar := ""
	if player1, _ := l.svcCtx.UserModel.FindById(match.UserId); player1 != nil {
		if player1.Nickname != "" {
			player1Name = player1.Nickname
		}
		player1Avatar = player1.Avatar
	}

	player2Id := int64(0)
	player2Name := "玩家2"
	player2Avatar := ""
	if match.OpponentId != nil {
		player2Id = *match.OpponentId
		if player2, _ := l.svcCtx.UserModel.FindById(*match.OpponentId); player2 != nil {
			if player2.Nickname != "" {
				player2Name = player2.Nickname
			}
			player2Avatar = player2.Avatar
		}
	}

	return &types.RefereePreviewResp{
		Success: true,
		Preview: &types.RefereePreviewInfo{
			Player1Id:     match.UserId,
			Player1Name:   player1Name,
			Player1Avatar: player1Avatar,
			Player2Id:     player2Id,
			Player2Name:   player2Name,
			Player2Avatar: player2Avatar,
			Player1Score:  match.MyScore,
			Player2Score:  match.OpponentScore,
			GameType:      match.GameType,
			GameTypeName:  GetGameTypeName(match.GameType),
			MatchMode:     model.NormalizeMatchMode(match.MatchMode),
			Visibility:    model.NormalizeMatchVisibility(match.Visibility, match.MatchMode),
			Status:        match.Status,
			StatusText:    "进行中",
		},
	}, nil
}
