package http

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testTimeout  = 5 * time.Second
	startupDelay = 50 * time.Millisecond
)

// noopOpen is an Opener that does nothing — no browser, no side effects.
var noopOpen Opener = func(_ context.Context, _ string) {}

// recordingOpen captures the URL passed to it for assertion.
func recordingOpen(got *string) Opener {
	return func(_ context.Context, url string) { *got = url }
}

// freePort returns an available TCP port on localhost.
func freePort(t *testing.T) int {
	t.Helper()

	lc := net.ListenConfig{}
	ln, err := lc.Listen(t.Context(), "tcp", ":0")

	require.NoError(t, err, "could not find free port")

	addr, ok := ln.Addr().(*net.TCPAddr)
	require.True(t, ok)

	_ = ln.Close()

	return addr.Port
}

// occupiedPort starts a listener, registers cleanup, and returns its port.
func occupiedPort(t *testing.T) int {
	t.Helper()

	lc := net.ListenConfig{}
	ln, err := lc.Listen(t.Context(), "tcp", ":0")
	require.NoError(t, err, "could not occupy port")

	addr, ok := ln.Addr().(*net.TCPAddr)
	require.True(t, ok)

	t.Cleanup(func() { _ = ln.Close() })

	return addr.Port
}

func TestServe(t *testing.T) {
	t.Parallel()

	okHandler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, "ok")
	})

	tests := []struct {
		name        string
		handler     http.HandlerFunc
		portFn      func(t *testing.T) int
		open        Opener
		wantErr     error
		checkHTTP   bool
		wantStatus  int
		wantOpenURL string
	}{
		{
			name:       "graceful shutdown via context cancel",
			handler:    okHandler,
			portFn:     freePort,
			open:       noopOpen,
			checkHTTP:  true,
			wantStatus: http.StatusOK,
		},
		{
			name:    "open is called with correct URL",
			handler: okHandler,
			portFn: func(t *testing.T) int {
				t.Helper()

				return 19876 // fixed so we can predict the URL
			},
			wantOpenURL: "http://localhost:19876",
		},
		{
			name:    "port already in use returns listen error",
			handler: okHandler,
			portFn:  occupiedPort,
			open:    noopOpen,
			wantErr: net.ErrClosed, // any non-nil sentinel; real check is assert.Error below
		},
		{
			name: "handler returning 404 is reachable",
			handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusNotFound)
			}),
			portFn:     freePort,
			open:       noopOpen,
			checkHTTP:  true,
			wantStatus: http.StatusNotFound,
		},
		{
			name: "handler writing body is reachable",
			handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				_, _ = fmt.Fprint(w, "hello world")
			}),
			portFn:     freePort,
			open:       noopOpen,
			checkHTTP:  true,
			wantStatus: http.StatusOK,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
			defer cancel()

			port := tc.portFn(t)

			open := tc.open

			var gotURL string
			if tc.wantOpenURL != "" {
				open = recordingOpen(&gotURL)
			}

			// Error-path: Serve must return without needing shutdown.
			if tc.wantErr != nil {
				assert.Error(t, Serve(ctx, tc.handler, port, open))

				return
			}

			// Happy-path: start server, exercise it, then cancel ctx to shut down.
			errCh := make(chan error, 1)

			go func() { errCh <- Serve(ctx, tc.handler, port, open) }()

			time.Sleep(startupDelay)

			if tc.checkHTTP {
				url := fmt.Sprintf("http://localhost:%d/", port)
				resp, err := http.Get(url) //nolint:noctx
				require.NoError(t, err, "GET %s", url)

				defer func() {
					_ = resp.Body.Close()
				}()

				assert.Equal(t, tc.wantStatus, resp.StatusCode)
			}

			assert.Equal(t, tc.wantOpenURL, gotURL)

			cancel()

			select {
			case err := <-errCh:
				if !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) {
					require.NoError(t, err, "unexpected error from Serve")
				}
			case <-time.After(testTimeout):
				t.Fatal("timed out waiting for Serve to return")
			}
		})
	}
}
