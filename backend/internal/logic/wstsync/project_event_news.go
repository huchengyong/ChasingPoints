package wstsync

import (
	"strconv"
	"strings"
	"time"

	"chasing_points/internal/model"
)

const ()

type EventNewsProjector struct {
	eventNewsModel *model.EventNewsModel
}

func NewEventNewsProjector(eventNewsModel *model.EventNewsModel) *EventNewsProjector {
	return &EventNewsProjector{eventNewsModel: eventNewsModel}
}

func (p *EventNewsProjector) ProjectTournamentEventNews(tournament model.Tournament, matches []model.TournamentMatch, publish bool) (*model.EventNews, error) {
	if p == nil || p.eventNewsModel == nil {
		return nil, nil
	}

	existing, err := p.eventNewsModel.FindByTournamentAndSourceType(tournament.Id, wstSourceType)
	if err != nil {
		return nil, err
	}

	var item *model.EventNews
	now := time.Now()
	if existing != nil {
		item = existing
	} else {
		item = &model.EventNews{
			TournamentId: tournament.Id,
			SourceType:   wstSourceType,
			SourceName:   wstSourceName,
			CreatedAt:    now,
		}
	}

	item.Title = strings.TrimSpace(tournament.Name)
	item.GameType = tournament.GameType
	item.SourceUrl = strings.TrimSpace(tournament.InformationPage)
	item.CoverImage = strings.TrimSpace(tournament.CoverImage)
	item.Country = strings.TrimSpace(tournament.Country)
	item.City = strings.TrimSpace(tournament.City)
	item.Venue = strings.TrimSpace(tournament.VenueName)
	item.StartDate = tournament.StartDate
	item.EndDate = tournament.EndDate
	item.StartTime = tournament.StartTime
	item.EndTime = tournament.EndTime
	item.Status = tournament.Status
	item.SortTime = resolveTournamentSortTime(tournament)
	item.Published = publish
	item.Summary = buildTournamentSummary(matches)
	item.UpdatedAt = now
	if publish {
		item.PublishedAt = &now
	} else {
		item.PublishedAt = nil
	}

	if item.Id == 0 {
		if err := p.eventNewsModel.Create(item); err != nil {
			return nil, err
		}
		return item, nil
	}

	if err := p.eventNewsModel.Update(item); err != nil {
		return nil, err
	}
	return item, nil
}

func resolveTournamentSortTime(tournament model.Tournament) *time.Time {
	switch {
	case tournament.StartTime != nil:
		return tournament.StartTime
	case tournament.StartDate != nil:
		return tournament.StartDate
	case tournament.LastSyncedAt != nil:
		return tournament.LastSyncedAt
	default:
		return nil
	}
}

func buildTournamentSummary(matches []model.TournamentMatch) string {
	matchCount := len(matches)
	if matchCount == 0 {
		return "赛事赛程已收录，比赛结果待同步"
	}

	currentRound := strings.TrimSpace(resolveCurrentRoundName(matches))
	if currentRound == "" {
		currentRound = "轮次待更新"
	}

	return "共 " + strconv.Itoa(matchCount) + " 场比赛，当前轮次 " + currentRound
}

func resolveCurrentRoundName(matches []model.TournamentMatch) string {
	if len(matches) == 0 {
		return ""
	}

	pickByStatus := func(status int) string {
		bestRound := ""
		bestOrder := -1
		for _, item := range matches {
			if item.Status != status {
				continue
			}
			order := item.RoundOrder
			if order > bestOrder {
				bestOrder = order
				bestRound = strings.TrimSpace(item.RoundName)
			}
		}
		return bestRound
	}

	for _, status := range []int{model.EventNewsStatusLive, model.EventNewsStatusUpcoming, model.EventNewsStatusFinished} {
		if round := pickByStatus(status); round != "" {
			return round
		}
	}
	return strings.TrimSpace(matches[0].RoundName)
}
