package wstsync

import "time"

type SyncMode string

const (
	SyncModeSeason SyncMode = "season"
	SyncModeYear   SyncMode = "year"
	SyncModeRange  SyncMode = "range"

	wstSourceType               = "official"
	wstSourceName               = "WST"
	wstSiteBaseURL              = "https://www.wst.tv"
	defaultTournamentCoverImage = "https://images.gc.wstservices.co.uk/fit-in/400x600/4ddad400-99d3-11ee-94e8-c9d138e537ff.png"
	defaultMatchesPageSize      = 200
)

type SyncParams struct {
	Mode              SyncMode
	Season            int
	Year              int
	From              *time.Time
	To                *time.Time
	Publish           bool
	DryRun            bool
	IncludeQualifiers bool
	GameType          int
}

type DateWindow struct {
	From time.Time
	To   time.Time
}

func (w DateWindow) Empty() bool {
	return w.From.IsZero() && w.To.IsZero()
}

type SeasonListResponse struct {
	Data []SeasonResource `json:"data"`
}

type SeasonResource struct {
	ID         string           `json:"id"`
	Attributes SeasonAttributes `json:"attributes"`
}

type SeasonAttributes struct {
	Name string `json:"name"`
}

type TournamentListResponse struct {
	Data []TournamentResource `json:"data"`
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
	Season          TournamentSeason `json:"season"`
}

type TournamentSeason struct {
	ID string `json:"id"`
}

type MatchListResponse struct {
	Data []MatchResource `json:"data"`
}

type MatchResource struct {
	ID         string          `json:"id"`
	Attributes MatchAttributes `json:"attributes"`
}

type MatchAttributes struct {
	Name             string        `json:"name"`
	HomePlayerID     string        `json:"homePlayerID"`
	HomePlayerScore  int           `json:"homePlayerScore"`
	AwayPlayerID     string        `json:"awayPlayerID"`
	AwayPlayerScore  int           `json:"awayPlayerScore"`
	TournamentID     string        `json:"tournamentID"`
	StartDateTime    string        `json:"startDateTime"`
	Round            string        `json:"round"`
	Status           string        `json:"status"`
	NumberOfFrames   int           `json:"numberOfFrames"`
	FixtureNumber    int           `json:"fixtureNumber"`
	PlayersAllocated bool          `json:"playersAllocated"`
	Published        bool          `json:"published"`
	HomePlayer       WstPlayer     `json:"homePlayer"`
	AwayPlayer       WstPlayer     `json:"awayPlayer"`
	Tournament       WstTournament `json:"tournament"`
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
