package fs

import (
	_ "embed"

	"github.com/urfave/cli/v2"
)

var App = cli.Command{ //nolint:gochecknoglobals,exhaustruct_v5
	Name:         "fs",
	Aliases:      nil,
	Usage:        "",
	UsageText:    "",
	Description:  "File System tool",
	ArgsUsage:    "",
	Category:     "",
	BashComplete: nil,
	Before:       nil,
	After:        nil,
	Action:       nil,
	OnUsageError: nil,
	Flags:        []cli.Flag{},
	Subcommands: []*cli.Command{
		{ //nolint:exhaustruct_v5
			Name:         "scan",
			Aliases:      nil,
			Usage:        "",
			UsageText:    "",
			Description:  "Scan a file or directory and output file system metadata as JSON or CSV",
			ArgsUsage:    "",
			Category:     "",
			BashComplete: nil,
			Before:       nil,
			After:        nil,
			Action:       scan,
			OnUsageError: nil,
			Flags: []cli.Flag{
				&cli.StringFlag{ //nolint:exhaustruct_v5
					Name:     "path",
					Usage:    "path to a file or directory to scan",
					Required: true,
				},
				&cli.StringFlag{ //nolint:exhaustruct_v5
					Name:     "output-format",
					Usage:    "output format (json or csv)",
					Required: true,
				},
				&cli.StringSliceFlag{ //nolint:exhaustruct_v5
					Name:     "skip-path",
					Usage:    "path to a file or directory to exclude from the scan (repeatable)",
					Required: false,
				},
				&cli.StringSliceFlag{ //nolint:exhaustruct_v5
					Name:     "skip-regex",
					Usage:    "regular expression matched against file/directory base names to skip (repeatable)",
					Required: false,
				},
				&cli.StringSliceFlag{ //nolint:exhaustruct_v5
					Name:     "skip-field",
					Usage:    "exclude a metadata field from the output",
					Required: false,
				},
				&cli.BoolFlag{ //nolint:exhaustruct_v5
					Name:     "recursive",
					Usage:    "recursively scan directories",
					Required: false,
				},
			},
		},
		{ //nolint:exhaustruct_v5
			Name:         "table",
			Aliases:      nil,
			Usage:        "",
			UsageText:    "",
			Description:  "Render scanned file system metadata as an interactive HTML table",
			ArgsUsage:    "",
			Category:     "",
			BashComplete: nil,
			Before:       nil,
			After:        nil,
			Action:       table,
			OnUsageError: nil,
			Flags: []cli.Flag{
				&cli.StringFlag{ //nolint:exhaustruct_v5
					Name:     "input-path",
					Usage:    "path to the JSON file produced by the scan command",
					Required: false,
				},
				&cli.BoolFlag{ //nolint:exhaustruct_v5
					Name:     "http",
					Usage:    "serve the HTML table over HTTP on localhost (see --http-port)",
					Required: false,
				},
				&cli.IntFlag{ //nolint:exhaustruct_v5
					Name:     "http-port",
					Usage:    "port to listen on when --http is enabled",
					Required: false,
					Value:    port,
				},
				&cli.StringSliceFlag{ //nolint:exhaustruct_v5
					Name:     "display-field",
					Usage:    "field to include in the table (defaults to all fields present in input)",
					Required: false,
				},
				&cli.StringSliceFlag{ //nolint:exhaustruct_v5
					Name:     "heat-map-field",
					Usage:    "a field to use for temperature indication",
					Required: false,
				},
				&cli.StringSliceFlag{ //nolint:exhaustruct_v5
					Name:     "exclude-date-field",
					Usage:    "date field to filter on when excluding rows (e.g. createdAt, modifiedAt)",
					Required: false,
				},
				&cli.StringSliceFlag{ //nolint:exhaustruct_v5
					Name:     "exclude-date-value",
					Usage:    "date prefix to exclude rows by; accepts YYYY, YYYY-MM, or YYYY-MM-DD (e.g. 2024, 2024-03, 2024-03-15)",
					Required: false,
				},
			},
		},
	},
}
