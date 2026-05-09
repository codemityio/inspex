package fs

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/codemityio/go/pkg/validator"
	"github.com/codemityio/inspex/internal/browser"
	inthttp "github.com/codemityio/inspex/internal/http"
	"github.com/codemityio/inspex/internal/io"
	"github.com/urfave/cli/v2"
)

func table(ctx *cli.Context) error {
	inputPath := ctx.String("input-path")
	serveHTTP := ctx.Bool("http")
	httpPort := ctx.Int("http-port")
	displayFields := ctx.StringSlice("display-field")
	heatMapFields := ctx.StringSlice("heat-map-field")
	excludeDateFields := ctx.StringSlice("exclude-date-field")
	excludeDateValues := ctx.StringSlice("exclude-date-value")

	var (
		data []byte
		err  error
	)

	if inputPath != "" {
		//nolint:gosec
		data, err = os.ReadFile(inputPath)
	} else {
		data, err = io.ReadStdin()
	}

	if err != nil {
		return fmt.Errorf("%w:%w", errRead, err)
	}

	errs, err := validator.New().ValidateJSON(schema, data)
	if err != nil {
		return fmt.Errorf("%w: %w", errValidate, err)
	}

	if len(errs) > 0 {
		return fmt.Errorf("%w: %v", errValidate, errs)
	}

	content, err := buildTableData(
		data,
		heatMapFields,
		displayFields,
		excludeDateFields,
		excludeDateValues,
	)
	if err != nil {
		return err
	}

	if serveHTTP {
		if e := inthttp.Serve(ctx.Context, func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")

			if e := tableTemplate.Execute(w, content); e != nil {
				http.Error(w, "error: "+e.Error(), http.StatusInternalServerError)
			}
		}, httpPort, browser.Open); e != nil {
			return fmt.Errorf("%w: %w", errServe, e)
		}
	}

	if e := tableTemplate.Execute(os.Stdout, content); e != nil {
		return fmt.Errorf("%w: %w", errWrite, e)
	}

	return nil
}

func buildTableData(
	data []byte,
	heatMapFields []string,
	displayFields []string,
	excludeDateFields []string,
	excludeDateValues []string,
) (tableData, error) {
	var raw []map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return tableData{}, fmt.Errorf("%w :JSON input: %w", errParse, err)
	}

	if len(raw) == 0 {
		return tableData{Columns: nil, Files: nil}, nil
	}

	if e := validateHeatMapFields(heatMapFields); e != nil {
		return tableData{Columns: nil, Files: nil}, e
	}

	if e := validateExcludeDateFields(excludeDateFields); e != nil {
		return tableData{Columns: nil, Files: nil}, e
	}

	if e := validateExcludeDateValues(excludeDateValues); e != nil {
		return tableData{Columns: nil, Files: nil}, e
	}

	raw = applyDateExclusions(raw, excludeDateFields, excludeDateValues)

	if len(raw) == 0 {
		return tableData{Columns: nil, Files: nil}, nil
	}

	presentFields := resolvePresentFields(raw[0], displayFields)
	cols := buildColumns(presentFields)
	fieldNorms := computeFieldNorms(raw, heatMapFields)
	rows := buildRows(raw, presentFields, heatMapFields, fieldNorms)

	return tableData{Columns: cols, Files: rows}, nil
}

// validateHeatMapFields returns an error if any requested field is not heatable.
func validateHeatMapFields(fields []string) error {
	for _, f := range fields {
		if _, ok := heatableFields[f]; !ok {
			return fmt.Errorf("%w: field %q is not suitable", errHeatMapField, f)
		}
	}

	return nil
}

// validateExcludeDateFields returns an error if any requested field is not a date field.
func validateExcludeDateFields(fields []string) error {
	for _, f := range fields {
		if _, ok := excludableDateFields[f]; !ok {
			return fmt.Errorf("%w: field %q is not a date field", errExcludeDateField, f)
		}
	}

	return nil
}

// validateExcludeDateValues returns an error if any value is not in YYYY, YYYY-MM, or YYYY-MM-DD format.
func validateExcludeDateValues(values []string) error {
	for _, v := range values {
		if !isValidDatePrefix(v) {
			return fmt.Errorf(
				"%w: value %q must be in YYYY, YYYY-MM, or YYYY-MM-DD format",
				errExcludeDateValue,
				v,
			)
		}
	}

	return nil
}

func isValidDatePrefix(val string) bool {
	switch len(val) {
	case datePosYYYY:
		_, err := time.Parse("2006", val)

		return err == nil
	case datePosYYYYMM:
		_, err := time.Parse("2006-01", val)

		return err == nil
	case datePosYYYYMMDD:
		_, err := time.Parse("2006-01-02", val)

		return err == nil
	default:
		return false
	}
}

// applyDateExclusions removes rows where any of the excludeDateFields matches
// any of the excludeDateValues as a prefix (YYYY, YYYY-MM, or YYYY-MM-DD).
func applyDateExclusions(
	raw []map[string]json.RawMessage,
	excludeDateFields []string,
	excludeDateValues []string,
) []map[string]json.RawMessage {
	if len(excludeDateFields) == 0 || len(excludeDateValues) == 0 {
		return raw
	}

	filtered := raw[:0]

	for _, record := range raw {
		if !recordMatchesDateExclusion(record, excludeDateFields, excludeDateValues) {
			filtered = append(filtered, record)
		}
	}

	return filtered
}

// recordMatchesDateExclusion returns true if any of the record's date fields
// has a value that starts with any of the exclusion prefixes.
func recordMatchesDateExclusion(
	record map[string]json.RawMessage,
	excludeDateFields []string,
	excludeDateValues []string,
) bool {
	for _, field := range excludeDateFields {
		raw, ok := record[field]
		if !ok {
			continue
		}

		var s string
		if err := json.Unmarshal(raw, &s); err != nil {
			continue
		}

		for _, prefix := range excludeDateValues {
			if len(s) >= len(prefix) && s[:len(prefix)] == prefix {
				return true
			}
		}
	}

	return false
}

// resolvePresentFields returns the ordered subset of allFields present in the first record.
func resolvePresentFields(first map[string]json.RawMessage, displayFields []string) []string {
	source := allFields
	if len(displayFields) > 0 {
		source = displayFields
	}

	present := make([]string, 0, len(source))
	for _, f := range source {
		if _, ok := first[f]; ok {
			present = append(present, f)
		}
	}

	return present
}

// buildColumns produces a column descriptor for each present field.
func buildColumns(presentFields []string) []column {
	cols := make([]column, len(presentFields))
	for i, f := range presentFields {
		cols[i] = column{Label: f, Type: fieldType(f)}
	}

	return cols
}

// computeFieldNorms pre-computes per-field normalised [0,1] values across all rows.
func computeFieldNorms(
	raw []map[string]json.RawMessage,
	heatMapFields []string,
) map[string][]float64 {
	fieldNorms := make(map[string][]float64, len(heatMapFields))

	for _, field := range heatMapFields {
		vals := make([]float64, len(raw))
		for i, record := range raw {
			vals[i] = extractHeatValue(record[field])
		}

		mn, mx := minMax(vals)
		norms := make([]float64, len(raw))

		for i, v := range vals {
			if mx > mn {
				norms[i] = (v - mn) / (mx - mn)
			} else {
				norms[i] = 0.5
			}
		}

		fieldNorms[field] = norms
	}

	return fieldNorms
}

// buildRows renders all cells for every record, applying heat colours where requested.
func buildRows(
	raw []map[string]json.RawMessage,
	presentFields, heatMapFields []string,
	fieldNorms map[string][]float64,
) []row {
	heatSet := make(map[string]struct{}, len(heatMapFields))
	for _, f := range heatMapFields {
		heatSet[f] = struct{}{}
	}

	rows := make([]row, 0, len(raw))

	for i, record := range raw {
		cells := make([]cell, 0, len(presentFields))

		for _, f := range presentFields {
			c := renderCell(f, record[f])
			if _, ok := heatSet[f]; ok {
				c.HeatColor = heatColor(fieldNorms[f][i])
			}

			cells = append(cells, c)
		}

		rows = append(rows, row{Cells: cells, IsDir: false})
	}

	return rows
}

// extractHeatValue converts a raw JSON field value into a float64 for heat
// normalisation. Dates become Unix timestamps; numbers stay as-is.
func extractHeatValue(raw json.RawMessage) float64 {
	if raw == nil {
		return 0
	}

	var v any

	_ = json.Unmarshal(raw, &v)

	switch val := v.(type) {
	case float64:
		return val
	case string:
		t, err := time.Parse(time.RFC3339, val)
		if err == nil {
			return float64(t.Unix())
		}
	}

	return 0
}

// minMax returns the min and max of a float64 slice.
func minMax(vals []float64) (float64, float64) {
	if len(vals) == 0 {
		return 0, 0
	}

	mn, mx := vals[0], vals[0]
	for _, v := range vals[1:] {
		if v < mn {
			mn = v
		}

		if v > mx {
			mx = v
		}
	}

	return mn, mx
}

// heatColor maps a normalised value [0,1] to a subtle but fully opaque
// pastel background colour.
func heatColor(temp float64) template.CSS {
	type stop struct {
		pos     float64
		r, g, b int
	}

	stops := []stop{
		{0.00, 219, 234, 254}, // pastel blue
		{0.25, 220, 252, 231}, // pastel green
		{0.50, 254, 249, 195}, // pastel yellow
		{0.75, 255, 237, 213}, // pastel orange
		{1.00, 254, 226, 226}, // pastel red
	}

	for i := 1; i < len(stops); i++ {
		if temp <= stops[i].pos {
			lo, hi := stops[i-1], stops[i]
			f := (temp - lo.pos) / (hi.pos - lo.pos)
			r := int(float64(lo.r) + f*float64(hi.r-lo.r))
			g := int(float64(lo.g) + f*float64(hi.g-lo.g))
			b := int(float64(lo.b) + f*float64(hi.b-lo.b))

			//nolint:gosec // RGB values are computed from hardcoded stops, no user input involved
			return template.CSS(fmt.Sprintf("rgb(%d,%d,%d)", r, g, b))
		}
	}

	last := stops[len(stops)-1]

	//nolint:gosec // RGB values are computed from hardcoded stops, no user input involved
	return template.CSS(fmt.Sprintf("rgb(%d,%d,%d)", last.r, last.g, last.b))
}

func fieldType(field string) string {
	switch field {
	case "size", "lines":
		return "number"
	default:
		return "string"
	}
}

func renderCell(field string, raw json.RawMessage) cell {
	var vl any

	_ = json.Unmarshal(raw, &vl)

	display := fmt.Sprintf("%v", vl)
	val := display

	if s, ok := vl.(string); ok {
		display = s
		val = s
	}

	if n, ok := vl.(float64); ok {
		val = fmt.Sprintf("%v", n)
		display = strconv.FormatInt(int64(n), 10)
	}

	var class string

	switch field {
	case "size", "lines":
		class = "col-num"
	case "createdAt", "modifiedAt", "accessedAt", "changedAt":
		class = "col-date"

		if len(display) == rfc3339Len {
			display = display[:19]
		}
	case "mode":
		class = "col-mode"
	case "isDir", "isLink":
		class = "col-bool"
	}

	return cell{
		Class: class, Display: display, Val: val, HeatColor: "",
	}
}
