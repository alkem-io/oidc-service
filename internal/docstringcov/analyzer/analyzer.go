// Package analyzer inspects Go packages and computes docstring coverage metrics.
package analyzer

import (
	"context"
	"errors"
	"fmt"
	"go/ast"
	"go/token"
	"math"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"golang.org/x/tools/go/packages"
)

const (
	kindPackage = "package"
	kindFunc    = "func"
	kindMethod  = "method"
	kindType    = "type"

	reasonMissing          = "missing"
	reasonMismatchedPrefix = "mismatched-prefix"
	reasonFormatting       = "formatting"
)

// Config describes how the analyzer should load Go packages.
type Config struct {
	// Root is the absolute or relative directory that hosts the Go module to scan.
	Root string
	// Patterns default to ./... when empty and support globbing understood by go/packages.
	Patterns []string
}

// Result captures docstring coverage metrics for the entire repository.
type Result struct {
	GeneratedAt       time.Time       `json:"generatedAt"`
	OverallCoverage   float64         `json:"overallCoverage"`
	DocumentedExports int             `json:"documentedExports"`
	TotalExports      int             `json:"totalExports"`
	Threshold         float64         `json:"threshold"`
	Packages          []PackageResult `json:"packages"`
}

// PackageResult describes coverage for a specific Go package.
type PackageResult struct {
	Path           string          `json:"path"`
	Coverage       float64         `json:"coverage"`
	Documented     int             `json:"documented"`
	Undocumented   int             `json:"undocumented"`
	MissingSymbols []MissingSymbol `json:"missingSymbols"`
}

// MissingSymbol enumerates an exported declaration with a missing or invalid docstring.
type MissingSymbol struct {
	Symbol string `json:"symbol"`
	Kind   string `json:"kind"`
	File   string `json:"file"`
	Line   int    `json:"line"`
	Reason string `json:"reason"`
}

// Analyze walks every package rooted at cfg.Root and produces coverage metrics.
func Analyze(ctx context.Context, cfg Config) (*Result, error) {
	root := cfg.Root
	if root == "" {
		root = "."
	}
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("resolve root: %w", err)
	}

	loadCfg := &packages.Config{
		Dir:  absRoot,
		Mode: packages.NeedName | packages.NeedSyntax | packages.NeedFiles | packages.NeedCompiledGoFiles | packages.NeedModule | packages.NeedTypes,
	}
	loadCfg.Context = ctx
	patterns := cfg.Patterns
	if len(patterns) == 0 {
		patterns = []string{"./..."}
	}
	pkgs, err := packages.Load(loadCfg, patterns...)
	if err != nil {
		return nil, fmt.Errorf("load packages: %w", err)
	}
	if len(pkgs) == 0 {
		return nil, errors.New("no packages discovered for coverage analysis")
	}

	result := &Result{GeneratedAt: time.Now().UTC()}
	for _, pkg := range pkgs {
		if !shouldConsiderPackage(pkg, absRoot) {
			continue
		}
		pkgResult, err := analyzePackage(pkg, absRoot)
		if err != nil {
			return nil, err
		}
		result.Packages = append(result.Packages, pkgResult)
		result.DocumentedExports += pkgResult.Documented
		result.TotalExports += pkgResult.Documented + pkgResult.Undocumented
	}

	sort.Slice(result.Packages, func(i, j int) bool {
		return result.Packages[i].Path < result.Packages[j].Path
	})
	result.OverallCoverage = coveragePercent(result.DocumentedExports, result.TotalExports)

	return result, nil
}

func shouldConsiderPackage(pkg *packages.Package, absRoot string) bool {
	if pkg == nil {
		return false
	}
	if pkg.Module == nil {
		return false
	}
	if pkg.Module.Dir != absRoot {
		return false
	}
	if len(pkg.GoFiles) == 0 && len(pkg.CompiledGoFiles) == 0 {
		return false
	}
	if len(pkg.Errors) > 0 {
		return false
	}
	return true
}

func analyzePackage(pkg *packages.Package, root string) (PackageResult, error) {
	files := pkg.Syntax
	if len(files) == 0 {
		return PackageResult{}, fmt.Errorf("package %s has no syntax files", pkg.PkgPath)
	}
	path := packageRelativePath(pkg, root)
	symbols := collectSymbols(pkg, root)
	documented := 0
	missing := make([]MissingSymbol, 0)
	for _, sym := range symbols {
		valid, reason := docIsValid(pkg.Name, sym)
		if valid {
			documented++
			continue
		}
		missing = append(missing, MissingSymbol{
			Symbol: sym.name,
			Kind:   sym.kind,
			File:   sym.file,
			Line:   sym.line,
			Reason: reason,
		})
	}
	sort.Slice(missing, func(i, j int) bool {
		if missing[i].Symbol == missing[j].Symbol {
			return missing[i].File < missing[j].File
		}
		return missing[i].Symbol < missing[j].Symbol
	})
	total := len(symbols)
	return PackageResult{
		Path:           path,
		Coverage:       coveragePercent(documented, total),
		Documented:     documented,
		Undocumented:   total - documented,
		MissingSymbols: missing,
	}, nil
}

func packageRelativePath(pkg *packages.Package, root string) string {
	files := pkg.CompiledGoFiles
	if len(files) == 0 {
		files = pkg.GoFiles
	}
	if len(files) == 0 {
		return pkg.PkgPath
	}
	dir := filepath.Dir(files[0])
	rel, err := filepath.Rel(root, dir)
	if err != nil || rel == "." {
		return pkg.PkgPath
	}
	return filepath.ToSlash(rel)
}

type symbolRecord struct {
	name string
	kind string
	file string
	line int
	doc  *ast.CommentGroup
}

func collectSymbols(pkg *packages.Package, root string) []symbolRecord {
	fset := pkg.Fset
	var symbols []symbolRecord //nolint:prealloc // size unknown upfront, accumulated from multiple files
	files := pkg.Syntax
	compiled := pkg.CompiledGoFiles
	goFiles := pkg.GoFiles

	firstFile := ""
	if len(compiled) > 0 {
		firstFile = compiled[0]
	} else if len(goFiles) > 0 {
		firstFile = goFiles[0]
	}
	firstLine := 1
	var packageDoc *ast.CommentGroup
	if len(files) > 0 {
		packageDoc = files[0].Doc
		if packageDoc != nil {
			firstLine = fset.Position(packageDoc.Pos()).Line
		}
	}

	for i, file := range files {
		filename := filePathAtIndex(compiled, goFiles, i)
		symbols = append(symbols, collectFileSymbols(file, fset, root, filename)...)

		if packageDoc == nil && file.Doc != nil {
			packageDoc = file.Doc
			firstLine = fset.Position(packageDoc.Pos()).Line
			if firstFile == "" {
				firstFile = filename
			}
		}
	}

	if firstFile == "" {
		firstFile = relativePath(root, root)
	}
	symbols = append(symbols, symbolRecord{
		name: pkg.Name,
		kind: kindPackage,
		file: relativePath(root, firstFile),
		line: firstLine,
		doc:  packageDoc,
	})

	return symbols
}

func collectFileSymbols(file *ast.File, fset *token.FileSet, root, filename string) []symbolRecord {
	symbols := make([]symbolRecord, 0, len(file.Decls))
	for _, decl := range file.Decls {
		switch node := decl.(type) {
		case *ast.FuncDecl:
			if sym := collectFuncSymbol(node, fset, root, filename); sym != nil {
				symbols = append(symbols, *sym)
			}
		case *ast.GenDecl:
			symbols = append(symbols, collectGenSymbols(node, fset, root, filename)...)
		}
	}
	return symbols
}

func collectFuncSymbol(node *ast.FuncDecl, fset *token.FileSet, root, filename string) *symbolRecord {
	if node.Name == nil || !node.Name.IsExported() {
		return nil
	}
	kind := kindFunc
	if node.Recv != nil {
		kind = kindMethod
	}
	return &symbolRecord{
		name: node.Name.Name,
		kind: kind,
		file: relativePath(root, filename),
		line: fset.Position(node.Pos()).Line,
		doc:  node.Doc,
	}
}

func collectGenSymbols(node *ast.GenDecl, fset *token.FileSet, root, filename string) []symbolRecord {
	symbols := make([]symbolRecord, 0, len(node.Specs))
	if node.Tok != token.TYPE {
		return nil
	}
	for _, spec := range node.Specs {
		typeSpec, ok := spec.(*ast.TypeSpec)
		if !ok || typeSpec.Name == nil || !typeSpec.Name.IsExported() {
			continue
		}
		doc := typeSpec.Doc
		if doc == nil {
			doc = node.Doc
		}
		symbols = append(symbols, symbolRecord{
			name: typeSpec.Name.Name,
			kind: kindType,
			file: relativePath(root, filename),
			line: fset.Position(typeSpec.Pos()).Line,
			doc:  doc,
		})
	}
	return symbols
}

func filePathAtIndex(compiled, plain []string, idx int) string {
	if idx < len(compiled) && compiled[idx] != "" {
		return compiled[idx]
	}
	if idx < len(plain) {
		return plain[idx]
	}
	return ""
}

func relativePath(root, file string) string {
	if file == "" {
		return ""
	}
	abs := file
	if !filepath.IsAbs(file) {
		abs, _ = filepath.Abs(file)
	}
	rel, err := filepath.Rel(root, abs)
	if err != nil {
		return filepath.ToSlash(file)
	}
	return filepath.ToSlash(rel)
}

func docIsValid(pkgName string, sym symbolRecord) (bool, string) {
	if sym.kind == kindPackage {
		return validatePackageComment(pkgName, sym.doc)
	}
	return validateDocstring(sym.name, sym.doc)
}

func validatePackageComment(pkgName string, doc *ast.CommentGroup) (bool, string) {
	if doc == nil {
		return false, reasonMissing
	}
	text := normalizeDoc(doc.Text())
	if text == "" {
		return false, reasonMissing
	}
	if !strings.HasPrefix(text, "Package "+pkgName) {
		return false, reasonMismatchedPrefix
	}
	if !looksSentence(text) {
		return false, reasonFormatting
	}
	return true, ""
}

func validateDocstring(symbol string, doc *ast.CommentGroup) (bool, string) {
	if doc == nil {
		return false, reasonMissing
	}
	text := normalizeDoc(doc.Text())
	if text == "" {
		return false, reasonMissing
	}
	if !strings.HasPrefix(text, symbol) {
		return false, reasonMismatchedPrefix
	}
	if !looksSentence(text) {
		return false, reasonFormatting
	}
	return true, ""
}

func normalizeDoc(doc string) string {
	return strings.TrimSpace(strings.ReplaceAll(doc, "\n", " "))
}

func looksSentence(text string) bool {
	if text == "" {
		return false
	}
	firstRune, _ := utf8.DecodeRuneInString(text)
	if !unicode.IsUpper(firstRune) {
		return false
	}
	return strings.Contains(text, ".")
}

func coveragePercent(documented, total int) float64 {
	if total == 0 {
		return 100
	}
	value := (float64(documented) / float64(total)) * 100
	return math.Round(value*100) / 100
}
