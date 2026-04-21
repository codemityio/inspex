package app

import (
	"github.com/urfave/cli/v2"
)

func New(
	options ...Option,
) *cli.App {
	app := cli.NewApp()

	app.Commands = []*cli.Command{}

	for i := range options {
		options[i](app)
	}

	return app
}
