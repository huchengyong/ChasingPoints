package logic

import (
	"context"
	"encoding/json"

	"billiard_master/internal/model"
	"billiard_master/internal/svc"
	"billiard_master/internal/types"
	"billiard_master/internal/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

type CreatePostLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 发布动态
func NewCreatePostLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreatePostLogic {
	return &CreatePostLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *CreatePostLogic) CreatePost(req *types.CreatePostReq) (resp *types.CreatePostResp, err error) {
	userIdInt, err := utils.GetUserIDFromCtx(l.ctx)
	if err != nil {
		l.Logger.Errorf("获取用户ID失败: %v", err)
		return &types.CreatePostResp{Success: false}, nil
	}

	var imagesPtr *string
	if len(req.Images) > 0 {
		imagesBytes, marshalErr := json.Marshal(req.Images)
		if marshalErr != nil {
			l.Logger.Errorf("序列化动态图片失败: %v", marshalErr)
			return &types.CreatePostResp{Success: false}, nil
		}
		imagesStr := string(imagesBytes)
		imagesPtr = &imagesStr
	}

	var matchIdPtr *int64
	if req.MatchId > 0 {
		matchId := req.MatchId
		matchIdPtr = &matchId
	}

	postType := req.PostType
	if postType <= 0 {
		postType = 3
	}

	post := &model.SocialPost{
		UserId:   userIdInt,
		Content:  req.Content,
		Images:   imagesPtr,
		PostType: postType,
		MatchId:  matchIdPtr,
	}

	if err = l.svcCtx.SocialPostModel.Create(post); err != nil {
		l.Logger.Errorf("创建动态失败: userId=%d err=%v", userIdInt, err)
		return &types.CreatePostResp{Success: false}, nil
	}

	return &types.CreatePostResp{Success: true, PostId: post.Id}, nil
}
