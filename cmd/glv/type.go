package glv

import "time"

// Change represents a single line change in a diff hunk.
type Change struct {
	Type    string `json:"type"`
	Content string `json:"content"`
}

// Hunk represents a single @@ block in a diff.
type Hunk struct {
	FromLine int       `json:"fromLine"`
	ToLine   int       `json:"toLine"`
	Context  string    `json:"context"`
	Changes  []*Change `json:"changes"`
}

// FileChange represents a single commit's changes to a file.
type FileChange struct {
	Hash         string  `json:"hash"`
	Date         string  `json:"date"`
	Time         string  `json:"time"`
	Author       string  `json:"author"`
	Message      string  `json:"message"`
	LinesAdded   int     `json:"linesAdded"`
	LinesRemoved int     `json:"linesRemoved"`
	Diff         []*Hunk `json:"diff"`
}

// fileChangeDateTime is used only to peek at the date and time fields during filtering.
// All other fields are preserved via json.RawMessage.
type fileChangeDateTime struct {
	Date string `json:"date"`
	Time string `json:"time"` // "2006-01-02T15:04:05"
}

type showConfig struct {
	inputPath       string
	http            bool
	httpPort        int
	excludeDates    []string
	includeDates    []string
	includeTimeFrom *time.Time
	includeTimeTo   *time.Time
}
