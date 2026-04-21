package app

import (
	"runtime/debug"
	"time"

	"github.com/urfave/cli/v2"
)

// WithValues set values.
func WithValues(
	name, description, version, copyright, authorName, authorEmail, buildTime string,
) Option {
	return func(app *cli.App) {
		bi, _ := debug.ReadBuildInfo()

		app.Name = name
		app.Description = description
		app.Version = resolveVersion(version, bi)
		app.Copyright = copyright

		app.Authors = []*cli.Author{
			{
				Name:  authorName,
				Email: authorEmail,
			},
		}

		app.HideVersion = false

		resolvedBuildTime := resolveBuildTime(buildTime, bi)
		if resolvedBuildTime == "" {
			return
		}

		parsedBuildTime, err := time.Parse(time.RFC3339, resolvedBuildTime)
		if err != nil {
			panic(err)
		}

		app.Compiled = parsedBuildTime
	}
}

func resolveVersion(fallback string, bi *debug.BuildInfo) string {
	if fallback != "" {
		return fallback
	}

	if bi == nil {
		return "latest"
	}

	if bi.Main.Version != "" && bi.Main.Version != "(devel)" {
		return bi.Main.Version
	}

	return "latest"
}

func resolveBuildTime(fallback string, bi *debug.BuildInfo) string {
	if bi == nil {
		return fallback
	}

	for _, s := range bi.Settings {
		if s.Key == "vcs.time" {
			return s.Value
		}
	}

	return fallback
}
