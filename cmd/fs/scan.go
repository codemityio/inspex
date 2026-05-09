package fs

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"github.com/urfave/cli/v2"
)

func scan(ctx *cli.Context) error {
	path := ctx.String("path")
	outputFormat := ctx.String("output-format")
	recursive := ctx.Bool("recursive")
	skipFields := buildSkipFields(ctx.StringSlice("skip-field"))

	skipPatterns, err := compileSkipRegex(ctx.StringSlice("skip-regex"))
	if err != nil {
		return err
	}

	basePath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("%w: unable to resolve %s: %w", errPath, path, err)
	}

	skp, err := newSkipper(basePath, ctx.StringSlice("skip-path"), skipPatterns)
	if err != nil {
		return fmt.Errorf("%w: unable to initialise skipper: %w", errPkg, err)
	}

	files, err := collectFiles(path, basePath, recursive, skp)
	if err != nil {
		return fmt.Errorf("%w: %w", errPkg, err)
	}

	switch outputFormat {
	case "json":
		return printJSON(files, skipFields)
	case "csv":
		return printCSV(files, skipFields)
	default:
		return fmt.Errorf("%w: unsupported %s", errOutputFormat, outputFormat)
	}
}

// buildSkipFields builds a set of field names to exclude from output.
func buildSkipFields(fields []string) map[string]struct{} {
	m := make(map[string]struct{}, len(fields))
	for _, f := range fields {
		m[f] = struct{}{}
	}

	return m
}

// compileSkipRegex compiles regex patterns, returning an error on the first invalid one.
func compileSkipRegex(patterns []string) ([]*regexp.Regexp, error) {
	compiled := make([]*regexp.Regexp, 0, len(patterns))

	for _, pattern := range patterns {
		re, err := regexp.Compile(pattern)
		if err != nil {
			return nil, fmt.Errorf("%w: invalid pattern %q: %w", errRegex, pattern, err)
		}

		compiled = append(compiled, re)
	}

	return compiled, nil
}

// collectFiles walks basePath and returns all entries that should not be skipped.
func collectFiles(path, basePath string, recursive bool, s *skipper) ([]*Info, error) {
	info, err := os.Stat(basePath)
	if err != nil {
		return nil, fmt.Errorf("unable to access %s: %w", basePath, err)
	}

	if !info.IsDir() {
		return collectSingleFile(path, basePath, s)
	}

	return walkDir(path, basePath, recursive, s)
}

// collectSingleFile handles the case where basePath is a single file.
func collectSingleFile(path, basePath string, s *skipper) ([]*Info, error) {
	if s.match(basePath, false) {
		return nil, nil
	}

	d, err := os.Lstat(basePath)
	if err != nil {
		return nil, fmt.Errorf("unable to stat file %s: %w", basePath, err)
	}

	entry, err := getInfo(path, basePath, fs.FileInfoToDirEntry(d))
	if err != nil {
		return nil, err
	}

	return []*Info{entry}, nil
}

// walkDir recursively walks basePath and collects non-skipped files and directories.
func walkDir(path, basePath string, recursive bool, skp *skipper) ([]*Info, error) {
	var infos []*Info

	err := filepath.WalkDir(basePath, func(pth string, dir fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if dir.IsDir() {
			return handleDir(pth, basePath, path, recursive, skp, &infos)
		}

		if skp.match(pth, false) {
			return nil
		}

		fi, err := buildInfo(path, basePath, pth, dir)
		if err != nil {
			return err
		}

		infos = append(infos, fi)

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("unable to walk: path %s: %w", basePath, err)
	}

	return infos, nil
}

// handleDir decides whether to skip or descend into a directory, and collects
// its metadata when it is not the base path.
func handleDir(
	path, basePath, displayBase string,
	recursive bool,
	s *skipper,
	files *[]*Info,
) error {
	if s.match(path, true) {
		return fs.SkipDir
	}

	if !recursive && path != basePath {
		return fs.SkipDir
	}

	// Collect the directory itself, but skip the root basePath entry since
	// it is the scan target, not a result.
	if path != basePath {
		fi, err := buildInfo(displayBase, basePath, path, nil)
		if err != nil {
			return err
		}

		*files = append(*files, fi)
	}

	return nil
}

// buildInfo resolves the display path and calls getInfo for a single entry.
func buildInfo(path, basePath, pth string, dir fs.DirEntry) (*Info, error) {
	rel, err := filepath.Rel(basePath, pth)
	if err != nil {
		return nil, fmt.Errorf("unable to resolve relative path %s: %w", pth, err)
	}

	// When d is nil (directory case from handleDir), stat it ourselves.
	if dir == nil {
		lstat, e := os.Lstat(pth)
		if e != nil {
			return nil, fmt.Errorf("unable to stat %s: %w", pth, e)
		}

		dir = fs.FileInfoToDirEntry(lstat)
	}

	return getInfo(filepath.Join(path, rel), pth, dir)
}

// infoToMap converts an Info struct into an ordered map keyed by JSON field names.
func infoToMap(info *Info) map[string]any {
	return map[string]any{
		"file":       info.File,
		"createdAt":  info.CreatedAt,
		"modifiedAt": info.ModifiedAt,
		"accessedAt": info.AccessedAt,
		"changedAt":  info.ChangedAt,
		"size":       info.Size,
		"lines":      info.Lines,
		"mode":       info.Mode,
		"isLink":     info.IsLink,
		"isDir":      info.IsDir,
	}
}

// filteredMap returns a copy of m with skipped fields removed, preserving allFields order.
func filteredMap(m map[string]any, skipFields map[string]struct{}) map[string]any {
	out := make(map[string]any, len(m))
	for k, v := range m {
		if _, skip := skipFields[k]; !skip {
			out[k] = v
		}
	}

	return out
}

func printJSON(files []*Info, skipFields map[string]struct{}) error {
	rows := make([]map[string]any, 0, len(files))
	for _, f := range files {
		rows = append(rows, filteredMap(infoToMap(f), skipFields))
	}

	output, err := json.MarshalIndent(rows, "", "  ")
	if err != nil {
		return fmt.Errorf("unable to marshal JSON: %w", err)
	}

	if _, e := fmt.Fprintln(os.Stdout, string(output)); e != nil {
		return fmt.Errorf("%w: %w", errWrite, e)
	}

	return nil
}

func printCSV(files []*Info, skipFields map[string]struct{}) error {
	wr := csv.NewWriter(os.Stdout)

	// Build the ordered list of active (non-skipped) fields.
	activeFields := make([]string, 0, len(allFields))
	for _, field := range allFields {
		if _, skip := skipFields[field]; !skip {
			activeFields = append(activeFields, field)
		}
	}

	if err := wr.Write(activeFields); err != nil {
		return fmt.Errorf("%w: unable to write CSV header: %w", errWrite, err)
	}

	for _, f := range files {
		m := infoToMap(f)
		r := make([]string, 0, len(activeFields))

		for _, field := range activeFields {
			r = append(r, fmt.Sprintf("%v", m[field]))
		}

		if err := wr.Write(r); err != nil {
			return fmt.Errorf("%w: unable to write CSV row: %w", errWrite, err)
		}
	}

	wr.Flush()

	if err := wr.Error(); err != nil {
		return fmt.Errorf("%w: %w", errWrite, err)
	}

	return nil
}

func countLines(path string) (int, error) {
	//nolint:gosec // path is always derived from filepath.WalkDir, not user input directly
	file, err := os.Open(path)
	if err != nil {
		return 0, fmt.Errorf("unable to open file %s: %w", path, err)
	}

	defer func() {
		_ = file.Close()
	}()

	count := 0
	buf := make([]byte, readBufSize)

	for {
		n, e := file.Read(buf)
		for _, b := range buf[:n] {
			if b == '\n' {
				count++
			}
		}

		if e != nil {
			break
		}
	}

	return count, nil
}

func formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}

	return t.UTC().Format(time.RFC3339)
}

func getInfo(displayPath string, absPath string, d fs.DirEntry) (*Info, error) {
	info, err := d.Info()
	if err != nil {
		return nil, fmt.Errorf("stat %s: %w", absPath, err)
	}

	fi := &Info{
		File:       displayPath,
		CreatedAt:  "",
		ModifiedAt: formatTime(info.ModTime()),
		AccessedAt: "",
		ChangedAt:  "",
		Size:       info.Size(),
		Lines:      0,
		Mode:       info.Mode().String(),
		IsLink:     info.Mode()&fs.ModeSymlink != 0,
		IsDir:      info.IsDir(),
	}

	fi.AccessedAt, fi.ChangedAt, fi.CreatedAt = statTimes(absPath)

	// Directories don't have a meaningful line count.
	if !fi.IsDir {
		fi.Lines, err = countLines(absPath)
		if err != nil {
			fi.Lines = -1
		}
	}

	return fi, nil
}
