package wstsync

import (
	"flag"
	"fmt"
	"io"
	"time"
)

func ParseSyncParams(args []string) (SyncParams, error) {
	fs := flag.NewFlagSet("wst-sync", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var (
		season            int
		year              int
		fromText          string
		toText            string
		publish           bool
		dryRun            bool
		includeQualifiers bool
		gameType          int
	)

	fs.IntVar(&season, "season", 0, "WST season id")
	fs.IntVar(&year, "year", 0, "calendar year")
	fs.StringVar(&fromText, "from", "", "start date")
	fs.StringVar(&toText, "to", "", "end date")
	fs.BoolVar(&publish, "publish", true, "publish to event news")
	fs.BoolVar(&dryRun, "dry-run", false, "dry run only")
	fs.BoolVar(&includeQualifiers, "include-qualifiers", true, "include qualifier events")
	fs.IntVar(&gameType, "game-type", 1, "game type")

	if err := fs.Parse(args); err != nil {
		return SyncParams{}, err
	}
	if fs.NArg() != 0 {
		return SyncParams{}, fmt.Errorf("unexpected positional arguments: %v", fs.Args())
	}

	modeCount := 0
	params := SyncParams{
		Publish:           publish,
		DryRun:            dryRun,
		IncludeQualifiers: includeQualifiers,
		GameType:          gameType,
	}

	if season > 0 {
		modeCount++
		params.Mode = SyncModeSeason
		params.Season = season
	}
	if year > 0 {
		modeCount++
		params.Mode = SyncModeYear
		params.Year = year
	}
	if fromText != "" || toText != "" {
		modeCount++
		params.Mode = SyncModeRange
		from, to, err := parseSyncDateWindow(fromText, toText)
		if err != nil {
			return SyncParams{}, err
		}
		params.From = from
		params.To = to
	}

	if modeCount == 0 {
		return SyncParams{}, fmt.Errorf("season, year, or from/to must be specified")
	}
	if modeCount > 1 {
		return SyncParams{}, fmt.Errorf("season, year, and from/to are mutually exclusive")
	}

	if params.Mode == SyncModeYear {
		window, err := buildYearWindow(params.Year)
		if err != nil {
			return SyncParams{}, err
		}
		params.From = &window.From
		params.To = &window.To
	}

	return params, nil
}

func parseSyncDateWindow(fromText, toText string) (*time.Time, *time.Time, error) {
	from, err := parseDateOnly(fromText)
	if err != nil {
		return nil, nil, err
	}
	to, err := parseDateOnly(toText)
	if err != nil {
		return nil, nil, err
	}
	if from == nil || to == nil {
		return nil, nil, fmt.Errorf("both from and to must be provided")
	}
	if to.Before(*from) {
		return nil, nil, fmt.Errorf("to date must not be before from date")
	}
	return from, to, nil
}
