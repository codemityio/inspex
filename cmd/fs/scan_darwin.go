//go:build darwin

package fs

import (
	"syscall"
	"time"
)

func statTimes(path string) (string, string, string) {
	var s syscall.Stat_t

	if err := syscall.Lstat(path, &s); err != nil {
		return "", "", ""
	}

	return formatTime(timeFromTimespec(s.Atimespec)),
		formatTime(timeFromTimespec(s.Ctimespec)),
		formatTime(timeFromTimespec(s.Birthtimespec))
}

func timeFromTimespec(ts syscall.Timespec) time.Time {
	return time.Unix(ts.Sec, ts.Nsec)
}
