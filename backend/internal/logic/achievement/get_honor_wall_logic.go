package achievement

import (
	"context"
	"time"

	"chasing_points/internal/model"
	seasonx "chasing_points/internal/season"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"
	"chasing_points/internal/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetHonorWallLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 获取本人或好友荣誉墙
func NewGetHonorWallLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetHonorWallLogic {
	return &GetHonorWallLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetHonorWallLogic) GetHonorWall(req *types.GetHonorWallReq) (resp *types.GetHonorWallResp, err error) {
	requesterId, err := utils.GetUserIDFromCtx(l.ctx)
	if err != nil {
		l.Logger.Errorf("获取用户ID失败: %v", err)
		return emptyHonorWallResponse(false), nil
	}
	if req == nil {
		req = &types.GetHonorWallReq{}
	}
	specialtyGameType := normalizeSeasonChallengeGameType(req.GameType)

	targetUserId := req.UserId
	if targetUserId <= 0 {
		targetUserId = requesterId
	}
	viewerScope := "self"
	if targetUserId != requesterId {
		allowed, allowErr := l.canViewFriendHonorWall(requesterId, targetUserId)
		if allowErr != nil {
			return nil, allowErr
		}
		if !allowed {
			return emptyHonorWallResponse(false), nil
		}
		viewerScope = "friend"
	}

	user, err := l.svcCtx.UserModel.FindById(targetUserId)
	if err != nil {
		return nil, err
	}
	if user == nil || user.Status != 1 {
		return emptyHonorWallResponse(false), nil
	}

	definitions, err := l.svcCtx.AchievementModel.FindActive()
	if err != nil {
		return nil, err
	}
	userAchievements, err := l.svcCtx.UserAchievementModel.FindByUserId(targetUserId)
	if err != nil {
		return nil, err
	}
	titles, err := l.svcCtx.UserTitleModel.FindByUserId(targetUserId)
	if err != nil {
		return nil, err
	}

	includeLocked := viewerScope == "self"
	careerAchievements, unlockedTotal := buildCareerAchievementDefs(definitions, userAchievements, includeLocked)
	universalUnlocked, universalTotal, specialtyUnlocked, specialtyTotal := countCareerAchievementScopes(definitions, userAchievements, specialtyGameType)
	recentHonors := buildRecentHonors(userAchievements, definitions, titles)
	equippedTitle, err := l.svcCtx.UserTitleModel.FindEquippedByUserId(targetUserId)
	if err != nil {
		return nil, err
	}

	page, pageSize := normalizeHonorHistoryPage(req.HistoryPage, req.HistoryPageSize)
	permanentTitles, permanentTotal, err := l.svcCtx.UserTitleModel.FindPermanentByUserId(targetUserId, page, pageSize)
	if err != nil {
		return nil, err
	}
	historyHonors := make([]types.HonorItem, 0, len(permanentTitles))
	for _, title := range permanentTitles {
		historyHonors = append(historyHonors, titleToHonorItem(title))
	}

	seasonHonors := 0
	tournamentHonors := 0
	for _, title := range titles {
		switch title.SourceType {
		case SourceTypeSeason:
			seasonHonors++
		case SourceTypeTournament:
			tournamentHonors++
		}
	}

	resp = &types.GetHonorWallResp{
		Success:     true,
		SeasonState: seasonx.StateNotStarted,
		ViewerScope: viewerScope,
		Profile: types.HonorWallProfile{
			UserId:   user.Id,
			Nickname: user.Nickname,
			Avatar:   user.Avatar,
		},
		EquippedTitle: titleToInfo(equippedTitle),
		Summary: types.HonorWallSummary{
			CareerUnlocked:    unlockedTotal,
			CareerTotal:       len(definitions),
			UniversalUnlocked: universalUnlocked,
			UniversalTotal:    universalTotal,
			SpecialtyGameType: specialtyGameType,
			SpecialtyUnlocked: specialtyUnlocked,
			SpecialtyTotal:    specialtyTotal,
			SeasonHonors:      seasonHonors,
			TournamentHonors:  tournamentHonors,
		},
		RecentHonors:       recentHonors,
		CareerAchievements: careerAchievements,
		History: types.HonorWallHistory{
			Total:            permanentTotal,
			Honors:           historyHonors,
			ChallengeRecords: []types.SeasonChallengeInfo{},
		},
	}

	if viewerScope == "self" {
		if err := l.fillSelfSeasonData(resp, targetUserId, req); err != nil {
			return nil, err
		}
	}
	return resp, nil
}

func (l *GetHonorWallLogic) canViewFriendHonorWall(requesterId, targetUserId int64) (bool, error) {
	if l.svcCtx.FriendModel == nil {
		return false, nil
	}
	blocked, err := l.svcCtx.FriendModel.HasBlacklistRelation(requesterId, targetUserId)
	if err != nil || blocked {
		return false, err
	}
	return l.svcCtx.FriendModel.AreFriends(requesterId, targetUserId)
}

func (l *GetHonorWallLogic) fillSelfSeasonData(resp *types.GetHonorWallResp, userId int64, req *types.GetHonorWallReq) error {
	gameType := normalizeSeasonChallengeGameType(req.GameType)
	challengeService := NewSeasonChallengeService(l.svcCtx)
	currentSeason, state, currentErr := l.resolveCurrentSeason(time.Now())
	resp.SeasonState = state
	if currentErr != nil {
		l.Logger.Errorf("解析荣誉墙当前赛季失败: err=%v", currentErr)
		resp.SeasonState = seasonx.StateUnavailable
	} else if state == seasonx.StateActive && currentSeason != nil {
		progress, progressErr := challengeService.GetProgress(userId, currentSeason, gameType)
		if progressErr != nil {
			l.Logger.Errorf("加载荣誉墙赛季挑战失败: err=%v", progressErr)
			resp.SeasonState = seasonx.StateUnavailable
		} else {
			resp.CurrentSeason = honorWallSeasonInfo(currentSeason, gameType, seasonChallengeProgressToTypes(progress))
		}
	}

	if req.HistorySeasonId <= 0 {
		return nil
	}
	historySeason, err := l.svcCtx.SeasonModel.FindById(req.HistorySeasonId)
	if err != nil || historySeason == nil {
		return err
	}
	snapshots, err := challengeService.FindArchived(userId, historySeason.Id, gameType)
	if err != nil {
		return err
	}
	if len(snapshots) == 0 {
		return nil
	}
	resp.History.ChallengeSeason = honorWallSeasonInfo(historySeason, gameType, []types.SeasonChallengeInfo{})
	resp.History.ChallengeRecords = seasonChallengeSnapshotsToTypes(snapshots)
	return nil
}

func (l *GetHonorWallLogic) resolveCurrentSeason(now time.Time) (*model.Season, string, error) {
	if l == nil || l.svcCtx == nil || l.svcCtx.SeasonModel == nil {
		return nil, seasonx.StateUnavailable, nil
	}
	policy, err := seasonx.NewPolicy(l.svcCtx.Config.SeasonLifecycle)
	if err != nil {
		return nil, seasonx.StateUnavailable, nil
	}
	if !policy.Enabled {
		current, findErr := l.svcCtx.SeasonModel.FindCurrent()
		if findErr != nil {
			return nil, seasonx.StateUnavailable, findErr
		}
		if current != nil {
			return current, seasonx.StateActive, nil
		}
		return nil, seasonx.StateNotStarted, nil
	}
	if !policy.StartedAt(now) {
		return nil, seasonx.StateNotStarted, nil
	}
	seasons, err := l.svcCtx.SeasonModel.ListAll()
	if err != nil {
		return nil, seasonx.StateUnavailable, err
	}
	resolved := seasonx.Resolve(policy, now, seasons)
	if resolved.Season != nil && resolved.State == seasonx.StateActive {
		current := *resolved.Season
		current.Status = 1
		return &current, resolved.State, nil
	}
	return resolved.Season, resolved.State, nil
}

func honorWallSeasonInfo(season *model.Season, gameType int, challenges []types.SeasonChallengeInfo) *types.HonorWallSeasonInfo {
	if season == nil {
		return nil
	}
	return &types.HonorWallSeasonInfo{
		SeasonId:   season.Id,
		SeasonName: season.Name,
		StartDate:  season.StartDate.Format("2006-01-02"),
		EndDate:    season.EndDate.Format("2006-01-02"),
		Status:     season.Status,
		GameType:   gameType,
		Challenges: challenges,
	}
}

func normalizeHonorHistoryPage(page, pageSize int) (int, int) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	return page, pageSize
}

func emptyHonorWallResponse(success bool) *types.GetHonorWallResp {
	return &types.GetHonorWallResp{
		Success:            success,
		SeasonState:        seasonx.StateNotStarted,
		RecentHonors:       []types.HonorItem{},
		CareerAchievements: []types.AchievementDef{},
		History: types.HonorWallHistory{
			Honors:           []types.HonorItem{},
			ChallengeRecords: []types.SeasonChallengeInfo{},
		},
	}
}
