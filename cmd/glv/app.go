package glv

import "github.com/urfave/cli/v2"

// App main application.
var App = cli.Command{ //nolint:gochecknoglobals,exhaustruct
	Name:         "glv",
	Aliases:      nil,
	Usage:        "",
	UsageText:    "",
	Description:  "",
	ArgsUsage:    "",
	Category:     "",
	BashComplete: nil,
	After:        nil,
	OnUsageError: nil,
	Flags:        []cli.Flag{},
	Subcommands: []*cli.Command{
		{
			Name:         "parse",
			Aliases:      nil,
			Usage:        "",
			UsageText:    "",
			Description:  "",
			ArgsUsage:    "",
			Category:     "",
			BashComplete: nil,
			Before:       nil,
			After:        nil,
			Action:       parse,
			OnUsageError: nil,
			Flags: []cli.Flag{
				&cli.StringFlag{ //nolint: exhaustruct
					Name:  "path",
					Usage: "path to the git repository (defaults to current directory)",
				},
			},
		},
		{
			Name:         "show",
			Aliases:      nil,
			Usage:        "",
			UsageText:    "",
			Description:  "",
			ArgsUsage:    "",
			Category:     "",
			BashComplete: nil,
			Before:       nil,
			After:        nil,
			Action:       show,
			OnUsageError: nil,
			Flags: []cli.Flag{
				&cli.StringFlag{ //nolint:exhaustruct
					Name:     "input-path",
					Usage:    "path to the JSON file produced by the parse command",
					Required: false,
				},
				&cli.BoolFlag{ //nolint:exhaustruct
					Name:     "http",
					Usage:    "serve the HTML table over HTTP on localhost (see --http-port)",
					Required: false,
				},
				&cli.IntFlag{ //nolint:exhaustruct
					Name:     "http-port",
					Usage:    "port to listen on when --http is enabled",
					Required: false,
					Value:    port,
				},
				&cli.StringSliceFlag{ //nolint:exhaustruct
					Name:  "include-date",
					Usage: "only show commits on this date (YYYY-MM-DD), repeatable",
				},
				&cli.StringSliceFlag{ //nolint:exhaustruct
					Name:  "exclude-date",
					Usage: "exclude all commits on this date (YYYY-MM-DD), repeatable",
				},
				&cli.TimestampFlag{ //nolint:exhaustruct
					Name:   "include-time-from",
					Usage:  "only show commits at or after this time of day (HH:MM)",
					Layout: "15:04:05",
				},
				&cli.TimestampFlag{ //nolint:exhaustruct
					Name:   "include-time-to",
					Usage:  "only show commits at or before this time of day (HH:MM)",
					Layout: "15:04:05",
				},
			},
		},
	},
}
