package fs

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSkipper(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	subDir := filepath.Join(dir, "sub")
	file := filepath.Join(dir, "file.txt")

	require.NoError(t, os.Mkdir(subDir, 0o755))
	require.NoError(t, os.WriteFile(file, []byte(""), 0o600))

	tests := []struct {
		name      string
		skipPaths []string
		setup     func(t *testing.T, dir string) []string
		wantDirs  int
		wantFiles int
		wantErr   bool
	}{
		{
			name:      "no skip paths",
			skipPaths: []string{},
			wantDirs:  0,
			wantFiles: 0,
		},
		{
			name:      "skip existing directory",
			skipPaths: []string{subDir},
			wantDirs:  1,
			wantFiles: 0,
		},
		{
			name:      "skip existing file",
			skipPaths: []string{file},
			wantDirs:  0,
			wantFiles: 1,
		},
		{
			name:      "skip non-existent path registers as file",
			skipPaths: []string{filepath.Join(dir, "nonexistent.txt")},
			wantDirs:  0,
			wantFiles: 1,
		},
		{
			name:      "relative path is resolved against basePath",
			skipPaths: []string{"sub"},
			wantDirs:  1,
			wantFiles: 0,
		},
		{
			name:    "stat error that is not ErrNotExist returns error",
			wantErr: true,
			setup: func(t *testing.T, dir string) []string {
				t.Helper()
				// A path exceeding NAME_MAX (255 bytes) causes os.Stat to return
				// ENAMETOOLONG, which is not os.IsNotExist, reliably on both Linux and Darwin.
				return []string{filepath.Join(dir, strings.Repeat("a", 300))}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			skipPaths := tt.skipPaths
			if tt.setup != nil {
				skipPaths = tt.setup(t, dir)
			}

			skp, err := newSkipper(dir, skipPaths, nil)

			if tt.wantErr {
				assert.Error(t, err)

				return
			}

			require.NoError(t, err)
			assert.Len(t, skp.dirs, tt.wantDirs)
			assert.Len(t, skp.files, tt.wantFiles)
		})
	}
}

func TestSkipperMatch(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	subDir := filepath.Join(dir, "vendor")
	nestedFile := filepath.Join(subDir, "lib.go")
	skippedFile := filepath.Join(dir, "skip.go")

	require.NoError(t, os.Mkdir(subDir, 0o755))
	require.NoError(t, os.WriteFile(nestedFile, []byte(""), 0o600))
	require.NoError(t, os.WriteFile(skippedFile, []byte(""), 0o600))

	tests := []struct {
		name    string
		skipper *skipper
		path    string
		isDir   bool
		want    bool
	}{
		{
			name:    "empty skipper never matches file",
			skipper: &skipper{files: map[string]struct{}{}, dirs: map[string]struct{}{}},
			path:    skippedFile,
			isDir:   false,
			want:    false,
		},
		{
			name:    "empty skipper never matches dir",
			skipper: &skipper{files: map[string]struct{}{}, dirs: map[string]struct{}{}},
			path:    subDir,
			isDir:   true,
			want:    false,
		},
		{
			name: "explicitly skipped file matches",
			skipper: &skipper{
				files: map[string]struct{}{skippedFile: {}},
				dirs:  map[string]struct{}{},
			},
			path:  skippedFile,
			isDir: false,
			want:  true,
		},
		{
			name:    "explicitly skipped dir matches",
			skipper: &skipper{files: map[string]struct{}{}, dirs: map[string]struct{}{subDir: {}}},
			path:    subDir,
			isDir:   true,
			want:    true,
		},
		{
			name:    "file inside skipped dir matches",
			skipper: &skipper{files: map[string]struct{}{}, dirs: map[string]struct{}{subDir: {}}},
			path:    nestedFile,
			isDir:   false,
			want:    true,
		},
		{
			name:    "file outside skipped dir does not match",
			skipper: &skipper{files: map[string]struct{}{}, dirs: map[string]struct{}{subDir: {}}},
			path:    skippedFile,
			isDir:   false,
			want:    false,
		},
		{
			name: "regex pattern matches file base name",
			skipper: &skipper{
				files:    map[string]struct{}{},
				dirs:     map[string]struct{}{},
				patterns: []*regexp.Regexp{regexp.MustCompile(`\.go$`)},
			},
			path:  skippedFile,
			isDir: false,
			want:  true,
		},
		{
			name: "regex pattern matches dir base name",
			skipper: &skipper{
				files:    map[string]struct{}{},
				dirs:     map[string]struct{}{},
				patterns: []*regexp.Regexp{regexp.MustCompile(`^vendor$`)},
			},
			path:  subDir,
			isDir: true,
			want:  true,
		},
		{
			name: "regex pattern does not match unrelated file",
			skipper: &skipper{
				files:    map[string]struct{}{},
				dirs:     map[string]struct{}{},
				patterns: []*regexp.Regexp{regexp.MustCompile(`\.mod$`)},
			},
			path:  skippedFile,
			isDir: false,
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, tt.skipper.match(tt.path, tt.isDir))
		})
	}
}

func TestSkipperMatchesRegex(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		patterns []string
		path     string
		want     bool
	}{
		{
			name:     "no patterns",
			patterns: []string{},
			path:     "main.go",
			want:     false,
		},
		{
			name:     "pattern matches base name only not full path",
			patterns: []string{`\.go$`},
			path:     "/some/deep/path/main.go",
			want:     true,
		},
		{
			name:     "pattern does not match different extension",
			patterns: []string{`\.go$`},
			path:     "README.md",
			want:     false,
		},
		{
			name:     "hidden file pattern",
			patterns: []string{`^\.`},
			path:     ".gitignore",
			want:     true,
		},
		{
			name:     "first of multiple patterns matches",
			patterns: []string{`\.go$`, `\.mod$`},
			path:     "main.go",
			want:     true,
		},
		{
			name:     "second of multiple patterns matches",
			patterns: []string{`\.go$`, `\.mod$`},
			path:     "go.mod",
			want:     true,
		},
		{
			name:     "no pattern matches",
			patterns: []string{`\.go$`, `\.mod$`},
			path:     "README.md",
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			compiled := make([]*regexp.Regexp, 0, len(tt.patterns))
			for _, p := range tt.patterns {
				compiled = append(compiled, regexp.MustCompile(p))
			}

			s := &skipper{patterns: compiled}

			assert.Equal(t, tt.want, s.matchesRegex(tt.path))
		})
	}
}

func TestSkipperIsUnderSkippedDir(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		dirs    map[string]struct{}
		absPath string
		want    bool
	}{
		{
			name:    "no skipped dirs",
			dirs:    map[string]struct{}{},
			absPath: "/project/main.go",
			want:    false,
		},
		{
			name:    "file directly inside skipped dir",
			dirs:    map[string]struct{}{"/project/vendor": {}},
			absPath: "/project/vendor/lib.go",
			want:    true,
		},
		{
			name:    "file nested deeply inside skipped dir",
			dirs:    map[string]struct{}{"/project/vendor": {}},
			absPath: "/project/vendor/pkg/sub/lib.go",
			want:    true,
		},
		{
			name:    "file outside skipped dir",
			dirs:    map[string]struct{}{"/project/vendor": {}},
			absPath: "/project/main.go",
			want:    false,
		},
		{
			name:    "sibling directory not considered under skipped dir",
			dirs:    map[string]struct{}{"/project/vendor": {}},
			absPath: "/project/internal/pkg.go",
			want:    false,
		},
		{
			name: "file under one of multiple skipped dirs",
			dirs: map[string]struct{}{
				"/project/vendor":   {},
				"/project/testdata": {},
			},
			absPath: "/project/testdata/fixture.json",
			want:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			s := &skipper{dirs: tt.dirs}

			assert.Equal(t, tt.want, s.isUnderSkippedDir(tt.absPath))
		})
	}
}
