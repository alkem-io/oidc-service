// Package main implements the docstring coverage CLI.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/alkem-io/oidc-service/internal/config"
	"github.com/alkem-io/oidc-service/internal/docstringcov/analyzer"
	"github.com/alkem-io/oidc-service/internal/docstringcov/formatter"
)

func main() {
	ctx := context.Background()
	if err := run(ctx, os.Args[1:], os.Stdout, os.Stderr); err != nil {
		fmt.Fprintf(os.Stderr, "docstringcov: %v\n", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	coverageCfg, err := config.LoadDocstringCoverage()
	if err != nil {
		return err
	}

	fs := flag.NewFlagSet("docstringcov", flag.ContinueOnError)
	fs.SetOutput(stderr)
	root := fs.String("root", ".", "module root to analyze")
	output := fs.String("out", filepath.Join("docs", "coverage.json"), "path to write JSON report")
	threshold := fs.Float64("threshold", coverageCfg.Threshold, "minimum coverage percentage required")
	quiet := fs.Bool("quiet", false, "suppress human summary output")
	if err := fs.Parse(args); err != nil {
		return err
	}

	absRoot, err := filepath.Abs(*root)
	if err != nil {
		return fmt.Errorf("resolve root: %w", err)
	}
	res, err := analyzer.Analyze(ctx, analyzer.Config{Root: absRoot})
	if err != nil {
		return err
	}
	res.Threshold = *threshold
	if err := formatter.WriteJSON(*output, res); err != nil {
		return err
	}
	if !*quiet {
		printSummary(stdout, res)
	}
	return enforceThreshold(res, *threshold)
}

func printSummary(w io.Writer, res *analyzer.Result) {
	_, _ = fmt.Fprintf(w, "Overall coverage: %.2f%% (%d/%d exports documented)\n", res.OverallCoverage, res.DocumentedExports, res.TotalExports)
	for _, pkg := range res.Packages {
		missing := len(pkg.MissingSymbols)
		_, _ = fmt.Fprintf(w, "- %s: %.2f%%", pkg.Path, pkg.Coverage)
		if missing > 0 {
			_, _ = fmt.Fprintf(w, " (%d missing)\n", missing)
			continue
		}
		_, _ = fmt.Fprintln(w)
	}
}

func enforceThreshold(res *analyzer.Result, threshold float64) error {
	if threshold <= 0 {
		return nil
	}
	var failing []string
	for _, pkg := range res.Packages {
		total := pkg.Documented + pkg.Undocumented
		if total == 0 {
			continue
		}
		if pkg.Coverage+1e-9 < threshold {
			failing = append(failing, fmt.Sprintf("%s (%.2f%%)", pkg.Path, pkg.Coverage))
		}
	}
	var reasons []string
	if res.TotalExports > 0 && res.OverallCoverage+1e-9 < threshold {
		reasons = append(reasons, fmt.Sprintf("overall %.2f%% < %.2f%%", res.OverallCoverage, threshold))
	}
	if len(failing) > 0 {
		reasons = append(reasons, fmt.Sprintf("packages below threshold: %s", strings.Join(failing, ", ")))
	}
	if len(reasons) == 0 {
		return nil
	}
	return errors.New(strings.Join(reasons, "; "))
}
