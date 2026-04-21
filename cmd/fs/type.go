package fs

import (
	"html/template"
)

type Info struct {
	File       string `json:"file"`
	CreatedAt  string `json:"createdAt"`
	ModifiedAt string `json:"modifiedAt"`
	AccessedAt string `json:"accessedAt"`
	ChangedAt  string `json:"changedAt"`
	Size       int64  `json:"size"`
	Lines      int    `json:"lines"`
	Mode       string `json:"mode"`
	IsLink     bool   `json:"isLink"`
	IsDir      bool   `json:"isDir"`
}

// column describes a table column for the template.
type column struct {
	Label string
	Type  string // "string" | "number"
}

// cell is one rendered table cell.
type cell struct {
	Class     string
	Display   string
	Val       string
	HeatColor template.CSS // typed as CSS so the template engine does not escape it
}

// row is one rendered table row.
type row struct {
	Cells []cell
	IsDir bool
}

// tableData is passed to the HTML template.
type tableData struct {
	Columns []column
	Files   []row
}
