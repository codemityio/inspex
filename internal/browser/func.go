package browser

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
)

// starter is the subset of exec.Cmd we need — allows injection in tests.
type starter interface {
	Start() error
}

func defaultCommander(ctx context.Context, name string, arg ...string) starter { //nolint:ireturn
	return exec.CommandContext(ctx, name, arg...) //nolint:gosec // name is allowlisted above
}

// Open launches the OS default browser once the context is live.
func Open(ctx context.Context, url string) {
	open(ctx, url, runtime.GOOS, defaultCommander)
}

func open(
	ctx context.Context,
	url string,
	goos string,
	commander func(ctx context.Context, name string, arg ...string) starter,
) {
	var name string

	switch goos {
	case "darwin":
		name = "open"
	case "linux":
		name = "xdg-open"
	default:
		return
	}

	cmd := commander(ctx, name, url)
	if e := cmd.Start(); e != nil {
		_, _ = fmt.Fprintf(os.Stderr, "unable to open browser: %s\n", e)
	}
}
