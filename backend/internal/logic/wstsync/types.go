package wstsync

import (
	"bytes"
	"encoding/json"
	"time"
)

type SyncMode string

const (
	SyncModeSeason   SyncMode = "season"
	SyncModeYear     SyncMode = "year"
	SyncModeRange    SyncMode = "range"
	SyncModeBackfill SyncMode = "backfill"

	DefaultSeasonsURL           = "https://seasons.snooker.web.gc.wstservices.co.uk"
	DefaultTournamentsURL       = "https://tournaments.snooker.web.gc.wstservices.co.uk"
	DefaultMatchesURL           = "https://matches.snooker.web.gc.wstservices.co.uk"
	wstSourceType               = "official"
	wstSourceName               = "WST"
	wstSiteBaseURL              = "https://www.wst.tv"
	defaultTournamentCoverImage = "https://images.gc.wstservices.co.uk/fit-in/400x600/4ddad400-99d3-11ee-94e8-c9d138e537ff.png"
	defaultMatchesPageSize      = 200

	// BackfillStatus* are summary status values for backfill mode.
	BackfillStatusCompleted   = "completed"
	BackfillStatusNeedsReview = "needs_review"
	BackfillStatusFailed      = "failed"

	// Exit codes for the backfill CLI.
	ExitCodeCompleted   = 0
	ExitCodeFailed      = 1
	ExitCodeNeedsReview = 2
)

type SyncParams struct {
	Mode              SyncMode
	Season            int
	Year              int
	From              *time.Time
	To                *time.Time
	Backfill          bool
	Publish           bool
	DryRun            bool
	IncludeQualifiers bool
	GameType          int
}

type BackfillYearSummary struct {
	Year        int
	Tournaments int
	Matches     int
}

type DateWindow struct {
	From time.Time
	To   time.Time
}

func (w DateWindow) Empty() bool {
	return w.From.IsZero() && w.To.IsZero()
}

func intPtr(v int) *int { return &v }

type SeasonListResponse struct {
	Data  []SeasonResource `json:"data"`
	Meta  *PaginationMeta  `json:"meta"`
	Links *PaginationLinks `json:"links"`
}

type PaginationMeta struct {
	TotalCount *int `json:"totalCount"`
	Count      *int `json:"count"`
}

type PaginationLinks struct {
	Next *string `json:"next"`
	Last *string `json:"last"`
}

type SeasonResource struct {
	ID         string           `json:"id"`
	Attributes SeasonAttributes `json:"attributes"`
}

type SeasonAttributes struct {
	Name string `json:"name"`
}

type TournamentListResponse struct {
	Data  []TournamentResource `json:"data"`
	Meta  *PaginationMeta      `json:"meta"`
	Links *PaginationLinks     `json:"links"`
}

type TournamentResource struct {
	ID         string               `json:"id"`
	Attributes TournamentAttributes `json:"attributes"`
}

type TournamentAttributes struct {
	Name            string           `json:"name"`
	StartDate       string           `json:"startDate"`
	EndDate         string           `json:"endDate"`
	City            string           `json:"city"`
	Country         string           `json:"country"`
	InformationPage string           `json:"informationPage"`
	TicketingLink   string           `json:"ticketingLink"`
	MatchCount      *int             `json:"matchCount"`
	DatedMatchCount *int             `json:"matchesWithStartDateCount"`
	Season          TournamentSeason `json:"season"`
}

type TournamentSeason struct {
	ID string `json:"id"`
}

type MatchListResponse struct {
	Data  []MatchResource  `json:"data"`
	Meta  *PaginationMeta  `json:"meta"`
	Links *PaginationLinks `json:"links"`
}

type MatchResource struct {
	ID         string          `json:"id"`
	Attributes MatchAttributes `json:"attributes"`
}

type MatchAttributes struct {
	Name                    string        `json:"name"`
	HomePlayerID            string        `json:"homePlayerID"`
	HomePlayerScore         *int          `json:"homePlayerScore"`
	AwayPlayerID            string        `json:"awayPlayerID"`
	AwayPlayerScore         *int          `json:"awayPlayerScore"`
	TournamentID            string        `json:"tournamentID"`
	StartDateTime           string        `json:"startDateTime"`
	Round                   string        `json:"round"`
	Status                  string        `json:"status"`
	NumberOfFrames          int           `json:"numberOfFrames"`
	FixtureNumber           int           `json:"fixtureNumber"`
	PlayersAllocated        bool          `json:"playersAllocated"`
	PlayersAllocatedPresent bool          `json:"-"`
	Published               bool          `json:"published"`
	HomePlayer              WstPlayer     `json:"homePlayer"`
	AwayPlayer              WstPlayer     `json:"awayPlayer"`
	Tournament              WstTournament `json:"tournament"`
}

func (m *MatchAttributes) UnmarshalJSON(data []byte) error {
	type matchAttributes MatchAttributes
	var decoded matchAttributes
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return err
	}
	raw, ok := fields["playersAllocated"]
	decoded.PlayersAllocatedPresent = ok && !bytes.Equal(bytes.TrimSpace(raw), []byte("null"))
	*m = MatchAttributes(decoded)
	return nil
}

type WstPlayer struct {
	PlayerID    string         `json:"playerID"`
	FirstName   string         `json:"firstName"`
	Surname     string         `json:"surname"`
	CountryCode string         `json:"countryCode"`
	Media       WstPlayerMedia `json:"media"`
}

type WstPlayerMedia struct {
	Profile    string `json:"profile"`
	AppProfile string `json:"appProfile"`
	LeftSide   string `json:"leftSide"`
	RightSide  string `json:"rightSide"`
}

type WstTournament struct {
	TournamentID    string `json:"tournamentID"`
	Name            string `json:"name"`
	StartDate       string `json:"startDate"`
	EndDate         string `json:"endDate"`
	City            string `json:"city"`
	Country         string `json:"country"`
	InformationPage string `json:"informationPage"`
	TicketingLink   string `json:"ticketingLink"`
}
