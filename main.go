package main

import (
	"log"
	"os"

	"github.com/codemityio/gotpl/internal/app"
	"github.com/urfave/cli/v2"
)

func main() {
	application := app.New(
		app.WithValues(
			name,
			``,
			version,
			copyright,
			authorName,
			authorEmail,
			buildTime,
		),
	)

	application.Commands = []*cli.Command{}

	if e := application.Run(os.Args); e != nil {
		log.Fatalf("error: %v", e)
	}
}
