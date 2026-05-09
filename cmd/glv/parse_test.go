package glv

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func mustGit(t *testing.T, dir string, args ...string) string {
	t.Helper()

	cmd := exec.CommandContext(context.TODO(), "git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	require.NoErrorf(t, err, "git %s: %s", strings.Join(args, " "), out)

	return strings.TrimSpace(string(out))
}

func initRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	mustGit(t, dir, "init")
	mustGit(t, dir, "config", "user.email", "test@example.com")
	mustGit(t, dir, "config", "user.name", "Test User")

	return dir
}

func addCommit(t *testing.T, dir, filename, content, message, isoDate string) string {
	t.Helper()
	require.NoError(t, os.WriteFile(filepath.Join(dir, filename), []byte(content), 0o644))
	mustGit(t, dir, "add", filename)

	cmd := exec.CommandContext(context.TODO(), "git", "commit", "-m", message, "--date="+isoDate)
	cmd.Dir = dir

	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_DATE="+isoDate,
		"GIT_COMMITTER_DATE="+isoDate,
		"GIT_AUTHOR_NAME=Test User",
		"GIT_AUTHOR_EMAIL=test@example.com",
		"GIT_COMMITTER_NAME=Test User",
		"GIT_COMMITTER_EMAIL=test@example.com",
	)
	out, err := cmd.CombinedOutput()
	require.NoErrorf(t, err, "git commit: %s", out)

	return mustGit(t, dir, "rev-parse", "--short", "HEAD")
}

func runParse(t *testing.T, dir string) map[string][]*FileChange {
	t.Helper()

	cmd := exec.CommandContext(
		context.TODO(),
		"git", "-C", dir,
		"log",
		"--pretty=format:COMMIT:%h|%ad|%an|%s",
		"--date=format:%Y-%m-%dT%H:%M:%S",
		"--numstat",
		"--diff-filter=ACDMRT",
	)
	out, err := cmd.Output()
	require.NoError(t, err, "git log failed")

	result := map[string][]*FileChange{}

	var current *FileChange

	for line := range strings.SplitSeq(string(out), "\n") {
		line = strings.TrimRight(line, "\r")
		switch {
		case strings.HasPrefix(line, "COMMIT:"):
			parts := strings.SplitN(line[7:], "|", 4)
			if len(parts) == 4 {
				tm, _ := time.Parse("2006-01-02T15:04:05", parts[1])
				current = &FileChange{
					Hash:    parts[0],
					Date:    tm.Format("2006-01-02"),
					Time:    tm.Format("2006-01-02T15:04:05"),
					Author:  parts[2],
					Message: parts[3],
				}
			}
		case current.Hash != "" && line != "":
			cols := strings.Split(line, "\t")
			if len(cols) == 3 {
				fc := current
				fc.LinesAdded = parseInt(cols[0])
				fc.LinesRemoved = parseInt(cols[1])
				fc.Diff = getFileDiff(context.TODO(), current.Hash, cols[2], dir)
				result[cols[2]] = append(result[cols[2]], fc)
			}
		}
	}

	return result
}

func TestParseInt(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  int
	}{
		{"zero", "0", 0},
		{"positive", "42", 42},
		{"large", "999", 999},
		{"dash sentinel", "-", 0},
		{"empty string", "", 0},
		{"non-numeric", "abc", 0},
		{"leading space", " 1", 0},
		{"float", "1.5", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, parseInt(tt.input))
		})
	}
}

func TestValidateRepoPath(t *testing.T) {
	existingFile := func(t *testing.T) string {
		t.Helper()
		f, err := os.CreateTemp(t.TempDir(), "*.txt")
		require.NoError(t, err)
		require.NoError(t, f.Close())

		return f.Name()
	}

	tests := []struct {
		name    string
		path    func(t *testing.T) string
		wantErr error
	}{
		{"empty path", func(_ *testing.T) string { return "" }, nil},
		{
			"valid directory",
			func(t *testing.T) string {
				t.Helper()

				return t.TempDir()
			},
			nil,
		},
		{
			"nonexistent path",
			func(_ *testing.T) string {
				t.Helper()

				return "/no/such/path"
			},
			errArgs,
		},
		{"file not a directory", existingFile, errArgs},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.ErrorIs(t, validateRepoPath(tt.path(t)), tt.wantErr)
		})
	}
}

func TestParseCommitLine(t *testing.T) {
	tests := []struct {
		name        string
		line        string
		wantHash    string
		wantDate    string
		wantAuthor  string
		wantMessage string
	}{
		{
			name:        "well-formed line",
			line:        "COMMIT:abc123|2024-03-10T09:30:00|Alice|fix: bug",
			wantHash:    "abc123",
			wantDate:    "2024-03-10",
			wantAuthor:  "Alice",
			wantMessage: "fix: bug",
		},
		{
			name:        "message with pipe characters",
			line:        "COMMIT:def|2024-01-01T00:00:00|Bob|feat: a|b|c",
			wantMessage: "feat: a|b|c",
		},
		{
			name:     "invalid timestamp falls back to zero date",
			line:     "COMMIT:abc|NOT-A-DATE|Author|msg",
			wantDate: "0001-01-01",
		},
		{
			name: "too few fields",
			line: "COMMIT:abc|2024-01-01T00:00:00|OnlyThree",
		},
		{
			name: "empty payload",
			line: "COMMIT:",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fc := parseCommitLine(tt.line)

			if tt.wantHash != "" {
				assert.Equal(t, tt.wantHash, fc.Hash)
			}

			if tt.wantDate != "" {
				assert.Equal(t, tt.wantDate, fc.Date)
			}

			if tt.wantAuthor != "" {
				assert.Equal(t, tt.wantAuthor, fc.Author)
			}

			if tt.wantMessage != "" {
				assert.Equal(t, tt.wantMessage, fc.Message)
			}
		})
	}
}

func TestParseNumStatLine(t *testing.T) {
	base := &FileChange{Hash: "aaa", Date: "2024-01-01", Author: "Dev", Message: "msg"}

	tests := []struct {
		name         string
		line         string
		wantOK       bool
		wantFilename string
		wantAdded    int
		wantRemoved  int
	}{
		{"normal line", "5\t3\tmain.go", true, "main.go", 5, 3},
		{"binary file dashes", "-\t-\tbinary.bin", true, "binary.bin", 0, 0},
		{"wrong column count", "5\tmain.go", false, "", 0, 0},
		{"empty line", "", false, "", 0, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filename, ent, ok := parseNumStatLine(context.TODO(), tt.line, base, "")
			assert.Equal(t, tt.wantOK, ok)

			assert.Equal(t, tt.wantFilename, filename)

			if !tt.wantOK {
				return
			}

			assert.Equal(t, tt.wantAdded, ent.LinesAdded)
			assert.Equal(t, tt.wantRemoved, ent.LinesRemoved)
			assert.Equal(t, base.Hash, ent.Hash)
			assert.Equal(t, base.Author, ent.Author)
		})
	}
}

func TestParseChange(t *testing.T) {
	tests := []struct {
		name        string
		line        string
		wantOK      bool
		wantType    string
		wantContent string
	}{
		{"added line", "+hello world", true, "added", "hello world"},
		{"removed line", "-goodbye", true, "removed", "goodbye"},
		{"context line", " stays", true, "context", "stays"},
		{"context preserves inner leading space", "  double", true, "context", " double"},
		{"unknown prefix", "\\No newline", false, "", ""},
		{"empty line", "", false, "", ""},
		{"hunk header", "@@ -1,2 +1,2 @@", false, "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ch, ok := parseChange(tt.line)
			assert.Equal(t, tt.wantOK, ok)

			if !tt.wantOK {
				assert.Nil(t, ch)

				return
			}

			require.NotNil(t, ch)
			assert.Equal(t, tt.wantType, ch.Type)
			assert.Equal(t, tt.wantContent, ch.Content)
		})
	}
}

func TestParseHunkHeader(t *testing.T) {
	tests := []struct {
		name         string
		line         string
		wantNil      bool
		wantFromLine int
		wantToLine   int
		wantContext  string
	}{
		{
			name:         "full header with context",
			line:         "@@ -10,4 +10,5 @@ func foo()",
			wantNil:      false,
			wantFromLine: 10,
			wantToLine:   10,
			wantContext:  "func foo()",
		},
		{
			name:         "header without context string",
			line:         "@@ -1,3 +1,3 @@",
			wantNil:      false,
			wantFromLine: 1,
			wantToLine:   1,
			wantContext:  "",
		},
		{
			name:         "no regex match returns nil line pointers",
			line:         "not a hunk header",
			wantNil:      true,
			wantFromLine: 0,
			wantToLine:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			head := parseHunkHeader(tt.line)

			if tt.wantNil {
				assert.Nil(t, head)

				return
			}

			assert.NotNil(t, head.Changes)
			require.NotNil(t, head.FromLine)
			require.NotNil(t, head.ToLine)
			assert.Equal(t, tt.wantFromLine, head.FromLine)
			assert.Equal(t, tt.wantToLine, head.ToLine)
			assert.Equal(t, tt.wantContext, head.Context)
		})
	}
}

func TestParseDiff(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantHunks  int
		checkFirst func(t *testing.T, h *Hunk)
	}{
		{
			name:      "empty input",
			input:     "",
			wantHunks: 0,
		},
		{
			name:      "no hunk marker",
			input:     "just some text\nwithout any hunk headers",
			wantHunks: 0,
		},
		{
			name:      "empty context field",
			input:     "@@ -5,2 +5,2 @@\n-old\n+new",
			wantHunks: 1,
			checkFirst: func(t *testing.T, h *Hunk) {
				t.Helper()

				assert.Empty(t, h.Context)
				assert.Len(t, h.Changes, 2)
			},
		},
		{
			name:      "only context lines",
			input:     "@@ -1,3 +1,3 @@\n line one\n line two\n line three",
			wantHunks: 1,
			checkFirst: func(t *testing.T, h *Hunk) {
				t.Helper()

				for i, ch := range h.Changes {
					assert.Equalf(t, "context", ch.Type, "change[%d].Type", i)
				}
			},
		},
		{
			name: "single hunk all change types",
			input: `@@ -1,4 +1,4 @@ func hello()
 package main
-import "fmt"
+import "log"
 func main() {`,
			wantHunks: 1,
			checkFirst: func(t *testing.T, hnk *Hunk) {
				t.Helper()

				require.NotNil(t, hnk.FromLine)
				require.NotNil(t, hnk.ToLine)
				assert.Equal(t, 1, hnk.FromLine)
				assert.Equal(t, 1, hnk.ToLine)
				assert.Equal(t, "func hello()", hnk.Context)
				require.Len(t, hnk.Changes, 4)
				assert.Equal(t, "context", hnk.Changes[0].Type)
				assert.Equal(t, "removed", hnk.Changes[1].Type)
				assert.Equal(t, `import "fmt"`, hnk.Changes[1].Content)
				assert.Equal(t, "added", hnk.Changes[2].Type)
				assert.Equal(t, `import "log"`, hnk.Changes[2].Content)
				assert.Equal(t, "context", hnk.Changes[3].Type)
			},
		},
		{
			name: "multiple hunks",
			input: `@@ -1,3 +1,3 @@ funcA
-old line A
+new line A
 context A
@@ -10,3 +10,3 @@ funcB
 context B
-old line B
+new line B`,
			wantHunks: 2,
			checkFirst: func(t *testing.T, h *Hunk) {
				t.Helper()

				assert.Equal(t, "funcA", h.Context)
				require.NotNil(t, h.FromLine)
				assert.Equal(t, 1, h.FromLine)
			},
		},
		{
			name:      "no-newline marker line is ignored",
			input:     "@@ -1,1 +1,1 @@\n-old\n+new\n\\ No newline at end of file",
			wantHunks: 1,
			checkFirst: func(t *testing.T, h *Hunk) {
				t.Helper()

				assert.Len(t, h.Changes, 2)
			},
		},
		{
			name:      "content preserves leading whitespace",
			input:     "@@ -1,1 +1,1 @@\n+    indented line",
			wantHunks: 1,
			checkFirst: func(t *testing.T, h *Hunk) {
				t.Helper()

				require.Len(t, h.Changes, 1)
				assert.Equal(t, "    indented line", h.Changes[0].Content)
			},
		},
		{
			name: "git diff header lines before hunk are ignored",
			input: `diff --git a/f.go b/f.go
index abc..def 100644
--- a/f.go
+++ b/f.go
@@ -1,1 +1,1 @@
-old
+new`,
			wantHunks: 1,
			checkFirst: func(t *testing.T, h *Hunk) {
				t.Helper()

				require.Len(t, h.Changes, 2)
				assert.Equal(t, "removed", h.Changes[0].Type)
				assert.Equal(t, "old", h.Changes[0].Content)
				assert.Equal(t, "added", h.Changes[1].Type)
				assert.Equal(t, "new", h.Changes[1].Content)
			},
		},
		{
			name: "second hunk has correct line numbers",
			input: `@@ -1,1 +1,1 @@ funcA
-old A
+new A
@@ -50,2 +50,2 @@ funcB
-old B
+new B`,
			wantHunks:  2,
			checkFirst: func(t *testing.T, _ *Hunk) { t.Helper() },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hunks := parseDiff(tt.input)
			assert.Len(t, hunks, tt.wantHunks)

			if tt.checkFirst != nil && len(hunks) > 0 {
				tt.checkFirst(t, hunks[0])
			}
		})
	}

	t.Run("second hunk from-line is 50", func(t *testing.T) {
		input := `@@ -1,1 +1,1 @@ funcA
-old A
+new A
@@ -50,2 +50,2 @@ funcB
-old B
+new B`
		hunks := parseDiff(input)
		require.Len(t, hunks, 2)
		require.NotNil(t, hunks[1].FromLine)
		assert.Equal(t, 50, hunks[1].FromLine)
		assert.Equal(t, "funcB", hunks[1].Context)
	})
}

func TestSortedMap(t *testing.T) {
	tests := []struct {
		name      string
		input     map[string][]*FileChange
		wantOrder []string
	}{
		{
			name: "keys are sorted lexicographically",
			input: map[string][]*FileChange{
				"z.go": {},
				"a.go": {},
				"m.go": {},
			},
			wantOrder: []string{"a.go", "m.go", "z.go"},
		},
		{
			name:      "empty map",
			input:     map[string][]*FileChange{},
			wantOrder: []string{},
		},
		{
			name:      "single entry is unchanged",
			input:     map[string][]*FileChange{"only.go": {{Hash: "abc"}}},
			wantOrder: []string{"only.go"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out := sortedMap(tt.input)
			b, err := json.Marshal(out)
			require.NoError(t, err)

			prev := -1

			for _, key := range tt.wantOrder {
				idx := strings.Index(string(b), `"`+key+`"`)
				assert.Greater(t, idx, prev, "key %q out of order", key)
				prev = idx
			}
		})
	}
}

func TestEncodeJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		wantJSON string
	}{
		{
			name:     "map of ints",
			input:    map[string]int{"a": 1, "b": 2},
			wantJSON: `{"a": 1, "b": 2}`,
		},
		{
			name:     "empty map",
			input:    map[string][]*FileChange{},
			wantJSON: `{}`,
		},
		{
			name:     "nil pointer",
			input:    (*FileChange)(nil),
			wantJSON: `null`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer

			require.NoError(t, encodeJSON(&buf, tt.input))
			assert.True(t, json.Valid(buf.Bytes()))
			assert.Contains(t, buf.String(), "\n")
			assert.JSONEq(t, tt.wantJSON, buf.String())
		})
	}
}

func TestScanGitLog(t *testing.T) {
	tests := []struct {
		name  string
		input string
		check func(t *testing.T, result map[string][]*FileChange)
		isErr error
	}{
		{
			name:  "numstat lines before first commit are ignored",
			input: "5\t3\torphan.go\nCOMMIT:abc|2024-01-01T00:00:00|Dev|msg\n2\t1\treal.go\n",
			check: func(t *testing.T, result map[string][]*FileChange) {
				t.Helper()

				assert.NotContains(t, result, "orphan.go")
				assert.Contains(t, result, "real.go")
			},
		},
		{
			name:  "blank lines between commits are skipped",
			input: "COMMIT:abc|2024-01-01T00:00:00|Dev|msg\n\n3\t0\tfile.go\n",
			check: func(t *testing.T, result map[string][]*FileChange) {
				t.Helper()

				require.Contains(t, result, "file.go")
				assert.Len(t, result["file.go"], 1)
			},
		},
		{
			name:  "malformed COMMIT line causes following numstat to be dropped",
			input: "COMMIT:abc|2024-01-01T00:00:00|OnlyThree\n5\t2\tfile.go\n",
			check: func(t *testing.T, result map[string][]*FileChange) {
				t.Helper()

				assert.Empty(t, result)
			},
		},
		{
			name:  "malformed COMMIT line after valid one drops subsequent numstat",
			input: "COMMIT:aaa|2024-01-01T00:00:00|Dev|msg\n1\t0\tgood.go\nCOMMIT:bad|2024-01-01T00:00:00|OnlyThree\n5\t2\tfile.go\n",
			check: func(t *testing.T, result map[string][]*FileChange) {
				t.Helper()

				require.Contains(t, result, "good.go")
				assert.NotContains(t, result, "file.go")
			},
		},
		{
			name:  "malformed numstat line is skipped",
			input: "COMMIT:abc|2024-01-01T00:00:00|Dev|msg\nbad-line-no-tabs\n",
			check: func(t *testing.T, result map[string][]*FileChange) {
				t.Helper()

				assert.Empty(t, result)
			},
		},
		{
			name: "two consecutive commits attribute files to correct authors",
			input: strings.Join([]string{
				"COMMIT:aaa|2024-01-01T00:00:00|Alice|first",
				"1\t0\talpha.go",
				"COMMIT:bbb|2024-01-02T00:00:00|Bob|second",
				"2\t1\tbeta.go",
				"",
			}, "\n"),
			check: func(t *testing.T, result map[string][]*FileChange) {
				t.Helper()

				require.Contains(t, result, "alpha.go")
				require.Contains(t, result, "beta.go")
				assert.Equal(t, "Alice", result["alpha.go"][0].Author)
				assert.Equal(t, "Bob", result["beta.go"][0].Author)
			},
		},
		{
			name: "same file touched by multiple commits accumulates entries",
			input: strings.Join([]string{
				"COMMIT:aaa|2024-01-01T00:00:00|Alice|first",
				"1\t0\tshared.go",
				"COMMIT:bbb|2024-01-02T00:00:00|Bob|second",
				"2\t1\tshared.go",
				"",
			}, "\n"),
			check: func(t *testing.T, result map[string][]*FileChange) {
				t.Helper()

				require.Contains(t, result, "shared.go")
				assert.Len(t, result["shared.go"], 2)
				assert.Equal(t, "Alice", result["shared.go"][0].Author)
				assert.Equal(t, "Bob", result["shared.go"][1].Author)
			},
		},
		{
			name:  "empty input produces empty result",
			input: "",
			check: func(t *testing.T, result map[string][]*FileChange) {
				t.Helper()

				assert.Empty(t, result)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := scanGitLog(context.TODO(), strings.NewReader(tt.input), "")

			require.ErrorIs(t, err, tt.isErr)

			tt.check(t, result)
		})
	}
}

func TestGetFileDiff(t *testing.T) {
	dir := initRepo(t)
	addCommit(t, dir, "hello.go", "package main\n", "initial", "2024-01-01T09:00:00")
	validHash := addCommit(
		t,
		dir,
		"hello.go",
		"package main\n\nfunc main() {}\n",
		"add main",
		"2024-01-02T10:00:00",
	)

	tests := []struct {
		name       string
		hash       string
		filename   string
		repoPath   string
		checkHunks func(t *testing.T, hunks []*Hunk)
	}{
		{
			name:     "valid hash and file returns hunks",
			hash:     validHash,
			filename: "hello.go",
			repoPath: dir,
			checkHunks: func(t *testing.T, hunks []*Hunk) {
				t.Helper()

				hasAdded := false

				for _, h := range hunks {
					for _, ch := range h.Changes {
						if ch.Type == "added" {
							hasAdded = true
						}
					}
				}

				assert.True(t, hasAdded, "expected at least one added change")
			},
		},
		{
			name:     "invalid hash returns empty slice",
			hash:     "deadbeef",
			filename: "hello.go",
			repoPath: dir,
		},
		{
			name:     "file not in commit returns empty slice",
			hash:     validHash,
			filename: "nonexistent.go",
			repoPath: dir,
		},
		{
			name:     "empty repoPath falls back to cwd",
			hash:     validHash,
			filename: "hello.go",
			repoPath: "", // cwd is not the repo — git will fail, expect empty
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hunks := getFileDiff(context.TODO(), tt.hash, tt.filename, tt.repoPath)
			require.NotNil(t, hunks)

			if tt.checkHunks != nil {
				tt.checkHunks(t, hunks)
			}
		})
	}
}

func TestParse(t *testing.T) {
	tests := []struct {
		name  string
		setup func(t *testing.T, dir string)
		check func(t *testing.T, result map[string][]*FileChange)
	}{
		{
			name: "single commit populates all fields",
			setup: func(t *testing.T, dir string) {
				t.Helper()

				addCommit(
					t,
					dir,
					"main.go",
					"package main\n",
					"first commit",
					"2024-03-10T09:30:00",
				)
			},
			check: func(t *testing.T, result map[string][]*FileChange) {
				t.Helper()

				require.Contains(t, result, "main.go")
				require.Len(t, result["main.go"], 1)
				c := result["main.go"][0]
				assert.Equal(t, "2024-03-10", c.Date)
				assert.Equal(t, "2024-03-10T09:30:00", c.Time)
				assert.Equal(t, "Test User", c.Author)
				assert.Equal(t, "first commit", c.Message)
				require.NotNil(t, c.LinesAdded)
				assert.Positive(t, c.LinesAdded)
			},
		},
		{
			name: "multiple commits to same file",
			setup: func(t *testing.T, dir string) {
				t.Helper()

				addCommit(
					t,
					dir,
					"a.go",
					"package a\n",
					"commit 1",
					"2024-01-01T08:00:00",
				)
				addCommit(
					t,
					dir,
					"a.go",
					"package a\n\nconst X = 1\n",
					"commit 2",
					"2024-01-02T08:00:00",
				)
				addCommit(
					t,
					dir,
					"a.go",
					"package a\n\nconst X = 2\n",
					"commit 3",
					"2024-01-03T08:00:00",
				)
			},
			check: func(t *testing.T, result map[string][]*FileChange) {
				t.Helper()

				require.Contains(t, result, "a.go")
				assert.Len(t, result["a.go"], 3)
			},
		},
		{
			name: "multiple files appear independently",
			setup: func(t *testing.T, dir string) {
				t.Helper()

				addCommit(t, dir, "alpha.go", "package alpha\n", "add alpha", "2024-02-01T10:00:00")
				addCommit(t, dir, "beta.go", "package beta\n", "add beta", "2024-02-02T10:00:00")
			},
			check: func(t *testing.T, result map[string][]*FileChange) {
				t.Helper()

				assert.Contains(t, result, "alpha.go")
				assert.Contains(t, result, "beta.go")
			},
		},
		{
			name: "diff hunks populated after first commit",
			setup: func(t *testing.T, dir string) {
				t.Helper()

				addCommit(t, dir, "x.go", "package x\n", "base", "2024-05-01T09:00:00")
				addCommit(
					t,
					dir,
					"x.go",
					"package x\n\nfunc F() {}\n",
					"add func",
					"2024-05-02T09:00:00",
				)
			},
			check: func(t *testing.T, result map[string][]*FileChange) {
				t.Helper()

				require.Contains(t, result, "x.go")

				hasHunks := false

				for _, c := range result["x.go"] {
					if len(c.Diff) > 0 {
						hasHunks = true
					}
				}

				assert.True(t, hasHunks, "expected at least one commit with diff hunks")
			},
		},
		{
			name: "lines added and removed are accurate",
			setup: func(t *testing.T, dir string) {
				t.Helper()

				addCommit(
					t,
					dir,
					"f.go",
					"line1\nline2\nline3\n",
					"add 3 lines",
					"2024-06-01T09:00:00",
				)
				addCommit(t, dir, "f.go", "only\n", "shrink", "2024-06-02T09:00:00")
			},
			check: func(t *testing.T, result map[string][]*FileChange) {
				t.Helper()

				require.Contains(t, result, "f.go")
				shrink := result["f.go"][0]
				require.NotNil(t, shrink.LinesAdded)
				require.NotNil(t, shrink.LinesRemoved)
				assert.Equal(t, 1, shrink.LinesAdded)
				assert.Equal(t, 3, shrink.LinesRemoved)
			},
		},
		{
			name: "result is JSON serialisable",
			setup: func(t *testing.T, dir string) {
				t.Helper()

				addCommit(t, dir, "z.go", "package z\n", "init", "2024-07-01T12:00:00")
			},
			check: func(t *testing.T, result map[string][]*FileChange) {
				t.Helper()

				_, err := json.Marshal(result)
				assert.NoError(t, err)
			},
		},
		{
			name: "date and time fields have correct format",
			setup: func(t *testing.T, dir string) {
				t.Helper()

				addCommit(t, dir, "d.go", "package d\n", "check formats", "2024-08-15T14:22:33")
			},
			check: func(t *testing.T, result map[string][]*FileChange) {
				t.Helper()

				require.Contains(t, result, "d.go")
				require.NotEmpty(t, result["d.go"])

				cng := result["d.go"][0]
				_, err := time.Parse("2006-01-02", cng.Date)
				require.NoError(t, err, "Date %q is not YYYY-MM-DD", cng.Date)

				_, err = time.Parse("2006-01-02T15:04:05", cng.Time)
				require.NoError(t, err, "Time %q is not YYYY-MM-DDTHH:MM:SS", cng.Time)
				assert.True(
					t,
					strings.HasPrefix(cng.Time, cng.Date),
					"Time %q does not start with Date %q",
					cng.Time,
					cng.Date,
				)
			},
		},
		{
			name: "single commit touching two files shares the same hash",
			setup: func(t *testing.T, dir string) {
				t.Helper()

				require.NoError(
					t,
					os.WriteFile(filepath.Join(dir, "p.go"), []byte("package p\n"), 0o644),
				)
				require.NoError(
					t,
					os.WriteFile(filepath.Join(dir, "q.go"), []byte("package q\n"), 0o644),
				)
				mustGit(t, dir, "add", "p.go", "q.go")

				cmd := exec.CommandContext(context.TODO(), "git", "commit", "-m", "two files")
				cmd.Dir = dir

				cmd.Env = append(os.Environ(),
					"GIT_AUTHOR_NAME=Test User", "GIT_AUTHOR_EMAIL=test@example.com",
					"GIT_COMMITTER_NAME=Test User", "GIT_COMMITTER_EMAIL=test@example.com",
				)
				out, err := cmd.CombinedOutput()
				require.NoErrorf(t, err, "git commit: %s", out)
			},
			check: func(t *testing.T, result map[string][]*FileChange) {
				t.Helper()

				require.Contains(t, result, "p.go")
				require.Contains(t, result, "q.go")
				assert.Equal(t, result["p.go"][0].Hash, result["q.go"][0].Hash)
			},
		},
		{
			name: "new file has positive LinesAdded and zero LinesRemoved",
			setup: func(t *testing.T, dir string) {
				t.Helper()

				addCommit(
					t,
					dir,
					"new.go",
					"package new\n\nconst A = 1\n",
					"add file",
					"2024-09-01T09:00:00",
				)
			},
			check: func(t *testing.T, result map[string][]*FileChange) {
				t.Helper()

				require.Contains(t, result, "new.go")
				assert.Positive(t, result["new.go"][0].LinesAdded)
				assert.Zero(t, result["new.go"][0].LinesRemoved)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := initRepo(t)
			tt.setup(t, dir)
			result := runParse(t, dir)
			tt.check(t, result)
		})
	}
}
