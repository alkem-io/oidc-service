package formatter

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/alkem-io/oidc-service/internal/docstringcov/analyzer"
)

func TestWriteJSONIncludesAllPackages(t *testing.T) {
	t.Parallel()

	res := &analyzer.Result{
		OverallCoverage:   90.5,
		DocumentedExports: 9,
		TotalExports:      10,
		Packages: []analyzer.PackageResult{
			{
				Path:         "pkg/empty",
				Coverage:     100,
				Documented:   0,
				Undocumented: 0,
			},
			{
				Path:         "internal/beta",
				Coverage:     50,
				Documented:   2,
				Undocumented: 2,
			},
		},
	}

	output := filepath.Join(t.TempDir(), "coverage.json")
	require.NoError(t, WriteJSON(output, res))

	data, err := os.ReadFile(output) //nolint:gosec // test file reading from temp dir
	require.NoError(t, err)

	var decoded struct {
		Packages []struct {
			Path       string  `json:"path"`
			Coverage   float64 `json:"coverage"`
			Documented int     `json:"documented"`
			Undoc      int     `json:"undocumented"`
		}
	}
	require.NoError(t, json.Unmarshal(data, &decoded))
	require.Len(t, decoded.Packages, 2)

	foundEmpty := false
	for _, pkg := range decoded.Packages {
		if pkg.Path == "pkg/empty" {
			foundEmpty = true
			require.Equal(t, 0, pkg.Undoc)
		}
	}
	require.True(t, foundEmpty, "expected zero-export package to be present in report")
}
