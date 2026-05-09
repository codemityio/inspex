//go:build darwin

package fs

import (
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTimeFromTimespec(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		ts   syscall.Timespec
		want time.Time
	}{
		{
			name: "zero timespec returns unix epoch",
			ts:   syscall.Timespec{Sec: 0, Nsec: 0},
			want: time.Unix(0, 0),
		},
		{
			name: "seconds only",
			ts:   syscall.Timespec{Sec: 1_000_000, Nsec: 0},
			want: time.Unix(1_000_000, 0),
		},
		{
			name: "nanoseconds only",
			ts:   syscall.Timespec{Sec: 0, Nsec: 999_999_999},
			want: time.Unix(0, 999_999_999),
		},
		{
			name: "seconds and nanoseconds",
			ts:   syscall.Timespec{Sec: 1_745_000_000, Nsec: 123_456_789},
			want: time.Unix(1_745_000_000, 123_456_789),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, timeFromTimespec(tt.ts))
		})
	}
}

func TestStatTimes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		setup   func(t *testing.T) string
		wantErr bool
		check   func(t *testing.T, accessed, changed, created string)
	}{
		{
			name: "returns non-empty times for a real file",
			setup: func(t *testing.T) string {
				t.Helper()

				f := filepath.Join(t.TempDir(), "file.txt")
				require.NoError(t, os.WriteFile(f, []byte("hello"), 0o600))

				return f
			},
			check: func(t *testing.T, accessed, changed, created string) {
				t.Helper()

				assert.NotEmpty(t, accessed, "accessed time")
				assert.NotEmpty(t, changed, "changed time")
				assert.NotEmpty(t, created, "created time")
			},
		},
		{
			name: "all three times are valid RFC3339 strings",
			setup: func(t *testing.T) string {
				t.Helper()

				f := filepath.Join(t.TempDir(), "file.txt")
				require.NoError(t, os.WriteFile(f, []byte("hello"), 0o600))

				return f
			},
			check: func(t *testing.T, accessed, changed, created string) {
				t.Helper()

				for _, ts := range []string{accessed, changed, created} {
					_, err := time.Parse(time.RFC3339, ts)
					assert.NoError(t, err, "expected RFC3339 format, got %q", ts)
				}
			},
		},
		{
			name: "non-existent path returns empty strings",
			setup: func(t *testing.T) string {
				t.Helper()

				return filepath.Join(t.TempDir(), "nonexistent.txt")
			},
			check: func(t *testing.T, accessed, changed, created string) {
				t.Helper()

				assert.Empty(t, accessed, "accessed time")
				assert.Empty(t, changed, "changed time")
				assert.Empty(t, created, "created time")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			gotAccessed, gotChanged, gotCreated := statTimes(tt.setup(t))

			tt.check(t, gotAccessed, gotChanged, gotCreated)
		})
	}
}
