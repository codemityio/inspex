package io

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// pipeWith creates an *os.File that reads from content, via an OS pipe.
// A pipe's Stat() returns ModeNamedPipe, so ModeCharDevice is NOT set —
// exactly what ReadStdin sees when data is piped in on the real stdin.
func pipeWith(t *testing.T, content []byte) *os.File {
	t.Helper()

	r, w, err := os.Pipe()
	require.NoError(t, err, "os.Pipe")

	_, err = w.Write(content)
	require.NoError(t, err, "pipe write")

	_ = w.Close()

	t.Cleanup(func() { _ = r.Close() })

	return r
}

func TestReadFrom(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		fileFn  func(t *testing.T) *os.File
		want    []byte
		wantErr error
	}{
		{
			name: "reads JSON from pipe",
			fileFn: func(t *testing.T) *os.File {
				t.Helper()

				return pipeWith(t, []byte(`{"key":"value"}`))
			},
			want: []byte(`{"key":"value"}`),
		},
		{
			name: "reads empty input from pipe",
			fileFn: func(t *testing.T) *os.File {
				t.Helper()

				return pipeWith(t, []byte{})
			},
			want: []byte{},
		},
		{
			name: "returns errStdin when stdin is a char device (no pipe)",
			fileFn: func(t *testing.T) *os.File {
				t.Helper()

				f, err := os.Open("/dev/tty")
				if err != nil {
					t.Skip("no /dev/tty on this platform")
				}

				t.Cleanup(func() { _ = f.Close() })

				return f
			},
			wantErr: errStdin,
		},
		{
			name: "returns errStdin when stat fails (closed file)",
			fileFn: func(t *testing.T) *os.File {
				t.Helper()

				r, w, err := os.Pipe()
				require.NoError(t, err, "os.Pipe")

				_ = w.Close()
				_ = r.Close()

				return r
			},
			wantErr: errStdin,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := readFrom(tc.fileFn(t))

			require.ErrorIs(t, err, tc.wantErr)
			assert.Equal(t, tc.want, got)
		})
	}
}
