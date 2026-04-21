package browser

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

// fakeStarter is a starter that records calls and returns a controlled error.
type fakeStarter struct {
	startErr error
	started  bool
}

func (f *fakeStarter) Start() error {
	f.started = true

	return f.startErr
}

func TestOpen(t *testing.T) {
	t.Parallel()

	const testURL = "http://localhost:8080"

	tests := []struct {
		name        string
		goos        string
		wantCmd     string
		startErr    error
		wantStarted bool
	}{
		{
			name:        "darwin: starts open command",
			goos:        "darwin",
			wantCmd:     "open",
			wantStarted: true,
		},
		{
			name:        "linux: starts xdg-open command",
			goos:        "linux",
			wantCmd:     "xdg-open",
			wantStarted: true,
		},
		{
			name:        "unsupported OS: does nothing",
			goos:        "windows",
			wantStarted: false,
		},
		{
			name:        "start error is handled without panic",
			goos:        "linux",
			wantCmd:     "xdg-open",
			startErr:    errors.New("no xdg-open found"),
			wantStarted: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			fs := &fakeStarter{startErr: tc.startErr}

			commander := func(_ context.Context, name string, arg ...string) starter {
				assert.Equal(t, tc.wantCmd, name)
				assert.Equal(t, []string{testURL}, arg)

				return fs
			}

			open(context.Background(), testURL, tc.goos, commander)

			assert.Equal(t, tc.wantStarted, fs.started)
		})
	}
}
