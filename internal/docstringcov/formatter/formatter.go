// Package formatter serializes docstring coverage results to persistent formats.
package formatter

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/alkem-io/oidc-service/internal/docstringcov/analyzer"
)

// WriteJSON serializes the analyzer result to the provided path, creating parent directories as needed.
func WriteJSON(path string, res *analyzer.Result) error {
	if res == nil {
		return errors.New("coverage result is required")
	}
	if path == "" {
		return errors.New("output path is required")
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return fmt.Errorf("create coverage directory: %w", err)
	}
	payload, err := json.MarshalIndent(res, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal coverage report: %w", err)
	}
	payload = append(payload, '\n')
	if err := os.WriteFile(path, payload, 0o600); err != nil {
		return fmt.Errorf("write coverage report: %w", err)
	}
	return nil
}
