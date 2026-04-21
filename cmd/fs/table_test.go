package fs

import (
	"encoding/json"
	"html/template"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadInputFile(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		setup   func(t *testing.T) string
		wantErr bool
	}{
		{
			name: "reads existing file",
			setup: func(t *testing.T) string {
				t.Helper()

				path := filepath.Join(t.TempDir(), "input.json")
				require.NoError(t, os.WriteFile(path, []byte(`[{"file":"a.go"}]`), 0o600))

				return path
			},
		},
		{
			name: "non-existent file returns error",
			setup: func(t *testing.T) string {
				t.Helper()

				return filepath.Join(t.TempDir(), "nonexistent.json")
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			path := tt.setup(t)
			got, err := os.ReadFile(path)

			if tt.wantErr {
				require.Error(t, err)
				assert.Nil(t, got)

				return
			}

			require.NoError(t, err)
			assert.NotEmpty(t, got)
		})
	}
}

func TestValidateHeatMapFields(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		fields  []string
		wantErr error
	}{
		{
			name:   "empty fields",
			fields: []string{},
		},
		{
			name:   "valid single field",
			fields: []string{"size"},
		},
		{
			name: "all valid heatable fields",
			fields: []string{
				"size",
				"lines",
				"createdAt",
				"modifiedAt",
				"accessedAt",
				"changedAt",
			},
		},
		{
			name:    "file field is not heatable",
			fields:  []string{"file"},
			wantErr: errHeatMapField,
		},
		{
			name:    "mode field is not heatable",
			fields:  []string{"size", "mode"},
			wantErr: errHeatMapField,
		},
		{
			name:    "unknown field returns error",
			fields:  []string{"nonexistent"},
			wantErr: errHeatMapField,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			require.ErrorIs(t, validateHeatMapFields(tt.fields), tt.wantErr)
		})
	}
}

func TestValidateExcludeDateFields(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		fields  []string
		wantErr error
	}{
		{
			name:   "empty fields",
			fields: []string{},
		},
		{
			name:   "valid single date field",
			fields: []string{"createdAt"},
		},
		{
			name:   "all valid date fields",
			fields: []string{"createdAt", "modifiedAt", "accessedAt", "changedAt"},
		},
		{
			name:    "file field is not a date field",
			fields:  []string{"file"},
			wantErr: errExcludeDateField,
		},
		{
			name:    "size field is not a date field",
			fields:  []string{"createdAt", "size"},
			wantErr: errExcludeDateField,
		},
		{
			name:    "unknown field returns error",
			fields:  []string{"nonexistent"},
			wantErr: errExcludeDateField,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			require.ErrorIs(t, validateExcludeDateFields(tt.fields), tt.wantErr)
		})
	}
}

func TestValidateExcludeDateValues(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		values  []string
		wantErr error
	}{
		{
			name:   "empty values",
			values: []string{},
		},
		{
			name:   "valid YYYY",
			values: []string{"2024"},
		},
		{
			name:   "valid YYYY-MM",
			values: []string{"2024-03"},
		},
		{
			name:   "valid YYYY-MM-DD",
			values: []string{"2024-03-15"},
		},
		{
			name:   "multiple valid values",
			values: []string{"2024", "2023-06", "2022-01-01"},
		},
		{
			name:    "invalid format returns error",
			values:  []string{"24"},
			wantErr: errExcludeDateValue,
		},
		{
			name:    "invalid month returns error",
			values:  []string{"2024-13"},
			wantErr: errExcludeDateValue,
		},
		{
			name:    "invalid day returns error",
			values:  []string{"2024-01-32"},
			wantErr: errExcludeDateValue,
		},
		{
			name:    "full RFC3339 is rejected",
			values:  []string{"2024-01-15T00:00:00Z"},
			wantErr: errExcludeDateValue,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			require.ErrorIs(t, validateExcludeDateValues(tt.values), tt.wantErr)
		})
	}
}

func TestApplyDateExclusions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name              string
		raw               []map[string]json.RawMessage
		excludeDateFields []string
		excludeDateValues []string
		wantLen           int
	}{
		{
			name: "no exclusion fields returns all rows",
			raw: []map[string]json.RawMessage{
				{"createdAt": json.RawMessage(`"2024-01-15T10:00:00Z"`)},
				{"createdAt": json.RawMessage(`"2023-06-01T10:00:00Z"`)},
			},
			excludeDateFields: []string{},
			excludeDateValues: []string{"2024"},
			wantLen:           2,
		},
		{
			name: "no exclusion values returns all rows",
			raw: []map[string]json.RawMessage{
				{"createdAt": json.RawMessage(`"2024-01-15T10:00:00Z"`)},
				{"createdAt": json.RawMessage(`"2023-06-01T10:00:00Z"`)},
			},
			excludeDateFields: []string{"createdAt"},
			excludeDateValues: []string{},
			wantLen:           2,
		},
		{
			name: "YYYY prefix excludes matching rows",
			raw: []map[string]json.RawMessage{
				{"createdAt": json.RawMessage(`"2024-01-15T10:00:00Z"`)},
				{"createdAt": json.RawMessage(`"2023-06-01T10:00:00Z"`)},
				{"createdAt": json.RawMessage(`"2024-11-30T10:00:00Z"`)},
			},
			excludeDateFields: []string{"createdAt"},
			excludeDateValues: []string{"2024"},
			wantLen:           1,
		},
		{
			name: "YYYY-MM prefix excludes matching rows",
			raw: []map[string]json.RawMessage{
				{"createdAt": json.RawMessage(`"2024-03-15T10:00:00Z"`)},
				{"createdAt": json.RawMessage(`"2024-04-01T10:00:00Z"`)},
				{"createdAt": json.RawMessage(`"2023-03-15T10:00:00Z"`)},
			},
			excludeDateFields: []string{"createdAt"},
			excludeDateValues: []string{"2024-03"},
			wantLen:           2,
		},
		{
			name: "YYYY-MM-DD prefix excludes matching rows",
			raw: []map[string]json.RawMessage{
				{"createdAt": json.RawMessage(`"2024-03-15T10:00:00Z"`)},
				{"createdAt": json.RawMessage(`"2024-03-15T22:00:00Z"`)},
				{"createdAt": json.RawMessage(`"2024-03-16T10:00:00Z"`)},
			},
			excludeDateFields: []string{"createdAt"},
			excludeDateValues: []string{"2024-03-15"},
			wantLen:           1,
		},
		{
			name: "multiple exclusion values",
			raw: []map[string]json.RawMessage{
				{"createdAt": json.RawMessage(`"2024-01-15T10:00:00Z"`)},
				{"createdAt": json.RawMessage(`"2023-06-01T10:00:00Z"`)},
				{"createdAt": json.RawMessage(`"2022-01-01T10:00:00Z"`)},
			},
			excludeDateFields: []string{"createdAt"},
			excludeDateValues: []string{"2024", "2023"},
			wantLen:           1,
		},
		{
			name: "multiple exclusion fields any match excludes row",
			raw: []map[string]json.RawMessage{
				{
					"createdAt":  json.RawMessage(`"2023-01-15T10:00:00Z"`),
					"modifiedAt": json.RawMessage(`"2024-06-01T10:00:00Z"`),
				},
				{
					"createdAt":  json.RawMessage(`"2023-01-15T10:00:00Z"`),
					"modifiedAt": json.RawMessage(`"2023-06-01T10:00:00Z"`),
				},
			},
			excludeDateFields: []string{"createdAt", "modifiedAt"},
			excludeDateValues: []string{"2024"},
			wantLen:           1,
		},
		{
			name: "all rows excluded returns empty slice",
			raw: []map[string]json.RawMessage{
				{"createdAt": json.RawMessage(`"2024-01-15T10:00:00Z"`)},
				{"createdAt": json.RawMessage(`"2024-06-01T10:00:00Z"`)},
			},
			excludeDateFields: []string{"createdAt"},
			excludeDateValues: []string{"2024"},
			wantLen:           0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Len(
				t,
				applyDateExclusions(tt.raw, tt.excludeDateFields, tt.excludeDateValues),
				tt.wantLen,
			)
		})
	}
}

func TestResolvePresentFields(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		first         map[string]json.RawMessage
		displayFields []string
		expect        []string
	}{
		{
			name:   "empty record returns empty slice",
			first:  map[string]json.RawMessage{},
			expect: []string{},
		},
		{
			name: "subset of fields present returns ordered subset",
			first: map[string]json.RawMessage{
				"file": json.RawMessage(`"main.go"`),
				"size": json.RawMessage(`1024`),
			},
			expect: []string{"file", "size"},
		},
		{
			name: "unknown fields in record are ignored",
			first: map[string]json.RawMessage{
				"file":    json.RawMessage(`"main.go"`),
				"unknown": json.RawMessage(`"ignored"`),
			},
			expect: []string{"file"},
		},
		{
			name: "display fields override allFields and control order",
			first: map[string]json.RawMessage{
				"file": json.RawMessage(`"main.go"`),
				"size": json.RawMessage(`1024`),
				"mode": json.RawMessage(`"-rw-r--r--"`),
			},
			displayFields: []string{"size", "file"},
			expect:        []string{"size", "file"},
		},
		{
			name: "display fields not in record are silently ignored",
			first: map[string]json.RawMessage{
				"file": json.RawMessage(`"main.go"`),
			},
			displayFields: []string{"file", "size"},
			expect:        []string{"file"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := resolvePresentFields(tt.first, tt.displayFields)

			require.Len(t, got, len(tt.expect))

			for i, f := range tt.expect {
				assert.Equal(t, f, got[i], "position %d", i)
			}
		})
	}
}

func TestBuildColumns(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		presentFields []string
		wantLabels    []string
		wantTypes     []string
	}{
		{
			name:          "empty fields produces empty columns",
			presentFields: []string{},
			wantLabels:    []string{},
			wantTypes:     []string{},
		},
		{
			name:          "numeric fields produce number type",
			presentFields: []string{"size", "lines"},
			wantLabels:    []string{"size", "lines"},
			wantTypes:     []string{"number", "number"},
		},
		{
			name:          "string fields produce string type",
			presentFields: []string{"file", "mode"},
			wantLabels:    []string{"file", "mode"},
			wantTypes:     []string{"string", "string"},
		},
		{
			name:          "mixed fields preserve order",
			presentFields: []string{"file", "size", "mode", "lines"},
			wantLabels:    []string{"file", "size", "mode", "lines"},
			wantTypes:     []string{"string", "number", "string", "number"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := buildColumns(tt.presentFields)

			require.Len(t, got, len(tt.wantLabels))

			for i, col := range got {
				assert.Equal(t, tt.wantLabels[i], col.Label, "column %d label", i)
				assert.Equal(t, tt.wantTypes[i], col.Type, "column %d type", i)
			}
		})
	}
}

func TestExtractHeatValue(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		raw  json.RawMessage
		want float64
	}{
		{
			name: "nil returns zero",
			raw:  nil,
			want: 0,
		},
		{
			name: "numeric value",
			raw:  json.RawMessage(`1024`),
			want: 1024,
		},
		{
			name: "zero numeric value",
			raw:  json.RawMessage(`0`),
			want: 0,
		},
		{
			name: "valid RFC3339 date returns unix timestamp",
			raw:  json.RawMessage(`"2026-01-01T00:00:00Z"`),
			want: float64(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC).Unix()),
		},
		{
			name: "invalid date string returns zero",
			raw:  json.RawMessage(`"not-a-date"`),
			want: 0,
		},
		{
			name: "boolean returns zero",
			raw:  json.RawMessage(`true`),
			want: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.InDelta(t, tt.want, extractHeatValue(tt.raw), 1e-9)
		})
	}
}

func TestMinMax(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		vals    []float64
		wantMin float64
		wantMax float64
	}{
		{
			name:    "empty slice returns zeros",
			vals:    []float64{},
			wantMin: 0,
			wantMax: 0,
		},
		{
			name:    "single value",
			vals:    []float64{42},
			wantMin: 42,
			wantMax: 42,
		},
		{
			name:    "all same values",
			vals:    []float64{5, 5, 5},
			wantMin: 5,
			wantMax: 5,
		},
		{
			name:    "ascending values",
			vals:    []float64{1, 2, 3, 4, 5},
			wantMin: 1,
			wantMax: 5,
		},
		{
			name:    "descending values",
			vals:    []float64{5, 4, 3, 2, 1},
			wantMin: 1,
			wantMax: 5,
		},
		{
			name:    "negative values",
			vals:    []float64{-10, -5, 0, 5, 10},
			wantMin: -10,
			wantMax: 10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			gotMin, gotMax := minMax(tt.vals)

			assert.InDelta(t, tt.wantMin, gotMin, 1e-9, "min")
			assert.InDelta(t, tt.wantMax, gotMax, 1e-9, "max")
		})
	}
}

func TestHeatColor(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		temp float64
		want template.CSS
	}{
		{
			name: "zero maps to pastel blue",
			temp: 0.0,
			want: template.CSS("rgb(219,234,254)"),
		},
		{
			name: "one maps to pastel red",
			temp: 1.0,
			want: template.CSS("rgb(254,226,226)"),
		},
		{
			name: "midpoint maps to pastel yellow",
			temp: 0.5,
			want: template.CSS("rgb(254,249,195)"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, heatColor(tt.temp))
		})
	}
}

func TestFieldType(t *testing.T) {
	t.Parallel()

	tests := []struct {
		field string
		want  string
	}{
		{field: "size", want: "number"},
		{field: "lines", want: "number"},
		{field: "file", want: "string"},
		{field: "mode", want: "string"},
		{field: "createdAt", want: "string"},
		{field: "modifiedAt", want: "string"},
		{field: "accessedAt", want: "string"},
		{field: "changedAt", want: "string"},
		{field: "isLink", want: "string"},
		{field: "isDir", want: "string"},
		{field: "unknown", want: "string"},
	}

	for _, tt := range tests {
		t.Run(tt.field, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, fieldType(tt.field))
		})
	}
}

func TestRenderCell(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		field       string
		raw         json.RawMessage
		wantClass   string
		wantDisplay string
		wantVal     string
	}{
		{
			name:        "file field",
			field:       "file",
			raw:         json.RawMessage(`"main.go"`),
			wantClass:   "",
			wantDisplay: "main.go",
			wantVal:     "main.go",
		},
		{
			name:        "size field formats as integer",
			field:       "size",
			raw:         json.RawMessage(`1024`),
			wantClass:   "col-num",
			wantDisplay: "1024",
			wantVal:     "1024",
		},
		{
			name:        "lines field",
			field:       "lines",
			raw:         json.RawMessage(`42`),
			wantClass:   "col-num",
			wantDisplay: "42",
			wantVal:     "42",
		},
		{
			name:        "mode field",
			field:       "mode",
			raw:         json.RawMessage(`"-rw-r--r--"`),
			wantClass:   "col-mode",
			wantDisplay: "-rw-r--r--",
			wantVal:     "-rw-r--r--",
		},
		{
			name:        "createdAt trims timezone suffix for display",
			field:       "createdAt",
			raw:         json.RawMessage(`"2026-04-18T08:30:00Z"`),
			wantClass:   "col-date",
			wantDisplay: "2026-04-18T08:30:00",
			wantVal:     "2026-04-18T08:30:00Z",
		},
		{
			name:        "modifiedAt trims timezone suffix for display",
			field:       "modifiedAt",
			raw:         json.RawMessage(`"2026-04-18T08:30:00Z"`),
			wantClass:   "col-date",
			wantDisplay: "2026-04-18T08:30:00",
			wantVal:     "2026-04-18T08:30:00Z",
		},
		{
			name:        "accessedAt trims timezone suffix for display",
			field:       "accessedAt",
			raw:         json.RawMessage(`"2026-04-18T08:30:00Z"`),
			wantClass:   "col-date",
			wantDisplay: "2026-04-18T08:30:00",
			wantVal:     "2026-04-18T08:30:00Z",
		},
		{
			name:        "changedAt trims timezone suffix for display",
			field:       "changedAt",
			raw:         json.RawMessage(`"2026-04-18T08:30:00Z"`),
			wantClass:   "col-date",
			wantDisplay: "2026-04-18T08:30:00",
			wantVal:     "2026-04-18T08:30:00Z",
		},
		{
			name:        "isDir field",
			field:       "isDir",
			raw:         json.RawMessage(`true`),
			wantClass:   "col-bool",
			wantDisplay: "true",
			wantVal:     "true",
		},
		{
			name:        "isLink field",
			field:       "isLink",
			raw:         json.RawMessage(`false`),
			wantClass:   "col-bool",
			wantDisplay: "false",
			wantVal:     "false",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := renderCell(tt.field, tt.raw)

			assert.Equal(t, tt.wantClass, got.Class, "Class")
			assert.Equal(t, tt.wantDisplay, got.Display, "Display")
			assert.Equal(t, tt.wantVal, got.Val, "Val")
		})
	}
}

func TestComputeFieldNorms(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		raw           []map[string]json.RawMessage
		heatMapFields []string
		wantField     string
		wantNorms     []float64
	}{
		{
			name: "single field normalised correctly",
			raw: []map[string]json.RawMessage{
				{"size": json.RawMessage(`100`)},
				{"size": json.RawMessage(`200`)},
				{"size": json.RawMessage(`300`)},
			},
			heatMapFields: []string{"size"},
			wantField:     "size",
			wantNorms:     []float64{0, 0.5, 1},
		},
		{
			name: "all same values normalise to 0.5",
			raw: []map[string]json.RawMessage{
				{"size": json.RawMessage(`100`)},
				{"size": json.RawMessage(`100`)},
			},
			heatMapFields: []string{"size"},
			wantField:     "size",
			wantNorms:     []float64{0.5, 0.5},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := computeFieldNorms(tt.raw, tt.heatMapFields)

			norms, ok := got[tt.wantField]
			require.True(t, ok, "field %q missing from result", tt.wantField)
			require.Len(t, norms, len(tt.wantNorms))

			for i, want := range tt.wantNorms {
				assert.InDelta(t, want, norms[i], 1e-9, "norm[%d]", i)
			}
		})
	}
}

func TestBuildRows(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		raw           []map[string]json.RawMessage
		presentFields []string
		heatMapFields []string
		fieldNorms    map[string][]float64
		wantLen       int
		checkRow      func(t *testing.T, rows []row)
	}{
		{
			name:          "empty input returns empty rows",
			raw:           []map[string]json.RawMessage{},
			presentFields: []string{"file"},
			heatMapFields: []string{},
			fieldNorms:    map[string][]float64{},
			wantLen:       0,
		},
		{
			name: "single row with no heat map",
			raw: []map[string]json.RawMessage{
				{"file": json.RawMessage(`"main.go"`), "size": json.RawMessage(`1024`)},
			},
			presentFields: []string{"file", "size"},
			heatMapFields: []string{},
			fieldNorms:    map[string][]float64{},
			wantLen:       1,
			checkRow: func(t *testing.T, rows []row) {
				t.Helper()
				assert.Len(t, rows[0].Cells, 2)
				assert.Empty(t, rows[0].Cells[0].HeatColor)
			},
		},
		{
			name: "heat map field sets HeatColor on cell",
			raw: []map[string]json.RawMessage{
				{"size": json.RawMessage(`100`)},
				{"size": json.RawMessage(`200`)},
			},
			presentFields: []string{"size"},
			heatMapFields: []string{"size"},
			fieldNorms:    map[string][]float64{"size": {0.0, 1.0}},
			wantLen:       2,
			checkRow: func(t *testing.T, rows []row) {
				t.Helper()
				assert.NotEmpty(t, rows[0].Cells[0].HeatColor)
				assert.NotEmpty(t, rows[1].Cells[0].HeatColor)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := buildRows(tt.raw, tt.presentFields, tt.heatMapFields, tt.fieldNorms)

			assert.Len(t, got, tt.wantLen)

			if tt.checkRow != nil {
				tt.checkRow(t, got)
			}
		})
	}
}

func TestBuildTableData(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name              string
		data              []byte
		heatMapFields     []string
		displayFields     []string
		excludeDateFields []string
		excludeDateValues []string
		wantCols          int
		wantRows          int
		wantErr           bool
	}{
		{
			name:    "invalid JSON returns error",
			data:    []byte(`not json`),
			wantErr: true,
		},
		{
			name:     "empty array returns empty tableData",
			data:     []byte(`[]`),
			wantCols: 0,
			wantRows: 0,
		},
		{
			name:     "valid data with no heat map fields",
			data:     []byte(`[{"file":"main.go","size":1024,"lines":42}]`),
			wantCols: 3,
			wantRows: 1,
		},
		{
			name:          "valid heat map field",
			data:          []byte(`[{"file":"main.go","size":1024},{"file":"app.go","size":2048}]`),
			heatMapFields: []string{"size"},
			wantCols:      2,
			wantRows:      2,
		},
		{
			name:          "invalid heat map field returns error",
			data:          []byte(`[{"file":"main.go","size":1024}]`),
			heatMapFields: []string{"file"},
			wantErr:       true,
		},
		{
			name:          "display fields limits columns shown",
			data:          []byte(`[{"file":"main.go","size":1024,"lines":42}]`),
			displayFields: []string{"file", "size"},
			wantCols:      2,
			wantRows:      1,
		},
		{
			name:              "invalid exclude date field returns error",
			data:              []byte(`[{"file":"main.go","createdAt":"2024-01-15T10:00:00Z"}]`),
			excludeDateFields: []string{"file"},
			excludeDateValues: []string{"2024"},
			wantErr:           true,
		},
		{
			name:              "invalid exclude date value returns error",
			data:              []byte(`[{"file":"main.go","createdAt":"2024-01-15T10:00:00Z"}]`),
			excludeDateFields: []string{"createdAt"},
			excludeDateValues: []string{"24"},
			wantErr:           true,
		},
		{
			name: "exclude by year filters matching rows",
			data: []byte(
				`[{"file":"a.go","createdAt":"2024-01-15T10:00:00Z"},{"file":"b.go","createdAt":"2023-06-01T10:00:00Z"}]`,
			),
			excludeDateFields: []string{"createdAt"},
			excludeDateValues: []string{"2024"},
			wantCols:          2,
			wantRows:          1,
		},
		{
			name: "exclude by month filters matching rows",
			data: []byte(
				`[{"file":"a.go","createdAt":"2024-03-15T10:00:00Z"},{"file":"b.go","createdAt":"2024-04-01T10:00:00Z"}]`,
			),
			excludeDateFields: []string{"createdAt"},
			excludeDateValues: []string{"2024-03"},
			wantCols:          2,
			wantRows:          1,
		},
		{
			name: "exclude by day filters matching rows",
			data: []byte(
				`[
  {"file":"a.go","createdAt":"2024-03-15T10:00:00Z"},
  {"file":"b.go","createdAt":"2024-03-15T22:00:00Z"},
  {"file":"c.go","createdAt":"2024-03-16T10:00:00Z"}
]`,
			),
			excludeDateFields: []string{"createdAt"},
			excludeDateValues: []string{"2024-03-15"},
			wantCols:          2,
			wantRows:          1,
		},
		{
			name: "all rows excluded returns empty tableData",
			data: []byte(
				`[{"file":"a.go","createdAt":"2024-01-15T10:00:00Z"},{"file":"b.go","createdAt":"2024-06-01T10:00:00Z"}]`,
			),
			excludeDateFields: []string{"createdAt"},
			excludeDateValues: []string{"2024"},
			wantCols:          0,
			wantRows:          0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := buildTableData(
				tt.data,
				tt.heatMapFields,
				tt.displayFields,
				tt.excludeDateFields,
				tt.excludeDateValues,
			)

			if tt.wantErr {
				assert.Error(t, err)

				return
			}

			require.NoError(t, err)
			assert.Len(t, got.Columns, tt.wantCols)
			assert.Len(t, got.Files, tt.wantRows)
		})
	}
}
