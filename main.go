package main

import (
	"log"
	"os"

	"github.com/codemityio/inspex/cmd/fs"
	"github.com/codemityio/inspex/cmd/glv"
	"github.com/codemityio/inspex/internal/app"
	"github.com/urfave/cli/v2"
)

func main() {
	application := app.New(
		app.WithValues(
			name,
			`A collection of command-line tools for the codebase analysis.`,
			version,
			copyright,
			authorName,
			authorEmail,
			buildTime,
		),
	)

	application.Commands = []*cli.Command{
		&fs.App,
		&glv.App,
	}

	application.CommandNotFound = func(_ *cli.Context, cmd string) {
		log.Fatalf("error: unknown command %q", cmd)
	}

	if e := application.Run(os.Args); e != nil {
		log.Fatalf("error: %v", e)
	}
}
