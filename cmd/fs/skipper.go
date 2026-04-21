package fs

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
)

// skipper holds the state needed to decide whether a path should be skipped.
type skipper struct {
	files    map[string]struct{}
	dirs     map[string]struct{}
	patterns []*regexp.Regexp
}

// newSkipper resolves skip paths relative to basePath and builds a skipper.
func newSkipper(basePath string, skipPaths []string, patterns []*regexp.Regexp) (*skipper, error) {
	skp := &skipper{
		files:    make(map[string]struct{}),
		dirs:     make(map[string]struct{}),
		patterns: patterns,
	}

	for _, p := range skipPaths {
		if err := skp.add(basePath, p); err != nil {
			return nil, err
		}
	}

	return skp, nil
}

// add resolves a single skip path and registers it as a file or directory.
func (s *skipper) add(basePath, path string) error {
	abs := path

	if !filepath.IsAbs(path) {
		abs = filepath.Join(basePath, path)
	}

	var err error

	abs, err = filepath.Abs(abs)
	if err != nil {
		return fmt.Errorf("unable to resolve path %s: %w", path, err)
	}

	info, err := os.Stat(abs)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("unable to access path %s: %w", path, err)
	}

	if info != nil && info.IsDir() {
		s.dirs[abs] = struct{}{}
	} else {
		s.files[abs] = struct{}{}
	}

	return nil
}

// match reports whether p should be skipped.
func (s *skipper) match(p string, isDir bool) bool {
	abs, err := filepath.Abs(p)
	if err != nil {
		return false
	}

	if s.matchesRegex(p) {
		return true
	}

	if isDir {
		_, ok := s.dirs[abs]

		return ok
	}

	if _, ok := s.files[abs]; ok {
		return true
	}

	return s.isUnderSkippedDir(abs)
}

// matchesRegex reports whether the base name of p matches any compiled pattern.
func (s *skipper) matchesRegex(p string) bool {
	base := filepath.Base(p)
	for _, re := range s.patterns {
		if re.MatchString(base) {
			return true
		}
	}

	return false
}

// isUnderSkippedDir reports whether abs lives inside any skipped directory.
func (s *skipper) isUnderSkippedDir(abs string) bool {
	for dir := range s.dirs {
		rel, err := filepath.Rel(dir, abs)
		if err != nil {
			continue
		}

		if len(rel) >= 2 && rel[:2] != ".." {
			return true
		}
	}

	return false
}
