package achievement

import (
	"sort"
	"time"

	"chasing_points/internal/model"
	"chasing_points/internal/types"
)

type honorCandidate struct {
	item     types.HonorItem
	earnedAt time.Time
	stableId int64
}

func buildRecentHonors(userAchievements []model.UserAchievement, definitions []model.Achievement, titles []model.UserTitle) []types.HonorItem {
	definitionById := make(map[int64]model.Achievement, len(definitions))
	for _, definition := range definitions {
		definitionById[definition.Id] = definition
	}

	candidates := make([]honorCandidate, 0, len(userAchievements)+len(titles))
	for _, unlocked := range userAchievements {
		if unlocked.Unlocked != 1 || unlocked.UnlockedAt == nil {
			continue
		}
		definition, ok := definitionById[unlocked.AchievementId]
		if !ok {
			continue
		}
		candidates = append(candidates, honorCandidate{
			item: types.HonorItem{
				Id:              definition.Id,
				Type:            SourceTypeAchievement,
				Name:            definition.Name,
				Description:     definition.Description,
				Icon:            definition.Icon,
				SourceType:      SourceTypeAchievement,
				SourceRefId:     definition.Id,
				RewardTitleName: definition.RewardTitleName,
				EarnedAt:        formatHonorTime(*unlocked.UnlockedAt),
			},
			earnedAt: *unlocked.UnlockedAt,
			stableId: definition.Id,
		})
	}

	for _, title := range titles {
		if title.SourceType != SourceTypeSeason && title.SourceType != SourceTypeTournament {
			continue
		}
		earnedAt := title.CreatedAt
		if title.GrantedAt != nil {
			earnedAt = *title.GrantedAt
		}
		candidates = append(candidates, honorCandidate{
			item: types.HonorItem{
				Id:            title.Id,
				Type:          title.SourceType,
				Name:          title.TitleName,
				SourceType:    title.SourceType,
				SourceRefId:   title.SourceRefId,
				SourceRefName: title.SourceRefName,
				EarnedAt:      formatHonorTime(earnedAt),
			},
			earnedAt: earnedAt,
			stableId: title.Id,
		})
	}

	sort.SliceStable(candidates, func(i, j int) bool {
		if !candidates[i].earnedAt.Equal(candidates[j].earnedAt) {
			return candidates[i].earnedAt.After(candidates[j].earnedAt)
		}
		return candidates[i].stableId > candidates[j].stableId
	})
	if len(candidates) > 3 {
		candidates = candidates[:3]
	}

	result := make([]types.HonorItem, 0, len(candidates))
	for _, candidate := range candidates {
		result = append(result, candidate.item)
	}
	return result
}

func buildCareerAchievementDefs(definitions []model.Achievement, userAchievements []model.UserAchievement, includeLocked bool) ([]types.AchievementDef, int) {
	progressByAchievement := make(map[int64]model.UserAchievement, len(userAchievements))
	for _, userAchievement := range userAchievements {
		progressByAchievement[userAchievement.AchievementId] = userAchievement
	}

	list := make([]types.AchievementDef, 0, len(definitions))
	unlockedTotal := 0
	for _, definition := range definitions {
		progress := 0
		unlocked := false
		unlockedAt := ""
		if userAchievement, ok := progressByAchievement[definition.Id]; ok {
			progress = userAchievement.Progress
			unlocked = userAchievement.Unlocked == 1
			if unlocked && userAchievement.UnlockedAt != nil {
				unlockedAt = formatHonorTime(*userAchievement.UnlockedAt)
			}
		}
		if unlocked {
			unlockedTotal++
		}
		if !includeLocked && !unlocked {
			continue
		}
		list = append(list, types.AchievementDef{
			Id:          definition.Id,
			Key:         definition.Key,
			Name:        definition.Name,
			Description: definition.Description,
			Icon:        definition.Icon,
			Category:    definition.Category,
			Threshold:   definition.Threshold,
			Progress:    progress,
			Unlocked:    unlocked,
			UnlockedAt:  unlockedAt,
		})
	}
	return list, unlockedTotal
}

func titleToInfo(title *model.UserTitle) *types.TitleInfo {
	if title == nil {
		return nil
	}
	grantedAt := ""
	if title.GrantedAt != nil {
		grantedAt = formatHonorTime(*title.GrantedAt)
	}
	return &types.TitleInfo{
		Id:            title.Id,
		TitleName:     title.TitleName,
		Source:        title.Source,
		SourceType:    title.SourceType,
		SourceRefId:   title.SourceRefId,
		SourceRefName: title.SourceRefName,
		Equipped:      title.Equipped == 1,
		GrantedAt:     grantedAt,
		CreatedAt:     formatHonorTime(title.CreatedAt),
	}
}

func titleToHonorItem(title model.UserTitle) types.HonorItem {
	earnedAt := title.CreatedAt
	if title.GrantedAt != nil {
		earnedAt = *title.GrantedAt
	}
	return types.HonorItem{
		Id:            title.Id,
		Type:          title.SourceType,
		Name:          title.TitleName,
		SourceType:    title.SourceType,
		SourceRefId:   title.SourceRefId,
		SourceRefName: title.SourceRefName,
		EarnedAt:      formatHonorTime(earnedAt),
	}
}

func seasonChallengeProgressToTypes(list []SeasonChallengeProgress) []types.SeasonChallengeInfo {
	result := make([]types.SeasonChallengeInfo, 0, len(list))
	for _, item := range list {
		result = append(result, types.SeasonChallengeInfo{
			Key:         item.Key,
			Name:        item.Name,
			Description: item.Description,
			Threshold:   item.Threshold,
			Progress:    item.Progress,
			Completed:   item.Completed,
		})
	}
	return result
}

func seasonChallengeSnapshotsToTypes(list []model.SeasonChallengeSnapshot) []types.SeasonChallengeInfo {
	result := make([]types.SeasonChallengeInfo, 0, len(list))
	definitionByKey := make(map[string]SeasonChallengeDefinition, len(seasonChallengeDefinitions))
	for _, definition := range seasonChallengeDefinitions {
		definitionByKey[definition.Key] = definition
	}
	for _, item := range list {
		description := ""
		if definition, ok := definitionByKey[item.ChallengeKey]; ok {
			description = definition.Description
		}
		result = append(result, types.SeasonChallengeInfo{
			Key:         item.ChallengeKey,
			Name:        item.ChallengeName,
			Description: description,
			Threshold:   item.Threshold,
			Progress:    item.Progress,
			Completed:   item.Completed == 1,
		})
	}
	return result
}

func formatHonorTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.Format("2006-01-02 15:04:05")
}
