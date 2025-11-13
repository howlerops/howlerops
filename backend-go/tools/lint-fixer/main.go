package main

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Automated lint fixer for 379 golangci-lint errors

type Fixer struct {
	fset     *token.FileSet
	modified map[string]bool
	stats    *Stats
}

type Stats struct {
	Dupword     int
	Errorlint   int
	NilContext  int
	Unused      int
	Ineffassign int
	Gosimple    int
	Errcheck    int
	Govet       int
	Gosec       int
	Total       int
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run main.go <root-directory>")
		os.Exit(1)
	}

	rootDir := os.Args[1]

	fixer := &Fixer{
		fset:     token.NewFileSet(),
		modified: make(map[string]bool),
		stats:    &Stats{},
	}

	fmt.Println("Starting automated lint fixes...")
	fmt.Println("Target: 379 errors")
	fmt.Println("")

	err := filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories and non-Go files
		if info.IsDir() || !strings.HasSuffix(path, ".go") {
			return nil
		}

		// Skip vendor and generated files
		if strings.Contains(path, "/vendor/") || strings.Contains(path, "/.git/") {
			return nil
		}

		return fixer.processFile(path)
	})

	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("\n=== Fix Summary ===")
	fmt.Printf("Dupword fixes:     %d\n", fixer.stats.Dupword)
	fmt.Printf("Errorlint fixes:   %d\n", fixer.stats.Errorlint)
	fmt.Printf("Nil context fixes: %d\n", fixer.stats.NilContext)
	fmt.Printf("Gosimple fixes:    %d\n", fixer.stats.Gosimple)
	fmt.Printf("Total fixes:       %d\n", fixer.stats.Total)
	fmt.Printf("Files modified:    %d\n", len(fixer.modified))
	fmt.Println("")
	fmt.Println("Note: Some errors require manual fixes:")
	fmt.Println("  - errcheck: Add proper error handling")
	fmt.Println("  - govet shadow: Rename shadowed variables")
	fmt.Println("  - gosec: Fix security issues")
	fmt.Println("  - SA1029: Define custom context key types")
	fmt.Println("  - nilerr: Return err instead of nil")
}

func (f *Fixer) processFile(path string) error {
	// #nosec G304 - path from filepath.Walk, not user input
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	original := string(content)
	modified := original

	// Apply text-based fixes
	modified = f.fixDupword(modified)
	modified = f.fixErrorlint(modified)
	modified = f.fixNilContext(modified)
	modified = f.fixGosimple(modified)

	// If changed, write back
	if modified != original {
		// Try to parse and format
		fset := token.NewFileSet()
		node, err := parser.ParseFile(fset, path, modified, parser.ParseComments)
		if err != nil {
			// Write as-is if parsing fails
			fmt.Printf("Warning: %s - parsing failed, writing unformatted\n", path)
			return os.WriteFile(path, []byte(modified), 0600)
		}

		var buf bytes.Buffer
		if err := format.Node(&buf, fset, node); err != nil {
			return os.WriteFile(path, []byte(modified), 0600)
		}

		f.modified[path] = true
		return os.WriteFile(path, buf.Bytes(), 0600)
	}

	return nil
}

func (f *Fixer) fixDupword(content string) string {
	// Fix "TiDB TiDB" -> "TiDB"
	re1 := regexp.MustCompile(`\bTiDB\s+TiDB\b`)
	before1 := content
	content = re1.ReplaceAllString(content, "TiDB")
	if content != before1 {
		f.stats.Dupword++
		f.stats.Total++
	}

	// Fix "user user" -> "user"
	re2 := regexp.MustCompile(`\buser\s+user\b`)
	before2 := content
	content = re2.ReplaceAllString(content, "user")
	if content != before2 {
		f.stats.Dupword++
		f.stats.Total++
	}

	return content
}

func (f *Fixer) fixErrorlint(content string) string {
	count := 0

	// Fix fmt.Errorf with %v -> %w for error wrapping
	// Pattern 1: fmt.Errorf("...: %v", err)
	re1 := regexp.MustCompile(`fmt\.Errorf\(("(?:[^"\\]|\\.)*?): %v", err\)`)
	if re1.MatchString(content) {
		content = re1.ReplaceAllString(content, `fmt.Errorf($1: %w", err)`)
		count++
	}

	// Pattern 2: fmt.Errorf("...%v...", ..., err)
	re2 := regexp.MustCompile(`fmt\.Errorf\(("(?:[^"\\]|\\.)*?)%v((?:[^"\\]|\\.)*?"), ([^,]+?), err\)`)
	if re2.MatchString(content) {
		content = re2.ReplaceAllString(content, `fmt.Errorf($1%w$2, $3, err)`)
		count++
	}

	if count > 0 {
		f.stats.Errorlint += count
		f.stats.Total += count
	}

	return content
}

func (f *Fixer) fixNilContext(content string) string {
	count := 0

	patterns := []struct {
		old string
		new string
	}{
		{", nil)", ", context.TODO())"},
		{"(nil,", "(context.TODO(),"},
		{"(nil)", "(context.TODO())"},
	}

	for _, p := range patterns {
		if strings.Contains(content, p.old) {
			content = strings.ReplaceAll(content, p.old, p.new)
			count++
		}
	}

	// Ensure context is imported
	if count > 0 && !strings.Contains(content, `"context"`) {
		content = f.ensureImport(content, "context")
	}

	if count > 0 {
		f.stats.NilContext += count
		f.stats.Total += count
	}

	return content
}

func (f *Fixer) fixGosimple(content string) string {
	count := 0

	// S1025: Remove unnecessary fmt.Sprintf with single string argument
	re1 := regexp.MustCompile(`fmt\.Sprintf\("%s", ([a-zA-Z_][a-zA-Z0-9_]*)\)`)
	if re1.MatchString(content) {
		content = re1.ReplaceAllString(content, "$1")
		count++
	}

	if count > 0 {
		f.stats.Gosimple += count
		f.stats.Total += count
	}

	return content
}

func (f *Fixer) ensureImport(content string, pkg string) string {
	importStmt := fmt.Sprintf(`"%s"`, pkg)
	if strings.Contains(content, importStmt) {
		return content
	}

	// Find import block and add
	importBlock := regexp.MustCompile(`import \(\n`)
	if importBlock.MatchString(content) {
		content = importBlock.ReplaceAllString(content, fmt.Sprintf("import (\n\t%s\n", importStmt))
	} else {
		// Find package line and add after
		packageLine := regexp.MustCompile(`package [a-z_]+\n`)
		content = packageLine.ReplaceAllString(content, "$0\nimport "+importStmt+"\n")
	}

	return content
}

// AST-based fixes for more complex issues
func (f *Fixer) fixWithAST(path string) error {
	node, err := parser.ParseFile(f.fset, path, nil, parser.ParseComments)
	if err != nil {
		return err
	}

	modified := false

	// Walk AST and fix issues
	ast.Inspect(node, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.AssignStmt:
			// Fix ineffectual assignments
			_ = x
		case *ast.CallExpr:
			// Fix error checking
			_ = x
		}
		return true
	})

	if !modified {
		return nil
	}

	// Write back
	var buf bytes.Buffer
	if err := format.Node(&buf, f.fset, node); err != nil {
		return err
	}

	return os.WriteFile(path, buf.Bytes(), 0600)
}
