package http

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
)

// Opener is a function that opens a URL, e.g. in a browser.
type Opener func(ctx context.Context, url string)

// Serve starts the HTTP server and blocks until ctx is canceled.
// The caller is responsible for cancelling ctx on shutdown (e.g. on SIGINT/SIGTERM).
// The Opener is called with the server URL once the listener is ready.
func Serve(ctx context.Context, handler http.HandlerFunc, port int, open Opener) error {
	addr := fmt.Sprintf(":%d", port)
	url := "http://localhost" + addr

	_, _ = fmt.Fprintf(os.Stderr, "serving on %s (press Ctrl+C to stop)\n", url)

	mux := http.NewServeMux()
	mux.HandleFunc("/", handler)

	//nolint:exhaustruct_v5
	srv := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: readHeaderTimeout,
	}

	lc := &net.ListenConfig{
		Control:   nil,
		KeepAlive: 0,
		KeepAliveConfig: net.KeepAliveConfig{
			Enable:   false,
			Idle:     0,
			Interval: 0,
			Count:    0,
		},
	}

	ln, err := lc.Listen(ctx, "tcp", addr)
	if err != nil {
		return fmt.Errorf("%w: listening on %s: %w", errListen, addr, err)
	}

	go func() {
		if e := srv.Serve(ln); e != nil && !errors.Is(e, http.ErrServerClosed) {
			_, _ = fmt.Fprintf(os.Stderr, "error: %s\n", e)
		}
	}()

	open(ctx, url)

	<-ctx.Done()

	_, _ = fmt.Fprintln(os.Stderr, "\nshutting down...")

	//nolint:contextcheck
	if e := srv.Shutdown(context.Background()); e != nil {
		return fmt.Errorf("%w: %w", errShutdown, e)
	}

	return nil
}
