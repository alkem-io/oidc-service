package analyzer

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAnalyzeComputesCoverage(t *testing.T) {
	t.Parallel()

	root := filepath.Join("..", "testdata", "project")
	res, err := Analyze(context.Background(), Config{Root: root})
	require.NoError(t, err)
	require.NotNil(t, res)

	require.Equal(t, 4, res.DocumentedExports)
	require.Equal(t, 8, res.TotalExports)
	require.InEpsilon(t, 50.0, res.OverallCoverage, 0.001)

	alpha := findPackage(t, res, "internal/alpha")
	require.Equal(t, 3, alpha.Documented)
	require.Equal(t, 0, alpha.Undocumented)
	require.Len(t, alpha.MissingSymbols, 0)

	beta := findPackage(t, res, "internal/beta")
	require.Equal(t, 0, beta.Documented)
	require.Equal(t, 4, beta.Undocumented)
	require.Len(t, beta.MissingSymbols, 4)

	reasons := map[string]bool{}
	for _, m := range beta.MissingSymbols {
		reasons[m.Reason] = true
	}
	require.True(t, reasons[reasonMissing])
	require.True(t, reasons[reasonMismatchedPrefix])
	require.True(t, reasons[reasonFormatting])

	empty := findPackage(t, res, "pkg/empty")
	require.Equal(t, 1, empty.Documented)
	require.Equal(t, 0, empty.Undocumented)
	require.Len(t, empty.MissingSymbols, 0)
}

func findPackage(t *testing.T, res *Result, path string) PackageResult {
	t.Helper()
	for _, pkg := range res.Packages {
		if pkg.Path == path {
			return pkg
		}
	}
	t.Fatalf("package %s not found in result", path)
	return PackageResult{}
}
