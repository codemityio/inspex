package glv

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/codemityio/inspex/internal/app"
	"github.com/urfave/cli/v2"
)

func parse(ctx *cli.Context) error {
	if e := app.CheckCommand(ctx, "git", "git not found"); e != nil {
		return fmt.Errorf("%w: %w", errDep, e)
	}

	repoPath := ctx.String("path")
	if e := validateRepoPath(repoPath); e != nil {
		return e
	}

	stdout, cmd, err := startGitLog(ctx.Context, repoPath)
	if err != nil {
		return err
	}

	defer func() {
		_ = stdout.Close()
	}()

	result, err := scanGitLog(ctx.Context, stdout, repoPath)
	if err != nil {
		return err
	}

	if e := cmd.Wait(); e != nil {
		return fmt.Errorf("%w: git log exited with error: %w", errExec, e)
	}

	return encodeJSON(os.Stdout, sortedMap(result))
}

func validateRepoPath(repoPath string) error {
	if repoPath == "" {
		return nil
	}

	info, err := os.Stat(repoPath)
	if err != nil {
		return fmt.Errorf("%w: path %q: %w", errArgs, repoPath, err)
	}

	if !info.IsDir() {
		return fmt.Errorf("%w: path %q is not a directory", errArgs, repoPath)
	}

	return nil
}

func startGitLog(ctx context.Context, repoPath string) (io.ReadCloser, *exec.Cmd, error) {
	args := make([]string, 0, gitLogArgsLength)
	if repoPath != "" {
		args = append(args, "-C", repoPath)
	}

	args = append(
		args,
		"log",
		"--pretty=format:COMMIT:%h|%ad|%an|%s",
		"--date=format:%Y-%m-%dT%H:%M:%S",
		"--numstat",
		"--diff-filter=ACDMRT",
	)

	cmd := exec.CommandContext(ctx, "git", args...) //nolint:gosec
	cmd.Stderr = os.Stderr

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, nil, fmt.Errorf("%w: failed to get stdout pipe: %w", errExec, err)
	}

	if e := cmd.Start(); e != nil {
		defer func() {
			_ = stdout.Close()
		}()

		return nil, nil, fmt.Errorf("%w: failed to start git log: %w", errExec, e)
	}

	return stdout, cmd, nil
}

func scanGitLog(
	ctx context.Context,
	r io.Reader,
	repoPath string,
) (map[string][]*FileChange, error) {
	result := map[string][]*FileChange{}

	var current *FileChange

	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "COMMIT:") {
			current = parseCommitLine(line)
		} else if current != nil && line != "" {
			if filename, entry, ok := parseNumStatLine(ctx, line, current, repoPath); ok {
				result[filename] = append(result[filename], entry)
			}
		}
	}

	if e := scanner.Err(); e != nil {
		return nil, fmt.Errorf("%w: %w", errScan, e)
	}

	return result, nil
}

func parseCommitLine(line string) *FileChange {
	parts := strings.SplitN(line[7:], "|", linePartsNumber)
	if len(parts) != linePartsNumber {
		return nil
	}

	t, err := time.Parse("2006-01-02T15:04:05", parts[1])
	if err != nil {
		t = time.Time{}
	}

	return &FileChange{
		Hash:         parts[0],
		Date:         t.Format(time.DateOnly),
		Time:         t.Format(time.RFC3339),
		Author:       parts[2],
		Message:      parts[3],
		LinesAdded:   0,
		LinesRemoved: 0,
		Diff:         nil,
	}
}

func parseNumStatLine(
	ctx context.Context,
	line string,
	current *FileChange,
	repoPath string,
) (string, *FileChange, bool) {
	cols := strings.Split(line, "\t")
	if len(cols) != colsNumber {
		return "", nil, false
	}

	filename := cols[2]

	return filename, &FileChange{
		Hash:         current.Hash,
		Date:         current.Date,
		Time:         current.Time,
		Author:       current.Author,
		Message:      current.Message,
		LinesAdded:   parseInt(cols[0]),
		LinesRemoved: parseInt(cols[1]),
		Diff:         getFileDiff(ctx, current.Hash, filename, repoPath),
	}, true
}

func sortedMap(m map[string][]*FileChange) map[string][]*FileChange {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}

	sort.Strings(keys)

	ordered := make(map[string][]*FileChange, len(keys))
	for _, k := range keys {
		ordered[k] = m[k]
	}

	return ordered
}

func encodeJSON(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")

	if e := enc.Encode(v); e != nil {
		return fmt.Errorf("%w: failed to encode JSON: %w", errEncode, e)
	}

	return nil
}

func parseDiff(diffText string) []*Hunk {
	var (
		hunks       []*Hunk
		currentHunk *Hunk
	)

	for line := range strings.SplitSeq(diffText, "\n") {
		if strings.HasPrefix(line, "@@") {
			if currentHunk != nil {
				hunks = append(hunks, currentHunk)
			}

			currentHunk = parseHunkHeader(line)

			continue
		}

		if currentHunk != nil {
			if change, ok := parseChange(line); ok {
				currentHunk.Changes = append(currentHunk.Changes, change)
			}
		}
	}

	if currentHunk != nil {
		hunks = append(hunks, currentHunk)
	}

	return hunks
}

func parseHunkHeader(line string) *Hunk {
	match := hunkHeaderRegex.FindStringSubmatch(line)
	if match == nil {
		return nil
	}

	fromLine, _ := strconv.Atoi(match[1])
	toLine, _ := strconv.Atoi(match[2])

	hunk := &Hunk{
		FromLine: fromLine,
		ToLine:   toLine,
		Context:  strings.TrimSpace(match[3]),
		Changes:  []*Change{},
	}

	return hunk
}

func parseChange(line string) (*Change, bool) {
	prefixes := map[byte]string{'+': "added", '-': "removed", ' ': "context"}

	if len(line) == 0 {
		return nil, false
	}

	changeType, ok := prefixes[line[0]]
	if !ok {
		return nil, false
	}

	return &Change{Type: changeType, Content: line[1:]}, true
}

func getFileDiff(ctx context.Context, commitHash, filename, repoPath string) []*Hunk {
	args := make([]string, 0, argsLength)
	if repoPath != "" {
		args = append(args, "-C", repoPath)
	}

	args = append(args, "show", "--unified=3", commitHash, "--", filename)

	out, err := exec.CommandContext(ctx, "git", args...).Output() //nolint:gosec
	if err != nil {
		return []*Hunk{}
	}

	raw := string(out)

	loc := hunkStartRegex.FindStringIndex(raw)
	if loc == nil {
		return []*Hunk{}
	}

	return parseDiff(strings.TrimSpace(raw[loc[0]:]))
}

func parseInt(s string) int {
	if s == "-" {
		return 0
	}

	v, err := strconv.Atoi(s)
	if err != nil {
		return 0
	}

	return v
}
