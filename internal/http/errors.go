package http

import (
	"errors"
	"fmt"
)

var (
	errPkg      = errors.New("http")
	errListen   = fmt.Errorf("%w: listen", errPkg)
	errShutdown = fmt.Errorf("%w: shutdown", errPkg)
)
