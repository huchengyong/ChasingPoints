package match

import (
	"context"
	"encoding/json"
	"time"

	"chasing_points/internal/svc"
	"chasing_points/internal/types"
	"chasing_points/internal/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetMatchRefereeQRCodeLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 获取本场裁判二维码
func NewGetMatchRefereeQRCodeLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetMatchRefereeQRCodeLogic {
	return &GetMatchRefereeQRCodeLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetMatchRefereeQRCodeLogic) GetMatchRefereeQRCode(req *types.GetMatchRefereeQRCodeReq) (resp *types.GetMatchRefereeQRCodeResp, err error) {
	userId, err := utils.GetUserIDFromCtx(l.ctx)
	if err != nil {
		l.Logger.Errorf("获取用户ID失败: %v", err)
		return &types.GetMatchRefereeQRCodeResp{Success: false}, nil
	}
	if req.MatchId <= 0 {
		return &types.GetMatchRefereeQRCodeResp{Success: false}, nil
	}

	match, err := l.svcCtx.MatchModel.FindById(req.MatchId)
	if err != nil || match == nil {
		l.Logger.Errorf("获取对局失败: matchId=%d err=%v", req.MatchId, err)
		return &types.GetMatchRefereeQRCodeResp{Success: false}, nil
	}
	if match.Status != 1 {
		return &types.GetMatchRefereeQRCodeResp{Success: false}, nil
	}

	capabilities := resolveMatchViewerCapabilities(match, userId)
	if capabilities.ViewerRole != matchViewerRolePlayer1 && capabilities.ViewerRole != matchViewerRolePlayer2 {
		return &types.GetMatchRefereeQRCodeResp{Success: false}, nil
	}
	if capabilities.RefereeBound {
		return &types.GetMatchRefereeQRCodeResp{Success: false}, nil
	}

	joinToken, err := newMatchRefereeJoinToken()
	if err != nil {
		l.Logger.Errorf("生成裁判加入 token 失败: %v", err)
		return &types.GetMatchRefereeQRCodeResp{Success: false}, nil
	}
	if err := l.svcCtx.Redis.Set(l.ctx, buildMatchRefereeJoinTokenKey(match.Id), joinToken, matchRefereeJoinTokenTTL).Err(); err != nil {
		l.Logger.Errorf("写入裁判加入 token 失败: %v", err)
		return &types.GetMatchRefereeQRCodeResp{Success: false}, nil
	}

	payload, err := json.Marshal(matchRefereeQRCodePayload{
		Type:      matchRefereePayloadType,
		MatchId:   match.Id,
		JoinToken: joinToken,
		Ts:        time.Now().Unix(),
	})
	if err != nil {
		l.Logger.Errorf("序列化裁判二维码失败: %v", err)
		return &types.GetMatchRefereeQRCodeResp{Success: false}, nil
	}

	return &types.GetMatchRefereeQRCodeResp{
		Success:          true,
		QrcodeData:       string(payload),
		ExpiresInSeconds: int64(matchRefereeJoinTokenTTL.Seconds()),
	}, nil
}
