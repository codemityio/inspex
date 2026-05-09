package glv

import (
	"errors"
	"fmt"
)

var (
	errPkg      = errors.New("glv")
	errDep      = fmt.Errorf("%w: unable to find dependency", errPkg)
	errArgs     = fmt.Errorf("%w: args", errPkg)
	errRead     = fmt.Errorf("%w: read error", errPkg)
	errMarshal  = fmt.Errorf("%w: marshal error", errPkg)
	errEncode   = fmt.Errorf("%w: encode error", errPkg)
	errValidate = fmt.Errorf("%w: validate error", errPkg)
	errScan     = fmt.Errorf("%w: unable to scan", errPkg)
	errExec     = fmt.Errorf("%w: unable to exec", errPkg)
	errServe    = fmt.Errorf("%w: unable to serve", errPkg)
	errWrite    = fmt.Errorf("%w: unable to write", errPkg)
)
