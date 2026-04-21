package app

import (
	"github.com/urfave/cli/v2"
)

// Option to be used to build with optional deps.
type Option func(app *cli.App)
