//go:build linux

package fs

import (
	"time"

	"golang.org/x/sys/unix"
)

func statTimes(path string) (string, string, string) {
	var sx unix.Statx_t

	err := unix.Statx(
		unix.AT_FDCWD,
		path,
		unix.AT_STATX_SYNC_AS_STAT,
		unix.STATX_ATIME|unix.STATX_CTIME|unix.STATX_BTIME,
		&sx,
	)
	if err != nil {
		return "", "", ""
	}

	return formatTime(timeFromStatxTimestamp(sx.Atime)),
		formatTime(timeFromStatxTimestamp(sx.Ctime)),
		formatTime(timeFromStatxTimestamp(sx.Btime))
}

func timeFromStatxTimestamp(ts unix.StatxTimestamp) time.Time {
	return time.Unix(ts.Sec, int64(ts.Nsec))
}
