package main

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRunGeneratesReport(t *testing.T) {
	t.Parallel()

	root := filepath.Join("..", "..", "internal", "docstringcov", "testdata", "project")
	out := filepath.Join(t.TempDir(), "coverage.json")
	err := run(context.Background(), []string{"-root", root, "-out", out, "-threshold", "0"}, io.Discard, io.Discard)
	require.NoError(t, err)

	data, err := os.ReadFile(out)
	require.NoError(t, err)

	var decoded struct {
		Packages []struct {
			Path string `json:"path"`
		}
	}
	require.NoError(t, json.Unmarshal(data, &decoded))
	require.NotEmpty(t, decoded.Packages)
}
