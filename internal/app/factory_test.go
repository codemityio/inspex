package app

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	application := New(
		WithValues(
			"name",
			"version",
			"description",
			"copyright",
			"authorName",
			"authorEmail",
			time.Now().Format(time.RFC3339),
		),
	)

	require.NotNil(t, application)
}
