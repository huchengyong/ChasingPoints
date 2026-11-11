package social

import (
	"context"
	"encoding/json"
	"net/url"
	"strings"
	"unicode/utf8"

	"chasing_points/internal/config"
	"chasing_points/internal/model"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"
	"chasing_points/internal/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

const (
	socialPostMaxContentRunes = 1000
	socialPostMaxImages       = 9
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
		svcCtx: svcCtx.WithContext(ctx),
	}
}

func (l *CreatePostLogic) CreatePost(req *types.CreatePostReq) (resp *types.CreatePostResp, err error) {
	if !l.svcCtx.Config.SocialEnabled() {
		return &types.CreatePostResp{Success: false, Message: config.DisabledFeatureMessage("social")}, nil
	}

	userIdInt, err := utils.GetUserIDFromCtx(l.ctx)
	if err != nil {
		l.Logger.Errorf("获取用户ID失败: %v", err)
		return &types.CreatePostResp{Success: false, Message: "获取用户信息失败"}, nil
	}

	content := strings.TrimSpace(req.Content)
	if utf8.RuneCountInString(content) > socialPostMaxContentRunes {
		return &types.CreatePostResp{Success: false, Message: "内容不能超过 1000 字"}, nil
	}

	images, message := l.normalizeImages(req.Images)
	if message != "" {
		return &types.CreatePostResp{Success: false, Message: message}, nil
	}

	if content == "" && len(images) == 0 {
		return &types.CreatePostResp{Success: false, Message: "说点什么再发布吧"}, nil
	}

	postType := req.PostType
	if postType <= 0 {
		postType = 3
	}
	if postType < 1 || postType > 3 {
		return &types.CreatePostResp{Success: false, Message: "动态类型无效"}, nil
	}

	var matchIdPtr *int64
	if postType == 1 {
		if req.MatchId <= 0 {
			return &types.CreatePostResp{Success: false, Message: "战报需要关联比赛"}, nil
		}
		match, matchErr := l.svcCtx.MatchModel.FindById(req.MatchId)
		if matchErr != nil {
			l.Logger.Errorf("查询比赛失败: matchId=%d err=%v", req.MatchId, matchErr)
			return &types.CreatePostResp{Success: false, Message: "只能分享自己参与的比赛"}, nil
		}
		involved := match != nil && (match.UserId == userIdInt || (match.OpponentId != nil && *match.OpponentId == userIdInt))
		if !involved {
			return &types.CreatePostResp{Success: false, Message: "只能分享自己参与的比赛"}, nil
		}
		matchId := req.MatchId
		matchIdPtr = &matchId
	}

	var imagesPtr *string
	if len(images) > 0 {
		imagesBytes, marshalErr := json.Marshal(images)
		if marshalErr != nil {
			l.Logger.Errorf("序列化动态图片失败: %v", marshalErr)
			return &types.CreatePostResp{Success: false, Message: "发布失败"}, nil
		}
		imagesStr := string(imagesBytes)
		imagesPtr = &imagesStr
	}

	post := &model.SocialPost{
		UserId:   userIdInt,
		Content:  content,
		Images:   imagesPtr,
		PostType: postType,
		MatchId:  matchIdPtr,
		Status:   model.SocialPostStatusPending,
	}

	if err = l.svcCtx.SocialPostModel.Create(post); err != nil {
		l.Logger.Errorf("创建动态失败: userId=%d err=%v", userIdInt, err)
		return &types.CreatePostResp{Success: false, Message: "发布失败"}, nil
	}

	return &types.CreatePostResp{Success: true, PostId: post.Id}, nil
}

// normalizeImages 校验图片数量与来源，返回 trim 后的地址；失败时第二返回值为给用户的提示。
func (l *CreatePostLogic) normalizeImages(images []string) ([]string, string) {
	if len(images) > socialPostMaxImages {
		return nil, "最多上传 9 张图片"
	}
	if len(images) == 0 {
		return nil, ""
	}

	publicDomain := strings.TrimSpace(l.svcCtx.Config.Qiniu.PublicDomain)
	domainURL, domainErr := url.Parse(publicDomain)
	if publicDomain == "" || domainErr != nil || domainURL.Hostname() == "" {
		l.Logger.Errorf("动态图片地址校验失败，七牛 PublicDomain 未配置或无效: %q err=%v", publicDomain, domainErr)
		return nil, "图片地址无效"
	}

	normalized := make([]string, 0, len(images))
	for _, raw := range images {
		imageURL := strings.TrimSpace(raw)
		parsed, parseErr := url.Parse(imageURL)
		if parseErr != nil || parsed.Scheme != "https" || !strings.EqualFold(parsed.Hostname(), domainURL.Hostname()) {
			return nil, "图片地址无效"
		}
		normalized = append(normalized, imageURL)
	}
	return normalized, ""
}
