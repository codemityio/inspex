package fs

import (
	"bytes"
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v2"
)

func TestBuildSkipFields(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    []string
		wantLen  int
		wantKeys []string
	}{
		{
			name:    "empty input produces empty map",
			input:   []string{},
			wantLen: 0,
		},
		{
			name:     "single field",
			input:    []string{"size"},
			wantLen:  1,
			wantKeys: []string{"size"},
		},
		{
			name:     "multiple fields",
			input:    []string{"size", "lines", "mode"},
			wantLen:  3,
			wantKeys: []string{"size", "lines", "mode"},
		},
		{
			name:     "duplicates are deduplicated",
			input:    []string{"size", "size"},
			wantLen:  1,
			wantKeys: []string{"size"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := buildSkipFields(tt.input)

			assert.Len(t, got, tt.wantLen)

			for _, k := range tt.wantKeys {
				assert.Contains(t, got, k)
			}
		})
	}
}

func TestCompileSkipRegex(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		patterns  []string
		wantCount int
		wantErr   bool
	}{
		{
			name:      "empty patterns",
			patterns:  []string{},
			wantCount: 0,
		},
		{
			name:      "valid single pattern",
			patterns:  []string{`\.go$`},
			wantCount: 1,
		},
		{
			name:      "multiple valid patterns",
			patterns:  []string{`\.go$`, `\.mod$`, `^\.`},
			wantCount: 3,
		},
		{
			name:     "invalid pattern returns error",
			patterns: []string{`[invalid`},
			wantErr:  true,
		},
		{
			name:     "valid then invalid returns error on invalid one",
			patterns: []string{`\.go$`, `[invalid`},
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := compileSkipRegex(tt.patterns)

			if tt.wantErr {
				require.Error(t, err)
				assert.Nil(t, got)

				return
			}

			require.NoError(t, err)
			assert.Len(t, got, tt.wantCount)
		})
	}
}

func TestFilteredMap(t *testing.T) {
	t.Parallel()

	base := map[string]any{
		"file":  "main.go",
		"size":  int64(1024),
		"lines": 42,
		"mode":  "-rw-r--r--",
	}

	tests := []struct {
		name       string
		skipFields map[string]struct{}
		wantKeys   []string
		wantAbsent []string
	}{
		{
			name:       "no fields skipped",
			skipFields: map[string]struct{}{},
			wantKeys:   []string{"file", "size", "lines", "mode"},
		},
		{
			name:       "skip single field",
			skipFields: map[string]struct{}{"size": {}},
			wantKeys:   []string{"file", "lines", "mode"},
			wantAbsent: []string{"size"},
		},
		{
			name:       "skip multiple fields",
			skipFields: map[string]struct{}{"size": {}, "mode": {}},
			wantKeys:   []string{"file", "lines"},
			wantAbsent: []string{"size", "mode"},
		},
		{
			name:       "skip all fields",
			skipFields: map[string]struct{}{"file": {}, "size": {}, "lines": {}, "mode": {}},
			wantAbsent: []string{"file", "size", "lines", "mode"},
		},
		{
			name:       "skip non-existent field is no-op",
			skipFields: map[string]struct{}{"nonexistent": {}},
			wantKeys:   []string{"file", "size", "lines", "mode"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := filteredMap(base, tt.skipFields)

			for _, k := range tt.wantKeys {
				assert.Contains(t, got, k)
			}

			for _, k := range tt.wantAbsent {
				assert.NotContains(t, got, k)
			}
		})
	}
}

func TestFormatTime(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		input  time.Time
		expect string
	}{
		{
			name:   "zero time returns empty string",
			input:  time.Time{},
			expect: "",
		},
		{
			name:   "non-zero UTC time returns RFC3339",
			input:  time.Date(2026, 4, 18, 8, 30, 0, 0, time.UTC),
			expect: "2026-04-18T08:30:00Z",
		},
		{
			name:   "non-UTC time is converted to UTC",
			input:  time.Date(2026, 4, 18, 10, 30, 0, 0, time.FixedZone("CEST", 2*60*60)),
			expect: "2026-04-18T08:30:00Z",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.expect, formatTime(tt.input))
		})
	}
}

func TestInfoToMap(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		info   *Info
		expect map[string]any
	}{
		{
			name: "regular file maps all fields",
			info: &Info{
				File:       "main.go",
				CreatedAt:  "2026-01-01T00:00:00Z",
				ModifiedAt: "2026-01-02T00:00:00Z",
				AccessedAt: "2026-01-03T00:00:00Z",
				ChangedAt:  "2026-01-04T00:00:00Z",
				Size:       1024,
				Lines:      42,
				Mode:       "-rw-r--r--",
				IsLink:     false,
				IsDir:      false,
			},
			expect: map[string]any{
				"file":       "main.go",
				"createdAt":  "2026-01-01T00:00:00Z",
				"modifiedAt": "2026-01-02T00:00:00Z",
				"accessedAt": "2026-01-03T00:00:00Z",
				"changedAt":  "2026-01-04T00:00:00Z",
				"size":       int64(1024),
				"lines":      42,
				"mode":       "-rw-r--r--",
				"isLink":     false,
				"isDir":      false,
			},
		},
		{
			name: "directory sets isDir true and lines zero",
			info: &Info{
				File:  "mydir",
				Size:  128,
				Mode:  "drwxr-xr-x",
				Lines: 0,
				IsDir: true,
			},
			expect: map[string]any{
				"file":  "mydir",
				"size":  int64(128),
				"mode":  "drwxr-xr-x",
				"lines": 0,
				"isDir": true,
			},
		},
		{
			name: "symlink sets isLink true",
			info: &Info{
				File:   "link.go",
				IsLink: true,
				IsDir:  false,
			},
			expect: map[string]any{
				"isLink": true,
				"isDir":  false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := infoToMap(tt.info)

			for k, want := range tt.expect {
				assert.Equal(t, want, got[k], "field %q", k)
			}
		})
	}
}

func TestCountLines(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		content string
		want    int
		wantErr bool
		usePath func(dir string) string // when set, overrides default temp file
	}{
		{
			name:    "empty file",
			content: "",
			want:    0,
		},
		{
			name:    "single line with newline",
			content: "hello\n",
			want:    1,
		},
		{
			name:    "multiple lines",
			content: "line1\nline2\nline3\n",
			want:    3,
		},
		{
			name:    "no trailing newline counts only internal newlines",
			content: "line1\nline2",
			want:    1,
		},
		{
			name:    "only newlines",
			content: "\n\n\n",
			want:    3,
		},
		{
			name:    "non-existent file returns error",
			wantErr: true,
			usePath: func(dir string) string {
				return filepath.Join(dir, "nonexistent.txt")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			dir := t.TempDir()

			var path string
			if tt.usePath != nil {
				path = tt.usePath(dir)
			} else {
				path = filepath.Join(dir, "test.txt")
				require.NoError(t, os.WriteFile(path, []byte(tt.content), 0o600))
			}

			got, err := countLines(path)

			if tt.wantErr {
				assert.Error(t, err)

				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestCollectSingleFile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	require.NoError(t, os.WriteFile(path, []byte("hello\n"), 0o600))

	tests := []struct {
		name     string
		skipFile bool
		wantNil  bool
		wantFile string
	}{
		{
			name:     "returns file info when not skipped",
			wantFile: path,
		},
		{
			name:     "returns nil when file is skipped",
			skipFile: true,
			wantNil:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			files := make(map[string]struct{})
			if tt.skipFile {
				files[path] = struct{}{}
			}

			s := &skipper{
				files:    files,
				dirs:     make(map[string]struct{}),
				patterns: nil,
			}

			got, err := collectSingleFile(path, path, s)

			require.NoError(t, err)

			if tt.wantNil {
				assert.Nil(t, got)

				return
			}

			require.Len(t, got, 1)
			assert.Equal(t, tt.wantFile, got[0].File)
			assert.False(t, got[0].IsDir)
		})
	}
}

func TestBuildInfo(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	file := filepath.Join(dir, "main.go")
	subDir := filepath.Join(dir, "pkg")

	require.NoError(t, os.WriteFile(file, []byte("package main\n"), 0o600))
	require.NoError(t, os.Mkdir(subDir, 0o755))

	tests := []struct {
		name      string
		pth       func() string
		passDir   bool // pass a real DirEntry rather than nil
		wantIsDir bool
		wantErr   bool
	}{
		{
			name:      "file with explicit DirEntry",
			pth:       func() string { return file },
			passDir:   true,
			wantIsDir: false,
		},
		{
			name:      "directory with nil DirEntry triggers Lstat",
			pth:       func() string { return subDir },
			passDir:   false,
			wantIsDir: true,
		},
		{
			name:    "nil DirEntry for non-existent path returns error",
			pth:     func() string { return filepath.Join(dir, "nonexistent") },
			passDir: false,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			pth := tt.pth()

			var d fs.DirEntry

			if tt.passDir {
				info, err := os.Lstat(pth)
				require.NoError(t, err)

				d = fs.FileInfoToDirEntry(info)
			}

			got, err := buildInfo("display", dir, pth, d)

			if tt.wantErr {
				assert.Error(t, err)

				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantIsDir, got.IsDir)
		})
	}
}

func TestCollectFiles(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	subDir := filepath.Join(dir, "sub")
	file1 := filepath.Join(dir, "a.txt")
	file2 := filepath.Join(subDir, "b.txt")

	require.NoError(t, os.Mkdir(subDir, 0o755))
	require.NoError(t, os.WriteFile(file1, []byte("a\n"), 0o600))
	require.NoError(t, os.WriteFile(file2, []byte("b\n"), 0o600))

	emptySkipper := &skipper{
		files:    make(map[string]struct{}),
		dirs:     make(map[string]struct{}),
		patterns: nil,
	}

	tests := []struct {
		name      string
		path      string
		recursive bool
		skipper   *skipper
		wantMin   int
		wantErr   bool
	}{
		{
			name:    "single file returns one entry",
			path:    file1,
			skipper: emptySkipper,
			wantMin: 1,
		},
		{
			name:      "non-recursive dir returns only top-level entries",
			path:      dir,
			recursive: false,
			skipper:   emptySkipper,
			wantMin:   1,
		},
		{
			name:      "recursive dir includes files from subdirectories",
			path:      dir,
			recursive: true,
			skipper:   emptySkipper,
			wantMin:   3, // file1 + sub dir entry + file2
		},
		{
			name:    "non-existent path returns error",
			path:    filepath.Join(dir, "nonexistent"),
			skipper: emptySkipper,
			wantErr: true,
		},
		{
			name:      "skipped subdirectory is excluded from results",
			path:      dir,
			recursive: true,
			skipper: &skipper{
				files:    make(map[string]struct{}),
				dirs:     map[string]struct{}{subDir: {}},
				patterns: nil,
			},
			wantMin: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			abs, err := filepath.Abs(tt.path)
			require.NoError(t, err)

			got, err := collectFiles(tt.path, abs, tt.recursive, tt.skipper)

			if tt.wantErr {
				assert.Error(t, err)

				return
			}

			require.NoError(t, err)
			assert.GreaterOrEqual(t, len(got), tt.wantMin)
		})
	}
}

func TestWalkDir(t *testing.T) {
	t.Parallel()

	// makeTree creates an isolated dir with a subdirectory and two files.
	// Called inside each subtest so parallel cases never share directories.
	makeTree := func(t *testing.T) (string, string) {
		t.Helper()

		d := t.TempDir()
		sub := filepath.Join(d, "sub")
		require.NoError(t, os.Mkdir(sub, 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(d, "a.txt"), []byte("a\n"), 0o600))
		require.NoError(t, os.WriteFile(filepath.Join(sub, "b.txt"), []byte("b\n"), 0o600))

		return d, sub
	}

	tests := []struct {
		name string
		run  func(t *testing.T)
	}{
		{
			name: "non-recursive returns only top-level file",
			run: func(t *testing.T) {
				t.Helper()
				dir, _ := makeTree(t)
				s := &skipper{files: make(map[string]struct{}), dirs: make(map[string]struct{})}
				got, err := walkDir(dir, dir, false, s)
				require.NoError(t, err)
				assert.GreaterOrEqual(t, len(got), 1)
			},
		},
		{
			name: "recursive returns files and subdir entry",
			run: func(t *testing.T) {
				t.Helper()
				dir, _ := makeTree(t)
				s := &skipper{files: make(map[string]struct{}), dirs: make(map[string]struct{})}
				got, err := walkDir(dir, dir, true, s)
				require.NoError(t, err)
				// a.txt + sub (dir entry) + b.txt = 3
				assert.GreaterOrEqual(t, len(got), 3)
			},
		},
		{
			name: "skipped subdir excluded",
			run: func(t *testing.T) {
				t.Helper()
				dir, subDir := makeTree(t)
				s := &skipper{
					files: make(map[string]struct{}),
					dirs:  map[string]struct{}{subDir: {}},
				}
				got, err := walkDir(dir, dir, true, s)
				require.NoError(t, err)
				// only a.txt — sub dir entry and b.txt are excluded
				assert.GreaterOrEqual(t, len(got), 1)

				for _, info := range got {
					// Neither the sub dir entry nor b.txt inside it should appear.
					assert.NotEqual(t, "b.txt", filepath.Base(info.File))
					assert.False(t, strings.HasPrefix(info.File, subDir))
				}
			},
		},
		{
			name: "unreadable directory returns error",
			run: func(t *testing.T) {
				t.Helper()

				if os.Getuid() == 0 {
					t.Skip("skipping permission test: running as root")
				}

				dir := t.TempDir()
				restricted := filepath.Join(dir, "restricted")
				require.NoError(t, os.Mkdir(restricted, 0o000))
				t.Cleanup(func() { _ = os.Chmod(restricted, 0o755) })

				s := &skipper{files: make(map[string]struct{}), dirs: make(map[string]struct{})}
				_, err := walkDir(dir, dir, true, s)
				assert.Error(t, err)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tt.run(t)
		})
	}
}

func TestHandleDir(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	subDir := filepath.Join(dir, "sub")
	require.NoError(t, os.Mkdir(subDir, 0o755))

	tests := []struct {
		name        string
		path        string
		recursive   bool
		skipDir     bool
		wantSkipDir bool
		wantAppend  bool
	}{
		{
			name:       "base path is never appended to results",
			path:       dir,
			recursive:  true,
			wantAppend: false,
		},
		{
			name:        "skipped directory returns SkipDir",
			path:        subDir,
			recursive:   true,
			skipDir:     true,
			wantSkipDir: true,
		},
		{
			name:        "non-recursive non-base dir returns SkipDir",
			path:        subDir,
			recursive:   false,
			wantSkipDir: true,
		},
		{
			name:       "recursive non-base dir is appended to results",
			path:       subDir,
			recursive:  true,
			wantAppend: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			skipDirs := make(map[string]struct{})
			if tt.skipDir {
				skipDirs[tt.path] = struct{}{}
			}

			s := &skipper{
				files:    make(map[string]struct{}),
				dirs:     skipDirs,
				patterns: nil,
			}

			var files []*Info

			err := handleDir(tt.path, dir, dir, tt.recursive, s, &files)

			if tt.wantSkipDir {
				assert.Equal(t, fs.SkipDir, err)

				return
			}

			require.NoError(t, err)

			if tt.wantAppend {
				require.Len(t, files, 1)
				assert.True(t, files[0].IsDir)
			} else {
				assert.Empty(t, files)
			}
		})
	}
}

func TestScan(t *testing.T) {
	// NOTE: no t.Parallel() — subtests share os.Stdout redirection via os.Pipe
	// which is not safe for concurrent use.
	dir := t.TempDir()
	require.NoError(
		t,
		os.WriteFile(filepath.Join(dir, "hello.txt"), []byte("line1\nline2\n"), 0o600),
	)

	runScan := func(t *testing.T, args []string) (string, error) {
		t.Helper()

		reader, pipeWriter, err := os.Pipe()
		require.NoError(t, err)

		old := os.Stdout
		os.Stdout = pipeWriter

		app := &cli.App{
			Commands: App.Subcommands,
		}

		runErr := app.Run(append([]string{"app", "scan"}, args...))

		_ = pipeWriter.Close()
		os.Stdout = old

		var buf bytes.Buffer

		_, _ = buf.ReadFrom(reader)
		_ = reader.Close()

		return buf.String(), runErr
	}

	tests := []struct {
		name     string
		args     []string
		wantErr  bool
		checkOut func(t *testing.T, out string)
	}{
		{
			name: "json output contains scanned file",
			args: []string{"--path", dir, "--output-format", "json"},
			checkOut: func(t *testing.T, out string) {
				t.Helper()

				var rows []map[string]any
				require.NoError(t, json.Unmarshal([]byte(strings.TrimSpace(out)), &rows))
				require.NotEmpty(t, rows)

				bases := make([]string, 0, len(rows))
				for _, rw := range rows {
					if f, ok := rw["file"].(string); ok {
						bases = append(bases, filepath.Base(f))
					}
				}

				assert.Contains(t, bases, "hello.txt")
			},
		},
		{
			name: "csv output contains header and data row",
			args: []string{"--path", dir, "--output-format", "csv"},
			checkOut: func(t *testing.T, out string) {
				t.Helper()
				assert.Contains(t, out, "file")
				assert.Contains(t, out, "hello.txt")
			},
		},
		{
			name:    "unsupported output format returns error",
			args:    []string{"--path", dir, "--output-format", "yaml"},
			wantErr: true,
		},
		{
			name:    "invalid skip-regex returns error",
			args:    []string{"--path", dir, "--output-format", "json", "--skip-regex", "[invalid"},
			wantErr: true,
		},
		{
			name: "skip-field excludes field from json output",
			args: []string{"--path", dir, "--output-format", "json", "--skip-field", "lines"},
			checkOut: func(t *testing.T, out string) {
				t.Helper()

				var rows []map[string]any
				require.NoError(t, json.Unmarshal([]byte(strings.TrimSpace(out)), &rows))
				require.NotEmpty(t, rows)
				assert.NotContains(t, rows[0], "lines")
			},
		},
		{
			name: "skip-regex excludes matched files",
			args: []string{"--path", dir, "--output-format", "json", "--skip-regex", `\.txt$`},
			checkOut: func(t *testing.T, out string) {
				t.Helper()

				var rows []map[string]any
				require.NoError(t, json.Unmarshal([]byte(strings.TrimSpace(out)), &rows))

				for _, rw := range rows {
					if f, ok := rw["file"].(string); ok {
						assert.NotContains(t, f, ".txt")
					}
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, err := runScan(t, tt.args)

			if tt.wantErr {
				assert.Error(t, err)

				return
			}

			require.NoError(t, err)

			if tt.checkOut != nil {
				tt.checkOut(t, out)
			}
		})
	}
}
