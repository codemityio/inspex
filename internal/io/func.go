package io

import (
	"fmt"
	"io"
	"os"
)

// ReadStdin reads JSON from stdin, failing clearly if nothing is piped.
func ReadStdin() ([]byte, error) {
	return readFrom(os.Stdin)
}

// readFrom is the testable core.
func readFrom(file *os.File) ([]byte, error) {
	stat, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("%w: stat stdin: %w", errStdin, err)
	}

	if stat.Mode()&os.ModeCharDevice != 0 {
		return nil, fmt.Errorf(
			"%w: no input provided: use --input-path or pipe JSON via stdin",
			errStdin,
		)
	}

	data, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("%w: unable to read: %w", errStdin, err)
	}

	return data, nil
}
