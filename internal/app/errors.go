package app

import (
	"errors"
	"fmt"
)

var (
	errPkg   = errors.New("app")
	errWrite = fmt.Errorf("%w: unable to write", errPkg)
)
