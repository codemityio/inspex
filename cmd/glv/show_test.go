package glv

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v2"
)

func makeJSON(t *testing.T, v any) []byte {
	t.Helper()

	b, err := json.Marshal(v)
	require.NoError(t, err)

	return b
}

func entry(t *testing.T, date, tim string) json.RawMessage {
	t.Helper()

	return makeJSON(t, map[string]string{
		"date":    date,
		"time":    tim,
		"message": "test commit",
	})
}

func decodeResult(t *testing.T, raw map[string][]json.RawMessage) map[string][]fileChangeDateTime {
	t.Helper()

	out := make(map[string][]fileChangeDateTime, len(raw))
	for file, entries := range raw {
		fcs := make([]fileChangeDateTime, 0, len(entries))
		for _, e := range entries {
			var fc fileChangeDateTime
			require.NoError(t, json.Unmarshal(e, &fc))
			fcs = append(fcs, fc)
		}

		out[file] = fcs
	}

	return out
}

func parseTime(s string) *time.Time {
	if s == "" {
		return nil
	}

	t, err := time.Parse("15:04:05", s)
	if err != nil {
		panic(fmt.Sprintf("parseTime: invalid time %q: %v", s, err))
	}

	return &t
}

func TestTimeOfDay(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"full ISO timestamp", "2024-03-10T14:32:05", "14:32:05"},
		{"midnight", "2024-01-01T00:00:00", "00:00:00"},
		{"end of day", "2024-12-31T23:59:59", "23:59:59"},
		{"empty string", "", ""},
		{"no T separator", "2024-03-10 14:32:05", ""},
		{"too short after T", "2024-03-10T14:3", ""},
		{"date only", "2024-03-10", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, timeOfDay(tt.input))
		})
	}
}

func TestParseRaw(t *testing.T) {
	tests := []struct {
		name    string
		input   []byte
		wantErr error
		check   func(t *testing.T, result map[string][]json.RawMessage)
	}{
		{
			name:  "valid input with one file and one entry",
			input: makeJSON(t, map[string][]map[string]string{"a.go": {{"date": "2024-01-01"}}}),
			check: func(t *testing.T, result map[string][]json.RawMessage) {
				t.Helper()

				require.Contains(t, result, "a.go")
				assert.Len(t, result["a.go"], 1)
			},
		},
		{
			name:  "empty object",
			input: []byte(`{}`),
			check: func(t *testing.T, result map[string][]json.RawMessage) {
				t.Helper()

				assert.Empty(t, result)
			},
		},
		{
			name:    "invalid JSON returns error",
			input:   []byte(`not json`),
			wantErr: errMarshal,
		},
		{
			name:    "wrong shape — array instead of object",
			input:   []byte(`[]`),
			wantErr: errMarshal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseRaw(tt.input)

			require.ErrorIs(t, err, tt.wantErr)

			if tt.check != nil {
				tt.check(t, result)
			}
		})
	}
}

func TestMarshalRaw(t *testing.T) {
	tests := []struct {
		name  string
		input map[string][]json.RawMessage
		check func(t *testing.T, out []byte)
	}{
		{
			name:  "empty map produces valid JSON object",
			input: map[string][]json.RawMessage{},
			check: func(t *testing.T, out []byte) {
				t.Helper()

				assert.JSONEq(t, `{}`, string(out))
			},
		},
		{
			name: "entries round-trip without data loss",
			input: map[string][]json.RawMessage{
				"a.go": {json.RawMessage(`{"date":"2024-01-01","message":"msg"}`)},
			},
			check: func(t *testing.T, out []byte) {
				t.Helper()

				var decoded map[string][]map[string]string
				require.NoError(t, json.Unmarshal(out, &decoded))
				require.Contains(t, decoded, "a.go")
				assert.Equal(t, "2024-01-01", decoded["a.go"][0]["date"])
			},
		},
		{
			name: "output is indented",
			input: map[string][]json.RawMessage{
				"a.go": {json.RawMessage(`{"date":"2024-01-01"}`)},
			},
			check: func(t *testing.T, out []byte) {
				t.Helper()

				assert.Contains(t, string(out), "\n")
				assert.Contains(t, string(out), "  ")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, err := marshalRaw(tt.input)
			require.NoError(t, err)
			tt.check(t, out)
		})
	}
}

func TestApplyFilter(t *testing.T) {
	tests := []struct {
		name        string
		input       map[string][]json.RawMessage
		keep        func(fc fileChangeDateTime) bool
		wantEntries map[string]int
		wantAbsent  []string
	}{
		{
			name: "keep-all predicate retains everything",
			input: map[string][]json.RawMessage{
				"a.go": {entry(t, "2024-01-01", "2024-01-01T09:00:00")},
				"b.go": {entry(t, "2024-01-02", "2024-01-02T09:00:00")},
			},
			keep:        func(_ fileChangeDateTime) bool { return true },
			wantEntries: map[string]int{"a.go": 1, "b.go": 1},
		},
		{
			name: "keep-none predicate drops all files",
			input: map[string][]json.RawMessage{
				"a.go": {entry(t, "2024-01-01", "2024-01-01T09:00:00")},
			},
			keep:       func(_ fileChangeDateTime) bool { return false },
			wantAbsent: []string{"a.go"},
		},
		{
			name: "entry that fails to unmarshal is kept unconditionally",
			input: map[string][]json.RawMessage{
				"a.go": {json.RawMessage(`not valid json`)},
			},
			keep:        func(_ fileChangeDateTime) bool { return false },
			wantEntries: map[string]int{"a.go": 1},
		},
		{
			name: "file key dropped when all entries filtered out",
			input: map[string][]json.RawMessage{
				"a.go": {entry(t, "2024-01-01", "2024-01-01T09:00:00")},
				"b.go": {entry(t, "2024-01-02", "2024-01-02T09:00:00")},
			},
			keep: func(fc fileChangeDateTime) bool {
				return fc.Date == "2024-01-02"
			},
			wantEntries: map[string]int{"b.go": 1},
			wantAbsent:  []string{"a.go"},
		},
		{
			name: "mixed valid and invalid entries in same file",
			input: map[string][]json.RawMessage{
				"a.go": {
					json.RawMessage(`not valid json`),
					entry(t, "2024-01-01", "2024-01-01T09:00:00"),
				},
			},
			// invalid entry is kept unconditionally, valid entry is filtered out
			keep:        func(_ fileChangeDateTime) bool { return false },
			wantEntries: map[string]int{"a.go": 1},
		},
		{
			name:        "empty input returns empty result",
			input:       map[string][]json.RawMessage{},
			keep:        func(_ fileChangeDateTime) bool { return true },
			wantEntries: map[string]int{},
		},
		{
			name: "multiple entries partially filtered",
			input: map[string][]json.RawMessage{
				"a.go": {
					entry(t, "2024-01-01", "2024-01-01T09:00:00"),
					entry(t, "2024-01-02", "2024-01-02T09:00:00"),
					entry(t, "2024-01-03", "2024-01-03T09:00:00"),
				},
			},
			keep: func(fc fileChangeDateTime) bool {
				return fc.Date != "2024-01-02"
			},
			wantEntries: map[string]int{"a.go": 2},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := applyFilter(tt.input, tt.keep)

			for file, wantCount := range tt.wantEntries {
				if assert.Contains(t, result, file) {
					assert.Len(t, result[file], wantCount)
				}
			}

			for _, f := range tt.wantAbsent {
				assert.NotContains(t, result, f)
			}
		})
	}
}

func TestApplyFilters(t *testing.T) {
	buildInput := func(t *testing.T) []byte {
		t.Helper()

		return makeJSON(t, map[string][]map[string]string{
			"a.go": {
				{"date": "2024-01-01", "time": "2024-01-01T09:00:00", "message": "m"},
				{"date": "2024-01-02", "time": "2024-01-02T14:00:00", "message": "m"},
				{"date": "2024-01-03", "time": "2024-01-03T20:00:00", "message": "m"},
			},
			"b.go": {
				{"date": "2024-01-01", "time": "2024-01-01T10:00:00", "message": "m"},
			},
		})
	}

	tests := []struct {
		name        string
		cfg         func() *showConfig
		wantErr     bool
		wantEntries map[string]int
		wantAbsent  []string
	}{
		{
			name:        "no filters returns all data unchanged",
			cfg:         func() *showConfig { return &showConfig{} },
			wantEntries: map[string]int{"a.go": 3, "b.go": 1},
		},
		{
			name: "exclude date removes matching entries",
			cfg: func() *showConfig {
				return &showConfig{excludeDates: []string{"2024-01-01"}}
			},
			wantEntries: map[string]int{"a.go": 2},
			wantAbsent:  []string{"b.go"},
		},
		{
			name: "include date keeps only matching entries",
			cfg: func() *showConfig {
				return &showConfig{includeDates: []string{"2024-01-02"}}
			},
			wantEntries: map[string]int{"a.go": 1},
			wantAbsent:  []string{"b.go"},
		},
		{
			name: "time range filter keeps only in-range entries",
			cfg: func() *showConfig {
				from := parseTime("12:00:00")
				to := parseTime("18:00:00")

				return &showConfig{includeTimeFrom: from, includeTimeTo: to}
			},
			wantEntries: map[string]int{"a.go": 1},
			wantAbsent:  []string{"b.go"},
		},
	}

	data := buildInput(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, err := applyFilters(data, tt.cfg())
			require.NoError(t, err)

			var decoded map[string][]json.RawMessage
			require.NoError(t, json.Unmarshal(out, &decoded))

			for file, wantCount := range tt.wantEntries {
				if assert.Contains(t, decoded, file) {
					assert.Len(t, decoded[file], wantCount)
				}
			}

			for _, f := range tt.wantAbsent {
				assert.NotContains(t, decoded, f)
			}
		})
	}

	t.Run("invalid JSON input returns error", func(t *testing.T) {
		_, err := applyFilters([]byte(`not json`), &showConfig{})
		require.Error(t, err)
	})
}

func TestFilterDates(t *testing.T) {
	tests := []struct {
		name        string
		input       map[string][]json.RawMessage
		dates       []string
		includeOnly bool
		wantEntries map[string]int
		wantAbsent  []string
	}{
		{
			name: "exclude mode removes matching date",
			input: map[string][]json.RawMessage{
				"a.go": {entry(t, "2024-01-01", "2024-01-01T09:00:00")},
				"b.go": {entry(t, "2024-01-02", "2024-01-02T09:00:00")},
			},
			dates:       []string{"2024-01-01"},
			includeOnly: false,
			wantEntries: map[string]int{"b.go": 1},
			wantAbsent:  []string{"a.go"},
		},
		{
			name: "exclude mode keeps non-matching",
			input: map[string][]json.RawMessage{
				"a.go": {
					entry(t, "2024-01-01", "2024-01-01T09:00:00"),
					entry(t, "2024-01-03", "2024-01-03T09:00:00"),
				},
			},
			dates:       []string{"2024-01-02"},
			includeOnly: false,
			wantEntries: map[string]int{"a.go": 2},
		},
		{
			name: "include mode keeps only matching date",
			input: map[string][]json.RawMessage{
				"a.go": {
					entry(t, "2024-01-01", "2024-01-01T09:00:00"),
					entry(t, "2024-01-02", "2024-01-02T09:00:00"),
				},
			},
			dates:       []string{"2024-01-01"},
			includeOnly: true,
			wantEntries: map[string]int{"a.go": 1},
		},
		{
			name: "file dropped when include matches nothing",
			input: map[string][]json.RawMessage{
				"a.go": {entry(t, "2024-01-01", "2024-01-01T09:00:00")},
			},
			dates:       []string{"2024-02-01"},
			includeOnly: true,
			wantAbsent:  []string{"a.go"},
		},
		{
			name: "multiple dates in include set",
			input: map[string][]json.RawMessage{
				"a.go": {
					entry(t, "2024-01-01", "2024-01-01T09:00:00"),
					entry(t, "2024-01-02", "2024-01-02T09:00:00"),
					entry(t, "2024-01-03", "2024-01-03T09:00:00"),
				},
			},
			dates:       []string{"2024-01-01", "2024-01-03"},
			includeOnly: true,
			wantEntries: map[string]int{"a.go": 2},
		},
		{
			name: "multiple dates in exclude set",
			input: map[string][]json.RawMessage{
				"a.go": {
					entry(t, "2024-01-01", "2024-01-01T09:00:00"),
					entry(t, "2024-01-02", "2024-01-02T09:00:00"),
					entry(t, "2024-01-03", "2024-01-03T09:00:00"),
				},
			},
			dates:       []string{"2024-01-01", "2024-01-03"},
			includeOnly: false,
			wantEntries: map[string]int{"a.go": 1},
		},
		{
			name:        "empty input returns empty result",
			input:       map[string][]json.RawMessage{},
			dates:       []string{"2024-01-01"},
			includeOnly: false,
			wantEntries: map[string]int{},
		},
		{
			name: "empty date set in exclude mode keeps everything",
			input: map[string][]json.RawMessage{
				"a.go": {entry(t, "2024-01-01", "2024-01-01T09:00:00")},
			},
			dates:       []string{},
			includeOnly: false,
			wantEntries: map[string]int{"a.go": 1},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := decodeResult(t, filterDates(tt.input, tt.dates, tt.includeOnly))

			for file, wantCount := range tt.wantEntries {
				if assert.Contains(t, result, file) {
					assert.Len(t, result[file], wantCount)
				}
			}

			for _, f := range tt.wantAbsent {
				assert.NotContains(t, result, f)
			}
		})
	}
}

func TestFilterTimeRange(t *testing.T) {
	tests := []struct {
		name        string
		input       map[string][]json.RawMessage
		timeFrom    *time.Time
		timeTo      *time.Time
		wantEntries map[string]int
		wantAbsent  []string
	}{
		{
			name: "keeps entry within range",
			input: map[string][]json.RawMessage{
				"a.go": {entry(t, "2024-01-01", "2024-01-01T10:00:00")},
			},
			timeFrom:    parseTime("09:00:00"),
			timeTo:      parseTime("18:00:00"),
			wantEntries: map[string]int{"a.go": 1},
		},
		{
			name: "drops entry before range",
			input: map[string][]json.RawMessage{
				"a.go": {entry(t, "2024-01-01", "2024-01-01T07:59:00")},
			},
			timeFrom:   parseTime("08:00:00"),
			timeTo:     parseTime("18:00:00"),
			wantAbsent: []string{"a.go"},
		},
		{
			name: "drops entry after range",
			input: map[string][]json.RawMessage{
				"a.go": {entry(t, "2024-01-01", "2024-01-01T18:01:00")},
			},
			timeFrom:   parseTime("08:00:00"),
			timeTo:     parseTime("18:00:00"),
			wantAbsent: []string{"a.go"},
		},
		{
			name: "keeps entry exactly at lower bound",
			input: map[string][]json.RawMessage{
				"a.go": {entry(t, "2024-01-01", "2024-01-01T08:00:00")},
			},
			timeFrom:    parseTime("08:00:00"),
			timeTo:      parseTime("18:00:00"),
			wantEntries: map[string]int{"a.go": 1},
		},
		{
			name: "keeps entry exactly at upper bound",
			input: map[string][]json.RawMessage{
				"a.go": {entry(t, "2024-01-01", "2024-01-01T18:00:00")},
			},
			timeFrom:    parseTime("08:00:00"),
			timeTo:      parseTime("18:00:00"),
			wantEntries: map[string]int{"a.go": 1},
		},
		{
			name: "nil timeFrom defaults to 00:00",
			input: map[string][]json.RawMessage{
				"a.go": {entry(t, "2024-01-01", "2024-01-01T00:01:00")},
			},
			timeFrom:    nil,
			timeTo:      parseTime("23:59:59"),
			wantEntries: map[string]int{"a.go": 1},
		},
		{
			name: "nil timeTo defaults to 23:59",
			input: map[string][]json.RawMessage{
				"a.go": {entry(t, "2024-01-01", "2024-01-01T23:58:00")},
			},
			timeFrom:    parseTime("00:00:00"),
			timeTo:      nil,
			wantEntries: map[string]int{"a.go": 1},
		},
		{
			name: "entry with no time field is kept unconditionally",
			input: map[string][]json.RawMessage{
				"a.go": {json.RawMessage(`{"date":"2024-01-01","message":"no time field"}`)},
			},
			timeFrom:    parseTime("09:00:00"),
			timeTo:      parseTime("17:00:00"),
			wantEntries: map[string]int{"a.go": 1},
		},
		{
			name: "entry with malformed time field is kept unconditionally",
			input: map[string][]json.RawMessage{
				"a.go": {
					json.RawMessage(
						`{"date":"2024-01-01","time":"2024-01-01Tnot-a-time","message":"m"}`,
					),
				},
			},
			timeFrom:    parseTime("09:00:00"),
			timeTo:      parseTime("17:00:00"),
			wantEntries: map[string]int{"a.go": 1},
		},
		{
			name: "mixed entries — only in-range kept",
			input: map[string][]json.RawMessage{
				"a.go": {
					entry(t, "2024-01-01", "2024-01-01T07:00:00"),
					entry(t, "2024-01-01", "2024-01-01T10:00:00"),
					entry(t, "2024-01-01", "2024-01-01T19:00:00"),
				},
			},
			timeFrom:    parseTime("08:00:00"),
			timeTo:      parseTime("18:00:00"),
			wantEntries: map[string]int{"a.go": 1},
		},
		{
			name: "file dropped when all entries outside range",
			input: map[string][]json.RawMessage{
				"a.go": {entry(t, "2024-01-01", "2024-01-01T06:00:00")},
			},
			timeFrom:   parseTime("09:00:00"),
			timeTo:     parseTime("17:00:00"),
			wantAbsent: []string{"a.go"},
		},
		{
			name:        "empty input returns empty result",
			input:       map[string][]json.RawMessage{},
			timeFrom:    parseTime("09:00:00"),
			timeTo:      parseTime("17:00:00"),
			wantEntries: map[string]int{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := decodeResult(t, filterTimeRange(tt.input, tt.timeFrom, tt.timeTo))

			for file, wantCount := range tt.wantEntries {
				if assert.Contains(t, result, file) {
					assert.Len(t, result[file], wantCount)
				}
			}

			for _, f := range tt.wantAbsent {
				assert.NotContains(t, result, f)
			}
		})
	}
}

func TestParseShowConfig(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		args    []string
		wantErr error
		check   func(t *testing.T, cfg *showConfig)
	}{
		{
			name: "valid with no filters",
			check: func(t *testing.T, cfg *showConfig) {
				t.Helper()

				assert.Empty(t, cfg.excludeDates)
				assert.Empty(t, cfg.includeDates)
				assert.Nil(t, cfg.includeTimeFrom)
				assert.Nil(t, cfg.includeTimeTo)
			},
		},
		{
			name:    "exclude-date and include-date are mutually exclusive",
			args:    []string{"--exclude-date", "2024-01-01", "--include-date", "2024-01-02"},
			wantErr: errArgs,
		},
		{
			name:    "include-time-from without include-time-to",
			args:    []string{"--include-time-from", "09:00"},
			wantErr: errArgs,
		},
		{
			name: "both time bounds provided",
			args: []string{"--include-time-from", "09:00", "--include-time-to", "17:00"},
			check: func(t *testing.T, cfg *showConfig) {
				t.Helper()

				assert.NotNil(t, cfg.includeTimeFrom)
				assert.NotNil(t, cfg.includeTimeTo)
			},
		},
		{
			name: "input-path and http-port are passed through",
			args: []string{"--input-path", "data.json", "--http-port", "9090"},
			check: func(t *testing.T, cfg *showConfig) {
				t.Helper()

				assert.Equal(t, "data.json", cfg.inputPath)
				assert.Equal(t, 9090, cfg.httpPort)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var (
				cfg *showConfig
				err error
			)

			app := &cli.App{
				HideHelp:    true,
				HideVersion: true,
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "input-path"},
					&cli.BoolFlag{Name: "http"},
					&cli.IntFlag{Name: "http-port"},
					&cli.StringSliceFlag{Name: "exclude-date"},
					&cli.StringSliceFlag{Name: "include-date"},
					&cli.TimestampFlag{Name: "include-time-from", Layout: "15:04"},
					&cli.TimestampFlag{Name: "include-time-to", Layout: "15:04"},
				},
				Action: func(ctx *cli.Context) error {
					cfg, err = parseShowConfig(ctx)

					return nil
				},
			}

			runArgs := append([]string{"app"}, tt.args...)
			require.NoError(t, app.Run(runArgs))

			require.ErrorIs(t, err, tt.wantErr)

			if tt.check != nil {
				require.NoError(t, err)
				tt.check(t, cfg)
			}
		})
	}
}

func TestLoadInput(t *testing.T) {
	t.Parallel()

	validJSON := makeJSON(t, map[string][]map[string]any{
		"a.go": {{
			"hash":         "abc123",
			"author":       "dev",
			"date":         "2024-01-01",
			"time":         "2024-01-01T09:00:00Z",
			"message":      "m",
			"linesAdded":   1,
			"linesRemoved": 0,
			"diff": []map[string]any{{
				"fromLine": 0,
				"toLine":   1,
				"context":  "ctx",
				"changes": []map[string]string{{
					"type":    "added",
					"content": "+foo",
				}},
			}},
		}},
	})

	tests := []struct {
		name    string
		setup   func(t *testing.T) string
		want    []byte
		wantErr error
	}{
		{
			name: "reads valid JSON from file",
			setup: func(t *testing.T) string {
				t.Helper()

				p := filepath.Join(t.TempDir(), "input.json")
				require.NoError(t, os.WriteFile(p, validJSON, 0o600))

				return p
			},
			want: validJSON,
		},
		{
			name: "non-existent file returns read error",
			setup: func(t *testing.T) string {
				t.Helper()

				return "/nonexistent/path/file.json"
			},
			wantErr: errRead,
		},
		{
			name: "invalid JSON returns validate error",
			setup: func(t *testing.T) string {
				t.Helper()

				p := filepath.Join(t.TempDir(), "bad.json")
				require.NoError(t, os.WriteFile(p, []byte("not json"), 0o600))

				return p
			},
			wantErr: errValidate,
		},
		{
			name: "JSON failing schema validation returns validate error",
			setup: func(t *testing.T) string {
				t.Helper()

				p := filepath.Join(t.TempDir(), "wrong.json")
				require.NoError(t, os.WriteFile(p, []byte(`{"a.go": "not-an-array"}`), 0o600))

				return p
			},
			wantErr: errValidate,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			data, err := loadInput(tt.setup(t))

			require.ErrorIs(t, err, tt.wantErr)
			assert.Equal(t, tt.want, data)
		})
	}
}
