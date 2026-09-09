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
		backfill          bool
		season            int
		year              int
		fromText          string
		toText            string
		publish           bool
		dryRun            bool
		includeQualifiers bool
		gameType          int
	)

	fs.BoolVar(&backfill, "backfill", false, "historical backfill mode (default from 2023-01-01 to today UTC)")
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

	params := SyncParams{
		Backfill:          backfill,
		Publish:           publish,
		DryRun:            dryRun,
		IncludeQualifiers: includeQualifiers,
		GameType:          gameType,
	}

	if backfill {
		return buildBackfillParams(params, fromText, toText, gameType, season, year)
	}

	return buildLegacyParams(params, season, year, fromText, toText)
}

func buildBackfillParams(params SyncParams, fromText, toText string, gameType, season, year int) (SyncParams, error) {
	if season > 0 || year > 0 {
		return SyncParams{}, fmt.Errorf("--backfill is mutually exclusive with --season and --year")
	}
	if gameType != 1 {
		return SyncParams{}, fmt.Errorf("--backfill only supports snooker (game-type=1)")
	}

	now := time.Now().UTC()
	defaultFrom := time.Date(2023, time.January, 1, 0, 0, 0, 0, time.UTC)
	defaultTo := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)

	from, err := parseDateOnly(fromText)
	if err != nil {
		return SyncParams{}, fmt.Errorf("invalid --from date: %w", err)
	}
	to, err := parseDateOnly(toText)
	if err != nil {
		return SyncParams{}, fmt.Errorf("invalid --to date: %w", err)
	}

	// Apply defaults for unspecified endpoints.
	if from == nil {
		from = &defaultFrom
	}
	if to == nil {
		to = &defaultTo
	}

	if to.Before(*from) {
		return SyncParams{}, fmt.Errorf("--to must not be before --from")
	}

	params.Mode = SyncModeBackfill
	params.From = from
	params.To = to
	return params, nil
}

func buildLegacyParams(params SyncParams, season, year int, fromText, toText string) (SyncParams, error) {
	modeCount := 0

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
