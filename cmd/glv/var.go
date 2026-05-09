package glv

import (
	_ "embed"
	"html/template"
	"regexp"
)

var (
	//go:embed schema.json
	schema []byte

	//go:embed index.html.tpl
	page string

	hunkHeaderRegex = regexp.MustCompile(`@@ -(\d+)(?:,\d+)? \+(\d+)(?:,\d+)? @@(.*)`)
	hunkStartRegex  = regexp.MustCompile(`(@@.+)`)

	tableTemplate = template.Must(
		template.New("page").Delims("[[", "]]").Parse(page),
	)
)
