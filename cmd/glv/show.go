package glv

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/codemityio/go/pkg/validator"
	"github.com/codemityio/inspex/internal/browser"
	inthttp "github.com/codemityio/inspex/internal/http"
	"github.com/codemityio/inspex/internal/io"
	"github.com/urfave/cli/v2"
)

func show(ctx *cli.Context) error {
	cfg, err := parseShowConfig(ctx)
	if err != nil {
		return err
	}

	data, err := loadInput(cfg.inputPath)
	if err != nil {
		return err
	}

	content, err := applyFilters(data, cfg)
	if err != nil {
		return err
	}

	if cfg.http {
		if e := inthttp.Serve(ctx.Context, func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")

			if e := tableTemplate.Execute(w, template.JS(content)); e != nil { //nolint:gosec
				http.Error(w, "error: "+e.Error(), http.StatusInternalServerError)
			}
		}, cfg.httpPort, browser.Open); e != nil {
			return fmt.Errorf("%w: %w", errServe, e)
		}
	}

	if e := tableTemplate.Execute(os.Stdout, data); e != nil {
		return fmt.Errorf("%w: %w", errWrite, e)
	}

	return nil
}

// parseShowConfig validates CLI args and returns a config struct.
func parseShowConfig(ctx *cli.Context) (*showConfig, error) {
	cfg := showConfig{
		inputPath:       ctx.String("input-path"),
		http:            ctx.Bool("http"),
		httpPort:        ctx.Int("http-port"),
		excludeDates:    ctx.StringSlice("exclude-date"),
		includeDates:    ctx.StringSlice("include-date"),
		includeTimeFrom: ctx.Timestamp("include-time-from"),
		includeTimeTo:   ctx.Timestamp("include-time-to"),
	}

	if len(cfg.excludeDates) > 0 && len(cfg.includeDates) > 0 {
		return nil, fmt.Errorf(
			"%w: --exclude-date and --include-date are mutually exclusive",
			errArgs,
		)
	}

	fromSet := cfg.includeTimeFrom != nil && !cfg.includeTimeFrom.IsZero()
	toSet := cfg.includeTimeTo != nil && !cfg.includeTimeTo.IsZero()

	if fromSet != toSet {
		return nil, fmt.Errorf(
			"%w: --include-time-from and --include-time-to must be provided together",
			errArgs,
		)
	}

	return &cfg, nil
}

// loadInput reads from a file path or stdin.
func loadInput(path string) ([]byte, error) {
	var (
		data []byte
		err  error
	)

	if path != "" {
		//nolint:gosec
		data, err = os.ReadFile(path)
	} else {
		data, err = io.ReadStdin()
	}

	if err != nil {
		return nil, fmt.Errorf("%w:%w", errRead, err)
	}

	errs, err := validator.New().ValidateJSON(schema, data)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", errValidate, err)
	}

	if len(errs) > 0 {
		return nil, fmt.Errorf("%w: %v", errValidate, errs)
	}

	return data, nil
}

// applyFilters applies date and time filters to the data.
func applyFilters(data []byte, cfg *showConfig) ([]byte, error) {
	raw, err := parseRaw(data)
	if err != nil {
		return nil, err
	}

	switch {
	case len(cfg.excludeDates) > 0:
		raw = filterDates(raw, cfg.excludeDates, false)
	case len(cfg.includeDates) > 0:
		raw = filterDates(raw, cfg.includeDates, true)
	}

	if cfg.includeTimeFrom != nil || cfg.includeTimeTo != nil {
		raw = filterTimeRange(raw, cfg.includeTimeFrom, cfg.includeTimeTo)
	}

	return marshalRaw(raw)
}

// parseRaw unmarshals the outer map once.
func parseRaw(data []byte) (map[string][]json.RawMessage, error) {
	var raw map[string][]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("%w: unmarshal: %w", errMarshal, err)
	}

	return raw, nil
}

// marshalRaw marshals the filtered map once at the very end.
func marshalRaw(raw map[string][]json.RawMessage) ([]byte, error) {
	out, err := json.MarshalIndent(raw, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("%w: marshal: %w", errMarshal, err)
	}

	return out, nil
}

// filterDates filters FileChange entries by date.
// When includeOnly is false, entries matching dates are removed (exclude mode).
// When includeOnly is true, only entries matching dates are kept (include mode).
// Files with no remaining commits are dropped entirely.
func filterDates(
	raw map[string][]json.RawMessage,
	dates []string,
	includeOnly bool,
) map[string][]json.RawMessage {
	set := make(map[string]struct{}, len(dates))
	for _, d := range dates {
		set[d] = struct{}{}
	}

	return applyFilter(raw, func(fc fileChangeDateTime) bool {
		_, inSet := set[fc.Date]

		return (includeOnly && inSet) || (!includeOnly && !inSet)
	})
}

func filterTimeRange(
	raw map[string][]json.RawMessage,
	includeTimeFrom, includeTimeTo *time.Time,
) map[string][]json.RawMessage {
	from := time.Date(0, 1, 1, 0, 0, 0, 0, time.UTC)
	if includeTimeFrom != nil {
		from = *includeTimeFrom
	}

	to := time.Date(0, 1, 1, 23, 59, 59, 0, time.UTC)
	if includeTimeTo != nil {
		to = *includeTimeTo
	}

	return applyFilter(raw, func(fc fileChangeDateTime) bool {
		if fc.Time == "" {
			return true
		}

		t, err := time.Parse("15:04:05", timeOfDay(fc.Time))
		if err != nil {
			return true
		}

		return !t.Before(from) && !t.After(to)
	})
}

// applyFilter applies a predicate to a parsed map, returning a new map with
// only the entries (and files) that pass. Entries that fail to unmarshal are kept unconditionally.
func applyFilter(
	raw map[string][]json.RawMessage,
	keep func(fc fileChangeDateTime) bool,
) map[string][]json.RawMessage {
	filtered := make(map[string][]json.RawMessage, len(raw))

	for file, changes := range raw {
		var kept []json.RawMessage

		for _, entry := range changes {
			var fc fileChangeDateTime
			if err := json.Unmarshal(entry, &fc); err != nil {
				kept = append(kept, entry)

				continue
			}

			if keep(fc) {
				kept = append(kept, entry)
			}
		}

		if len(kept) > 0 {
			filtered[file] = kept
		}
	}

	return filtered
}

// timeOfDay extracts the "HH:MM:SS" portion from an ISO timestamp "2006-01-02T15:04:05".
// Returns empty string if the format is unexpected.
func timeOfDay(iso string) string {
	tIdx := strings.IndexByte(iso, 'T')
	if tIdx < 0 || len(iso) < tIdx+9 {
		return ""
	}

	return iso[tIdx+1 : tIdx+9]
}
