package io

import (
	"errors"
	"fmt"
)

var (
	errPkg   = errors.New("io")
	errStdin = fmt.Errorf("%w: stdin error", errPkg)
)
